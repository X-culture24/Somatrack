package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// AcademicsService implements real CRUD for the school-setup reference data
// (academic years, terms, streams, subjects, classes, grade requirements)
// that registerPlaceholderCRUD used to stub out. Kept directly in the
// gateway (like auth.go/dashboard) rather than as its own microservice —
// these are simple, single-table-ish resources with no async workflow.
type AcademicsService struct {
	pool *pgxpool.Pool
}

func NewAcademicsService(pool *pgxpool.Pool) *AcademicsService {
	return &AcademicsService{pool: pool}
}

func (s *AcademicsService) RegisterRoutes(r chi.Router) {
	r.Route("/academic-years/", func(r chi.Router) {
		r.Get("/", s.listAcademicYears)
		r.Post("/", adminOnly()(http.HandlerFunc(s.createAcademicYear)).ServeHTTP)
		r.Get("/{id}/", s.getAcademicYear)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.updateAcademicYear)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.updateAcademicYear)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.deleteAcademicYear)).ServeHTTP)
	})
	r.Route("/terms/", func(r chi.Router) {
		r.Get("/", s.listTerms)
		r.Post("/", adminOnly()(http.HandlerFunc(s.createTerm)).ServeHTTP)
		r.Get("/{id}/", s.getTerm)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.updateTerm)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.updateTerm)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.deleteTerm)).ServeHTTP)
	})
	r.Route("/streams/", func(r chi.Router) {
		r.Get("/", s.listStreams)
		r.Post("/", adminOnly()(http.HandlerFunc(s.createStream)).ServeHTTP)
		r.Get("/{id}/", s.getStream)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.updateStream)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.updateStream)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.deleteStream)).ServeHTTP)
	})
	r.Route("/subjects/", func(r chi.Router) {
		r.Get("/", s.listSubjects)
		r.Post("/", adminOnly()(http.HandlerFunc(s.createSubject)).ServeHTTP)
		r.Get("/{id}/", s.getSubject)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.updateSubject)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.updateSubject)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.deleteSubject)).ServeHTTP)
	})
	r.Route("/classes/", func(r chi.Router) {
		r.Get("/", s.listClasses)
		r.Post("/", adminOnly()(http.HandlerFunc(s.createClass)).ServeHTTP)
		r.Get("/{id}/", s.getClass)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.updateClass)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.updateClass)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.deleteClass)).ServeHTTP)
	})
	r.Route("/grade-requirements/", func(r chi.Router) {
		r.Get("/", s.listGradeRequirements)
		r.Post("/", adminOnly()(http.HandlerFunc(s.createGradeRequirement)).ServeHTTP)
		r.Get("/{id}/", s.getGradeRequirement)
		r.Put("/{id}/", adminOnly()(http.HandlerFunc(s.updateGradeRequirement)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.updateGradeRequirement)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.deleteGradeRequirement)).ServeHTTP)
	})
}

func pageParams(r *http.Request) (page, pageSize, offset int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	return page, pageSize, (page - 1) * pageSize
}

func idParam(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}

func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " AND "
		}
		out += p
	}
	return out
}

// ---------------------------------------------------------------- Academic Years

type AcademicYear struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name" validate:"required"`
	StartDate string    `json:"start_date" validate:"required"`
	EndDate   string    `json:"end_date" validate:"required"`
	IsActive  bool      `json:"is_active"`
}

func (s *AcademicsService) listAcademicYears(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id, name, start_date::text, end_date::text, is_active
		FROM academic_years ORDER BY start_date DESC`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []AcademicYear{}
	for rows.Next() {
		var y AcademicYear
		if err := rows.Scan(&y.ID, &y.Name, &y.StartDate, &y.EndDate, &y.IsActive); err != nil {
			continue
		}
		out = append(out, y)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AcademicsService) createAcademicYear(w http.ResponseWriter, r *http.Request) {
	var body AcademicYear
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO academic_years (name, start_date, end_date, is_active)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		body.Name, body.StartDate, body.EndDate, body.IsActive).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *AcademicsService) getAcademicYear(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var y AcademicYear
	err = s.pool.QueryRow(r.Context(), `
		SELECT id, name, start_date::text, end_date::text, is_active
		FROM academic_years WHERE id = $1`, id).Scan(&y.ID, &y.Name, &y.StartDate, &y.EndDate, &y.IsActive)
	if err != nil {
		httpx.ErrNotFound(w, "academic year not found")
		return
	}
	httpx.JSON(w, http.StatusOK, y)
}

func (s *AcademicsService) updateAcademicYear(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	// Plain decode + COALESCE, not BindAndValidate against the full Create
	// struct: this handler is shared by PUT and PATCH, and full-struct
	// `required` validation rejected any partial PATCH (e.g. just flipping
	// is_active) before it ever reached the DB — the years were "preloaded"
	// precisely so only editing, never re-creating, would be needed here.
	var body struct {
		Name      *string `json:"name,omitempty"`
		StartDate *string `json:"start_date,omitempty"`
		EndDate   *string `json:"end_date,omitempty"`
		IsActive  *bool   `json:"is_active,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE academic_years SET
			name=COALESCE(NULLIF($1,''), name), start_date=COALESCE($2, start_date),
			end_date=COALESCE($3, end_date), is_active=COALESCE($4, is_active), updated_at=NOW()
		WHERE id = $5`, body.Name, body.StartDate, body.EndDate, body.IsActive, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	var y AcademicYear
	if err := s.pool.QueryRow(r.Context(), `
		SELECT id, name, start_date::text, end_date::text, is_active FROM academic_years WHERE id = $1`, id).
		Scan(&y.ID, &y.Name, &y.StartDate, &y.EndDate, &y.IsActive); err != nil {
		httpx.ErrNotFound(w, "academic year not found")
		return
	}
	httpx.JSON(w, http.StatusOK, y)
}

func (s *AcademicsService) deleteAcademicYear(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM academic_years WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// ---------------------------------------------------------------- Terms

type Term struct {
	ID             uuid.UUID `json:"id"`
	AcademicYearID uuid.UUID `json:"academic_year_id" validate:"required,uuid"`
	Name           string    `json:"name" validate:"required"`
	TermNumber     int       `json:"term_number" validate:"required,oneof=1 2 3"`
	StartDate      string    `json:"start_date" validate:"required"`
	EndDate        string    `json:"end_date" validate:"required"`
	IsCurrent      bool      `json:"is_current"`
}

func (s *AcademicsService) listTerms(w http.ResponseWriter, r *http.Request) {
	where := "true"
	args := []interface{}{}
	if ay := r.URL.Query().Get("academic_year_id"); ay != "" {
		if id, err := uuid.Parse(ay); err == nil {
			where = "academic_year_id = $1"
			args = append(args, id)
		}
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT id, academic_year_id, name, term_number, start_date::text, end_date::text, is_current
		FROM terms WHERE `+where+` ORDER BY start_date DESC`, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Term{}
	for rows.Next() {
		var t Term
		if err := rows.Scan(&t.ID, &t.AcademicYearID, &t.Name, &t.TermNumber, &t.StartDate, &t.EndDate, &t.IsCurrent); err != nil {
			continue
		}
		out = append(out, t)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AcademicsService) createTerm(w http.ResponseWriter, r *http.Request) {
	var body Term
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO terms (academic_year_id, name, term_number, start_date, end_date, is_current)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		body.AcademicYearID, body.Name, body.TermNumber, body.StartDate, body.EndDate, body.IsCurrent).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *AcademicsService) getTerm(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var t Term
	err = s.pool.QueryRow(r.Context(), `
		SELECT id, academic_year_id, name, term_number, start_date::text, end_date::text, is_current
		FROM terms WHERE id = $1`, id).Scan(&t.ID, &t.AcademicYearID, &t.Name, &t.TermNumber, &t.StartDate, &t.EndDate, &t.IsCurrent)
	if err != nil {
		httpx.ErrNotFound(w, "term not found")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (s *AcademicsService) updateTerm(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		AcademicYearID *uuid.UUID `json:"academic_year_id,omitempty"`
		Name           *string    `json:"name,omitempty"`
		TermNumber     *int       `json:"term_number,omitempty"`
		StartDate      *string    `json:"start_date,omitempty"`
		EndDate        *string    `json:"end_date,omitempty"`
		IsCurrent      *bool      `json:"is_current,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE terms SET
			academic_year_id=COALESCE($1, academic_year_id), name=COALESCE(NULLIF($2,''), name),
			term_number=COALESCE(NULLIF($3,0), term_number), start_date=COALESCE($4, start_date),
			end_date=COALESCE($5, end_date), is_current=COALESCE($6, is_current), updated_at=NOW()
		WHERE id = $7`, body.AcademicYearID, body.Name, body.TermNumber, body.StartDate, body.EndDate, body.IsCurrent, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	var t Term
	if err := s.pool.QueryRow(r.Context(), `
		SELECT id, academic_year_id, name, term_number, start_date::text, end_date::text, is_current
		FROM terms WHERE id = $1`, id).Scan(&t.ID, &t.AcademicYearID, &t.Name, &t.TermNumber, &t.StartDate, &t.EndDate, &t.IsCurrent); err != nil {
		httpx.ErrNotFound(w, "term not found")
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (s *AcademicsService) deleteTerm(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM terms WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// ---------------------------------------------------------------- Streams

type Stream struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name" validate:"required"`
	Code string    `json:"code" validate:"required"`
}

func (s *AcademicsService) listStreams(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `SELECT id, name, code FROM streams ORDER BY name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Stream{}
	for rows.Next() {
		var st Stream
		if err := rows.Scan(&st.ID, &st.Name, &st.Code); err != nil {
			continue
		}
		out = append(out, st)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AcademicsService) createStream(w http.ResponseWriter, r *http.Request) {
	var body Stream
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO streams (name, code) VALUES ($1, $2) RETURNING id`, body.Name, body.Code).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *AcademicsService) getStream(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var st Stream
	err = s.pool.QueryRow(r.Context(), `SELECT id, name, code FROM streams WHERE id = $1`, id).Scan(&st.ID, &st.Name, &st.Code)
	if err != nil {
		httpx.ErrNotFound(w, "stream not found")
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (s *AcademicsService) updateStream(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		Name *string `json:"name,omitempty"`
		Code *string `json:"code,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE streams SET name=COALESCE(NULLIF($1,''), name), code=COALESCE(NULLIF($2,''), code), updated_at=NOW()
		WHERE id = $3`, body.Name, body.Code, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	var st Stream
	if err := s.pool.QueryRow(r.Context(), `SELECT id, name, code FROM streams WHERE id = $1`, id).Scan(&st.ID, &st.Name, &st.Code); err != nil {
		httpx.ErrNotFound(w, "stream not found")
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (s *AcademicsService) deleteStream(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM streams WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// ---------------------------------------------------------------- Subjects

type Subject struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name" validate:"required"`
	Code         string    `json:"code" validate:"required"`
	Description  string    `json:"description"`
	LearningArea string    `json:"learning_area"`
	IsActive     bool      `json:"is_active"`
}

func (s *AcademicsService) listSubjects(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id, name, code, description, learning_area, is_active FROM subjects ORDER BY name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Subject{}
	for rows.Next() {
		var sub Subject
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.Code, &sub.Description, &sub.LearningArea, &sub.IsActive); err != nil {
			continue
		}
		out = append(out, sub)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AcademicsService) createSubject(w http.ResponseWriter, r *http.Request) {
	var body Subject
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.LearningArea == "" {
		body.LearningArea = "General"
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO subjects (name, code, description, learning_area, is_active) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		body.Name, body.Code, body.Description, body.LearningArea, body.IsActive).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *AcademicsService) getSubject(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var sub Subject
	err = s.pool.QueryRow(r.Context(), `
		SELECT id, name, code, description, learning_area, is_active FROM subjects WHERE id = $1`, id).
		Scan(&sub.ID, &sub.Name, &sub.Code, &sub.Description, &sub.LearningArea, &sub.IsActive)
	if err != nil {
		httpx.ErrNotFound(w, "subject not found")
		return
	}
	httpx.JSON(w, http.StatusOK, sub)
}

func (s *AcademicsService) updateSubject(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		Name         *string `json:"name,omitempty"`
		Code         *string `json:"code,omitempty"`
		Description  *string `json:"description,omitempty"`
		LearningArea *string `json:"learning_area,omitempty"`
		IsActive     *bool   `json:"is_active,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE subjects SET
			name=COALESCE(NULLIF($1,''), name), code=COALESCE(NULLIF($2,''), code),
			description=COALESCE($3, description), learning_area=COALESCE(NULLIF($4,''), learning_area),
			is_active=COALESCE($5, is_active), updated_at=NOW()
		WHERE id = $6`, body.Name, body.Code, body.Description, body.LearningArea, body.IsActive, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	var sub Subject
	if err := s.pool.QueryRow(r.Context(), `SELECT id, name, code, description, learning_area, is_active FROM subjects WHERE id = $1`, id).
		Scan(&sub.ID, &sub.Name, &sub.Code, &sub.Description, &sub.LearningArea, &sub.IsActive); err != nil {
		httpx.ErrNotFound(w, "subject not found")
		return
	}
	httpx.JSON(w, http.StatusOK, sub)
}

func (s *AcademicsService) deleteSubject(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM subjects WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// ---------------------------------------------------------------- Classes (class_groups)

type ClassGroup struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name" validate:"required"`
	// No "required" — PP2 is grade 0 (Playgroup=-2, PP1=-1, PP2=0, Grade N=N),
	// and Go's validator treats a plain int's zero value as "not provided".
	Grade          int          `json:"grade" validate:"min=-2,max=13"`
	StreamID       OptionalUUID `json:"stream_id,omitempty"`
	StreamName     string       `json:"stream_name,omitempty"`
	ClassTeacherID OptionalUUID `json:"class_teacher_id,omitempty"`
	AcademicYearID uuid.UUID    `json:"academic_year_id" validate:"required,uuid"`
	Capacity       int          `json:"capacity"`
}

func (s *AcademicsService) listClasses(w http.ResponseWriter, r *http.Request) {
	where := []string{"true"}
	args := []interface{}{}
	argIdx := 1
	if ay := r.URL.Query().Get("academic_year_id"); ay != "" {
		if id, err := uuid.Parse(ay); err == nil {
			where = append(where, "cg.academic_year_id = $"+strconv.Itoa(argIdx))
			args = append(args, id)
			argIdx++
		}
	}
	if g := r.URL.Query().Get("grade"); g != "" {
		if gi, err := strconv.Atoi(g); err == nil {
			where = append(where, "cg.grade = $"+strconv.Itoa(argIdx))
			args = append(args, gi)
			argIdx++
		}
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT cg.id, cg.name, cg.grade, cg.stream_id, COALESCE(st.name, ''), cg.class_teacher_id, cg.academic_year_id, cg.capacity,
		       COALESCE(sc.student_count, 0),
		       COALESCE(fin.total_billed, 0), COALESCE(fin.total_paid, 0), COALESCE(fin.total_outstanding, 0)
		FROM class_groups cg
		LEFT JOIN streams st ON st.id = cg.stream_id
		LEFT JOIN (
			SELECT current_class_id, COUNT(*) AS student_count FROM students
			WHERE is_active = TRUE GROUP BY current_class_id
		) sc ON sc.current_class_id = cg.id
		LEFT JOIN (
			SELECT s.current_class_id, SUM(i.total_amount) AS total_billed,
			       SUM(i.total_paid) AS total_paid, SUM(i.balance) AS total_outstanding
			FROM students s JOIN invoices i ON i.student_id = s.id
			GROUP BY s.current_class_id
		) fin ON fin.current_class_id = cg.id
		WHERE `+joinAnd(where)+`
		ORDER BY cg.grade, st.name`, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type classGroupWithStats struct {
		ClassGroup
		StudentCount     int     `json:"student_count"`
		TotalBilled      float64 `json:"total_billed"`
		TotalPaid        float64 `json:"total_paid"`
		TotalOutstanding float64 `json:"total_outstanding"`
	}
	out := []classGroupWithStats{}
	for rows.Next() {
		var c classGroupWithStats
		var streamID, teacherID *uuid.UUID
		if err := rows.Scan(&c.ID, &c.Name, &c.Grade, &streamID, &c.StreamName, &teacherID, &c.AcademicYearID, &c.Capacity,
			&c.StudentCount, &c.TotalBilled, &c.TotalPaid, &c.TotalOutstanding); err != nil {
			continue
		}
		c.StreamID = OptionalUUIDFromPtr(streamID)
		c.ClassTeacherID = OptionalUUIDFromPtr(teacherID)
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AcademicsService) createClass(w http.ResponseWriter, r *http.Request) {
	var body ClassGroup
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.Capacity <= 0 {
		body.Capacity = 40
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO class_groups (name, grade, stream_id, class_teacher_id, academic_year_id, capacity)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		body.Name, body.Grade, body.StreamID.Ptr(), body.ClassTeacherID.Ptr(), body.AcademicYearID, body.Capacity).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func getClassByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (ClassGroup, error) {
	var c ClassGroup
	var streamID, teacherID *uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT cg.id, cg.name, cg.grade, cg.stream_id, COALESCE(st.name, ''), cg.class_teacher_id, cg.academic_year_id, cg.capacity
		FROM class_groups cg LEFT JOIN streams st ON st.id = cg.stream_id
		WHERE cg.id = $1`, id).Scan(&c.ID, &c.Name, &c.Grade, &streamID, &c.StreamName, &teacherID, &c.AcademicYearID, &c.Capacity)
	c.StreamID = OptionalUUIDFromPtr(streamID)
	c.ClassTeacherID = OptionalUUIDFromPtr(teacherID)
	return c, err
}

func (s *AcademicsService) getClass(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	c, err := getClassByID(r.Context(), s.pool, id)
	if err != nil {
		httpx.ErrNotFound(w, "class not found")
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (s *AcademicsService) updateClass(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	// Grade uses a *int (not COALESCE(NULLIF(x,0),...)) because 0 is a
	// legitimate grade (PP2) here, unlike term_number/other int fields
	// elsewhere where 0 can safely mean "not provided".
	var body struct {
		Name           *string       `json:"name,omitempty"`
		Grade          *int          `json:"grade,omitempty"`
		StreamID       *OptionalUUID `json:"stream_id,omitempty"`
		ClassTeacherID *OptionalUUID `json:"class_teacher_id,omitempty"`
		AcademicYearID *uuid.UUID    `json:"academic_year_id,omitempty"`
		Capacity       *int          `json:"capacity,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	var streamPtr, teacherPtr *uuid.UUID
	if body.StreamID != nil {
		streamPtr = body.StreamID.Ptr()
	}
	if body.ClassTeacherID != nil {
		teacherPtr = body.ClassTeacherID.Ptr()
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE class_groups SET
			name=COALESCE(NULLIF($1,''), name), grade=COALESCE($2, grade),
			stream_id=CASE WHEN $3::boolean THEN $4 ELSE stream_id END,
			class_teacher_id=CASE WHEN $5::boolean THEN $6 ELSE class_teacher_id END,
			academic_year_id=COALESCE($7, academic_year_id), capacity=COALESCE($8, capacity), updated_at=NOW()
		WHERE id = $9`,
		body.Name, body.Grade, body.StreamID != nil, streamPtr, body.ClassTeacherID != nil, teacherPtr,
		body.AcademicYearID, body.Capacity, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	c, err := getClassByID(r.Context(), s.pool, id)
	if err != nil {
		httpx.ErrNotFound(w, "class not found")
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (s *AcademicsService) deleteClass(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM class_groups WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// ---------------------------------------------------------------- Grade Requirements

type GradeRequirement struct {
	ID          uuid.UUID `json:"id"`
	Grade       int       `json:"grade" validate:"required,min=1,max=13"`
	MinScore    float64   `json:"min_score"`
	MaxScore    float64   `json:"max_score"`
	GradeLetter string    `json:"grade_letter" validate:"required"`
	GradePoint  float64   `json:"grade_point"`
	Remarks     string    `json:"remarks"`
}

func (s *AcademicsService) listGradeRequirements(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id, grade, min_score, max_score, grade_letter, grade_point, remarks
		FROM grade_requirements ORDER BY grade, min_score DESC`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []GradeRequirement{}
	for rows.Next() {
		var g GradeRequirement
		if err := rows.Scan(&g.ID, &g.Grade, &g.MinScore, &g.MaxScore, &g.GradeLetter, &g.GradePoint, &g.Remarks); err != nil {
			continue
		}
		out = append(out, g)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AcademicsService) createGradeRequirement(w http.ResponseWriter, r *http.Request) {
	var body GradeRequirement
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO grade_requirements (grade, min_score, max_score, grade_letter, grade_point, remarks)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		body.Grade, body.MinScore, body.MaxScore, body.GradeLetter, body.GradePoint, body.Remarks).Scan(&body.ID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, body)
}

func (s *AcademicsService) getGradeRequirement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var g GradeRequirement
	err = s.pool.QueryRow(r.Context(), `
		SELECT id, grade, min_score, max_score, grade_letter, grade_point, remarks
		FROM grade_requirements WHERE id = $1`, id).
		Scan(&g.ID, &g.Grade, &g.MinScore, &g.MaxScore, &g.GradeLetter, &g.GradePoint, &g.Remarks)
	if err != nil {
		httpx.ErrNotFound(w, "grade requirement not found")
		return
	}
	httpx.JSON(w, http.StatusOK, g)
}

func (s *AcademicsService) updateGradeRequirement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		Grade       *int     `json:"grade,omitempty"`
		MinScore    *float64 `json:"min_score,omitempty"`
		MaxScore    *float64 `json:"max_score,omitempty"`
		GradeLetter *string  `json:"grade_letter,omitempty"`
		GradePoint  *float64 `json:"grade_point,omitempty"`
		Remarks     *string  `json:"remarks,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE grade_requirements SET
			grade=COALESCE($1, grade), min_score=COALESCE($2, min_score), max_score=COALESCE($3, max_score),
			grade_letter=COALESCE(NULLIF($4,''), grade_letter), grade_point=COALESCE($5, grade_point),
			remarks=COALESCE($6, remarks)
		WHERE id = $7`, body.Grade, body.MinScore, body.MaxScore, body.GradeLetter, body.GradePoint, body.Remarks, id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	var gr GradeRequirement
	if err := s.pool.QueryRow(r.Context(), `
		SELECT id, grade, min_score, max_score, grade_letter, grade_point, remarks FROM grade_requirements WHERE id = $1`, id).
		Scan(&gr.ID, &gr.Grade, &gr.MinScore, &gr.MaxScore, &gr.GradeLetter, &gr.GradePoint, &gr.Remarks); err != nil {
		httpx.ErrNotFound(w, "grade requirement not found")
		return
	}
	httpx.JSON(w, http.StatusOK, gr)
}

func (s *AcademicsService) deleteGradeRequirement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM grade_requirements WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// ================================================================ Exam Threads

func (s *AcademicsService) RegisterExamRoutes(r chi.Router) {
	// Use /exams/ prefix to avoid conflict with the generic resource engine's /exam-threads/
	r.Route("/exams/threads/", func(r chi.Router) {
		r.Get("/", s.listExamThreads)
		r.Post("/", s.createExamThread)
		r.Patch("/{id}/", s.updateExamThread)
		r.Delete("/{id}/", s.deleteExamThread)
		r.Post("/{id}/lock/", s.lockExamThread)
	})
	// Assignments decide who may enter marks, so only admins change them —
	// otherwise a teacher could assign themselves any subject.
	r.Route("/exams/class-subjects/", func(r chi.Router) {
		r.Get("/", s.listClassSubjects)
		r.Post("/", adminOnly()(http.HandlerFunc(s.upsertClassSubject)).ServeHTTP)
		r.Patch("/{id}/", adminOnly()(http.HandlerFunc(s.updateClassSubjectTeacher)).ServeHTTP)
	})
	r.Patch("/exams/classes/{id}/class-teacher/", adminOnly()(http.HandlerFunc(s.assignClassTeacher)).ServeHTTP)
}

func (s *AcademicsService) listExamThreads(w http.ResponseWriter, r *http.Request) {
	termFilter := r.URL.Query().Get("term_id")
	args := []interface{}{}
	where := "TRUE"
	if termFilter != "" {
		if id, err := uuid.Parse(termFilter); err == nil {
			where = "et.term_id = $1"
			args = append(args, id)
		}
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT et.id, et.name, et.term_id, t.name, et.curriculum,
		       et.cats_weight, et.main_exam_weight, et.is_locked, et.description,
		       COUNT(DISTINCT a.id)::int AS assessment_count,
		       COUNT(DISTINCT m.student_id)::int AS students_with_marks
		FROM exam_threads et
		JOIN terms t ON t.id = et.term_id
		LEFT JOIN assessments a ON a.exam_thread_id = et.id
		LEFT JOIN marks m ON m.assessment_id = a.id
		WHERE `+where+`
		GROUP BY et.id, et.name, et.term_id, t.name, et.curriculum,
		         et.cats_weight, et.main_exam_weight, et.is_locked, et.description
		ORDER BY et.name`, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, termID uuid.UUID
		var name, termName, curriculum, description string
		var catsW, mainW float64
		var locked bool
		var assessCount, studentsWithMarks int
		if rows.Scan(&id, &name, &termID, &termName, &curriculum, &catsW, &mainW, &locked, &description, &assessCount, &studentsWithMarks) == nil {
			out = append(out, map[string]interface{}{
				"id": id, "name": name, "term_id": termID, "term_name": termName,
				"curriculum": curriculum, "cats_weight": catsW, "main_exam_weight": mainW,
				"is_locked": locked, "description": description,
				"assessment_count": assessCount, "students_with_marks": studentsWithMarks,
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AcademicsService) createExamThread(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name           string    `json:"name" validate:"required"`
		TermID         uuid.UUID `json:"term_id" validate:"required"`
		Curriculum     string    `json:"curriculum"`
		CatsWeight     float64   `json:"cats_weight"`
		MainExamWeight float64   `json:"main_exam_weight"`
		Description    string    `json:"description"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.Curriculum == "" {
		body.Curriculum = "cbc"
	}
	if body.CatsWeight+body.MainExamWeight == 0 {
		body.MainExamWeight = 100
	}
	var id uuid.UUID
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO exam_threads (name, term_id, curriculum, cats_weight, main_exam_weight, description)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		body.Name, body.TermID, body.Curriculum, body.CatsWeight, body.MainExamWeight, body.Description,
	).Scan(&id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]interface{}{"id": id, "status": "created"})
}

func (s *AcademicsService) updateExamThread(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		Name           *string  `json:"name"`
		CatsWeight     *float64 `json:"cats_weight"`
		MainExamWeight *float64 `json:"main_exam_weight"`
		Description    *string  `json:"description"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE exam_threads SET
		  name             = COALESCE($2, name),
		  cats_weight      = COALESCE($3, cats_weight),
		  main_exam_weight = COALESCE($4, main_exam_weight),
		  description      = COALESCE($5, description)
		WHERE id = $1`, id, body.Name, body.CatsWeight, body.MainExamWeight, body.Description)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *AcademicsService) deleteExamThread(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var count int
	s.pool.QueryRow(r.Context(), `SELECT COUNT(*) FROM assessments WHERE exam_thread_id=$1`, id).Scan(&count)
	if count > 0 {
		httpx.ErrBadRequest(w, "cannot delete: this exam thread has assessments with marks")
		return
	}
	s.pool.Exec(r.Context(), `DELETE FROM exam_threads WHERE id=$1`, id)
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *AcademicsService) lockExamThread(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		Locked bool `json:"locked"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(), `UPDATE exam_threads SET is_locked=$2 WHERE id=$1`, id, body.Locked)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	status := "unlocked"
	if body.Locked {
		status = "locked"
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": status})
}

// ================================================================ Class-subject teacher assignment

func (s *AcademicsService) listClassSubjects(w http.ResponseWriter, r *http.Request) {
	classID, err := uuid.Parse(r.URL.Query().Get("class_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "class_id required")
		return
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT cs.id, cs.class_id, cg.name, cs.subject_id, s.name, cs.teacher_id,
		       COALESCE(u.first_name||' '||u.last_name, '') AS teacher_name,
		       COALESCE(u.email, ''), cs.is_elective
		FROM class_subjects cs
		JOIN class_groups cg ON cg.id = cs.class_id
		JOIN subjects s ON s.id = cs.subject_id
		LEFT JOIN users u ON u.id = cs.teacher_id
		WHERE cs.class_id = $1
		ORDER BY s.name`, classID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var csID, cID, sID uuid.UUID
		var cName, sName, tName, tEmail string
		var tID *uuid.UUID
		var isElective bool
		if rows.Scan(&csID, &cID, &cName, &sID, &sName, &tID, &tName, &tEmail, &isElective) == nil {
			out = append(out, map[string]interface{}{
				"id": csID, "class_id": cID, "class_name": cName,
				"subject_id": sID, "subject_name": sName,
				"teacher_id": tID, "teacher_name": tName, "teacher_email": tEmail,
				"is_elective": isElective,
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *AcademicsService) upsertClassSubject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClassID    uuid.UUID  `json:"class_id" validate:"required"`
		SubjectID  uuid.UUID  `json:"subject_id" validate:"required"`
		TeacherID  *uuid.UUID `json:"teacher_id"`
		IsElective bool       `json:"is_elective"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	var id uuid.UUID
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO class_subjects (class_id, subject_id, teacher_id, is_elective)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (class_id, subject_id) DO UPDATE
		  SET teacher_id  = EXCLUDED.teacher_id,
		      is_elective = EXCLUDED.is_elective,
		      updated_at  = NOW()
		RETURNING id`, body.ClassID, body.SubjectID, body.TeacherID, body.IsElective,
	).Scan(&id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"id": id, "status": "saved"})
}

func (s *AcademicsService) updateClassSubjectTeacher(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		TeacherID *uuid.UUID `json:"teacher_id"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(),
		`UPDATE class_subjects SET teacher_id=$2, updated_at=NOW() WHERE id=$1`, id, body.TeacherID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *AcademicsService) assignClassTeacher(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		TeacherID *uuid.UUID `json:"teacher_id"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	_, err = s.pool.Exec(r.Context(),
		`UPDATE class_groups SET class_teacher_id=$2, updated_at=NOW() WHERE id=$1`, id, body.TeacherID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
