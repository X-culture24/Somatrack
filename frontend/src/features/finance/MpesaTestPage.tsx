import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { paymentsApi } from "../../api/finance";
import { studentsApi } from "../../api/students";
import { Button, Card, Field, Input, PageHeader, Select, useConfirm } from "../../components/ui";

export function MpesaTestPage() {
  const { confirm, dialog } = useConfirm();
  const { data: students } = useQuery({ queryKey: ["students", ""], queryFn: () => studentsApi.list() });
  const [studentId, setStudentId] = useState("");
  const [amount, setAmount] = useState("5000");
  const [phone, setPhone] = useState("254708374149");
  const [payerName, setPayerName] = useState("");
  const [result, setResult] = useState<any>(null);

  const selected: any = (students ?? []).find((s: any) => s.id === studentId);
  // The real listener matches on the payer's name in the M-Pesa account
  // reference, not the student id — so we send the learner's name, exactly
  // like a parent typing it into the Paybill account-number field.
  const billRef = selected ? `${selected.first_name} ${selected.last_name}` : "";

  const fire = useMutation({
    mutationFn: () =>
      paymentsApi.simulate({
        amount: Number(amount),
        bill_ref_number: billRef,
        phone,
        payer_name: payerName || undefined,
      }),
    onSuccess: (data) => setResult(data),
  });

  async function handleFire() {
    if (!selected) return;
    const ok = await confirm({
      title: "Simulate M-Pesa payment",
      message: `Simulate KES ${amount} paid for ${selected.full_name}? This runs the real fee-listener pipeline (match → post payment → update invoice).`,
      confirmLabel: "Fire payment",
    });
    if (ok) { setResult(null); fire.mutate(); }
  }

  return (
    <div className="max-w-xl space-y-5">
      {dialog}
      <PageHeader
        title="Simulate a fee payment"
        subtitle="Fires a Safaricom-shaped payment through the real listener pipeline — for demos and manual QA, without needing an actual M-Pesa transaction."
      />

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Payment details</h3>
        <div className="grid gap-3">
          <Field label="Learner">
            <Select value={studentId} onChange={(e) => setStudentId(e.target.value)}>
              <option value="">Select learner…</option>
              {(students ?? []).map((s: any) => (
                <option key={s.id} value={s.id}>{s.full_name} ({s.admission_no})</option>
              ))}
            </Select>
          </Field>
          <Field label="Amount (KES)">
            <Input type="number" value={amount} onChange={(e) => setAmount(e.target.value)} />
          </Field>
          <Field label="Payer phone (M-Pesa MSISDN)">
            <Input value={phone} onChange={(e) => setPhone(e.target.value)} />
          </Field>
          <Field label="Payer name (optional, for the payment record)">
            <Input value={payerName} onChange={(e) => setPayerName(e.target.value)} placeholder="e.g. parent's name" />
          </Field>
        </div>

        {billRef && (
          <p className="mt-3 text-xs text-slate-500">
            Will send account reference <span className="font-mono font-medium text-slate-700">"{billRef}"</span> —
            the same field a parent enters at a Paybill terminal.
          </p>
        )}

        <div className="mt-4 rounded-xl bg-slate-50 p-3 text-xs text-slate-500">
          <p className="mb-1 font-medium text-slate-600">How it works</p>
          <ol className="list-decimal list-inside space-y-0.5">
            <li>Sends a Safaricom-shaped confirmation to the real payment service.</li>
            <li>Event recorded, then matched by fuzzy name match (≥80% confidence) against active students.</li>
            <li>If matched → posts a payment and recomputes the student's invoice balance atomically.</li>
            <li>If unmatched → the event is flagged for manual review instead.</li>
          </ol>
        </div>

        <Button className="mt-4 w-full" onClick={handleFire} disabled={fire.isPending || !selected}>
          {fire.isPending ? "Firing…" : "Fire simulated payment"}
        </Button>
      </Card>

      {result && (
        <Card>
          {result.reconciliation_status === "posted" ? (
            <p className="mb-2 font-semibold text-green-600">Payment matched and posted</p>
          ) : (
            <p className="mb-2 font-semibold text-yellow-600">Payment recorded but not matched — check Unreconciled</p>
          )}
          <dl className="space-y-1 text-sm">
            <div className="flex justify-between"><dt className="text-slate-500">Payment ID</dt><dd className="font-mono text-xs">{result.payment_id ?? "—"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Invoice ID</dt><dd className="font-mono text-xs">{result.invoice_id ?? "—"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Status</dt><dd>{result.reconciliation_status}</dd></div>
          </dl>
          <p className="mt-3 text-xs text-slate-400">
            Open Invoices to see the updated balance — it auto-refreshes every 5 seconds.
          </p>
        </Card>
      )}
    </div>
  );
}
