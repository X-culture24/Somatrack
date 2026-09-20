import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { studentsApi } from "../../api/students";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

export function MarksEntryPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [classId, setClassId] = useState("");
  const [assessmentId, setAssessmentId] = useState("");
  const [marks, setMarks] = useState<Record<number, string>>({});
  const [competency, setCompetency] = useState<Record<number, string>>({});

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: assessments } = useQuery({
    queryKey: ["assessments", classId],
    enabled: Boolean(classId),
    queryFn: () => academicsApi.assessments({ class_group: classId }),
  });
  const { data: students } = useQuery({
    queryKey: ["students", classId],
    enabled: Boolean(classId),
    queryFn: () => studentsApi.list({ class_group: classId }),
  });

  const save = useMutation({
    mutationFn: async () => {
      await Promise.all(
        (students as any[] ?? []).map((s: any) =>
          marks[s.id]
            ? academicsApi.saveMark({
                assessment: assessmentId,
                student: s.id,
                score: marks[s.id],
                competency_level: competency[s.id] ?? "",
              })
            : Promise.resolve()
        )
      );
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["marks"] }); setMarks({}); },
  });

  async function handleSave() {
    const ok = await confirm({ title: "Save marks", message: "Save marks for all entered students?", confirmLabel: "Save" });
    if (ok) save.mutate();
  }

  return (
    <div className="space-y-4">
      {dialog}
      <PageHeader title="Marks entry / CBC gradebook" />
      <Card>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Class">
            <Select value={classId} onChange={(e) => { setClassId(e.target.value); setAssessmentId(""); }}>
              <option value="">Select class…</option>
              {(classes as any[] ?? []).map((c: any) => <option key={c.id} value={c.id}>{c.grade} {c.stream_name}</option>)}
            </Select>
          </Field>
          <Field label="Assessment">
            <Select value={assessmentId} onChange={(e) => setAssessmentId(e.target.value)} disabled={!classId}>
              <option value="">Select assessment…</option>
              {(assessments as any[] ?? []).map((a: any) => <option key={a.id} value={a.id}>{a.title} — {a.subject_name}</option>)}
            </Select>
          </Field>
        </div>
      </Card>

      {classId && assessmentId && students && (
        <>
          <Table headers={["Learner", "Score", "Competency level"]}>
            {(students as any[]).map((s: any) => (
              <tr key={s.id} className="hover:bg-slate-50">
                <td className="px-4 py-3 font-medium">{s.full_name}</td>
                <td className="px-4 py-3 w-28">
                  <Input type="number" min="0" max="100" value={marks[s.id] ?? ""} onChange={(e) => setMarks({ ...marks, [s.id]: e.target.value })} placeholder="—" />
                </td>
                <td className="px-4 py-3 w-52">
                  <Select value={competency[s.id] ?? ""} onChange={(e) => setCompetency({ ...competency, [s.id]: e.target.value })}>
                    <option value="">Select…</option>
                    <option value="Exceeding expectation">Exceeding expectation</option>
                    <option value="Meeting expectation">Meeting expectation</option>
                    <option value="Approaching expectation">Approaching expectation</option>
                    <option value="Below expectation">Below expectation</option>
                  </Select>
                </td>
              </tr>
            ))}
          </Table>
          <Button onClick={handleSave} disabled={save.isPending}>Save marks</Button>
        </>
      )}
    </div>
  );
}
