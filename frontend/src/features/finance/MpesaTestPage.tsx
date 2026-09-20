import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { api } from "../../api/client";
import { Button, Card, Field, Input, PageHeader, useConfirm } from "../../components/ui";

export function MpesaTestPage() {
  const { confirm, dialog } = useConfirm();
  const [form, setForm] = useState({
    amount: "5000",
    account_ref: "137101#FAITHOTIENO",
    trans_id: "",
    phone: "254712345678",
  });
  const [result, setResult] = useState<any>(null);

  const fire = useMutation({
    mutationFn: () => api.post("/webhooks/mpesa/test/", form).then((r) => r.data),
    onSuccess: (data) => setResult(data),
  });

  async function handleFire() {
    const ok = await confirm({
      title: "Fire test M-Pesa payment",
      message: `Simulate KES ${form.amount} from account ref "${form.account_ref}". This will run the full fee listener pipeline.`,
      confirmLabel: "Fire webhook",
    });
    if (ok) { setResult(null); fire.mutate(); }
  }

  return (
    <div className="max-w-xl space-y-5">
      {dialog}
      <PageHeader
        title="M-Pesa fee listener test"
        subtitle="Simulate a Safaricom C2B confirmation to test the full webhook → match → balance update pipeline."
      />

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Simulated payment payload</h3>
        <div className="grid gap-3">
          <Field label="Amount (KES)">
            <Input type="number" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} />
          </Field>
          <Field label="BillRefNumber (account ref)">
            <Input value={form.account_ref} onChange={(e) => setForm({ ...form, account_ref: e.target.value })}
              placeholder="137101#CHILDNAME" />
          </Field>
          <Field label="TransID (leave blank for random)">
            <Input value={form.trans_id} onChange={(e) => setForm({ ...form, trans_id: e.target.value })}
              placeholder="e.g. TEST001 — blank = auto" />
          </Field>
          <Field label="MSISDN (payer phone)">
            <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          </Field>
        </div>

        <div className="mt-4 rounded-xl bg-slate-50 p-3 text-xs text-slate-500">
          <p className="font-medium text-slate-600 mb-1">How it works</p>
          <ol className="list-decimal list-inside space-y-0.5">
            <li>POST fires a Safaricom-shaped JSON to the confirmation endpoint.</li>
            <li>Event saved to WebhookEvent table with status <code>pending</code>.</li>
            <li>Background worker parses <code>BillRefNumber</code>, strips prefix <code>137101#</code>.</li>
            <li>Fuzzy-matches name against active students (threshold 80%).</li>
            <li>If matched → creates Payment + updates invoice balance atomically.</li>
            <li>If unmatched → appears in Unreconciled queue.</li>
          </ol>
        </div>

        <Button className="mt-4 w-full" onClick={handleFire} disabled={fire.isPending}>
          {fire.isPending ? "Firing…" : "🔥 Fire test webhook"}
        </Button>
      </Card>

      {result && (
        <Card>
          <p className="mb-2 font-semibold text-green-600">✓ Webhook queued</p>
          <dl className="space-y-1 text-sm">
            <div className="flex justify-between"><dt className="text-slate-500">Event ID</dt><dd className="font-mono font-medium">#{result.event_id}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">TransID</dt><dd className="font-mono">{result.trans_id}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Amount</dt><dd>KES {result.payload?.TransAmount}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Account ref</dt><dd>{result.payload?.BillRefNumber}</dd></div>
          </dl>
          <p className="mt-3 text-xs text-slate-400">
            Check Unreconciled Payments if the student name didn't match, or refresh Invoices to see the updated balance.
          </p>
          <div className="mt-3 rounded-xl bg-slate-50 p-3 text-xs font-mono text-slate-600 overflow-x-auto whitespace-pre">
            {JSON.stringify(result.payload, null, 2)}
          </div>
        </Card>
      )}
    </div>
  );
}
