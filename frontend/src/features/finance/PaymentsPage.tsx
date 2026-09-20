import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { paymentsApi } from "../../api/finance";
import { studentsApi } from "../../api/students";
import { Button, Card, Field, Input, PageHeader, Select, Table, kes, useConfirm } from "../../components/ui";

export function PaymentsPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();

  const { data: payments } = useQuery({ queryKey: ["payments"], queryFn: () => paymentsApi.list() });
  const { data: students } = useQuery({ queryKey: ["students", ""], queryFn: () => studentsApi.list() });
  const [form, setForm] = useState({ student: "", amount: "", method: "cash", notes: "" });

  const save = useMutation({
    mutationFn: () => paymentsApi.manual(form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["payments"] });
      setForm({ student: "", amount: "", method: "cash", notes: "" });
    },
  });

  async function handlePost() {
    const student = (students as any[] ?? []).find((s: any) => String(s.id) === form.student);
    const ok = await confirm({
      title: "Post payment",
      message: `Post ${kes(form.amount)} (${form.method}) for ${student?.full_name ?? "selected learner"}? This will update their invoice balance.`,
      confirmLabel: "Post payment",
    });
    if (ok) save.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Payments" subtitle="Cash, bank, and M-Pesa all update the same invoice balance." />
      <Card>
        <h3 className="mb-3 font-semibold">Record cash / bank payment</h3>
        <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
          <Field label="Student">
            <Select value={form.student} onChange={(e) => setForm({ ...form, student: e.target.value })}>
              <option value="">Select…</option>
              {(students ?? []).map((s: any) => (
                <option key={s.id} value={s.id}>{s.full_name} ({s.admission_no})</option>
              ))}
            </Select>
          </Field>
          <Field label="Amount (KES)">
            <Input type="number" min="0" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} />
          </Field>
          <Field label="Method">
            <Select value={form.method} onChange={(e) => setForm({ ...form, method: e.target.value })}>
              <option value="cash">Cash</option>
              <option value="bank">Bank transfer</option>
              <option value="mpesa">M-Pesa (manual)</option>
            </Select>
          </Field>
          <Field label="Notes (optional)">
            <Input value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} placeholder="Receipt no., reference…" />
          </Field>
        </div>
        <div className="mt-4">
          <Button
            onClick={handlePost}
            disabled={save.isPending || !form.student || !form.amount}
          >
            {save.isPending ? "Posting…" : "Post payment"}
          </Button>
        </div>
      </Card>
      <Table headers={["Date", "Learner", "Method", "Amount", "Receipt / ref", "Status"]}>
        {(payments ?? []).map((p: any) => (
          <tr key={p.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 text-xs text-slate-500">{new Date(p.transaction_date).toLocaleString()}</td>
            <td className="px-4 py-3">{p.student_name || "—"}</td>
            <td className="px-4 py-3 uppercase text-xs font-medium">{p.method}</td>
            <td className="px-4 py-3 font-medium">{kes(p.amount)}</td>
            <td className="px-4 py-3 text-xs">{p.mpesa_receipt_no || p.paybill_account_ref || "—"}</td>
            <td className="px-4 py-3">
              <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                p.reconciliation_status === "posted" ? "bg-green-100 text-green-700" : "bg-yellow-100 text-yellow-700"
              }`}>
                {p.reconciliation_status}
              </span>
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
