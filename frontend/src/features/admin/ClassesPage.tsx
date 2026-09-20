import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { schoolApi } from "../../api/school";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

const GRADES = [
  "Playgroup", "Pre-Primary 1", "Pre-Primary 2", "Grade 1", "Grade 2", "Grade 3",
  "Grade 4", "Grade 5", "Grade 6", "Grade 7", "Grade 8", "Grade 9",
];

export function ClassesPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: years } = useQuery({ queryKey: ["academic-years"], queryFn: academicsApi.years });
  const { data: streams } = useQuery({ queryKey: ["streams"], queryFn: academicsApi.streams });
  const { data: staff } = useQuery({ queryKey: ["staff"], queryFn: () => schoolApi.staff() });

  const [form, setForm] = useState({ academic_year: "", grade: "Grade 1", stream: "", capacity: "40" });
  const [editingId, setEditingId] = useState<number | null>(null);
  const [teacherChoice, setTeacherChoice] = useState("");

  const create = useMutation({
    mutationFn: () => academicsApi.createClass(form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["classes"] });
      setForm({ ...form, stream: "" });
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.non_field_errors?.[0] ?? "Could not create the class. It may already exist for this grade/stream/year.");
    },
  });
  const assignTeacher = useMutation({
    mutationFn: (id: number) => academicsApi.updateClass(id, { class_teacher: teacherChoice || null }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["classes"] });
      setEditingId(null);
    },
  });

  async function handleCreate() {
    if (!form.academic_year || !form.stream) return;
    const ok = await confirm({
      title: "Create class",
      message: `Create ${form.grade} — stream ${(streams as any[] ?? []).find((s: any) => String(s.id) === form.stream)?.name ?? ""}?`,
      confirmLabel: "Create class",
    });
    if (ok) create.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Classes & streams" subtitle="Class groups for the current academic year, with homeroom teacher assignment." />

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New class</h3>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Academic year">
            <Select value={form.academic_year} onChange={(e) => setForm({ ...form, academic_year: e.target.value })}>
              <option value="">Select year…</option>
              {(years as any[] ?? []).map((y: any) => <option key={y.id} value={y.id}>{y.name}</option>)}
            </Select>
          </Field>
          <Field label="Grade">
            <Select value={form.grade} onChange={(e) => setForm({ ...form, grade: e.target.value })}>
              {GRADES.map((g) => <option key={g}>{g}</option>)}
            </Select>
          </Field>
          <Field label="Stream">
            <Select value={form.stream} onChange={(e) => setForm({ ...form, stream: e.target.value })}>
              <option value="">Select stream…</option>
              {(streams as any[] ?? []).map((s: any) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </Select>
          </Field>
          <Field label="Capacity">
            <Input type="number" value={form.capacity} onChange={(e) => setForm({ ...form, capacity: e.target.value })} />
          </Field>
        </div>
        <Button className="mt-4" onClick={handleCreate} disabled={create.isPending || !form.academic_year || !form.stream}>
          {create.isPending ? "Creating…" : "Create class"}
        </Button>
      </Card>

      <Table headers={["Class", "Grade", "Stream", "Class teacher", "Learners", ""]}>
        {(data ?? []).map((c: any) => (
          <tr key={c.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 font-medium">{c.grade} {c.stream_name}</td>
            <td className="px-4 py-3">{c.grade}</td>
            <td className="px-4 py-3">{c.stream_name}</td>
            <td className="px-4 py-3">
              {editingId === c.id ? (
                <div className="flex items-center gap-2">
                  <Select className="w-44" value={teacherChoice} onChange={(e) => setTeacherChoice(e.target.value)}>
                    <option value="">Unassigned</option>
                    {(staff as any[] ?? []).map((s: any) => (
                      <option key={s.id} value={s.id}>{s.full_name}</option>
                    ))}
                  </Select>
                  <Button onClick={() => assignTeacher.mutate(c.id)} disabled={assignTeacher.isPending}>Save</Button>
                  <Button variant="ghost" onClick={() => setEditingId(null)}>Cancel</Button>
                </div>
              ) : (
                c.teacher_name || <span className="text-slate-400">Unassigned</span>
              )}
            </td>
            <td className="px-4 py-3">{c.student_count}</td>
            <td className="px-4 py-3">
              {editingId !== c.id && (
                <Button variant="ghost" onClick={() => { setEditingId(c.id); setTeacherChoice(String(c.class_teacher ?? "")); }}>
                  Assign teacher
                </Button>
              )}
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
