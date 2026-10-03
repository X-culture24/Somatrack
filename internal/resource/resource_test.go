package resource

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/stmaryskabete/lms/internal/shared/types"
)

var sample = Resource{
	Name: "visitors", Title: "Visitor", Module: "Front office", Table: "visitors",
	Fields: []Field{
		{Name: "visitor_name", Label: "Name", Type: Text, Required: true, MaxLen: 10, Search: true},
		{Name: "phone", Label: "Phone", Type: Phone},
		{Name: "purpose", Label: "Purpose", Type: Enum, Options: []string{"meeting", "delivery"}, Required: true},
		{Name: "people", Label: "People", Type: Int, Min: Float(1)},
		{Name: "visit_date", Label: "Date", Type: Date},
		{Name: "time_in", Label: "In", Type: Time},
		{Name: "fee", Label: "Fee", Type: Decimal},
		{Name: "student_id", Label: "Student", Type: Ref, Ref: &RefSpec{Resource: "students", Table: "students", Label: "x.full_name"}},
		{Name: "badge", Label: "Badge", Type: Text, ReadOnly: true},
	},
}

func TestSpecCheck(t *testing.T) {
	if err := sample.Check(); err != nil {
		t.Fatalf("sample spec: %v", err)
	}
	bad := []Resource{
		{Name: "x", Title: "x", Table: "x; DROP TABLE users", Fields: nil},
		{Name: "x", Title: "x", Table: "x", Fields: []Field{{Name: "a b", Label: "A", Type: Text}}},
		{Name: "x", Title: "x", Table: "x", Fields: []Field{{Name: "a", Label: "A", Type: Enum}}},
		{Name: "x", Title: "x", Table: "x", Fields: []Field{{Name: "a", Label: "A", Type: Ref}}},
		{Name: "x", Title: "x", Table: "x", Fields: []Field{{Name: "a", Label: "A", Type: Int, Search: true}}},
		{Name: "x", Title: "x", Table: "x", Fields: []Field{{Name: "a", Label: "A", Type: Text}, {Name: "a", Label: "A", Type: Text}}},
		{Name: "x", Title: "x", Table: "x", StudentField: "student_id"},
		{Name: "Bad Name", Title: "x", Table: "x"},
	}
	for i, r := range bad {
		if err := r.Check(); err == nil {
			t.Errorf("bad spec %d passed Check", i)
		}
	}
}

func decode(t *testing.T, body string, partial bool) (map[string]interface{}, map[string]string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	vals, errs, err := decodeBody(req, &sample, partial)
	if err != nil {
		t.Fatalf("decodeBody(%s): %v", body, err)
	}
	return vals, errs
}

func TestDecodeValid(t *testing.T) {
	sid := uuid.New()
	vals, errs := decode(t, `{"visitor_name":" Jane ","phone":"0712 345-678","purpose":"meeting","people":2,
		"visit_date":"2026-09-25","time_in":"08:30","fee":"1,500.50","student_id":"`+sid.String()+`",
		"badge":"ignored","id":"x","student_id_label":"echoed"}`, false)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	want := map[string]interface{}{
		"visitor_name": "Jane", "phone": "0712345678", "purpose": "meeting", "people": int64(2),
		"visit_date": time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), "time_in": "08:30", "fee": 1500.5, "student_id": sid,
	}
	for k, v := range want {
		if vals[k] != v {
			t.Errorf("%s = %#v, want %#v", k, vals[k], v)
		}
	}
	if _, ok := vals["badge"]; ok {
		t.Error("read-only field was accepted")
	}
}

func TestDecodeErrors(t *testing.T) {
	_, errs := decode(t, `{"visitor_name":"Far too long a name","phone":"abc","purpose":"party","people":0,
		"visit_date":"25/09/2026","time_in":"25:00","fee":"lots","student_id":"nope","hacker":1}`, false)
	for _, f := range []string{"visitor_name", "phone", "purpose", "people", "visit_date", "time_in", "fee", "student_id", "hacker"} {
		if errs[f] == "" {
			t.Errorf("expected an error for %s, got none (all: %v)", f, errs)
		}
	}
}

func TestRequired(t *testing.T) {
	_, errs := decode(t, `{"phone":"0712345678"}`, false)
	if errs["visitor_name"] == "" || errs["purpose"] == "" {
		t.Errorf("missing required fields not reported: %v", errs)
	}
	// A partial update may omit required fields but not blank them.
	_, errs = decode(t, `{"phone":"0712345678"}`, true)
	if len(errs) != 0 {
		t.Errorf("partial update: %v", errs)
	}
	_, errs = decode(t, `{"visitor_name":""}`, true)
	if errs["visitor_name"] == "" {
		t.Error("blanking a required field must fail")
	}
}

func TestPermissions(t *testing.T) {
	staffOnly := sample // no StudentField
	if canRead(&staffOnly, types.RoleParent) {
		t.Error("parents must not read resources without a student scope")
	}
	scoped := sample
	scoped.StudentField = "student_id"
	if !canRead(&scoped, types.RoleParent) || canWrite(&scoped, types.RoleParent) {
		t.Error("parents read-only on student-scoped resources")
	}
	if canWrite(&sample, types.RoleClassTeacher) {
		t.Error("default writers are admin roles only")
	}
	if !canWrite(&sample, types.RoleSystemAdmin) || !canRead(&sample, types.RoleClassTeacher) {
		t.Error("admin writes, staff read by default")
	}
	finance := sample
	finance.ReadRoles = types.FinanceRoles
	finance.WriteRoles = types.FinanceRoles
	if canRead(&finance, types.RoleClassTeacher) {
		t.Error("ReadRoles must restrict staff")
	}
	if canRead(&sample, "") {
		t.Error("no role, no access")
	}
}

func TestSelectList(t *testing.T) {
	g := NewRegistry(nil, nil)
	cols, names := g.selectList(&sample)
	if !strings.Contains(cols, "(SELECT x.full_name FROM students x WHERE x.id = t.student_id)") {
		t.Errorf("ref label subselect missing: %s", cols)
	}
	if names[len(names)-2] != "student_id_label" {
		t.Errorf("names = %v", names)
	}
	if !strings.Contains(cols, "to_char(t.time_in, 'HH24:MI')") || !strings.Contains(cols, "t.fee::float8") {
		t.Errorf("type casts missing: %s", cols)
	}
}

func TestSchemaJSONHidesSQL(t *testing.T) {
	b, _ := json.Marshal(sample)
	s := string(b)
	for _, leak := range []string{"x.full_name", `"Table"`, "visitors\""} {
		if strings.Contains(s, leak) && leak != "visitors\"" {
			t.Errorf("schema JSON leaks %q: %s", leak, s)
		}
	}
	if strings.Contains(s, `"table"`) || strings.Contains(s, "full_name") {
		t.Errorf("schema JSON leaks SQL details: %s", s)
	}
}
