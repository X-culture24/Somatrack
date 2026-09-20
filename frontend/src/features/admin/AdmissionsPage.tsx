import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { admissionsApi } from "../../api/students";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

const GRADES = ["Playgroup","Pre-Primary 1","Pre-Primary 2","Grade 1","Grade 2","Grade 3","Grade 4","Grade 5","Grade 6","Grade 7","Grade 8","Grade 9"];

export function AdmissionsPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({ queryKey: ["admissions"], queryFn: admissionsApi.list });
  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const [form, setForm] = useState({
    applicant_name: "", gender: "F", seeking_grade: "Grade 1",
    guardian_name: "", guardian_phone: "",
  });
  const [enrollForm, setEnrollForm] = useState<{ id: number | null; class_group: string; admission_no: string }>({
    id: null, class_group: "", admission_no: "",
  });

  const create = useMutation({
    mutationFn: () => {
      const names = form.applicant_name.trim().split(/\s+/);
      return admissionsApi.create({
        pupil_first_name: names[0],
        pupil_last_name: names.length > 1 ? names[names.length - 1] : names[0],
        pupil_middle_name: names.length > 2 ? names.slice(1, -1).join(" ") : "",
        gender: form.gender,
        seeking_grade: form.seeking_grade,
        parent_guardian_name: form.guardian_name,
        parent_guardian_phone: form.guardian_phone,
      });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admissions"] });
      setForm({ applicant_name: "", gender: "F", seeking_grade: "Grade 1", guardian_name: "", guardian_phone: "" });
    },
  });
  const approve = useMutation({
    mutationFn: (id: number) => admissionsApi.update(id, { status: "approved" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admissions"] }),
  });
  const enroll = useMutation({
    mutationFn: () => admissionsApi.enroll(enrollForm.id!, { class_group: enrollForm.class_group, admission_no: enrollForm.admission_no }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admissions"] });
      qc.invalidateQueries({ queryKey: ["students"] });
      setEnrollForm({ id: null, class_group: "", admission_no: "" });
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.detail ?? "Could not enroll this application. Check the class and admission number.");
    },
  });

  async function handleSubmit() {
    const ok = await confirm({
      title: "Submit application",
      message: `Submit admission application for ${form.applicant_name} (${form.seeking_grade})?`,
      confirmLabel: "Submit",
    });
    if (ok) create.mutate();
  }

  async function handleApprove(a: any) {
    const ok = await confirm({
      title: "Approve application",
      message: `Approve admission for ${a.applicant_name} into ${a.seeking_grade}? An admission letter can be generated after approval.`,
      confirmLabel: "Approve",
    });
    if (ok) approve.mutate(a.id);
  }

  async function handleEnroll(a: any) {
    if (!enrollForm.class_group || !enrollForm.admission_no) return;
    const cls = (classes as any[] ?? []).find((c: any) => String(c.id) === enrollForm.class_group);
    const ok = await confirm({
      title: "Enroll learner",
      message: `Enroll ${a.applicant_name} as ${enrollForm.admission_no} into ${cls?.grade ?? ""} ${cls?.stream_name ?? ""}? This creates the learner record and the first term invoice.`,
      confirmLabel: "Enroll",
    });
    if (ok) enroll.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Admissions" subtitle="Application → KES 500 assessment fee → interview → enrolment." />
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New application</h3>
        <div className="grid gap-3 md:grid-cols-2">
          <Field label="Learner name">
            <Input value={form.applicant_name} onChange={(e) => setForm({ ...form, applicant_name: e.target.value })} />
          </Field>
          <Field label="Gender">
            <Select value={form.gender} onChange={(e) => setForm({ ...form, gender: e.target.value })}>
              <option value="F">Female</option>
              <option value="M">Male</option>
            </Select>
          </Field>
          <Field label="Guardian name">
            <Input value={form.guardian_name} onChange={(e) => setForm({ ...form, guardian_name: e.target.value })} />
          </Field>
          <Field label="Guardian phone">
            <Input value={form.guardian_phone} onChange={(e) => setForm({ ...form, guardian_phone: e.target.value })} />
          </Field>
          <Field label="Seeking grade">
            <Select value={form.seeking_grade} onChange={(e) => setForm({ ...form, seeking_grade: e.target.value })}>
              {GRADES.map((g) => <option key={g}>{g}</option>)}
            </Select>
          </Field>
        </div>
        <Button
          className="mt-4"
          onClick={handleSubmit}
          disabled={create.isPending || !form.applicant_name || !form.guardian_phone}
        >
          Submit application
        </Button>
      </Card>
      <Table headers={["Applicant", "Grade", "Guardian", "Status", ""]}>
        {(data ?? []).map((a: any) => (
          <tr key={a.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 font-medium">{a.applicant_name}</td>
            <td className="px-4 py-3">{a.seeking_grade}</td>
            <td className="px-4 py-3">{a.guardian_name} · {a.guardian_phone}</td>
            <td className="px-4 py-3">
              <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                a.status === "enrolled" ? "bg-green-100 text-green-700"
                : a.status === "approved" ? "bg-blue-100 text-blue-700"
                : a.status === "declined" ? "bg-red-100 text-red-700"
                : "bg-slate-100 text-slate-600"
              }`}>
                {a.status}
              </span>
            </td>
            <td className="px-4 py-3">
              {a.status !== "approved" && a.status !== "enrolled" && a.status !== "declined" ? (
                <Button variant="ghost" onClick={() => handleApprove(a)}>Approve</Button>
              ) : null}
              {a.status === "approved" ? (
                enrollForm.id === a.id ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <Select
                      className="w-40"
                      value={enrollForm.class_group}
                      onChange={(e) => setEnrollForm({ ...enrollForm, class_group: e.target.value })}
                    >
                      <option value="">Class…</option>
                      {(classes as any[] ?? []).map((c: any) => (
                        <option key={c.id} value={c.id}>{c.grade} {c.stream_name}</option>
                      ))}
                    </Select>
                    <Input
                      className="w-36"
                      placeholder="Admission no."
                      value={enrollForm.admission_no}
                      onChange={(e) => setEnrollForm({ ...enrollForm, admission_no: e.target.value })}
                    />
                    <Button
                      onClick={() => handleEnroll(a)}
                      disabled={enroll.isPending || !enrollForm.class_group || !enrollForm.admission_no}
                    >
                      Enroll
                    </Button>
                    <Button variant="ghost" onClick={() => setEnrollForm({ id: null, class_group: "", admission_no: "" })}>
                      Cancel
                    </Button>
                  </div>
                ) : (
                  <Button variant="ghost" onClick={() => setEnrollForm({ id: a.id, class_group: "", admission_no: "" })}>
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
