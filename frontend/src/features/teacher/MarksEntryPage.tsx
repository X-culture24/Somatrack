import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { Lock, Plus } from "lucide-react";
import { academicsApi } from "../../api/academics";
import { resourcesApi, formErrors } from "../../api/resources";
import { ASSESSMENT_TYPES, COMPETENCY, resultsApi, type MarkRow } from "../../api/results";
import { Button, Card, Field, Input, PageHeader, Select, Table } from "../../components/ui";

// Marks entry: pick a class and an assessment, fill the marksheet, save it in
// one go. The server grades each mark (school bands, else CBC EE/ME/AE/BE),
// and refuses changes once the exam thread is locked.

type Draft = { score: string; competency_level: string; comments: string };

export function MarksEntryPage() {
  const [classId, setClassId] = useState("");
  const [assessmentId, setAssessmentId] = useState("");
  const [creating, setCreating] = useState(false);

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: assessments } = useQuery({
    queryKey: ["assessments", classId],
    enabled: Boolean(classId),
    queryFn: () => resultsApi.assessments({ class_id: classId }),
  });

  return (
    <div className="space-y-4">
      <PageHeader title="Marks entry" subtitle="Record CATs and exams. Results feed report cards and each learner's results trail." />
      <Card>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Class">
            <Select value={classId} onChange={(e) => { setClassId(e.target.value); setAssessmentId(""); setCreating(false); }}>
              <option value="">Select class…</option>
              {(classes as any[] ?? []).map((c: any) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </Field>
          <Field label="Assessment (current term)">
            <div className="flex gap-2">
              <Select value={assessmentId} onChange={(e) => setAssessmentId(e.target.value)} disabled={!classId}>
                <option value="">{assessments?.length === 0 ? "None yet — create one" : "Select assessment…"}</option>
                {(assessments ?? []).map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.subject_name} — {a.name} (/{a.total_marks}){a.is_locked ? " 🔒" : ""}
                  </option>
                ))}
              </Select>
              <Button variant="ghost" disabled={!classId} onClick={() => setCreating((v) => !v)} title="New assessment"><Plus size={16} /></Button>
            </div>
          </Field>
        </div>
        {creating && classId && (
          <NewAssessment classId={classId} onCreated={(id) => { setCreating(false); setAssessmentId(id); }} />
        )}
      </Card>

      {assessmentId && <Marksheet key={assessmentId} assessmentId={assessmentId} />}
    </div>
  );
}

function NewAssessment({ classId, onCreated }: { classId: string; onCreated: (id: string) => void }) {
  const qc = useQueryClient();
  const [form, setForm] = useState({ name: "", assessment_type: "cat1", subject_id: "", total_marks: "100", exam_thread_id: "", assessment_date: "" });
  const [err, setErr] = useState<{ message: string; fields: Record<string, string> } | null>(null);
  const { data: subjects } = useQuery({ queryKey: ["subjects"], queryFn: () => academicsApi.subjects() });
  const { data: threads } = useQuery({
    queryKey: ["exam-threads-open"],
    queryFn: () => resourcesApi.rows("exam-threads", { is_locked: "false", page_size: "100" }),
  });

  const create = useMutation({
    mutationFn: () => resultsApi.createAssessment({
      name: form.name, assessment_type: form.assessment_type, class_id: classId, subject_id: form.subject_id,
      total_marks: Number(form.total_marks),
      ...(form.exam_thread_id ? { exam_thread_id: form.exam_thread_id } : {}),
      ...(form.assessment_date ? { assessment_date: form.assessment_date } : {}),
    }),
    onSuccess: (a) => { qc.invalidateQueries({ queryKey: ["assessments", classId] }); onCreated(a.id); },
    onError: (e) => setErr(formErrors(e)),
  });
  const set = (k: keyof typeof form) => (e: { target: { value: string } }) => setForm({ ...form, [k]: e.target.value });

  return (
    <form className="mt-4 grid gap-3 border-t border-slate-100 pt-4 sm:grid-cols-3" onSubmit={(e) => { e.preventDefault(); setErr(null); create.mutate(); }}>
      <Field label="Name *"><Input required value={form.name} onChange={set("name")} placeholder="e.g. Maths CAT 1" /></Field>
      <Field label="Subject *">
        <Select required value={form.subject_id} onChange={set("subject_id")}>
          <option value="">Choose…</option>
          {(subjects as any[] ?? []).map((s: any) => <option key={s.id} value={s.id}>{s.name}</option>)}
        </Select>
      </Field>
      <Field label="Type">
        <Select value={form.assessment_type} onChange={set("assessment_type")}>
          {ASSESSMENT_TYPES.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
        </Select>
      </Field>
      <Field label="Out of"><Input type="number" min="1" max="1000" value={form.total_marks} onChange={set("total_marks")} /></Field>
      <Field label="Exam thread">
        <Select value={form.exam_thread_id} onChange={set("exam_thread_id")}>
          <option value="">None</option>
          {(threads?.results ?? []).map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
        </Select>
      </Field>
      <Field label="Date"><Input type="date" value={form.assessment_date} onChange={set("assessment_date")} /></Field>
      {err && (
        <p className="text-sm text-red-700 sm:col-span-3">
          {Object.keys(err.fields).length ? Object.entries(err.fields).map(([k, v]) => `${k.replace(/_id$/, "")}: ${v}`).join(" · ") : err.message}
        </p>
      )}
      <div className="sm:col-span-3"><Button type="submit" disabled={create.isPending}>{create.isPending ? "Creating…" : "Create assessment"}</Button></div>
    </form>
  );
}

function Marksheet({ assessmentId }: { assessmentId: string }) {
  const qc = useQueryClient();
  const { data, isLoading, isError } = useQuery({
    queryKey: ["marksheet", assessmentId],
    queryFn: () => resultsApi.marksheet(assessmentId),
  });
  const [drafts, setDrafts] = useState<Record<string, Draft>>({});
  const [errors, setErrors] = useState<{ message: string; fields: Record<string, string> } | null>(null);
  const [saved, setSaved] = useState("");

  useEffect(() => {
    if (!data) return;
    const d: Record<string, Draft> = {};
    for (const r of data.results) {
      d[r.student_id] = { score: r.score === null ? "" : String(r.score), competency_level: r.competency_level, comments: r.comments };
    }
    setDrafts(d);
  }, [data]);

  const a = data?.assessment;
  const locked = Boolean(a?.is_locked);

  const changed = (r: MarkRow) => {
    const d = drafts[r.student_id];
    if (!d) return false;
    return d.score !== (r.score === null ? "" : String(r.score)) || d.competency_level !== r.competency_level || d.comments !== r.comments;
  };

  const save = useMutation({
    mutationFn: () => resultsApi.saveMarks(assessmentId, (data?.results ?? []).filter(changed).map((r) => {
      const d = drafts[r.student_id];
      return { student_id: r.student_id, score: d.score === "" ? null : Number(d.score), competency_level: d.competency_level, comments: d.comments };
    })),
    onSuccess: (res) => {
      setSaved(`Saved ${res.saved} mark${res.saved === 1 ? "" : "s"}${res.cleared ? `, cleared ${res.cleared}` : ""}.`);
      qc.invalidateQueries({ queryKey: ["marksheet", assessmentId] });
      qc.invalidateQueries({ queryKey: ["assessments"] });
    },
    onError: (e) => setErrors(formErrors(e)),
  });

  if (isLoading) return <Card><p className="text-sm text-slate-500">Loading marksheet…</p></Card>;
  if (isError || !data || !a) return <Card><p className="text-sm text-red-700">Could not load this marksheet.</p></Card>;

  const dirty = data.results.filter(changed).length;
  const setDraft = (id: string, patch: Partial<Draft>) => { setSaved(""); setDrafts((s) => ({ ...s, [id]: { ...s[id], ...patch } })); };

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-slate-600">
          <span className="font-semibold text-[#1b365d]">{a.subject_name} — {a.name}</span> · {a.class_name} · {a.term_name} · out of {a.total_marks}
          {a.exam_thread_name && ` · ${a.exam_thread_name}`}
        </p>
        {locked && <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-3 py-1 text-xs font-medium text-amber-800"><Lock size={12} /> Locked — marks are final</span>}
      </div>
      {errors && <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">{errors.message}</p>}

      <Table headers={["Learner", `Score /${a.total_marks}`, "%", "Competency", "Comment"]}>
        {data.results.length === 0 && (
          <tr><td colSpan={5} className="px-4 py-8 text-center text-slate-500">No learners in this class.</td></tr>
        )}
        {data.results.map((r) => {
          const d = drafts[r.student_id] ?? { score: "", competency_level: "", comments: "" };
          const pct = d.score === "" ? null : (Number(d.score) / a.total_marks) * 100;
          const bad = errors?.fields[r.student_id];
          return (
            <tr key={r.student_id} className={changed(r) ? "bg-amber-50/50" : "hover:bg-slate-50"}>
              <td className="px-4 py-2">
                <p className="font-medium">{r.full_name}</p>
                <p className="text-xs text-slate-400">{r.admission_no}</p>
                {bad && <p className="text-xs text-red-600">{bad}</p>}
              </td>
              <td className="w-28 px-4 py-2">
                <Input type="number" min="0" max={a.total_marks} step="0.5" disabled={locked} value={d.score}
                  onChange={(e) => setDraft(r.student_id, { score: e.target.value })} placeholder="—"
                  className={bad ? "border-red-400" : ""} />
              </td>
              <td className="w-20 px-4 py-2 tabular-nums text-slate-600">{pct === null ? "—" : `${pct.toFixed(1)}%`}</td>
              <td className="w-52 px-4 py-2">
                <Select disabled={locked} value={d.competency_level} onChange={(e) => setDraft(r.student_id, { competency_level: e.target.value })}>
                  <option value="">{pct === null ? "—" : "From score"}</option>
                  {COMPETENCY.map(([v, l]) => <option key={v} value={v}>{v} — {l}</option>)}
                </Select>
              </td>
              <td className="px-4 py-2">
                <Input disabled={locked} value={d.comments} onChange={(e) => setDraft(r.student_id, { comments: e.target.value })} placeholder="Optional" />
              </td>
            </tr>
          );
        })}
      </Table>

      {!locked && (
        <div className="flex items-center gap-3">
          <Button onClick={() => { setErrors(null); save.mutate(); }} disabled={save.isPending || dirty === 0}>
            {save.isPending ? "Saving…" : dirty ? `Save ${dirty} change${dirty === 1 ? "" : "s"}` : "No changes"}
          </Button>
          {saved && <span className="text-sm text-green-700">{saved}</span>}
        </div>
      )}
    </div>
  );
}
