import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { AlertTriangle, ChevronLeft, ChevronRight, Clock } from "lucide-react";
import { novaApi } from "../../api/nova";
import { Button, Card, cn, useConfirm } from "../../components/ui";
import { useTimer } from "./useTimer";

const PER_PAGE = 5;

export function QuizAttemptPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { confirm, dialog } = useConfirm();

  const { data: quiz, isLoading } = useQuery({
    queryKey: ["quiz", id],
    queryFn: () => novaApi.quiz(id!),
  });

  const questions: any[] = quiz?.questions ?? [];
  const totalPages = Math.ceil(questions.length / PER_PAGE);

  const [page, setPage] = useState(0);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const startedRef = useRef(new Date().toISOString());

  const { formatted, urgent, elapsed, start } = useTimer(
    (quiz?.duration_minutes ?? 30) * 60,
    () => handleSubmit(true),
  );

  // Start timer once quiz loads
  useEffect(() => {
    if (quiz) start();
  }, [!!quiz]);

  const currentQuestions = questions.slice(page * PER_PAGE, page * PER_PAGE + PER_PAGE);
  const answered = Object.keys(answers).length;

  async function handleSubmit(autoSubmit = false) {
    if (!autoSubmit) {
      const unanswered = questions.length - answered;
      const ok = await confirm({
        title: "Submit quiz",
        message: unanswered > 0
          ? `You have ${unanswered} unanswered question${unanswered > 1 ? "s" : ""}. Submit anyway?`
          : `Submit your quiz now? You cannot change answers after submission.`,
        confirmLabel: "Submit quiz",
        variant: "primary",
      });
      if (!ok) return;
    }
    setSubmitting(true);
    try {
      const result = await novaApi.submitQuiz(id!, {
        answers,
        started_at: startedRef.current,
        time_taken_seconds: elapsed,
      });
      navigate(`/nova/quiz/${id}/result`, { state: result });
    } catch {
      setSubmitting(false);
    }
  }

  if (isLoading) return <div className="p-8 text-center text-slate-400">Loading…</div>;
  if (!quiz) return null;

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-4">
      {dialog}

      {/* Sticky timer bar */}
      <div className={cn(
        "sticky top-0 z-10 flex items-center justify-between rounded-2xl px-4 py-3 shadow-md",
        urgent ? "bg-red-600 text-white" : "bg-[#1b365d] text-white"
      )}>
        <div>
          <p className="text-xs opacity-70">Quiz</p>
          <p className="font-semibold text-sm">{quiz.title}</p>
        </div>
        <div className="flex items-center gap-3">
          {urgent && <AlertTriangle size={16} className="animate-pulse" />}
          <div className="text-right">
            <p className="text-xs opacity-70">Time remaining</p>
            <p className={cn("text-xl font-bold tabular-nums", urgent && "animate-pulse")}>{formatted}</p>
          </div>
        </div>
      </div>

      {/* Progress */}
      <div className="flex items-center gap-2 text-sm text-slate-500">
        <span>{answered} / {questions.length} answered</span>
        <div className="flex-1 h-1.5 rounded-full bg-slate-200">
          <div
            className="h-1.5 rounded-full bg-[#4a9eca] transition-all"
            style={{ width: `${questions.length ? (answered / questions.length) * 100 : 0}%` }}
          />
        </div>
        <span>Page {page + 1} / {totalPages}</span>
      </div>

      {/* Question page indicator dots */}
      <div className="flex gap-1.5 flex-wrap">
        {Array.from({ length: totalPages }).map((_, i) => (
          <button
            key={i}
            onClick={() => setPage(i)}
            className={cn(
              "w-7 h-7 rounded-full text-xs font-semibold transition",
              i === page
                ? "bg-[#1b365d] text-white"
                : "bg-slate-200 text-slate-600 hover:bg-slate-300"
            )}
          >
            {i + 1}
          </button>
        ))}
      </div>

      {/* Questions */}
      {currentQuestions.map((q: any, idx: number) => {
        const qNum = page * PER_PAGE + idx + 1;
        return (
          <Card key={q.id} className={cn(answers[q.id] ? "border-[#4a9eca]" : "")}>
            <div className="flex items-start gap-3">
              <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[#1b365d] text-xs font-bold text-white">
                {qNum}
              </span>
              <div className="flex-1">
                <p className="text-sm font-medium text-slate-800 mb-3">{q.prompt}</p>

                {q.kind === "mcq" && Array.isArray(q.choices) && (
                  <div className="space-y-2">
                    {q.choices.map((choice: string, ci: number) => (
                      <label key={ci} className={cn(
                        "flex items-center gap-3 rounded-xl border px-3 py-2.5 cursor-pointer transition",
                        answers[q.id] === choice
                          ? "border-[#1b365d] bg-[#1b365d]/5 font-medium"
                          : "border-slate-200 hover:border-slate-300"
                      )}>
                        <input
                          type="radio"
                          name={`q-${q.id}`}
                          value={choice}
                          checked={answers[q.id] === choice}
                          onChange={() => setAnswers({ ...answers, [q.id]: choice })}
                          className="accent-[#1b365d]"
                        />
                        <span className="text-sm">{choice}</span>
                      </label>
                    ))}
                  </div>
                )}

                {q.kind === "short" && (
                  <textarea
                    rows={3}
                    className="w-full rounded-xl border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]"
                    placeholder="Write your answer here…"
                    value={answers[q.id] ?? ""}
                    onChange={(e) => setAnswers({ ...answers, [q.id]: e.target.value })}
                  />
                )}
              </div>
            </div>
          </Card>
        );
      })}

      {/* Navigation */}
      <div className="flex justify-between gap-3">
        <Button variant="ghost" onClick={() => setPage((p) => p - 1)} disabled={page === 0}>
          <ChevronLeft size={16} /> Prev
        </Button>
        {page < totalPages - 1 ? (
          <Button onClick={() => setPage((p) => p + 1)}>
            Next <ChevronRight size={16} />
          </Button>
        ) : (
          <Button onClick={() => handleSubmit(false)} disabled={submitting} variant="gold">
            {submitting ? "Submitting…" : "Submit quiz"}
          </Button>
        )}
      </div>
    </div>
  );
}
