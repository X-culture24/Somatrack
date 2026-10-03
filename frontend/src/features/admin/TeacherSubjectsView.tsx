import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Trash2 } from "lucide-react";
import { teachersApi } from "../../api/teacher";
import { Badge, Button, Card, Input } from "../../components/ui";
import { type PickedAssignment, SubjectAssignmentPicker } from "./SubjectAssignmentPicker";

type Assignment = { id: string; class_id: string; class_name: string; subject_id: string; subject_name: string; is_elective: boolean };
type Teacher = {
  id: string; full_name: string; email: string; phone: string; role: string; is_active: boolean;
  employee_no: string; tsc_no: string; designation: string; department_name: string; highest_qualification: string;
  subjects: string[]; assignments: Assignment[];
  homeroom_classes: { id: string; name: string }[];
  lesson_plans_this_term: number;
};
type Conflict = { class_name: string; subject_name: string; current_teacher_name: string };

// Teacher-first assignment: the left column is every teacher with what they
// teach (so workload is visible at a glance), the right is one teacher's
// full record with their subjects grouped by class, where subjects can be
// added, moved from another teacher, or removed.
export function TeacherSubjectsView() {
  const qc = useQueryClient();
  const [q, setQ] = useState("");
  const [selected, setSelected] = useState("");
  const [adding, setAdding] = useState<PickedAssignment[]>([]);
  const [conflicts, setConflicts] = useState<Conflict[]>([]);

  const { data: teachers = [], isLoading } = useQuery({
    queryKey: ["teachers"],
    queryFn: () => teachersApi.list() as Promise<Teacher[]>,
  });
  const { data: teacher } = useQuery({
    queryKey: ["teacher", selected],
    queryFn: () => teachersApi.get(selected) as Promise<Teacher>,
    enabled: !!selected,
  });

  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["teachers"] });
    qc.invalidateQueries({ queryKey: ["teacher"] });
    qc.invalidateQueries({ queryKey: ["class-subjects"] });
    qc.invalidateQueries({ queryKey: ["staff"] });
  };

  const add = useMutation({
    mutationFn: (reassign: boolean) =>
      teachersApi.addAssignments(selected, adding.map(({ class_id, subject_id }) => ({ class_id, subject_id })), reassign),
    onSuccess: () => { setAdding([]); setConflicts([]); refresh(); },
    onError: (err: any) => {
      const d = err?.response?.data;
      if (err?.response?.status === 409 && d?.conflicts) setConflicts(d.conflicts);
      else window.alert(d?.message ?? "Could not assign these subjects.");
    },
  });
  const remove = useMutation({
    mutationFn: (assignmentId: string) => teachersApi.removeAssignment(selected, assignmentId),
    onSuccess: refresh,
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not remove this subject."),
  });

  const needle = q.trim().toLowerCase();
  const shown = teachers.filter((t) =>
    !needle || t.full_name.toLowerCase().includes(needle) || t.subjects.some((s) => s.toLowerCase().includes(needle)));

  // group the selected teacher's subjects by class
  const byClass = new Map<string, Assignment[]>();
  (teacher?.assignments ?? []).forEach((a) => byClass.set(a.class_name, [...(byClass.get(a.class_name) ?? []), a]));

  function pick(id: string) {
    setSelected(id);
    setAdding([]);
    setConflicts([]);
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.6fr)]">
      <Card className="p-0">
        <div className="border-b p-3">
          <Input placeholder="Search teacher or subject" value={q} onChange={(e) => setQ(e.target.value)} />
        </div>
        {isLoading && <p className="p-4 text-sm text-slate-400">Loading teachers…</p>}
        <ul className="max-h-[70vh] divide-y overflow-y-auto">
          {shown.map((t) => (
            <li key={t.id}>
              <button
                type="button"
                onClick={() => pick(t.id)}
                className={`w-full px-4 py-3 text-left hover:bg-slate-50 ${selected === t.id ? "bg-[#4a9eca]/10" : ""}`}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-medium text-slate-800">{t.full_name}</span>
                  <span className={`text-xs ${t.assignments.length ? "text-slate-500" : "text-amber-600"}`}>
                    {t.assignments.length ? `${t.assignments.length} subject${t.assignments.length === 1 ? "" : "s"}` : "no subjects"}
                  </span>
                </div>
                <p className="mt-0.5 truncate text-xs text-slate-400">
                  {t.subjects.length ? t.subjects.join(", ") : t.role.replaceAll("_", " ").toLowerCase()}
                </p>
              </button>
            </li>
          ))}
          {!isLoading && shown.length === 0 && <li className="p-4 text-sm text-slate-400">No teachers match.</li>}
        </ul>
      </Card>

      {!selected && (
        <Card className="flex items-center justify-center p-10 text-sm text-slate-400">
          Select a teacher to see and change the subjects they teach.
        </Card>
      )}

      {selected && teacher && (
        <div className="space-y-4">
          <Card>
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <h3 className="text-lg font-semibold text-[#1b365d]">{teacher.full_name}</h3>
                <p className="text-sm text-slate-500">
                  {[teacher.designation, teacher.department_name, teacher.highest_qualification].filter(Boolean).join(" · ") ||
                    teacher.role.replaceAll("_", " ").toLowerCase()}
                </p>
                <p className="mt-1 text-xs text-slate-400">
                  {[teacher.employee_no && `Emp. ${teacher.employee_no}`, teacher.tsc_no && `TSC ${teacher.tsc_no}`, teacher.email, teacher.phone]
                    .filter(Boolean).join(" · ")}
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Badge variant="blue">{teacher.assignments.length} subjects</Badge>
                <Badge variant="default">{teacher.lesson_plans_this_term} lesson plans this term</Badge>
                {teacher.homeroom_classes.map((c) => <Badge key={c.id} variant="navy">Class teacher · {c.name}</Badge>)}
              </div>
            </div>
          </Card>

          <Card className="p-0">
            <div className="border-b p-4">
              <h3 className="font-semibold">Subjects taught</h3>
            </div>
            {byClass.size === 0 && <p className="p-4 text-sm text-slate-400">Not assigned any subjects yet.</p>}
            {[...byClass.entries()].map(([className, rows]) => (
              <div key={className} className="border-b px-4 py-3 last:border-0">
                <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">{className}</p>
                <div className="flex flex-wrap gap-2">
                  {rows.map((a) => (
                    <span key={a.id} className="inline-flex items-center gap-1.5 rounded-full bg-[#4a9eca]/10 px-3 py-1 text-sm text-[#1b365d]">
                      {a.subject_name}{a.is_elective && <span className="text-xs text-slate-400">(elective)</span>}
                      <button
                        type="button"
                        title="Remove from this teacher"
                        disabled={remove.isPending}
                        onClick={() => { if (window.confirm(`Remove ${a.subject_name} (${className}) from ${teacher.full_name}?`)) remove.mutate(a.id); }}
                      >
                        <Trash2 size={12} className="text-red-400" />
                      </button>
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </Card>

          <Card>
            <h3 className="mb-1 font-semibold">Assign more subjects</h3>
            <p className="mb-3 text-xs text-slate-500">
              Pick a class and tick subjects. A subject another teacher holds can be ticked too — you'll be asked to confirm the move.
            </p>
            <SubjectAssignmentPicker value={adding} onChange={(v) => { setAdding(v); setConflicts([]); }} teacherId={selected} allowTaken />

            {conflicts.length > 0 && (
              <div className="mt-3 rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-800">
                <p className="font-medium">These subjects already have a teacher:</p>
                <ul className="ml-4 mt-1 list-disc">
                  {conflicts.map((c) => (
                    <li key={c.class_name + c.subject_name}>{c.class_name} · {c.subject_name} — {c.current_teacher_name}</li>
                  ))}
                </ul>
                <div className="mt-2 flex gap-2">
                  <Button variant="accent" onClick={() => add.mutate(true)} disabled={add.isPending}>Move them to {teacher.full_name}</Button>
                  <Button variant="ghost" onClick={() => setConflicts([])}>Cancel</Button>
                </div>
              </div>
            )}

            {conflicts.length === 0 && (
              <Button className="mt-4" onClick={() => add.mutate(false)} disabled={add.isPending || adding.length === 0}>
                {add.isPending ? "Saving…" : `Assign ${adding.length || ""} subject${adding.length === 1 ? "" : "s"}`}
              </Button>
            )}
          </Card>
        </div>
      )}
    </div>
  );
}
