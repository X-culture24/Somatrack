package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// ResultsService owns assessments, marks entry and the results trail — a
// student's results for every term since they joined. It replaced the
// placeholder /assessments/ and /marks/ stubs.
//
// Two rules matter for the trail:
//   - a result belongs to the class the exam was set for (assessments.class_id),
//     never the student's current class, so promotions don't rewrite history;
//   - scores are stored raw (out of total_marks) and always compared as
//     percentages.
type ResultsService struct {
	pool  *pgxpool.Pool
	links *LinksService // signs the transcript's QR code
}

func NewResultsService(pool *pgxpool.Pool, links *LinksService) *ResultsService {
	return &ResultsService{pool: pool, links: links}
}

func (s *ResultsService) RegisterRoutes(r chi.Router) {
	r.Route("/assessments/", func(r chi.Router) {
		r.Get("/", s.listAssessments)
		r.Post("/", s.createAssessment)
		r.Patch("/{id}/", s.updateAssessment)
		r.Put("/{id}/", s.updateAssessment)
		r.Delete("/{id}/", s.deleteAssessment)
		r.Get("/{id}/marksheet/", s.marksheet)
		r.Put("/{id}/marks/", s.saveMarks)
	})
	r.Get("/marks/", s.listMarks)
	r.Get("/results/class-performance/", s.classPerformance)
	r.Get("/results/students/{id}/history/", s.history)
	r.Get("/results/students/{id}/transcript/", s.transcript)
	r.Get("/pocket-money-balances/", pocketMoneyBalances(s.pool))
}

// ---------------------------------------------------------------------------
// Who may do what

func callerRole(r *http.Request) types.Role {
	v, _ := r.Context().Value("role").(types.Role)
	return v
}

func callerID(r *http.Request) uuid.UUID {
	v, _ := r.Context().Value("user_id").(uuid.UUID)
	return v
}

func hasRole(rl types.Role, list []types.Role) bool {
	for _, x := range list {
		if x == rl {
			return true
		}
	}
	return false
}

// academicLead roles see and edit every class's results.
func academicLead(rl types.Role) bool {
	return hasRole(rl, types.AdminRoles)
}

// teaches reports whether the caller may enter marks for subject in class:
// academic leads always; teachers when they're the class teacher or are
// assigned the subject in class_subjects.
func (s *ResultsService) teaches(ctx context.Context, r *http.Request, classID, subjectID uuid.UUID) (bool, error) {
	return callerTeaches(ctx, s.pool, r, classID, subjectID)
}

// callerTeaches is the one rule for "may this caller work on subject in
// class" — shared by marks entry, CBC marksheets and lesson plans so they
// can't drift apart.
func callerTeaches(ctx context.Context, pool *pgxpool.Pool, r *http.Request, classID, subjectID uuid.UUID) (bool, error) {
	rl := callerRole(r)
	if academicLead(rl) {
		return true, nil
	}
	if !hasRole(rl, types.TeacherRoles) {
		return false, nil
	}
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM class_groups WHERE id = $1 AND class_teacher_id = $3)
		    OR EXISTS (SELECT 1 FROM class_subjects WHERE class_id = $1 AND subject_id = $2 AND teacher_id = $3)`,
		classID, subjectID, callerID(r)).Scan(&ok)
	return ok, err
}

// ---------------------------------------------------------------------------
// Assessments

type Assessment struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Title          string     `json:"title"` // = Name; older pages read .title
	AssessmentType string     `json:"assessment_type"`
	ClassID        uuid.UUID  `json:"class_id"`
	ClassName      string     `json:"class_name"`
	SubjectID      uuid.UUID  `json:"subject_id"`
	SubjectName    string     `json:"subject_name"`
	TermID         uuid.UUID  `json:"term_id"`
	TermName       string     `json:"term_name"`
	ExamThreadID   *uuid.UUID `json:"exam_thread_id"`
	ExamThreadName string     `json:"exam_thread_name"`
	TotalMarks     float64    `json:"total_marks"`
	Weight         float64    `json:"weight"`
	AssessmentDate *string    `json:"assessment_date"`
	MarksCount     int        `json:"marks_count"`
	Locked         bool       `json:"is_locked"`
}

const assessmentSelect = `
	SELECT a.id, a.name, a.assessment_type, a.class_id, cg.name, a.subject_id, subj.name,
	       a.term_id, t.name, a.exam_thread_id, COALESCE(et.name, ''), a.total_marks, a.weight,
	       a.assessment_date::text,
	       (SELECT count(*) FROM marks m WHERE m.assessment_id = a.id),
	       COALESCE(et.is_locked, false)
	FROM assessments a
	JOIN class_groups cg ON cg.id = a.class_id
	JOIN subjects subj ON subj.id = a.subject_id
	JOIN terms t ON t.id = a.term_id
	LEFT JOIN exam_threads et ON et.id = a.exam_thread_id`

func scanAssessment(row pgx.Row) (Assessment, error) {
	var a Assessment
	err := row.Scan(&a.ID, &a.Name, &a.AssessmentType, &a.ClassID, &a.ClassName, &a.SubjectID, &a.SubjectName,
		&a.TermID, &a.TermName, &a.ExamThreadID, &a.ExamThreadName, &a.TotalMarks, &a.Weight,
		&a.AssessmentDate, &a.MarksCount, &a.Locked)
	a.Title = a.Name
	return a, err
}

func (s *ResultsService) loadAssessment(ctx context.Context, id uuid.UUID) (Assessment, error) {
	return scanAssessment(s.pool.QueryRow(ctx, assessmentSelect+` WHERE a.id = $1`, id))
}

func (s *ResultsService) listAssessments(w http.ResponseWriter, r *http.Request) {
	rl := callerRole(r)
	if !academicLead(rl) && !hasRole(rl, types.TeacherRoles) {
		httpx.ErrForbidden(w, "staff only")
		return
	}
	q := r.URL.Query()
	where := []string{"TRUE"}
	var args []interface{}
	add := func(col, key string) error {
		// class_group is what the older Marks page sends.
		v := q.Get(key)
		if key == "class_id" && v == "" {
			v = q.Get("class_group")
		}
		if v == "" {
			return nil
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return fmt.Errorf("invalid %s", key)
		}
		args = append(args, id)
		where = append(where, fmt.Sprintf("%s = $%d", col, len(args)))
		return nil
	}
	for _, p := range [][2]string{{"a.class_id", "class_id"}, {"a.subject_id", "subject_id"}, {"a.term_id", "term_id"}, {"a.exam_thread_id", "exam_thread_id"}} {
		if err := add(p[0], p[1]); err != nil {
			httpx.ErrBadRequest(w, err.Error())
			return
		}
	}
	if q.Get("term_id") == "" && q.Get("all_terms") == "" {
		where = append(where, "t.is_current = TRUE")
	}
	if !academicLead(rl) {
		args = append(args, callerID(r))
		n := len(args)
		where = append(where, fmt.Sprintf(`(cg.class_teacher_id = $%d OR EXISTS (
			SELECT 1 FROM class_subjects cs WHERE cs.class_id = a.class_id AND cs.subject_id = a.subject_id AND cs.teacher_id = $%d))`, n, n))
	}
	rows, err := s.pool.Query(r.Context(), assessmentSelect+" WHERE "+strings.Join(where, " AND ")+
		" ORDER BY a.assessment_date DESC NULLS LAST, a.created_at DESC", args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []Assessment{}
	for rows.Next() {
		a, err := scanAssessment(rows)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, a)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

var assessmentTypes = []string{"cat1", "cat2", "cat3", "midterm", "endterm", "mock", "assignment", "project", "practical", "homework", "other"}

type assessmentInput struct {
	Name           *string    `json:"name"`
	AssessmentType *string    `json:"assessment_type"`
	ClassID        *uuid.UUID `json:"class_id"`
	SubjectID      *uuid.UUID `json:"subject_id"`
	TermID         *uuid.UUID `json:"term_id"`
	ExamThreadID   *uuid.UUID `json:"exam_thread_id"`
	TotalMarks     *float64   `json:"total_marks"`
	Weight         *float64   `json:"weight"`
	AssessmentDate *string    `json:"assessment_date"`
}

func (in *assessmentInput) validate(creating bool) map[string]string {
	e := map[string]string{}
	if creating {
		if in.Name == nil || strings.TrimSpace(*in.Name) == "" {
			e["name"] = "is required"
		}
		if in.ClassID == nil {
			e["class_id"] = "is required"
		}
		if in.SubjectID == nil {
			e["subject_id"] = "is required"
		}
	}
	if in.AssessmentType != nil && !containsStr(assessmentTypes, *in.AssessmentType) {
		e["assessment_type"] = "must be one of: " + strings.Join(assessmentTypes, ", ")
	}
	if in.TotalMarks != nil && (*in.TotalMarks <= 0 || *in.TotalMarks > 1000) {
		e["total_marks"] = "must be between 1 and 1000"
	}
	if in.Weight != nil && (*in.Weight <= 0 || *in.Weight > 100) {
		e["weight"] = "must be between 1 and 100"
	}
	if in.AssessmentDate != nil && *in.AssessmentDate != "" {
		if _, err := time.Parse("2006-01-02", *in.AssessmentDate); err != nil {
			e["assessment_date"] = "must be a date (YYYY-MM-DD)"
		}
	}
	return e
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func fieldErrors(w http.ResponseWriter, fields map[string]string) {
	httpx.JSON(w, http.StatusBadRequest, map[string]interface{}{
		"error": "validation", "message": "Please correct the highlighted fields.", "fields": fields,
	})
}

// threadLocked reports whether the exam thread is locked; nil = no thread.
func (s *ResultsService) threadLocked(ctx context.Context, id *uuid.UUID) (bool, error) {
	if id == nil {
		return false, nil
	}
	var locked bool
	err := s.pool.QueryRow(ctx, `SELECT is_locked FROM exam_threads WHERE id = $1`, *id).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, errors.New("exam thread not found")
	}
	return locked, err
}

func (s *ResultsService) createAssessment(w http.ResponseWriter, r *http.Request) {
	var in assessmentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.ErrBadRequest(w, "request body must be a JSON object")
		return
	}
	if e := in.validate(true); len(e) > 0 {
		fieldErrors(w, e)
		return
	}
	ok, err := s.teaches(r.Context(), r, *in.ClassID, *in.SubjectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "you don't teach this subject in this class")
		return
	}
	if locked, err := s.threadLocked(r.Context(), in.ExamThreadID); err != nil || locked {
		if err != nil {
			fieldErrors(w, map[string]string{"exam_thread_id": err.Error()})
		} else {
			httpx.ErrConflict(w, "this exam thread is locked")
		}
		return
	}
	// Default to the exam thread's term, else the current term.
	var termID uuid.UUID
	switch {
	case in.TermID != nil:
		termID = *in.TermID
	case in.ExamThreadID != nil:
		err = s.pool.QueryRow(r.Context(), `SELECT term_id FROM exam_threads WHERE id = $1`, *in.ExamThreadID).Scan(&termID)
	default:
		err = s.pool.QueryRow(r.Context(), `SELECT id FROM terms WHERE is_current ORDER BY start_date DESC LIMIT 1`).Scan(&termID)
	}
	if err != nil {
		fieldErrors(w, map[string]string{"term_id": "no current term is set — choose a term"})
		return
	}
	typ, total, weight := "other", 100.0, 100.0
	if in.AssessmentType != nil {
		typ = *in.AssessmentType
	}
	if in.TotalMarks != nil {
		total = *in.TotalMarks
	}
	if in.Weight != nil {
		weight = *in.Weight
	}
	var date *string
	if in.AssessmentDate != nil && *in.AssessmentDate != "" {
		date = in.AssessmentDate
	}
	var id uuid.UUID
	err = s.pool.QueryRow(r.Context(), `
		INSERT INTO assessments (term_id, subject_id, class_id, exam_thread_id, name, assessment_type,
			total_marks, weight, assessment_date, created_by_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::date,$10) RETURNING id`,
		termID, *in.SubjectID, *in.ClassID, in.ExamThreadID, strings.TrimSpace(*in.Name), typ,
		total, weight, date, callerID(r)).Scan(&id)
	if err != nil {
		resultsDBError(w, err)
		return
	}
	a, err := s.loadAssessment(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, a)
}

func (s *ResultsService) updateAssessment(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	cur, err := s.loadAssessment(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "assessment not found")
		return
	}
	if ok, err := s.teaches(r.Context(), r, cur.ClassID, cur.SubjectID); err != nil || !ok {
		httpx.ErrForbidden(w, "you don't teach this subject in this class")
		return
	}
	if cur.Locked {
		httpx.ErrConflict(w, "this exam thread is locked")
		return
	}
	var in assessmentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.ErrBadRequest(w, "request body must be a JSON object")
		return
	}
	if e := in.validate(false); len(e) > 0 {
		fieldErrors(w, e)
		return
	}
	if in.ClassID != nil || in.SubjectID != nil || in.TermID != nil {
		httpx.ErrBadRequest(w, "class, subject and term can't change — create a new assessment instead")
		return
	}
	if in.TotalMarks != nil && *in.TotalMarks != cur.TotalMarks && cur.MarksCount > 0 {
		httpx.ErrConflict(w, "marks are already entered out of the current total; changing it would change every student's percentage")
		return
	}
	if locked, err := s.threadLocked(r.Context(), in.ExamThreadID); err != nil || locked {
		if err != nil {
			fieldErrors(w, map[string]string{"exam_thread_id": err.Error()})
		} else {
			httpx.ErrConflict(w, "that exam thread is locked")
		}
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE assessments SET
			name = COALESCE($2, name),
			assessment_type = COALESCE($3, assessment_type),
			exam_thread_id = COALESCE($4, exam_thread_id),
			total_marks = COALESCE($5, total_marks),
			weight = COALESCE($6, weight),
			assessment_date = COALESCE($7::date, assessment_date),
			updated_at = now()
		WHERE id = $1`, id, in.Name, in.AssessmentType, in.ExamThreadID, in.TotalMarks, in.Weight, in.AssessmentDate)
	if err != nil {
		resultsDBError(w, err)
		return
	}
	a, _ := s.loadAssessment(r.Context(), id)
	httpx.JSON(w, http.StatusOK, a)
}

func (s *ResultsService) deleteAssessment(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	cur, err := s.loadAssessment(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "assessment not found")
		return
	}
	if ok, err := s.teaches(r.Context(), r, cur.ClassID, cur.SubjectID); err != nil || !ok {
		httpx.ErrForbidden(w, "you don't teach this subject in this class")
		return
	}
	if cur.MarksCount > 0 {
		httpx.ErrConflict(w, "marks have been entered for this assessment; clear them first")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM assessments WHERE id = $1`, id); err != nil {
		resultsDBError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func resultsDBError(w http.ResponseWriter, err error) {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23503":
			httpx.ErrBadRequest(w, "a linked record (class, subject, term or exam thread) does not exist")
			return
		case "23514", "22P02", "22007":
			httpx.ErrBadRequest(w, "invalid value: "+pg.Message)
			return
		}
	}
	httpx.Error(w, http.StatusInternalServerError, err)
}

// ---------------------------------------------------------------------------
// Marks entry

type MarkRow struct {
	StudentID       uuid.UUID `json:"student_id"`
	FullName        string    `json:"full_name"`
	AdmissionNo     string    `json:"admission_no"`
	Score           *float64  `json:"score"`
	Percentage      *float64  `json:"percentage"`
	GradeLetter     string    `json:"grade_letter"`
	CompetencyLevel string    `json:"competency_level"`
	Comments        string    `json:"comments"`
}

// marksheet lists the class with any marks already entered. Students who
// have since left the class but hold a mark stay on the sheet, so a
// promotion never hides results that were recorded.
func (s *ResultsService) marksheet(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	a, err := s.loadAssessment(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "assessment not found")
		return
	}
	if ok, err := s.teaches(r.Context(), r, a.ClassID, a.SubjectID); err != nil || !ok {
		httpx.ErrForbidden(w, "you don't teach this subject in this class")
		return
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT st.id, st.full_name, st.admission_no, m.score, COALESCE(m.grade_letter, ''),
		       COALESCE(m.competency_level, ''), COALESCE(m.comments, '')
		FROM students st
		LEFT JOIN marks m ON m.student_id = st.id AND m.assessment_id = $1
		WHERE (st.current_class_id = $2 AND st.is_active) OR m.id IS NOT NULL
		ORDER BY st.full_name`, id, a.ClassID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []MarkRow{}
	for rows.Next() {
		var m MarkRow
		if err := rows.Scan(&m.StudentID, &m.FullName, &m.AdmissionNo, &m.Score, &m.GradeLetter, &m.CompetencyLevel, &m.Comments); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		if m.Score != nil {
			p := round1(*m.Score / a.TotalMarks * 100)
			m.Percentage = &p
		}
		out = append(out, m)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"assessment": a, "results": out, "count": len(out)})
}

type markInput struct {
	StudentID       uuid.UUID `json:"student_id"`
	Score           *float64  `json:"score"` // null clears the mark
	CompetencyLevel string    `json:"competency_level"`
	Comments        string    `json:"comments"`
}

var competencyLevels = []string{"", "EE", "ME", "AE", "BE"}

// saveMarks upserts a whole marksheet in one transaction: either every row
// is saved or none is, and the reply says which rows were wrong.
func (s *ResultsService) saveMarks(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	a, err := s.loadAssessment(r.Context(), id)
	if err != nil {
		httpx.ErrNotFound(w, "assessment not found")
		return
	}
	if ok, err := s.teaches(r.Context(), r, a.ClassID, a.SubjectID); err != nil || !ok {
		httpx.ErrForbidden(w, "you don't teach this subject in this class")
		return
	}
	if a.Locked {
		httpx.ErrConflict(w, "this exam thread is locked — marks can no longer be changed")
		return
	}
	var body struct {
		Marks []markInput `json:"marks"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, `body must be {"marks": [...]}`)
		return
	}

	// Only students on this marksheet may be marked.
	allowed := map[uuid.UUID]bool{}
	rows, err := s.pool.Query(r.Context(), `
		SELECT id FROM students WHERE current_class_id = $2 AND is_active
		UNION SELECT student_id FROM marks WHERE assessment_id = $1`, id, a.ClassID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	for rows.Next() {
		var sid uuid.UUID
		if rows.Scan(&sid) == nil {
			allowed[sid] = true
		}
	}
	rows.Close()

	grades, err := loadGrading(r.Context(), s.pool, a.ClassID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	errs := map[string]string{}
	for _, m := range body.Marks {
		key := m.StudentID.String()
		switch {
		case !allowed[m.StudentID]:
			errs[key] = "student is not in this class"
		case m.Score != nil && (*m.Score < 0 || *m.Score > a.TotalMarks || math.IsNaN(*m.Score)):
			errs[key] = fmt.Sprintf("score must be between 0 and %g", a.TotalMarks)
		case !containsStr(competencyLevels, m.CompetencyLevel):
			errs[key] = "competency level must be EE, ME, AE or BE"
		}
	}
	if len(errs) > 0 {
		fieldErrors(w, errs)
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())
	saved, cleared := 0, 0
	for _, m := range body.Marks {
		if m.Score == nil && m.CompetencyLevel == "" {
			tag, err := tx.Exec(r.Context(), `DELETE FROM marks WHERE assessment_id = $1 AND student_id = $2`, id, m.StudentID)
			if err != nil {
				resultsDBError(w, err)
				return
			}
			cleared += int(tag.RowsAffected())
			continue
		}
		score := 0.0
		var letter string
		var point *float64
		var weighted *float64
		comp := m.CompetencyLevel
		if m.Score != nil {
			score = *m.Score
			pct := score / a.TotalMarks * 100
			g := grades.grade(pct)
			letter, point = g.Letter, &g.Point
			wv := pct * a.Weight / 100
			weighted = &wv
			if comp == "" && containsStr(competencyLevels, g.Letter) {
				comp = g.Letter
			}
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO marks (assessment_id, student_id, score, weighted_score, grade_letter, grade_point,
				competency_level, comments, marked_by_id, marked_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, now())
			ON CONFLICT (assessment_id, student_id) DO UPDATE SET
				score = EXCLUDED.score, weighted_score = EXCLUDED.weighted_score,
				grade_letter = EXCLUDED.grade_letter, grade_point = EXCLUDED.grade_point,
				competency_level = EXCLUDED.competency_level, comments = EXCLUDED.comments,
				marked_by_id = EXCLUDED.marked_by_id, marked_at = now(), updated_at = now()`,
			id, m.StudentID, score, weighted, nullIfEmpty(letter), point, comp, m.Comments, callerID(r))
		if err != nil {
			resultsDBError(w, err)
			return
		}
		saved++
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]int{"saved": saved, "cleared": cleared})
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// listMarks is the mark list: every uploaded result with who entered it and
// when. Staff see their classes, parents and students only their own.
// Filters: student_id, assessment_id, class_id, subject_id, exam_thread_id,
// marked_by (a teacher's user id, or "me"), term_id (default: current term).
func (s *ResultsService) listMarks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var where []string
	var args []interface{}
	filters := [][2]string{
		{"m.student_id", "student_id"}, {"m.assessment_id", "assessment_id"}, {"a.class_id", "class_id"},
		{"a.subject_id", "subject_id"}, {"a.exam_thread_id", "exam_thread_id"}, {"a.term_id", "term_id"},
		{"m.marked_by_id", "marked_by"},
	}
	for _, p := range filters {
		v := q.Get(p[1])
		if v == "" {
			continue
		}
		var id uuid.UUID
		if p[1] == "marked_by" && v == "me" {
			id = callerID(r)
		} else {
			var err error
			if id, err = uuid.Parse(v); err != nil {
				httpx.ErrBadRequest(w, "invalid "+p[1])
				return
			}
		}
		args = append(args, id)
		where = append(where, fmt.Sprintf("%s = $%d", p[0], len(args)))
	}
	if q.Get("term_id") == "" && q.Get("all_terms") == "" {
		where = append(where, "t.is_current")
	}
	rl := callerRole(r)
	switch {
	case academicLead(rl):
	case hasRole(rl, types.TeacherRoles):
		args = append(args, callerID(r))
		n := len(args)
		where = append(where, fmt.Sprintf(`(cg.class_teacher_id = $%d OR EXISTS (
			SELECT 1 FROM class_subjects cs WHERE cs.class_id = a.class_id AND cs.subject_id = a.subject_id AND cs.teacher_id = $%d))`, n, n))
	case rl == types.RoleParent || rl == types.RoleStudent:
		ids, err := resolveStudentScope(r.Context(), s.pool, rl, callerID(r))
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		args = append(args, ids)
		where = append(where, fmt.Sprintf("m.student_id = ANY($%d)", len(args)))
	default:
		httpx.ErrForbidden(w, "not allowed")
		return
	}
	if len(where) == 0 {
		where = []string{"TRUE"}
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT m.id, m.student_id, st.full_name, st.admission_no, a.id, a.name, a.assessment_type,
		       a.class_id, cg.name, subj.id, subj.name, t.id, t.name, m.score, a.total_marks,
		       COALESCE(m.grade_letter,''), COALESCE(m.competency_level,''), COALESCE(m.comments,''),
		       m.marked_by_id, COALESCE(u.first_name || ' ' || u.last_name, ''), m.marked_at
		FROM marks m
		JOIN assessments a ON a.id = m.assessment_id
		JOIN class_groups cg ON cg.id = a.class_id
		JOIN subjects subj ON subj.id = a.subject_id
		JOIN terms t ON t.id = a.term_id
		JOIN students st ON st.id = m.student_id
		LEFT JOIN users u ON u.id = m.marked_by_id
		WHERE `+strings.Join(where, " AND ")+` ORDER BY st.full_name, subj.name, a.name LIMIT 2000`, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type row struct {
		ID              uuid.UUID  `json:"id"`
		StudentID       uuid.UUID  `json:"student_id"`
		StudentName     string     `json:"student_name"`
		AdmissionNo     string     `json:"admission_no"`
		AssessmentID    uuid.UUID  `json:"assessment_id"`
		AssessmentTitle string     `json:"assessment_title"`
		AssessmentType  string     `json:"assessment_type"`
		ClassID         uuid.UUID  `json:"class_id"`
		ClassName       string     `json:"class_name"`
		SubjectID       uuid.UUID  `json:"subject_id"`
		SubjectName     string     `json:"subject_name"`
		TermID          uuid.UUID  `json:"term_id"`
		TermName        string     `json:"term_name"`
		Score           float64    `json:"score"`
		TotalMarks      float64    `json:"total_marks"`
		Percentage      float64    `json:"percentage"`
		GradeLetter     string     `json:"grade_letter"`
		CompetencyLevel string     `json:"competency_level"`
		Comments        string     `json:"comments"`
		MarkedByID      *uuid.UUID `json:"marked_by_id"`
		MarkedByName    string     `json:"marked_by_name"`
		MarkedAt        *time.Time `json:"marked_at"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.StudentID, &x.StudentName, &x.AdmissionNo, &x.AssessmentID, &x.AssessmentTitle,
			&x.AssessmentType, &x.ClassID, &x.ClassName, &x.SubjectID, &x.SubjectName, &x.TermID, &x.TermName,
			&x.Score, &x.TotalMarks, &x.GradeLetter, &x.CompetencyLevel, &x.Comments,
			&x.MarkedByID, &x.MarkedByName, &x.MarkedAt); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		if x.TotalMarks > 0 {
			x.Percentage = round1(x.Score / x.TotalMarks * 100)
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

// ---------------------------------------------------------------------------
// Grading

type gradeBand struct {
	Min, Max float64
	Letter   string
	Point    float64
	Remarks  string
}

type grading []gradeBand

// cbcDefault is used when the school hasn't configured bands for a grade:
// the four CBC performance levels.
var cbcDefault = grading{
	{80, 100, "EE", 4, "Exceeding expectations"},
	{50, 79.999, "ME", 3, "Meeting expectations"},
	{30, 49.999, "AE", 2, "Approaching expectations"},
	{0, 29.999, "BE", 1, "Below expectations"},
}

func (g grading) grade(pct float64) gradeBand {
	pct = math.Max(0, math.Min(100, pct))
	for _, b := range g {
		if pct >= b.Min && pct <= b.Max {
			return b
		}
	}
	// Configured bands with gaps: fall back to the CBC levels.
	for _, b := range cbcDefault {
		if pct >= b.Min && pct <= b.Max {
			return b
		}
	}
	return cbcDefault[len(cbcDefault)-1]
}

// loadGrading returns the school's bands for the class's grade level, or
// the CBC defaults when none are configured.
func loadGrading(ctx context.Context, pool *pgxpool.Pool, classID uuid.UUID) (grading, error) {
	rows, err := pool.Query(ctx, `
		SELECT gr.min_score, gr.max_score, gr.grade_letter, COALESCE(gr.grade_point, 0), COALESCE(gr.remarks, '')
		FROM grade_requirements gr JOIN class_groups cg ON cg.grade = gr.grade
		WHERE cg.id = $1 ORDER BY gr.min_score DESC`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var g grading
	for rows.Next() {
		var b gradeBand
		if err := rows.Scan(&b.Min, &b.Max, &b.Letter, &b.Point, &b.Remarks); err == nil {
			g = append(g, b)
		}
	}
	if len(g) == 0 {
		return cbcDefault, nil
	}
	return g, nil
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// ---------------------------------------------------------------------------
// Results trail

type SubjectResult struct {
	SubjectID   uuid.UUID `json:"subject_id"`
	SubjectName string    `json:"subject_name"`
	Percentage  float64   `json:"percentage"`
	Grade       string    `json:"grade"`
	Remarks     string    `json:"remarks"`
	Assessments int       `json:"assessments"`
}

type TermResult struct {
	TermID       uuid.UUID       `json:"term_id"`
	TermName     string          `json:"term_name"`
	AcademicYear string          `json:"academic_year"`
	StartDate    string          `json:"start_date"`
	ClassID      uuid.UUID       `json:"class_id"`
	ClassName    string          `json:"class_name"`
	Subjects     []SubjectResult `json:"subjects"`
	Mean         float64         `json:"mean"`
	MeanGrade    string          `json:"mean_grade"`
}

type scoredAssessment struct {
	pct, weight          float64
	kind                 string
	hasThread            bool
	catsWeight, mainWght float64
}

// isMainExam: end-of-term papers count toward the thread's main-exam weight;
// everything else is continuous assessment (CATs).
func isMainExam(kind string) bool { return kind == "endterm" || kind == "mock" }

// subjectPercentage combines one subject's assessments for a term. With an
// exam thread, CATs and main exams are averaged separately and blended by the
// thread's weights (a component with no marks yet isn't counted as zero).
// Without one, it's the weighted mean of the assessments.
func subjectPercentage(items []scoredAssessment) float64 {
	threaded := false
	for _, it := range items {
		if it.hasThread {
			threaded = true
			break
		}
	}
	if !threaded {
		var sum, wsum float64
		for _, it := range items {
			sum += it.pct * it.weight
			wsum += it.weight
		}
		if wsum == 0 {
			return 0
		}
		return sum / wsum
	}
	var cats, main []float64
	cw, mw := 0.0, 100.0
	for _, it := range items {
		if it.hasThread {
			cw, mw = it.catsWeight, it.mainWght
		}
		if isMainExam(it.kind) {
			main = append(main, it.pct)
		} else {
			cats = append(cats, it.pct)
		}
	}
	avg := func(v []float64) float64 {
		s := 0.0
		for _, x := range v {
			s += x
		}
		return s / float64(len(v))
	}
	switch {
	case len(cats) > 0 && len(main) > 0 && cw+mw > 0:
		return (avg(cats)*cw + avg(main)*mw) / (cw + mw)
	case len(main) > 0:
		return avg(main)
	default:
		return avg(cats)
	}
}

// StudentHistory builds the results trail: one entry per term the student
// has marks in, oldest first, each labelled with the class they sat it in.
func (s *ResultsService) StudentHistory(ctx context.Context, studentID uuid.UUID) ([]TermResult, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.name, COALESCE(ay.name, ''), t.start_date::text, a.class_id, cg.name,
		       subj.id, subj.name, m.score / NULLIF(a.total_marks, 0) * 100, a.weight, a.assessment_type,
		       a.exam_thread_id IS NOT NULL, COALESCE(et.cats_weight, 0)::float8, COALESCE(et.main_exam_weight, 100)::float8
		FROM marks m
		JOIN assessments a ON a.id = m.assessment_id
		JOIN terms t ON t.id = a.term_id
		LEFT JOIN academic_years ay ON ay.id = t.academic_year_id
		JOIN class_groups cg ON cg.id = a.class_id
		JOIN subjects subj ON subj.id = a.subject_id
		LEFT JOIN exam_threads et ON et.id = a.exam_thread_id
		WHERE m.student_id = $1 AND m.score IS NOT NULL
		ORDER BY t.start_date, subj.name`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type termKey struct{ term, class uuid.UUID }
	terms := map[termKey]*TermResult{}
	var order []termKey
	perSubject := map[termKey]map[uuid.UUID][]scoredAssessment{}
	subjectNames := map[uuid.UUID]string{}
	for rows.Next() {
		var tr TermResult
		var subjID uuid.UUID
		var subjName string
		var pct *float64
		var it scoredAssessment
		if err := rows.Scan(&tr.TermID, &tr.TermName, &tr.AcademicYear, &tr.StartDate, &tr.ClassID, &tr.ClassName,
			&subjID, &subjName, &pct, &it.weight, &it.kind, &it.hasThread, &it.catsWeight, &it.mainWght); err != nil {
			return nil, err
		}
		if pct == nil {
			continue
		}
		it.pct = *pct
		k := termKey{tr.TermID, tr.ClassID}
		if _, ok := terms[k]; !ok {
			t := tr
			terms[k] = &t
			order = append(order, k)
			perSubject[k] = map[uuid.UUID][]scoredAssessment{}
		}
		perSubject[k][subjID] = append(perSubject[k][subjID], it)
		subjectNames[subjID] = subjName
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]TermResult, 0, len(order))
	for _, k := range order {
		t := terms[k]
		g, err := loadGrading(ctx, s.pool, t.ClassID)
		if err != nil {
			return nil, err
		}
		var sum float64
		for subjID, items := range perSubject[k] {
			pct := round1(subjectPercentage(items))
			b := g.grade(pct)
			t.Subjects = append(t.Subjects, SubjectResult{
				SubjectID: subjID, SubjectName: subjectNames[subjID], Percentage: pct,
				Grade: b.Letter, Remarks: b.Remarks, Assessments: len(items),
			})
			sum += pct
		}
		sort.Slice(t.Subjects, func(i, j int) bool { return t.Subjects[i].SubjectName < t.Subjects[j].SubjectName })
		if len(t.Subjects) > 0 {
			t.Mean = round1(sum / float64(len(t.Subjects)))
			t.MeanGrade = g.grade(t.Mean).Letter
		}
		out = append(out, *t)
	}
	return out, nil
}

func (s *ResultsService) history(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if !s.canSeeResults(r, id) {
		httpx.ErrForbidden(w, "cannot view this student's results")
		return
	}
	var name, adm string
	if err := s.pool.QueryRow(r.Context(), `SELECT full_name, admission_no FROM students WHERE id = $1`, id).Scan(&name, &adm); err != nil {
		httpx.ErrNotFound(w, "student not found")
		return
	}
	hist, err := s.StudentHistory(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	// ?term_id= narrows the trail to one term — what a parent opens when
	// results for that term come out.
	if v := r.URL.Query().Get("term_id"); v != "" {
		termID, err := uuid.Parse(v)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid term_id")
			return
		}
		filtered := []TermResult{}
		for _, t := range hist {
			if t.TermID == termID {
				filtered = append(filtered, t)
			}
		}
		hist = filtered
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"student_id": id, "full_name": name, "admission_no": adm, "terms": hist,
	})
}

// classPerformance is the overall performance of a class for a term: each
// student's subject scores, mean and position, plus each subject's class
// mean. Built from StudentHistory so it always agrees with the report card
// and transcript. Staff only — it names every learner in the class.
func (s *ResultsService) classPerformance(w http.ResponseWriter, r *http.Request) {
	rl := callerRole(r)
	if !academicLead(rl) && !hasRole(rl, types.TeacherRoles) {
		httpx.ErrForbidden(w, "staff only")
		return
	}
	classID, err1 := uuid.Parse(r.URL.Query().Get("class_id"))
	termID, err2 := uuid.Parse(r.URL.Query().Get("term_id"))
	if err1 != nil || err2 != nil {
		httpx.ErrBadRequest(w, "class_id and term_id are required")
		return
	}
	var className, termName string
	if err := s.pool.QueryRow(r.Context(), `SELECT name FROM class_groups WHERE id = $1`, classID).Scan(&className); err != nil {
		httpx.ErrNotFound(w, "class not found")
		return
	}
	if err := s.pool.QueryRow(r.Context(), `SELECT name FROM terms WHERE id = $1`, termID).Scan(&termName); err != nil {
		httpx.ErrNotFound(w, "term not found")
		return
	}
	// Everyone who sat an assessment set for this class in this term — not
	// the class's current roll, so promotions don't change past rankings.
	rows, err := s.pool.Query(r.Context(), `
		SELECT DISTINCT st.id, st.full_name, st.admission_no
		FROM marks m
		JOIN assessments a ON a.id = m.assessment_id
		JOIN students st ON st.id = m.student_id
		WHERE a.class_id = $1 AND a.term_id = $2 AND m.score IS NOT NULL`, classID, termID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	type student struct {
		StudentID   uuid.UUID       `json:"student_id"`
		FullName    string          `json:"full_name"`
		AdmissionNo string          `json:"admission_no"`
		Subjects    []SubjectResult `json:"subjects"`
		Mean        float64         `json:"mean"`
		MeanGrade   string          `json:"mean_grade"`
		Position    int             `json:"position"`
	}
	var students []student
	for rows.Next() {
		var st student
		if err := rows.Scan(&st.StudentID, &st.FullName, &st.AdmissionNo); err != nil {
			rows.Close()
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		students = append(students, st)
	}
	rows.Close()

	type subjAgg struct {
		name  string
		sum   float64
		count int
	}
	subjects := map[uuid.UUID]*subjAgg{}
	out := make([]student, 0, len(students))
	for _, st := range students {
		hist, err := s.StudentHistory(r.Context(), st.StudentID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		for _, t := range hist {
			if t.TermID != termID || t.ClassID != classID {
				continue
			}
			st.Subjects, st.Mean, st.MeanGrade = t.Subjects, t.Mean, t.MeanGrade
			for _, sub := range t.Subjects {
				a := subjects[sub.SubjectID]
				if a == nil {
					a = &subjAgg{name: sub.SubjectName}
					subjects[sub.SubjectID] = a
				}
				a.sum += sub.Percentage
				a.count++
			}
		}
		out = append(out, st)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Mean != out[j].Mean {
			return out[i].Mean > out[j].Mean
		}
		return out[i].FullName < out[j].FullName
	})
	// Standard competition ranking: equal means share a position.
	var classSum float64
	for i := range out {
		if i > 0 && out[i].Mean == out[i-1].Mean {
			out[i].Position = out[i-1].Position
		} else {
			out[i].Position = i + 1
		}
		classSum += out[i].Mean
	}
	g, err := loadGrading(r.Context(), s.pool, classID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	type subjectMean struct {
		SubjectID   uuid.UUID `json:"subject_id"`
		SubjectName string    `json:"subject_name"`
		Mean        float64   `json:"mean"`
		Grade       string    `json:"grade"`
		Students    int       `json:"students"`
	}
	subjOut := []subjectMean{}
	for id, a := range subjects {
		m := round1(a.sum / float64(a.count))
		subjOut = append(subjOut, subjectMean{id, a.name, m, g.grade(m).Letter, a.count})
	}
	sort.Slice(subjOut, func(i, j int) bool { return subjOut[i].Mean > subjOut[j].Mean })
	classMean := 0.0
	if len(out) > 0 {
		classMean = round1(classSum / float64(len(out)))
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"class_id": classID, "class_name": className, "term_id": termID, "term_name": termName,
		"class_mean": classMean, "class_mean_grade": g.grade(classMean).Letter,
		"student_count": len(out), "students": out, "subjects": subjOut,
	})
}

// canSeeResults: parents/students their own; academic leads and teachers
// everyone (teachers routinely look at a learner's past performance);
// finance staff via resolveStudentScope's "all".
func (s *ResultsService) canSeeResults(r *http.Request, studentID uuid.UUID) bool {
	rl := callerRole(r)
	if academicLead(rl) || hasRole(rl, types.TeacherRoles) {
		return true
	}
	return canViewStudent(r, s.pool, studentID)
}

// transcript renders the whole results trail as a PDF with a signed QR code,
// so another school can confirm it wasn't altered.
func (s *ResultsService) transcript(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if !s.canSeeResults(r, id) {
		httpx.ErrForbidden(w, "cannot view this student's results")
		return
	}
	var name, adm string
	if err := s.pool.QueryRow(r.Context(), `SELECT full_name, admission_no FROM students WHERE id = $1`, id).Scan(&name, &adm); err != nil {
		httpx.ErrNotFound(w, "student not found")
		return
	}
	hist, err := s.StudentHistory(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}

	pdf := newDocPDF("Academic Record")
	pdfKeyValue(pdf, "Student:", name)
	pdfKeyValue(pdf, "Admission No:", adm)
	pdfKeyValue(pdf, "Terms on record:", fmt.Sprintf("%d", len(hist)))
	pdf.Ln(4)
	if len(hist) == 0 {
		pdf.SetFont("Helvetica", "I", 10)
		pdf.CellFormat(0, 8, "No results have been recorded for this student yet.", "", 1, "L", false, 0, "")
	}
	for _, t := range hist {
		if pdf.GetY() > 240 {
			pdf.AddPage()
		}
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetFillColor(27, 54, 93)
		pdf.SetTextColor(255, 255, 255)
		pdf.CellFormat(0, 8, fmt.Sprintf("%s %s  -  %s", t.TermName, t.AcademicYear, t.ClassName), "1", 1, "L", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(80, 7, "Subject", "1", 0, "L", true, 0, "")
		pdf.CellFormat(25, 7, "Score %", "1", 0, "R", true, 0, "")
		pdf.CellFormat(20, 7, "Grade", "1", 0, "C", true, 0, "")
		pdf.CellFormat(0, 7, "Remarks", "1", 1, "L", true, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		for _, sub := range t.Subjects {
			pdf.CellFormat(80, 6, sub.SubjectName, "1", 0, "L", false, 0, "")
			pdf.CellFormat(25, 6, fmt.Sprintf("%.1f", sub.Percentage), "1", 0, "R", false, 0, "")
			pdf.CellFormat(20, 6, sub.Grade, "1", 0, "C", false, 0, "")
			pdf.CellFormat(0, 6, sub.Remarks, "1", 1, "L", false, 0, "")
		}
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(80, 7, "Mean", "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 7, fmt.Sprintf("%.1f", t.Mean), "1", 0, "R", false, 0, "")
		pdf.CellFormat(20, 7, t.MeanGrade, "1", 0, "C", false, 0, "")
		pdf.CellFormat(0, 7, "", "1", 1, "L", false, 0, "")
		pdf.Ln(4)
	}
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(0, 5, "Issued on "+time.Now().Format("02 Jan 2006")+". Results are shown against the class the student was in that term.", "", 1, "L", false, 0, "")

	latest := 0.0
	if len(hist) > 0 {
		latest = hist[len(hist)-1].Mean
	}
	if url, err := s.links.ResultsVerifyURL(id, len(hist), latest); err == nil {
		if pdf.GetY() > 250 {
			pdf.AddPage()
		}
		addVerifyQR(pdf, url, "Scan to verify this record",
			"The code is digitally signed by the school. Scanning it confirms who this record belongs to, when it was issued and how many terms it covers.")
	}
	streamPDF(w, pdf, "academic-record-"+adm+".pdf")
}
