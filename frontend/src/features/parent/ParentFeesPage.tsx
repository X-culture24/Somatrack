import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { invoicesApi } from "../../api/finance";
import { kes } from "../../components/ui";

export function ParentFeesPage() {
  const { data, isLoading } = useQuery({ queryKey: ["invoices"], queryFn: () => invoicesApi.list() });
  const [openId, setOpenId] = useState<number | null>(null);

  const statement = useQuery({
    queryKey: ["stmt", openId],
    enabled: Boolean(openId),
    queryFn: () => invoicesApi.statement(openId!),
  });

  return (
    <div className="space-y-4 p-4">
      <h1 className="text-xl font-semibold text-[#1b365d]">Fee statements</h1>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <div key={i} className="h-20 animate-pulse rounded-2xl bg-slate-200" />)}
        </div>
      )}

      {(data as any[] ?? []).map((inv: any) => (
        <div key={inv.id} className="rounded-2xl border border-slate-200 bg-white shadow-sm overflow-hidden">
          <div className="flex items-center justify-between p-4">
            <div>
              <p className="text-xs text-slate-400">{inv.term_label}</p>
              <p className="mt-0.5 font-semibold text-[#1b365d]">{inv.student_name}</p>
            </div>
            <span
              className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                inv.status === "paid" ? "bg-green-100 text-green-700"
                : inv.status === "partial" ? "bg-yellow-100 text-yellow-700"
                : "bg-red-100 text-red-700"
              }`}
            >
              {inv.status}
            </span>
          </div>

          <div className="grid grid-cols-3 divide-x divide-slate-100 border-t border-slate-100 text-center">
            <div className="py-2">
              <p className="text-xs text-slate-400">Billed</p>
              <p className="text-sm font-medium">{kes(inv.total_due)}</p>
            </div>
            <div className="py-2">
              <p className="text-xs text-slate-400">Paid</p>
              <p className="text-sm font-medium text-green-600">{kes(inv.total_paid)}</p>
            </div>
            <div className="py-2">
              <p className="text-xs text-slate-400">Balance</p>
              <p className={`text-sm font-bold ${Number(inv.balance) > 0 ? "text-red-600" : "text-green-600"}`}>
                {kes(inv.balance)}
              </p>
            </div>
          </div>

          <div className="border-t border-slate-100 px-4 py-2">
            <button
              className="w-full text-center text-sm font-medium text-[#1b365d]"
              onClick={() => setOpenId(openId === inv.id ? null : inv.id)}
            >
              {openId === inv.id ? "Hide statement ▲" : "View statement ▼"}
            </button>
          </div>

          {openId === inv.id && (
            <div className="border-t border-slate-100 bg-[#f9f7f3] px-4 pb-4 pt-3">
              {statement.isLoading && <p className="text-xs text-slate-400">Loading…</p>}
              {statement.data && (
                <>
                  <p className="mb-2 text-xs text-slate-500">
                    Pay via Paybill <strong>{statement.data.school.paybill}</strong> · Account{" "}
                    <strong>{statement.data.school.account_ref}</strong>
                  </p>
                  <ul className="divide-y divide-slate-200 text-sm">
                    {statement.data.invoice.line_items.map((l: any) => (
                      <li key={l.id} className="flex justify-between py-1.5">
                        <span className="text-slate-600">{l.description}</span>
                        <span className={`font-medium ${Number(l.amount) < 0 ? "text-green-600" : ""}`}>
                          {kes(l.amount)}
                        </span>
                      </li>
                    ))}
                  </ul>
                  <div className="mt-2 flex justify-between border-t border-slate-300 pt-2 font-semibold">
                    <span>Balance due</span>
                    <span className="text-[#1b365d]">{kes(statement.data.invoice.balance)}</span>
                  </div>
                  {statement.data.payments?.length > 0 && (
                    <div className="mt-3">
                      <p className="mb-1 text-xs font-medium text-slate-500 uppercase tracking-wide">Payments received</p>
                      {statement.data.payments.map((p: any) => (
                        <div key={p.id} className="flex justify-between text-xs py-1 text-slate-600">
                          <span>{new Date(p.transaction_date).toLocaleDateString()} · {p.method.toUpperCase()}</span>
                          <span className="text-green-600 font-medium">{kes(p.amount)}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
