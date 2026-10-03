import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router-dom";
import { Download } from "lucide-react";
import { academicsApi } from "../../api/academics";
import { resultsApi } from "../../api/results";
import { teachersApi } from "../../api/teacher";
import { Badge, Card, Field, PageHeader, Select, Stat, Table } from "../../components/ui";

const fmt = (iso?: string | null) =>
  iso ? new Date(iso).toLocaleString("en-KE", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" }) : "—";

// Exam results for academic leads: every mark a teacher has uploaded (and
// who uploaded it), and the ranked class performance that report cards are
// built from.
export function ResultsOverviewPage() {
  const [tab, setTab] = useState<"marks" | "performance">("marks");
  const { data: terms } = useQuery({ queryKey: ["terms"], queryFn: () => academicsApi.terms() });
  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const current = ((terms as any[]) ?? []).find((t: any) => t.is_current);

  return (
    <div className="space-y-4 p-6">
      <PageHeader title="Results" subtitle="Marks uploaded by teachers, class rankings and report cards." />
      <div className="inline-flex rounded-lg border border-slate-200 bg-white p-1 text-sm">
        {([["marks", "Mark list"], ["performance", "Class performance"]] as const).map(([v, label]) => (
          <button
            key={v}
            type="button"
            onClick={() => setTab(v)}
            className={`rounded-md px-4 py-1.5 font-medium ${tab === v ? "bg-[#1b365d] text-white" : "text-slate-600 hover:bg-slate-50"}`}
          >
            {label}
          </button>
        ))}
      </div>
      {tab === "marks"
        ? <MarkList terms={(terms as any[]) ?? []} classes={(classes as any[]) ?? []} currentTermId={current?.id ?? ""} />
        : <Performance terms={(terms as any[]) ?? []} classes={(classes as any[]) ?? []} currentTermId={current?.id ?? ""} />}
    </div>
  );
}

function TermSelect({ terms, value, onChange }: { terms: any[]; value: string; onChange: (v: string) => void }) {
  return (
    <Field label="Term">
      <Select value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">Current term</option>
        {terms.map((t) => <option key={t.id} value={t.id}>{t.name} {t.year ?? ""}{t.is_current ? " (current)" : ""}</option>)}
      </Select>
    </Field>
  );
}

function MarkList({ terms, classes, currentTermId }: { terms: any[]; classes: any[]; currentTermId: string }) {
  const [f, setF] = useState({ class_id: "", subject_id: "", term_id: "", marked_by: "" });
  const { data: subjects } = useQuery({ queryKey: ["subjects"], queryFn: () => academicsApi.subjects() });
  const { data: teachers } = useQuery({ queryKey: ["teachers"], queryFn: () => teachersApi.list() });
  const params = Object.fromEntries(Object.entries(f).filter(([, v]) => v));
  const ready = !!(f.class_id || f.subject_id || f.marked_by);
  const { data: marks = [], isLoading } = useQuery({
    queryKey: ["marks", params],
    queryFn: () => academicsApi.marks(params) as Promise<any[]>,
    enabled: ready,
  });

  const avg = marks.length ? marks.reduce((s, m) => s + (m.percentage ?? 0), 0) / marks.length : 0;
  const uploaders = new Set(marks.map((m) => m.marked_by_id).filter(Boolean)).size;

  return (
    <div className="space-y-4">
      <Card className="p-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Class">
            <Select value={f.class_id} onChange={(e) => setF({ ...f, class_id: e.target.value })}>
              <option value="">All classes</option>
              {classes.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </Field>
          <Field label="Subject">
            <Select value={f.subject_id} onChange={(e) => setF({ ...f, subject_id: e.target.value })}>
              <option value="">All subjects</option>
              {((subjects as any[]) ?? []).map((s: any) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </Select>
          </Field>
          <Field label="Uploaded by">
            <Select value={f.marked_by} onChange={(e) => setF({ ...f, marked_by: e.target.value })}>
              <option value="">Any teacher</option>
              {((teachers as any[]) ?? []).map((t: any) => <option key={t.id} value={t.id}>{t.full_name}</option>)}
            </Select>
          </Field>
          <TermSelect terms={terms} value={f.term_id} onChange={(v) => setF({ ...f, term_id: v })} />
        </div>
      </Card>

      {!ready && (
        <Card className="p-8 text-center text-sm text-slate-400">Choose a class, subject or teacher to see the marks uploaded.</Card>
      )}

      {ready && (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <Stat label="Marks" value={marks.length} />
            <Stat label="Average" value={`${avg.toFixed(1)}%`} />
            <Stat label="Teachers who uploaded" value={uploaders} />
          </div>
          <Table headers={["Learner", "Class · Subject", "Assessment", "Score", "%", "Grade", "Comment", "Uploaded by"]}>
            {isLoading && <tr><td colSpan={8} className="px-4 py-8 text-center text-sm text-slate-400">Loading…</td></tr>}
            {!isLoading && marks.length === 0 && (
              <tr><td colSpan={8} className="px-4 py-8 text-center text-sm text-slate-400">
                No marks uploaded for these filters{!f.term_id && currentTermId ? " this term" : ""}.
              </td></tr>
            )}
            {marks.map((m: any) => (
              <tr key={m.id} className="hover:bg-slate-50">
                <td className="px-4 py-3">
                  <Link to={`/admin/students/${m.student_id}`} className="font-medium text-[#1b365d] hover:underline">{m.student_name}</Link>
                  <p className="text-xs text-slate-400">{m.admission_no}</p>
                </td>
                <td className="px-4 py-3">{m.class_name} · {m.subject_name}</td>
                <td className="px-4 py-3">{m.assessment_title}</td>
                <td className="px-4 py-3 tabular-nums">{m.score} / {m.total_marks}</td>
                <td className="px-4 py-3 tabular-nums">{m.percentage}</td>
                <td className="px-4 py-3">{m.grade_letter || m.competency_level || "—"}</td>
                <td className="px-4 py-3 text-xs text-slate-500">{m.comments || "—"}</td>
                <td className="px-4 py-3">
                  <p>{m.marked_by_name || "—"}</p>
                  <p className="text-xs text-slate-400">{fmt(m.marked_at)}</p>
                </td>
              </tr>
            ))}
          </Table>
        </>
      )}
    </div>
  );
}

function Performance({ terms, classes, currentTermId }: { terms: any[]; classes: any[]; currentTermId: string }) {
  const [classId, setClassId] = useState("");
  const [termId, setTermId] = useState("");
  const [err, setErr] = useState("");
  const term = termId || currentTermId;
  const { data, isLoading } = useQuery({
    queryKey: ["class-performance", classId, term],
    queryFn: () => academicsApi.classPerformance(classId, term),
    enabled: !!classId && !!term,
  });
  const students: any[] = data?.students ?? [];
  const subjects: any[] = data?.subjects ?? [];

  async function reportCard(s: any) {
    setErr("");
    try {
      await resultsApi.downloadReportCard(s.student_id, term, `report-card-${s.admission_no}-${data?.term_name ?? "term"}.pdf`);
    } catch {
      setErr(`Could not download ${s.full_name}'s report card.`);
    }
  }

  return (
    <div className="space-y-4">
      <Card className="p-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Class">
            <Select value={classId} onChange={(e) => setClassId(e.target.value)}>
              <option value="">Choose a class…</option>
              {classes.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </Field>
          <TermSelect terms={terms} value={termId} onChange={setTermId} />
        </div>
      </Card>

      {!classId && <Card className="p-8 text-center text-sm text-slate-400">Choose a class to see its ranking.</Card>}
      {classId && isLoading && <p className="text-sm text-slate-400">Loading…</p>}
      {err && <p className="text-sm text-red-600">{err}</p>}

      {classId && data && (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <Stat label="Learners with results" value={data.student_count} />
            <Stat label="Class mean" value={`${Number(data.class_mean).toFixed(1)}% ${data.class_mean_grade ?? ""}`} />
            <Stat label="Subjects examined" value={subjects.length} />
          </div>

          <Card className="p-4">
            <h3 className="mb-2 font-semibold text-[#1b365d]">Subject means</h3>
            <div className="flex flex-wrap gap-2">
              {subjects.map((s) => (
                <span key={s.subject_id} className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm">
                  {s.subject_name} <span className="font-semibold">{s.mean}%</span> <Badge>{s.grade}</Badge>
                </span>
              ))}
              {subjects.length === 0 && <p className="text-sm text-slate-400">No marks entered for this class and term.</p>}
            </div>
          </Card>

          <Table headers={["Pos.", "Learner", ...subjects.map((s) => s.subject_name), "Mean", "Grade", ""]}>
            {students.map((s) => {
              const bySubject = new Map((s.subjects ?? []).map((r: any) => [r.subject_id, r]));
              return (
                <tr key={s.student_id} className="hover:bg-slate-50">
                  <td className="px-4 py-3 font-semibold">{s.position}</td>
                  <td className="px-4 py-3">
                    <Link to={`/admin/students/${s.student_id}`} className="font-medium text-[#1b365d] hover:underline">{s.full_name}</Link>
                    <p className="text-xs text-slate-400">{s.admission_no}</p>
                  </td>
                  {subjects.map((sub) => {
                    const r: any = bySubject.get(sub.subject_id);
                    return <td key={sub.subject_id} className="px-4 py-3 tabular-nums">{r ? `${r.percentage}` : "—"}</td>;
                  })}
                  <td className="px-4 py-3 font-semibold tabular-nums">{s.mean}</td>
                  <td className="px-4 py-3">{s.mean_grade}</td>
                  <td className="px-4 py-3">
                    <button type="button" onClick={() => reportCard(s)} className="flex items-center gap-1 text-xs font-medium text-[#4a9eca] hover:underline">
                      <Download size={12} /> Report card
                    </button>
                  </td>
                </tr>
              );
            })}
          </Table>
        </>
      )}
    </div>
  );
}
