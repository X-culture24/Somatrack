//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/api"
	lmsmiddleware "github.com/stmaryskabete/lms/internal/middleware"
	"github.com/stmaryskabete/lms/internal/payments"
	jwtutil "github.com/stmaryskabete/lms/internal/shared/jwt"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
	"github.com/stmaryskabete/lms/internal/shared/types"
	"github.com/stmaryskabete/lms/internal/signedlink"
	"github.com/stmaryskabete/lms/internal/testutil"
)

// Fixed ids keep failures readable.
var (
	yearID    = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	termID    = uuid.MustParse("10000000-0000-0000-0000-000000000002") // current
	eastID    = uuid.MustParse("20000000-0000-0000-0000-00000000000e")
	westID    = uuid.MustParse("20000000-0000-0000-0000-00000000000f")
	mathID    = uuid.MustParse("30000000-0000-0000-0000-000000000001")
	engID     = uuid.MustParse("30000000-0000-0000-0000-000000000002")
	adminID   = uuid.MustParse("40000000-0000-0000-0000-000000000001")
	westTchID = uuid.MustParse("40000000-0000-0000-0000-000000000002") // class teacher of West
	otherTch  = uuid.MustParse("40000000-0000-0000-0000-000000000003") // teaches nothing
	parentID  = uuid.MustParse("40000000-0000-0000-0000-000000000004") // Brian's parent
	financeID = uuid.MustParse("40000000-0000-0000-0000-000000000005")
	faithID   = uuid.MustParse("50000000-0000-0000-0000-00000000000a") // Grade 4 East
	brianID   = uuid.MustParse("50000000-0000-0000-0000-00000000000b") // Grade 4 West
)

const portalURL = "https://portal.test"
const accountPrefix = "137101"

// env is one running copy of the platform over a private schema.
type env struct {
	t       *testing.T
	pool    *pgxpool.Pool
	srv     *httptest.Server
	jwt     *jwtutil.Manager
	keys    *signedlink.Keyring
	sms     *fakeSMS
	payment *payments.Service
	notify  *payments.Notifier
}

func (e *env) notifier() *payments.Notifier { return e.notify }

func newEnv(t *testing.T) *env {
	t.Helper()
	// Every env has its own schema, so tests can run side by side.
	t.Parallel()
	pool := testutil.DB(t)
	seed(t, pool)

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keys, err := signedlink.NewKeyring(1, priv, nil)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, pool: pool, jwt: jwtutil.NewManager("integration-secret", time.Hour, time.Hour, false), keys: keys}

	// Stand-in notification-svc capturing every SMS it's asked to send.
	e.sms = &fakeSMS{}
	smsSrv := httptest.NewServer(e.sms)
	t.Cleanup(smsSrv.Close)
	notif := rpc.NewNotificationClient(smsSrv.URL, "k")

	loc, _ := time.LoadLocation("Africa/Nairobi")
	rec := payments.NewReconciler(pool, loc)
	e.notify = payments.NewNotifier(pool, notif, portalURL)
	rec.SetNotifier(e.notify)
	matcher := payments.NewMatcher(pool, accountPrefix, loc)
	e.payment = payments.NewService(matcher, rec, nil, "", nil, notif, loc, false, nil)

	links := api.NewLinksService(pool, keys, portalURL)
	resources := api.NewResourceRegistry(pool)
	if err := resources.Verify(context.Background()); err != nil {
		t.Fatalf("resource specs vs schema: %v", err)
	}

	r := chi.NewRouter()
	links.RegisterPublicRoutes(r)
	r.Route("/internal", func(r chi.Router) { e.payment.RegisterInternalRoutes(r) })
	r.Route("/api", func(r chi.Router) {
		r.Use(lmsmiddleware.Auth(e.jwt, jwtutil.ScopePortal))
		links.RegisterRoutes(r)
		resources.Mount(r)
		api.NewResultsService(pool, links).RegisterRoutes(r)
		api.NewDocumentsService(pool, links).RegisterRoutes(r)
	})
	e.srv = httptest.NewServer(r)
	t.Cleanup(e.srv.Close)
	return e
}

func seed(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	testutil.MustExec(t, pool, `
		INSERT INTO academic_years (id, name, start_date, end_date, is_active) VALUES ($1, '2026', '2026-01-01', '2026-12-31', TRUE);
	`, yearID)
	testutil.MustExec(t, pool, `
		INSERT INTO terms (id, academic_year_id, name, term_number, start_date, end_date, is_current)
		VALUES ($1, $2, 'Term 1', 1, '2026-01-05', '2026-04-03', TRUE)`, termID, yearID)
	for _, u := range []struct {
		id          uuid.UUID
		email, role string
	}{
		{adminID, "admin@it", "system_admin"}, {westTchID, "west@it", "class_teacher"},
		{otherTch, "other@it", "class_teacher"}, {parentID, "parent@it", "parent"}, {financeID, "bursar@it", "finance_officer"},
	} {
		testutil.MustExec(t, pool, `INSERT INTO users (id, password, email, role, first_name, last_name) VALUES ($1, 'x', $2, $3, 'T', 'User')`, u.id, u.email, u.role)
	}
	testutil.MustExec(t, pool, `
		INSERT INTO class_groups (id, name, grade, academic_year_id, class_teacher_id) VALUES
			($1, 'Grade 4 East', 4, $3, NULL), ($2, 'Grade 4 West', 4, $3, $4)`, eastID, westID, yearID, westTchID)
	testutil.MustExec(t, pool, `
		INSERT INTO subjects (id, name, code) VALUES ($1, 'Mathematics', 'MAT'), ($2, 'English', 'ENG')`, mathID, engID)
	testutil.MustExec(t, pool, `
		INSERT INTO class_subjects (class_id, subject_id) VALUES ($1, $3), ($1, $4), ($2, $3), ($2, $4)`, eastID, westID, mathID, engID)
	testutil.MustExec(t, pool, `
		INSERT INTO students (id, admission_no, first_name, last_name, full_name, search_name, gender, current_class_id, is_active) VALUES
			($1, 'ADM-001', 'Faith', 'Otieno', 'Faith Otieno', 'faith otieno', 'female', $3, TRUE),
			($2, 'ADM-002', 'Brian', 'Kamau', 'Brian Kamau', 'brian kamau', 'male', $4, TRUE)`, faithID, brianID, eastID, westID)
	testutil.MustExec(t, pool, `
		WITH g AS (INSERT INTO guardians (first_name, last_name, phone, user_id) VALUES ('Mary', 'Kamau', '0722000111', $1) RETURNING id),
		     g2 AS (INSERT INTO guardians (first_name, last_name, phone) VALUES ('Peter', 'Otieno', '0733000222') RETURNING id)
		INSERT INTO student_guardians (student_id, guardian_id, is_primary)
		SELECT $2::uuid, id, TRUE FROM g UNION ALL SELECT $3::uuid, id, TRUE FROM g2`, parentID, brianID, faithID)
	// Brian owes 20,000 this term.
	testutil.MustExec(t, pool, `
		WITH i AS (
			INSERT INTO invoices (student_id, term_id, invoice_no, invoice_date, due_date, total_amount, total_paid, balance, status)
			VALUES ($1, $2, 'INV-ADM-002', '2026-01-05', '2026-02-01', 20000, 0, 20000, 'unpaid') RETURNING id)
		INSERT INTO invoice_lines (invoice_id, line_type, description, unit_price, amount) SELECT id, 'tuition', 'Tuition', 20000, 20000 FROM i`, brianID, termID)
}

// ---------------------------------------------------------------- HTTP

type caller struct {
	e     *env
	token string
}

func (e *env) as(id uuid.UUID, role types.Role) caller {
	p, err := e.jwt.IssueTokenPair(&types.User{ID: id, Role: role, Portal: types.RoleToPortal(role)}, jwtutil.ScopePortal)
	if err != nil {
		e.t.Fatal(err)
	}
	return caller{e: e, token: p.Access}
}

func (e *env) admin() caller   { return e.as(adminID, types.RoleSystemAdmin) }
func (e *env) westT() caller   { return e.as(westTchID, types.RoleClassTeacher) }
func (e *env) otherT() caller  { return e.as(otherTch, types.RoleClassTeacher) }
func (e *env) parent() caller  { return e.as(parentID, types.RoleParent) }
func (e *env) finance() caller { return e.as(financeID, types.RoleFinanceOfficer) }
func (e *env) anon() caller    { return caller{e: e} }

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) JSON(t *testing.T) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("not JSON (%d): %s", r.Status, r.Body)
	}
	return m
}

func (c caller) do(method, path string, body interface{}) resp {
	c.e.t.Helper()
	var rdr io.Reader
	if body != nil {
		if s, ok := body.(string); ok {
			rdr = strings.NewReader(s)
		} else {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		}
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.e.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{Status: res.StatusCode, Body: b, Header: res.Header}
}

func (c caller) get(path string) resp                     { return c.do("GET", path, nil) }
func (c caller) post(path string, body interface{}) resp  { return c.do("POST", path, body) }
func (c caller) patch(path string, body interface{}) resp { return c.do("PATCH", path, body) }
func (c caller) put(path string, body interface{}) resp   { return c.do("PUT", path, body) }
func (c caller) del(path string) resp                     { return c.do("DELETE", path, nil) }

// want fails the test unless the response has the expected status.
func want(t *testing.T, r resp, status int) resp {
	t.Helper()
	if r.Status != status {
		t.Fatalf("status %d, want %d: %s", r.Status, status, r.Body)
	}
	return r
}

// id pulls "id" from a JSON object response.
func id(t *testing.T, r resp) string {
	t.Helper()
	v, _ := r.JSON(t)["id"].(string)
	if v == "" {
		t.Fatalf("no id in %s", r.Body)
	}
	return v
}

// ---------------------------------------------------------------- fake SMS

type sentSMS struct {
	To      string            `json:"to"`
	Message string            `json:"message"`
	Meta    map[string]string `json:"meta"`
}

type fakeSMS struct {
	mu   sync.Mutex
	sent []sentSMS
}

func (f *fakeSMS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var m sentSMS
	_ = json.NewDecoder(r.Body).Decode(&m)
	f.mu.Lock()
	f.sent = append(f.sent, m)
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"queued"}`))
}

func (f *fakeSMS) reset() {
	f.mu.Lock()
	f.sent = nil
	f.mu.Unlock()
}

func (f *fakeSMS) all() []sentSMS {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentSMS(nil), f.sent...)
}

// waitFor polls until n messages matching kind have arrived (the payment
// SMS is sent from a goroutine after the payment commits).
func (f *fakeSMS) waitFor(t *testing.T, kind string, n int) []sentSMS {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var got []sentSMS
		for _, m := range f.all() {
			if m.Meta["kind"] == kind {
				got = append(got, m)
			}
		}
		if len(got) >= n || time.Now().After(deadline) {
			if len(got) < n {
				t.Fatalf("got %d %q SMS, want %d", len(got), kind, n)
			}
			return got
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
