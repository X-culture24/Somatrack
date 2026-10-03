import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { admissionsApi } from "../../api/students";
import { feeStructuresApi, GRADE_LABELS } from "../../api/finance";
import { Button, Card, Field, Input, PageHeader, Select, Table, kes, useConfirm } from "../../components/ui";

// applying_for_grade uses the same grade numbering as fee_structures/
// class_groups (Playgroup=-2, PP1=-1, PP2=0, Grade N=N) so an admitted
// application's grade directly matches the fee structure it gets billed
// against once enrolled — GRADE_LABELS is the single source of truth,
// shared with Fee Structures and Classes.
const GRADE_LEVELS = Object.entries(GRADE_LABELS).map(([value, label]) => ({ value: Number(value), label }));

function gradeLabel(grade: number) {
  return GRADE_LABELS[grade] ?? `Grade level ${grade}`;
}

const STATUS_STYLES: Record<string, string> = {
  pending: "bg-slate-100 text-slate-600",
  reviewing: "bg-blue-100 text-blue-700",
  shortlisted: "bg-blue-100 text-blue-700",
  interview: "bg-amber-100 text-amber-700",
  accepted: "bg-green-100 text-green-700",
  admitted: "bg-green-100 text-green-700",
  rejected: "bg-red-100 text-red-700",
  deferred: "bg-slate-100 text-slate-600",
};

const emptyForm = {
  first_name: "",
  middle_name: "",
  last_name: "",
  gender: "female",
  date_of_birth: "",
  applying_for_grade: 4,
  applying_for_year: new Date().getFullYear(),
  applying_for_term: 1,
  former_school: "",
  kcpe_marks: "",
  guardian_name: "",
  guardian_phone: "",
  guardian_email: "",
  guardian_relation: "parent",
  residential_address: "",
};

export function AdmissionsPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({ queryKey: ["admissions"], queryFn: () => admissionsApi.list() });
  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: years } = useQuery({ queryKey: ["academic-years"], queryFn: academicsApi.years });
  const [form, setForm] = useState(emptyForm);

  // Fee estimate preview: resolves the academic year by matching its name
  // against the year the applicant is applying for, then looks up the fee
  // structure for the selected grade/term — the same data the school will
  // actually bill against once this application is enrolled.
  const matchingYear: any = (years ?? []).find((y: any) => String(y.name) === String(form.applying_for_year));
  const { data: feeStructures } = useQuery({
    queryKey: ["fees", matchingYear?.id, form.applying_for_grade],
    queryFn: () => feeStructuresApi.list({ academic_year_id: matchingYear!.id, grade: String(form.applying_for_grade) }),
    enabled: Boolean(matchingYear?.id),
  });
  const feeEstimate = (feeStructures ?? []).find((f) => f.term_number === form.applying_for_term);
  const [enrollForm, setEnrollForm] = useState<{ id: string | null; class_id: string; admission_no: string }>({
    id: null, class_id: "", admission_no: "",
  });

  const create = useMutation({
    mutationFn: () =>
      admissionsApi.create({
        ...form,
        kcpe_marks: form.kcpe_marks === "" ? null : Number(form.kcpe_marks),
        date_of_birth: form.date_of_birth || null,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admissions"] });
      setForm(emptyForm);
    },
  });
  const advance = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => admissionsApi.update(id, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admissions"] }),
  });
  const enroll = useMutation({
    mutationFn: () => admissionsApi.enroll(enrollForm.id!, { admission_no: enrollForm.admission_no, class_id: enrollForm.class_id || undefined }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admissions"] });
      qc.invalidateQueries({ queryKey: ["students"] });
      setEnrollForm({ id: null, class_id: "", admission_no: "" });
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not enroll this application. Check the class and admission number.");
    },
  });

  async function handleSubmit() {
    const fullName = [form.first_name, form.last_name].filter(Boolean).join(" ");
    const ok = await confirm({
      title: "Submit application",
      message: `Submit admission application for ${fullName} (${gradeLabel(form.applying_for_grade)})?`,
      confirmLabel: "Submit",
    });
    if (ok) create.mutate();
  }

  async function handleAdvance(a: any, status: string, label: string) {
    const ok = await confirm({
      title: label,
      message: `Mark ${a.first_name} ${a.last_name}'s application as "${status}"?`,
      confirmLabel: label,
    });
    if (ok) advance.mutate({ id: a.id, status });
  }

  async function handleEnroll(a: any) {
    if (!enrollForm.admission_no) return;
    const cls = (classes as any[] ?? []).find((c: any) => String(c.id) === enrollForm.class_id);
    const ok = await confirm({
      title: "Enroll learner",
      message: `Enroll ${a.first_name} ${a.last_name} as ${enrollForm.admission_no}${cls ? ` into ${cls.name}` : ""}? This creates the student and guardian records.`,
      confirmLabel: "Enroll",
    });
    if (ok) enroll.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Admissions" subtitle="Application → review → accept → enrol." />
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New application</h3>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="First name">
            <Input value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} />
          </Field>
          <Field label="Middle name">
            <Input value={form.middle_name} onChange={(e) => setForm({ ...form, middle_name: e.target.value })} />
          </Field>
          <Field label="Last name">
            <Input value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} />
          </Field>
          <Field label="Gender">
            <Select value={form.gender} onChange={(e) => setForm({ ...form, gender: e.target.value })}>
              <option value="female">Female</option>
              <option value="male">Male</option>
              <option value="other">Other</option>
            </Select>
          </Field>
          <Field label="Date of birth">
            <Input type="date" value={form.date_of_birth} onChange={(e) => setForm({ ...form, date_of_birth: e.target.value })} />
          </Field>
          <Field label="Applying for">
            <Select
              value={form.applying_for_grade}
              onChange={(e) => setForm({ ...form, applying_for_grade: Number(e.target.value) })}
            >
              {GRADE_LEVELS.map((g) => <option key={g.value} value={g.value}>{g.label}</option>)}
            </Select>
          </Field>
          <Field label="Year">
            <Input type="number" value={form.applying_for_year} onChange={(e) => setForm({ ...form, applying_for_year: Number(e.target.value) })} />
          </Field>
          <Field label="Term">
            <Select value={form.applying_for_term} onChange={(e) => setForm({ ...form, applying_for_term: Number(e.target.value) })}>
              <option value={1}>Term 1</option>
              <option value={2}>Term 2</option>
              <option value={3}>Term 3</option>
            </Select>
          </Field>
          <Field label="Former school">
            <Input value={form.former_school} onChange={(e) => setForm({ ...form, former_school: e.target.value })} />
          </Field>
          <Field label="KCPE marks (if applicable)">
            <Input type="number" value={form.kcpe_marks} onChange={(e) => setForm({ ...form, kcpe_marks: e.target.value })} />
          </Field>
          <Field label="Guardian name">
            <Input value={form.guardian_name} onChange={(e) => setForm({ ...form, guardian_name: e.target.value })} />
          </Field>
          <Field label="Guardian relation">
            <Select value={form.guardian_relation} onChange={(e) => setForm({ ...form, guardian_relation: e.target.value })}>
              <option value="mother">Mother</option>
              <option value="father">Father</option>
              <option value="guardian">Guardian</option>
              <option value="sibling">Sibling</option>
              <option value="other">Other</option>
            </Select>
          </Field>
          <Field label="Guardian phone">
            <Input value={form.guardian_phone} onChange={(e) => setForm({ ...form, guardian_phone: e.target.value })} />
          </Field>
          <Field label="Guardian email">
            <Input type="email" value={form.guardian_email} onChange={(e) => setForm({ ...form, guardian_email: e.target.value })} />
          </Field>
          <Field label="Residential address">
            <Input value={form.residential_address} onChange={(e) => setForm({ ...form, residential_address: e.target.value })} />
          </Field>
        </div>

        {matchingYear && (
          <div className="mt-4 rounded-xl bg-slate-50 p-3 text-sm">
            {feeEstimate ? (
              <div className="flex items-center justify-between">
                <span className="text-slate-600">
                  Estimated fee for {gradeLabel(form.applying_for_grade)}, Term {form.applying_for_term}:
                </span>
                <span className="font-semibold text-[#1b365d]">{kes(feeEstimate.total_day)}</span>
              </div>
            ) : (
              <p className="text-slate-400">
                No fee structure set up yet for {gradeLabel(form.applying_for_grade)} Term {form.applying_for_term},{" "}
                {form.applying_for_year} — set one up under Fee Structures before enrolling.
              </p>
            )}
          </div>
        )}
        <Button
          className="mt-4"
          onClick={handleSubmit}
          disabled={create.isPending || !form.first_name || !form.last_name || !form.guardian_phone || !form.guardian_name}
        >
          Submit application
        </Button>
      </Card>
      <Table headers={["Applicant", "Grade", "Guardian", "Status", ""]}>
        {(data ?? []).map((a: any) => (
          <tr key={a.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 font-medium">{a.first_name} {a.last_name}</td>
            <td className="px-4 py-3">{gradeLabel(a.applying_for_grade)}</td>
            <td className="px-4 py-3">{a.guardian_name} · {a.guardian_phone}</td>
            <td className="px-4 py-3">
              <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${STATUS_STYLES[a.status] ?? "bg-slate-100 text-slate-600"}`}>
                {a.status}
              </span>
            </td>
            <td className="px-4 py-3">
              {a.status === "pending" && (
                <Button variant="ghost" onClick={() => handleAdvance(a, "reviewing", "Start review")}>Review</Button>
              )}
              {(a.status === "reviewing" || a.status === "shortlisted" || a.status === "interview") && (
                <div className="flex gap-2">
                  <Button variant="ghost" onClick={() => handleAdvance(a, "accepted", "Accept")}>Accept</Button>
                  <Button variant="ghost" onClick={() => handleAdvance(a, "rejected", "Reject")}>Reject</Button>
                </div>
              )}
              {a.status === "accepted" ? (
                enrollForm.id === a.id ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <Select
                      className="w-40"
                      value={enrollForm.class_id}
                      onChange={(e) => setEnrollForm({ ...enrollForm, class_id: e.target.value })}
                    >
                      <option value="">Class…</option>
                      {(classes as any[] ?? []).map((c: any) => (
                        <option key={c.id} value={c.id}>{c.name}</option>
                      ))}
                    </Select>
                    <Input
                      className="w-36"
                      placeholder="Admission no."
                      value={enrollForm.admission_no}
                      onChange={(e) => setEnrollForm({ ...enrollForm, admission_no: e.target.value })}
                    />
                    <Button onClick={() => handleEnroll(a)}>Confirm</Button>
                  </div>
                ) : (
                  <Button variant="ghost" onClick={() => setEnrollForm({ id: a.id, class_id: "", admission_no: "" })}>
                    Enroll
                  </Button>
                )
              ) : null}
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
