import { useLocation, useNavigate, useParams } from "react-router-dom";
import { CheckCircle, Clock, Trophy } from "lucide-react";
import { Button, Card } from "../../components/ui";

function formatDuration(seconds: number) {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m === 0) return `${s}s`;
  return `${m}m ${s}s`;
}

export function QuizResultPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const result = location.state as any;

  if (!result) {
    navigate(`/nova/quiz/${id}/instructions`);
    return null;
  }

  const score = Number(result.score ?? 0);
  const total = Number(result.total_points ?? 0);
  const pct = total > 0 ? Math.round((score / total) * 100) : 0;
  const timeTaken = result.time_taken_seconds;

  const grade =
    pct >= 80 ? { label: "Exceeding expectation", color: "text-green-600", bg: "bg-green-50" } :
    pct >= 60 ? { label: "Meeting expectation", color: "text-blue-600", bg: "bg-blue-50" } :
    pct >= 40 ? { label: "Approaching expectation", color: "text-yellow-600", bg: "bg-yellow-50" } :
    { label: "Below expectation", color: "text-red-600", bg: "bg-red-50" };

  return (
    <div className="mx-auto max-w-md space-y-5 p-6">
      {/* Score circle */}
      <div className="rounded-2xl bg-[#1b365d] p-8 text-center text-white">
        <Trophy size={32} className="mx-auto mb-3 text-[#4a9eca]" />
        <p className="text-5xl font-bold">{pct}%</p>
        <p className="mt-1 text-sm text-white/70">{score} / {total} points</p>
      </div>

      {/* Competency badge */}
      <Card className={grade.bg}>
        <div className="flex items-center gap-3">
          <CheckCircle size={20} className={grade.color} />
          <div>
            <p className="text-xs text-slate-500">CBC Competency level</p>
            <p className={`font-semibold ${grade.color}`}>{grade.label}</p>
          </div>
        </div>
      </Card>

      {/* Timing */}
      <Card>
        <div className="flex items-center gap-3">
          <Clock size={18} className="text-slate-400" />
          <div>
            <p className="text-xs text-slate-500">Time used</p>
            <p className="font-semibold text-[#1b365d]">
              {timeTaken ? formatDuration(timeTaken) : "—"}
            </p>
          </div>
        </div>
        {result.submitted_at && (
          <p className="mt-2 text-xs text-slate-400">
            Submitted at {new Date(result.submitted_at).toLocaleString()}
          </p>
        )}
      </Card>

      <div className="flex gap-3">
        <Button variant="ghost" className="flex-1" onClick={() => navigate("/nova")}>
          Back to courses
        </Button>
        <Button className="flex-1" onClick={() => navigate(`/nova/courses/${result.course_id ?? ""}`)}>
          Return to course
        </Button>
      </div>
    </div>
  );
}
