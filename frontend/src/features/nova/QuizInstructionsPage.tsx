import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { Clock, FileText, HelpCircle } from "lucide-react";
import { novaApi } from "../../api/nova";
import { Button, Card } from "../../components/ui";

export function QuizInstructionsPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: quiz, isLoading } = useQuery({
    queryKey: ["quiz", id],
    queryFn: () => novaApi.quiz(id!),
  });

  if (isLoading) return <div className="p-8 text-center text-slate-400">Loading quiz…</div>;
  if (!quiz) return null;

  const questionCount = quiz.question_count ?? quiz.questions?.length ?? 0;

  return (
    <div className="mx-auto max-w-xl space-y-6 p-6">
      {/* Header */}
      <div className="rounded-2xl bg-[#1b365d] p-6 text-white">
        <p className="text-xs tracking-widest text-[#4a9eca] uppercase">Quiz</p>
        <h1 className="mt-1 text-xl font-semibold">{quiz.title}</h1>
        {quiz.due_at && (
          <p className="mt-2 text-sm text-white/70">
            Due: {new Date(quiz.due_at).toLocaleString()}
          </p>
        )}
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-3">
        <Card className="flex flex-col items-center gap-1 py-4">
          <HelpCircle size={20} className="text-[#1b365d]" />
          <p className="text-lg font-bold text-[#1b365d]">{questionCount}</p>
          <p className="text-xs text-slate-500">Questions</p>
        </Card>
        <Card className="flex flex-col items-center gap-1 py-4">
          <Clock size={20} className="text-[#1b365d]" />
          <p className="text-lg font-bold text-[#1b365d]">{quiz.duration_minutes}</p>
          <p className="text-xs text-slate-500">Minutes</p>
        </Card>
        <Card className="flex flex-col items-center gap-1 py-4">
          <FileText size={20} className="text-[#1b365d]" />
          <p className="text-lg font-bold text-[#1b365d]">
            {Math.ceil(questionCount / 5)}
          </p>
          <p className="text-xs text-slate-500">Pages</p>
        </Card>
      </div>

      {/* Instructions */}
      {quiz.instructions && (
        <Card>
          <h3 className="mb-2 font-semibold text-[#1b365d]">Instructions</h3>
          <p className="text-sm text-slate-600 whitespace-pre-line">{quiz.instructions}</p>
        </Card>
      )}

      {/* Rules */}
      <Card>
        <h3 className="mb-2 font-semibold text-[#1b365d]">Before you begin</h3>
        <ul className="space-y-1.5 text-sm text-slate-600 list-disc list-inside">
          <li>The timer starts the moment you click <strong>Start Quiz</strong>.</li>
          <li>Questions are shown 5 per page — navigate freely before submitting.</li>
          <li>You can only submit once. Answers cannot be changed after submission.</li>
          <li>If time runs out, your answers are automatically submitted.</li>
          <li>Ensure you have a stable internet connection before starting.</li>
        </ul>
      </Card>

      <Button className="w-full" onClick={() => navigate(`/nova/quiz/${id}/attempt`)}>
        Start quiz →
      </Button>
    </div>
  );
}
