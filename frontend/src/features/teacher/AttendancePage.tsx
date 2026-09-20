import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { studentsApi } from "../../api/students";
import { Button, Card, Field, PageHeader, Select, Table, useConfirm } from "../../components/ui";

const DAYS = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"];

export function AttendancePage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: subjects } = useQuery({ queryKey: ["subjects"], queryFn: () => academicsApi.subjects() });

  const [classId, setClassId] = useState("");
  const [subjectId, setSubjectId] = useState("");  // empty = daily register
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [marks, setMarks] = useState<Record<number, string>>({});

  const { data: students } = useQuery({
    queryKey: ["students", classId],
    enabled: Boolean(classId),
    queryFn: () => studentsApi.list({ class_group: classId }),
  });

  // Show existing attendance for today/date if already saved
  const { data: existing } = useQuery({
    queryKey: ["attendance-existing", classId, date, subjectId],
    enabled: Boolean(classId && date),
    queryFn: () =>
      academicsApi.attendance({
        class_group: classId,
        date,
        ...(subjectId ? { subject: subjectId } : {}),
      }),
    select: (rows: any[]) => Object.fromEntries(rows.map((r) => [r.student, r.status])),
  });

  const effectiveMarks = { ...existing, ...marks };

  const save = useMutation({
    mutationFn: async () => {
      await Promise.all(
        (students as any[] ?? []).map((s: any) =>
          academicsApi.saveAttendance({
            student: s.id,
            class_group: classId,
            date,
            status: effectiveMarks[s.id] ?? "present",
            ...(subjectId ? { subject: subjectId } : {}),
          })
        ),
      );
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["attendance-existing", classId, date, subjectId] });
      setMarks({});
    },
  });

  async function handleSave() {
    const cls = (classes as any[] ?? []).find((c: any) => String(c.id) === classId);
    const subj = (subjects as any[] ?? []).find((s: any) => String(s.id) === subjectId);
    const label = subj ? `${subj.name} (subject attendance)` : "daily register";
    const ok = await confirm({
      title: "Save attendance",
      message: `Save ${label} for ${cls?.grade ?? "class"} on ${date}?`,
      confirmLabel: "Save register",
    });
    if (ok) save.mutate();
  }

  const absentCount = Object.values(effectiveMarks).filter((v) => v === "absent").length;
  const presentCount = (students as any[] ?? []).length - absentCount;

  return (
    <div className="space-y-4">
      {dialog}
      <PageHeader title="Attendance register" />

      {/* Controls */}
      <Card>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Class">
            <Select value={classId} onChange={(e) => { setClassId(e.target.value); setMarks({}); }}>
              <option value="">Select class…</option>
              {(classes as any[] ?? []).map((c: any) => (
                <option key={c.id} value={c.id}>{c.grade} {c.stream_name}</option>
              ))}
            </Select>
          </Field>
          <Field label="Subject (leave blank for daily register)">
            <Select value={subjectId} onChange={(e) => { setSubjectId(e.target.value); setMarks({}); }}>
              <option value="">— Daily register —</option>
              {(subjects as any[] ?? []).map((s: any) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </Select>
          </Field>
          <Field label="Date">
            <input
              type="date"
              value={date}
              onChange={(e) => { setDate(e.target.value); setMarks({}); }}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none ring-[#4a9eca] focus:ring-2"
            />
          </Field>
          {classId && students && (
            <div className="flex items-end gap-3 text-sm">
              <span className="rounded-full bg-green-100 px-3 py-1 text-green-700 font-medium">{presentCount} present</span>
              <span className="rounded-full bg-red-100 px-3 py-1 text-red-700 font-medium">{absentCount} absent</span>
            </div>
          )}
        </div>
      </Card>

      {/* Register table */}
      {classId && students ? (
        <>
          <Table headers={["#", "Learner", "Status", "Notes"]}>
            {(students as any[]).map((s: any, i: number) => (
              <tr key={s.id} className={effectiveMarks[s.id] === "absent" ? "bg-red-50" : "hover:bg-slate-50"}>
                <td className="px-4 py-3 text-slate-400 text-xs">{i + 1}</td>
                <td className="px-4 py-3 font-medium">{s.full_name}</td>
                <td className="px-4 py-3">
                  <Select
                    value={effectiveMarks[s.id] ?? "present"}
                    onChange={(e) => setMarks({ ...marks, [s.id]: e.target.value })}
                  >
                    <option value="present">Present</option>
                    <option value="absent">Absent</option>
                    <option value="late">Late</option>
                    <option value="excused">Excused</option>
                  </Select>
                </td>
                <td className="px-4 py-3 text-xs text-slate-500">
                  {effectiveMarks[s.id] === "absent" && "Follow up required"}
                </td>
              </tr>
            ))}
          </Table>
          <Button disabled={!classId || save.isPending} onClick={handleSave}>
            {save.isPending ? "Saving…" : "Save register"}
          </Button>
        </>
      ) : (
        <p className="text-sm text-slate-500">Select a class to load the register.</p>
      )}
    </div>
  );
}
