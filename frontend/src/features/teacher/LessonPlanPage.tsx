import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

export function LessonPlanPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [form, setForm] = useState({ class_group: "", subject: "", week_starting: "", strand: "", objectives: "", activities: "" });

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: subjects } = useQuery({ queryKey: ["subjects"], queryFn: () => academicsApi.subjects() });
  const { data: plans } = useQuery({ queryKey: ["lesson-plans"], queryFn: () => academicsApi.lessonPlans() });

  const save = useMutation({
    mutationFn: () => academicsApi.createLessonPlan(form),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["lesson-plans"] }); setForm({ class_group: "", subject: "", week_starting: "", strand: "", objectives: "", activities: "" }); },
  });

  async function handleSave() {
    const ok = await confirm({ title: "Save lesson plan", message: `Save plan for week of ${form.week_starting}?`, confirmLabel: "Save" });
    if (ok) save.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Lesson plans" subtitle="CBC scheme of work — weekly planning per class and subject." />
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New lesson plan</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Class"><Select value={form.class_group} onChange={(e) => setForm({ ...form, class_group: e.target.value })}><option value="">Select…</option>{(classes as any[] ?? []).map((c: any) => <option key={c.id} value={c.id}>{c.grade} {c.stream_name}</option>)}</Select></Field>
          <Field label="Subject"><Select value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })}><option value="">Select…</option>{(subjects as any[] ?? []).map((s: any) => <option key={s.id} value={s.id}>{s.name}</option>)}</Select></Field>
          <Field label="Week starting"><Input type="date" value={form.week_starting} onChange={(e) => setForm({ ...form, week_starting: e.target.value })} /></Field>
          <Field label="CBC strand"><Input value={form.strand} onChange={(e) => setForm({ ...form, strand: e.target.value })} placeholder="e.g. Numbers" /></Field>
          <Field label="Objectives">
            <textarea rows={2} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]" value={form.objectives} onChange={(e) => setForm({ ...form, objectives: e.target.value })} />
          </Field>
          <Field label="Activities">
            <textarea rows={2} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]" value={form.activities} onChange={(e) => setForm({ ...form, activities: e.target.value })} />
          </Field>
        </div>
        <Button className="mt-3" onClick={handleSave} disabled={save.isPending || !form.class_group || !form.week_starting}>Save plan</Button>
      </Card>
      <Table headers={["Week", "Class", "Subject", "Strand"]}>
        {(plans as any[] ?? []).map((p: any) => (
          <tr key={p.id} className="hover:bg-slate-50">
            <td className="px-4 py-3">{p.week_starting}</td>
            <td className="px-4 py-3">{p.class_group}</td>
            <td className="px-4 py-3">{p.subject}</td>
            <td className="px-4 py-3">{p.strand || "—"}</td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
