package api

import (
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stmaryskabete/lms/internal/shared/httpx"
)

// CurriculumService is the real backend behind the "CBC Curriculum" admin
// page — it replaced a placeholderHandler stub. GET /school/curriculum/
// aggregates class_subjects (which subject, and whose teacher, is taught to
// each class) grouped by grade and learning area; class-subjects/ is the
// small CRUD surface for actually assigning a subject+teacher to a class,
// since class_subjects had zero rows and no way to add any.
type CurriculumService struct {
	pool *pgxpool.Pool
}

func NewCurriculumService(pool *pgxpool.Pool) *CurriculumService {
	return &CurriculumService{pool: pool}
}

func (s *CurriculumService) RegisterRoutes(r chi.Router) {
	r.Get("/school/curriculum/", s.curriculum)
	r.Route("/class-subjects/", func(r chi.Router) {
		r.Get("/", s.listClassSubjects)
		r.Post("/", adminOnly()(http.HandlerFunc(s.createClassSubject)).ServeHTTP)
		r.Delete("/{id}/", adminOnly()(http.HandlerFunc(s.deleteClassSubject)).ServeHTTP)
	})
}

var gradeLabels = map[int]string{
	-2: "Playgroup", -1: "PP1", 0: "PP2",
	1: "Grade 1", 2: "Grade 2", 3: "Grade 3", 4: "Grade 4", 5: "Grade 5",
	6: "Grade 6", 7: "Grade 7", 8: "Grade 8", 9: "Grade 9",
	10: "Grade 10", 11: "Grade 11", 12: "Grade 12", 13: "Grade 13",
}

func gradeLabel(grade int) string {
	if l, ok := gradeLabels[grade]; ok {
		return l
	}
	return "Grade"
}

func uniformBand(grade int) string {
	switch {
	case grade <= 0:
		return "Kinder"
	case grade <= 3:
		return "1-3"
	case grade <= 6:
		return "4-6"
	default:
		return "JSS"
	}
}

type curriculumSubject struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Teacher string    `json:"teacher,omitempty"`
}

type curriculumLearningArea struct {
	Area     string              `json:"area"`
	Subjects []curriculumSubject `json:"subjects"`
}

type curriculumGradeRow struct {
	Grade         string                    `json:"grade"`
	Band          string                    `json:"band"`
	LearningAreas []curriculumLearningArea  `json:"learning_areas"`
}

func (s *CurriculumService) curriculum(w http.ResponseWriter, r *http.Request) {
	gradeFilter := r.URL.Query().Get("grade")

	rows, err := s.pool.Query(r.Context(), `
		SELECT DISTINCT ON (cg.grade, subj.id)
			cg.grade, subj.id, subj.name, subj.learning_area,
			COALESCE(u.first_name || ' ' || u.last_name, '')
		FROM class_groups cg
		JOIN class_subjects cs ON cs.class_id = cg.id
		JOIN subjects subj ON subj.id = cs.subject_id AND subj.is_active = TRUE
		JOIN academic_years ay ON ay.id = cg.academic_year_id AND ay.is_active = TRUE
		LEFT JOIN users u ON u.id = cs.teacher_id
		ORDER BY cg.grade, subj.id, u.first_name NULLS LAST`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type flatRow struct {
		Grade   int
		SubjID  uuid.UUID
		Name    string
		Area    string
		Teacher string
	}
	var flat []flatRow
	for rows.Next() {
		var f flatRow
		if err := rows.Scan(&f.Grade, &f.SubjID, &f.Name, &f.Area, &f.Teacher); err != nil {
			continue
		}
		flat = append(flat, f)
	}

	byGrade := map[int]map[string][]curriculumSubject{}
	for _, f := range flat {
		if byGrade[f.Grade] == nil {
			byGrade[f.Grade] = map[string][]curriculumSubject{}
		}
		byGrade[f.Grade][f.Area] = append(byGrade[f.Grade][f.Area], curriculumSubject{ID: f.SubjID, Name: f.Name, Teacher: f.Teacher})
	}

	grades := make([]int, 0, len(byGrade))
	for g := range byGrade {
		grades = append(grades, g)
	}
	sort.Ints(grades)

	out := []curriculumGradeRow{}
	for _, g := range grades {
		label := gradeLabel(g)
		if gradeFilter != "" && gradeFilter != label {
			continue
		}
		areas := make([]string, 0, len(byGrade[g]))
		for a := range byGrade[g] {
			areas = append(areas, a)
		}
		sort.Strings(areas)
		row := curriculumGradeRow{Grade: label, Band: uniformBand(g)}
		for _, a := range areas {
			row.LearningAreas = append(row.LearningAreas, curriculumLearningArea{Area: a, Subjects: byGrade[g][a]})
		}
		out = append(out, row)
	}
	httpx.JSON(w, http.StatusOK, out)
}

type ClassSubject struct {
	ID          uuid.UUID  `json:"id"`
	ClassID     uuid.UUID  `json:"class_id" validate:"required"`
	SubjectID   uuid.UUID  `json:"subject_id" validate:"required"`
	TeacherID   *uuid.UUID `json:"teacher_id,omitempty"`
	IsElective  bool       `json:"is_elective"`
	ClassName   string     `json:"class_name,omitempty"`
	SubjectName string     `json:"subject_name,omitempty"`
	TeacherName string     `json:"teacher_name,omitempty"`
}

const classSubjectCols = `cs.id, cs.class_id, cs.subject_id, cs.teacher_id, cs.is_elective,
	COALESCE(cg.name, ''), COALESCE(subj.name, ''), COALESCE(u.first_name || ' ' || u.last_name, '')`
const classSubjectJoin = `class_subjects cs
	JOIN class_groups cg ON cg.id = cs.class_id
	JOIN subjects subj ON subj.id = cs.subject_id
	LEFT JOIN users u ON u.id = cs.teacher_id`

func scanClassSubject(row interface {
	Scan(dest ...interface{}) error
}) (ClassSubject, error) {
	var c ClassSubject
	err := row.Scan(&c.ID, &c.ClassID, &c.SubjectID, &c.TeacherID, &c.IsElective, &c.ClassName, &c.SubjectName, &c.TeacherName)
	return c, err
}

func (s *CurriculumService) listClassSubjects(w http.ResponseWriter, r *http.Request) {
	classID := r.URL.Query().Get("class_id")
	var rows pgx.Rows
	var err error
	if classID != "" {
		rows, err = s.pool.Query(r.Context(), `SELECT `+classSubjectCols+` FROM `+classSubjectJoin+`
			WHERE cs.class_id = $1 ORDER BY subj.name`, classID)
	} else {
		rows, err = s.pool.Query(r.Context(), `SELECT `+classSubjectCols+` FROM `+classSubjectJoin+`
			ORDER BY cg.grade, subj.name`)
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []ClassSubject{}
	for rows.Next() {
		c, err := scanClassSubject(rows)
		if err != nil {
			continue
		}
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"results": out, "count": len(out)})
}

func (s *CurriculumService) createClassSubject(w http.ResponseWriter, r *http.Request) {
	var body ClassSubject
	if err := httpx.BindAndValidate(r, &body); err != nil {
		httpx.ErrBadRequest(w, err.Error())
		return
	}
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO class_subjects (class_id, subject_id, teacher_id, is_elective)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		body.ClassID, body.SubjectID, body.TeacherID, body.IsElective).Scan(&body.ID)
	if err != nil {
		if isClassSubjectDuplicate(err) {
			httpx.ErrConflict(w, "this subject is already assigned to this class")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	c, err := scanClassSubject(s.pool.QueryRow(r.Context(), `SELECT `+classSubjectCols+` FROM `+classSubjectJoin+` WHERE cs.id = $1`, body.ID))
	if err != nil {
		httpx.ErrNotFound(w, "class subject not found")
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func isClassSubjectDuplicate(err error) bool {
	return err != nil && strings.Contains(err.Error(), "class_subjects_class_id_subject_id_key")
}

func (s *CurriculumService) deleteClassSubject(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.ErrBadRequest(w, "invalid id")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM class_subjects WHERE id = $1`, id); err != nil {
		httpx.Error(w, http.StatusBadRequest, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id.String()})
}
