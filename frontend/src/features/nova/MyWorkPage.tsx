import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router-dom";
import { CheckCircle, Clock, FileText, HelpCircle } from "lucide-react";
import { novaApi } from "../../api/nova";
import { Card, PageHeader } from "../../components/ui";

function formatDuration(seconds: number | null | undefined) {
  if (!seconds) return "—";
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return m > 0 ? `${m}m ${s}s` : `${s}s`;
}

export function MyWorkPage() {
  const [tab, setTab] = useState<"quizzes" | "assignments">("quizzes");

  const { data: attempts } = useQuery({
    queryKey: ["my-attempts"],
    queryFn: () => novaApi.myAttempts(),
  });
  const { data: submissions } = useQuery({
    queryKey: ["my-submissions"],
    queryFn: () => novaApi.mySubmissions(),
  });

  return (
    <div className="space-y-5">
      <PageHeader title="My work" subtitle="Quiz attempts and assignment submissions." />

      {/* Tab switcher */}
      <div className="flex rounded-xl bg-slate-100 p-1 w-full max-w-xs">
        <button
          className={`flex-1 rounded-lg py-1.5 text-sm font-medium transition ${tab === "quizzes" ? "bg-white text-[#1b365d] shadow-sm" : "text-slate-500"}`}
          onClick={() => setTab("quizzes")}
        >
          Quizzes
        </button>
        <button
          className={`flex-1 rounded-lg py-1.5 text-sm font-medium transition ${tab === "assignments" ? "bg-white text-[#1b365d] shadow-sm" : "text-slate-500"}`}
          onClick={() => setTab("assignments")}
        >
          Assignments
        </button>
      </div>

      {/* Quiz attempts */}
      {tab === "quizzes" && (
        <div className="space-y-3">
          {(attempts as any[] ?? []).length === 0 && (
            <p className="rounded-2xl border border-slate-200 bg-white py-10 text-center text-sm text-slate-400">
              No quiz attempts yet.
            </p>
          )}
          {(attempts as any[] ?? []).map((a: any) => {
            const score = Number(a.score ?? 0);
            const pct = a.total_points > 0 ? Math.round((score / Number(a.total_points)) * 100) : null;
            return (
              <Card key={a.id}>
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-3">
                    <HelpCircle size={18} className="mt-0.5 text-[#1b365d]" />
                    <div>
                      <p className="font-semibold text-[#1b365d]">{a.quiz_title}</p>
                      <p className="text-xs text-slate-400">
                        Submitted {new Date(a.submitted_at).toLocaleDateString("en-KE", { day: "numeric", month: "short", year: "numeric" })}
                      </p>
                    </div>
                  </div>
                  {pct !== null && (
                    <span className={`rounded-full px-2 py-0.5 text-sm font-bold ${pct >= 60 ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>
                      {pct}%
                    </span>
                  )}
                </div>
                <div className="mt-3 flex gap-4 text-xs text-slate-500">
                  <span className="flex items-center gap-1">
                    <CheckCircle size={12} /> Score: {a.score ?? "—"} pts
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock size={12} /> Time: {formatDuration(a.time_taken_seconds)}
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock size={12} /> Allowed: {a.duration_minutes}m
                  </span>
                </div>
              </Card>
            );
          })}
        </div>
      )}

      {/* Assignment submissions */}
      {tab === "assignments" && (
        <div className="space-y-3">
          {(submissions as any[] ?? []).length === 0 && (
            <p className="rounded-2xl border border-slate-200 bg-white py-10 text-center text-sm text-slate-400">
              No assignment submissions yet.
            </p>
          )}
          {(submissions as any[] ?? []).map((s: any) => (
            <Card key={s.id}>
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3">
                  <FileText size={18} className="mt-0.5 text-[#1b365d]" />
                  <div>
                    <p className="font-semibold text-[#1b365d]">{s.assignment_title}</p>
                    <p className="text-xs text-slate-400">
                      Submitted {new Date(s.submitted_at).toLocaleDateString("en-KE", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" })}
                    </p>
                  </div>
                </div>
                <div className="flex flex-col items-end gap-1">
                  {s.is_late && (
                    <span className="rounded-full bg-orange-100 px-2 py-0.5 text-xs font-medium text-orange-600">Late</span>
                  )}
                  {s.score !== null && s.score !== undefined ? (
                    <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700">
                      {s.score} marks
                    </span>
                  ) : (
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500">Awaiting grade</span>
                  )}
                </div>
              </div>
              {s.feedback && (
                <p className="mt-2 rounded-lg bg-slate-50 px-3 py-2 text-xs text-slate-600 italic">
                  Teacher: "{s.feedback}"
                </p>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
