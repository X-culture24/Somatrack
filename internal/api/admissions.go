package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/rpc"
)

// AdmissionsService implements the real admission-application workflow:
// submit → review → accept/reject → enroll (which creates the actual
// Student + Guardian records). The previous frontend posted fields like
// pupil_first_name/seeking_grade ("Grade 1" as a string) that don't exist
// on admission_applications at all — the real column is applying_for_grade,
// an integer 1-13 — so every field name and the status vocabulary below
// matches the actual table exactly.
type AdmissionsService struct {
	pool    *pgxpool.Pool
	finance *rpc.FinanceClient
}

func NewAdmissionsService(pool *pgxpool.Pool, finance *rpc.FinanceClient) *AdmissionsService {
	return &AdmissionsService{pool: pool, finance: finance}
}

func (s *AdmissionsService) RegisterRoutes(r chi.Router) {
	r.Route("/admissions/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Post("/", s.create)
		r.Get("/{id}/", s.get)
		r.Put("/{id}/", s.update)
		r.Patch("/{id}/", s.update)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.delete)).ServeHTTP)
		r.Post("/{id}/enroll/", s.enroll)
	})
}

// Valid admission_applications.status values (matches the DB CHECK
// constraint exactly): pending, reviewing, shortlisted, interview,
// accepted, rejected, deferred, admitted.
type AdmissionApplication struct {
	ID                 uuid.UUID  `json:"id"`
	FirstName          string     `json:"first_name" validate:"required"`
	MiddleName         string     `json:"middle_name"`
	LastName           string     `json:"last_name" validate:"required"`
	Gender             string     `json:"gender" validate:"required,oneof=male female other"`
	DateOfBirth        *string    `json:"date_of_birth,omitempty"`
	// No "required" — PP2 is grade 0 (Playgroup=-2, PP1=-1, PP2=0, Grade N=N,
	// same convention as fee_structures/class_groups), and Go's validator
	// treats a plain int's zero value as "not provided".
	ApplyingForGrade   int        `json:"applying_for_grade" validate:"min=-2,max=13"`
	ApplyingForYear    int        `json:"applying_for_year" validate:"required"`
	ApplyingForTerm    int        `json:"applying_for_term" validate:"required,oneof=1 2 3"`
	FormerSchool       string     `json:"former_school"`
	KcpeMarks          *int       `json:"kcpe_marks,omitempty"`
	GuardianName       string     `json:"guardian_name" validate:"required"`
	GuardianPhone      string     `json:"guardian_phone" validate:"required"`
	GuardianEmail      string     `json:"guardian_email"`
	GuardianRelation   string     `json:"guardian_relation"`
	ResidentialAddress string     `json:"residential_address"`
	Status             string     `json:"status" validate:"omitempty,oneof=pending reviewing shortlisted interview accepted rejected deferred admitted"`
	ReviewedByID       *uuid.UUID `json:"reviewed_by_id,omitempty"`
	AdmissionDate      *string    `json:"admission_date,omitempty"`
	Notes              string     `json:"notes"`
}

const admissionCols = `id, first_name, middle_name, last_name, gender, date_of_birth::text,
	applying_for_grade, applying_for_year, applying_for_term, former_school, kcpe_marks,
	guardian_name, guardian_phone, COALESCE(guardian_email,''), guardian_relation, residential_address,
	status, reviewed_by_id, admission_date::text, notes`

func scanAdmission(row interface {
	Scan(dest ...interface{}) error
}) (AdmissionApplication, error) {
	var a AdmissionApplication
	err := row.Scan(&a.ID, &a.FirstName, &a.MiddleName, &a.LastName, &a.Gender, &a.DateOfBirth,
		&a.ApplyingForGrade, &a.ApplyingForYear, &a.ApplyingForTerm, &a.FormerSchool, &a.KcpeMarks,
		&a.GuardianName, &a.GuardianPhone, &a.GuardianEmail, &a.GuardianRelation, &a.ResidentialAddress,
		&a.Status, &a.ReviewedByID, &a.AdmissionDate, &a.Notes)
	return a, err
}

func (s *AdmissionsService) list(w http.ResponseWriter, r *http.Request) {
	where := "true"
	args := []interface{}{}
	if v := r.URL.Query().Get("status"); v != "" {
		where = "status = $1"
		args = append(args, v)
	}
	rows, err := s.pool.Query(r.Context(), `SELECT `+admissionCols+` FROM admission_applications WHERE `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []AdmissionApplication{}
	for rows.Next() {
		a, err := scanAdmission(rows)
		if err != nil {
			continue
		}
		out = append(out, a)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AdmissionsService) create(w http.ResponseWriter, r *http.Request) {
	var body AdmissionApplication
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.Status == "" {
		body.Status = "pending"
	}
	if body.GuardianRelation == "" {
		body.GuardianRelation = "parent"
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO admission_applications (first_name, middle_name, last_name, gender, date_of_birth,
			applying_for_grade, applying_for_year, applying_for_term, former_school, kcpe_marks,
			guardian_name, guardian_phone, guardian_email, guardian_relation, residential_address, status, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) RETURNING id`,
		body.FirstName, body.MiddleName, body.LastName, body.Gender, body.DateOfBirth,
		body.ApplyingForGrade, body.ApplyingForYear, body.ApplyingForTerm, body.FormerSchool, body.KcpeMarks,
		body.GuardianName, body.GuardianPhone, body.GuardianEmail, body.GuardianRelation, body.ResidentialAddress,
		body.Status, body.Notes,
	).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *AdmissionsService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	a, err := scanAdmission(s.pool.QueryRow(r.Context(), `SELECT `+admissionCols+` FROM admission_applications WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "admission application not found")
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}

func (s *AdmissionsService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	// Plain decode, no full-struct validation: this handler is shared by
	// PUT and PATCH, and BindAndValidate's `required` tags (needed for a
	// real Create) reject any partial PATCH body — e.g. the common
	// {"status":"accepted"} review-workflow step — before it ever reaches
	// the COALESCE-safe SQL below. Enum-like columns (gender, status,
	// applying_for_term/grade) are still guarded by DB CHECK constraints.
	var body AdmissionApplication
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE admission_applications SET
			first_name=COALESCE(NULLIF($1,''), first_name),
			middle_name=COALESCE(NULLIF($2,''), middle_name),
			last_name=COALESCE(NULLIF($3,''), last_name),
			gender=COALESCE(NULLIF($4,''), gender),
			date_of_birth=COALESCE($5, date_of_birth),
			applying_for_grade=COALESCE(NULLIF($6,0), applying_for_grade),
			applying_for_year=COALESCE(NULLIF($7,0), applying_for_year),
			applying_for_term=COALESCE(NULLIF($8,0), applying_for_term),
			former_school=COALESCE(NULLIF($9,''), former_school),
			kcpe_marks=COALESCE($10, kcpe_marks),
			guardian_name=COALESCE(NULLIF($11,''), guardian_name),
			guardian_phone=COALESCE(NULLIF($12,''), guardian_phone),
			guardian_email=COALESCE(NULLIF($13,''), guardian_email),
			guardian_relation=COALESCE(NULLIF($14,''), guardian_relation),
			residential_address=COALESCE(NULLIF($15,''), residential_address),
			status=COALESCE(NULLIF($16,''), status),
			notes=COALESCE(NULLIF($17,''), notes),
			updated_at=NOW()
		WHERE id = $18`,
		body.FirstName, body.MiddleName, body.LastName, body.Gender, body.DateOfBirth,
		body.ApplyingForGrade, body.ApplyingForYear, body.ApplyingForTerm, body.FormerSchool, body.KcpeMarks,
		body.GuardianName, body.GuardianPhone, body.GuardianEmail, body.GuardianRelation, body.ResidentialAddress,
		body.Status, body.Notes, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	a, err := scanAdmission(s.pool.QueryRow(r.Context(), `SELECT `+admissionCols+` FROM admission_applications WHERE id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "admission application not found")
		return
	}
	httpx.JSON(w, http.StatusOK, a)
}

func (s *AdmissionsService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM admission_applications WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

type EnrollRequest struct {
	AdmissionNo string       `json:"admission_no" validate:"required"`
	ClassID     OptionalUUID `json:"class_id,omitempty"`
	StreamID    OptionalUUID `json:"stream_id,omitempty"`
}

// enroll converts an admission application into a real Student + Guardian
// record and marks the application "admitted".
func (s *AdmissionsService) enroll(w http.ResponseWriter, r *http.Request) {
	appID, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body EnrollRequest
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	a, err := scanAdmission(s.pool.QueryRow(r.Context(), `SELECT `+admissionCols+` FROM admission_applications WHERE id = $1`, appID))
	if err != nil {
		httpx.ErrNotFound(w, "admission application not found")
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())

	fullName := a.FirstName
	if a.MiddleName != "" {
		fullName += " " + a.MiddleName
	}
	fullName += " " + a.LastName
	searchName := a.FirstName + " " + a.LastName

	var studentID uuid.UUID
	err = tx.QueryRow(r.Context(), `
		INSERT INTO students (admission_no, first_name, middle_name, last_name, full_name, search_name, gender,
			date_of_birth, current_class_id, stream_id, admission_date, former_school, kcpe_marks,
			residential_address, status, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),$11,$12,$13,'active',true) RETURNING id`,
		body.AdmissionNo, a.FirstName, a.MiddleName, a.LastName, fullName, searchName, a.Gender,
		a.DateOfBirth, body.ClassID.Ptr(), body.StreamID.Ptr(), a.FormerSchool, a.KcpeMarks, a.ResidentialAddress,
	).Scan(&studentID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}

	guardianNames := splitGuardianName(a.GuardianName)
	relation := a.GuardianRelation
	if relation == "" {
		relation = "parent"
	}
	if err := linkOrCreateGuardianTx(r.Context(), tx, studentID, GuardianLink{
		FirstName: guardianNames[0], LastName: guardianNames[1],
		Phone: a.GuardianPhone, Email: a.GuardianEmail, Relation: relation,
		IsPrimary: true, HasCustody: true, ContactPriority: 1,
	}); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}

	if _, err := tx.Exec(r.Context(), `
		UPDATE admission_applications SET status='admitted', admission_date=NOW(), updated_at=NOW() WHERE id = $1`, appID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	// Bill immediately if a class was assigned at enroll time — a fee
	// balance should exist from the moment enrollment completes, not wait
	// for a separate manual billing run. Non-fatal: a grade with no fee
	// structure configured yet shouldn't block the enrollment itself.
	if body.ClassID.Valid && s.finance != nil {
		if _, err := s.finance.BillStudent(r.Context(), studentID); err != nil {
			log.Printf("[admissions] auto-bill for %s: %v", studentID, err)
		}
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{
		"status": "enrolled", "student_id": studentID, "admission_no": body.AdmissionNo,
	})
}

func splitGuardianName(name string) [2]string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == ' ' {
			return [2]string{name[:i], name[i+1:]}
		}
	}
	return [2]string{name, name}
}
