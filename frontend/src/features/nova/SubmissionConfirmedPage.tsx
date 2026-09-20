import { useLocation, useNavigate } from "react-router-dom";
import { CheckCircle, Clock, FileText } from "lucide-react";
import { Button, Card } from "../../components/ui";

export function SubmissionConfirmedPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const state = location.state as any;

  if (!state) {
    navigate("/nova");
    return null;
  }

  const isLate = state.is_late;
  const submittedAt = state.submitted_at ? new Date(state.submitted_at) : new Date();

  return (
    <div className="mx-auto max-w-md space-y-5 p-6">
      {/* Banner */}
      <div className={`rounded-2xl p-8 text-center text-white ${isLate ? "bg-orange-500" : "bg-green-600"}`}>
        <CheckCircle size={40} className="mx-auto mb-3" />
        <h1 className="text-xl font-bold">
          {isLate ? "Submitted (late)" : "Submitted!"}
        </h1>
        <p className="mt-1 text-sm text-white/80">
          {isLate
            ? "Your work has been received but was submitted past the due date."
            : "Your assignment has been received successfully."}
        </p>
      </div>

      {/* Assignment details */}
      <Card>
        <div className="flex items-start gap-3">
          <FileText size={18} className="mt-0.5 text-[#1b365d]" />
          <div>
            <p className="text-xs text-slate-500">Assignment</p>
            <p className="font-semibold text-[#1b365d]">{state.assignment?.title ?? "Assignment"}</p>
          </div>
        </div>
      </Card>

      {/* Timing */}
      <Card>
        <div className="flex items-center gap-3">
          <Clock size={18} className="text-slate-400" />
          <div>
            <p className="text-xs text-slate-500">Submitted at</p>
            <p className="font-semibold text-slate-800">
              {submittedAt.toLocaleString("en-KE", {
                weekday: "short", day: "numeric", month: "short",
                year: "numeric", hour: "2-digit", minute: "2-digit",
              })}
            </p>
          </div>
        </div>
        {state.assignment?.max_score && (
          <p className="mt-3 text-xs text-slate-500">
            Maximum marks: <span className="font-medium text-slate-700">{state.assignment.max_score}</span>.
            Your teacher will grade and provide feedback.
          </p>
        )}
      </Card>

      <div className="flex gap-3">
        <Button variant="ghost" className="flex-1" onClick={() => navigate("/nova")}>
          Back to Nova
        </Button>
        <Button className="flex-1" onClick={() => navigate(-2)}>
          Back to course
        </Button>
      </div>
    </div>
  );
}
