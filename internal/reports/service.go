package reports

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

type Service struct {
	pool       *pgxpool.Pool
	nairobiLoc *time.Location
}

func NewService(pool *pgxpool.Pool, loc *time.Location) *Service {
	return &Service{pool: pool, nairobiLoc: loc}
}

func (s *Service) RegisterInternalRoutes(r chi.Router) {
	r.Post("/reports/attendance", s.handleAttendanceReport)
	r.Post("/reports/grades", s.handleGradesReport)
	r.Post("/reports/fee-collection", s.handleFeeCollectionReport)
	r.Post("/reports/class-performance", s.handleClassPerformance)
}

func (s *Service) handleAttendanceReport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StartDate string     `json:"start_date" validate:"required"`
		EndDate   string     `json:"end_date" validate:"required"`
		ClassID   *uuid.UUID `json:"class_id,omitempty"`
		StreamID  *uuid.UUID `json:"stream_id,omitempty"`
		StudentID *uuid.UUID `json:"student_id,omitempty"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	sd, err := time.ParseInLocation("2006-01-02", body.StartDate, s.nairobiLoc)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid start_date (YYYY-MM-DD)")
		return
	}
	ed, err := time.ParseInLocation("2006-01-02", body.EndDate, s.nairobiLoc)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid end_date (YYYY-MM-DD)")
		return
	}
	_ = sd
	_ = ed
	q := `
		SELECT s.id,
		       s.first_name || ' ' || s.last_name AS student_name,
		       c.id AS class_id,
		       COALESCE(c.name, '') AS class_name,
		       COUNT(DISTINCT a.date) FILTER (WHERE a.status IN ('present','late')) AS present_days,
		       COUNT(DISTINCT a.date) FILTER (WHERE a.status = 'absent') AS absent_days,
		       COUNT(DISTINCT dates.d) AS expected_days
		FROM generate_series($1::date, $2::date, interval '1 day') AS dates(d)
		CROSS JOIN students s
		LEFT JOIN classes c ON c.id = s.class_id
		LEFT JOIN attendance a ON a.student_id = s.id AND a.date = dates.d
		WHERE EXTRACT(ISODOW FROM dates.d) BETWEEN 1 AND 5
		  AND ($3::uuid IS NULL OR s.class_id = $3)
		  AND ($4::uuid IS NULL OR s.stream_id = $4)
		  AND ($5::uuid IS NULL OR s.id = $5)
		  AND s.is_active = TRUE
		GROUP BY s.id, s.first_name, s.last_name, c.id, c.name`
	rows, err := s.pool.Query(r.Context(), q, body.StartDate, body.EndDate, body.ClassID, body.StreamID, body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type Row struct {
		StudentID    uuid.UUID `json:"student_id"`
		StudentName  string    `json:"student_name"`
		ClassID      uuid.UUID `json:"class_id"`
		ClassName    string    `json:"class_name"`
		ExpectedDays int       `json:"expected_days"`
		PresentDays  int       `json:"present_days"`
		AbsentDays   int       `json:"absent_days"`
		AttendancePct float64  `json:"attendance_pct"`
	}
	out := []Row{}
	for rows.Next() {
		var r Row
		var cid *uuid.UUID
		var cname *string
		if err := rows.Scan(&r.StudentID, &r.StudentName, &cid, &cname, &r.PresentDays, &r.AbsentDays, &r.ExpectedDays); err != nil {
			continue
		}
		if cid != nil {
			r.ClassID = *cid
		}
		if cname != nil {
			r.ClassName = *cname
		}
		if r.ExpectedDays > 0 {
			r.AttendancePct = float64(r.PresentDays) / float64(r.ExpectedDays) * 100
		}
		out = append(out, r)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *Service) handleGradesReport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TermID    uuid.UUID  `json:"term_id" validate:"required,uuid"`
		ClassID   *uuid.UUID `json:"class_id,omitempty"`
		SubjectID *uuid.UUID `json:"subject_id,omitempty"`
		StudentID *uuid.UUID `json:"student_id,omitempty"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	q := `
		SELECT s.id AS student_id,
		       s.first_name || ' ' || s.last_name AS student_name,
		       c.id AS class_id, COALESCE(c.name,'') AS class_name,
		       subj.id AS subject_id, COALESCE(subj.name,'') AS subject_name,
		       COALESCE(AVG(m.score), 0) AS mean_score,
		       COUNT(DISTINCT s.id) OVER (PARTITION BY subj.id) AS total_students
		FROM marks m
		JOIN assessments a ON a.id = m.assessment_id
		JOIN students s ON s.id = m.student_id
		LEFT JOIN classes c ON c.id = s.class_id
		LEFT JOIN subjects subj ON subj.id = a.subject_id
		WHERE a.term_id = $1
		  AND ($2::uuid IS NULL OR s.class_id = $2)
		  AND ($3::uuid IS NULL OR a.subject_id = $3)
		  AND ($4::uuid IS NULL OR s.id = $4)
		GROUP BY s.id, s.first_name, s.last_name, c.id, c.name, subj.id, subj.name`
	rows, err := s.pool.Query(r.Context(), q, body.TermID, body.ClassID, body.SubjectID, body.StudentID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type Row struct {
		StudentID     uuid.UUID `json:"student_id"`
		StudentName   string    `json:"student_name"`
		ClassID       uuid.UUID `json:"class_id,omitempty"`
		ClassName     string    `json:"class_name,omitempty"`
		SubjectID     uuid.UUID `json:"subject_id,omitempty"`
		SubjectName   string    `json:"subject_name,omitempty"`
		MeanScore     float64   `json:"mean_score"`
		MeanGrade     string    `json:"mean_grade,omitempty"`
		TotalStudents int       `json:"total_students,omitempty"`
		Rank          int       `json:"rank,omitempty"`
	}
	out := []Row{}
	for rows.Next() {
		var r Row
		var cid *uuid.UUID
		var cname *string
		var sid *uuid.UUID
		var sname *string
		if err := rows.Scan(&r.StudentID, &r.StudentName, &cid, &cname, &sid, &sname, &r.MeanScore, &r.TotalStudents); err != nil {
			continue
		}
		if cid != nil {
			r.ClassID = *cid
		}
		if cname != nil {
			r.ClassName = *cname
		}
		if sid != nil {
			r.SubjectID = *sid
		}
		if sname != nil {
			r.SubjectName = *sname
		}
		r.MeanGrade = gradeFromScore(r.MeanScore)
		out = append(out, r)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func gradeFromScore(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "A-"
	case score >= 75:
		return "B+"
	case score >= 70:
		return "B"
	case score >= 65:
		return "B-"
	case score >= 60:
		return "C+"
	case score >= 55:
		return "C"
	case score >= 50:
		return "C-"
	case score >= 45:
		return "D+"
	case score >= 40:
		return "D"
	}
	return "E"
}

func (s *Service) handleFeeCollectionReport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TermID    *uuid.UUID `json:"term_id,omitempty"`
		StartDate *time.Time `json:"start_date,omitempty"`
		EndDate   *time.Time `json:"end_date,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	q := `
		SELECT c.id AS class_id, COALESCE(c.name,'') AS class_name,
		       COUNT(DISTINCT s.id) AS total_students,
		       COALESCE(SUM(i.total_due),0) AS expected_amount,
		       COALESCE(SUM(i.total_paid),0) AS collected_amount,
		       COALESCE(SUM(CASE WHEN i.balance > 0 THEN i.balance ELSE 0 END),0) AS outstanding
		FROM students s
		LEFT JOIN classes c ON c.id = s.class_id
		LEFT JOIN invoices i ON i.student_id = s.id
		  AND ($1::uuid IS NULL OR i.term_id = $1)
		WHERE s.is_active = TRUE
		GROUP BY c.id, c.name`
	rows, err := s.pool.Query(r.Context(), q, body.TermID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type Row struct {
		ClassID         uuid.UUID `json:"class_id,omitempty"`
		ClassName       string    `json:"class_name,omitempty"`
		TotalStudents   int       `json:"total_students"`
		ExpectedAmount  float64   `json:"expected_amount"`
		CollectedAmount float64   `json:"collected_amount"`
		Outstanding     float64   `json:"outstanding"`
		CollectionPct   float64   `json:"collection_pct"`
	}
	out := []Row{}
	for rows.Next() {
		var r Row
		var cid *uuid.UUID
		var cname *string
		if err := rows.Scan(&cid, &cname, &r.TotalStudents, &r.ExpectedAmount, &r.CollectedAmount, &r.Outstanding); err != nil {
			continue
		}
		if cid != nil {
			r.ClassID = *cid
		}
		if cname != nil {
			r.ClassName = *cname
		}
		if r.ExpectedAmount > 0 {
			r.CollectionPct = r.CollectedAmount / r.ExpectedAmount * 100
		}
		out = append(out, r)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *Service) handleClassPerformance(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TermID  uuid.UUID `json:"term_id" validate:"required,uuid"`
		ClassID uuid.UUID `json:"class_id" validate:"required,uuid"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	q := `
		SELECT subj.id AS subject_id, COALESCE(subj.name,'') AS subject_name,
		       COALESCE(AVG(m.score), 0) AS mean_score,
		       COUNT(DISTINCT CASE WHEN m.score >= 50 THEN s.id END) AS pass_count,
		       COUNT(DISTINCT s.id) AS total_students
		FROM marks m
		JOIN assessments a ON a.id = m.assessment_id
		JOIN students s ON s.id = m.student_id
		JOIN subjects subj ON subj.id = a.subject_id
		WHERE a.term_id = $1 AND s.class_id = $2
		GROUP BY subj.id, subj.name`
	rows, err := s.pool.Query(r.Context(), q, body.TermID, body.ClassID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type Row struct {
		SubjectID     uuid.UUID `json:"subject_id"`
		SubjectName   string    `json:"subject_name"`
		MeanScore     float64   `json:"mean_score"`
		PassCount     int       `json:"pass_count"`
		TotalStudents int       `json:"total_students"`
		PassPct       float64   `json:"pass_pct"`
	}
	out := []Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.SubjectID, &r.SubjectName, &r.MeanScore, &r.PassCount, &r.TotalStudents); err != nil {
			continue
		}
		if r.TotalStudents > 0 {
			r.PassPct = float64(r.PassCount) / float64(r.TotalStudents) * 100
		}
		out = append(out, r)
	}
	httpx.JSON(w, http.StatusOK, out)
}
