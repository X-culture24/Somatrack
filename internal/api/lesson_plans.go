package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// LessonPlansService is the real backend for CBC lesson planning — it
// replaced a registerPlaceholderCRUD stub. A teacher only ever sees and
// edits their own plans (teacher_id is always the caller, never trusted
// from the request body, so one teacher can't edit another's plan), and
// only for subjects they're assigned in that class. Admin-tier roles (the
// headteacher) see everyone's, newest change first, review them, and see
// per-subject coverage so a subject nobody is planning for stands out.
type LessonPlansService struct {
	pool *pgxpool.Pool
}

func NewLessonPlansService(pool *pgxpool.Pool) *LessonPlansService {
	return &LessonPlansService{pool: pool}
}

func (s *LessonPlansService) RegisterRoutes(r chi.Router) {
	r.Route("/lesson-plans/", func(r chi.Router) {
		r.Get("/", s.list)
		r.Post("/", s.create)
		r.Get("/coverage/", s.coverage)
		r.Post("/{id}/review/", adminOnly()(http.HandlerFunc(s.review)).ServeHTTP)
		r.Get("/{id}/", s.get)
		r.Put("/{id}/", s.update)
		r.Patch("/{id}/", s.update)
		r.Delete("/{id}/", s.delete)
	})
}

type LessonPlan struct {
	ID                 uuid.UUID `json:"id"`
	TermID             uuid.UUID `json:"term_id"` // defaults to the current term
	SubjectID          uuid.UUID `json:"subject_id" validate:"required"`
	ClassID            uuid.UUID `json:"class_id" validate:"required"`
	TeacherID          uuid.UUID `json:"teacher_id,omitempty"`
	WeekNumber         int       `json:"week_number" validate:"required"`
	LessonDate         string    `json:"lesson_date" validate:"required"`
	Topic              string    `json:"topic" validate:"required"`
	SubTopic           string    `json:"sub_topic"`
	LearningObjectives string    `json:"learning_objectives"`
	LearningOutcomes   string    `json:"learning_outcomes"`
	TeachingMethods    string    `json:"teaching_methods"`
	Materials          string    `json:"materials"`
	Introduction       string    `json:"introduction"`
	LessonDevelopment  string    `json:"lesson_development"`
	Conclusion         string    `json:"conclusion"`
	Assessment         string    `json:"assessment"`
	Homework           string    `json:"homework"`
	Reflection         string    `json:"reflection"`
	Status             string    `json:"status" validate:"omitempty,oneof=draft submitted"`
	TeacherName        string    `json:"teacher_name,omitempty"`
	SubjectName        string    `json:"subject_name,omitempty"`
	ClassName          string    `json:"class_name,omitempty"`
	TermName           string    `json:"term_name,omitempty"`

	ReviewComment  string     `json:"review_comment"`
	ReviewedByID   *uuid.UUID `json:"reviewed_by_id"`
	ReviewedByName string     `json:"reviewed_by_name"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	SubmittedAt    *time.Time `json:"submitted_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Teachers move a plan between draft and submitted; approved, reviewed and
// returned are the headteacher's verdicts, set through /review/.
var teacherPlanStatuses = []string{"draft", "submitted"}
var reviewStatuses = []string{"approved", "reviewed", "returned"}

const lessonPlanCols = `lp.id, lp.term_id, lp.subject_id, lp.class_id, lp.teacher_id, lp.week_number,
	lp.lesson_date::text, lp.topic, lp.sub_topic, lp.learning_objectives, lp.learning_outcomes,
	lp.teaching_methods, lp.materials, lp.introduction, lp.lesson_development, lp.conclusion,
	lp.assessment, lp.homework, lp.reflection, lp.status,
	COALESCE(u.first_name || ' ' || u.last_name, ''), COALESCE(subj.name, ''), COALESCE(cg.name, ''),
	COALESCE(t.name, ''), lp.review_comment, lp.approved_by_id,
	COALESCE(rv.first_name || ' ' || rv.last_name, ''), lp.reviewed_at, lp.submitted_at, lp.created_at, lp.updated_at`

const lessonPlanJoin = `lesson_plans lp
	LEFT JOIN users u ON u.id = lp.teacher_id
	LEFT JOIN users rv ON rv.id = lp.approved_by_id
	LEFT JOIN subjects subj ON subj.id = lp.subject_id
	LEFT JOIN class_groups cg ON cg.id = lp.class_id
	LEFT JOIN terms t ON t.id = lp.term_id`

func scanLessonPlan(row interface {
	Scan(dest ...interface{}) error
}) (LessonPlan, error) {
	var p LessonPlan
	err := row.Scan(&p.ID, &p.TermID, &p.SubjectID, &p.ClassID, &p.TeacherID, &p.WeekNumber,
		&p.LessonDate, &p.Topic, &p.SubTopic, &p.LearningObjectives, &p.LearningOutcomes,
		&p.TeachingMethods, &p.Materials, &p.Introduction, &p.LessonDevelopment, &p.Conclusion,
		&p.Assessment, &p.Homework, &p.Reflection, &p.Status,
		&p.TeacherName, &p.SubjectName, &p.ClassName,
		&p.TermName, &p.ReviewComment, &p.ReviewedByID, &p.ReviewedByName, &p.ReviewedAt, &p.SubmittedAt,
		&p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func isAdminRole(role types.Role) bool {
	if role == types.RoleSystemAdmin {
		return true
	}
	for _, a := range types.AdminRoles {
		if a == role {
			return true
		}
	}
	return false
}

// list — GET /lesson-plans/ — newest change first, so the headteacher sees
// what teachers have just written or updated at the top. Filters:
// teacher_id, class_id, subject_id, term_id, status, updated_since (RFC3339
// or YYYY-MM-DD). Teachers only ever get their own; parents get plans for
// their children's classes.
func (s *LessonPlansService) list(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("role").(types.Role)
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	q := r.URL.Query()

	var where []string
	var args []interface{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	for _, f := range [][2]string{{"lp.class_id", "class_id"}, {"lp.subject_id", "subject_id"}, {"lp.term_id", "term_id"}, {"lp.teacher_id", "teacher_id"}} {
		if v := q.Get(f[1]); v != "" {
			id, err := uuid.Parse(v)
			if err != nil {
				httpx.ErrBadRequest(w, "invalid "+f[1])
				return
			}
			where = append(where, f[0]+" = "+arg(id))
		}
	}
	if v := q.Get("status"); v != "" {
		where = append(where, "lp.status = "+arg(v))
	}
	if v := q.Get("updated_since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			if t, err = time.Parse("2006-01-02", v); err != nil {
				httpx.ErrBadRequest(w, "updated_since must be RFC3339 or YYYY-MM-DD")
				return
			}
		}
		where = append(where, "lp.updated_at >= "+arg(t))
	}

	limit := 200
	switch {
	case isAdminRole(role):
	// A parent has no teacher_id of their own, so the teacher-scoped branch
	// below would always return empty for them — this is what makes "see
	// what's being taught in my child's class" (their "work") possible at
	// all. Scoped to classes their own children are actually in, never an
	// arbitrary class_id.
	case role == types.RoleParent:
		where = append(where, `lp.class_id IN (
			SELECT st.current_class_id FROM guardians g
			JOIN student_guardians sg ON sg.guardian_id = g.id
			JOIN students st ON st.id = sg.student_id
			WHERE g.user_id = `+arg(userID)+` AND st.current_class_id IS NOT NULL)`)
		limit = 100
	default:
		where = append(where, "lp.teacher_id = "+arg(userID))
	}
	if len(where) == 0 {
		where = []string{"TRUE"}
	}
	rows, err := s.pool.Query(r.Context(), `SELECT `+lessonPlanCols+` FROM `+lessonPlanJoin+`
		WHERE `+strings.Join(where, " AND ")+fmt.Sprintf(` ORDER BY lp.updated_at DESC LIMIT %d`, limit), args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []LessonPlan{}
	for rows.Next() {
		p, err := scanLessonPlan(rows)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, p)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *LessonPlansService) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	var body LessonPlan
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.Status == "" {
		body.Status = "draft"
	}
	if body.TermID == uuid.Nil {
		if err := s.pool.QueryRow(r.Context(),
			`SELECT id FROM terms WHERE is_current ORDER BY start_date DESC LIMIT 1`).Scan(&body.TermID); err != nil {
			fieldErrors(w, map[string]string{"term_id": "no current term is set — pick a term or ask the admin to set one"})
			return
		}
	}
	ok, err := callerTeaches(r.Context(), s.pool, r, body.ClassID, body.SubjectID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "you aren't assigned this subject in this class")
		return
	}
	err = s.pool.QueryRow(r.Context(), `
		INSERT INTO lesson_plans (term_id, subject_id, class_id, teacher_id, week_number, lesson_date, topic,
			sub_topic, learning_objectives, learning_outcomes, teaching_methods, materials, introduction,
			lesson_development, conclusion, assessment, homework, reflection, status, submitted_by_id, submitted_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19::text,$4,
			CASE WHEN $19::text = 'submitted' THEN NOW() END) RETURNING id`,
		body.TermID, body.SubjectID, body.ClassID, userID, body.WeekNumber, body.LessonDate, body.Topic,
		body.SubTopic, body.LearningObjectives, body.LearningOutcomes, body.TeachingMethods, body.Materials,
		body.Introduction, body.LessonDevelopment, body.Conclusion, body.Assessment, body.Homework,
		body.Reflection, body.Status).Scan(&body.ID)
	if err != nil {
		resultsDBError(w, err)
		return
	}
	p, err := scanLessonPlan(s.pool.QueryRow(r.Context(), `SELECT `+lessonPlanCols+` FROM `+lessonPlanJoin+` WHERE lp.id = $1`, body.ID))
	if err != nil {
		httpx.ErrNotFound(w, "lesson plan not found")
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

// ownsOrAdmin loads a plan's teacher_id and checks the caller either wrote
// it or holds an admin-tier role — used by get/update/delete so a teacher
// can't read or edit a colleague's lesson plan by guessing its id.
func (s *LessonPlansService) ownsOrAdmin(r *http.Request, id uuid.UUID) (bool, error) {
	role, _ := r.Context().Value("role").(types.Role)
	if isAdminRole(role) {
		return true, nil
	}
	userID, _ := r.Context().Value("user_id").(uuid.UUID)
	var teacherID uuid.UUID
	if err := s.pool.QueryRow(r.Context(), `SELECT teacher_id FROM lesson_plans WHERE id = $1`, id).Scan(&teacherID); err != nil {
		return false, err
	}
	return teacherID == userID, nil
}

func (s *LessonPlansService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	ok, err := s.ownsOrAdmin(r, id)
	if err != nil {
		httpx.ErrNotFound(w, "lesson plan not found")
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "not your lesson plan")
		return
	}
	p, err := scanLessonPlan(s.pool.QueryRow(r.Context(), `SELECT `+lessonPlanCols+` FROM `+lessonPlanJoin+` WHERE lp.id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "lesson plan not found")
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (s *LessonPlansService) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	ok, err := s.ownsOrAdmin(r, id)
	if err != nil {
		httpx.ErrNotFound(w, "lesson plan not found")
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "not your lesson plan")
		return
	}
	var body struct {
		TermID             *uuid.UUID `json:"term_id,omitempty"`
		SubjectID          *uuid.UUID `json:"subject_id,omitempty"`
		ClassID            *uuid.UUID `json:"class_id,omitempty"`
		WeekNumber         *int       `json:"week_number,omitempty"`
		LessonDate         *string    `json:"lesson_date,omitempty"`
		Topic              *string    `json:"topic,omitempty"`
		SubTopic           *string    `json:"sub_topic,omitempty"`
		LearningObjectives *string    `json:"learning_objectives,omitempty"`
		LearningOutcomes   *string    `json:"learning_outcomes,omitempty"`
		TeachingMethods    *string    `json:"teaching_methods,omitempty"`
		Materials          *string    `json:"materials,omitempty"`
		Introduction       *string    `json:"introduction,omitempty"`
		LessonDevelopment  *string    `json:"lesson_development,omitempty"`
		Conclusion         *string    `json:"conclusion,omitempty"`
		Assessment         *string    `json:"assessment,omitempty"`
		Homework           *string    `json:"homework,omitempty"`
		Reflection         *string    `json:"reflection,omitempty"`
		Status             *string    `json:"status,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	if body.Status != nil && *body.Status != "" && !containsStr(teacherPlanStatuses, *body.Status) {
		fieldErrors(w, map[string]string{"status": "must be draft or submitted — the headteacher approves through /review/"})
		return
	}
	var curClass, curSubject uuid.UUID
	var curStatus string
	if err := s.pool.QueryRow(r.Context(), `SELECT class_id, subject_id, status FROM lesson_plans WHERE id = $1`, id).
		Scan(&curClass, &curSubject, &curStatus); err != nil {
		httpx.ErrNotFound(w, "lesson plan not found")
		return
	}
	if body.ClassID != nil || body.SubjectID != nil {
		c, sub := curClass, curSubject
		if body.ClassID != nil {
			c = *body.ClassID
		}
		if body.SubjectID != nil {
			sub = *body.SubjectID
		}
		if ok, err := callerTeaches(r.Context(), s.pool, r, c, sub); err != nil || !ok {
			httpx.ErrForbidden(w, "you aren't assigned this subject in this class")
			return
		}
	}
	// A teacher changing a plan the headteacher already looked at sends it
	// back for review, so the headteacher always sees the updated version.
	if !isAdminRole(callerRole(r)) && body.Status == nil && containsStr(reviewStatuses, curStatus) {
		resubmit := "submitted"
		body.Status = &resubmit
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE lesson_plans SET
			term_id=COALESCE($1, term_id), subject_id=COALESCE($2, subject_id), class_id=COALESCE($3, class_id),
			week_number=COALESCE($4, week_number), lesson_date=COALESCE($5, lesson_date),
			topic=COALESCE(NULLIF($6,''), topic), sub_topic=COALESCE($7, sub_topic),
			learning_objectives=COALESCE($8, learning_objectives), learning_outcomes=COALESCE($9, learning_outcomes),
			teaching_methods=COALESCE($10, teaching_methods), materials=COALESCE($11, materials),
			introduction=COALESCE($12, introduction), lesson_development=COALESCE($13, lesson_development),
			conclusion=COALESCE($14, conclusion), assessment=COALESCE($15, assessment),
			homework=COALESCE($16, homework), reflection=COALESCE($17, reflection),
			submitted_at=CASE WHEN $18::text = 'submitted' THEN NOW() ELSE submitted_at END,
			status=COALESCE(NULLIF($18::text,''), status), updated_at=NOW()
		WHERE id = $19`,
		body.TermID, body.SubjectID, body.ClassID, body.WeekNumber, body.LessonDate, body.Topic, body.SubTopic,
		body.LearningObjectives, body.LearningOutcomes, body.TeachingMethods, body.Materials, body.Introduction,
		body.LessonDevelopment, body.Conclusion, body.Assessment, body.Homework, body.Reflection, body.Status, id)
	if err != nil {
		resultsDBError(w, err)
		return
	}
	p, err := scanLessonPlan(s.pool.QueryRow(r.Context(), `SELECT `+lessonPlanCols+` FROM `+lessonPlanJoin+` WHERE lp.id = $1`, id))
	if err != nil {
		httpx.ErrNotFound(w, "lesson plan not found")
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (s *LessonPlansService) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	ok, err := s.ownsOrAdmin(r, id)
	if err != nil {
		httpx.ErrNotFound(w, "lesson plan not found")
		return
	}
	if !ok {
		httpx.ErrForbidden(w, "not your lesson plan")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM lesson_plans WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}

// review — POST /lesson-plans/{id}/review/ — the headteacher's verdict:
// approved, reviewed (seen, no action needed) or returned (needs changes),
// with an optional comment the teacher sees on the plan.
func (s *LessonPlansService) review(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrBadRequest(w, "request body must be a JSON object")
		return
	}
	if !containsStr(reviewStatuses, body.Status) {
		fieldErrors(w, map[string]string{"status": "must be approved, reviewed or returned"})
		return
	}
	if body.Status == "returned" && strings.TrimSpace(body.Comment) == "" {
		fieldErrors(w, map[string]string{"comment": "say what needs changing when returning a plan"})
		return
	}
	tag, err := s.pool.Exec(r.Context(), `
		UPDATE lesson_plans SET status = $2, review_comment = $3, approved_by_id = $4, reviewed_at = NOW()
		WHERE id = $1`, id, body.Status, strings.TrimSpace(body.Comment), callerID(r))
	if err != nil {
		resultsDBError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.ErrNotFound(w, "lesson plan not found")
		return
	}
	p, err := scanLessonPlan(s.pool.QueryRow(r.Context(), `SELECT `+lessonPlanCols+` FROM `+lessonPlanJoin+` WHERE lp.id = $1`, id))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

type PlanCoverage struct {
	ClassID        uuid.UUID  `json:"class_id"`
	ClassName      string     `json:"class_name"`
	SubjectID      uuid.UUID  `json:"subject_id"`
	SubjectName    string     `json:"subject_name"`
	TeacherID      *uuid.UUID `json:"teacher_id"`
	TeacherName    string     `json:"teacher_name"`
	Plans          int        `json:"plans"`
	Drafts         int        `json:"drafts"`
	Submitted      int        `json:"submitted"`
	Approved       int        `json:"approved"`
	Returned       int        `json:"returned"`
	WeeksCovered   int        `json:"weeks_covered"`
	LastWeek       *int       `json:"last_week"`
	LastUpdated    *time.Time `json:"last_updated"`
	AwaitingReview bool       `json:"awaiting_review"`
}

// coverage — GET /lesson-plans/coverage/?term_id=&class_id=&teacher_id= —
// one row per subject on each class's curriculum, with how many plans its
// teacher has written this term and when they last touched one. A subject
// with plans = 0 is one nobody is planning for. Teachers see their own rows.
func (s *LessonPlansService) coverage(w http.ResponseWriter, r *http.Request) {
	rl := callerRole(r)
	if !isAdminRole(rl) && !hasRole(rl, types.TeacherRoles) {
		httpx.ErrForbidden(w, "staff only")
		return
	}
	var termID uuid.UUID
	if v := r.URL.Query().Get("term_id"); v != "" {
		var err error
		if termID, err = uuid.Parse(v); err != nil {
			httpx.ErrBadRequest(w, "invalid term_id")
			return
		}
	} else if err := s.pool.QueryRow(r.Context(),
		`SELECT id FROM terms WHERE is_current ORDER BY start_date DESC LIMIT 1`).Scan(&termID); err != nil {
		httpx.ErrBadRequest(w, "no current term is set — pass term_id")
		return
	}
	classID := queryUUID(r, "class_id")
	teacherID := queryUUID(r, "teacher_id")
	if !isAdminRole(rl) {
		me := callerID(r)
		teacherID = &me
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT cs.class_id, cg.name, cs.subject_id, subj.name, cs.teacher_id,
		       COALESCE(u.first_name || ' ' || u.last_name, ''),
		       count(lp.id)::int,
		       count(lp.id) FILTER (WHERE lp.status = 'draft')::int,
		       count(lp.id) FILTER (WHERE lp.status = 'submitted')::int,
		       count(lp.id) FILTER (WHERE lp.status IN ('approved','reviewed'))::int,
		       count(lp.id) FILTER (WHERE lp.status = 'returned')::int,
		       count(DISTINCT lp.week_number)::int,
		       max(lp.week_number),
		       max(lp.updated_at)
		FROM class_subjects cs
		JOIN class_groups cg ON cg.id = cs.class_id
		JOIN subjects subj ON subj.id = cs.subject_id
		LEFT JOIN users u ON u.id = cs.teacher_id
		LEFT JOIN lesson_plans lp ON lp.class_id = cs.class_id AND lp.subject_id = cs.subject_id AND lp.term_id = $1
		WHERE ($2::uuid IS NULL OR cs.class_id = $2) AND ($3::uuid IS NULL OR cs.teacher_id = $3)
		GROUP BY cs.class_id, cg.name, cg.grade, cs.subject_id, subj.name, cs.teacher_id, u.first_name, u.last_name
		ORDER BY cg.grade, cg.name, subj.name`, termID, classID, teacherID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []PlanCoverage{}
	summary := map[string]int{"subjects": 0, "without_plans": 0, "without_teacher": 0, "awaiting_review": 0}
	for rows.Next() {
		var c PlanCoverage
		if err := rows.Scan(&c.ClassID, &c.ClassName, &c.SubjectID, &c.SubjectName, &c.TeacherID, &c.TeacherName,
			&c.Plans, &c.Drafts, &c.Submitted, &c.Approved, &c.Returned, &c.WeeksCovered, &c.LastWeek, &c.LastUpdated); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
		c.AwaitingReview = c.Submitted > 0
		summary["subjects"]++
		if c.Plans == 0 {
			summary["without_plans"]++
		}
		if c.TeacherID == nil {
			summary["without_teacher"]++
		}
		if c.AwaitingReview {
			summary["awaiting_review"]++
		}
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"term_id": termID, "summary": summary, "results": out, "count": len(out),
	})
}
