package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
	"github.com/stmaryskabete/lms/internal/shared/types"
)

// TeacherService answers "which classes am I assigned to" — previously
// there was no such data at all: class_groups.class_teacher_id and
// class_subjects.teacher_id both existed on the schema, but nothing ever
// read them back for the teacher who holds those assignments (the teacher
// dashboard hardcoded homeroom_classes: [] regardless of DB state).
type TeacherService struct {
	pool *pgxpool.Pool
}

func NewTeacherService(pool *pgxpool.Pool) *TeacherService {
	return &TeacherService{pool: pool}
}

func (s *TeacherService) RegisterRoutes(r chi.Router) {
	r.Get("/teacher/my-classes/", s.myClasses)
	r.Route("/teachers/", func(r chi.Router) {
		r.Get("/", adminOnly()(http.HandlerFunc(s.list)).ServeHTTP)
		r.Get("/{id}/", s.get) // admin, or a teacher looking at themselves
		r.Group(func(r chi.Router) {
			r.Use(adminOnly())
			r.Put("/{id}/assignments/", s.replaceAssignments)
			r.Post("/{id}/assignments/", s.addAssignments)
			r.Delete("/{id}/assignments/{assignment_id}/", s.removeAssignment)
		})
	})
}

type TeacherClass struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Grade          int       `json:"grade"`
	StreamName     string    `json:"stream_name,omitempty"`
	StudentCount   int       `json:"student_count"`
	IsHomeroom     bool      `json:"is_homeroom"`
	SubjectsTaught []string  `json:"subjects_taught"`
}

// myClassesFor is shared with dashboard.teacher() so the dashboard summary
// and the dedicated /teacher/my-classes/ endpoint can never drift apart.
func myClassesFor(ctx context.Context, pool *pgxpool.Pool, teacherID uuid.UUID) ([]TeacherClass, error) {
	rows, err := pool.Query(ctx, `
		SELECT cg.id, cg.name, cg.grade, COALESCE(st.name, ''),
		       (SELECT COUNT(*) FROM students s WHERE s.current_class_id = cg.id),
		       COALESCE(bool_or(cg.class_teacher_id = $1), false), -- NULL when the class has no class teacher
		       COALESCE(array_agg(DISTINCT subj.name) FILTER (WHERE subj.name IS NOT NULL), '{}')
		FROM class_groups cg
		LEFT JOIN streams st ON st.id = cg.stream_id
		LEFT JOIN class_subjects cs ON cs.class_id = cg.id AND cs.teacher_id = $1
		LEFT JOIN subjects subj ON subj.id = cs.subject_id
		WHERE cg.class_teacher_id = $1 OR EXISTS (
			SELECT 1 FROM class_subjects cs2 WHERE cs2.class_id = cg.id AND cs2.teacher_id = $1
		)
		GROUP BY cg.id, cg.name, cg.grade, st.name
		ORDER BY cg.grade, cg.name`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TeacherClass{}
	for rows.Next() {
		var c TeacherClass
		if err := rows.Scan(&c.ID, &c.Name, &c.Grade, &c.StreamName, &c.StudentCount, &c.IsHomeroom, &c.SubjectsTaught); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (s *TeacherService) myClasses(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		httpx.ErrUnauthorized(w, "auth required")
		return
	}
	classes, err := myClassesFor(r.Context(), s.pool, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": classes, "count": len(classes)})
}

// ---------------------------------------------------------------- Teachers & subject assignments
//
// A teacher's subjects are rows in class_subjects with teacher_id set: one
// row per (class, subject), so "Mathematics in Grade 4 East" has exactly one
// teacher. That row is also what callerTeaches checks before letting a
// teacher enter marks or write a lesson plan, so assigning here is what
// gives a teacher access to that subject's mark list.

type SubjectAssignment struct {
	ID          uuid.UUID `json:"id"` // class_subjects.id
	ClassID     uuid.UUID `json:"class_id"`
	ClassName   string    `json:"class_name"`
	Grade       int       `json:"grade"`
	SubjectID   uuid.UUID `json:"subject_id"`
	SubjectName string    `json:"subject_name"`
	SubjectCode string    `json:"subject_code"`
	IsElective  bool      `json:"is_elective"`
}

type HomeroomClass struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Grade int       `json:"grade"`
}

type Teacher struct {
	UserID         uuid.UUID           `json:"id"`
	FullName       string              `json:"full_name"`
	Email          string              `json:"email"`
	Phone          string              `json:"phone"`
	Role           string              `json:"role"`
	IsActive       bool                `json:"is_active"`
	StaffID        *uuid.UUID          `json:"staff_id"`
	EmployeeNo     string              `json:"employee_no"`
	TSCNo          string              `json:"tsc_no"`
	Designation    string              `json:"designation"`
	DepartmentName string              `json:"department_name"`
	Qualification  string              `json:"highest_qualification"`
	Subjects       []string            `json:"subjects"` // distinct subject names, for display
	Assignments    []SubjectAssignment `json:"assignments"`
	HomeroomOf     []HomeroomClass     `json:"homeroom_classes"`
	LessonPlans    int                 `json:"lesson_plans_this_term"`
}

// teacherRoleNames are the users.role values (stored lower-case) that can
// hold teaching assignments.
func teacherRoleNames() []string {
	out := make([]string, 0, len(types.TeacherRoles))
	for _, r := range types.TeacherRoles {
		out = append(out, strings.ToLower(string(r)))
	}
	return out
}

// loadAssignments returns each teacher's class-subject assignments, keyed by
// user id. Shared with StaffService so pulling a staff member who teaches
// also pulls their subjects.
func loadAssignments(ctx context.Context, pool *pgxpool.Pool, userIDs []uuid.UUID) (map[uuid.UUID][]SubjectAssignment, error) {
	rows, err := pool.Query(ctx, `
		SELECT cs.teacher_id, cs.id, cs.class_id, cg.name, cg.grade, cs.subject_id, subj.name, subj.code, cs.is_elective
		FROM class_subjects cs
		JOIN class_groups cg ON cg.id = cs.class_id
		JOIN subjects subj ON subj.id = cs.subject_id
		WHERE cs.teacher_id = ANY($1)
		ORDER BY cg.grade, cg.name, subj.name`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID][]SubjectAssignment{}
	for rows.Next() {
		var tid uuid.UUID
		var a SubjectAssignment
		if err := rows.Scan(&tid, &a.ID, &a.ClassID, &a.ClassName, &a.Grade, &a.SubjectID, &a.SubjectName, &a.SubjectCode, &a.IsElective); err != nil {
			return nil, err
		}
		out[tid] = append(out[tid], a)
	}
	return out, rows.Err()
}

func distinctSubjects(as []SubjectAssignment) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, a := range as {
		if !seen[a.SubjectName] {
			seen[a.SubjectName] = true
			out = append(out, a.SubjectName)
		}
	}
	return out
}

func (s *TeacherService) loadTeachers(ctx context.Context, onlyID *uuid.UUID) ([]Teacher, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, TRIM(COALESCE(u.first_name,'') || ' ' || COALESCE(u.last_name,'')), u.email, COALESCE(u.phone,''),
		       UPPER(u.role), u.is_active, sp.id, COALESCE(sp.employee_no,''), COALESCE(sp.tsc_no,''),
		       COALESCE(sp.designation,''), COALESCE(d.name,''), COALESCE(sp.highest_qualification,''),
		       (SELECT count(*) FROM lesson_plans lp JOIN terms t ON t.id = lp.term_id
		         WHERE lp.teacher_id = u.id AND t.is_current)
		FROM users u
		LEFT JOIN staff_profiles sp ON sp.user_id = u.id
		LEFT JOIN departments d ON d.id = sp.department_id
		WHERE u.role = ANY($1) AND ($2::uuid IS NULL OR u.id = $2)
		ORDER BY u.first_name, u.last_name`, teacherRoleNames(), onlyID)
	if err != nil {
		return nil, err
	}
	var out []Teacher
	var ids []uuid.UUID
	for rows.Next() {
		var t Teacher
		if err := rows.Scan(&t.UserID, &t.FullName, &t.Email, &t.Phone, &t.Role, &t.IsActive, &t.StaffID,
			&t.EmployeeNo, &t.TSCNo, &t.Designation, &t.DepartmentName, &t.Qualification, &t.LessonPlans); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, t)
		ids = append(ids, t.UserID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	assignments, err := loadAssignments(ctx, s.pool, ids)
	if err != nil {
		return nil, err
	}
	homeroom := map[uuid.UUID][]HomeroomClass{}
	hrows, err := s.pool.Query(ctx, `SELECT class_teacher_id, id, name, grade FROM class_groups
		WHERE class_teacher_id = ANY($1) ORDER BY grade, name`, ids)
	if err != nil {
		return nil, err
	}
	for hrows.Next() {
		var tid uuid.UUID
		var c HomeroomClass
		if err := hrows.Scan(&tid, &c.ID, &c.Name, &c.Grade); err != nil {
			hrows.Close()
			return nil, err
		}
		homeroom[tid] = append(homeroom[tid], c)
	}
	hrows.Close()

	for i := range out {
		out[i].Assignments = assignments[out[i].UserID]
		if out[i].Assignments == nil {
			out[i].Assignments = []SubjectAssignment{}
		}
		out[i].Subjects = distinctSubjects(out[i].Assignments)
		out[i].HomeroomOf = homeroom[out[i].UserID]
		if out[i].HomeroomOf == nil {
			out[i].HomeroomOf = []HomeroomClass{}
		}
	}
	return out, nil
}

// list — GET /teachers/?subject_id=&class_id= — every teacher with the
// subjects and classes they teach. The filters answer "who teaches X".
func (s *TeacherService) list(w http.ResponseWriter, r *http.Request) {
	teachers, err := s.loadTeachers(r.Context(), nil)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	subjectID := queryUUID(r, "subject_id")
	classID := queryUUID(r, "class_id")
	out := []Teacher{}
	for _, t := range teachers {
		if subjectID == nil && classID == nil {
			out = append(out, t)
			continue
		}
		for _, a := range t.Assignments {
			if (subjectID == nil || a.SubjectID == *subjectID) && (classID == nil || a.ClassID == *classID) {
				out = append(out, t)
				break
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *TeacherService) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if !isAdminRole(callerRole(r)) && callerID(r) != id {
		httpx.ErrForbidden(w, "admin role required")
		return
	}
	teachers, err := s.loadTeachers(r.Context(), &id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if len(teachers) == 0 {
		httpx.ErrNotFound(w, "teacher not found")
		return
	}
	httpx.JSON(w, http.StatusOK, teachers[0])
}

type assignmentInput struct {
	ClassID   uuid.UUID `json:"class_id"`
	SubjectID uuid.UUID `json:"subject_id"`
}

type assignmentsBody struct {
	Assignments []assignmentInput `json:"assignments"`
	// Reassign takes a subject away from whoever teaches it now. Without it,
	// assigning a subject that already has a teacher is refused with the
	// list of clashes, so nobody is silently removed.
	Reassign bool `json:"reassign"`
}

type assignmentConflict struct {
	ClassID            uuid.UUID `json:"class_id"`
	ClassName          string    `json:"class_name"`
	SubjectID          uuid.UUID `json:"subject_id"`
	SubjectName        string    `json:"subject_name"`
	CurrentTeacherID   uuid.UUID `json:"current_teacher_id"`
	CurrentTeacherName string    `json:"current_teacher_name"`
}

var errNotTeacher = errors.New("not a teacher")

// applyAssignments assigns teacherID to every (class, subject) pair inside
// tx, creating the class_subjects row when the subject isn't on that
// class's curriculum yet. Returns clashes instead of writing when another
// teacher holds a pair and reassign is false.
func applyAssignments(ctx context.Context, tx pgx.Tx, teacherID uuid.UUID, in []assignmentInput, reassign bool) ([]assignmentConflict, map[string]string, error) {
	var role string
	if err := tx.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, teacherID).Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, errNotTeacher
		}
		return nil, nil, err
	}
	isTeacher := false
	for _, r := range teacherRoleNames() {
		if r == role {
			isTeacher = true
		}
	}
	if !isTeacher {
		return nil, nil, errNotTeacher
	}

	fields := map[string]string{}
	var conflicts []assignmentConflict
	for i, a := range in {
		key := "assignments[" + strconv.Itoa(i) + "]"
		var className, subjectName string
		var curTeacher *uuid.UUID
		var curName string
		err := tx.QueryRow(ctx, `
			SELECT cg.name, subj.name, cs.teacher_id, COALESCE(u.first_name || ' ' || u.last_name, '')
			FROM class_groups cg
			CROSS JOIN subjects subj
			LEFT JOIN class_subjects cs ON cs.class_id = cg.id AND cs.subject_id = subj.id
			LEFT JOIN users u ON u.id = cs.teacher_id
			WHERE cg.id = $1 AND subj.id = $2`, a.ClassID, a.SubjectID).Scan(&className, &subjectName, &curTeacher, &curName)
		if errors.Is(err, pgx.ErrNoRows) {
			fields[key] = "class or subject does not exist"
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if curTeacher != nil && *curTeacher != teacherID && !reassign {
			conflicts = append(conflicts, assignmentConflict{a.ClassID, className, a.SubjectID, subjectName, *curTeacher, curName})
		}
	}
	if len(fields) > 0 || len(conflicts) > 0 {
		return conflicts, fields, nil
	}
	for _, a := range in {
		if _, err := tx.Exec(ctx, `
			INSERT INTO class_subjects (class_id, subject_id, teacher_id) VALUES ($1, $2, $3)
			ON CONFLICT (class_id, subject_id) DO UPDATE SET teacher_id = EXCLUDED.teacher_id, updated_at = NOW()`,
			a.ClassID, a.SubjectID, teacherID); err != nil {
			return nil, nil, err
		}
	}
	return nil, nil, nil
}

func (s *TeacherService) writeAssignments(w http.ResponseWriter, r *http.Request, replace bool) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	var body assignmentsBody
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, `body must be {"assignments": [{"class_id", "subject_id"}], "reassign": false}`)
		return
	}
	if !replace && len(body.Assignments) == 0 {
		httpx.ErrBadRequest(w, "assignments is required")
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback(r.Context())

	if replace {
		// Release whatever this teacher held that isn't in the new set; the
		// class keeps the subject, it just has no teacher until reassigned.
		keep := make([]string, 0, len(body.Assignments))
		for _, a := range body.Assignments {
			keep = append(keep, a.ClassID.String()+":"+a.SubjectID.String())
		}
		if _, err := tx.Exec(r.Context(), `
			UPDATE class_subjects SET teacher_id = NULL, updated_at = NOW()
			WHERE teacher_id = $1 AND NOT (class_id::text || ':' || subject_id::text = ANY($2))`, id, keep); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err)
			return
		}
	}
	conflicts, fields, err := applyAssignments(r.Context(), tx, id, body.Assignments, body.Reassign)
	switch {
	case errors.Is(err, errNotTeacher):
		httpx.ErrBadRequest(w, "this user is not a teacher — only teaching roles can be assigned subjects")
		return
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	case len(fields) > 0:
		fieldErrors(w, fields)
		return
	case len(conflicts) > 0:
		httpx.JSON(w, http.StatusConflict, map[string]interface{}{
			"error":     "conflict",
			"message":   "Some of these subjects already have a teacher. Send reassign: true to move them to this teacher.",
			"conflicts": conflicts,
		})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	teachers, err := s.loadTeachers(r.Context(), &id)
	if err != nil || len(teachers) == 0 {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	httpx.JSON(w, http.StatusOK, teachers[0])
}

// replaceAssignments — PUT /teachers/{id}/assignments/ — the teacher's full
// timetable of subjects; anything not listed is unassigned.
func (s *TeacherService) replaceAssignments(w http.ResponseWriter, r *http.Request) {
	s.writeAssignments(w, r, true)
}

// addAssignments — POST /teachers/{id}/assignments/ — adds to what the
// teacher already has.
func (s *TeacherService) addAssignments(w http.ResponseWriter, r *http.Request) {
	s.writeAssignments(w, r, false)
}

func (s *TeacherService) removeAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	aid, err := uuid.Parse(chi.URLParam(r, "assignment_id"))
	if err != nil {
		httpx.ErrBadRequest(w, "invalid assignment id")
		return
	}
	tag, err := s.pool.Exec(r.Context(),
		`UPDATE class_subjects SET teacher_id = NULL, updated_at = NOW() WHERE id = $1 AND teacher_id = $2`, aid, id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.ErrNotFound(w, "this teacher has no such assignment")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
