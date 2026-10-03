import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { UserCheck, Save } from "lucide-react";
import { academicsApi } from "../../api/academics";
import { api } from "../../api/client";
import { Button, Card, PageHeader, Select } from "../../components/ui";
import { teachersApi } from "../../api/teacher";
import { TeacherSubjectsView } from "./TeacherSubjectsView";

function unwrap(d: unknown) {
  if (d && typeof d === "object") {
    if ("results" in (d as Record<string, unknown>)) return (d as { results: unknown }).results;
    if ("data" in (d as Record<string, unknown>)) return (d as { data: unknown }).data;
  }
  return d;
}

const classSubjectsApi = {
  list: (class_id: string) =>
    api.get("/exams/class-subjects/", { params: { class_id } }).then(r => unwrap(r.data) as SubjectRow[]),
  assignTeacher: (id: string, teacher_id: string | null) =>
    api.patch(`/exams/class-subjects/${id}/`, { teacher_id: teacher_id || null }).then(r => r.data),
  assignClassTeacher: (class_id: string, teacher_id: string | null) =>
    api.patch(`/exams/classes/${class_id}/class-teacher/`, { teacher_id: teacher_id || null }).then(r => r.data),
};

interface SubjectRow {
  id: string;
  class_id: string;
  class_name: string;
  subject_id: string;
  subject_name: string;
  teacher_id: string | null;
  teacher_name: string;
  teacher_email: string;
  is_elective: boolean;
}

interface ClassInfo {
  id: string;
  name: string;
  class_teacher_id?: string | null;
}

export function ClassTeacherAssignmentPage() {
  const [view, setView] = useState<"teacher" | "class">("teacher");
  return (
    <div className="p-6 space-y-4">
      <PageHeader title="Teacher Assignment" subtitle="Who teaches what — by teacher, or class by class" />
      <div className="inline-flex rounded-lg border border-slate-200 bg-white p-1 text-sm">
        {([["teacher", "By teacher"], ["class", "By class"]] as const).map(([v, label]) => (
          <button
            key={v}
            type="button"
            onClick={() => setView(v)}
            className={`rounded-md px-4 py-1.5 font-medium ${view === v ? "bg-[#1b365d] text-white" : "text-slate-600 hover:bg-slate-50"}`}
          >
            {label}
          </button>
        ))}
      </div>
      {view === "teacher" ? <TeacherSubjectsView /> : <ByClassView />}
    </div>
  );
}

function ByClassView() {
  const [classId, setClassId] = useState("");
  const [pending, setPending] = useState<Record<string, string | null>>({}); // cs_id → teacher_id
  const [pendingClassTeacher, setPendingClassTeacher] = useState<string | null | undefined>(undefined);

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: subjects = [], isLoading: subjectsLoading } = useQuery({
    queryKey: ["class-subjects", classId],
    queryFn: () => classSubjectsApi.list(classId),
    enabled: !!classId,
  });

  // Every teaching-role user (class/subject teacher, head, deputy, coordinator).
  const { data: teachers = [] } = useQuery({
    queryKey: ["teachers"],
    queryFn: () => teachersApi.list(),
  });

  const classList = (classes as ClassInfo[] | undefined) ?? [];
  const selectedClass = classList.find(c => c.id === classId);

  const qc = useQueryClient();
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["class-subjects", classId] });
    qc.invalidateQueries({ queryKey: ["teachers"] });
  };

  const saveSubjectTeacher = useMutation({
    mutationFn: ({ id, tid }: { id: string; tid: string | null }) => classSubjectsApi.assignTeacher(id, tid),
    onSuccess: invalidate,
  });

  const saveClassTeacher = useMutation({
    mutationFn: ({ cid, tid }: { cid: string; tid: string | null }) => classSubjectsApi.assignClassTeacher(cid, tid),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["classes"] }); setPendingClassTeacher(undefined); },
  });

  function getTeacher(csId: string, fallback: string | null) {
    return csId in pending ? pending[csId] : (fallback ?? "");
  }

  function hasPendingChanges() {
    return Object.keys(pending).length > 0 || pendingClassTeacher !== undefined;
  }

  function handleSaveAll() {
    Object.entries(pending).forEach(([csId, tid]) => saveSubjectTeacher.mutate({ id: csId, tid }));
    if (pendingClassTeacher !== undefined) {
      saveClassTeacher.mutate({ cid: classId, tid: pendingClassTeacher });
    }
    setPending({});
  }

  const teacherOptions = teachers as { id: string; full_name: string }[];

  return (
    <div className="space-y-4">
      <Card className="p-4">
        <div className="flex gap-4 flex-wrap items-end">
          <div className="flex-1 min-w-[220px]">
            <label className="block text-sm font-medium mb-1">Select Class</label>
            <Select value={classId} onChange={e => { setClassId(e.target.value); setPending({}); setPendingClassTeacher(undefined); }}>
              <option value="">Choose a class…</option>
              {classList.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </div>
          {hasPendingChanges() && (
            <Button onClick={handleSaveAll} disabled={saveSubjectTeacher.isPending || saveClassTeacher.isPending}>
              <Save size={14} className="mr-1" />
              {saveSubjectTeacher.isPending || saveClassTeacher.isPending ? "Saving…" : "Save all changes"}
            </Button>
          )}
        </div>
      </Card>

      {classId && (
        <>
          {/* Homeroom / class teacher */}
          <Card className="p-4">
            <div className="flex items-center gap-2 mb-3">
              <UserCheck size={16} className="text-blue-600" />
              <h3 className="font-semibold">Homeroom / Class Teacher</h3>
              <span className="text-xs text-gray-400 ml-1">(responsible for attendance and class administration)</span>
            </div>
            <div className="flex gap-3 items-center">
              <Select
                className="max-w-xs"
                value={pendingClassTeacher !== undefined ? (pendingClassTeacher ?? "") : (selectedClass?.class_teacher_id ?? "")}
                onChange={e => setPendingClassTeacher(e.target.value || null)}
              >
                <option value="">— No homeroom teacher —</option>
                {teacherOptions.map(t => (
                  <option key={t.id} value={t.id}>{t.full_name}</option>
                ))}
              </Select>
              {pendingClassTeacher !== undefined && (
                <span className="text-xs text-amber-600 font-medium">unsaved change</span>
              )}
            </div>
          </Card>

          {/* Subject teachers */}
          <Card className="overflow-hidden">
            <div className="p-4 border-b">
              <h3 className="font-semibold">Subject Teachers</h3>
              <p className="text-xs text-gray-500 mt-0.5">{(subjects as SubjectRow[]).length} subjects assigned to this class</p>
            </div>
            {subjectsLoading && <p className="p-4 text-sm text-gray-500">Loading subjects…</p>}
            {!subjectsLoading && (subjects as SubjectRow[]).length === 0 && (
              <p className="p-8 text-center text-gray-400 text-sm">No subjects assigned to this class yet. Add them in Curriculum → Class Subjects.</p>
            )}
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-xs font-medium text-gray-600">
                <tr>
                  <th className="text-left p-3">Subject</th>
                  <th className="text-left p-3">Assigned Teacher</th>
                  <th className="text-left p-3 text-gray-400">Type</th>
                </tr>
              </thead>
              <tbody>
                {(subjects as SubjectRow[]).map(sub => {
                  const currentTid = getTeacher(sub.id, sub.teacher_id);
                  const changed = sub.id in pending;
                  return (
                    <tr key={sub.id} className={`border-t ${changed ? "bg-amber-50" : ""}`}>
                      <td className="p-3 font-medium">{sub.subject_name}</td>
                      <td className="p-3">
                        <Select
                          value={currentTid ?? ""}
                          onChange={e => setPending(p => ({ ...p, [sub.id]: e.target.value || null }))}
                          className={changed ? "border-amber-400" : ""}
                        >
                          <option value="">— Unassigned —</option>
                          {teacherOptions.map(t => (
                            <option key={t.id} value={t.id}>{t.full_name}</option>
                          ))}
                        </Select>
                        {!changed && sub.teacher_email && (
                          <p className="text-xs text-gray-400 mt-0.5">{sub.teacher_email}</p>
                        )}
                      </td>
                      <td className="p-3 text-gray-400 text-xs">
                        {sub.is_elective ? "Elective" : "Core"}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </Card>
        </>
      )}
    </div>
  );
}
