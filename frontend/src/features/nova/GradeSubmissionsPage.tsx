import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useParams } from "react-router-dom";
import { novaApi } from "../../api/nova";
import { Button, Card, Field, Input, PageHeader, useConfirm } from "../../components/ui";

export function GradeSubmissionsPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();

  const { data: submissions } = useQuery({
    queryKey: ["submissions", id],
    queryFn: () => novaApi.courseSubmissions(id!),
  });

  const [grading, setGrading] = useState<Record<number, { score: string; feedback: string }>>({});

  const grade = useMutation({
    mutationFn: ({ subId, score, feedback }: { subId: number; score: string; feedback: string }) =>
      novaApi.gradeSubmission(subId, { score, feedback }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["submissions", id] }),
  });

  async function handleGrade(sub: any) {
    const g = grading[sub.id];
    if (!g?.score) return;
    const ok = await confirm({
      title: "Submit grade",
      message: `Give ${sub.student_name} a score of ${g.score}?`,
      confirmLabel: "Save grade",
    });
    if (ok) grade.mutate({ subId: sub.id, score: g.score, feedback: g.feedback ?? "" });
  }

  return (
    <div className="space-y-5">
      {dialog}
      <PageHeader title="Grade submissions" subtitle="Review student work and assign marks." />

      {(submissions as any[] ?? []).length === 0 && (
        <p className="rounded-2xl border border-slate-200 bg-white py-10 text-center text-sm text-slate-400">
          No submissions yet for this course.
        </p>
      )}

      <div className="space-y-4">
        {(submissions as any[] ?? []).map((s: any) => (
          <Card key={s.id}>
            <div className="flex items-start justify-between">
              <div>
                <p className="font-semibold text-[#1b365d]">{s.student_name}</p>
                <p className="text-xs text-slate-500">{s.assignment_title}</p>
                <p className="text-xs text-slate-400 mt-0.5">
                  Submitted {new Date(s.submitted_at).toLocaleString()}
                  {s.is_late && <span className="ml-2 rounded-full bg-orange-100 px-1.5 py-0.5 text-xs text-orange-600">Late</span>}
                </p>
              </div>
              {s.score !== null && s.score !== undefined ? (
                <span className="rounded-full bg-green-100 px-3 py-1 text-sm font-bold text-green-700">{s.score} marks</span>
              ) : (
                <span className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-500">Not graded</span>
              )}
            </div>

            {s.text_response && (
              <div className="mt-3 rounded-xl bg-slate-50 px-3 py-2 text-sm text-slate-700">
                <p className="text-xs font-medium text-slate-400 mb-1">Student response:</p>
                <p className="whitespace-pre-line">{s.text_response}</p>
              </div>
            )}

            {s.file && (
              <button onClick={() => novaApi.downloadSubmission(s.id)}
                className="mt-2 inline-flex items-center gap-1 text-xs text-[#1b365d] underline">
                📎 View attached file
              </button>
            )}

            {s.feedback && (
              <p className="mt-2 text-xs italic text-slate-500">Your feedback: "{s.feedback}"</p>
            )}

            <div className="mt-3 grid gap-2 sm:grid-cols-3">
              <Field label="Score">
                <Input type="number" min="0"
                  value={grading[s.id]?.score ?? s.score ?? ""}
                  onChange={(e) => setGrading({ ...grading, [s.id]: { ...grading[s.id], score: e.target.value } })}
                  placeholder="—" />
              </Field>
              <div className="sm:col-span-2">
                <Field label="Feedback">
                  <Input
                    value={grading[s.id]?.feedback ?? s.feedback ?? ""}
                    onChange={(e) => setGrading({ ...grading, [s.id]: { ...grading[s.id], feedback: e.target.value } })}
                    placeholder="Optional feedback…" />
                </Field>
              </div>
            </div>
            <Button className="mt-2" variant="accent"
              disabled={!grading[s.id]?.score || grade.isPending}
              onClick={() => handleGrade(s)}>
              Save grade
            </Button>
          </Card>
        ))}
      </div>
    </div>
  );
}
