import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Smartphone } from "lucide-react";
import { feePaymentApi } from "../../api/finance";
import { studentsApi } from "../../api/students";
import { Button, Card, Field, Input, PageHeader, Select } from "../../components/ui";

export function StkPushPage() {
  const { data: students } = useQuery({
    queryKey: ["students", "all"],
    queryFn: () => studentsApi.list(),
  });

  const [studentId, setStudentId]   = useState("");
  const [phone, setPhone]           = useState("0725926141");
  const [amount, setAmount]         = useState("");
  const [narrative, setNarrative]   = useState("School fees");
  const [result, setResult]         = useState<{ ok: boolean; msg: string } | null>(null);

  const selected: any = (students ?? []).find((s: any) => s.id === studentId);

  const push = useMutation({
    mutationFn: () =>
      feePaymentApi.initiateMpesa({
        student_id: studentId,
        phone,
        amount: Number(amount),
      }),
    onSuccess: () =>
      setResult({ ok: true, msg: `STK push sent to ${phone}. Ask the parent to enter their M-Pesa PIN.` }),
    onError: (err: any) =>
      setResult({ ok: false, msg: err?.response?.data?.message ?? "Request failed — check the phone number and try again." }),
  });

  function handleSend() {
    if (!studentId || !phone || Number(amount) < 1) return;
    setResult(null);
    push.mutate();
  }

  return (
    <div className="max-w-xl space-y-5">
      <PageHeader
        title="STK Push / Ad-hoc payment"
        subtitle="Send a real M-Pesa payment prompt to any phone — works for outstanding balances or one-off ad-hoc charges."
      />

      <Card>
        <div className="space-y-4">
          <Field label="Student">
            <Select value={studentId} onChange={(e) => setStudentId(e.target.value)}>
              <option value="">Select student…</option>
              {(students ?? []).map((s: any) => (
                <option key={s.id} value={s.id}>
                  {s.full_name} ({s.admission_no})
                </option>
              ))}
            </Select>
          </Field>

          {selected && (
            <div className="rounded-xl bg-slate-50 px-3 py-2 text-sm">
              <p className="text-slate-500 text-xs">Class · Balance</p>
              <p className="font-medium text-[#1b365d]">
                {selected.class_name ?? "—"} ·{" "}
                <span className={selected.fee_balance > 0 ? "text-red-600" : "text-green-600"}>
                  KES {Number(selected.fee_balance ?? 0).toLocaleString()}
                </span>
              </p>
            </div>
          )}

          <Field label="Amount (KES)">
            <Input
              type="number"
              min="1"
              placeholder="e.g. 5000"
              value={amount}
              onChange={(e) => { setAmount(e.target.value); setResult(null); }}
            />
          </Field>

          <Field label="Phone number (Safaricom)">
            <Input
              type="tel"
              placeholder="07XXXXXXXX"
              value={phone}
              onChange={(e) => { setPhone(e.target.value); setResult(null); }}
            />
          </Field>

          <Field label="Narrative / description">
            <Input
              value={narrative}
              onChange={(e) => setNarrative(e.target.value)}
              placeholder="e.g. School fees, Library fine, Uniform…"
            />
          </Field>
        </div>

        <div className="mt-4 rounded-xl bg-blue-50 px-3 py-2 text-xs text-blue-700">
          The parent's phone will receive an M-Pesa popup. Once they enter their PIN, the
          payment is captured automatically and posted against the student's account.
          Use any amount — even if the student has no outstanding balance (for ad-hoc charges).
        </div>

        <Button
          className="mt-4 w-full flex items-center justify-center gap-2"
          onClick={handleSend}
          disabled={push.isPending || !studentId || !phone || Number(amount) < 1}
        >
          <Smartphone size={16} />
          {push.isPending ? "Sending prompt…" : "Send M-Pesa prompt"}
        </Button>

        {result && (
          <div className={`mt-3 rounded-xl px-4 py-3 text-sm ${result.ok ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"}`}>
            {result.ok ? "✓ " : "✗ "}{result.msg}
          </div>
        )}
      </Card>
    </div>
  );
}
