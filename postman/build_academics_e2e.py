#!/usr/bin/env python3
"""Generates St_Marys_Academics_E2E.postman_collection.json.

End-to-end check of the academic workflow, run as the real roles:
admin registers a teacher (full staff form) and students (full admission
form), assigns subjects, the teacher sets an exam and uploads marks, the mark
list / report card / class performance are pulled, a parent reads only their
own child's results, and the headteacher reviews lesson plans the teacher
writes and updates. Every request asserts status + body, and the negative
cases (teacher assigning themselves, parent reading another child, teacher
approving their own plan...) assert the request is refused.

Self-contained: every record it needs is created with a per-run suffix and
deleted in the Cleanup folder, so it can run repeatedly against a live DB.

    python3 postman/build_academics_e2e.py
    npx newman run postman/St_Marys_Academics_E2E.postman_collection.json
"""
import json
import uuid
from pathlib import Path

OUT = Path(__file__).with_name("St_Marys_Academics_E2E.postman_collection.json")


def js(*lines):
    return [l for block in lines for l in block.strip("\n").split("\n")]


def req(name, method, path, token="admin_token", body=None, tests="", pre="", query=None):
    url = "{{base_url}}" + path
    item = {
        "name": name,
        "request": {
            "method": method,
            "header": [{"key": "Content-Type", "value": "application/json"}],
            # a trailing "" keeps the trailing slash the gateway's routes require
            "url": {"raw": url, "host": ["{{base_url}}"], "path": path.split("?")[0].lstrip("/").split("/")},
        },
        "event": [],
    }
    if "?" in path:
        item["request"]["url"]["query"] = [
            {"key": kv.split("=", 1)[0], "value": kv.split("=", 1)[1] if "=" in kv else ""}
            for kv in path.split("?", 1)[1].split("&")
        ]
    if token:
        item["request"]["auth"] = {"type": "bearer", "bearer": [{"key": "token", "value": "{{" + token + "}}", "type": "string"}]}
    else:
        item["request"]["auth"] = {"type": "noauth"}
    if body is not None:
        item["request"]["body"] = {"mode": "raw", "raw": json.dumps(body, indent=2), "options": {"raw": {"language": "json"}}}
    if tests:
        item["event"].append({"listen": "test", "script": {"type": "text/javascript", "exec": js(tests)}})
    if pre:
        item["event"].append({"listen": "prerequest", "script": {"type": "text/javascript", "exec": js(pre)}})
    return item


def folder(name, desc, items):
    return {"name": name, "description": desc, "item": items}


def status(code):
    codes = code if isinstance(code, list) else [code]
    return f"""
pm.test("status is {' or '.join(map(str, codes))}", function () {{
    pm.expect(pm.response.code, pm.response.text()).to.be.oneOf({json.dumps(codes)});
}});
"""


def refused(code=403):
    return status(code)


def login(name, email_var, password_var, token_var):
    return req(
        name, "POST", "/auth/login/", token=None,
        body={"email": "{{" + email_var + "}}", "password": "{{" + password_var + "}}"},
        tests=status(200) + f"""
const b = pm.response.json();
pm.test("returns an access token", () => pm.expect(b.access).to.be.a("string").and.not.empty);
pm.collectionVariables.set("{token_var}", b.access);
""",
    )


# ---------------------------------------------------------------------------
# Payloads

TEACHER_STAFF = {
    "user_id": "{{teacher_user_id}}",
    "employee_no": "E2E-T-{{run}}",
    "national_id": "2{{run}}",
    "tsc_no": "TSC{{run}}",
    "gender": "female",
    "date_of_birth": "1988-04-12",
    "employment_type": "tsc",
    "designation": "Subject Teacher",
    "date_of_joining": "2026-01-05",
    "highest_qualification": "B.Ed (Science)",
    "basic_salary": 45000,
    "kra_pin": "A0{{run}}Z",
    "nhif_no": "NH{{run}}",
    "nssf_no": "NS{{run}}",
    "bank_name": "Equity Bank",
    "bank_branch": "Kabete",
    "bank_account_no": "0{{run}}01",
    "emergency_contact_name": "Peter Achieng",
    "emergency_contact_phone": "254711000111",
    "emergency_contact_relation": "spouse",
    "middle_name": 'Atieno',
    "marital_status": 'married',
    "religion": 'Christian',
    "has_disability": False,
    "alternate_phone": '254711222333',
    "postal_address": 'P.O. Box 22, Kikuyu',
    "citizenship": 'Kenya',
    "home_county": 'Siaya',
    "residence_county": 'Kiambu',
    "sub_county": 'Kabete',
    "blood_group": 'A+',
    "allergies": 'Penicillin',
    "doctor_name": 'Dr. Otieno',
    "doctor_phone": '254733444555',
    "preferred_hospital": 'PCEA Kikuyu Hospital',
    "teaching_subjects": 'Biology / Chemistry',
    "training_institution": 'Kenyatta University',
    "year_of_graduation": '2011',
    "previous_employer": 'Kabete Girls',
    "contract_end_date": '2027-12-31',
    "photo_consent": True,
    "residential_address": "Kabete, Nairobi",
    "assign_subjects": [{"class_id": "{{class_id}}", "subject_id": "{{subject_id}}"}],
}

STUDENT_1 = {
    "admission_no": "E2E-{{run}}-1",
    "admission_date": "2026-01-06",
    "first_name": "Amani",
    "middle_name": "Wairimu",
    "last_name": "E2E{{run}}",
    "gender": "female",
    "date_of_birth": "2016-03-14",
    "birth_certificate_no": "BC{{run}}1",
    "upi": "UPI{{run}}1",
    "kemis_number": "KEMIS{{run}}1",
    "assessment_number": "ASN{{run}}1",
    "phone": "",
    "email": "",
    "current_class_id": "{{class_id}}",
    "boarding_status": "day",
    "blood_group": "O+",
    "allergies": "Peanuts",
    "chronic_conditions": "Asthma",
    "residential_address": "Kabete, Kiambu",
    "former_school": "Kabete Primary",
    "status": "active",
    "uses_transport": False,
    "kcpe_marks": 312,
    "photo_consent": True,
    "parental_status": "father_deceased",
    "has_special_needs": True,
    "special_needs": "Mild hearing impairment - sits at the front",
    "religion": "Christian",
    "doctor_name": "Dr. Wanjiru Kamau",
    "doctor_phone": "254722000111",
    "preferred_hospital": "Kikuyu Mission Hospital",
    "has_scholarship": True,
    "scholarship_type": "partial",
    "sponsor_name": "Equity Wings to Fly",
    "sponsor_phone": "254763000000",
    "sponsor_location": "Nairobi",
    "sponsor_contact_person": "Jane Mwangi",
    "sponsor_contact_details": "scholarships@equity.test",
    "citizenship": "Kenya",
    "home_county": "Murang'a",
    "residence_county": "Kiambu",
    "sub_county": "Kabete",
    "guardians": [
        {
            "first_name": "Grace", "last_name": "Parent{{run}}", "phone": "2547{{run}}1",
            "alternate_phone": "2547{{run}}2", "email": "parent1.{{run}}@e2e.test", "national_id": "3{{run}}1",
            "occupation": "Nurse", "employer": "Kenyatta Hospital", "residential_address": "Kabete",
            "postal_address": "P.O. Box 1, Kabete", "relation": "mother", "is_primary": True, "has_custody": True,
            "contact_priority": 1,
        },
        {
            "first_name": "James", "last_name": "Parent{{run}}", "phone": "2547{{run}}3",
            "occupation": "Engineer", "relation": "father", "is_primary": False, "has_custody": True,
            "contact_priority": 2,
        },
    ],
}

STUDENT_2 = {
    "admission_no": "E2E-{{run}}-2",
    "first_name": "Baraka",
    "last_name": "E2E{{run}}",
    "gender": "male",
    "date_of_birth": "2016-07-02",
    "current_class_id": "{{class_id}}",
    "guardians": [
        {"first_name": "Other", "last_name": "Guardian{{run}}", "phone": "2547{{run}}9",
         "email": "parent2.{{run}}@e2e.test", "relation": "guardian", "is_primary": True}
    ],
}

# ---------------------------------------------------------------------------

setup = folder("0. Setup", "Admin login, current term, and two throw-away subjects for this run.", [
    req("Admin login", "POST", "/auth/login/", token=None,
        body={"email": "{{admin_email}}", "password": "{{admin_password}}"},
        pre="""
// one suffix per run keeps emails/admission numbers unique
pm.collectionVariables.set("run", String(Date.now()).slice(-7));
""",
        tests=status(200) + """
const b = pm.response.json();
pm.test("admin token issued", () => pm.expect(b.access).to.be.a("string"));
pm.collectionVariables.set("admin_token", b.access);
pm.collectionVariables.set("admin_user_id", b.user && b.user.id);
"""),
    req("Current term", "GET", "/terms/", tests=status(200) + """
const cur = pm.response.json().results.find(t => t.is_current);
pm.test("a current term is set", () => pm.expect(cur, "mark a term as current first").to.exist);
pm.collectionVariables.set("term_id", cur.id);
"""),
    req("Class exists", "GET", "/classes/{{class_id}}/", tests=status(200) + """
pm.test("class has a name", () => pm.expect(pm.response.json().name).to.be.a("string"));
"""),
    req("Create subject A (assigned to teacher A)", "POST", "/subjects/",
        body={"name": "E2E Science {{run}}", "code": "E2ES{{run}}", "learning_area": "Science", "is_active": True},
        tests=status(201) + 'pm.collectionVariables.set("subject_id", pm.response.json().id);\n'
              'pm.collectionVariables.set("subject_name", pm.response.json().name);'),
    req("Create subject B (assigned to teacher B)", "POST", "/subjects/",
        body={"name": "E2E Art {{run}}", "code": "E2EA{{run}}", "learning_area": "Creative Arts", "is_active": True},
        tests=status(201) + 'pm.collectionVariables.set("subject2_id", pm.response.json().id);'),
])

staff = folder("1. Add teacher (full staff form)", "Creates the login, then the staff profile with every form field and the subject to teach in one call; checks every field reads back and that pulling the staff member also pulls the subjects they teach.", [
    req("Create teacher A login", "POST", "/users/",
        body={"first_name": "Akinyi", "last_name": "Teacher{{run}}", "email": "teacher.a.{{run}}@e2e.test",
              "role": "SUBJECT_TEACHER", "phone": "254708374149", "password": "{{test_password}}"},
        tests=status(201) + """
const u = pm.response.json().user;
pm.test("no forced password change when admin sets one", () => pm.expect(u.must_change_password).to.be.false);
pm.collectionVariables.set("teacher_user_id", u.id);
pm.collectionVariables.set("teacher_email", u.email);
"""),
    req("Create teacher A staff profile + assign subject", "POST", "/staff/", body=TEACHER_STAFF,
        tests=status(201) + """
const sp = pm.response.json();
const sent = JSON.parse(pm.request.body.raw.replace(/{{run}}/g, pm.collectionVariables.get("run"))
    .replace(/{{teacher_user_id}}/g, pm.collectionVariables.get("teacher_user_id"))
    .replace(/{{class_id}}/g, pm.collectionVariables.get("class_id"))
    .replace(/{{subject_id}}/g, pm.collectionVariables.get("subject_id")));
pm.collectionVariables.set("staff_id", sp.id);
["employee_no","national_id","tsc_no","gender","date_of_birth","employment_type","designation",
 "date_of_joining","highest_qualification","basic_salary","kra_pin","nhif_no","nssf_no","bank_name",
 "bank_branch","bank_account_no","emergency_contact_name","emergency_contact_phone",
 "emergency_contact_relation","residential_address",
 "middle_name","marital_status","religion","has_disability","alternate_phone","postal_address","citizenship","home_county","residence_county","sub_county","blood_group","allergies","doctor_name","doctor_phone","preferred_hospital","teaching_subjects","training_institution","year_of_graduation","previous_employer","contract_end_date","photo_consent"].forEach(f => {
    pm.test("staff field saved: " + f, () => pm.expect(sp[f]).to.eql(sent[f]));
});
pm.test("assigned subject comes back with the staff record", () => {
    pm.expect(sp.assignments.map(a => a.subject_id)).to.include(pm.collectionVariables.get("subject_id"));
    pm.expect(sp.subjects).to.include(pm.collectionVariables.get("subject_name"));
});
"""),
    req("Duplicate employee no. is refused", "POST", "/staff/",
        body={"user_id": "{{admin_user_id}}", "employee_no": "E2E-T-{{run}}"}, tests=status(409)),
    req("Invalid gender is refused", "POST", "/staff/",
        body={"user_id": "{{admin_user_id}}", "employee_no": "E2E-X-{{run}}", "gender": "unknown"}, tests=status(400)),
    req("Invalid marital status is refused", "POST", "/staff/",
        body={"user_id": "{{admin_user_id}}", "employee_no": "E2E-X-{{run}}", "marital_status": "complicated"}, tests=status(400)),
    req("Invalid graduation year is refused", "POST", "/staff/",
        body={"user_id": "{{admin_user_id}}", "employee_no": "E2E-X-{{run}}", "year_of_graduation": "20x1"}, tests=status(400)),
    req("Full name includes the middle name", "GET", "/staff/{{staff_id}}/", tests=status(200) + """
pm.test("first middle last", () => pm.expect(pm.response.json().full_name).to.eql("Akinyi Atieno Teacher" + pm.collectionVariables.get("run")));
"""),
    req("Edit staff: clear contract date, change religion, keep the rest", "PATCH", "/staff/{{staff_id}}/",
        body={"contract_end_date": "", "religion": "Other"}, tests=status(200) + """
const sp = pm.response.json();
pm.test("changed", () => { pm.expect(sp.contract_end_date).to.be.undefined; pm.expect(sp.religion).to.eql("Other"); });
pm.test("kept", () => { pm.expect(sp.home_county).to.eql("Siaya"); pm.expect(sp.tsc_no).to.eql("TSC" + pm.collectionVariables.get("run")); });
"""),
    req("Edit staff: invalid marital status -> 400", "PATCH", "/staff/{{staff_id}}/",
        body={"marital_status": "complicated"}, tests=status(400)),
    req("Pull teacher A -> includes subjects taught", "GET", "/staff/{{staff_id}}/", tests=status(200) + """
const sp = pm.response.json();
pm.test("assignment names class and subject", () => {
    const a = sp.assignments.find(x => x.subject_id === pm.collectionVariables.get("subject_id"));
    pm.expect(a).to.exist;
    pm.expect(a.class_id).to.eql(pm.collectionVariables.get("class_id"));
    pm.expect(a.class_name).to.be.a("string").and.not.empty;
    pm.expect(a.subject_name).to.eql(pm.collectionVariables.get("subject_name"));
});
"""),
    req("Staff list -> every row carries subjects", "GET", "/staff/", tests=status(200) + """
const rows = pm.response.json().results;
pm.test("every staff row has assignments[] and subjects[]", () => rows.forEach(r => {
    pm.expect(r.assignments).to.be.an("array"); pm.expect(r.subjects).to.be.an("array");
}));
const me = rows.find(r => r.id === pm.collectionVariables.get("staff_id"));
pm.test("teacher A listed with their subject", () => pm.expect(me.subjects).to.include(pm.collectionVariables.get("subject_name")));
"""),
    req("Create teacher B login", "POST", "/users/",
        body={"first_name": "Brian", "last_name": "Teacher{{run}}", "email": "teacher.b.{{run}}@e2e.test",
              "role": "SUBJECT_TEACHER", "password": "{{test_password}}"},
        tests=status(201) + 'pm.collectionVariables.set("teacher2_user_id", pm.response.json().user.id);\n'
              'pm.collectionVariables.set("teacher2_email", pm.response.json().user.email);'),
    req("Create headteacher login", "POST", "/users/",
        body={"first_name": "Hellen", "last_name": "Head{{run}}", "email": "head.{{run}}@e2e.test",
              "role": "HEAD_TEACHER", "password": "{{test_password}}"},
        tests=status(201) + 'pm.collectionVariables.set("head_user_id", pm.response.json().user.id);\n'
              'pm.collectionVariables.set("head_email", pm.response.json().user.email);'),
])

assign = folder("2. Assign teachers to subjects", "Bulk assignment, clash detection, 'who teaches X', and that teachers can't assign themselves.", [
    req("Teacher detail shows subjects + classes", "GET", "/teachers/{{teacher_user_id}}/", tests=status(200) + """
const t = pm.response.json();
pm.test("subjects taught", () => pm.expect(t.subjects).to.include(pm.collectionVariables.get("subject_name")));
pm.test("staff profile joined", () => pm.expect(t.tsc_no).to.eql("TSC" + pm.collectionVariables.get("run")));
"""),
    req("Assigning a subject that already has a teacher -> 409 with the clash", "POST", "/teachers/{{teacher2_user_id}}/assignments/",
        body={"assignments": [{"class_id": "{{class_id}}", "subject_id": "{{subject_id}}"}]},
        tests=status(409) + """
const c = pm.response.json().conflicts;
pm.test("names the current teacher", () => pm.expect(c[0].current_teacher_id).to.eql(pm.collectionVariables.get("teacher_user_id")));
"""),
    req("Assign subject B to teacher B", "POST", "/teachers/{{teacher2_user_id}}/assignments/",
        body={"assignments": [{"class_id": "{{class_id}}", "subject_id": "{{subject2_id}}"}]},
        tests=status(200) + """
pm.test("teacher B now teaches subject B", () =>
    pm.expect(pm.response.json().assignments.map(a => a.subject_id)).to.include(pm.collectionVariables.get("subject2_id")));
"""),
    req("Bulk replace: teacher A takes subject B too (reassign)", "PUT", "/teachers/{{teacher_user_id}}/assignments/",
        body={"assignments": [{"class_id": "{{class_id}}", "subject_id": "{{subject_id}}"},
                              {"class_id": "{{class_id}}", "subject_id": "{{subject2_id}}"}], "reassign": True},
        tests=status(200) + 'pm.test("two subjects", () => pm.expect(pm.response.json().assignments).to.have.length(2));'),
    req("Bulk replace: teacher A back to subject A only", "PUT", "/teachers/{{teacher_user_id}}/assignments/",
        body={"assignments": [{"class_id": "{{class_id}}", "subject_id": "{{subject_id}}"}]},
        tests=status(200) + 'pm.test("one subject", () => pm.expect(pm.response.json().assignments).to.have.length(1));'),
    req("Give subject B back to teacher B", "POST", "/teachers/{{teacher2_user_id}}/assignments/",
        body={"assignments": [{"class_id": "{{class_id}}", "subject_id": "{{subject2_id}}"}]}, tests=status(200) + """
const a = pm.response.json().assignments.find(x => x.subject_id === pm.collectionVariables.get("subject2_id"));
pm.collectionVariables.set("t2_assignment_id", a.id);
"""),
    req("Remove subject B from teacher B (single assignment)", "DELETE", "/teachers/{{teacher2_user_id}}/assignments/{{t2_assignment_id}}/",
        tests=status([200, 204])),
    req("Teacher B no longer teaches subject B", "GET", "/teachers/{{teacher2_user_id}}/", tests=status(200) + """
pm.test("subject B gone", () => pm.expect(pm.response.json().assignments.map(a => a.subject_id)).to.not.include(pm.collectionVariables.get("subject2_id")));
"""),
    req("Re-assign subject B to teacher B", "POST", "/teachers/{{teacher2_user_id}}/assignments/",
        body={"assignments": [{"class_id": "{{class_id}}", "subject_id": "{{subject2_id}}"}]}, tests=status(200)),
    req("Class curriculum shows who teaches each subject", "GET", "/exams/class-subjects/?class_id={{class_id}}", tests=status(200) + """
const rows = pm.response.json().results || pm.response.json();
const a = rows.find(r => r.subject_id === pm.collectionVariables.get("subject_id"));
const b = rows.find(r => r.subject_id === pm.collectionVariables.get("subject2_id"));
pm.test("subject A -> teacher A", () => { pm.expect(a.teacher_id).to.eql(pm.collectionVariables.get("teacher_user_id")); pm.expect(a.teacher_name).to.include("Akinyi"); });
pm.test("subject B -> teacher B", () => pm.expect(b.teacher_id).to.eql(pm.collectionVariables.get("teacher2_user_id")));
"""),
    req("Staff form assigning a taken subject -> 409, nothing saved", "POST", "/staff/",
        body={"user_id": "{{teacher2_user_id}}", "employee_no": "E2E-B-{{run}}",
              "assign_subjects": [{"class_id": "{{class_id}}", "subject_id": "{{subject_id}}"}]},
        tests=status(409) + """
pm.test("names the clash", () => pm.expect(pm.response.json().conflicts[0].current_teacher_id).to.eql(pm.collectionVariables.get("teacher_user_id")));
"""),
    req("Teachers list (assignment screen) shows each teacher's subjects", "GET", "/teachers/", tests=status(200) + """
const rows = pm.response.json().results;
const a = rows.find(t => t.id === pm.collectionVariables.get("teacher_user_id"));
pm.test("teacher A listed with subject A", () => pm.expect(a.subjects).to.include(pm.collectionVariables.get("subject_name")));
pm.test("every row has assignments[]", () => rows.forEach(t => pm.expect(t.assignments).to.be.an("array")));
"""),
    req("Who teaches subject A?", "GET", "/teachers/?subject_id={{subject_id}}", tests=status(200) + """
const ids = pm.response.json().results.map(t => t.id);
pm.test("only teacher A", () => pm.expect(ids).to.eql([pm.collectionVariables.get("teacher_user_id")]));
"""),
    req("Unknown class/subject -> 400", "POST", "/teachers/{{teacher_user_id}}/assignments/",
        body={"assignments": [{"class_id": str(uuid.UUID(int=1)), "subject_id": "{{subject_id}}"}]}, tests=status(400)),
    req("Non-teaching user can't be assigned -> 400", "POST", "/teachers/{{admin_user_id}}/assignments/",
        body={"assignments": [{"class_id": "{{class_id}}", "subject_id": "{{subject_id}}"}]}, tests=status(400)),
    login("Teacher A login", "teacher_email", "test_password", "teacher_token"),
    login("Teacher B login", "teacher2_email", "test_password", "teacher2_token"),
    login("Headteacher login", "head_email", "test_password", "head_token"),
    req("Teacher A: my classes shows the subject", "GET", "/teacher/my-classes/", token="teacher_token", tests=status(200) + """
const c = pm.response.json().results.find(x => x.id === pm.collectionVariables.get("class_id"));
pm.test("class listed with subject", () => pm.expect(c.subjects_taught).to.include(pm.collectionVariables.get("subject_name")));
"""),
    req("Teacher A can view own teacher record", "GET", "/teachers/{{teacher_user_id}}/", token="teacher_token", tests=status(200)),
    req("Teacher A can't view teacher B's record", "GET", "/teachers/{{teacher2_user_id}}/", token="teacher_token", tests=refused()),
    req("Teacher A can't assign themselves (teachers API)", "POST", "/teachers/{{teacher_user_id}}/assignments/", token="teacher_token",
        body={"assignments": [{"class_id": "{{class_id}}", "subject_id": "{{subject2_id}}"}], "reassign": True}, tests=refused()),
    req("Teacher A can't assign themselves (exams API)", "POST", "/exams/class-subjects/", token="teacher_token",
        body={"class_id": "{{class_id}}", "subject_id": "{{subject2_id}}", "teacher_id": "{{teacher_user_id}}"}, tests=refused()),
    req("Teacher A can't make themselves class teacher", "PATCH", "/exams/classes/{{class_id}}/class-teacher/", token="teacher_token",
        body={"teacher_id": "{{teacher_user_id}}"}, tests=refused()),
])

students = folder("3. Add students (full admission form)", "Every admission-form field plus two guardians inline; checks each field reads back. Second student belongs to a different family.", [
    req("Register student 1 (all fields, two guardians)", "POST", "/students/", body=STUDENT_1, tests=status(201) + """
pm.collectionVariables.set("student_id", pm.response.json().id);
"""),
    req("Student 1 reads back every field", "GET", "/students/{{student_id}}/", tests=status(200) + """
const s = pm.response.json();
const run = pm.collectionVariables.get("run");
const expect = {
  admission_no: "E2E-" + run + "-1", admission_date: "2026-01-06", first_name: "Amani", middle_name: "Wairimu",
  gender: "female", date_of_birth: "2016-03-14", birth_certificate_no: "BC" + run + "1", upi: "UPI" + run + "1",
  kemis_number: "KEMIS" + run + "1", assessment_number: "ASN" + run + "1", boarding_status: "day",
  blood_group: "O+", allergies: "Peanuts", chronic_conditions: "Asthma", residential_address: "Kabete, Kiambu",
  former_school: "Kabete Primary", current_class_id: pm.collectionVariables.get("class_id"), kcpe_marks: 312,
  photo_consent: true, parental_status: "father_deceased", has_special_needs: true,
  special_needs: "Mild hearing impairment - sits at the front", religion: "Christian",
  doctor_name: "Dr. Wanjiru Kamau", doctor_phone: "254722000111", preferred_hospital: "Kikuyu Mission Hospital",
  has_scholarship: true, scholarship_type: "partial", sponsor_name: "Equity Wings to Fly",
  sponsor_phone: "254763000000", sponsor_location: "Nairobi", sponsor_contact_person: "Jane Mwangi",
  sponsor_contact_details: "scholarships@equity.test", citizenship: "Kenya", home_county: "Murang'a",
  residence_county: "Kiambu", sub_county: "Kabete",
};
Object.keys(expect).forEach(k => pm.test("student field saved: " + k, () => pm.expect(s[k]).to.eql(expect[k])));
pm.test("both guardians linked", () => pm.expect(s.guardians).to.have.length(2));
const mum = s.guardians.find(g => g.relation === "mother");
pm.test("guardian details saved", () => {
    pm.expect(mum.occupation).to.eql("Nurse");
    pm.expect(mum.employer).to.eql("Kenyatta Hospital");
    pm.expect(mum.alternate_phone).to.eql("2547" + run + "2");
    pm.expect(mum.national_id).to.eql("3" + run + "1");
    pm.expect(mum.postal_address).to.eql("P.O. Box 1, Kabete");
    pm.expect(mum.is_primary).to.be.true;
});
pm.collectionVariables.set("guardian_id", mum.guardian_id);
pm.collectionVariables.set("guardian2_id", s.guardians.find(g => g.relation === "father").guardian_id);
"""),
    req("Missing required fields -> 400", "POST", "/students/", body={"first_name": "NoAdmNo"}, tests=status(400)),
    req("Invalid student status -> 400", "POST", "/students/",
        body={"admission_no": "E2E-{{run}}-X", "first_name": "Bad", "last_name": "Status", "gender": "male",
              "parental_status": "unknown"}, tests=status(400)),
    req("Edit one field keeps the rest", "PATCH", "/students/{{student_id}}/",
        body={"has_scholarship": False, "religion": "Muslim"}, tests=status(200) + """
const s = pm.response.json();
pm.test("changed", () => { pm.expect(s.has_scholarship).to.be.false; pm.expect(s.religion).to.eql("Muslim"); });
pm.test("untouched fields kept", () => { pm.expect(s.sponsor_name).to.eql("Equity Wings to Fly"); pm.expect(s.home_county).to.eql("Murang'a"); });
"""),
    req("Invalid scholarship type on edit -> 400", "PATCH", "/students/{{student_id}}/",
        body={"scholarship_type": "lottery"}, tests=status(400)),
    req("Register student 2 (other family)", "POST", "/students/", body=STUDENT_2, tests=status(201) + """
pm.collectionVariables.set("student2_id", pm.response.json().id);
"""),
    req("Student 2 guardian id", "GET", "/students/{{student2_id}}/", tests=status(200) + """
pm.collectionVariables.set("guardian3_id", pm.response.json().guardians[0].guardian_id);
"""),
    req("Student 2 defaults: citizenship Kenya, no scholarship", "GET", "/students/{{student2_id}}/", tests=status(200) + """
const s = pm.response.json();
pm.test("defaults", () => { pm.expect(s.citizenship).to.eql("Kenya"); pm.expect(s.has_scholarship).to.be.false; pm.expect(s.photo_consent).to.be.false; });
"""),
    req("Create parent login for student 1's mother", "POST", "/guardians/{{guardian_id}}/create-login/", tests=status(201) + """
const b = pm.response.json();
pm.collectionVariables.set("parent_user_id", b.user_id);
pm.collectionVariables.set("parent_password", b.temporary_password);
pm.collectionVariables.set("parent_email", "parent1." + pm.collectionVariables.get("run") + "@e2e.test");
"""),
    login("Parent login", "parent_email", "parent_password", "parent_token"),
])

exams = folder("4. Exam management: set exam, upload results", "Teacher A sets a CAT and an end-term paper under an exam thread and uploads marks; teacher B can't touch them.", [
    req("Admin creates exam thread (CATs 30% / main 70%)", "POST", "/exams/threads/",
        body={"name": "E2E End Term {{run}}", "term_id": "{{term_id}}", "curriculum": "cbc", "cats_weight": 30, "main_exam_weight": 70},
        tests=status(201) + 'pm.collectionVariables.set("thread_id", pm.response.json().id);'),
    req("Teacher A creates CAT 1 (out of 50)", "POST", "/assessments/", token="teacher_token",
        body={"name": "CAT 1", "assessment_type": "cat1", "class_id": "{{class_id}}", "subject_id": "{{subject_id}}",
              "exam_thread_id": "{{thread_id}}", "total_marks": 50, "assessment_date": "2026-02-10"},
        tests=status(201) + 'pm.collectionVariables.set("cat_id", pm.response.json().id);'),
    req("Teacher A creates End-term paper (out of 100)", "POST", "/assessments/", token="teacher_token",
        body={"name": "End Term Paper", "assessment_type": "endterm", "class_id": "{{class_id}}", "subject_id": "{{subject_id}}",
              "exam_thread_id": "{{thread_id}}", "total_marks": 100, "assessment_date": "2026-03-25"},
        tests=status(201) + """
const a = pm.response.json();
pm.collectionVariables.set("assessment_id", a.id);
pm.test("linked to thread and term", () => {
    pm.expect(a.exam_thread_id).to.eql(pm.collectionVariables.get("thread_id"));
    pm.expect(a.term_id).to.eql(pm.collectionVariables.get("term_id"));
});
"""),
    req("Teacher B can't set an exam in teacher A's subject", "POST", "/assessments/", token="teacher2_token",
        body={"name": "Sneaky", "assessment_type": "other", "class_id": "{{class_id}}", "subject_id": "{{subject_id}}"},
        tests=refused()),
    req("Teacher A marksheet lists both students", "GET", "/assessments/{{assessment_id}}/marksheet/", token="teacher_token",
        tests=status(200) + """
const ids = pm.response.json().results.map(r => r.student_id);
pm.test("both students on the sheet", () => pm.expect(ids).to.include.members([pm.collectionVariables.get("student_id"), pm.collectionVariables.get("student2_id")]));
"""),
    req("Upload CAT marks", "PUT", "/assessments/{{cat_id}}/marks/", token="teacher_token",
        body={"marks": [{"student_id": "{{student_id}}", "score": 40, "comments": "Good"},
                        {"student_id": "{{student2_id}}", "score": 20}]},
        tests=status(200) + 'pm.test("2 saved", () => pm.expect(pm.response.json().saved).to.eql(2));'),
    req("Upload End-term marks", "PUT", "/assessments/{{assessment_id}}/marks/", token="teacher_token",
        body={"marks": [{"student_id": "{{student_id}}", "score": 85, "comments": "Excellent work"},
                        {"student_id": "{{student2_id}}", "score": 45, "comments": "Keep trying"}]},
        tests=status(200) + 'pm.test("2 saved", () => pm.expect(pm.response.json().saved).to.eql(2));'),
    req("Score above total is rejected (nothing saved)", "PUT", "/assessments/{{assessment_id}}/marks/", token="teacher_token",
        body={"marks": [{"student_id": "{{student_id}}", "score": 150}]},
        tests=status(400) + 'pm.test("field error names the student", () => pm.expect(pm.response.json().fields).to.have.property(pm.collectionVariables.get("student_id")));'),
    req("Teacher B can't upload marks for teacher A's exam", "PUT", "/assessments/{{assessment_id}}/marks/", token="teacher2_token",
        body={"marks": [{"student_id": "{{student_id}}", "score": 1}]}, tests=refused()),
    req("Parent can't upload marks", "PUT", "/assessments/{{assessment_id}}/marks/", token="parent_token",
        body={"marks": [{"student_id": "{{student_id}}", "score": 100}]}, tests=refused()),
    req("Parent can't write CBC levels", "PUT", "/cbc/marksheet/", token="parent_token",
        body={"entries": [{"assessment_id": "{{cat_id}}", "student_id": "{{student_id}}", "level": "EE"}]}, tests=refused()),
    req("Teacher B can't write CBC levels in teacher A's subject", "PUT", "/cbc/marksheet/", token="teacher2_token",
        body={"entries": [{"assessment_id": "{{cat_id}}", "student_id": "{{student_id}}", "level": "BE"}]}, tests=refused()),
    req("Marksheet re-read shows saved score + grade", "GET", "/assessments/{{assessment_id}}/marksheet/", token="teacher_token",
        tests=status(200) + """
const r = pm.response.json().results.find(x => x.student_id === pm.collectionVariables.get("student_id"));
pm.test("score 85 kept (rejected upload changed nothing)", () => pm.expect(r.score).to.eql(85));
pm.test("percentage + grade computed", () => { pm.expect(r.percentage).to.eql(85); pm.expect(r.grade_letter).to.not.be.empty; });
"""),
])

marklist = folder("5. Mark list: find the results the teacher uploaded", "The mark list filtered by class, subject, term and uploader.", [
    req("Teacher A: mark list for class + subject", "GET", "/marks/?class_id={{class_id}}&subject_id={{subject_id}}&term_id={{term_id}}",
        token="teacher_token", tests=status(200) + """
const rows = pm.response.json().results;
pm.test("4 marks (2 students x 2 assessments)", () => pm.expect(rows).to.have.length(4));
const r = rows.find(x => x.student_id === pm.collectionVariables.get("student_id") && x.assessment_id === pm.collectionVariables.get("assessment_id"));
pm.test("row carries who uploaded it and when", () => {
    pm.expect(r.marked_by_id).to.eql(pm.collectionVariables.get("teacher_user_id"));
    pm.expect(r.marked_by_name).to.include("Akinyi");
    pm.expect(r.marked_at).to.be.a("string");
});
pm.test("row carries class, subject, term, %", () => {
    pm.expect(r.class_id).to.eql(pm.collectionVariables.get("class_id"));
    pm.expect(r.subject_name).to.eql(pm.collectionVariables.get("subject_name"));
    pm.expect(r.term_id).to.eql(pm.collectionVariables.get("term_id"));
    pm.expect(r.percentage).to.eql(85);
    pm.expect(r.comments).to.eql("Excellent work");
});
"""),
    req("Teacher A: everything I uploaded", "GET", "/marks/?marked_by=me", token="teacher_token", tests=status(200) + """
const rows = pm.response.json().results;
pm.test("all rows were uploaded by me", () => rows.forEach(r => pm.expect(r.marked_by_id).to.eql(pm.collectionVariables.get("teacher_user_id"))));
pm.test("includes my 4 marks", () => pm.expect(rows.length).to.be.at.least(4));
"""),
    req("Headteacher: marks uploaded by teacher A", "GET", "/marks/?marked_by={{teacher_user_id}}&subject_id={{subject_id}}",
        token="head_token", tests=status(200) + 'pm.test("4 marks", () => pm.expect(pm.response.json().results).to.have.length(4));'),
    req("Teacher B sees none of teacher A's marks", "GET", "/marks/?subject_id={{subject_id}}", token="teacher2_token",
        tests=status(200) + 'pm.test("empty", () => pm.expect(pm.response.json().results).to.have.length(0));'),
    req("Assessments list shows marks count", "GET", "/assessments/?class_id={{class_id}}&subject_id={{subject_id}}", token="teacher_token",
        tests=status(200) + """
const a = pm.response.json().results.find(x => x.id === pm.collectionVariables.get("assessment_id"));
pm.test("2 marks entered", () => pm.expect(a.marks_count).to.eql(2));
"""),
])

report = folder("6. Report card & overall performance", "Results trail, class ranking, report card and transcript PDFs. Expected subject score = 30% CAT + 70% end-term: student 1 = 0.3x80 + 0.7x85 = 83.5, student 2 = 0.3x40 + 0.7x45 = 43.5.", [
    req("Student 1 results for this term", "GET", "/results/students/{{student_id}}/history/?term_id={{term_id}}", token="head_token",
        tests=status(200) + """
const t = pm.response.json().terms;
pm.test("one term returned", () => pm.expect(t).to.have.length(1));
const sub = t[0].subjects.find(s => s.subject_id === pm.collectionVariables.get("subject_id"));
pm.test("subject score blends CAT 30% + exam 70% = 83.5", () => pm.expect(sub.percentage).to.eql(83.5));
pm.test("grade + remarks", () => { pm.expect(sub.grade).to.not.be.empty; pm.expect(sub.assessments).to.eql(2); });
pm.test("term mean", () => pm.expect(t[0].mean).to.eql(83.5));
"""),
    req("Class overall performance (ranked)", "GET", "/results/class-performance/?class_id={{class_id}}&term_id={{term_id}}",
        token="head_token", tests=status(200) + """
const b = pm.response.json();
const s1 = b.students.find(s => s.student_id === pm.collectionVariables.get("student_id"));
const s2 = b.students.find(s => s.student_id === pm.collectionVariables.get("student2_id"));
pm.test("both students ranked", () => { pm.expect(s1).to.exist; pm.expect(s2).to.exist; });
pm.test("means", () => { pm.expect(s1.mean).to.eql(83.5); pm.expect(s2.mean).to.eql(43.5); });
pm.test("student 1 ranks above student 2", () => pm.expect(s1.position).to.be.below(s2.position));
const subj = b.subjects.find(s => s.subject_id === pm.collectionVariables.get("subject_id"));
pm.test("subject class mean = 63.5", () => pm.expect(subj.mean).to.eql(63.5));
pm.test("class mean + grade present", () => { pm.expect(b.class_mean).to.be.a("number"); pm.expect(b.class_mean_grade).to.not.be.empty; });
"""),
    req("Teacher A can also pull class performance", "GET", "/results/class-performance/?class_id={{class_id}}&term_id={{term_id}}",
        token="teacher_token", tests=status(200)),
    req("Subject performance report (report-svc)", "GET", "/reports/class-performance/?class_id={{class_id}}&term_id={{term_id}}",
        token="head_token", tests=status(200) + """
const rows = pm.response.json();
const r = (Array.isArray(rows) ? rows : rows.results || []).find(x => x.subject_id === pm.collectionVariables.get("subject_id"));
pm.test("our subject appears", () => pm.expect(r).to.exist);
"""),
    req("Headteacher signs off student 1's report", "POST", "/cbc/signoffs/", token="head_token",
        body={"student_id": "{{student_id}}", "term_id": "{{term_id}}", "class_id": "{{class_id}}",
              "teacher_comment": "Hardworking learner", "general_performance_level": "EE",
              "headteacher_comment": "Well done", "headteacher_signed": True,
              "opening_date": "2026-05-04", "closing_date": "2026-04-03"},
        tests=status(200)),
    req("Report card PDF (headteacher)", "GET", "/documents/cbc-report/?student_id={{student_id}}&term_id={{term_id}}",
        token="head_token", tests=status(200) + """
pm.test("is a PDF", () => pm.expect(pm.response.headers.get("Content-Type")).to.include("pdf"));
pm.test("not empty", () => pm.expect(pm.response.responseSize).to.be.above(1000));
"""),
    req("Transcript PDF (all terms, signed QR)", "GET", "/results/students/{{student_id}}/transcript/", token="head_token",
        tests=status(200) + 'pm.test("is a PDF", () => pm.expect(pm.response.headers.get("Content-Type")).to.include("pdf"));'),
])

parent = folder("7. Parent access to results", "The parent sees their own child's results, mark list and report card, and nothing about other children or the class.", [
    req("Parent: child's results for the term", "GET", "/results/students/{{student_id}}/history/?term_id={{term_id}}", token="parent_token",
        tests=status(200) + """
const sub = pm.response.json().terms[0].subjects.find(s => s.subject_id === pm.collectionVariables.get("subject_id"));
pm.test("same score the school sees", () => pm.expect(sub.percentage).to.eql(83.5));
"""),
    req("Parent: full results trail", "GET", "/results/students/{{student_id}}/history/", token="parent_token",
        tests=status(200) + 'pm.test("at least one term", () => pm.expect(pm.response.json().terms.length).to.be.at.least(1));'),
    req("Parent: mark list only has their child", "GET", "/marks/", token="parent_token", tests=status(200) + """
const rows = pm.response.json().results;
pm.test("has the child's 2 marks", () => pm.expect(rows.filter(r => r.student_id === pm.collectionVariables.get("student_id"))).to.have.length(2));
pm.test("no other child's marks", () => rows.forEach(r => pm.expect(r.student_id).to.eql(pm.collectionVariables.get("student_id"))));
"""),
    req("Parent: child's report card PDF", "GET", "/documents/cbc-report/?student_id={{student_id}}&term_id={{term_id}}",
        token="parent_token", tests=status(200) + 'pm.test("is a PDF", () => pm.expect(pm.response.headers.get("Content-Type")).to.include("pdf"));'),
    req("Parent: child's transcript PDF", "GET", "/results/students/{{student_id}}/transcript/", token="parent_token", tests=status(200)),
    req("Parent can't read another child's results", "GET", "/results/students/{{student2_id}}/history/", token="parent_token", tests=refused()),
    req("Parent can't download another child's report card", "GET", "/documents/cbc-report/?student_id={{student2_id}}&term_id={{term_id}}",
        token="parent_token", tests=refused()),
    req("Parent can't filter the mark list to another child", "GET", "/marks/?student_id={{student2_id}}", token="parent_token",
        tests=status(200) + 'pm.test("empty", () => pm.expect(pm.response.json().results).to.have.length(0));'),
    req("Parent can't see class rankings", "GET", "/results/class-performance/?class_id={{class_id}}&term_id={{term_id}}",
        token="parent_token", tests=refused()),
    req("Parent can't see class performance report", "GET", "/reports/class-performance/?class_id={{class_id}}&term_id={{term_id}}",
        token="parent_token", tests=refused()),
    req("Parent can't see school-wide grades report", "GET", "/reports/grades/?term_id={{term_id}}", token="parent_token", tests=refused()),
    req("Parent can't read CBC class summary", "GET", "/cbc/class-summary/?class_id={{class_id}}&term_id={{term_id}}",
        token="parent_token", tests=refused()),
])

lock = folder("8. Locking results", "Once the exam thread is locked, marks can't change.", [
    req("Headteacher locks the exam thread", "POST", "/exams/threads/{{thread_id}}/lock/", token="head_token",
        body={"locked": True}, tests=status(200)),
    req("Teacher can't change marks after lock", "PUT", "/assessments/{{assessment_id}}/marks/", token="teacher_token",
        body={"marks": [{"student_id": "{{student_id}}", "score": 99}]}, tests=status(409)),
    req("Results still readable after lock", "GET", "/results/students/{{student_id}}/history/?term_id={{term_id}}", token="parent_token",
        tests=status(200) + 'pm.test("unchanged", () => pm.expect(pm.response.json().terms[0].mean).to.eql(83.5));'),
    req("Unlock (for cleanup)", "POST", "/exams/threads/{{thread_id}}/lock/", token="head_token", body={"locked": False}, tests=status(200)),
])

plans = folder("9. Lesson plans: teacher writes per subject, headteacher sees updates", "Teacher A plans their subject; the headteacher sees it, returns it with a comment, sees the teacher's update come back to the top, and approves it.", [
    req("Teacher A submits a lesson plan for their subject", "POST", "/lesson-plans/", token="teacher_token",
        body={"term_id": "{{term_id}}", "subject_id": "{{subject_id}}", "class_id": "{{class_id}}", "week_number": 3,
              "lesson_date": "2026-01-21", "topic": "Living things", "sub_topic": "Plants",
              "learning_objectives": "Name parts of a plant", "learning_outcomes": "Learner can label a plant",
              "teaching_methods": "Observation, discussion", "materials": "Potted plant, chart",
              "introduction": "Show a plant", "lesson_development": "Label parts in groups", "conclusion": "Recap",
              "assessment": "Oral questions", "homework": "Draw a plant", "status": "submitted"},
        tests=status(201) + """
const p = pm.response.json();
pm.collectionVariables.set("lesson_plan_id", p.id);
pm.test("submitted and timestamped", () => { pm.expect(p.status).to.eql("submitted"); pm.expect(p.submitted_at).to.be.a("string"); });
pm.test("teacher is the caller", () => pm.expect(p.teacher_id).to.eql(pm.collectionVariables.get("teacher_user_id")));
"""),
    req("Teacher A can't plan a subject they don't teach", "POST", "/lesson-plans/", token="teacher_token",
        body={"term_id": "{{term_id}}", "subject_id": "{{subject2_id}}", "class_id": "{{class_id}}", "week_number": 3,
              "lesson_date": "2026-01-21", "topic": "Not mine"}, tests=refused()),
    req("Teacher A can't approve their own plan", "PATCH", "/lesson-plans/{{lesson_plan_id}}/", token="teacher_token",
        body={"status": "approved"}, tests=status(400)),
    req("Teacher A can't use the review action", "POST", "/lesson-plans/{{lesson_plan_id}}/review/", token="teacher_token",
        body={"status": "approved"}, tests=refused()),
    req("Teacher B can't open teacher A's plan", "GET", "/lesson-plans/{{lesson_plan_id}}/", token="teacher2_token", tests=refused()),
    req("Headteacher sees teacher A's plans", "GET", "/lesson-plans/?teacher_id={{teacher_user_id}}", token="head_token",
        tests=status(200) + """
const p = pm.response.json().results.find(x => x.id === pm.collectionVariables.get("lesson_plan_id"));
pm.test("plan listed as submitted, with subject + teacher names", () => {
    pm.expect(p.status).to.eql("submitted");
    pm.expect(p.subject_name).to.eql(pm.collectionVariables.get("subject_name"));
    pm.expect(p.teacher_name).to.include("Akinyi");
});
"""),
    req("Headteacher coverage: which subjects have plans", "GET", "/lesson-plans/coverage/?class_id={{class_id}}&term_id={{term_id}}",
        token="head_token", tests=status(200) + """
const rows = pm.response.json().results;
const a = rows.find(r => r.subject_id === pm.collectionVariables.get("subject_id"));
const b = rows.find(r => r.subject_id === pm.collectionVariables.get("subject2_id"));
pm.test("subject A: 1 plan awaiting review by teacher A", () => {
    pm.expect(a.plans).to.eql(1); pm.expect(a.awaiting_review).to.be.true;
    pm.expect(a.teacher_id).to.eql(pm.collectionVariables.get("teacher_user_id"));
});
pm.test("subject B: no plans yet from teacher B", () => {
    pm.expect(b.plans).to.eql(0); pm.expect(b.teacher_id).to.eql(pm.collectionVariables.get("teacher2_user_id"));
});
pm.test("summary counts", () => pm.expect(pm.response.json().summary.without_plans).to.be.at.least(1));
"""),
    req("Returning without a comment is refused", "POST", "/lesson-plans/{{lesson_plan_id}}/review/", token="head_token",
        body={"status": "returned"}, tests=status(400)),
    req("Headteacher returns the plan with a comment", "POST", "/lesson-plans/{{lesson_plan_id}}/review/", token="head_token",
        body={"status": "returned", "comment": "Add a practical activity"}, tests=status(200) + """
const p = pm.response.json();
pm.test("returned with comment + reviewer", () => {
    pm.expect(p.status).to.eql("returned"); pm.expect(p.review_comment).to.eql("Add a practical activity");
    pm.expect(p.reviewed_by_id).to.eql(pm.collectionVariables.get("head_user_id"));
});
"""),
    req("Teacher A sees the headteacher's comment", "GET", "/lesson-plans/{{lesson_plan_id}}/", token="teacher_token",
        tests=status(200) + 'pm.test("comment visible", () => pm.expect(pm.response.json().review_comment).to.eql("Add a practical activity"));'),
    req("Teacher A updates the plan", "PATCH", "/lesson-plans/{{lesson_plan_id}}/", token="teacher_token",
        pre='pm.collectionVariables.set("before_update", new Date(Date.now() - 1000).toISOString());',
        body={"topic": "Living things (revised)", "lesson_development": "Label parts in groups, then plant seeds in cups"},
        tests=status(200) + """
const p = pm.response.json();
pm.test("edit saved", () => pm.expect(p.topic).to.eql("Living things (revised)"));
pm.test("goes back to the headteacher automatically", () => pm.expect(p.status).to.eql("submitted"));
"""),
    req("Headteacher: plans updated since then", "GET", "/lesson-plans/?updated_since={{before_update}}", token="head_token",
        tests=status(200) + """
const rows = pm.response.json().results;
pm.test("the updated plan is listed first", () => pm.expect(rows[0].id).to.eql(pm.collectionVariables.get("lesson_plan_id")));
pm.test("shows the new content", () => pm.expect(rows[0].topic).to.eql("Living things (revised)"));
"""),
    req("Headteacher: plans awaiting review", "GET", "/lesson-plans/?status=submitted&class_id={{class_id}}", token="head_token",
        tests=status(200) + 'pm.test("includes the plan", () => pm.expect(pm.response.json().results.map(p => p.id)).to.include(pm.collectionVariables.get("lesson_plan_id")));'),
    req("Headteacher approves", "POST", "/lesson-plans/{{lesson_plan_id}}/review/", token="head_token",
        body={"status": "approved", "comment": "Good"}, tests=status(200) + """
const p = pm.response.json();
pm.test("approved by headteacher", () => { pm.expect(p.status).to.eql("approved"); pm.expect(p.reviewed_by_name).to.include("Hellen"); });
"""),
    req("Teacher detail counts plans this term", "GET", "/teachers/{{teacher_user_id}}/", tests=status(200) + """
pm.test("1 plan", () => pm.expect(pm.response.json().lesson_plans_this_term).to.eql(1));
"""),
    req("Parent sees plans for their child's class", "GET", "/lesson-plans/", token="parent_token", tests=status(200) + """
pm.test("includes the class plan", () => pm.expect(pm.response.json().results.map(p => p.id)).to.include(pm.collectionVariables.get("lesson_plan_id")));
"""),
    req("Teacher A saves a draft without a term -> current term", "POST", "/lesson-plans/", token="teacher_token",
        body={"subject_id": "{{subject_id}}", "class_id": "{{class_id}}", "week_number": 4,
              "lesson_date": "2026-01-28", "topic": "Draft from the web form", "status": "draft"},
        tests=status(201) + """
const p = pm.response.json();
pm.collectionVariables.set("draft_plan_id", p.id);
pm.test("filed under the current term", () => pm.expect(p.term_id).to.eql(pm.collectionVariables.get("term_id")));
pm.test("stays a draft", () => pm.expect(p.status).to.eql("draft"));
"""),
    req("Headteacher's review queue doesn't include drafts", "GET", "/lesson-plans/?status=submitted&teacher_id={{teacher_user_id}}", token="head_token",
        tests=status(200) + 'pm.test("draft not listed", () => pm.expect(pm.response.json().results.map(p => p.id)).to.not.include(pm.collectionVariables.get("draft_plan_id")));'),
    req("Teacher A submits the draft", "PATCH", "/lesson-plans/{{draft_plan_id}}/", token="teacher_token",
        body={"status": "submitted"}, tests=status(200) + 'pm.test("submitted", () => pm.expect(pm.response.json().status).to.eql("submitted"));'),
    req("Headteacher marks it as seen", "POST", "/lesson-plans/{{draft_plan_id}}/review/", token="head_token",
        body={"status": "reviewed"}, tests=status(200) + 'pm.test("reviewed", () => pm.expect(pm.response.json().status).to.eql("reviewed"));'),
    req("Delete the extra plan", "DELETE", "/lesson-plans/{{draft_plan_id}}/", token="teacher_token", tests=status([200, 204])),
])

cleanup = folder("Z. Cleanup", "Deletes everything this run created.", [
    req("Delete lesson plan", "DELETE", "/lesson-plans/{{lesson_plan_id}}/", tests=status(200)),
    req("Clear CAT marks", "PUT", "/assessments/{{cat_id}}/marks/",
        body={"marks": [{"student_id": "{{student_id}}", "score": None}, {"student_id": "{{student2_id}}", "score": None}]}, tests=status(200)),
    req("Clear end-term marks", "PUT", "/assessments/{{assessment_id}}/marks/",
        body={"marks": [{"student_id": "{{student_id}}", "score": None}, {"student_id": "{{student2_id}}", "score": None}]}, tests=status(200)),
    req("Delete CAT", "DELETE", "/assessments/{{cat_id}}/", tests=status(204)),
    req("Delete end-term paper", "DELETE", "/assessments/{{assessment_id}}/", tests=status(204)),
    req("Delete exam thread", "DELETE", "/exams/threads/{{thread_id}}/", tests=status(200)),
    req("Delete student 1", "DELETE", "/students/{{student_id}}/", tests=status([200, 204])),
    req("Delete student 2", "DELETE", "/students/{{student2_id}}/", tests=status([200, 204])),
    req("Delete guardian (mother)", "DELETE", "/guardians/{{guardian_id}}/", tests=status([200, 204])),
    req("Delete guardian (father)", "DELETE", "/guardians/{{guardian2_id}}/", tests=status([200, 204])),
    req("Delete guardian (student 2)", "DELETE", "/guardians/{{guardian3_id}}/", tests=status([200, 204])),
    req("Delete staff profile", "DELETE", "/staff/{{staff_id}}/", tests=status(200)),
    req("Delete teacher A", "DELETE", "/users/{{teacher_user_id}}/", tests=status(200)),
    req("Delete teacher B", "DELETE", "/users/{{teacher2_user_id}}/", tests=status(200)),
    req("Delete headteacher", "DELETE", "/users/{{head_user_id}}/", tests=status(200)),
    req("Delete parent login", "DELETE", "/users/{{parent_user_id}}/", tests=status(200)),
    req("Delete subject A", "DELETE", "/subjects/{{subject_id}}/", tests=status([200, 204])),
    req("Delete subject B", "DELETE", "/subjects/{{subject2_id}}/", tests=status([200, 204])),
])

collection = {
    "info": {
        "_postman_id": "5b1f4f0e-8f3c-4b0a-9d62-acade0000e2e",
        "name": "ACK St. Mary's Kabete LMS - Academics E2E",
        "description": (__doc__ or "").strip() + "\n\nRun the folders in order (Collection Runner or newman): later "
        "folders use ids and tokens saved by earlier ones. Needs a current term and the class in {{class_id}} "
        "(defaults to the seed script's Grade 4 East). Set admin_email/admin_password for your database.",
        "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
    },
    "variable": [
        {"key": "base_url", "value": "http://localhost:8000/api"},
        {"key": "admin_email", "value": "admin@stmaryskabete.ac.ke"},
        {"key": "admin_password", "value": "Admin1234!"},
        {"key": "test_password", "value": "E2e-Passw0rd!"},
        {"key": "class_id", "value": "55555555-5555-5555-5555-555555555555"},
    ],
    "item": [setup, staff, assign, students, exams, marklist, report, parent, lock, plans, cleanup],
}

OUT.write_text(json.dumps(collection, indent=2) + "\n")
print(f"wrote {OUT} ({sum(len(f['item']) for f in collection['item'])} requests)")
