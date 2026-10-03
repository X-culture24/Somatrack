//go:build integration

package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/stmaryskabete/lms/internal/api"
	"github.com/stmaryskabete/lms/internal/testutil"
)

func marks(rows ...map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"marks": rows}
}

func mark(student uuid.UUID, score interface{}) map[string]interface{} {
	return map[string]interface{}{"student_id": student, "score": score}
}

func TestAssessmentPermissions(t *testing.T) {
	e := newEnv(t)
	body := map[string]interface{}{"name": "CAT 1", "class_id": westID, "subject_id": mathID}
	want(t, e.otherT().post("/api/assessments/", body), 403) // teaches nothing
	want(t, e.parent().get("/api/assessments/"), 403)
	aid := id(t, want(t, e.westT().post("/api/assessments/", body), 201))

	// The class teacher lists it; an unrelated teacher sees nothing.
	if n := want(t, e.westT().get("/api/assessments/"), 200).JSON(t)["count"]; n != 1.0 {
		t.Errorf("class teacher sees %v", n)
	}
	if n := want(t, e.otherT().get("/api/assessments/"), 200).JSON(t)["count"]; n != 0.0 {
		t.Errorf("other teacher sees %v", n)
	}
	want(t, e.otherT().get("/api/assessments/"+aid+"/marksheet/"), 403)
	want(t, e.otherT().put("/api/assessments/"+aid+"/marks/", marks(mark(brianID, 10))), 403)
	// Academic leads can work on any class.
	want(t, e.admin().get("/api/assessments/"+aid+"/marksheet/"), 200)
	// Missing fields are reported per field.
	f := want(t, e.admin().post("/api/assessments/", map[string]interface{}{"total_marks": 0}), 400).JSON(t)["fields"].(map[string]interface{})
	for _, k := range []string{"name", "class_id", "subject_id", "total_marks"} {
		if f[k] == nil {
			t.Errorf("no error for %s: %v", k, f)
		}
	}
}

func TestMarksEntry(t *testing.T) {
	e := newEnv(t)
	w := e.westT()
	aid := id(t, want(t, w.post("/api/assessments/", map[string]interface{}{
		"name": "CAT 1", "assessment_type": "cat1", "class_id": westID, "subject_id": mathID, "total_marks": 30}), 201))

	sheet := want(t, w.get("/api/assessments/"+aid+"/marksheet/"), 200).JSON(t)["results"].([]interface{})
	if len(sheet) != 1 || sheet[0].(map[string]interface{})["full_name"] != "Brian Kamau" {
		t.Fatalf("marksheet should list the class (Brian only): %v", sheet)
	}

	// One bad row rejects the whole sheet and names the rows.
	errs := want(t, w.put("/api/assessments/"+aid+"/marks/", marks(mark(brianID, 31), mark(faithID, 10))), 400).JSON(t)["fields"].(map[string]interface{})
	if !strings.Contains(errs[brianID.String()].(string), "between 0 and 30") || errs[faithID.String()] != "student is not in this class" {
		t.Errorf("row errors: %v", errs)
	}
	if n := testutil.Scalar[int](t, e.pool, `SELECT count(*) FROM marks`); n != 0 {
		t.Fatalf("partial save: %d marks", n)
	}

	want(t, w.put("/api/assessments/"+aid+"/marks/", marks(mark(brianID, 18))), 200)
	row := want(t, w.get("/api/assessments/"+aid+"/marksheet/"), 200).JSON(t)["results"].([]interface{})[0].(map[string]interface{})
	if row["score"] != 18.0 || row["percentage"] != 60.0 || row["grade_letter"] != "ME" || row["competency_level"] != "ME" {
		t.Errorf("stored mark: %v", row)
	}
	// Re-saving updates in place; null clears.
	want(t, w.put("/api/assessments/"+aid+"/marks/", marks(mark(brianID, 27))), 200)
	if n := testutil.Scalar[int](t, e.pool, `SELECT count(*) FROM marks`); n != 1 {
		t.Errorf("upsert made %d rows", n)
	}

	want(t, w.patch("/api/assessments/"+aid+"/", map[string]interface{}{"total_marks": 50}), 409)
	want(t, w.del("/api/assessments/"+aid+"/"), 409)

	r := want(t, w.put("/api/assessments/"+aid+"/marks/", marks(mark(brianID, nil))), 200).JSON(t)
	if r["cleared"] != 1.0 {
		t.Errorf("clear: %v", r)
	}
	want(t, w.del("/api/assessments/"+aid+"/"), 204)
}

func TestLockedThreadFreezesMarks(t *testing.T) {
	e := newEnv(t)
	th := id(t, want(t, e.admin().post("/api/exam-threads/", map[string]interface{}{
		"name": "T1", "term_id": termID, "cats_weight": 30, "main_exam_weight": 70}), 201))
	aid := id(t, want(t, e.westT().post("/api/assessments/", map[string]interface{}{
		"name": "End term", "assessment_type": "endterm", "class_id": westID, "subject_id": mathID, "exam_thread_id": th}), 201))
	want(t, e.westT().put("/api/assessments/"+aid+"/marks/", marks(mark(brianID, 70))), 200)

	want(t, e.admin().patch("/api/exam-threads/"+th+"/", map[string]interface{}{"is_locked": true}), 200)
	want(t, e.westT().put("/api/assessments/"+aid+"/marks/", marks(mark(brianID, 99))), 409)
	want(t, e.westT().post("/api/assessments/", map[string]interface{}{
		"name": "Late CAT", "class_id": westID, "subject_id": mathID, "exam_thread_id": th}), 409)
	if s := testutil.Scalar[float64](t, e.pool, `SELECT score FROM marks`); s != 70 {
		t.Errorf("locked mark changed to %v", s)
	}
}

// The results trail across a promotion, with exam-thread weighting.
func TestResultsTrail(t *testing.T) {
	e := newEnv(t)
	a, w := e.admin(), e.westT()
	th := id(t, want(t, a.post("/api/exam-threads/", map[string]interface{}{
		"name": "Term 1 2026", "term_id": termID, "cats_weight": 30, "main_exam_weight": 70}), 201))
	post := func(c caller, body map[string]interface{}) string { return id(t, want(t, c.post("/api/assessments/", body), 201)) }
	cat := post(w, map[string]interface{}{"name": "CAT", "assessment_type": "cat1", "class_id": westID, "subject_id": mathID, "exam_thread_id": th, "total_marks": 30})
	end := post(w, map[string]interface{}{"name": "End", "assessment_type": "endterm", "class_id": westID, "subject_id": mathID, "exam_thread_id": th})
	eng := post(w, map[string]interface{}{"name": "Eng", "assessment_type": "endterm", "class_id": westID, "subject_id": engID})
	want(t, w.put("/api/assessments/"+cat+"/marks/", marks(mark(brianID, 18))), 200) // 60%
	want(t, w.put("/api/assessments/"+end+"/marks/", marks(mark(brianID, 80))), 200)
	want(t, w.put("/api/assessments/"+eng+"/marks/", marks(mark(brianID, 45))), 200)

	// Promote Brian to East for Term 2.
	term2 := uuid.New()
	testutil.MustExec(t, e.pool, `INSERT INTO terms (id, academic_year_id, name, term_number, start_date, end_date, is_current)
		VALUES ($1, $2, 'Term 2', 2, '2026-05-04', '2026-08-07', FALSE)`, term2, yearID)
	testutil.MustExec(t, e.pool, `UPDATE students SET current_class_id = $1 WHERE id = $2`, eastID, brianID)
	t2 := post(a, map[string]interface{}{"name": "End T2", "assessment_type": "endterm", "class_id": eastID, "subject_id": mathID, "term_id": term2})
	want(t, a.put("/api/assessments/"+t2+"/marks/", marks(mark(brianID, 91))), 200)

	h := want(t, e.parent().get("/api/results/students/"+brianID.String()+"/history/"), 200).JSON(t)
	terms := h["terms"].([]interface{})
	if len(terms) != 2 {
		t.Fatalf("terms: %v", h)
	}
	t1 := terms[0].(map[string]interface{})
	if t1["term_name"] != "Term 1" || t1["class_name"] != "Grade 4 West" {
		t.Errorf("term 1 must keep the class it was sat in: %v", t1)
	}
	subj := map[string]map[string]interface{}{}
	for _, s := range t1["subjects"].([]interface{}) {
		m := s.(map[string]interface{})
		subj[m["subject_name"].(string)] = m
	}
	if subj["Mathematics"]["percentage"] != 74.0 || subj["Mathematics"]["grade"] != "ME" { // 0.3·60 + 0.7·80
		t.Errorf("maths: %v", subj["Mathematics"])
	}
	if subj["English"]["percentage"] != 45.0 || subj["English"]["grade"] != "AE" {
		t.Errorf("english: %v", subj["English"])
	}
	if t1["mean"] != 59.5 {
		t.Errorf("term 1 mean: %v", t1["mean"])
	}
	if t2r := terms[1].(map[string]interface{}); t2r["class_name"] != "Grade 4 East" || t2r["mean"] != 91.0 || t2r["mean_grade"] != "EE" {
		t.Errorf("term 2: %v", t2r)
	}

	want(t, e.parent().get("/api/results/students/"+faithID.String()+"/history/"), 403)
	want(t, e.parent().get("/api/marks/?student_id="+faithID.String()), 200) // filtered, not an error
	if n := want(t, e.parent().get("/api/marks/"), 200).JSON(t)["count"]; n != 3.0 {    // current term only
		t.Errorf("parent marks list: %v", n)
	}
}

func TestTranscriptPDFAndVerification(t *testing.T) {
	e := newEnv(t)
	aid := id(t, want(t, e.westT().post("/api/assessments/", map[string]interface{}{
		"name": "End", "assessment_type": "endterm", "class_id": westID, "subject_id": mathID}), 201))
	want(t, e.westT().put("/api/assessments/"+aid+"/marks/", marks(mark(brianID, 88))), 200)

	pdf := want(t, e.parent().get("/api/results/students/"+brianID.String()+"/transcript/"), 200)
	if pdf.Header.Get("Content-Type") != "application/pdf" || !bytes.HasPrefix(pdf.Body, []byte("%PDF")) {
		t.Fatalf("not a PDF: %s", pdf.Header.Get("Content-Type"))
	}
	if !bytes.Contains(pdf.Body, []byte("/Subtype /Image")) {
		t.Error("transcript has no QR image")
	}
	want(t, e.parent().get("/api/results/students/"+faithID.String()+"/transcript/"), 403)

	links := api.NewLinksService(e.pool, e.keys, portalURL)
	url, err := links.ResultsVerifyURL(brianID, 1, 88)
	if err != nil {
		t.Fatal(err)
	}
	v := want(t, e.anon().post("/api/verify/", map[string]string{"token": strings.SplitN(url, "#", 2)[1]}), 200).JSON(t)
	if v["valid"] != true || v["document"] != "Academic record" || v["student_initials"] != "B. K." ||
		v["number"] != "1 term(s), latest mean 88.0%" {
		t.Errorf("verify: %v", v)
	}
}
