package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// ScopeFunc reports which students the caller may see. restricted=false
// means all of them (staff); otherwise only ids in the slice.
type ScopeFunc func(r *http.Request) (ids []uuid.UUID, restricted bool, err error)

type Registry struct {
	pool      *pgxpool.Pool
	scope     ScopeFunc
	resources map[string]*Resource
	order     []string
}

func NewRegistry(pool *pgxpool.Pool, scope ScopeFunc) *Registry {
	return &Registry{pool: pool, scope: scope, resources: map[string]*Resource{}}
}

// Add registers resources; a bad spec panics at startup.
func (g *Registry) Add(rs ...Resource) {
	for i := range rs {
		res := rs[i]
		if err := res.Check(); err != nil {
			panic(err)
		}
		if _, dup := g.resources[res.Name]; dup {
			panic("resource: duplicate " + res.Name)
		}
		if res.OrderBy == "" {
			res.OrderBy = "t.created_at DESC"
		}
		g.resources[res.Name] = &res
		g.order = append(g.order, res.Name)
	}
}

// Mount wires every resource at /<name>/ plus the schema endpoints under
// /meta/resources/. r must already sit behind the portal auth middleware.
func (g *Registry) Mount(r chi.Router) {
	r.Get("/meta/resources/", g.listSchemas)
	r.Get("/meta/resources/{name}/", g.getSchema)
	r.Get("/meta/resources/{name}/options/{field}/", g.options)
	for _, name := range g.order {
		res := g.resources[name]
		r.Route("/"+name+"/", func(r chi.Router) {
			r.Get("/", g.list(res))
			r.Post("/", g.create(res))
			r.Get("/{id}/", g.get(res))
			r.Put("/{id}/", g.update(res))
			r.Patch("/{id}/", g.update(res))
			r.Delete("/{id}/", g.delete(res))
		})
	}
}

// Verify checks every spec against the live schema — each table and column
// must exist — so a typo or an unapplied migration fails at startup rather
// than on someone's form submit.
func (g *Registry) Verify(ctx context.Context) error {
	var problems []string
	for _, name := range g.order {
		res := g.resources[name]
		rows, err := g.pool.Query(ctx, `
			SELECT column_name FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1`, res.Table)
		if err != nil {
			return err
		}
		cols := map[string]bool{}
		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err == nil {
				cols[c] = true
			}
		}
		rows.Close()
		if len(cols) == 0 {
			problems = append(problems, fmt.Sprintf("%s: table %s missing", name, res.Table))
			continue
		}
		need := []string{"id", "created_at", res.CreatorField}
		if res.HasUpdatedAt {
			need = append(need, "updated_at")
		}
		for _, f := range res.Fields {
			need = append(need, f.Name)
		}
		for _, c := range need {
			if c != "" && !cols[c] {
				problems = append(problems, fmt.Sprintf("%s: column %s.%s missing", name, res.Table, c))
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("resource specs do not match the database (run migrations?):\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}

// Names lists the registered resources in registration order.
func (g *Registry) Names() []string { return append([]string(nil), g.order...) }

// ---------------------------------------------------------------------------
// Permissions

func role(r *http.Request) types.Role {
	v, _ := r.Context().Value("role").(types.Role)
	return v
}

func roleIn(rl types.Role, list []types.Role) bool {
	for _, x := range list {
		if x == rl {
			return true
		}
	}
	return false
}

func isFamily(rl types.Role) bool { return rl == types.RoleParent || rl == types.RoleStudent }

func canRead(res *Resource, rl types.Role) bool {
	if rl == types.RoleSystemAdmin {
		return true
	}
	if isFamily(rl) {
		return res.StudentField != "" && (len(res.ReadRoles) == 0 || roleIn(rl, res.ReadRoles))
	}
	if rl == "" {
		return false
	}
	return len(res.ReadRoles) == 0 || roleIn(rl, res.ReadRoles) || canWrite(res, rl)
}

func canWrite(res *Resource, rl types.Role) bool {
	if rl == types.RoleSystemAdmin {
		return true
	}
	if len(res.WriteRoles) == 0 {
		return roleIn(rl, types.AdminRoles)
	}
	return roleIn(rl, res.WriteRoles)
}

// scopeClause restricts family callers to their own children's rows.
func (g *Registry) scopeClause(r *http.Request, res *Resource, args *[]interface{}) (string, error) {
	if !isFamily(role(r)) || g.scope == nil {
		return "", nil
	}
	ids, restricted, err := g.scope(r)
	if err != nil {
		return "", err
	}
	if !restricted {
		return "", nil
	}
	*args = append(*args, ids)
	return fmt.Sprintf("t.%s = ANY($%d)", res.StudentField, len(*args)), nil
}

// ---------------------------------------------------------------------------
// Reading

func (g *Registry) selectList(res *Resource) (string, []string) {
	cols := []string{"t.id::text", "t.created_at"}
	names := []string{"id", "created_at"}
	for i := range res.Fields {
		f := &res.Fields[i]
		cols = append(cols, selectExpr(f))
		names = append(names, f.Name)
		if f.Type == Ref {
			cols = append(cols, fmt.Sprintf("(SELECT %s FROM %s x WHERE x.id = t.%s)", f.Ref.Label, f.Ref.Table, f.Name))
			names = append(names, f.Name+"_label")
		}
	}
	return strings.Join(cols, ", "), names
}

func scanRows(rows pgx.Rows, names []string) ([]map[string]interface{}, error) {
	out := []map[string]interface{}{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		m := make(map[string]interface{}, len(names))
		for i, n := range names {
			m[n] = vals[i]
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (g *Registry) list(res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !canRead(res, role(r)) {
			httpx.ErrForbidden(w, "not allowed to view "+res.Title)
			return
		}
		q := r.URL.Query()
		var where []string
		var args []interface{}

		sc, err := g.scopeClause(r, res, &args)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		if sc != "" {
			where = append(where, sc)
		}

		fieldErrs := map[string]string{}
		for i := range res.Fields {
			f := &res.Fields[i]
			raw := q.Get(f.Name)
			if !f.Filter || raw == "" {
				continue
			}
			v, err := convert(f, raw)
			if err != nil {
				fieldErrs[f.Name] = err.Error()
				continue
			}
			args = append(args, v)
			where = append(where, fmt.Sprintf("t.%s = %s", f.Name, placeholder(f, len(args))))
		}
		if len(fieldErrs) > 0 {
			validationError(w, fieldErrs)
			return
		}

		if s := strings.TrimSpace(q.Get("search")); s != "" {
			var ors []string
			args = append(args, "%"+escapeLike(s)+"%")
			for i := range res.Fields {
				if res.Fields[i].Search {
					ors = append(ors, fmt.Sprintf("t.%s ILIKE $%d", res.Fields[i].Name, len(args)))
				}
			}
			if len(ors) == 0 {
				args = args[:len(args)-1]
			} else {
				where = append(where, "("+strings.Join(ors, " OR ")+")")
			}
		}

		whereSQL := ""
		if len(where) > 0 {
			whereSQL = " WHERE " + strings.Join(where, " AND ")
		}

		var total int
		if err := g.pool.QueryRow(r.Context(), "SELECT count(*) FROM "+res.Table+" t"+whereSQL, args...).Scan(&total); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}

		page, size := pageParams(q.Get("page"), q.Get("page_size"))
		order := res.OrderBy
		if o := q.Get("ordering"); o != "" {
			desc := strings.HasPrefix(o, "-")
			name := strings.TrimPrefix(o, "-")
			if _, ok := res.field(name); ok || name == "created_at" {
				order = "t." + name
				if desc {
					order += " DESC"
				}
				order += ", t.id"
			}
		}
		cols, names := g.selectList(res)
		sql := fmt.Sprintf("SELECT %s FROM %s t%s ORDER BY %s LIMIT %d OFFSET %d",
			cols, res.Table, whereSQL, order, size, (page-1)*size)
		rows, err := g.pool.Query(r.Context(), sql, args...)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()
		out, err := scanRows(rows, names)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]interface{}{
			"results": out, "count": total, "page": page, "page_size": size,
		})
	}
}

func pageParams(p, s string) (int, int) {
	page, _ := strconv.Atoi(p)
	size, _ := strconv.Atoi(s)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 50
	}
	if size > 200 {
		size = 200
	}
	return page, size
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// fetch loads one row, honouring family scope. ok=false means not found
// (or not visible — the caller can't tell the difference).
func (g *Registry) fetch(r *http.Request, res *Resource, id uuid.UUID) (map[string]interface{}, bool, error) {
	args := []interface{}{id}
	where := "t.id = $1"
	sc, err := g.scopeClause(r, res, &args)
	if err != nil {
		return nil, false, err
	}
	if sc != "" {
		where += " AND " + sc
	}
	cols, names := g.selectList(res)
	rows, err := g.pool.Query(r.Context(), fmt.Sprintf("SELECT %s FROM %s t WHERE %s", cols, res.Table, where), args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out, err := scanRows(rows, names)
	if err != nil || len(out) == 0 {
		return nil, false, err
	}
	return out[0], true, nil
}

func (g *Registry) get(res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !canRead(res, role(r)) {
			httpx.ErrForbidden(w, "not allowed to view "+res.Title)
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			httpx.ErrBadRequest(w, "invalid id")
			return
		}
		row, ok, err := g.fetch(r, res, id)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			httpx.ErrNotFound(w, res.Title+" not found")
			return
		}
		httpx.JSON(w, http.StatusOK, row)
	}
}

// ---------------------------------------------------------------------------
// Writing

// decodeBody reads a JSON object, validates every key against the spec and
// converts values. partial=true (updates) skips the required-field check
// for fields that weren't sent.
func decodeBody(r *http.Request, res *Resource, partial bool) (map[string]interface{}, map[string]string, error) {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.UseNumber()
	var body map[string]interface{}
	if err := dec.Decode(&body); err != nil {
		return nil, nil, errors.New("request body must be a JSON object")
	}
	vals := map[string]interface{}{}
	errs := map[string]string{}
	for k := range body {
		f, ok := res.field(k)
		if !ok {
			// Server-managed keys echoed back from a GET are ignored;
			// anything else is almost certainly a client bug.
			if k == "id" || k == "created_at" || k == "updated_at" || strings.HasSuffix(k, "_label") {
				continue
			}
			errs[k] = "unknown field"
			continue
		}
		if f.ReadOnly {
			continue
		}
		v, err := convert(f, body[k])
		if err != nil {
			errs[k] = err.Error()
			continue
		}
		if f.Required && (v == nil || v == "") {
			errs[k] = errEmpty.Error()
			continue
		}
		vals[k] = v
	}
	if !partial {
		for i := range res.Fields {
			f := &res.Fields[i]
			if f.Required && !f.ReadOnly {
				if _, sent := vals[f.Name]; !sent {
					if _, bad := errs[f.Name]; !bad {
						errs[f.Name] = errEmpty.Error()
					}
				}
			}
		}
	}
	return vals, errs, nil
}

func (g *Registry) create(res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !canWrite(res, role(r)) {
			httpx.ErrForbidden(w, "not allowed to change "+res.Title)
			return
		}
		vals, errs, err := decodeBody(r, res, false)
		if err != nil {
			httpx.ErrBadRequest(w, err.Error())
			return
		}
		if len(errs) > 0 {
			validationError(w, errs)
			return
		}
		var cols, ph []string
		var args []interface{}
		for _, name := range sortedKeys(vals) {
			if vals[name] == nil {
				continue // let the column default apply
			}
			f, _ := res.field(name)
			args = append(args, vals[name])
			cols = append(cols, name)
			ph = append(ph, placeholder(f, len(args)))
		}
		if res.CreatorField != "" {
			if uid, ok := r.Context().Value("user_id").(uuid.UUID); ok {
				args = append(args, uid)
				cols = append(cols, res.CreatorField)
				ph = append(ph, "$"+strconv.Itoa(len(args)))
			}
		}
		sql := "INSERT INTO " + res.Table + " DEFAULT VALUES RETURNING id"
		if len(cols) > 0 {
			sql = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
				res.Table, strings.Join(cols, ", "), strings.Join(ph, ", "))
		}
		var id uuid.UUID
		if err := g.pool.QueryRow(r.Context(), sql, args...).Scan(&id); err != nil {
			dbError(w, res, err)
			return
		}
		row, _, err := g.fetch(r, res, id)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, row)
	}
}

func (g *Registry) update(res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !canWrite(res, role(r)) {
			httpx.ErrForbidden(w, "not allowed to change "+res.Title)
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			httpx.ErrBadRequest(w, "invalid id")
			return
		}
		vals, errs, err := decodeBody(r, res, true)
		if err != nil {
			httpx.ErrBadRequest(w, err.Error())
			return
		}
		if len(errs) > 0 {
			validationError(w, errs)
			return
		}
		var sets []string
		var args []interface{}
		for _, name := range sortedKeys(vals) {
			f, _ := res.field(name)
			args = append(args, vals[name])
			sets = append(sets, name+" = "+placeholder(f, len(args)))
		}
		if res.HasUpdatedAt {
			sets = append(sets, "updated_at = now()")
		}
		if len(sets) == 0 {
			httpx.ErrBadRequest(w, "nothing to update")
			return
		}
		args = append(args, id)
		tag, err := g.pool.Exec(r.Context(), fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d",
			res.Table, strings.Join(sets, ", "), len(args)), args...)
		if err != nil {
			dbError(w, res, err)
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.ErrNotFound(w, res.Title+" not found")
			return
		}
		row, _, err := g.fetch(r, res, id)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		httpx.JSON(w, http.StatusOK, row)
	}
}

func (g *Registry) delete(res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !canWrite(res, role(r)) {
			httpx.ErrForbidden(w, "not allowed to change "+res.Title)
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			httpx.ErrBadRequest(w, "invalid id")
			return
		}
		tag, err := g.pool.Exec(r.Context(), "DELETE FROM "+res.Table+" WHERE id = $1", id)
		if err != nil {
			dbError(w, res, err)
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.ErrNotFound(w, res.Title+" not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func validationError(w http.ResponseWriter, fields map[string]string) {
	httpx.JSON(w, http.StatusBadRequest, map[string]interface{}{
		"error": "validation", "message": "Please correct the highlighted fields.", "fields": fields,
	})
}

// dbError turns constraint violations into answers a form can show.
func dbError(w http.ResponseWriter, res *Resource, err error) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			httpx.ErrConflict(w, "a "+strings.ToLower(res.Title)+" with these details already exists")
			return
		case "23503":
			if strings.HasPrefix(pg.Message, "update or delete") {
				httpx.ErrConflict(w, "this "+strings.ToLower(res.Title)+" is still in use by other records")
				return
			}
			httpx.ErrBadRequest(w, "a linked record does not exist")
			return
		case "23502":
			validationError(w, map[string]string{pg.ColumnName: errEmpty.Error()})
			return
		case "23514":
			if msg, ok := res.Constraints[pg.ConstraintName]; ok {
				httpx.ErrBadRequest(w, msg)
				return
			}
			httpx.ErrBadRequest(w, "invalid value: "+pg.Message)
			return
		case "22P02", "22007", "22008", "22003":
			httpx.ErrBadRequest(w, "invalid value: "+pg.Message)
			return
		}
	}
	httpx.Error(w, http.StatusInternalServerError, err)
}

// ---------------------------------------------------------------------------
// Schemas — what the portal builds forms and tables from

type schemaSummary struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Module   string `json:"module"`
	CanWrite bool   `json:"can_write"`
}

func (g *Registry) listSchemas(w http.ResponseWriter, r *http.Request) {
	rl := role(r)
	out := []schemaSummary{}
	for _, name := range g.order {
		res := g.resources[name]
		if canRead(res, rl) {
			out = append(out, schemaSummary{res.Name, res.Title, res.Module, canWrite(res, rl)})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

// options serves {id,label} choices for one ref field's dropdown, using the
// field's own label expression so every form shows referenced rows the same
// way. Only callers who may write the resource get options: a parent can
// read their child's records but must not be able to list every student.
func (g *Registry) options(w http.ResponseWriter, r *http.Request) {
	res, ok := g.resources[chi.URLParam(r, "name")]
	if !ok {
		httpx.ErrNotFound(w, "unknown resource")
		return
	}
	f, ok := res.field(chi.URLParam(r, "field"))
	if !ok || f.Type != Ref {
		httpx.ErrNotFound(w, "unknown reference field")
		return
	}
	if !canWrite(res, role(r)) {
		httpx.ErrForbidden(w, "not allowed to change "+res.Title)
		return
	}
	var args []interface{}
	where := ""
	if s := strings.TrimSpace(r.URL.Query().Get("search")); s != "" {
		args = append(args, "%"+escapeLike(s)+"%")
		where = fmt.Sprintf(" WHERE (%s) ILIKE $1", f.Ref.Label)
	}
	if id := r.URL.Query().Get("id"); id != "" {
		// Resolve the label of a value already on the form.
		uid, err := uuid.Parse(id)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid id")
			return
		}
		args = []interface{}{uid}
		where = " WHERE x.id = $1"
	}
	rows, err := g.pool.Query(r.Context(), fmt.Sprintf(
		"SELECT x.id::text, COALESCE((%s)::text, '') AS label FROM %s x%s ORDER BY 2 LIMIT 50",
		f.Ref.Label, f.Ref.Table, where), args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type option struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	out := []option{}
	for rows.Next() {
		var o option
		if err := rows.Scan(&o.ID, &o.Label); err == nil {
			out = append(out, o)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (g *Registry) getSchema(w http.ResponseWriter, r *http.Request) {
	res, ok := g.resources[chi.URLParam(r, "name")]
	if !ok {
		// Nested names (library/book-fund) arrive URL-encoded.
		res, ok = g.resources[strings.ReplaceAll(chi.URLParam(r, "name"), "%2F", "/")]
	}
	if !ok || !canRead(res, role(r)) {
		httpx.ErrNotFound(w, "unknown resource")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"name": res.Name, "title": res.Title, "module": res.Module,
		"fields": res.Fields, "can_write": canWrite(res, role(r)),
	})
}
