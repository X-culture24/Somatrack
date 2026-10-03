package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// CBCService handles CBC (Competency-Based Curriculum) assessment data:
//   - Teacher marks entry per assessment (cat1/cat2/cat3) per subject/class/term
//   - Headteacher sign-off (teacher comments + HT signature)
//   - Class-level summary for the headteacher dashboard
type CBCService struct {
	pool *pgxpool.Pool
}

func NewCBCService(pool *pgxpool.Pool) *CBCService {
	return &CBCService{pool: pool}
}

func (s *CBCService) RegisterRoutes(r chi.Router) {
	r.Route("/cbc/", func(r chi.Router) {
		// Every CBC view names learners and their levels — staff only.
		r.Use(academicStaffOnly())
		r.Get("/marksheet/", s.marksheet)
		r.Put("/marksheet/", s.saveMarks)
		r.Get("/class-summary/", s.classSummary)
		r.Get("/signoffs/", s.listSignoffs)
		r.Post("/signoffs/", s.upsertSignoff)
		r.Get("/analytics/distribution/", s.cbcDistribution)
	})
}

// academicStaffOnly lets through academic leads and teachers.
func academicStaffOnly() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl := callerRole(r); !academicLead(rl) && !hasRole(rl, types.TeacherRoles) {
				httpx.ErrForbidden(w, "staff only")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// marksheet — GET /api/cbc/marksheet/?class_id=X&term_id=Y
//
// Returns every subject assigned to the class, the 3 assessments (cat1/2/3)
// that exist for each subject/term, and the competency marks already saved
// so the teacher can see what they've entered.
func (s *CBCService) marksheet(w http.ResponseWriter, r *http.Request) {
	classID, err := uuid.Parse(r.URL.Query().Get("class_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "class_id required")
		return
	}
	termID, err := uuid.Parse(r.URL.Query().Get("term_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "term_id required")
		return
	}

	// Students in this class
	srows, err := s.pool.Query(r.Context(), `
		SELECT id, full_name, admission_no FROM students
		WHERE current_class_id = $1 AND is_active = TRUE ORDER BY last_name, first_name`, classID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer srows.Close()
	type student struct {
		ID          uuid.UUID `json:"id"`
		FullName    string    `json:"full_name"`
		AdmissionNo string    `json:"admission_no"`
	}
	var students []student
	var studentIDs []uuid.UUID
	for srows.Next() {
		var st student
		if srows.Scan(&st.ID, &st.FullName, &st.AdmissionNo) == nil {
			students = append(students, st)
			studentIDs = append(studentIDs, st.ID)
		}
	}

	// Subjects assigned to this class
	subrows, err := s.pool.Query(r.Context(), `
		SELECT s.id, s.name, s.code, s.learning_area
		FROM class_subjects cs JOIN subjects s ON s.id = cs.subject_id
		WHERE cs.class_id = $1 AND s.is_active = TRUE ORDER BY s.name`, classID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer subrows.Close()
	type subject struct {
		ID           uuid.UUID `json:"id"`
		Name         string    `json:"name"`
		Code         string    `json:"code"`
		LearningArea string    `json:"learning_area"`
	}
	var subjects []subject
	var subjectIDs []uuid.UUID
	for subrows.Next() {
		var sub subject
		if subrows.Scan(&sub.ID, &sub.Name, &sub.Code, &sub.LearningArea) == nil {
			subjects = append(subjects, sub)
			subjectIDs = append(subjectIDs, sub.ID)
		}
	}

	// Assessments: cat1/cat2/cat3 for this class/term (auto-create if missing)
	// Map: subject_id -> {cat1_id, cat2_id, cat3_id}
	type assessRow struct {
		AssessmentID uuid.UUID
		SubjectID    uuid.UUID
		AssessType   string
	}
	arows, err := s.pool.Query(r.Context(), `
		SELECT id, subject_id, assessment_type FROM assessments
		WHERE class_id = $1 AND term_id = $2
		  AND assessment_type IN ('cat1','cat2','cat3')
		ORDER BY subject_id, assessment_type`, classID, termID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer arows.Close()
	assessMap := map[uuid.UUID]map[string]uuid.UUID{} // subjectID -> {cat1: id, cat2: id, cat3: id}
	for arows.Next() {
		var ar assessRow
		if arows.Scan(&ar.AssessmentID, &ar.SubjectID, &ar.AssessType) == nil {
			if assessMap[ar.SubjectID] == nil {
				assessMap[ar.SubjectID] = map[string]uuid.UUID{}
			}
			assessMap[ar.SubjectID][ar.AssessType] = ar.AssessmentID
		}
	}

	// Auto-create any missing cat1/cat2/cat3 assessments
	createdBy := callerID(r)
	for _, sub := range subjects {
		for i, cat := range []string{"cat1", "cat2", "cat3"} {
			if _, ok := assessMap[sub.ID][cat]; !ok {
				var newID uuid.UUID
				err := s.pool.QueryRow(r.Context(), `
					INSERT INTO assessments (name, assessment_type, class_id, subject_id, term_id,
					    total_marks, weight, created_by_id)
					VALUES ($1,$2,$3,$4,$5,100,100,$6)
					ON CONFLICT (class_id, subject_id, term_id, assessment_type)
					    WHERE assessment_type IN ('cat1','cat2','cat3')
					DO UPDATE SET name = EXCLUDED.name
					RETURNING id`,
					sub.Name+" Assessment "+[]string{"1", "2", "3"}[i], cat,
					classID, sub.ID, termID, createdBy,
				).Scan(&newID)
				if err == nil {
					if assessMap[sub.ID] == nil {
						assessMap[sub.ID] = map[string]uuid.UUID{}
					}
					assessMap[sub.ID][cat] = newID
				}
			}
		}
	}

	// All assessment IDs for marks lookup
	var allAssessIDs []uuid.UUID
	for _, cats := range assessMap {
		for _, aid := range cats {
			allAssessIDs = append(allAssessIDs, aid)
		}
	}

	// Existing marks: map assessment_id -> student_id -> {level, reflection}
	type markVal struct {
		Level      string
		Reflection string
	}
	markMap := map[uuid.UUID]map[uuid.UUID]markVal{} // assessID -> studentID -> val
	if len(allAssessIDs) > 0 && len(studentIDs) > 0 {
		mrows, err := s.pool.Query(r.Context(), `
			SELECT assessment_id, student_id, competency_level, reflection
			FROM marks
			WHERE assessment_id = ANY($1) AND student_id = ANY($2)`, allAssessIDs, studentIDs)
		if err == nil {
			defer mrows.Close()
			for mrows.Next() {
				var aid, sid uuid.UUID
				var level, refl string
				if mrows.Scan(&aid, &sid, &level, &refl) == nil {
					if markMap[aid] == nil {
						markMap[aid] = map[uuid.UUID]markVal{}
					}
					markMap[aid][sid] = markVal{level, refl}
				}
			}
		}
	}

	// Signoffs for these students/term
	signoffMap := map[uuid.UUID]map[string]string{} // studentID -> field -> value
	if len(studentIDs) > 0 {
		sgrows, err := s.pool.Query(r.Context(), `
			SELECT student_id, teacher_comment, general_performance_level,
			       headteacher_signed::text, closing_date, opening_date
			FROM cbc_signoffs WHERE student_id = ANY($1) AND term_id = $2`, studentIDs, termID)
		if err == nil {
			defer sgrows.Close()
			for sgrows.Next() {
				var sid uuid.UUID
				var tc, gpl, signed, cd, od string
				if sgrows.Scan(&sid, &tc, &gpl, &signed, &cd, &od) == nil {
					signoffMap[sid] = map[string]string{
						"teacher_comment": tc, "general_performance_level": gpl,
						"headteacher_signed": signed, "closing_date": cd, "opening_date": od,
					}
				}
			}
		}
	}

	// Build response: for each student, for each subject, assessments 1/2/3 with levels
	type markEntry struct {
		AssessmentID string `json:"assessment_id"`
		Level        string `json:"level"`
		Reflection   string `json:"reflection"`
	}
	type subjectEntry struct {
		SubjectID    string     `json:"subject_id"`
		SubjectName  string     `json:"subject_name"`
		LearningArea string     `json:"learning_area"`
		Cat1         markEntry  `json:"cat1"`
		Cat2         markEntry  `json:"cat2"`
		Cat3         markEntry  `json:"cat3"`
	}
	type studentRow struct {
		StudentID   string         `json:"student_id"`
		FullName    string         `json:"full_name"`
		AdmissionNo string         `json:"admission_no"`
		Subjects    []subjectEntry `json:"subjects"`
		Signoff     map[string]string `json:"signoff"`
	}

	var rows []studentRow
	for _, st := range students {
		var subjEntries []subjectEntry
		for _, sub := range subjects {
			cats := assessMap[sub.ID]
			getEntry := func(cat string) markEntry {
				aid, ok := cats[cat]
				if !ok {
					return markEntry{}
				}
				mv := markMap[aid][st.ID]
				return markEntry{AssessmentID: aid.String(), Level: mv.Level, Reflection: mv.Reflection}
			}
			subjEntries = append(subjEntries, subjectEntry{
				SubjectID:    sub.ID.String(),
				SubjectName:  sub.Name,
				LearningArea: sub.LearningArea,
				Cat1:         getEntry("cat1"),
				Cat2:         getEntry("cat2"),
				Cat3:         getEntry("cat3"),
			})
		}
		rows = append(rows, studentRow{
			StudentID:   st.ID.String(),
			FullName:    st.FullName,
			AdmissionNo: st.AdmissionNo,
			Subjects:    subjEntries,
			Signoff:     signoffMap[st.ID],
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"class_id": classID, "term_id": termID,
		"students": rows, "subjects": subjects,
	})
}

// ---------------------------------------------------------------------------
// saveMarks — PUT /api/cbc/marksheet/
//
// Body: { entries: [{assessment_id, student_id, level, reflection}] }
func (s *CBCService) saveMarks(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Entries []struct {
			AssessmentID string `json:"assessment_id"`
			StudentID    string `json:"student_id"`
			Level        string `json:"level"`       // EE / ME / AE / BE or ""
			Reflection   string `json:"reflection"`
		} `json:"entries"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	markedBy := callerID(r)
	// Check every assessment up front: a teacher may only enter levels for
	// subjects they're assigned in that class, and never on a locked exam.
	checked := map[uuid.UUID]bool{}
	for _, e := range body.Entries {
		aid, err := uuid.Parse(e.AssessmentID)
		if err != nil || checked[aid] {
			continue
		}
		var classID, subjectID uuid.UUID
		var locked bool
		err = s.pool.QueryRow(r.Context(), `
			SELECT a.class_id, a.subject_id, COALESCE(et.is_locked, false)
			FROM assessments a LEFT JOIN exam_threads et ON et.id = a.exam_thread_id
			WHERE a.id = $1`, aid).Scan(&classID, &subjectID, &locked)
		if err != nil {
			httpx.ErrBadRequest(w, "assessment "+aid.String()+" not found")
			return
		}
		ok, err := callerTeaches(r.Context(), s.pool, r, classID, subjectID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			httpx.ErrForbidden(w, "you don't teach this subject in this class")
			return
		}
		if locked {
			httpx.ErrConflict(w, "this exam thread is locked — marks can no longer be changed")
			return
		}
		checked[aid] = true
	}
	saved := 0
	for _, e := range body.Entries {
		aid, err1 := uuid.Parse(e.AssessmentID)
		sid, err2 := uuid.Parse(e.StudentID)
		if err1 != nil || err2 != nil {
			continue
		}
		level := e.Level
		if level != "" && level != "EE" && level != "ME" && level != "AE" && level != "BE" {
			continue
		}
		_, err := s.pool.Exec(r.Context(), `
			INSERT INTO marks (assessment_id, student_id, score, competency_level, reflection, marked_by_id, marked_at)
			VALUES ($1, $2, 0, $3, $4, $5, NOW())
			ON CONFLICT (assessment_id, student_id) DO UPDATE
			  SET competency_level = EXCLUDED.competency_level,
			      reflection       = EXCLUDED.reflection,
			      marked_by_id     = EXCLUDED.marked_by_id,
			      marked_at        = NOW(),
			      updated_at       = NOW()`,
			aid, sid, level, e.Reflection, markedBy)
		if err == nil {
			saved++
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]int{"saved": saved})
}

// ---------------------------------------------------------------------------
// classSummary — GET /api/cbc/class-summary/?class_id=X&term_id=Y
//
// Headteacher view: all students with their competency level per subject/assessment.
func (s *CBCService) classSummary(w http.ResponseWriter, r *http.Request) {
	classID, err := uuid.Parse(r.URL.Query().Get("class_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "class_id required")
		return
	}
	termID, err := uuid.Parse(r.URL.Query().Get("term_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "term_id required")
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		SELECT s.id, s.full_name, s.admission_no,
		       subj.name AS subject_name,
		       a.assessment_type,
		       m.competency_level,
		       COALESCE(so.headteacher_signed, FALSE) AS signed
		FROM students s
		JOIN marks m ON m.student_id = s.id
		JOIN assessments a ON a.id = m.assessment_id
		  AND a.class_id = $1 AND a.term_id = $2
		  AND a.assessment_type IN ('cat1','cat2','cat3')
		JOIN subjects subj ON subj.id = a.subject_id
		LEFT JOIN cbc_signoffs so ON so.student_id = s.id AND so.term_id = $2
		WHERE s.current_class_id = $1 AND s.is_active = TRUE
		ORDER BY s.last_name, s.first_name, subj.name, a.assessment_type`, classID, termID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type row struct {
		StudentID   uuid.UUID
		FullName    string
		AdmNo       string
		Subject     string
		AssessType  string
		Level       string
		HtSigned    bool
	}
	var data []row
	for rows.Next() {
		var d row
		if rows.Scan(&d.StudentID, &d.FullName, &d.AdmNo, &d.Subject, &d.AssessType, &d.Level, &d.HtSigned) == nil {
			data = append(data, d)
		}
	}

	// Group by student
	type subjectAssess struct {
		Cat1 string `json:"cat1"`
		Cat2 string `json:"cat2"`
		Cat3 string `json:"cat3"`
	}
	type studentSummary struct {
		StudentID   string                       `json:"student_id"`
		FullName    string                       `json:"full_name"`
		AdmNo       string                       `json:"admission_no"`
		HtSigned    bool                         `json:"headteacher_signed"`
		Subjects    map[string]*subjectAssess    `json:"subjects"`
	}

	studentMap := map[uuid.UUID]*studentSummary{}
	var order []uuid.UUID
	for _, d := range data {
		if _, ok := studentMap[d.StudentID]; !ok {
			studentMap[d.StudentID] = &studentSummary{
				StudentID: d.StudentID.String(), FullName: d.FullName,
				AdmNo: d.AdmNo, HtSigned: d.HtSigned,
				Subjects: map[string]*subjectAssess{},
			}
			order = append(order, d.StudentID)
		}
		ss := studentMap[d.StudentID]
		if ss.Subjects[d.Subject] == nil {
			ss.Subjects[d.Subject] = &subjectAssess{}
		}
		switch d.AssessType {
		case "cat1":
			ss.Subjects[d.Subject].Cat1 = d.Level
		case "cat2":
			ss.Subjects[d.Subject].Cat2 = d.Level
		case "cat3":
			ss.Subjects[d.Subject].Cat3 = d.Level
		}
	}

	out := make([]*studentSummary, 0, len(order))
	for _, id := range order {
		out = append(out, studentMap[id])
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

// ---------------------------------------------------------------------------
// signoffs
func (s *CBCService) listSignoffs(w http.ResponseWriter, r *http.Request) {
	classID, err := uuid.Parse(r.URL.Query().Get("class_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "class_id required")
		return
	}
	termID, err := uuid.Parse(r.URL.Query().Get("term_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "term_id required")
		return
	}
	// LEFT JOIN so ALL active students in the class appear, with or without a signoff row.
	rows, err := s.pool.Query(r.Context(), `
		SELECT COALESCE(so.id::text, ''),
		       s.id, s.full_name, s.admission_no,
		       COALESCE(so.teacher_comment, ''),
		       COALESCE(so.general_performance_level, ''),
		       COALESCE(so.headteacher_comment, ''),
		       COALESCE(so.headteacher_signed, FALSE),
		       COALESCE(so.opening_date, ''),
		       COALESCE(so.closing_date, '')
		FROM students s
		LEFT JOIN cbc_signoffs so ON so.student_id = s.id AND so.term_id = $2
		WHERE s.current_class_id = $1 AND s.is_active = TRUE
		ORDER BY s.last_name, s.first_name`, classID, termID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var signoffID, name, admNo, tc, gpl, htc, od, cd string
		var sid uuid.UUID
		var signed bool
		if rows.Scan(&signoffID, &sid, &name, &admNo, &tc, &gpl, &htc, &signed, &od, &cd) == nil {
			out = append(out, map[string]interface{}{
				"id": signoffID, "student_id": sid, "full_name": name,
				"admission_no": admNo,
				"teacher_comment": tc, "general_performance_level": gpl,
				"headteacher_comment": htc, "headteacher_signed": signed,
				"opening_date": od, "closing_date": cd,
			})
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *CBCService) upsertSignoff(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StudentID                string `json:"student_id"`
		TermID                   string `json:"term_id"`
		ClassID                  string `json:"class_id"`
		TeacherComment           string `json:"teacher_comment"`
		GeneralPerformanceLevel  string `json:"general_performance_level"`
		HeadteacherComment       string `json:"headteacher_comment"`
		HeadteacherSigned        bool   `json:"headteacher_signed"`
		OpeningDate              string `json:"opening_date"`
		ClosingDate              string `json:"closing_date"`
	}
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	sid, err1 := uuid.Parse(body.StudentID)
	tid, err2 := uuid.Parse(body.TermID)
	cid, err3 := uuid.Parse(body.ClassID)
	if err1 != nil || err2 != nil || err3 != nil {
		httpx.ErrBadRequest(w, "student_id, term_id, class_id required")
		return
	}

	caller := callerID(r)
	role := callerRole(r)
	isLead := hasRole(role, types.AdminRoles)
	if !isLead {
		// A teacher writes the class-teacher comment for their own class
		// only, and can never sign or overwrite the headteacher's part.
		var isClassTeacher bool
		if err := s.pool.QueryRow(r.Context(),
			`SELECT EXISTS (SELECT 1 FROM class_groups WHERE id = $1 AND class_teacher_id = $2)`, cid, caller,
		).Scan(&isClassTeacher); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		if !isClassTeacher {
			httpx.ErrForbidden(w, "only the class teacher or headteacher can comment on this report")
			return
		}
		var htComment string
		var htSigned bool
		_ = s.pool.QueryRow(r.Context(), `SELECT headteacher_comment, headteacher_signed FROM cbc_signoffs
			WHERE student_id = $1 AND term_id = $2`, sid, tid).Scan(&htComment, &htSigned)
		body.HeadteacherComment, body.HeadteacherSigned = htComment, htSigned
	}
	var htID *uuid.UUID
	if isLead && body.HeadteacherSigned {
		htID = &caller
	}

	var id uuid.UUID
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO cbc_signoffs (student_id, term_id, class_id, teacher_comment, general_performance_level,
		    headteacher_comment, headteacher_signed, headteacher_id, headteacher_signed_at,
		    opening_date, closing_date, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
		    CASE WHEN $7 THEN NOW() ELSE NULL END,
		    $9,$10,$11)
		ON CONFLICT (student_id, term_id) DO UPDATE SET
		    teacher_comment          = EXCLUDED.teacher_comment,
		    general_performance_level = EXCLUDED.general_performance_level,
		    headteacher_comment      = EXCLUDED.headteacher_comment,
		    headteacher_signed       = EXCLUDED.headteacher_signed,
		    -- a teacher re-saving their comment keeps the existing signature as-is
		    headteacher_id           = CASE WHEN EXCLUDED.headteacher_signed
		                                    THEN COALESCE(EXCLUDED.headteacher_id, cbc_signoffs.headteacher_id) END,
		    headteacher_signed_at    = CASE WHEN EXCLUDED.headteacher_signed AND cbc_signoffs.headteacher_signed
		                                    THEN cbc_signoffs.headteacher_signed_at
		                                    WHEN EXCLUDED.headteacher_signed THEN NOW() END,
		    opening_date             = EXCLUDED.opening_date,
		    closing_date             = EXCLUDED.closing_date,
		    updated_at               = NOW()
		RETURNING id`,
		sid, tid, cid, body.TeacherComment, body.GeneralPerformanceLevel,
		body.HeadteacherComment, body.HeadteacherSigned, htID,
		body.OpeningDate, body.ClosingDate, caller,
	).Scan(&id)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"id": id, "status": "saved"})
}

// ---------------------------------------------------------------------------
// cbcDistribution — GET /api/cbc/analytics/distribution/?term_id=X
//
// Returns EE/ME/AE/BE counts across all marks for dashboard pie/donut chart.
func (s *CBCService) cbcDistribution(w http.ResponseWriter, r *http.Request) {
	termQ := r.URL.Query().Get("term_id")

	var query string
	var args []interface{}

	if termQ != "" {
		tid, err := uuid.Parse(termQ)
		if err != nil {
			httpx.ErrBadRequest(w, "invalid term_id")
			return
		}
		query = `
			SELECT m.competency_level, COUNT(*) AS cnt
			FROM marks m
			JOIN assessments a ON a.id = m.assessment_id AND a.term_id = $1
			WHERE m.competency_level IS NOT NULL
			GROUP BY m.competency_level`
		args = []interface{}{tid}
	} else {
		query = `
			SELECT competency_level, COUNT(*) AS cnt
			FROM marks
			WHERE competency_level IS NOT NULL
			GROUP BY competency_level`
	}

	rows, err := s.pool.Query(r.Context(), query, args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	counts := map[string]int{"EE": 0, "ME": 0, "AE": 0, "BE": 0}
	for rows.Next() {
		var level string
		var cnt int
		if rows.Scan(&level, &cnt) == nil {
			counts[level] = cnt
		}
	}

	total := 0
	for _, v := range counts { total += v }

	type entry struct {
		Level   string  `json:"level"`
		Count   int     `json:"count"`
		Percent float64 `json:"percent"`
	}
	order := []string{"EE", "ME", "AE", "BE"}
	out := make([]entry, 0, 4)
	for _, l := range order {
		pct := 0.0
		if total > 0 {
			pct = float64(counts[l]) / float64(total) * 100
		}
		out = append(out, entry{Level: l, Count: counts[l], Percent: pct})
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "total": total})
}
