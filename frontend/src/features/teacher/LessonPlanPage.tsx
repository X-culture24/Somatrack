import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { teacherApi, teachersApi } from "../../api/teacher";
import { useAuth } from "../../store/auth";
import { Badge, Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

const emptyForm = {
  class_id: "", subject_id: "", week_number: "1", lesson_date: "",
  topic: "", sub_topic: "", learning_objectives: "", learning_outcomes: "",
  teaching_methods: "", materials: "", introduction: "", lesson_development: "",
  conclusion: "", assessment: "", homework: "", reflection: "",
};

const STATUS_BADGE: Record<string, "success" | "warning" | "danger" | "blue" | "default"> = {
  approved: "success", reviewed: "blue", submitted: "warning", returned: "danger", draft: "default",
};
const STATUS_LABEL: Record<string, string> = {
  approved: "approved", reviewed: "seen by HT", submitted: "awaiting review", returned: "returned — needs changes", draft: "draft",
};

export function LessonPlanPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [form, setForm] = useState(emptyForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  const me = useAuth((s) => s.user);

  // Scoped to classes this teacher actually teaches — not every class in the
  // school — since class_groups.class_teacher_id / class_subjects.teacher_id
  // now record real assignments (see Staff/Classes pages to assign one).
  const { data: myClasses } = useQuery({ queryKey: ["teacher-my-classes"], queryFn: teacherApi.myClasses });
  const { data: subjects } = useQuery({ queryKey: ["subjects"], queryFn: () => academicsApi.subjects() });
  const { data: plans } = useQuery({ queryKey: ["lesson-plans"], queryFn: () => academicsApi.lessonPlans() });
  // The subjects this teacher is assigned in each class; a homeroom teacher
  // may plan any subject for their own class.
  const { data: myRecord } = useQuery({
    queryKey: ["teacher", String(me?.id)],
    queryFn: () => teachersApi.get(String(me!.id)),
    enabled: !!me?.id,
  });

  const save = useMutation({
    mutationFn: (status: "draft" | "submitted") => {
      const data = { ...form, week_number: Number(form.week_number) || 1, status };
      return editingId ? academicsApi.updateLessonPlan(editingId, data) : academicsApi.createLessonPlan(data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["lesson-plans"] });
      setForm(emptyForm);
      setEditingId(null);
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not save this lesson plan.");
    },
  });

  async function handleSave(status: "draft" | "submitted") {
    const ok = await confirm({
      title: status === "draft" ? "Save draft" : "Submit to headteacher",
      message: status === "draft"
        ? `Save "${form.topic}" as a draft? The headteacher won't review it until you submit.`
        : `Submit "${form.topic}" (${form.lesson_date}) to the headteacher for review?`,
      confirmLabel: status === "draft" ? "Save draft" : "Submit",
    });
    if (ok) save.mutate(status);
  }

  function startEdit(p: any) {
    setEditingId(p.id);
    setForm(Object.fromEntries(Object.keys(emptyForm).map((k) => [k, String(p[k] ?? "")])) as typeof emptyForm);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  const classes = (myClasses as any[]) ?? [];
  const chosenClass = classes.find((c: any) => c.id === form.class_id);
  const assigned: any[] = (myRecord as any)?.assignments ?? [];
  const subjectOptions = chosenClass?.is_homeroom
    ? ((subjects as any[]) ?? []).map((s: any) => ({ id: s.id, name: s.name }))
    : assigned.filter((a) => a.class_id === form.class_id).map((a) => ({ id: a.subject_id, name: a.subject_name }));
  const returned = ((plans as any[]) ?? []).filter((p: any) => p.status === "returned");

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Lesson plans" subtitle="CBC scheme of work — one plan per lesson, for the classes you're assigned to." />

      {classes.length === 0 && (
        <Card className="border-2 border-amber-200 bg-amber-50">
          <p className="text-sm text-amber-800">
            You aren't assigned as homeroom or subject teacher for any class yet — ask an admin to assign you on the
            Classes or Staff page before you can plan a lesson.
          </p>
        </Card>
      )}

      {returned.length > 0 && (
        <Card className="border-2 border-red-200 bg-red-50">
          <p className="mb-2 text-sm font-semibold text-red-800">
            {returned.length} plan{returned.length === 1 ? " was" : "s were"} returned by the headteacher
          </p>
          <ul className="space-y-1 text-sm text-red-800">
            {returned.map((p: any) => (
              <li key={p.id} className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{p.subject_name} · {p.class_name} · wk {p.week_number}: {p.topic}</span>
                <span className="italic">“{p.review_comment}”</span>
                <button type="button" className="text-xs font-semibold text-[#1b365d] underline" onClick={() => startEdit(p)}>Edit &amp; resubmit</button>
              </li>
            ))}
          </ul>
        </Card>
      )}

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-semibold text-[#1b365d]">{editingId ? "Edit lesson plan" : "New lesson plan"}</h3>
          {editingId && <Button variant="ghost" onClick={() => { setEditingId(null); setForm(emptyForm); }}>Cancel edit</Button>}
        </div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Lesson</p>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Class">
            <Select value={form.class_id} onChange={(e) => setForm({ ...form, class_id: e.target.value, subject_id: "" })}>
              <option value="">Select…</option>
              {classes.map((c: any) => <option key={c.id} value={c.id}>{c.name}{c.is_homeroom ? " (homeroom)" : ""}</option>)}
            </Select>
          </Field>
          <Field label="Subject">
            <Select value={form.subject_id} onChange={(e) => setForm({ ...form, subject_id: e.target.value })}>
              <option value="">Select…</option>
              {subjectOptions.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </Select>
          </Field>
          <Field label="Week no."><Input type="number" min="1" value={form.week_number} onChange={(e) => setForm({ ...form, week_number: e.target.value })} /></Field>
          <Field label="Lesson date"><Input type="date" value={form.lesson_date} onChange={(e) => setForm({ ...form, lesson_date: e.target.value })} /></Field>
          <Field label="Topic"><Input value={form.topic} onChange={(e) => setForm({ ...form, topic: e.target.value })} /></Field>
          <Field label="Sub-topic"><Input value={form.sub_topic} onChange={(e) => setForm({ ...form, sub_topic: e.target.value })} /></Field>
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Planning</p>
        <div className="grid gap-3 sm:grid-cols-2">
          {[
            ["learning_objectives", "Learning objectives"],
            ["learning_outcomes", "Learning outcomes"],
            ["teaching_methods", "Teaching methods"],
            ["materials", "Materials / resources"],
          ].map(([key, label]) => (
            <Field key={key} label={label}>
              <textarea
                rows={2}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]"
                value={(form as any)[key]}
                onChange={(e) => setForm({ ...form, [key]: e.target.value })}
              />
            </Field>
          ))}
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Lesson flow</p>
        <div className="grid gap-3 sm:grid-cols-3">
          {[
            ["introduction", "Introduction"],
            ["lesson_development", "Lesson development"],
            ["conclusion", "Conclusion"],
          ].map(([key, label]) => (
            <Field key={key} label={label}>
              <textarea
                rows={3}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]"
                value={(form as any)[key]}
                onChange={(e) => setForm({ ...form, [key]: e.target.value })}
              />
            </Field>
          ))}
        </div>

        <p className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wide text-slate-400">Follow-up</p>
        <div className="grid gap-3 sm:grid-cols-3">
          {[
            ["assessment", "Assessment"],
            ["homework", "Homework"],
            ["reflection", "Reflection (fill in after teaching)"],
          ].map(([key, label]) => (
            <Field key={key} label={label}>
              <textarea
                rows={2}
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]"
                value={(form as any)[key]}
                onChange={(e) => setForm({ ...form, [key]: e.target.value })}
              />
            </Field>
          ))}
        </div>

        <div className="mt-4 flex gap-2">
          <Button
            onClick={() => handleSave("submitted")}
            disabled={save.isPending || !form.class_id || !form.subject_id || !form.lesson_date || !form.topic}
          >
            {save.isPending ? "Saving…" : editingId ? "Save & resubmit" : "Submit to headteacher"}
          </Button>
          <Button
            variant="ghost"
            onClick={() => handleSave("draft")}
            disabled={save.isPending || !form.class_id || !form.subject_id || !form.lesson_date || !form.topic}
          >
            Save as draft
          </Button>
        </div>
      </Card>

      <Table headers={["Date", "Week", "Class", "Subject", "Topic", "Status", ""]}>
        {(plans as any[] ?? []).length === 0 && (
          <tr><td colSpan={7} className="px-4 py-8 text-center text-sm text-slate-400">No lesson plans yet.</td></tr>
        )}
        {(plans as any[] ?? []).map((p: any) => (
          <>
            <tr key={p.id} className="cursor-pointer hover:bg-slate-50" onClick={() => setExpanded(expanded === p.id ? null : p.id)}>
              <td className="px-4 py-3 text-xs text-slate-500">{p.lesson_date}</td>
              <td className="px-4 py-3">{p.week_number}</td>
              <td className="px-4 py-3">{p.class_name}</td>
              <td className="px-4 py-3">{p.subject_name}</td>
              <td className="px-4 py-3 font-medium">{p.topic}</td>
              <td className="px-4 py-3"><Badge variant={STATUS_BADGE[p.status] ?? "default"}>{STATUS_LABEL[p.status] ?? p.status}</Badge></td>
              <td className="px-4 py-3 text-xs text-[#4a9eca]">{expanded === p.id ? "▲" : "▼"}</td>
            </tr>
            {expanded === p.id && (
              <tr key={`${p.id}-detail`}>
                <td colSpan={7} className="bg-slate-50 px-4 py-4">
                  {p.review_comment && (
                    <p className="mb-3 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm">
                      <span className="font-semibold text-[#1b365d]">Headteacher{p.reviewed_by_name ? ` (${p.reviewed_by_name})` : ""}:</span> {p.review_comment}
                    </p>
                  )}
                  {p.status !== "approved" && (
                    <Button variant="ghost" className="mb-3" onClick={() => startEdit(p)}>Edit this plan</Button>
                  )}
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
                        <dd className="text-sm text-slate-700">{value}</dd>
                      </div>
                    ))}
                  </dl>
                </td>
              </tr>
            )}
          </>
        ))}
      </Table>
    </div>
  );
}
