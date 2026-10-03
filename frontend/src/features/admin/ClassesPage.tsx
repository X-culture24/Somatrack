import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { GRADE_LABELS } from "../../api/finance";
import { Button, Card, Field, Input, PageHeader, Select, kes, useConfirm } from "../../components/ui";

// class_groups.grade uses the same numbering as fee_structures (Playgroup=-2,
// PP1=-1, PP2=0, Grade N=N) — GRADE_LABELS is the single source of truth,
// shared with Fee Structures and Admissions.
const GRADE_LEVELS = Object.entries(GRADE_LABELS).map(([value, label]) => ({ value: Number(value), label }));

function gradeLabel(grade: number) {
  return GRADE_LABELS[grade] ?? `Grade level ${grade}`;
}

export function ClassesPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: years } = useQuery({ queryKey: ["academic-years"], queryFn: academicsApi.years });
  const { data: streams } = useQuery({ queryKey: ["streams"], queryFn: academicsApi.streams });

  const [form, setForm] = useState({ academic_year_id: "", grade: 4, stream_id: "", capacity: "40" });

  const create = useMutation({
    mutationFn: () => {
      const stream = (streams as any[] ?? []).find((s: any) => String(s.id) === form.stream_id);
      const name = `${gradeLabel(form.grade)}${stream ? ` ${stream.name}` : ""}`;
      return academicsApi.createClass({
        name,
        grade: form.grade,
        stream_id: form.stream_id || null,
        academic_year_id: form.academic_year_id,
        capacity: Number(form.capacity) || 40,
      });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["classes"] });
      setForm({ ...form, stream_id: "" });
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not create the class. It may already exist for this grade/stream/year.");
    },
  });

  async function handleCreate() {
    if (!form.academic_year_id) return;
    const stream = (streams as any[] ?? []).find((s: any) => String(s.id) === form.stream_id);
    const ok = await confirm({
      title: "Create class",
      message: `Create ${gradeLabel(form.grade)}${stream ? ` — stream ${stream.name}` : ""}?`,
      confirmLabel: "Create class",
    });
    if (ok) create.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Classes & streams" subtitle="Class groups for the current academic year." />

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New class</h3>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Academic year">
            <Select value={form.academic_year_id} onChange={(e) => setForm({ ...form, academic_year_id: e.target.value })}>
              <option value="">Select year…</option>
              {(years as any[] ?? []).map((y: any) => <option key={y.id} value={y.id}>{y.name}</option>)}
            </Select>
          </Field>
          <Field label="Grade">
            <Select value={form.grade} onChange={(e) => setForm({ ...form, grade: Number(e.target.value) })}>
              {GRADE_LEVELS.map((g) => <option key={g.value} value={g.value}>{g.label}</option>)}
            </Select>
          </Field>
          <Field label="Stream">
            <Select value={form.stream_id} onChange={(e) => setForm({ ...form, stream_id: e.target.value })}>
              <option value="">No stream</option>
              {(streams as any[] ?? []).map((s: any) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </Select>
          </Field>
          <Field label="Capacity">
            <Input type="number" value={form.capacity} onChange={(e) => setForm({ ...form, capacity: e.target.value })} />
          </Field>
        </div>
        <Button className="mt-4" onClick={handleCreate} disabled={create.isPending || !form.academic_year_id}>
          {create.isPending ? "Creating…" : "Create class"}
        </Button>
      </Card>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {(data ?? []).length === 0 && (
          <p className="col-span-full py-8 text-center text-sm text-slate-400">No classes yet.</p>
        )}
        {(data ?? []).map((c: any) => {
          const outstanding = Number(c.total_outstanding ?? 0);
          const fillPct = c.capacity > 0 ? Math.min(100, Math.round((c.student_count / c.capacity) * 100)) : 0;
          return (
            <Card key={c.id}>
              <div className="flex items-start justify-between">
                <div>
                  <p className="font-semibold text-[#1b365d]">{c.name}</p>
                  <p className="text-xs text-slate-500">{gradeLabel(c.grade)}{c.stream_name ? ` · ${c.stream_name}` : ""}</p>
                </div>
                <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
                  {c.student_count}/{c.capacity}
                </span>
              </div>

              <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-slate-100">
                <div className="h-full rounded-full bg-[#4a9eca]" style={{ width: `${fillPct}%` }} />
              </div>

              <dl className="mt-4 grid grid-cols-2 gap-2 text-xs">
                <div>
                  <dt className="text-slate-400">Billed</dt>
                  <dd className="font-medium text-slate-700">{kes(c.total_billed)}</dd>
                </div>
                <div>
                  <dt className="text-slate-400">Collected</dt>
                  <dd className="font-medium text-green-600">{kes(c.total_paid)}</dd>
                </div>
                <div className="col-span-2 border-t border-slate-100 pt-2">
                  <dt className="text-slate-400">Outstanding</dt>
                  <dd className={`text-sm font-semibold ${outstanding > 0 ? "text-red-600" : "text-green-600"}`}>
                    {kes(Math.abs(outstanding))}{outstanding < 0 ? " credit" : ""}
                  </dd>
                </div>
              </dl>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
