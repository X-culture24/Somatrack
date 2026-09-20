import { useQuery } from "@tanstack/react-query";
import { BookOpen, CheckCircle, ClipboardList } from "lucide-react";
import { schoolApi } from "../../api/school";
import { openNova } from "../../api/nova";
import { Card } from "../../components/ui";

export function StudentHome() {
  const { data: dash } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const recentMarks: any[] = dash?.recent_marks ?? [];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-semibold text-[#1b365d]">
          Welcome, {dash?.name ?? "Student"}
        </h1>
        <p className="text-sm text-slate-500">{dash?.class} · {dash?.term}</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card className="flex flex-col items-center gap-2 py-5">
          <BookOpen size={24} className="text-[#1b365d]" />
          <p className="text-2xl font-bold text-[#1b365d]">NOVA</p>
          <p className="text-xs text-slate-500">Learning campus</p>
        </Card>
        <button onClick={() => openNova()} className="text-left">
          <Card className="flex flex-col items-center gap-2 py-5 hover:border-[#4a9eca] cursor-pointer">
            <ClipboardList size={24} className="text-[#1b365d]" />
            <p className="text-sm font-semibold text-[#1b365d]">My work</p>
            <p className="text-xs text-slate-500">Submissions & attempts</p>
          </Card>
        </button>
        <button onClick={() => openNova()} className="text-left">
          <Card className="flex flex-col items-center gap-2 py-5 hover:border-[#4a9eca] cursor-pointer">
            <CheckCircle size={24} className="text-[#1b365d]" />
            <p className="text-sm font-semibold text-[#1b365d]">Nova campus</p>
            <p className="text-xs text-slate-500">View all courses</p>
          </Card>
        </button>
      </div>

      <button onClick={() => openNova()} className="w-full text-left">
        <Card className="hover:border-[#4a9eca]">
          <p className="font-semibold text-[#1b365d]">Open my NOVA courses</p>
          <p className="text-xs text-slate-500">Continue securely to courses, CATs, and submissions.</p>
        </Card>
      </button>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Recent grades</h3>
        {recentMarks.length === 0 ? (
          <p className="text-sm text-slate-500">No grades recorded yet.</p>
        ) : (
          <ul className="space-y-2">
            {recentMarks.map((m: any, i: number) => (
              <li key={i} className="flex justify-between border-b border-slate-100 pb-2 text-sm last:border-0 last:pb-0">
                <span className="text-slate-700">{m.assessment__title} · {m.assessment__subject__name}</span>
                <span className="font-semibold text-[#1b365d]">{m.score}{m.competency_level ? ` (${m.competency_level})` : ""}</span>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
