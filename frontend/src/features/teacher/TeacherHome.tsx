import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { CalendarCheck, NotebookPen, FileText } from "lucide-react";
import { schoolApi } from "../../api/school";
import { Card, PageHeader, Stat } from "../../components/ui";

const QUICK_ACTIONS = [
  { to: "/teacher/attendance", label: "Take attendance", icon: CalendarCheck },
  { to: "/teacher/marks", label: "Enter marks", icon: NotebookPen },
  { to: "/teacher/lesson-plans", label: "New lesson plan", icon: FileText },
];

export function TeacherHome() {
  const { data } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const marks: any[] = data?.recent_marks ?? [];
  const attendance: any[] = data?.recent_attendance ?? [];

  return (
    <div>
      <PageHeader title="Teacher portal" subtitle="Attendance, CBC assessments, and Nova course shells." />
      <div className="grid gap-4 sm:grid-cols-3">
        <Stat label="Learners in homeroom" value={data?.learner_count ?? "—"} />
        <Stat label="Nova courses" value={data?.nova_courses ?? "—"} />
        <Stat label="Assignments" value={data?.open_assignments ?? "—"} />
      </div>

      <div className="mt-4 grid gap-3 sm:grid-cols-3">
        {QUICK_ACTIONS.map(({ to, label, icon: Icon }) => (
          <Link key={to} to={to} className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm hover:border-[#4a9eca]">
            <Icon size={20} className="text-[#1b365d]" />
            <span className="text-sm font-medium text-[#1b365d]">{label}</span>
          </Link>
        ))}
      </div>

      <div className="mt-6 grid gap-4 lg:grid-cols-2">
        <Card>
          <h3 className="mb-3 font-semibold text-[#1b365d]">Recently recorded marks</h3>
          {marks.length === 0 ? (
            <p className="text-sm text-slate-400">No marks recorded yet.</p>
          ) : (
            <ul className="space-y-2">
              {marks.map((m: any, i: number) => (
                <li key={i} className="flex justify-between border-b border-slate-100 pb-2 text-sm last:border-0 last:pb-0">
                  <span className="text-slate-700">{m.student__first_name} {m.student__last_name} — {m.assessment__title}</span>
                  <span className="font-semibold text-[#1b365d]">{m.score}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
        <Card>
          <h3 className="mb-3 font-semibold text-[#1b365d]">Recent attendance</h3>
          {attendance.length === 0 ? (
            <p className="text-sm text-slate-400">No attendance recorded yet.</p>
          ) : (
            <ul className="space-y-2">
              {attendance.map((a: any, i: number) => (
                <li key={i} className="flex justify-between border-b border-slate-100 pb-2 text-sm last:border-0 last:pb-0">
                  <span className="text-slate-700">{a.student__first_name} — {a.class_group__grade}</span>
                  <span className="capitalize text-slate-500">{a.status} · {a.date}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </div>
  );
}
