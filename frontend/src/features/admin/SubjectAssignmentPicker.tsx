import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { X } from "lucide-react";
import { academicsApi } from "../../api/academics";
import { api, unwrap } from "../../api/client";
import { Field, Select } from "../../components/ui";

// One subject in one class — the unit a teacher is assigned. Names are kept
// alongside the ids so the chosen list can be shown without refetching.
export type PickedAssignment = { class_id: string; subject_id: string; class_name: string; subject_name: string };

type ClassSubject = {
  id: string;
  subject_id: string;
  subject_name: string;
  teacher_id: string | null;
  teacher_name: string;
};

const classSubjectsFor = (classId: string) =>
  api.get("/exams/class-subjects/", { params: { class_id: classId } }).then((r) => unwrap<ClassSubject>(r.data));

const key = (a: { class_id: string; subject_id: string }) => `${a.class_id}:${a.subject_id}`;

// Pick a class, tick the subjects on its curriculum. Subjects another
// teacher already holds are marked; with allowTaken they can still be ticked
// (the caller then sends reassign: true), otherwise they're locked.
export function SubjectAssignmentPicker({
  value, onChange, teacherId, allowTaken = false,
}: {
  value: PickedAssignment[];
  onChange: (v: PickedAssignment[]) => void;
  teacherId?: string;
  allowTaken?: boolean;
}) {
  const [classId, setClassId] = useState("");
  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: subjects = [], isLoading } = useQuery({
    queryKey: ["class-subjects", classId],
    queryFn: () => classSubjectsFor(classId),
    enabled: !!classId,
  });
  const className = ((classes as any[]) ?? []).find((c: any) => c.id === classId)?.name ?? "";
  const picked = new Set(value.map(key));

  function toggle(s: ClassSubject) {
    const a = { class_id: classId, subject_id: s.subject_id, class_name: className, subject_name: s.subject_name };
    onChange(picked.has(key(a)) ? value.filter((v) => key(v) !== key(a)) : [...value, a]);
  }

  return (
    <div className="space-y-3">
      <div className="max-w-xs">
        <Field label="Class">
          <Select value={classId} onChange={(e) => setClassId(e.target.value)}>
            <option value="">Choose a class…</option>
            {((classes as any[]) ?? []).map((c: any) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </Select>
        </Field>
      </div>

      {classId && isLoading && <p className="text-sm text-slate-400">Loading subjects…</p>}
      {classId && !isLoading && subjects.length === 0 && (
        <p className="text-sm text-slate-400">This class has no subjects on its curriculum yet — add them under Curriculum.</p>
      )}
      {classId && subjects.length > 0 && (
        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {subjects.map((s) => {
            const takenBy = s.teacher_id && s.teacher_id !== teacherId ? s.teacher_name : "";
            const checked = picked.has(`${classId}:${s.subject_id}`);
            const locked = !!takenBy && !allowTaken && !checked;
            return (
              <label
                key={s.id}
                className={`flex items-start gap-2 rounded-lg border px-3 py-2 text-sm ${
                  checked ? "border-[#4a9eca] bg-[#4a9eca]/5" : "border-slate-200"
                } ${locked ? "opacity-60" : "cursor-pointer"}`}
              >
                <input type="checkbox" className="mt-0.5" checked={checked} disabled={locked} onChange={() => toggle(s)} />
                <span>
                  <span className="font-medium text-slate-700">{s.subject_name}</span>
                  {takenBy && (
                    <span className={`block text-xs ${checked ? "text-amber-600" : "text-slate-400"}`}>
                      {checked ? `will move from ${takenBy}` : `taught by ${takenBy}`}
                    </span>
                  )}
                </span>
              </label>
            );
          })}
        </div>
      )}

      {value.length > 0 && (
        <div>
          <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-400">Selected ({value.length})</p>
          <div className="flex flex-wrap gap-2">
            {value.map((a) => (
              <span key={key(a)} className="inline-flex items-center gap-1 rounded-full bg-[#4a9eca]/15 px-3 py-1 text-xs font-medium text-[#1b365d]">
                {a.class_name} · {a.subject_name}
                <button type="button" onClick={() => onChange(value.filter((v) => key(v) !== key(a)))} aria-label="Remove">
                  <X size={12} />
                </button>
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
