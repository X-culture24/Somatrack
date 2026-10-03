import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Fragment, useState } from "react";
import { academicsApi } from "../../api/academics";
import { teachersApi } from "../../api/teacher";
import { Badge, Button, Card, Field, PageHeader, Select, Stat, Table } from "../../components/ui";

const STATUS_BADGE: Record<string, "success" | "warning" | "danger" | "blue" | "default"> = {
  approved: "success", reviewed: "blue", submitted: "warning", returned: "danger", draft: "default",
};
const STATUS_LABEL: Record<string, string> = {
  approved: "approved", reviewed: "seen", submitted: "awaiting review", returned: "returned", draft: "draft",
};

// When the headteacher last opened this page, so plans teachers have
// written or changed since then can be flagged. Per-browser convenience only.
const SEEN_KEY = "lesson-plans-last-seen";
function readSeen(): string {
  try { return localStorage.getItem(SEEN_KEY) ?? ""; } catch { return ""; }
}
function writeSeen(v: string) {
  try { localStorage.setItem(SEEN_KEY, v); } catch { /* storage unavailable */ }
}

const fmt = (iso?: string | null) =>
  iso ? new Date(iso).toLocaleString("en-KE", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" }) : "—";

export function LessonPlanReviewPage() {
  const [tab, setTab] = useState<"plans" | "coverage">("plans");
  return (
    <div className="space-y-4 p-6">
      <PageHeader title="Lesson plans" subtitle="Review what teachers have planned, and see which subjects are behind." />
      <div className="inline-flex rounded-lg border border-slate-200 bg-white p-1 text-sm">
        {([["plans", "Plans to review"], ["coverage", "Coverage by subject"]] as const).map(([v, label]) => (
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
      {tab === "plans" ? <PlansTab /> : <CoverageTab />}
    </div>
  );
}

function PlansTab() {
  const qc = useQueryClient();
  const [filters, setFilters] = useState({ status: "submitted", class_id: "", teacher_id: "", term_id: "" });
  const [open, setOpen] = useState<string | null>(null);
  const [comment, setComment] = useState("");
  const [seen, setSeen] = useState(readSeen);

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: terms } = useQuery({ queryKey: ["terms"], queryFn: () => academicsApi.terms() });
  const { data: teachers } = useQuery({ queryKey: ["teachers"], queryFn: () => teachersApi.list() });
  const params = Object.fromEntries(Object.entries(filters).filter(([, v]) => v));
  const { data: plans = [], isLoading } = useQuery({
    queryKey: ["lesson-plans", params],
    queryFn: () => academicsApi.lessonPlans(params) as Promise<any[]>,
  });

  const review = useMutation({
    mutationFn: ({ id, status }: { id: string; status: "approved" | "reviewed" | "returned" }) =>
      academicsApi.reviewLessonPlan(id, status, comment),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["lesson-plans"] });
      qc.invalidateQueries({ queryKey: ["lesson-plan-coverage"] });
      setOpen(null);
      setComment("");
    },
    onError: (err: any) => {
      const d = err?.response?.data;
      window.alert(d?.fields?.comment ?? d?.message ?? "Could not save the review.");
    },
  });

  const isNew = (p: any) => !seen || new Date(p.updated_at) > new Date(seen);
  const newCount = plans.filter(isNew).length;

  function markAllSeen() {
    const now = new Date().toISOString();
    writeSeen(now);
    setSeen(now);
  }

  function toggle(p: any) {
    setOpen(open === p.id ? null : p.id);
    setComment(p.review_comment ?? "");
  }

  return (
    <div className="space-y-4">
      <Card className="p-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Status">
            <Select value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })}>
              <option value="submitted">Awaiting review</option>
              <option value="returned">Returned to teacher</option>
              <option value="approved">Approved</option>
              <option value="reviewed">Seen</option>
              <option value="draft">Drafts</option>
              <option value="">All</option>
            </Select>
          </Field>
          <Field label="Class">
            <Select value={filters.class_id} onChange={(e) => setFilters({ ...filters, class_id: e.target.value })}>
              <option value="">All classes</option>
              {((classes as any[]) ?? []).map((c: any) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </Field>
          <Field label="Teacher">
            <Select value={filters.teacher_id} onChange={(e) => setFilters({ ...filters, teacher_id: e.target.value })}>
              <option value="">All teachers</option>
              {((teachers as any[]) ?? []).map((t: any) => <option key={t.id} value={t.id}>{t.full_name}</option>)}
            </Select>
          </Field>
          <Field label="Term">
            <Select value={filters.term_id} onChange={(e) => setFilters({ ...filters, term_id: e.target.value })}>
              <option value="">All terms</option>
              {((terms as any[]) ?? []).map((t: any) => <option key={t.id} value={t.id}>{t.name} {t.year ?? ""}</option>)}
            </Select>
          </Field>
        </div>
        <div className="mt-3 flex flex-wrap items-center justify-between gap-2 text-sm">
          <p className="text-slate-500">
            {plans.length} plan{plans.length === 1 ? "" : "s"}, most recently updated first
            {newCount > 0 && <> · <span className="font-semibold text-[#1b6a9c]">{newCount} new or updated since you last looked</span></>}
          </p>
          {newCount > 0 && <Button variant="ghost" onClick={markAllSeen}>Mark all as seen</Button>}
        </div>
      </Card>

      <Table headers={["Updated", "Teacher", "Class · Subject", "Week", "Topic", "Status", ""]}>
        {isLoading && <tr><td colSpan={7} className="px-4 py-8 text-center text-sm text-slate-400">Loading…</td></tr>}
        {!isLoading && plans.length === 0 && (
          <tr><td colSpan={7} className="px-4 py-8 text-center text-sm text-slate-400">No lesson plans match these filters.</td></tr>
        )}
        {plans.map((p: any) => (
          <Fragment key={p.id}>
            <tr className="cursor-pointer hover:bg-slate-50" onClick={() => toggle(p)}>
              <td className="px-4 py-3 text-xs text-slate-500">
                {fmt(p.updated_at)}
                {isNew(p) && <span className="ml-1"><Badge variant="blue">new</Badge></span>}
              </td>
              <td className="px-4 py-3">{p.teacher_name}</td>
              <td className="px-4 py-3">{p.class_name} · {p.subject_name}</td>
              <td className="px-4 py-3">{p.week_number}</td>
              <td className="px-4 py-3 font-medium">{p.topic}</td>
              <td className="px-4 py-3"><Badge variant={STATUS_BADGE[p.status] ?? "default"}>{STATUS_LABEL[p.status] ?? p.status}</Badge></td>
              <td className="px-4 py-3 text-xs text-[#4a9eca]">{open === p.id ? "▲" : "▼"}</td>
            </tr>
            {open === p.id && (
              <tr>
                <td colSpan={7} className="bg-slate-50 px-4 py-4">
                  <p className="mb-3 text-xs text-slate-500">
                    Lesson date {p.lesson_date} · {p.term_name}
                    {p.submitted_at && <> · submitted {fmt(p.submitted_at)}</>}
                    {p.reviewed_at && <> · last reviewed {fmt(p.reviewed_at)}{p.reviewed_by_name ? ` by ${p.reviewed_by_name}` : ""}</>}
                  </p>
                  <dl className="grid gap-3 sm:grid-cols-2">
                    {[
                      ["Sub-topic", p.sub_topic], ["Objectives", p.learning_objectives],
                      ["Outcomes", p.learning_outcomes], ["Teaching methods", p.teaching_methods],
                      ["Materials", p.materials], ["Introduction", p.introduction],
                      ["Lesson development", p.lesson_development], ["Conclusion", p.conclusion],
                      ["Assessment", p.assessment], ["Homework", p.homework], ["Reflection", p.reflection],
                    ].filter(([, v]) => v).map(([label, value]) => (
                      <div key={label as string}>
                        <dt className="text-xs font-semibold uppercase tracking-wide text-slate-400">{label}</dt>
                        <dd className="whitespace-pre-line text-sm text-slate-700">{value}</dd>
                      </div>
                    ))}
                  </dl>
                  {p.status !== "draft" && (
                    <div className="mt-4 rounded-lg border border-slate-200 bg-white p-3">
                      <Field label="Comment to the teacher (required when returning)">
                        <textarea
                          rows={2}
                          className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]"
                          value={comment}
                          onChange={(e) => setComment(e.target.value)}
                        />
                      </Field>
                      <div className="mt-2 flex flex-wrap gap-2">
                        <Button onClick={() => review.mutate({ id: p.id, status: "approved" })} disabled={review.isPending}>Approve</Button>
                        <Button variant="ghost" onClick={() => review.mutate({ id: p.id, status: "reviewed" })} disabled={review.isPending}>Mark as seen</Button>
                        <Button
                          variant="ghost"
                          className="text-red-600"
                          onClick={() => review.mutate({ id: p.id, status: "returned" })}
                          disabled={review.isPending || !comment.trim()}
                        >
                          Return for changes
                        </Button>
                      </div>
                    </div>
                  )}
                </td>
              </tr>
            )}
          </Fragment>
        ))}
      </Table>
    </div>
  );
}

function CoverageTab() {
  const [classId, setClassId] = useState("");
  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const params: Record<string, string> = classId ? { class_id: classId } : {};
  const { data, isLoading } = useQuery({
    queryKey: ["lesson-plan-coverage", params],
    queryFn: () => academicsApi.lessonPlanCoverage(params),
  });
  const rows: any[] = data?.results ?? [];
  const sum = data?.summary ?? {};

  return (
    <div className="space-y-4">
      <Card className="p-4">
        <div className="max-w-xs">
          <Field label="Class">
            <Select value={classId} onChange={(e) => setClassId(e.target.value)}>
              <option value="">All classes</option>
              {((classes as any[]) ?? []).map((c: any) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </Field>
        </div>
      </Card>
      <div className="grid gap-3 sm:grid-cols-4">
        <Stat label="Subjects (this term)" value={sum.subjects ?? 0} />
        <Stat label="With no plans" value={sum.without_plans ?? 0} />
        <Stat label="With no teacher" value={sum.without_teacher ?? 0} />
        <Stat label="Awaiting your review" value={sum.awaiting_review ?? 0} />
      </div>
      <Table headers={["Class", "Subject", "Teacher", "Plans", "Approved", "Awaiting", "Returned", "Latest week", "Last updated"]}>
        {isLoading && <tr><td colSpan={9} className="px-4 py-8 text-center text-sm text-slate-400">Loading…</td></tr>}
        {rows.map((r) => (
          <tr key={`${r.class_id}${r.subject_id}`} className={r.plans === 0 ? "bg-amber-50/60" : ""}>
            <td className="px-4 py-3">{r.class_name}</td>
            <td className="px-4 py-3 font-medium">{r.subject_name}</td>
            <td className="px-4 py-3">{r.teacher_name || <span className="text-red-600">unassigned</span>}</td>
            <td className="px-4 py-3">{r.plans}</td>
            <td className="px-4 py-3">{r.approved}</td>
            <td className="px-4 py-3">{r.submitted > 0 ? <Badge variant="warning">{r.submitted}</Badge> : 0}</td>
            <td className="px-4 py-3">{r.returned}</td>
            <td className="px-4 py-3">{r.last_week ?? "—"}</td>
            <td className="px-4 py-3 text-xs text-slate-500">{fmt(r.last_updated)}</td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
