import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Download, Receipt } from "lucide-react";
import { feePaymentApi, invoicesApi, paymentsApi } from "../../api/finance";
import { schoolApi } from "../../api/school";
import { kes } from "../../components/ui";

export function ParentFeesPage() {
  const { data, isLoading } = useQuery({ queryKey: ["invoices"], queryFn: () => invoicesApi.list() });
  const [openId, setOpenId] = useState<string | null>(null);

  const detail = useQuery({
    queryKey: ["invoice-detail", openId],
    enabled: Boolean(openId),
    queryFn: () => invoicesApi.get(openId!),
  });

  return (
    <div className="space-y-4">
      <Receipts />

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
              <p className="text-xs text-slate-400">{inv.term_name}</p>
              <p className="mt-0.5 font-semibold text-[#1b365d]">{inv.invoice_no}</p>
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
              {detail.isLoading && <p className="text-xs text-slate-400">Loading…</p>}
              {detail.data && (
                <>
                  <ul className="divide-y divide-slate-200 text-sm">
                    {(detail.data as any).lines.map((l: any) => (
                      <li key={l.id} className="flex justify-between py-1.5">
                        <span className="text-slate-600">{l.description}</span>
                        <span className="font-medium">{kes(l.amount)}</span>
                      </li>
                    ))}
                  </ul>
                  <div className="mt-2 flex justify-between border-t border-slate-300 pt-2 font-semibold">
                    <span>Balance due</span>
                    <span className="text-[#1b365d]">{kes((detail.data as any).balance)}</span>
                  </div>
                  {Number((detail.data as any).balance) > 0 && inv.student_id && (
                    <PayPanel studentId={inv.student_id} balance={Number((detail.data as any).balance)} />
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

function PayPanel({ studentId, balance }: { studentId: string; balance: number }) {
  const [phone, setPhone] = useState("");
  const [amount, setAmount] = useState(String(balance));
  const [result, setResult] = useState<{ method: string; message: string; ok: boolean } | null>(null);

  const mpesa = useMutation({
    mutationFn: () => feePaymentApi.initiateMpesa({ student_id: studentId, phone, amount: Number(amount) }),
    onSuccess: () => setResult({ method: "M-Pesa", message: "STK push sent — check your phone and enter your M-Pesa PIN.", ok: true }),
    onError: (err: any) => setResult({ method: "M-Pesa", message: err?.response?.data?.message ?? "M-Pesa request failed. Try again.", ok: false }),
  });

  const equity = useMutation({
    mutationFn: () => feePaymentApi.initiateEquity({ student_id: studentId, phone, amount: Number(amount) }),
    onSuccess: (data: any) => setResult({ method: "Equity", message: data?.description ?? "Payment request sent to your Equity Mobile app. Approve it to complete the payment.", ok: true }),
    onError: (err: any) => setResult({ method: "Equity", message: err?.response?.data?.message ?? "Equity request failed. Try again.", ok: false }),
  });

  const busy = mpesa.isPending || equity.isPending;

  return (
    <div className="mt-4 rounded-xl border border-[#4a9eca]/30 bg-white p-4">
      <p className="mb-3 text-sm font-semibold text-[#1b365d]">Pay now</p>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
        <div className="flex-1">
          <label className="mb-1 block text-xs text-slate-500">Mobile number</label>
          <input
            type="tel"
            placeholder="07XXXXXXXX or 01XXXXXXXX"
            value={phone}
            onChange={(e) => { setPhone(e.target.value); setResult(null); }}
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[#4a9eca]/40"
          />
        </div>
        <div className="w-32">
          <label className="mb-1 block text-xs text-slate-500">Amount (KES)</label>
          <input
            type="number"
            min="1"
            value={amount}
            onChange={(e) => { setAmount(e.target.value); setResult(null); }}
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[#4a9eca]/40"
          />
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => { setResult(null); mpesa.mutate(); }}
            disabled={busy || !phone || !amount}
            className="rounded-lg bg-[#4CAF50] px-4 py-2 text-sm font-semibold text-white hover:bg-[#43a047] disabled:opacity-50"
          >
            {mpesa.isPending ? "Sending…" : "M-Pesa"}
          </button>
          <button
            onClick={() => { setResult(null); equity.mutate(); }}
            disabled={busy || !phone || !amount}
            className="rounded-lg bg-[#D32F2F] px-4 py-2 text-sm font-semibold text-white hover:bg-[#c62828] disabled:opacity-50"
          >
            {equity.isPending ? "Sending…" : "Equity Mobile"}
          </button>
        </div>
      </div>
      {result && (
        <p className={`mt-3 rounded-lg px-3 py-2 text-sm ${result.ok ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"}`}>
          <span className="font-medium">{result.method}:</span> {result.message}
        </p>
      )}
    </div>
  );
}

type Child = { student_id: string; full_name: string };

// Every receipt for each child. A parent arriving from the payment SMS
// (…/parent/fees?receipt=<id>) sees that receipt highlighted and in view.
function Receipts() {
  const [params] = useSearchParams();
  const highlight = params.get("receipt");
  const { data } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const kids: Child[] = data?.children ?? [];
  if (kids.length === 0) return null;
  return (
    <section>
      <h1 className="mb-2 text-xl font-semibold text-[#1b365d]">Receipts</h1>
      <div className="space-y-4">
        {kids.map((k) => <ChildReceipts key={k.student_id} child={k} highlight={highlight} showName={kids.length > 1} />)}
      </div>
    </section>
  );
}

function ChildReceipts({ child, highlight, showName }: { child: Child; highlight: string | null; showName: boolean }) {
  const { data, isLoading } = useQuery({
    queryKey: ["payments", child.student_id],
    queryFn: () => paymentsApi.list({ student_id: child.student_id }),
  });
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState("");
  const target = useRef<HTMLLIElement | null>(null);
  const rows = ((data as any[]) ?? []).filter((p) => p.status === "confirmed");

  useEffect(() => {
    target.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, [data]);

  async function download(p: any) {
    setBusy(p.id);
    setError("");
    try {
      await paymentsApi.downloadReceipt(p.id, p.receipt_no);
    } catch {
      setError("Could not download that receipt. Try again.");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      {showName && <p className="mb-1 text-sm font-medium text-slate-600">{child.full_name}</p>}
      {isLoading && <div className="h-16 animate-pulse rounded-2xl bg-slate-200" />}
      {!isLoading && rows.length === 0 && <p className="rounded-2xl border border-dashed border-slate-200 p-4 text-center text-sm text-slate-400">No payments yet.</p>}
      {error && <p className="mb-2 text-sm text-red-600">{error}</p>}
      <ul className="space-y-2">
        {rows.map((p) => {
          const on = p.id === highlight;
          return (
            <li key={p.id} ref={on ? target : undefined}
              className={`flex items-center gap-3 rounded-2xl border bg-white p-3 shadow-sm ${on ? "border-[#4a9eca] ring-2 ring-[#4a9eca]/30" : "border-slate-200"}`}>
              <span className="rounded-xl bg-[#4a9eca]/10 p-2 text-[#1b365d]"><Receipt size={18} /></span>
              <div className="min-w-0 flex-1">
                <p className="font-semibold text-[#1b365d]">{kes(p.amount)}</p>
                <p className="truncate text-xs text-slate-500">
                  {new Date(p.payment_date).toLocaleDateString("en-KE", { day: "numeric", month: "short", year: "numeric" })}
                  {" · "}{String(p.payment_method).replace("_", " ")} · {p.receipt_no}
                </p>
              </div>
              <button onClick={() => download(p)} disabled={busy === p.id}
                className="inline-flex items-center gap-1 rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50">
                <Download size={14} /> {busy === p.id ? "…" : "PDF"}
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
