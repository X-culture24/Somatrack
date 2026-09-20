import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";

const STATUS_COLOR: Record<string, string> = {
  present: "bg-green-100 text-green-700",
  absent: "bg-red-100 text-red-700",
  late: "bg-yellow-100 text-yellow-700",
  excused: "bg-blue-100 text-blue-700",
};

const COMPETENCY_COLOR: Record<string, string> = {
  "Exceeding expectation": "bg-green-100 text-green-700",
  "Meeting expectation": "bg-blue-100 text-blue-700",
  "Approaching expectation": "bg-yellow-100 text-yellow-700",
  "Below expectation": "bg-red-100 text-red-700",
};

export function ParentProgressPage() {
  const { data, isLoading } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const [tab, setTab] = useState<"marks" | "attendance">("marks");
  const marks: any[] = data?.recent_marks ?? [];
  const attendance: any[] = data?.attendance ?? [];

  return (
    <div className="p-4">
      <h1 className="text-xl font-semibold text-[#1b365d]">Academic progress</h1>
      {data?.term && <p className="mt-0.5 text-xs text-slate-500">{data.term}</p>}

      {/* Tab switcher */}
      <div className="mt-4 flex rounded-xl bg-slate-100 p-1">
        <button
          className={`flex-1 rounded-lg py-1.5 text-sm font-medium transition ${tab === "marks" ? "bg-white text-[#1b365d] shadow-sm" : "text-slate-500"}`}
          onClick={() => setTab("marks")}
        >
          Marks
        </button>
        <button
          className={`flex-1 rounded-lg py-1.5 text-sm font-medium transition ${tab === "attendance" ? "bg-white text-[#1b365d] shadow-sm" : "text-slate-500"}`}
          onClick={() => setTab("attendance")}
        >
          Attendance
        </button>
      </div>

      {isLoading && (
        <div className="mt-4 space-y-2">
          {[1, 2, 3, 4].map((i) => <div key={i} className="h-14 animate-pulse rounded-xl bg-slate-200" />)}
        </div>
      )}

      {/* Marks tab */}
      {tab === "marks" && !isLoading && (
        <div className="mt-4 space-y-2">
          {marks.length === 0 && (
            <p className="py-8 text-center text-sm text-slate-400">No marks recorded yet this term.</p>
          )}
          {marks.map((m: any, i: number) => (
            <div key={i} className="rounded-xl border border-slate-200 bg-white p-3">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-xs text-slate-400">{m.student__first_name}</p>
                  <p className="text-sm font-medium text-slate-800">{m.assessment__title}</p>
                </div>
                <span className="text-lg font-bold text-[#1b365d]">{m.score}</span>
              </div>
              {m.competency_level && (
                <span
                  className={`mt-1.5 inline-block rounded-full px-2 py-0.5 text-xs font-medium ${COMPETENCY_COLOR[m.competency_level] ?? "bg-slate-100 text-slate-600"}`}
                >
                  {m.competency_level}
                </span>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Attendance tab */}
      {tab === "attendance" && !isLoading && (
        <div className="mt-4 space-y-2">
          {attendance.length === 0 && (
            <p className="py-8 text-center text-sm text-slate-400">No attendance records yet.</p>
          )}
          {attendance.map((a: any, i: number) => (
            <div key={i} className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-3 py-2.5">
              <div>
                <p className="text-xs text-slate-400">{a.student__first_name}</p>
                <p className="text-sm text-slate-700">{new Date(a.date).toLocaleDateString("en-KE", { weekday: "short", day: "numeric", month: "short" })}</p>
              </div>
              <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${STATUS_COLOR[a.status] ?? "bg-slate-100 text-slate-600"}`}>
                {a.status}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
