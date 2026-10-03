import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { academicsApi } from "../../api/academics";
import { clubsApi, transportApi } from "../../api/clubs";
import { studentsApi } from "../../api/students";
import { FileUploadField } from "../../components/FileUploadField";
import { Button, Card, Field, Input, PageHeader, Select, Table, kes, useConfirm } from "../../components/ui";
import { COUNTRIES, KENYA_COUNTIES } from "../../lib/kenya";

const emptyForm = {
  admission_no: "",
  admission_date: "",
  first_name: "",
  middle_name: "",
  last_name: "",
  gender: "",
  date_of_birth: "",
  national_id: "",
  birth_certificate_no: "",
  kemis_number: "",
  assessment_number: "",
  phone: "",
  email: "",
  current_class_id: "",
  boarding_status: "day",
  blood_group: "",
  allergies: "",
  chronic_conditions: "",
  residential_address: "",
  former_school: "",
  kcpe_marks: "",
  kcpe_grade: "",
  upi: "",
  status: "active",
  uses_transport: false,
  transport_route_id: "",
  photo_path: "",
  birth_cert_document_path: "",
  club_ids: [] as string[],
  // welfare, medical contacts, sponsorship and home location
  photo_consent: false,
  parental_status: "both_alive",
  has_special_needs: false,
  special_needs: "",
  religion: "Christian",
  doctor_name: "",
  doctor_phone: "",
  preferred_hospital: "",
  has_scholarship: false,
  scholarship_type: "",
  sponsor_name: "",
  sponsor_phone: "",
  sponsor_location: "",
  sponsor_contact_person: "",
  sponsor_contact_details: "",
  citizenship: "Kenya",
  home_county: "",
  residence_county: "",
  sub_county: "",
};

const PARENTAL_STATUS = [
  ["both_alive", "Both parents alive"],
  ["father_deceased", "Single orphan — father deceased"],
  ["mother_deceased", "Single orphan — mother deceased"],
  ["both_deceased", "Total orphan — both deceased"],
];
const RELIGIONS = ["Christian", "Muslim", "Hindu", "Other", "None"];
const SCHOLARSHIP_TYPES = [
  ["full", "Full scholarship"], ["partial", "Partial scholarship"], ["bursary", "Bursary"],
  ["sponsorship", "Sponsorship"], ["other", "Other"],
];

// Inline guardian — every student needs at least one on file; a second
// (usually the other parent) is optional.
const emptyGuardian = {
  first_name: "", last_name: "", relation: "mother", phone: "", alternate_phone: "", email: "",
  national_id: "", occupation: "", employer: "", residential_address: "", postal_address: "",
  has_custody: true, id_document_path: "",
};
type GuardianForm = typeof emptyGuardian;

export function StudentsPage() {
  const [q, setQ] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [guardians, setGuardians] = useState<GuardianForm[]>([{ ...emptyGuardian }]);
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();

  const { data, isLoading } = useQuery({
    queryKey: ["students", q],
    queryFn: () => studentsApi.list({ search: q }),
  });
  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: clubs } = useQuery({ queryKey: ["clubs"], queryFn: () => clubsApi.list() });
  const { data: routes } = useQuery({ queryKey: ["transport-routes"], queryFn: () => transportApi.routes() });

  const create = useMutation({
    mutationFn: () =>
      studentsApi.create({
        admission_no: form.admission_no,
        admission_date: form.admission_date || null,
        first_name: form.first_name,
        middle_name: form.middle_name,
        last_name: form.last_name,
        gender: form.gender,
        date_of_birth: form.date_of_birth || null,
        national_id: form.national_id,
        birth_certificate_no: form.birth_certificate_no,
        kemis_number: form.kemis_number,
        assessment_number: form.assessment_number,
        phone: form.phone,
        email: form.email,
        current_class_id: form.current_class_id || null,
        boarding_status: form.boarding_status,
        blood_group: form.blood_group,
        allergies: form.allergies,
        chronic_conditions: form.chronic_conditions,
        residential_address: form.residential_address,
        former_school: form.former_school,
        kcpe_marks: form.kcpe_marks === "" ? null : Number(form.kcpe_marks),
        kcpe_grade: form.kcpe_grade,
        upi: form.upi,
        status: form.status,
        uses_transport: form.uses_transport,
        transport_route_id: form.uses_transport ? (form.transport_route_id || null) : null,
        photo_path: form.photo_path,
        birth_cert_document_path: form.birth_cert_document_path,
        club_ids: form.club_ids,
        photo_consent: form.photo_consent,
        parental_status: form.parental_status,
        has_special_needs: form.has_special_needs,
        special_needs: form.has_special_needs ? form.special_needs : "",
        religion: form.religion,
        doctor_name: form.doctor_name,
        doctor_phone: form.doctor_phone,
        preferred_hospital: form.preferred_hospital,
        has_scholarship: form.has_scholarship,
        scholarship_type: form.has_scholarship ? form.scholarship_type : "",
        sponsor_name: form.has_scholarship ? form.sponsor_name : "",
        sponsor_phone: form.has_scholarship ? form.sponsor_phone : "",
        sponsor_location: form.has_scholarship ? form.sponsor_location : "",
        sponsor_contact_person: form.has_scholarship ? form.sponsor_contact_person : "",
        sponsor_contact_details: form.has_scholarship ? form.sponsor_contact_details : "",
        citizenship: form.citizenship,
        home_county: form.home_county,
        residence_county: form.residence_county,
        sub_county: form.sub_county,
        guardians: guardians
          .filter((g) => g.phone)
          .map((g, i) => ({ ...g, is_primary: i === 0, contact_priority: i + 1 })),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["students"] });
      setForm(emptyForm);
      setGuardians([{ ...emptyGuardian }]);
      setShowForm(false);
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not register this student — check the fields above.");
    },
  });

  async function handleSubmit() {
    const ok = await confirm({
      title: "Register student",
      message: `Register ${form.first_name} ${form.last_name} (admission no. ${form.admission_no})?`,
      confirmLabel: "Register",
    });
    if (ok) create.mutate();
  }

  function setGuardian(i: number, patch: Partial<GuardianForm>) {
    setGuardians((gs) => gs.map((g, j) => (j === i ? { ...g, ...patch } : g)));
  }

  function toggleClub(clubId: string) {
    setForm((f) => ({
      ...f,
      club_ids: f.club_ids.includes(clubId) ? f.club_ids.filter((c) => c !== clubId) : [...f.club_ids, clubId],
    }));
  }

  return (
    <div>
      {dialog}
      <PageHeader
        title="Students"
        subtitle="Profiles, class placement, transport, and fee balance."
        actions={<Button onClick={() => setShowForm((v) => !v)}>{showForm ? "Close" : "+ Register student"}</Button>}
      />

      {showForm && (
        <Card className="mb-6">
          <h3 className="mb-3 font-semibold text-[#1b365d]">Register a new student</h3>

          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Admission</p>
          <div className="grid gap-3 md:grid-cols-4">
            <Field label="Admission no. *"><Input value={form.admission_no} onChange={(e) => setForm({ ...form, admission_no: e.target.value })} /></Field>
            <Field label="Date of admission"><Input type="date" value={form.admission_date} onChange={(e) => setForm({ ...form, admission_date: e.target.value })} /></Field>
            <Field label="Class / grade admitted to">
              <Select value={form.current_class_id} onChange={(e) => setForm({ ...form, current_class_id: e.target.value })}>
                <option value="">Unassigned</option>
                {(classes as any[] ?? []).map((c: any) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </Select>
            </Field>
            <Field label="Boarding / Day scholar *">
              <Select value={form.boarding_status} onChange={(e) => setForm({ ...form, boarding_status: e.target.value })}>
                <option value="day">Day scholar</option>
                <option value="boarding">Boarder</option>
              </Select>
            </Field>
          </div>

          <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Identification numbers</p>
          <div className="grid gap-3 md:grid-cols-4">
            <Field label="Student UPI number"><Input value={form.upi} onChange={(e) => setForm({ ...form, upi: e.target.value })} /></Field>
            <Field label="Birth certificate number"><Input value={form.birth_certificate_no} onChange={(e) => setForm({ ...form, birth_certificate_no: e.target.value })} /></Field>
            <Field label="Assessment number"><Input value={form.assessment_number} onChange={(e) => setForm({ ...form, assessment_number: e.target.value })} /></Field>
          </div>

          <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Photo & documents</p>
          <div className="grid gap-3 md:grid-cols-3">
            <FileUploadField label="Upload passport photo" category="photo" value={form.photo_path}
              onUploaded={(path) => setForm((f) => ({ ...f, photo_path: path }))} />
            <FileUploadField label="Upload birth certificate" category="birth_cert" value={form.birth_cert_document_path}
              onUploaded={(path) => setForm((f) => ({ ...f, birth_cert_document_path: path }))} />
            <Field label="Photographic consent">
              <label className="flex items-start gap-2 pt-2 text-sm">
                <input type="checkbox" className="mt-0.5" checked={form.photo_consent} onChange={(e) => setForm({ ...form, photo_consent: e.target.checked })} />
                Parent consents to the learner's photo being used in school publications
              </label>
            </Field>
          </div>

          <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Learner details</p>
          <div className="grid gap-3 md:grid-cols-4">
            <Field label="First name *"><Input placeholder="E.g John" value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></Field>
            <Field label="Middle name"><Input value={form.middle_name} onChange={(e) => setForm({ ...form, middle_name: e.target.value })} /></Field>
            <Field label="Last names *"><Input placeholder="E.g Doe Brown" value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></Field>
            <Field label="Date of birth"><Input type="date" value={form.date_of_birth} onChange={(e) => setForm({ ...form, date_of_birth: e.target.value })} /></Field>
            <Field label="Gender *">
              <Select value={form.gender} onChange={(e) => setForm({ ...form, gender: e.target.value })}>
                <option value="">Select…</option>
                <option value="male">Male</option>
                <option value="female">Female</option>
              </Select>
            </Field>
            <Field label="Student status">
              <Select value={form.parental_status} onChange={(e) => setForm({ ...form, parental_status: e.target.value })}>
                {PARENTAL_STATUS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
              </Select>
            </Field>
            <Field label="Religion">
              <Select value={form.religion} onChange={(e) => setForm({ ...form, religion: e.target.value })}>
                {RELIGIONS.map((r) => <option key={r} value={r}>{r}</option>)}
              </Select>
            </Field>
            <Field label="Special need">
              <Select value={form.has_special_needs ? "yes" : "no"} onChange={(e) => setForm({ ...form, has_special_needs: e.target.value === "yes" })}>
                <option value="no">No</option>
                <option value="yes">Yes</option>
              </Select>
            </Field>
          </div>

          {form.has_special_needs && (
            <div className="mt-3">
              <Field label="Describe the special need">
                <Input value={form.special_needs} onChange={(e) => setForm({ ...form, special_needs: e.target.value })} placeholder="E.g hearing impairment, uses a wheelchair" />
              </Field>
            </div>
          )}

          <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Previous schooling</p>
          <div className="grid gap-3 md:grid-cols-4">
            <Field label="Former school"><Input value={form.former_school} onChange={(e) => setForm({ ...form, former_school: e.target.value })} /></Field>
            <Field label="Entry marks"><Input type="number" value={form.kcpe_marks} onChange={(e) => setForm({ ...form, kcpe_marks: e.target.value })} /></Field>
          </div>

          <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Medical</p>
          <div className="grid gap-3 md:grid-cols-4">
            <Field label="Allergies"><Input value={form.allergies} onChange={(e) => setForm({ ...form, allergies: e.target.value })} /></Field>
            <Field label="Doctor's name"><Input value={form.doctor_name} onChange={(e) => setForm({ ...form, doctor_name: e.target.value })} /></Field>
            <Field label="Doctor's phone"><Input value={form.doctor_phone} onChange={(e) => setForm({ ...form, doctor_phone: e.target.value })} /></Field>
            <Field label="Preferred hospital"><Input value={form.preferred_hospital} onChange={(e) => setForm({ ...form, preferred_hospital: e.target.value })} /></Field>
          </div>

          <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Scholarship / sponsorship</p>
          <div className="grid gap-3 md:grid-cols-4">
            <Field label="Scholarship">
              <Select value={form.has_scholarship ? "yes" : "no"} onChange={(e) => setForm({ ...form, has_scholarship: e.target.value === "yes" })}>
                <option value="no">No</option>
                <option value="yes">Yes</option>
              </Select>
            </Field>
            {form.has_scholarship && (
              <>
                <Field label="Type">
                  <Select value={form.scholarship_type} onChange={(e) => setForm({ ...form, scholarship_type: e.target.value })}>
                    <option value="">Select option</option>
                    {SCHOLARSHIP_TYPES.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
                  </Select>
                </Field>
                <Field label="Sponsor details"><Input placeholder="E.g Equity Bank, KCB etc." value={form.sponsor_name} onChange={(e) => setForm({ ...form, sponsor_name: e.target.value })} /></Field>
                <Field label="Phone"><Input value={form.sponsor_phone} onChange={(e) => setForm({ ...form, sponsor_phone: e.target.value })} /></Field>
                <Field label="Location"><Input value={form.sponsor_location} onChange={(e) => setForm({ ...form, sponsor_location: e.target.value })} /></Field>
                <Field label="Contact person"><Input value={form.sponsor_contact_person} onChange={(e) => setForm({ ...form, sponsor_contact_person: e.target.value })} /></Field>
                <Field label="Contact details"><Input value={form.sponsor_contact_details} onChange={(e) => setForm({ ...form, sponsor_contact_details: e.target.value })} /></Field>
              </>
            )}
          </div>

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

          <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Transport</p>
          <div className="grid gap-3 md:grid-cols-4">
            <Field label="Uses school transport">
              <label className="flex items-center gap-2 pt-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.uses_transport}
                  onChange={(e) => setForm({ ...form, uses_transport: e.target.checked })}
                />
                Yes
              </label>
            </Field>
            {form.uses_transport && (
              <Field label="Route">
                <Select value={form.transport_route_id} onChange={(e) => setForm({ ...form, transport_route_id: e.target.value })}>
                  <option value="">Select route…</option>
                  {(routes as any[] ?? []).map((rt: any) => (
                    <option key={rt.id} value={rt.id}>{rt.route_name}</option>
                  ))}
                </Select>
              </Field>
            )}
          </div>

          <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Co-curricular (clubs)</p>
          <div className="flex flex-wrap gap-x-5 gap-y-2">
            {(clubs as any[] ?? []).length === 0 && <p className="text-sm text-slate-400">No clubs set up yet.</p>}
            {(clubs as any[] ?? []).map((c: any) => (
              <label key={c.id} className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={form.club_ids.includes(c.id)} onChange={() => toggleClub(c.id)} />
                {c.name}
              </label>
            ))}
          </div>

          <details className="mt-5">
            <summary className="cursor-pointer text-xs font-semibold uppercase tracking-wide text-slate-400">Other details (optional)</summary>
            <div className="mt-3 grid gap-3 md:grid-cols-4">
              <Field label="KEMIS number"><Input value={form.kemis_number} onChange={(e) => setForm({ ...form, kemis_number: e.target.value })} /></Field>
              <Field label="National ID (if any)"><Input value={form.national_id} onChange={(e) => setForm({ ...form, national_id: e.target.value })} /></Field>
              <Field label="Learner phone"><Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
              <Field label="Learner email"><Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
              <Field label="Blood group"><Input placeholder="e.g. O+" value={form.blood_group} onChange={(e) => setForm({ ...form, blood_group: e.target.value })} /></Field>
              <Field label="Chronic conditions"><Input value={form.chronic_conditions} onChange={(e) => setForm({ ...form, chronic_conditions: e.target.value })} /></Field>
            </div>
          </details>

          {guardians.map((g, i) => (
            <div key={i}>
              <div className="mb-2 mt-5 flex items-center justify-between">
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">
                  {i === 0 ? "Primary parent / guardian (contacted first)" : "Second parent / guardian"}
                </p>
                {i > 0 && (
                  <button type="button" className="text-xs text-red-600 hover:underline" onClick={() => setGuardians((gs) => gs.filter((_, j) => j !== i))}>
                    Remove
                  </button>
                )}
              </div>
              <div className="grid gap-3 md:grid-cols-4">
                <Field label="First name"><Input value={g.first_name} onChange={(e) => setGuardian(i, { first_name: e.target.value })} /></Field>
                <Field label="Last name"><Input value={g.last_name} onChange={(e) => setGuardian(i, { last_name: e.target.value })} /></Field>
                <Field label="Relation">
                  <Select value={g.relation} onChange={(e) => setGuardian(i, { relation: e.target.value })}>
                    <option value="mother">Mother</option>
                    <option value="father">Father</option>
                    <option value="guardian">Guardian</option>
                    <option value="sibling">Sibling</option>
                    <option value="other">Other</option>
                  </Select>
                </Field>
                <Field label="National ID"><Input value={g.national_id} onChange={(e) => setGuardian(i, { national_id: e.target.value })} /></Field>
                <Field label="Phone (SMS / M-Pesa)"><Input value={g.phone} onChange={(e) => setGuardian(i, { phone: e.target.value })} /></Field>
                <Field label="Alternate phone"><Input value={g.alternate_phone} onChange={(e) => setGuardian(i, { alternate_phone: e.target.value })} /></Field>
                <Field label="Email"><Input type="email" value={g.email} onChange={(e) => setGuardian(i, { email: e.target.value })} /></Field>
                <Field label="Occupation"><Input value={g.occupation} onChange={(e) => setGuardian(i, { occupation: e.target.value })} /></Field>
                <Field label="Employer"><Input value={g.employer} onChange={(e) => setGuardian(i, { employer: e.target.value })} /></Field>
                <Field label="Residential address"><Input value={g.residential_address} onChange={(e) => setGuardian(i, { residential_address: e.target.value })} /></Field>
                <Field label="Postal address"><Input value={g.postal_address} onChange={(e) => setGuardian(i, { postal_address: e.target.value })} /></Field>
                <Field label="Has custody">
                  <label className="flex items-center gap-2 pt-2 text-sm">
                    <input type="checkbox" checked={g.has_custody} onChange={(e) => setGuardian(i, { has_custody: e.target.checked })} />
                    Yes
                  </label>
                </Field>
                <FileUploadField label="ID document" category="guardian_id" value={g.id_document_path}
                  onUploaded={(path) => setGuardian(i, { id_document_path: path })} />
              </div>
            </div>
          ))}
          {guardians.length < 2 && (
            <Button variant="ghost" className="mt-3" onClick={() => setGuardians((gs) => [...gs, { ...emptyGuardian, relation: "father" }])}>
              + Add second parent / guardian
            </Button>
          )}
          <p className="mt-2 text-xs text-slate-400">Guardians are linked on save; a guardian without a phone number is skipped.</p>

          <Button
            className="mt-5"
            onClick={handleSubmit}
            disabled={create.isPending || !form.admission_no || !form.first_name || !form.last_name || !form.gender || !guardians[0].phone}
          >
            Register student
          </Button>
        </Card>
      )}

      <Input
        className="mb-4 max-w-sm"
        placeholder="Search name or admission no."
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      {isLoading ? (
        <p className="py-8 text-center text-sm text-slate-500">Loading students…</p>
      ) : (
        <Table headers={["#", "Adm No.", "Name", "Gender", "Class", "Status", "Parent", "Phone", "Balance", ""]}>
          {(data ?? []).map((s: any, i: number) => (
            <tr key={s.id} className="hover:bg-slate-50">
              <td className="px-3 py-2.5 text-slate-400 text-sm">{i + 1}</td>
              <td className="px-3 py-2.5 font-medium text-[#1b365d] text-sm">{s.admission_no}</td>
              <td className="px-3 py-2.5 font-medium text-sm">{s.full_name}</td>
              <td className="px-3 py-2.5 text-sm capitalize text-slate-600">{s.gender}</td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.class_name || "—"}</td>
              <td className="px-3 py-2.5">
                <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${
                  s.status === "active" ? "bg-green-100 text-green-700"
                  : s.status === "transferred" ? "bg-amber-100 text-amber-700"
                  : s.status === "withdrawn" ? "bg-red-100 text-red-700"
                  : "bg-slate-100 text-slate-600"}`}>
                  {s.status}
                </span>
              </td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.parent_name || "—"}</td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.parent_phone || "—"}</td>
              <td className={`px-3 py-2.5 text-sm font-medium ${Number(s.fee_balance ?? 0) > 0 ? "text-red-600" : "text-green-600"}`}>
                {kes(s.fee_balance)}
              </td>
              <td className="px-3 py-2.5">
                <button
                  onClick={() => navigate(`/admin/students/${s.id}`)}
                  className="rounded bg-[#1b365d] px-2.5 py-1 text-xs font-semibold text-white hover:bg-[#152a49]"
                >
                  Profile
                </button>
              </td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  );
}
