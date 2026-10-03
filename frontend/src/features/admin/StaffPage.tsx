import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Fragment, useState } from "react";
import { schoolApi } from "../../api/school";
import { FileUploadField } from "../../components/FileUploadField";
import { COUNTRIES, KENYA_COUNTIES } from "../../lib/kenya";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";
import { type PickedAssignment, SubjectAssignmentPicker } from "./SubjectAssignmentPicker";

const STAFF_ROLES = [
  "HEAD_TEACHER", "DEPUTY_HEAD_TEACHER", "ACADEMIC_COORDINATOR", "CLASS_TEACHER", "SUBJECT_TEACHER",
  "LIBRARIAN", "TRANSPORT_COORDINATOR", "ADMISSIONS_OFFICER", "RECEPTIONIST", "FINANCE_OFFICER",
];

// Safaricom sandbox only ever accepts payouts to a handful of published
// test MSISDNs — a real phone number here will just get rejected by B2C
// ("The initiator information is invalid" or similar). Defaulting every new
// staff member's phone to this number is what makes "Disburse" on the
// Payroll page actually work end-to-end in this environment; swap in real
// numbers once this points at production Daraja credentials.
const SANDBOX_PAYOUT_MSISDN = "254708374149";

// Roles that teach and so can be given subjects at registration.
const TEACHING_ROLES = ["HEAD_TEACHER", "DEPUTY_HEAD_TEACHER", "ACADEMIC_COORDINATOR", "CLASS_TEACHER", "SUBJECT_TEACHER"];

const EMPLOYMENT_TYPES = [["tsc", "TSC"], ["bom", "Board of Management (BOM)"], ["contract", "Contract"], ["intern", "Intern"], ["volunteer", "Volunteer"]];

const emptyForm = {
  first_name: "", last_name: "", email: "", role: "SUBJECT_TEACHER", phone: SANDBOX_PAYOUT_MSISDN,
  gender: "", date_of_birth: "", national_id: "", residential_address: "",
  employee_no: "", tsc_no: "", employment_type: "", department_id: "", designation: "", date_of_joining: "",
  highest_qualification: "", basic_salary: "0",
  kra_pin: "", nhif_no: "", nssf_no: "", bank_name: "", bank_branch: "", bank_account_no: "",
  emergency_contact_name: "", emergency_contact_phone: "", emergency_contact_relation: "",
  middle_name: "", marital_status: "", religion: "", has_disability: false, disability_details: "",
  alternate_phone: "", postal_address: "",
  citizenship: "Kenya", home_county: "", residence_county: "", sub_county: "",
  blood_group: "", allergies: "", doctor_name: "", doctor_phone: "", preferred_hospital: "",
  teaching_subjects: "", training_institution: "", year_of_graduation: "", previous_employer: "", contract_end_date: "",
  photo_consent: false, photo_path: "", id_document_path: "", certificates_path: "",
  appointment_letter_path: "", good_conduct_path: "",
};

const MARITAL_STATUS = [["single", "Single"], ["married", "Married"], ["divorced", "Divorced"], ["separated", "Separated"], ["widowed", "Widowed"]];
const RELIGIONS = ["Christian", "Muslim", "Hindu", "Other", "None"];
// Employment types with a fixed end date.
const FIXED_TERM = ["bom", "contract", "intern", "volunteer"];

export function StaffPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data: staff } = useQuery({ queryKey: ["staff"], queryFn: () => schoolApi.staff() });
  const { data: departments } = useQuery({ queryKey: ["departments"], queryFn: schoolApi.departments });

  const [form, setForm] = useState(emptyForm);
  const [subjects, setSubjects] = useState<PickedAssignment[]>([]);
  // The login is created first; if the staff profile then fails (duplicate
  // employee no., a subject taken meanwhile…) a retry reuses that login
  // instead of tripping over the now-taken email.
  const [pendingUser, setPendingUser] = useState<{ id: string; email: string; temporary_password?: string } | null>(null);
  const [deptForm, setDeptForm] = useState({ name: "", code: "" });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [openId, setOpenId] = useState<string | null>(null);

  const isTeaching = TEACHING_ROLES.includes(form.role);

  const createStaff = useMutation({
    mutationFn: async () => {
      let u = pendingUser && pendingUser.email === form.email ? pendingUser : null;
      if (!u) {
        const { user, temporary_password } = await schoolApi.createUser({
          first_name: form.first_name, last_name: form.last_name, email: form.email,
          role: form.role, phone: form.phone,
        });
        u = { id: user.id, email: form.email, temporary_password };
        setPendingUser(u);
      }
      const { first_name: _f, last_name: _l, email: _e, role: _r, phone: _p, ...profile } = form;
      const sp = await schoolApi.createStaff({
        ...profile,
        user_id: u.id,
        department_id: form.department_id || null,
        date_of_birth: form.date_of_birth || null,
        date_of_joining: form.date_of_joining || null,
        basic_salary: Number(form.basic_salary) || 0,
        disability_details: form.has_disability ? form.disability_details : "",
        contract_end_date: FIXED_TERM.includes(form.employment_type) ? form.contract_end_date : "",
        assign_subjects: isTeaching ? subjects.map(({ class_id, subject_id }) => ({ class_id, subject_id })) : [],
      });
      return { sp, temporary_password: u.temporary_password };
    },
    onSuccess: ({ temporary_password }) => {
      qc.invalidateQueries({ queryKey: ["staff"] });
      qc.invalidateQueries({ queryKey: ["users"] });
      qc.invalidateQueries({ queryKey: ["class-subjects"] });
      setForm(emptyForm);
      setSubjects([]);
      setPendingUser(null);
      if (temporary_password) {
        window.alert(`Staff account created. One-time temporary password: ${temporary_password}\n\nThis is shown only once — share it with them directly.`);
      }
    },
    onError: (err: any) => {
      const d = err?.response?.data;
      const fields = d?.fields ? "\n\n" + Object.entries(d.fields).map(([k, v]) => `• ${k.replaceAll("_", " ")}: ${v}`).join("\n") : "";
      const clashes = d?.conflicts ? "\n\n" + d.conflicts.map((c: any) => `• ${c.class_name} ${c.subject_name} is taught by ${c.current_teacher_name}`).join("\n") : "";
      window.alert((d?.message ?? "Could not create the staff record. Check the email and employee number are unique.") + fields + clashes);
    },
  });
  const createDept = useMutation({
    mutationFn: () => schoolApi.createDepartment(deptForm),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["departments"] }); setDeptForm({ name: "", code: "" }); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not create department."),
  });
  const toggleActive = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) => schoolApi.updateStaff(id, { is_active: isActive }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["staff"] }); setEditingId(null); },
  });

  async function handleCreate() {
    const n = isTeaching ? subjects.length : 0;
    const ok = await confirm({
      title: "Create staff record",
      message: `Create a staff record and login for ${form.first_name} ${form.last_name} (${form.role.replaceAll("_", " ").toLowerCase()})${n ? `, teaching ${n} subject${n === 1 ? "" : "s"}` : ""}?`,
      confirmLabel: "Create staff",
    });
    if (ok) createStaff.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Staff directory" subtitle="Teaching and support staff, departments, and statutory records." />

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New staff member</h3>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Photo & documents</p>
        <div className="grid gap-3 md:grid-cols-3">
          <FileUploadField label="Upload passport photo" category="photo" value={form.photo_path}
            onUploaded={(path) => setForm((f) => ({ ...f, photo_path: path }))} />
          <Field label="Photographic consent">
            <label className="flex items-start gap-2 pt-2 text-sm">
              <input type="checkbox" className="mt-0.5" checked={form.photo_consent} onChange={(e) => setForm({ ...form, photo_consent: e.target.checked })} />
              Consents to their photo being used in school publications
            </label>
          </Field>
          <FileUploadField label="National ID copy" category="staff_id" value={form.id_document_path}
            onUploaded={(path) => setForm((f) => ({ ...f, id_document_path: path }))} />
          <FileUploadField label="Academic & professional certificates" category="certificate" value={form.certificates_path}
            onUploaded={(path) => setForm((f) => ({ ...f, certificates_path: path }))} />
          <FileUploadField label="Appointment / TSC posting letter" category="appointment_letter" value={form.appointment_letter_path}
            onUploaded={(path) => setForm((f) => ({ ...f, appointment_letter_path: path }))} />
          <FileUploadField label="Certificate of good conduct" category="good_conduct" value={form.good_conduct_path}
            onUploaded={(path) => setForm((f) => ({ ...f, good_conduct_path: path }))} />
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Personal details</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="First name *"><Input placeholder="E.g Jane" value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></Field>
          <Field label="Middle name"><Input value={form.middle_name} onChange={(e) => setForm({ ...form, middle_name: e.target.value })} /></Field>
          <Field label="Last name *"><Input placeholder="E.g Achieng Otieno" value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></Field>
          <Field label="Gender">
            <Select value={form.gender} onChange={(e) => setForm({ ...form, gender: e.target.value })}>
              <option value="">Select…</option>
              <option value="male">Male</option>
              <option value="female">Female</option>
            </Select>
          </Field>
          <Field label="Date of birth"><Input type="date" value={form.date_of_birth} onChange={(e) => setForm({ ...form, date_of_birth: e.target.value })} /></Field>
          <Field label="National ID no."><Input value={form.national_id} onChange={(e) => setForm({ ...form, national_id: e.target.value })} /></Field>
          <Field label="Marital status">
            <Select value={form.marital_status} onChange={(e) => setForm({ ...form, marital_status: e.target.value })}>
              <option value="">Select…</option>
              {MARITAL_STATUS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
            </Select>
          </Field>
          <Field label="Religion">
            <Select value={form.religion} onChange={(e) => setForm({ ...form, religion: e.target.value })}>
              <option value="">Select…</option>
              {RELIGIONS.map((r) => <option key={r} value={r}>{r}</option>)}
            </Select>
          </Field>
          <Field label="Disability">
            <Select value={form.has_disability ? "yes" : "no"} onChange={(e) => setForm({ ...form, has_disability: e.target.value === "yes" })}>
              <option value="no">No</option>
              <option value="yes">Yes</option>
            </Select>
          </Field>
          {form.has_disability && (
            <div className="md:col-span-3">
              <Field label="Describe the disability / support needed"><Input value={form.disability_details} onChange={(e) => setForm({ ...form, disability_details: e.target.value })} /></Field>
            </div>
          )}
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Contact</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Email (login) *"><Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
          <Field label="Phone (payroll M-Pesa number)">
            <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          </Field>
          <Field label="Alternate phone"><Input value={form.alternate_phone} onChange={(e) => setForm({ ...form, alternate_phone: e.target.value })} /></Field>
          <Field label="Postal address"><Input placeholder="P.O. Box …" value={form.postal_address} onChange={(e) => setForm({ ...form, postal_address: e.target.value })} /></Field>
        </div>

        <p className="mt-1 text-xs text-slate-400">
          Phone defaults to the Safaricom sandbox test number so "Disburse" on the Payroll page works immediately —
          replace it with the staff member's real number once this is pointed at production M-Pesa.
        </p>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Citizenship & residence</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Citizenship">
            <Select value={form.citizenship} onChange={(e) => setForm({ ...form, citizenship: e.target.value })}>
              {COUNTRIES.map((c) => <option key={c} value={c}>{c}</option>)}
            </Select>
          </Field>
          <Field label="Home county">
            <Select value={form.home_county} onChange={(e) => setForm({ ...form, home_county: e.target.value })}>
              <option value="">Select county</option>
              {KENYA_COUNTIES.map((c) => <option key={c} value={c}>{c}</option>)}
            </Select>
          </Field>
          <Field label="Current residential county">
            <Select value={form.residence_county} onChange={(e) => setForm({ ...form, residence_county: e.target.value })}>
              <option value="">Select county</option>
              {KENYA_COUNTIES.map((c) => <option key={c} value={c}>{c}</option>)}
            </Select>
          </Field>
          <Field label="Sub county"><Input value={form.sub_county} onChange={(e) => setForm({ ...form, sub_county: e.target.value })} /></Field>
          <Field label="Residence"><Input placeholder="Estate / village, town" value={form.residential_address} onChange={(e) => setForm({ ...form, residential_address: e.target.value })} /></Field>
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Employment</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Role">
            <Select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
              {STAFF_ROLES.map((r) => <option key={r} value={r}>{r.replaceAll("_", " ")}</option>)}
            </Select>
          </Field>
          <Field label="Employee no. *"><Input value={form.employee_no} onChange={(e) => setForm({ ...form, employee_no: e.target.value })} /></Field>
          <Field label="TSC no."><Input value={form.tsc_no} onChange={(e) => setForm({ ...form, tsc_no: e.target.value })} /></Field>
          <Field label="Employment type">
            <Select value={form.employment_type} onChange={(e) => setForm({ ...form, employment_type: e.target.value })}>
              <option value="">Select…</option>
              {EMPLOYMENT_TYPES.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
            </Select>
          </Field>
          <Field label="Department">
            <Select value={form.department_id} onChange={(e) => setForm({ ...form, department_id: e.target.value })}>
              <option value="">None</option>
              {(departments as any[] ?? []).map((d: any) => <option key={d.id} value={d.id}>{d.name}</option>)}
            </Select>
          </Field>
          <Field label="Designation"><Input value={form.designation} onChange={(e) => setForm({ ...form, designation: e.target.value })} placeholder="e.g. Class Teacher" /></Field>
          <Field label="Date of joining"><Input type="date" value={form.date_of_joining} onChange={(e) => setForm({ ...form, date_of_joining: e.target.value })} /></Field>
          {FIXED_TERM.includes(form.employment_type) && (
            <Field label="Contract end date"><Input type="date" value={form.contract_end_date} onChange={(e) => setForm({ ...form, contract_end_date: e.target.value })} /></Field>
          )}
          <Field label="Basic salary (KES)"><Input type="number" value={form.basic_salary} onChange={(e) => setForm({ ...form, basic_salary: e.target.value })} /></Field>
          <Field label="Previous school / employer"><Input value={form.previous_employer} onChange={(e) => setForm({ ...form, previous_employer: e.target.value })} /></Field>
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Qualifications</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Highest qualification"><Input value={form.highest_qualification} onChange={(e) => setForm({ ...form, highest_qualification: e.target.value })} placeholder="e.g. B.Ed (Arts)" /></Field>
          <Field label="College / university"><Input value={form.training_institution} onChange={(e) => setForm({ ...form, training_institution: e.target.value })} /></Field>
          <Field label="Year of graduation"><Input inputMode="numeric" maxLength={4} placeholder="E.g 2015" value={form.year_of_graduation} onChange={(e) => setForm({ ...form, year_of_graduation: e.target.value })} /></Field>
          <Field label="Subject specialisation"><Input placeholder="E.g Mathematics / Physics" value={form.teaching_subjects} onChange={(e) => setForm({ ...form, teaching_subjects: e.target.value })} /></Field>
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Medical</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Blood group"><Input placeholder="e.g. O+" value={form.blood_group} onChange={(e) => setForm({ ...form, blood_group: e.target.value })} /></Field>
          <Field label="Allergies"><Input value={form.allergies} onChange={(e) => setForm({ ...form, allergies: e.target.value })} /></Field>
          <Field label="Doctor's name"><Input value={form.doctor_name} onChange={(e) => setForm({ ...form, doctor_name: e.target.value })} /></Field>
          <Field label="Doctor's phone"><Input value={form.doctor_phone} onChange={(e) => setForm({ ...form, doctor_phone: e.target.value })} /></Field>
          <Field label="Preferred hospital"><Input value={form.preferred_hospital} onChange={(e) => setForm({ ...form, preferred_hospital: e.target.value })} /></Field>
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Statutory numbers</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="KRA PIN"><Input value={form.kra_pin} onChange={(e) => setForm({ ...form, kra_pin: e.target.value })} placeholder="A012345678P" /></Field>
          <Field label="NHIF / SHA no."><Input value={form.nhif_no} onChange={(e) => setForm({ ...form, nhif_no: e.target.value })} /></Field>
          <Field label="NSSF no."><Input value={form.nssf_no} onChange={(e) => setForm({ ...form, nssf_no: e.target.value })} /></Field>
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Bank details (optional)</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Bank name"><Input value={form.bank_name} onChange={(e) => setForm({ ...form, bank_name: e.target.value })} /></Field>
          <Field label="Branch"><Input value={form.bank_branch} onChange={(e) => setForm({ ...form, bank_branch: e.target.value })} /></Field>
          <Field label="Account no."><Input value={form.bank_account_no} onChange={(e) => setForm({ ...form, bank_account_no: e.target.value })} /></Field>
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Emergency contact / next of kin</p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Name"><Input value={form.emergency_contact_name} onChange={(e) => setForm({ ...form, emergency_contact_name: e.target.value })} /></Field>
          <Field label="Phone"><Input value={form.emergency_contact_phone} onChange={(e) => setForm({ ...form, emergency_contact_phone: e.target.value })} /></Field>
          <Field label="Relationship"><Input value={form.emergency_contact_relation} onChange={(e) => setForm({ ...form, emergency_contact_relation: e.target.value })} placeholder="e.g. spouse" /></Field>
        </div>

        {isTeaching && (
          <>
            <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Subjects to teach</p>
            <p className="mb-2 text-xs text-slate-500">
              Pick a class, then tick the subjects this teacher takes there. Repeat for each class. Subjects already held by
              another teacher can be moved on the Teacher Assignment page.
            </p>
            <SubjectAssignmentPicker value={subjects} onChange={setSubjects} />
          </>
        )}

        <Button
          className="mt-5"
          onClick={handleCreate}
          disabled={createStaff.isPending || !form.first_name || !form.last_name || !form.email || !form.employee_no}
        >
          {createStaff.isPending ? "Creating…" : "Create staff record"}
        </Button>
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Departments</h3>
        <div className="flex flex-wrap items-end gap-3">
          <Field label="Name"><Input value={deptForm.name} onChange={(e) => setDeptForm({ ...deptForm, name: e.target.value })} className="w-48" /></Field>
          <Field label="Code"><Input value={deptForm.code} onChange={(e) => setDeptForm({ ...deptForm, code: e.target.value })} className="w-28" placeholder="e.g. ACAD" /></Field>
          <Button onClick={() => createDept.mutate()} disabled={createDept.isPending || !deptForm.name || !deptForm.code}>Add department</Button>
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          {(departments as any[] ?? []).map((d: any) => (
            <span key={d.id} className="rounded-full bg-[#4a9eca]/10 px-3 py-1 text-xs font-medium text-[#1b365d]">{d.name}</span>
          ))}
        </div>
      </Card>

      <Table headers={["Employee no.", "Name", "Designation", "Department", "Subjects taught", "Phone", "Status", ""]}>
        {(staff as any[] ?? []).map((s: any) => (
          <Fragment key={s.id}>
          <tr className="hover:bg-slate-50">
            <td className="px-4 py-3 font-mono text-xs">{s.employee_no}</td>
            <td className="px-4 py-3">
              <p className="font-medium">{s.full_name ?? "-"}</p>
              {s.tsc_no && <p className="text-xs text-slate-400">TSC {s.tsc_no}</p>}
            </td>
            <td className="px-4 py-3">{s.designation || "-"}</td>
            <td className="px-4 py-3">{s.department_name || "-"}</td>
            <td className="px-4 py-3">
              {(s.assignments ?? []).length === 0 ? (
                <span className="text-xs text-slate-400">—</span>
              ) : (
                <div className="flex max-w-xs flex-wrap gap-1">
                  {(s.assignments as any[]).map((a: any) => (
                    <span key={a.id ?? `${a.class_id}${a.subject_id}`} className="rounded-full bg-[#4a9eca]/10 px-2 py-0.5 text-xs text-[#1b365d]">
                      {a.subject_name} <span className="text-slate-400">· {a.class_name}</span>
                    </span>
                  ))}
                </div>
              )}
            </td>
            <td className="px-4 py-3 text-xs text-slate-500">{s.phone || "—"}</td>
            <td className="px-4 py-3">
              {editingId === s.id ? (
                <div className="flex items-center gap-2">
                  <Button onClick={() => toggleActive.mutate({ id: s.id, isActive: !s.is_active })} disabled={toggleActive.isPending}>
                    {s.is_active ? "Deactivate" : "Activate"}
                  </Button>
                  <Button variant="ghost" onClick={() => setEditingId(null)}>Cancel</Button>
                </div>
              ) : (
                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${s.is_active ? "bg-green-100 text-green-700" : "bg-slate-100 text-slate-600"}`}>
                  {s.is_active ? "Active" : "Inactive"}
                </span>
              )}
            </td>
            <td className="px-4 py-3">
              {editingId !== s.id && (
                <div className="flex gap-2">
                  <Button variant="ghost" onClick={() => setOpenId(openId === s.id ? null : s.id)}>{openId === s.id ? "Hide" : "Details"}</Button>
                  <Button variant="ghost" onClick={() => setEditingId(s.id)}>Change status</Button>
                </div>
              )}
            </td>
          </tr>
          {openId === s.id && (
            <tr>
              <td colSpan={8} className="bg-slate-50 px-4 py-4"><StaffDetailsPanel s={s} /></td>
            </tr>
          )}
          </Fragment>
        ))}
      </Table>
    </div>
  );
}

const MARITAL_LABEL: Record<string, string> = Object.fromEntries(MARITAL_STATUS);

// Everything captured on the registration form, read-only.
function StaffDetailsPanel({ s }: { s: any }) {
  const rows: [string, string][] = [
    ["Gender", s.gender], ["Date of birth", s.date_of_birth ?? ""], ["National ID", s.national_id],
    ["Marital status", MARITAL_LABEL[s.marital_status] ?? ""], ["Religion", s.religion],
    ["Disability", s.has_disability ? s.disability_details || "Yes" : "No"],
    ["Email", s.email], ["Alternate phone", s.alternate_phone], ["Postal address", s.postal_address],
    ["Citizenship", s.citizenship], ["Home county", s.home_county],
    ["Residence", [s.residential_address, s.sub_county, s.residence_county].filter(Boolean).join(", ")],
    ["TSC no.", s.tsc_no], ["Employment type", (s.employment_type || "").toUpperCase()],
    ["Joined", s.date_of_joining ?? ""], ["Contract ends", s.contract_end_date ?? ""],
    ["Previous employer", s.previous_employer],
    ["Qualification", [s.highest_qualification, s.training_institution, s.year_of_graduation].filter(Boolean).join(", ")],
    ["Specialisation", s.teaching_subjects],
    ["Medical", [s.blood_group, s.allergies && `allergies: ${s.allergies}`].filter(Boolean).join(" · ")],
    ["Doctor", [s.doctor_name, s.doctor_phone, s.preferred_hospital].filter(Boolean).join(" · ")],
    ["Next of kin", [s.emergency_contact_name, s.emergency_contact_relation, s.emergency_contact_phone].filter(Boolean).join(" · ")],
    ["KRA / NHIF / NSSF", [s.kra_pin, s.nhif_no, s.nssf_no].filter(Boolean).join(" · ")],
    ["Bank", [s.bank_name, s.bank_branch, s.bank_account_no].filter(Boolean).join(" · ")],
  ];
  const docs: [string, string][] = [
    ["Photo", s.photo_path], ["National ID", s.id_document_path], ["Certificates", s.certificates_path],
    ["Appointment letter", s.appointment_letter_path], ["Good conduct", s.good_conduct_path],
  ];
  return (
    <div className="space-y-3">
      <dl className="grid gap-x-6 gap-y-1.5 text-sm sm:grid-cols-2 lg:grid-cols-3">
        {rows.map(([label, value]) => (
          <div key={label} className="flex justify-between gap-3 border-b border-slate-100 py-1">
            <dt className="text-slate-500">{label}</dt>
            <dd className="text-right font-medium text-slate-700">{value || "—"}</dd>
          </div>
        ))}
      </dl>
      <div className="flex flex-wrap gap-1.5 text-xs">
        {docs.map(([label, path]) => (
          <span key={label} className={`rounded-full px-2 py-0.5 font-medium ${path ? "bg-green-100 text-green-700" : "bg-slate-100 text-slate-500"}`}>
            {path ? "✓" : "✗"} {label}
          </span>
        ))}
        <span className={`rounded-full px-2 py-0.5 font-medium ${s.photo_consent ? "bg-green-100 text-green-700" : "bg-slate-100 text-slate-500"}`}>
          Photo consent {s.photo_consent ? "given" : "not given"}
        </span>
      </div>
    </div>
  );
}
