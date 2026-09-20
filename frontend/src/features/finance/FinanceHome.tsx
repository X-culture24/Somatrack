import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { invoicesApi, paymentsApi } from "../../api/finance";
import { schoolApi } from "../../api/school";
import { Button, Card, PageHeader, Stat, kes, useConfirm } from "../../components/ui";

export function FinanceHome() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({ queryKey: ["recon"], queryFn: paymentsApi.reconciliation });
  const { data: dash } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const { data: recentPayments } = useQuery({ queryKey: ["payments", "recent"], queryFn: () => paymentsApi.list() });
  const [reminderResult, setReminderResult] = useState<{ invoices_reminded: number; skipped_no_guardian: number; skipped_already_sent_today: number } | null>(null);

  const generate = useMutation({
    mutationFn: invoicesApi.generateAll,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["invoices"] }),
  });
  const remind = useMutation({
    mutationFn: paymentsApi.sendReminders,
    onSuccess: (result) => setReminderResult(result),
  });

  async function handleGenerate() {
    const ok = await confirm({
      title: "Generate term invoices",
      message: "This will create or refresh invoices for every active learner in the current term. Existing invoices will be recalculated. Continue?",
      confirmLabel: "Generate invoices",
    });
    if (ok) generate.mutate();
  }

  async function handleRemind() {
    const ok = await confirm({
      title: "Send fee reminders",
      message: "This will log a fee reminder to every guardian with an outstanding balance on the current term's invoice. Guardians already reminded today are skipped.",
      confirmLabel: "Send reminders",
    });
    if (ok) remind.mutate();
  }

  return (
    <div>
      {dialog}
      <PageHeader
        title="Finance portal"
        subtitle="Paybill 247247 · Account 137101#CHILDSNAME · Equity Bank 1370263402101"
        actions={
          <div className="flex gap-2">
            <Button variant="ghost" onClick={handleRemind} disabled={remind.isPending}>
              {remind.isPending ? "Sending…" : "Send fee reminders"}
            </Button>
            <Button onClick={handleGenerate} disabled={generate.isPending}>
              {generate.isPending ? "Generating…" : "Generate term invoices"}
            </Button>
          </div>
        }
      />
      {reminderResult && (
        <p className="mb-4 rounded-lg bg-[#4a9eca]/10 px-3 py-2 text-sm text-[#1b365d]">
          Reminded {reminderResult.invoices_reminded} learner{reminderResult.invoices_reminded === 1 ? "" : "s"}'
          guardians. Skipped {reminderResult.skipped_already_sent_today} already reminded today and{" "}
          {reminderResult.skipped_no_guardian} with no guardian on file.
        </p>
      )}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat label="M-Pesa posted" value={kes(data?.mpesa_posted)} />
        <Stat label="All posted" value={kes(data?.all_posted)} />
        <Stat label="Unreconciled" value={kes(data?.unreconciled)} />
        <Stat label="Outstanding" value={kes(data?.outstanding_balances ?? dash?.outstanding_fees)} />
      </div>
      <Card className="mt-6">
        <h3 className="mb-3 font-semibold text-[#1b365d]">Recent payments</h3>
        {(recentPayments ?? []).length === 0 ? (
          <p className="text-sm text-slate-400">No payments recorded yet.</p>
        ) : (
          <ul className="space-y-2">
            {[...(recentPayments as any[] ?? [])]
              .sort((a: any, b: any) => new Date(b.transaction_date).getTime() - new Date(a.transaction_date).getTime())
              .slice(0, 8)
              .map((p: any) => (
                <li key={p.id} className="flex items-center justify-between border-b border-slate-100 pb-2 text-sm last:border-0 last:pb-0">
                  <span className="text-slate-700">{p.student_name ?? "Unmatched"} · <span className="uppercase text-xs text-slate-400">{p.method}</span></span>
                  <span className="flex items-center gap-3">
                    <span className="font-semibold text-[#1b365d]">{kes(p.amount)}</span>
                    <span className="text-xs text-slate-400">{new Date(p.transaction_date).toLocaleDateString()}</span>
                  </span>
                </li>
              ))}
          </ul>
        )}
      </Card>

      <Card className="mt-6">
        <h3 className="font-semibold text-[#1b365d]">Fee listener</h3>
        <p className="mt-2 text-sm text-slate-600">
          Webhooks at <code>/api/webhooks/mpesa/confirmation</code> persist the raw payload, then a background job
          matches the bill reference to a learner and updates the open term invoice atomically. Unmatched receipts
          appear under Unreconciled. Send header <code>X-Mpesa-Secret</code> in production.
        </p>
      </Card>
    </div>
  );
}
