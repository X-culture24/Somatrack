import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { CreditCard, TrendingUp } from "lucide-react";
import { schoolApi } from "../../api/school";
import { kes } from "../../components/ui";

export function ParentHome() {
  const { data, isLoading } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const kids: any[] = data?.children ?? [];
  const recentPayments: any[] = data?.recent_payments ?? [];

  return (
    <div className="space-y-4 p-4">
      <div>
        <h1 className="text-xl font-semibold text-[#1b365d]">Family dashboard</h1>
        {data?.term && <p className="mt-0.5 text-xs text-slate-500">Current term: {data.term}</p>}
      </div>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2].map((i) => (
            <div key={i} className="h-28 animate-pulse rounded-2xl bg-slate-200" />
          ))}
        </div>
      )}

      {kids.map((c: any) => (
        <div key={c.id} className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-xs text-slate-400">{c.admission_no}</p>
              <h2 className="mt-0.5 text-base font-semibold text-[#1b365d]">{c.name}</h2>
              <p className="text-sm text-slate-500">{c.class}</p>
            </div>
            <span
              className={`mt-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                Number(c.balance) === 0
                  ? "bg-green-100 text-green-700"
                  : "bg-red-100 text-red-700"
              }`}
            >
              {Number(c.balance) === 0 ? "Paid" : "Balance due"}
            </span>
          </div>
          <div className="mt-3 rounded-xl bg-[#f0f4f8] px-3 py-2">
            <p className="text-xs text-slate-500">Current term balance</p>
            <p className="text-lg font-bold text-[#1b365d]">{kes(c.balance)}</p>
            <p className="mt-0.5 text-xs text-slate-400">
              Paybill 247247 · Acc 137101#{String(c.name).replace(/\s+/g, "").toUpperCase()}
            </p>
          </div>
        </div>
      ))}

      {/* Quick-access links */}
      <div className="grid grid-cols-2 gap-3">
        <Link
          to="/parent/fees"
          className="flex flex-col items-center gap-2 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm hover:border-[#4a9eca]"
        >
          <CreditCard size={24} className="text-[#1b365d]" />
          <span className="text-sm font-medium text-[#1b365d]">Fee statements</span>
        </Link>
        <Link
          to="/parent/progress"
          className="flex flex-col items-center gap-2 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm hover:border-[#4a9eca]"
        >
          <TrendingUp size={24} className="text-[#1b365d]" />
          <span className="text-sm font-medium text-[#1b365d]">Progress</span>
        </Link>
      </div>

      {/* Recent payments */}
      {recentPayments.length > 0 && (
        <div>
          <h3 className="mb-2 text-sm font-semibold text-slate-600">Recent payments</h3>
          {recentPayments.slice(0, 5).map((p: any, i: number) => (
            <div key={i} className="mb-2 flex items-center justify-between rounded-xl border border-slate-100 bg-white px-3 py-2">
              <div>
                <p className="text-sm font-medium text-[#1b365d]">{p.student__first_name}</p>
                <p className="text-xs text-slate-400 uppercase">{p.method} · {new Date(p.transaction_date).toLocaleDateString()}</p>
              </div>
              <p className="text-sm font-semibold text-[#1b365d]">{kes(p.amount)}</p>
            </div>
          ))}
        </div>
      )}

      {/* Announcements preview */}
      {(data?.announcements ?? []).length > 0 && (
        <div>
          <h3 className="mb-2 text-sm font-semibold text-slate-600">Latest from school</h3>
          {data.announcements.slice(0, 3).map((a: any) => (
            <div key={a.id} className="mb-2 rounded-xl border border-slate-100 bg-white px-3 py-2">
              <p className="text-sm font-medium text-[#1b365d]">{a.title}</p>
              <p className="mt-0.5 line-clamp-2 text-xs text-slate-500">{a.body}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
