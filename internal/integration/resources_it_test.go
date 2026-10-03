//go:build integration

package integration

import (
	"strings"
	"testing"
)

func TestResourceSchemas(t *testing.T) {
	e := newEnv(t)
	all := want(t, e.admin().get("/api/meta/resources/"), 200).JSON(t)
	if n := int(all["count"].(float64)); n < 39 {
		t.Errorf("admin sees %d resources, want ≥39", n)
	}
	// Parents only see registers scoped to their children.
	for _, r := range want(t, e.parent().get("/api/meta/resources/"), 200).JSON(t)["results"].([]interface{}) {
		name := r.(map[string]interface{})["name"].(string)
		if name == "visitors" || name == "expenses" || name == "staff-appraisals" {
			t.Errorf("parent can see %s", name)
		}
	}
	sch := want(t, e.admin().get("/api/meta/resources/discipline-cases/"), 200).JSON(t)
	f0 := sch["fields"].([]interface{})[0].(map[string]interface{})
	if f0["name"] != "student_id" || f0["type"] != "ref" || f0["required"] != true ||
		f0["ref"].(map[string]interface{})["resource"] != "students" {
		t.Errorf("schema missing typed student field: %v", f0)
	}
	if strings.Contains(string(mustJSON(sch)), "discipline_cases") {
		t.Error("schema leaks the table name")
	}
}

func TestResourceCRUDValidationAndSearch(t *testing.T) {
	e := newEnv(t)
	a := e.admin()

	bad := want(t, a.post("/api/discipline-cases/", map[string]interface{}{
		"student_id": "nope", "severity": "extreme", "incident_date": "yesterday", "hacker": 1,
	}), 400).JSON(t)["fields"].(map[string]interface{})
	for _, f := range []string{"student_id", "severity", "incident_date", "offence", "hacker"} {
		if bad[f] == nil {
			t.Errorf("no validation error for %s: %v", f, bad)
		}
	}

	c := want(t, e.westT().post("/api/discipline-cases/", map[string]interface{}{
		"student_id": brianID, "incident_date": "2026-02-02", "offence": "Missing homework",
	}), 201).JSON(t)
	if c["student_id_label"] != "Brian Kamau (ADM-002)" || c["status"] != "open" || c["severity"] != "minor" {
		t.Errorf("created row: %v", c)
	}
	cid := c["id"].(string)
	up := want(t, a.patch("/api/discipline-cases/"+cid+"/", map[string]interface{}{"status": "resolved", "parent_notified": true}), 200).JSON(t)
	if up["status"] != "resolved" || up["parent_notified"] != true {
		t.Errorf("patched row: %v", up)
	}
	want(t, a.patch("/api/discipline-cases/"+cid+"/", map[string]interface{}{"offence": ""}), 400)

	if n := want(t, a.get("/api/discipline-cases/?status=resolved"), 200).JSON(t)["count"]; n != 1.0 {
		t.Errorf("filter count %v", n)
	}
	if n := want(t, a.get("/api/discipline-cases/?search=homework"), 200).JSON(t)["count"]; n != 1.0 {
		t.Errorf("search count %v", n)
	}
	if n := want(t, a.get("/api/discipline-cases/?search=100%25"), 200).JSON(t)["count"]; n != 0.0 {
		t.Errorf("LIKE wildcards must be escaped, got %v", n)
	}
	want(t, a.del("/api/discipline-cases/"+cid+"/"), 204)
	want(t, a.get("/api/discipline-cases/"+cid+"/"), 404)
}

func TestResourcePermissionsAndParentScope(t *testing.T) {
	e := newEnv(t)
	faithCase := id(t, want(t, e.admin().post("/api/discipline-cases/", map[string]interface{}{
		"student_id": faithID, "incident_date": "2026-02-02", "offence": "Late"}), 201))
	want(t, e.admin().post("/api/discipline-cases/", map[string]interface{}{
		"student_id": brianID, "incident_date": "2026-02-03", "offence": "Noise"}), 201)

	p := e.parent()
	list := want(t, p.get("/api/discipline-cases/"), 200).JSON(t)
	if list["count"] != 1.0 || !strings.Contains(string(mustJSON(list)), "Brian Kamau") {
		t.Errorf("parent should see only Brian's case: %v", list)
	}
	want(t, p.get("/api/discipline-cases/"+faithCase+"/"), 404) // not even confirmed to exist
	want(t, p.post("/api/discipline-cases/", map[string]interface{}{"student_id": brianID, "incident_date": "2026-02-03", "offence": "x"}), 403)
	want(t, p.get("/api/visitors/"), 403)
	want(t, p.get("/api/meta/resources/discipline-cases/options/student_id/"), 403) // can't list every student
	want(t, e.westT().get("/api/expenses/"), 403)
	want(t, e.westT().post("/api/visitors/", map[string]interface{}{"visitor_name": "X", "purpose": "Y"}), 403)
	want(t, e.anon().get("/api/visitors/"), 401)
}

func TestResourceRefOptions(t *testing.T) {
	e := newEnv(t)
	opts := want(t, e.admin().get("/api/meta/resources/discipline-cases/options/student_id/?search=brian"), 200).JSON(t)
	res := opts["results"].([]interface{})
	if len(res) != 1 || res[0].(map[string]interface{})["label"] != "Brian Kamau (ADM-002)" {
		t.Errorf("options: %v", opts)
	}
	byID := want(t, e.admin().get("/api/meta/resources/discipline-cases/options/student_id/?id="+faithID.String()), 200).JSON(t)
	if len(byID["results"].([]interface{})) != 1 {
		t.Errorf("lookup by id: %v", byID)
	}
}

func TestResourceConstraintsAndTypes(t *testing.T) {
	e := newEnv(t)
	a := e.admin()

	// Table-level CHECK → the spec's plain-language message.
	r := want(t, a.post("/api/exam-threads/", map[string]interface{}{"name": "T1", "term_id": termID, "cats_weight": 30, "main_exam_weight": 60}), 400)
	if !strings.Contains(string(r.Body), "must add up to 100") {
		t.Errorf("weights message: %s", r.Body)
	}
	want(t, a.post("/api/exam-threads/", map[string]interface{}{"name": "T1", "term_id": termID, "cats_weight": 30, "main_exam_weight": 70}), 201)

	h := id(t, want(t, a.post("/api/hostels/", map[string]interface{}{"name": "Elgon", "gender": "girls"}), 201))
	want(t, a.post("/api/hostels/", map[string]interface{}{"name": "Elgon"}), 409)
	want(t, a.post("/api/hostel-rooms/", map[string]interface{}{"hostel_id": h, "room_no": "A1"}), 201)
	want(t, a.del("/api/hostels/"+h+"/"), 409) // still has rooms

	tt := want(t, a.post("/api/timetable/", map[string]interface{}{
		"term_id": termID, "class_id": eastID, "subject_id": mathID, "day_of_week": 1, "period_number": 1,
		"start_time": "08:00", "end_time": "08:40"}), 201).JSON(t)
	if tt["start_time"] != "08:00" || tt["class_id_label"] != "Grade 4 East" {
		t.Errorf("timetable row: %v", tt)
	}
	v := want(t, a.post("/api/visitors/", map[string]interface{}{
		"visitor_name": "Jane", "purpose": "Meeting", "time_in": "2026-02-02T09:15", "phone": "0712 345-678"}), 201).JSON(t)
	if v["phone"] != "0712345678" || !strings.HasPrefix(v["time_in"].(string), "2026-02-02T09:15:00+03:00") {
		t.Errorf("visitor row: %v", v)
	}
}
