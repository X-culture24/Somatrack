import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, type FormEvent } from "react";
import { downloadFile } from "../../api/client";
import { paymentsApi, type ManualPaymentInput } from "../../api/finance";
import { formErrors } from "../../api/resources";
import { studentsApi } from "../../api/students";
import { Badge, Button, Card, Field, Input, PageHeader, Select, Table, kes } from "../../components/ui";

export function PaymentsPage() {
  const { data: payments } = useQuery({ queryKey: ["payments"], queryFn: () => paymentsApi.list() });

  return (
    <div className="space-y-6">
      <PageHeader
        title="Payments"
        subtitle="M-Pesa and Equity payments post automatically. Record cash, cheque and bank-slip payments here. Every payment updates the invoice and texts the family a receipt link."
      />
      <RecordPayment />
      <Table headers={["Date", "Receipt", "Payer", "Method", "Amount", "Status", ""]}>
        {(payments ?? []).length === 0 && (
          <tr><td colSpan={7} className="px-4 py-8 text-center text-sm text-slate-400">
            No payments recorded yet — try Simulate Payment.
          </td></tr>
        )}
        {(payments ?? []).map((p: any) => (
          <tr key={p.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 text-xs text-slate-500">{new Date(p.payment_date).toLocaleDateString()}</td>
            <td className="px-4 py-3 font-mono text-xs">{p.receipt_no}</td>
            <td className="px-4 py-3">{p.payer_name || "—"}</td>
            <td className="px-4 py-3 uppercase text-xs font-medium">{p.payment_method}</td>
            <td className="px-4 py-3 font-medium">{kes(p.amount)}</td>
            <td className="px-4 py-3">
              <Badge variant={p.status === "confirmed" ? "success" : "warning"}>{p.status}</Badge>
            </td>
            <td className="px-4 py-3">
              <Button
                variant="ghost"
                onClick={() => downloadFile(`/payments/${p.id}/receipt/`, `receipt-${p.receipt_no}.pdf`)}
              >
                Download receipt
              </Button>
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}

const METHODS: [ManualPaymentInput["payment_method"], string][] = [
  ["cash", "Cash"], ["cheque", "Cheque"], ["bank_transfer", "Bank deposit / transfer"], ["card", "Card"], ["other", "Other"],
];

// Counter payments at the bursar's desk. The idempotency key is fixed per
// form, so a double-click or a retry records the money once.
function RecordPayment() {
  const qc = useQueryClient();
  const blank = () => ({ student_id: "", amount: "", payment_method: "cash" as ManualPaymentInput["payment_method"], payment_date: "",
    reference: "", bank_name: "", cheque_no: "", payer_name: "", payer_phone: "", notes: "", idempotency_key: crypto.randomUUID() });
  const [open, setOpen] = useState(false);
  const [f, setF] = useState(blank);
  const [search, setSearch] = useState("");
  const [err, setErr] = useState<{ message: string; fields: Record<string, string> } | null>(null);
  const [done, setDone] = useState<{ payment_id: string; receipt_no: string } | null>(null);

  const { data: students } = useQuery({
    queryKey: ["students-search", search],
    enabled: open && search.trim().length >= 2,
    queryFn: () => studentsApi.list({ search }),
  });

  const save = useMutation({
    mutationFn: () => paymentsApi.recordManual({ ...f, amount: Number(f.amount) }),
    onSuccess: (res) => {
      setDone(res);
      setF(blank());
      setSearch("");
      qc.invalidateQueries({ queryKey: ["payments"] });
      qc.invalidateQueries({ queryKey: ["invoices"] });
    },
    onError: (e) => setErr(formErrors(e)),
  });

  const set = (k: keyof ReturnType<typeof blank>) => (e: { target: { value: string } }) => setF({ ...f, [k]: e.target.value });
  const fe = (k: string) => err?.fields[k] && <p className="mt-1 text-xs text-red-600">{err.fields[k]}</p>;

  function submit(e: FormEvent) {
    e.preventDefault();
    setErr(null);
    setDone(null);
    save.mutate();
  }

  if (!open) {
    return (
      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={() => setOpen(true)}>Record a payment</Button>
        {done && <ReceiptDone done={done} />}
      </div>
    );
  }

  return (
    <Card>
      <form onSubmit={submit} className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold text-[#1b365d]">Record a payment</h3>
          <button type="button" className="text-sm text-slate-500 hover:underline" onClick={() => { setOpen(false); setErr(null); }}>Close</button>
        </div>
        {err && !Object.keys(err.fields).length && <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">{err.message}</p>}
        <div className="grid gap-3 md:grid-cols-3">
          <div className="md:col-span-2">
            <Field label="Student *">
              <Input placeholder="Type a name or admission number…" value={search} onChange={(e) => setSearch(e.target.value)} />
              <Select required value={f.student_id} onChange={set("student_id")} className="mt-1">
                <option value="">{search.trim().length < 2 ? "Search above first" : "Choose student…"}</option>
                {((students as any[]) ?? []).map((s: any) => <option key={s.id} value={s.id}>{s.full_name} ({s.admission_no}){s.class_name ? ` · ${s.class_name}` : ""}</option>)}
              </Select>
            </Field>
            {fe("student_id")}
          </div>
          <div>
            <Field label="Amount (KES) *"><Input required type="number" min="1" step="0.01" value={f.amount} onChange={set("amount")} /></Field>
            {fe("amount")}
          </div>
          <div>
            <Field label="Paid by">
              <Select value={f.payment_method} onChange={set("payment_method")}>
                {METHODS.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
              </Select>
            </Field>
            {fe("payment_method")}
          </div>
          {f.payment_method === "cheque" && (
            <div><Field label="Cheque no. *"><Input required value={f.cheque_no} onChange={set("cheque_no")} /></Field>{fe("cheque_no")}</div>
          )}
          {(f.payment_method === "cheque" || f.payment_method === "bank_transfer") && (
            <div><Field label="Bank"><Input value={f.bank_name} onChange={set("bank_name")} placeholder="e.g. Equity Bank" /></Field></div>
          )}
          <div>
            <Field label={f.payment_method === "bank_transfer" ? "Bank slip / transaction no. *" : "Reference"}>
              <Input required={f.payment_method === "bank_transfer"} value={f.reference} onChange={set("reference")} />
            </Field>
            {fe("reference")}
          </div>
          <div><Field label="Date paid"><Input type="date" value={f.payment_date} onChange={set("payment_date")} /></Field>{fe("payment_date")}</div>
          <div><Field label="Paid in by"><Input value={f.payer_name} onChange={set("payer_name")} placeholder="Name of the person paying" /></Field></div>
          <div><Field label="Their phone"><Input type="tel" value={f.payer_phone} onChange={set("payer_phone")} /></Field></div>
          <div className="md:col-span-3"><Field label="Notes"><Input value={f.notes} onChange={set("notes")} /></Field></div>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit" disabled={save.isPending}>{save.isPending ? "Recording…" : "Record payment & text receipt"}</Button>
          {done && <ReceiptDone done={done} />}
        </div>
      </form>
    </Card>
  );
}

function ReceiptDone({ done }: { done: { payment_id: string; receipt_no: string } }) {
  return (
    <span className="flex items-center gap-2 text-sm text-green-700">
      Recorded {done.receipt_no}. The family has been sent a receipt link.
      <button type="button" className="font-semibold underline" onClick={() => paymentsApi.downloadReceipt(done.payment_id, done.receipt_no)}>Print receipt</button>
    </span>
  );
}
