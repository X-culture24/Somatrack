package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/crypto"
	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

type GuardiansService struct {
	pool *pgxpool.Pool
}

func NewGuardiansService(pool *pgxpool.Pool) *GuardiansService {
	return &GuardiansService{pool: pool}
}

func (s *GuardiansService) RegisterRoutes(r chi.Router) {
	r.Route("/guardians/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Post("/", s.create)
		r.Get("/siblings/", s.siblings)
		r.Get("/{id}/", s.get)
		r.Put("/{id}/", s.update)
		r.Patch("/{id}/", s.update)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.delete)).ServeHTTP)
		r.Post("/{id}/create-login/", adminOnly()(http.HandlerFunc(s.createLogin)).ServeHTTP)
	})
}

type Guardian struct {
	ID                 uuid.UUID  `json:"id"`
	UserID             *uuid.UUID `json:"user_id,omitempty"`
	FirstName          string     `json:"first_name" validate:"required"`
	LastName           string     `json:"last_name" validate:"required"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone" validate:"required"`
	AlternatePhone     string     `json:"alternate_phone"`
	NationalID         string     `json:"national_id"`
	Relation           string     `json:"relation"`
	Occupation         string     `json:"occupation"`
	Employer           string     `json:"employer"`
	ResidentialAddress string     `json:"residential_address"`
	PostalAddress      string     `json:"postal_address"`
	IsActive           bool       `json:"is_active"`
	IDDocumentPath     string     `json:"id_document_path"`
}

const guardianCols = `id, user_id, first_name, last_name, COALESCE(email,''), phone, alternate_phone,
	national_id, relation, occupation, employer, residential_address, postal_address, is_active, id_document_path`

func scanGuardian(row pgx.Row) (Guardian, error) {
	var g Guardian
	err := row.Scan(&g.ID, &g.UserID, &g.FirstName, &g.LastName, &g.Email, &g.Phone, &g.AlternatePhone,
		&g.NationalID, &g.Relation, &g.Occupation, &g.Employer, &g.ResidentialAddress, &g.PostalAddress, &g.IsActive,
		&g.IDDocumentPath)
	return g, err
}

func (s *GuardiansService) list(w http.ResponseWriter, r *http.Request) {
	where := "true"
	args := []interface{}{}
	if q := r.URL.Query().Get("search"); q != "" {
		where = "(first_name ILIKE $1 OR last_name ILIKE $1 OR phone ILIKE $1)"
		args = append(args, "%"+q+"%")
	}
	rows, err := s.pool.Query(r.Context(), `SELECT `+guardianCols+` FROM guardians WHERE `+where+` ORDER BY last_name, first_name`, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Guardian{}
	for rows.Next() {
		g, err := scanGuardian(rows)
		if err != nil {
			continue
		}
		out = append(out, g)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *GuardiansService) create(w http.ResponseWriter, r *http.Request) {
	var body Guardian
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	body.IsActive = true
	if body.Relation == "" {
		body.Relation = "parent"
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO guardians (first_name, last_name, email, phone, alternate_phone, national_id,
			relation, occupation, employer, residential_address, postal_address, is_active, id_document_path)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`,
		body.FirstName, body.LastName, body.Email, body.Phone, body.AlternatePhone, body.NationalID,
		body.Relation, body.Occupation, body.Employer, body.ResidentialAddress, body.PostalAddress, body.IsActive,
		body.IDDocumentPath,
	).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *GuardiansService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	g, err := scanGuardian(s.pool.QueryRow(r.Context(), `SELECT `+guardianCols+` FROM guardians WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "guardian not found")
		return
	}

	students, err := s.pool.Query(r.Context(), `
		SELECT s.id, s.full_name, s.admission_no, sg.relation, sg.is_primary
		FROM student_guardians sg JOIN students s ON s.id = sg.student_id
		WHERE sg.guardian_id = $1`, id)
	wards := []map[string]interface{}{}
	if err == nil {
		defer students.Close()
		for students.Next() {
			var sid uuid.UUID
			var name, adm, rel string
			var primary bool
			if students.Scan(&sid, &name, &adm, &rel, &primary) == nil {
				wards = append(wards, map[string]interface{}{
					"student_id": sid, "full_name": name, "admission_no": adm, "relation": rel, "is_primary": primary,
				})
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"id": g.ID, "user_id": g.UserID, "first_name": g.FirstName, "last_name": g.LastName,
		"email": g.Email, "phone": g.Phone, "alternate_phone": g.AlternatePhone, "national_id": g.NationalID,
		"relation": g.Relation, "occupation": g.Occupation, "employer": g.Employer,
		"residential_address": g.ResidentialAddress, "postal_address": g.PostalAddress, "is_active": g.IsActive,
		"id_document_path": g.IDDocumentPath, "wards": wards,
	})
}

func (s *GuardiansService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body Guardian
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE guardians SET first_name=$1, last_name=$2, email=NULLIF($3,''), phone=$4, alternate_phone=$5,
			national_id=$6, relation=$7, occupation=$8, employer=$9, residential_address=$10, postal_address=$11,
			is_active=$12, id_document_path=$13, updated_at=NOW()
		WHERE id = $14`,
		body.FirstName, body.LastName, body.Email, body.Phone, body.AlternatePhone, body.NationalID,
		body.Relation, body.Occupation, body.Employer, body.ResidentialAddress, body.PostalAddress, body.IsActive,
		body.IDDocumentPath, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	body.ID = id
	httpx.JSON(w, http.StatusOK, body)
}

// siblings lists guardians who have two or more students enrolled, grouped with their children.
func (s *GuardiansService) siblings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT g.id, g.first_name || ' ' || g.last_name, COALESCE(g.phone,'')
		FROM guardians g
		JOIN student_guardians sg ON sg.guardian_id = g.id
		JOIN students s ON s.id = sg.student_id AND s.is_active = TRUE
		GROUP BY g.id, g.first_name, g.last_name, g.phone
		HAVING COUNT(sg.student_id) >= 2
		ORDER BY g.last_name, g.first_name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type siblingGroup struct {
		GuardianID string                   `json:"guardian_id"`
		FullName   string                   `json:"full_name"`
		Phone      string                   `json:"phone"`
		Students   []map[string]interface{} `json:"students"`
	}
	var groups []*siblingGroup
	var guardianIDs []uuid.UUID
	byID := map[uuid.UUID]*siblingGroup{}
	for rows.Next() {
		var gid uuid.UUID
		var name, phone string
		if rows.Scan(&gid, &name, &phone) == nil {
			g := &siblingGroup{GuardianID: gid.String(), FullName: name, Phone: phone, Students: []map[string]interface{}{}}
			guardianIDs = append(guardianIDs, gid)
			byID[gid] = g
			groups = append(groups, g)
		}
	}
	if len(guardianIDs) > 0 {
		srows, err := s.pool.Query(r.Context(), `
			SELECT sg.guardian_id, s.id, s.full_name, s.admission_no, COALESCE(cg.name,'')
			FROM student_guardians sg
			JOIN students s ON s.id = sg.student_id AND s.is_active = TRUE
			LEFT JOIN class_groups cg ON cg.id = s.current_class_id
			WHERE sg.guardian_id = ANY($1)
			ORDER BY sg.guardian_id, s.last_name, s.first_name`, guardianIDs)
		if err == nil {
			defer srows.Close()
			for srows.Next() {
				var gid, sid uuid.UUID
				var name, admNo, className string
				if srows.Scan(&gid, &sid, &name, &admNo, &className) == nil {
					if g, ok := byID[gid]; ok {
						g.Students = append(g.Students, map[string]interface{}{
							"student_id": sid, "full_name": name, "admission_no": admNo, "class_name": className,
						})
					}
				}
			}
		}
	}
	out := make([]siblingGroup, len(groups))
	for i, g := range groups {
		out[i] = *g
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *GuardiansService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM guardians WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// createLogin is what actually makes "parents see their children's work/fees
// via the app" possible at all: a guardian record on its own has no way to
// log in (guardians.user_id was always NULL — nothing ever created a user
// account for one), so a parent portal login never existed. This creates a
// PARENT-role user and links it via user_id, reusing the students.go
// registration email/phone as a starting point when the guardian has none.
func (s *GuardiansService) createLogin(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	g, err := scanGuardian(s.pool.QueryRow(r.Context(), `SELECT `+guardianCols+` FROM guardians WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "guardian not found")
		return
	}
	if g.UserID != nil {
		httpx.ErrConflict(w, "this guardian already has a login")
		return
	}
	if g.Email == "" {
		httpx.ErrBadRequest(w, "guardian has no email on file — add one before creating a login")
		return
	}

	plainPassword := generateTempPassword()
	hash, err := crypto.HashPassword(plainPassword)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	userID := uuid.New()
	_, err = s.pool.Exec(r.Context(), `
		INSERT INTO users (id, email, password, first_name, last_name, phone, role, is_staff,
			is_active, must_change_password, date_joined, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'parent', false, true, true, NOW(), NOW(), NOW())`,
		userID, g.Email, hash, g.FirstName, g.LastName, g.Phone)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			httpx.ErrConflict(w, "a user with this email already exists")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	if _, err := s.pool.Exec(r.Context(), `UPDATE guardians SET user_id = $1 WHERE id = $2`, userID, id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{
		"user_id": userID, "temporary_password": plainPassword,
	})
}
