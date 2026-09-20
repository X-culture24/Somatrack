import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { UserPlus, Megaphone, ClipboardList } from "lucide-react";
import { schoolApi } from "../../api/school";
import { Card, PageHeader, Stat, kes } from "../../components/ui";

const QUICK_ACTIONS = [
  { to: "/admin/users", label: "New user", icon: UserPlus },
  { to: "/admin/announcements", label: "New announcement", icon: Megaphone },
  { to: "/admin/admissions", label: "New admission", icon: ClipboardList },
];

export function AdminHome() {
  const { data } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const activity: any[] = data?.recent_activity ?? [];

  return (
    <div>
      <PageHeader title="Administration" subtitle="School configuration, enrolment, and operations." />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat label="Active students" value={data?.students ?? "—"} />
        <Stat label="Classes" value={data?.classes ?? "—"} />
        <Stat label="Open admissions" value={data?.open_admissions ?? "—"} />
        <Stat label="Fees outstanding" value={kes(data?.outstanding_fees)} />
      </div>

      <div className="mt-4 grid gap-3 sm:grid-cols-3">
        {QUICK_ACTIONS.map(({ to, label, icon: Icon }) => (
          <Link key={to} to={to} className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm hover:border-[#4a9eca]">
            <Icon size={20} className="text-[#1b365d]" />
            <span className="text-sm font-medium text-[#1b365d]">{label}</span>
          </Link>
        ))}
      </div>

      <Card className="mt-6">
        <h3 className="mb-3 font-semibold text-[#1b365d]">Recent activity</h3>
        {activity.length === 0 ? (
          <p className="text-sm text-slate-400">No recent activity yet.</p>
        ) : (
          <ul className="space-y-2">
            {activity.map((a: any, i: number) => (
              <li key={i} className="flex items-start justify-between gap-4 border-b border-slate-100 pb-2 last:border-0 last:pb-0">
                <span className="text-sm text-slate-700">{a.label}</span>
                <span className="shrink-0 text-xs text-slate-400">{new Date(a.at).toLocaleString()}</span>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
