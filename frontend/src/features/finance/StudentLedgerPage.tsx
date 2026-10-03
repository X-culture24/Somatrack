import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { studentsApi } from "../../api/students";
import { Input, PageHeader, Table, kes } from "../../components/ui";

export function StudentLedgerPage() {
  const [q, setQ] = useState("");
  const [selectedId, setSelectedId] = useState("");

  const { data: students } = useQuery({
    queryKey: ["students-search", q],
    queryFn: () => studentsApi.list({ search: q }),
    enabled: q.length >= 2,
  });

  const { data: ledger, isLoading } = useQuery({
    queryKey: ["student-ledger", selectedId],
    queryFn: () => studentsApi.ledger(selectedId),
    enabled: !!selectedId,
  });

  const student = ledger?.student;
  const invoices: any[] = ledger?.invoices ?? [];
  const payments: any[] = ledger?.payments ?? [];

  const totalDue = invoices.reduce((s: number, i: any) => s + (i.total_amount ?? 0), 0);
  const totalPaid = invoices.reduce((s: number, i: any) => s + (i.total_paid ?? 0), 0);
  const totalBalance = invoices.reduce((s: number, i: any) => s + (i.balance ?? 0), 0);

  return (
    <div className="p-6">
      <PageHeader title="Student Ledger" subtitle="Per-student fee and payment history" />

      {/* Student lookup */}
      <div className="mb-6 max-w-lg rounded-lg border border-slate-200 bg-slate-50 p-4">
        <label className="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-500">
          Search student by name or admission no.
        </label>
        <Input
          placeholder="Type at least 2 characters…"
          value={q}
          onChange={(e) => { setQ(e.target.value); setSelectedId(""); }}
        />
        {q.length >= 2 && (students ?? []).length > 0 && !selectedId && (
          <div className="mt-2 rounded-lg border border-slate-200 bg-white shadow-sm">
            {(students ?? []).slice(0, 10).map((s: any) => (
              <button
                key={s.id}
                className="flex w-full items-center gap-3 px-4 py-2.5 text-left hover:bg-slate-50 border-b last:border-0"
                onClick={() => { setSelectedId(s.id); setQ(s.full_name); }}
              >
                <span className="text-sm font-medium text-[#1b365d]">{s.full_name}</span>
                <span className="text-xs text-slate-400 ml-1">{s.admission_no}</span>
                <span className="ml-auto text-xs text-slate-400">{s.class_name}</span>
              </button>
            ))}
          </div>
        )}
      </div>

      {selectedId && !isLoading && student && (
        <>
          {/* Student card */}
          <div className="mb-5 flex items-center gap-6 rounded-lg border border-[#1b365d]/20 bg-[#1b365d]/5 p-4">
            <div>
              <div className="text-lg font-bold text-[#1b365d]">{student.full_name}</div>
              <div className="text-sm text-slate-500">Adm: {student.admission_no} · {student.class_name || "No class"}</div>
            </div>
            <div className="ml-auto flex gap-6 text-center">
              <div>
                <div className="text-xs font-semibold uppercase text-slate-400">Total Billed</div>
                <div className="text-base font-bold text-slate-800">{kes(totalDue)}</div>
              </div>
              <div>
                <div className="text-xs font-semibold uppercase text-slate-400">Total Paid</div>
                <div className="text-base font-bold text-green-700">{kes(totalPaid)}</div>
              </div>
              <div>
                <div className="text-xs font-semibold uppercase text-slate-400">Balance</div>
                <div className={`text-base font-bold ${totalBalance > 0 ? "text-red-600" : "text-green-700"}`}>
                  {kes(totalBalance)}
                </div>
              </div>
            </div>
          </div>

          <div className="grid gap-5 lg:grid-cols-2">
            {/* Invoices */}
            <div>
              <h3 className="mb-2 text-sm font-semibold text-slate-600 uppercase tracking-wide">
                Invoices ({invoices.length})
              </h3>
              <Table headers={["Term", "Year", "Billed", "Paid", "Waivers", "Balance"]}>
                {invoices.map((inv: any) => (
                  <tr key={inv.invoice_id} className="hover:bg-slate-50">
                    <td className="px-3 py-2 text-sm font-medium">{inv.term || "—"}</td>
                    <td className="px-3 py-2 text-sm text-slate-500">{inv.year || "—"}</td>
                    <td className="px-3 py-2 text-sm">{kes(inv.total_amount)}</td>
                    <td className="px-3 py-2 text-sm text-green-700">{kes(inv.total_paid)}</td>
                    <td className="px-3 py-2 text-sm text-amber-600">{kes(inv.total_waivers)}</td>
                    <td className={`px-3 py-2 text-sm font-semibold ${inv.balance > 0 ? "text-red-600" : "text-green-700"}`}>
                      {kes(inv.balance)}
                    </td>
                  </tr>
                ))}
              </Table>
              {invoices.length === 0 && (
                <p className="py-6 text-center text-sm text-slate-400">No invoices found.</p>
              )}
            </div>

            {/* Payments */}
            <div>
              <h3 className="mb-2 text-sm font-semibold text-slate-600 uppercase tracking-wide">
                Payments ({payments.length})
              </h3>
              <Table headers={["Date", "Amount", "Method", "Ref", "Status"]}>
                {payments.map((p: any) => (
                  <tr key={p.payment_id} className="hover:bg-slate-50">
                    <td className="px-3 py-2 text-sm text-slate-600">{p.date}</td>
                    <td className="px-3 py-2 text-sm font-medium text-green-700">{kes(p.amount)}</td>
                    <td className="px-3 py-2">
                      <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 capitalize">
                        {p.method}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-xs text-slate-400 font-mono">{p.ref || "—"}</td>
                    <td className="px-3 py-2">
                      <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${
                        p.status === "confirmed" ? "bg-green-100 text-green-700" : "bg-amber-100 text-amber-700"
                      }`}>
                        {p.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </Table>
              {payments.length === 0 && (
                <p className="py-6 text-center text-sm text-slate-400">No payments found.</p>
              )}
            </div>
          </div>
        </>
      )}

      {!selectedId && (
        <div className="mt-8 rounded-lg border border-dashed border-slate-200 py-16 text-center">
          <p className="text-sm text-slate-400">Search for a student above to view their ledger.</p>
        </div>
      )}
    </div>
  );
}
