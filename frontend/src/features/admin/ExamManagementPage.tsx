import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Lock, LockOpen, Plus, Trash2, Pencil, ChevronDown, ChevronUp } from "lucide-react";
import { academicsApi } from "../../api/academics";
import { api } from "../../api/client";
import { Button, Card, Field, Input, PageHeader, Select } from "../../components/ui";

function unwrap(d: unknown) {
  if (d && typeof d === "object") {
    if ("results" in (d as Record<string, unknown>)) return (d as { results: unknown }).results;
    if ("data" in (d as Record<string, unknown>)) return (d as { data: unknown }).data;
  }
  return d;
}

const examApi = {
  list: (term_id?: string) =>
    api.get("/exams/threads/", { params: term_id ? { term_id } : {} }).then(r => unwrap(r.data) as ExamThread[]),
  create: (data: unknown) => api.post("/exams/threads/", data).then(r => r.data),
  update: (id: string, data: unknown) => api.patch(`/exams/threads/${id}/`, data).then(r => r.data),
  delete: (id: string) => api.delete(`/exams/threads/${id}/`).then(r => r.data),
  lock: (id: string, locked: boolean) => api.post(`/exams/threads/${id}/lock/`, { locked }).then(r => r.data),
};

interface ExamThread {
  id: string;
  name: string;
  term_id: string;
  term_name: string;
  curriculum: string;
  cats_weight: number;
  main_exam_weight: number;
  is_locked: boolean;
  description: string;
  assessment_count: number;
  students_with_marks: number;
}

const CURR_LABELS: Record<string, string> = { cbc: "CBC", "844_igcse": "8-4-4 / IGCSE" };

export function ExamManagementPage() {
  const [termId, setTermId] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [editing, setEditing] = useState<ExamThread | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [form, setForm] = useState({ name: "", term_id: "", curriculum: "cbc", cats_weight: 0, main_exam_weight: 100, description: "" });

  const { data: terms } = useQuery({ queryKey: ["terms"], queryFn: () => academicsApi.terms() });
  const { data: threads = [], isLoading } = useQuery({
    queryKey: ["exam-threads", termId],
    queryFn: () => examApi.list(termId || undefined),
  });

  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: ["exam-threads"] });

  const create = useMutation({ mutationFn: (d: unknown) => examApi.create(d), onSuccess: () => { invalidate(); setShowCreate(false); resetForm(); } });
  const update = useMutation({ mutationFn: ({ id, d }: { id: string; d: unknown }) => examApi.update(id, d), onSuccess: () => { invalidate(); setEditing(null); } });
  const remove = useMutation({ mutationFn: (id: string) => examApi.delete(id), onSuccess: invalidate });
  const lock = useMutation({ mutationFn: ({ id, locked }: { id: string; locked: boolean }) => examApi.lock(id, locked), onSuccess: invalidate });

  function resetForm() { setForm({ name: "", term_id: "", curriculum: "cbc", cats_weight: 0, main_exam_weight: 100, description: "" }); }

  function startEdit(t: ExamThread) {
    setEditing(t);
    setForm({ name: t.name, term_id: t.term_id, curriculum: t.curriculum, cats_weight: t.cats_weight, main_exam_weight: t.main_exam_weight, description: t.description });
  }

  function handleSubmit() {
    if (editing) {
      update.mutate({ id: editing.id, d: { name: form.name, cats_weight: form.cats_weight, main_exam_weight: form.main_exam_weight, description: form.description } });
    } else {
      create.mutate(form);
    }
  }

  const termList = (terms as { id: string; name: string; year?: number }[] | undefined) ?? [];
  const locked = threads.filter(t => t.is_locked).length;

  return (
    <div className="p-6 space-y-4">
      <PageHeader title="Exam Management" subtitle="Create exam threads, track mark entry, lock results" />

      {/* Filter + stats */}
      <div className="flex flex-wrap gap-3 items-end">
        <div className="w-56">
          <label className="block text-xs font-medium mb-1">Filter by term</label>
          <Select value={termId} onChange={e => setTermId(e.target.value)}>
            <option value="">All terms</option>
            {termList.map(t => <option key={t.id} value={t.id}>{t.name} {t.year}</option>)}
          </Select>
        </div>
        <Button onClick={() => { setShowCreate(true); setEditing(null); resetForm(); }}>
          <Plus size={14} className="mr-1" /> New Exam Thread
        </Button>
        {threads.length > 0 && (
          <div className="flex gap-3 ml-auto text-sm">
            <span className="px-3 py-1 bg-green-50 border border-green-200 rounded text-green-700 font-medium">{locked} locked</span>
            <span className="px-3 py-1 bg-amber-50 border border-amber-200 rounded text-amber-700 font-medium">{threads.length - locked} open</span>
          </div>
        )}
      </div>

      {/* Create / Edit form */}
      {(showCreate || editing) && (
        <Card className="p-4 border-blue-200 bg-blue-50/30">
          <h3 className="font-semibold mb-3">{editing ? "Edit Exam Thread" : "New Exam Thread"}</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <Field label="Name">
              <Input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="e.g. Term 1 End of Term Exam" />
            </Field>
            {!editing && (
              <Field label="Term">
                <Select value={form.term_id} onChange={e => setForm(f => ({ ...f, term_id: e.target.value }))}>
                  <option value="">Select term…</option>
                  {termList.map(t => <option key={t.id} value={t.id}>{t.name} {t.year}</option>)}
                </Select>
              </Field>
            )}
            {!editing && (
              <Field label="Curriculum">
                <Select value={form.curriculum} onChange={e => setForm(f => ({ ...f, curriculum: e.target.value }))}>
                  <option value="cbc">CBC</option>
                  <option value="844_igcse">8-4-4 / IGCSE</option>
                </Select>
              </Field>
            )}
            <Field label="CATs weight (%)">
              <Input type="number" min={0} max={100} value={form.cats_weight}
                onChange={e => { const v = +e.target.value; setForm(f => ({ ...f, cats_weight: v, main_exam_weight: 100 - v })); }} />
            </Field>
            <Field label="Main exam weight (%)">
              <Input type="number" min={0} max={100} value={form.main_exam_weight}
                onChange={e => { const v = +e.target.value; setForm(f => ({ ...f, main_exam_weight: v, cats_weight: 100 - v })); }} />
            </Field>
            <Field label="Description (optional)" className="md:col-span-2">
              <Input value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} />
            </Field>
          </div>
          {(form.cats_weight + form.main_exam_weight) !== 100 && (
            <p className="text-xs text-red-500 mt-1">Weights must add up to 100 (currently {form.cats_weight + form.main_exam_weight})</p>
          )}
          <div className="flex gap-2 mt-3">
            <Button onClick={handleSubmit} disabled={!form.name || (form.cats_weight + form.main_exam_weight) !== 100 || create.isPending || update.isPending}>
              {create.isPending || update.isPending ? "Saving…" : editing ? "Update" : "Create"}
            </Button>
            <Button variant="outline" onClick={() => { setShowCreate(false); setEditing(null); }}>Cancel</Button>
          </div>
        </Card>
      )}

      {isLoading && <p className="text-gray-500 text-sm">Loading…</p>}
      {!isLoading && threads.length === 0 && (
        <Card className="p-8 text-center text-gray-400">No exam threads yet. Create one above.</Card>
      )}

      <div className="space-y-3">
        {(threads as ExamThread[]).map(t => (
          <Card key={t.id} className={`overflow-hidden ${t.is_locked ? "border-green-200" : ""}`}>
            <div className="p-4 flex items-start justify-between gap-3">
              <div className="flex items-start gap-3">
                {t.is_locked
                  ? <Lock size={16} className="text-green-600 mt-0.5 shrink-0" />
                  : <LockOpen size={16} className="text-gray-400 mt-0.5 shrink-0" />
                }
                <div>
                  <p className="font-semibold">{t.name}</p>
                  <p className="text-xs text-gray-500 mt-0.5">
                    {t.term_name} · {CURR_LABELS[t.curriculum] ?? t.curriculum}
                    {" · "}CATs {t.cats_weight}% / Main {t.main_exam_weight}%
                  </p>
                  {t.description && <p className="text-xs text-gray-400 mt-0.5">{t.description}</p>}
                  <div className="flex gap-3 mt-1 text-xs">
                    <span className="text-blue-600">{t.assessment_count} assessments</span>
                    <span className="text-purple-600">{t.students_with_marks} students with marks</span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <Button
                  size="sm"
                  variant={t.is_locked ? "outline" : "default"}
                  onClick={() => lock.mutate({ id: t.id, locked: !t.is_locked })}
                  disabled={lock.isPending}
                  title={t.is_locked ? "Unlock results" : "Lock results"}
                >
                  {t.is_locked ? <><LockOpen size={13} className="mr-1" />Unlock</> : <><Lock size={13} className="mr-1" />Lock</>}
                </Button>
                <Button size="sm" variant="outline" onClick={() => startEdit(t)} title="Edit">
                  <Pencil size={13} />
                </Button>
                {!t.is_locked && t.assessment_count === 0 && (
                  <Button size="sm" variant="outline" onClick={() => { if (confirm("Delete this exam thread?")) remove.mutate(t.id); }} title="Delete">
                    <Trash2 size={13} className="text-red-400" />
                  </Button>
                )}
                <button className="text-gray-400 hover:text-gray-600 p-1" onClick={() => setExpanded(expanded === t.id ? null : t.id)}>
                  {expanded === t.id ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                </button>
              </div>
            </div>

            {expanded === t.id && (
              <div className="border-t bg-gray-50 px-4 py-3 text-sm text-gray-600">
                <p className="font-medium mb-1">Thread ID: <span className="font-mono text-xs">{t.id}</span></p>
                <p>Use this thread when creating assessments in the Marks Entry page to group all exams together.</p>
                {t.is_locked && (
                  <p className="mt-2 text-green-700 font-medium">🔒 Results are locked. Teachers cannot modify marks.</p>
                )}
                {!t.is_locked && t.assessment_count > 0 && (
                  <p className="mt-2 text-amber-600">Mark entry is still open. Lock when all teachers have submitted.</p>
                )}
              </div>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}
