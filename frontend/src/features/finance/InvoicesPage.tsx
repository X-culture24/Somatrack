import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { invoicesApi } from "../../api/finance";
import { studentsApi } from "../../api/students";
import { Button, Card, Field, Input, PageHeader, Select, Table, kes, useConfirm } from "../../components/ui";

// Preset ad-hoc categories. "other" triggers a free-text description field.
const ADHOC_PRESETS = [
  { label: "School trip", value: "School trip" },
  { label: "Educational excursion", value: "Educational excursion" },
  { label: "Swimming lessons", value: "Swimming lessons" },
  { label: "Music / instrument hire", value: "Music / instrument hire" },
  { label: "Drama / performance fee", value: "Drama / performance fee" },
  { label: "Sports day levy", value: "Sports day levy" },
  { label: "Replacement ID / diary", value: "Replacement ID / diary" },
  { label: "Lost / damaged book", value: "Lost / damaged book" },
  { label: "Boarding / overnight fee", value: "Boarding / overnight fee" },
  { label: "Other (specify below)", value: "other" },
];

const EMPTY_ADHOC = { invoiceId: "", preset: "", customDesc: "", amount: "" };

export function InvoicesPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();

  const { data } = useQuery({ queryKey: ["invoices"], queryFn: () => invoicesApi.list() });
  const { data: students } = useQuery({ queryKey: ["students", ""], queryFn: () => studentsApi.list() });

  // When a student is selected, auto-pick their current invoice
  const [selectedStudent, setSelectedStudent] = useState("");
  const [adhoc, setAdhoc] = useState(EMPTY_ADHOC);

  // Resolve invoice id from student selection or manual entry
  const studentInvoice = (data ?? [] as any[]).find(
    (inv: any) => String(inv.student) === selectedStudent || String(inv.id) === selectedStudent
  );

  const resolvedInvoiceId = adhoc.invoiceId || (studentInvoice ? String(studentInvoice.id) : "");
  const resolvedDesc = adhoc.preset === "other" ? adhoc.customDesc : adhoc.preset;

  const add = useMutation({
    mutationFn: () =>
      invoicesApi.addAdhoc(Number(resolvedInvoiceId), {
        description: resolvedDesc,
        amount: adhoc.amount,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["invoices"] });
      setAdhoc(EMPTY_ADHOC);
      setSelectedStudent("");
    },
  });

  async function handleAddCharge() {
    if (!resolvedInvoiceId || !resolvedDesc || !adhoc.amount) return;
    const invoice = (data as any[] ?? []).find((inv: any) => String(inv.id) === resolvedInvoiceId);
    const ok = await confirm({
      title: "Add charge to invoice",
      message: `Add ${kes(adhoc.amount)} for "${resolvedDesc}" to ${invoice?.student_name ?? `Invoice #${resolvedInvoiceId}`}?`,
      confirmLabel: "Add charge",
    });
    if (ok) add.mutate();
  }

  async function handleGenerateAll() {
    const ok = await confirm({
      title: "Generate all term invoices",
      message: "This will create or overwrite invoices for every active learner in the current term. Continue?",
      confirmLabel: "Generate",
    });
    if (ok) {
      invoicesApi.generateAll().then(() => qc.invalidateQueries({ queryKey: ["invoices"] }));
    }
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader
        title="Invoices & statements"
        actions={
          <Button variant="ghost" onClick={handleGenerateAll}>
            Generate term invoices
          </Button>
        }
      />

      {/* ── Ad-hoc charge form ── */}
      <Card>
        <h3 className="mb-4 font-semibold text-[#1b365d]">Add ad-hoc charge</h3>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">

          {/* Step 1 – pick a learner (auto-resolves invoice) */}
          <Field label="Learner">
            <Select
              value={selectedStudent}
              onChange={(e) => {
                setSelectedStudent(e.target.value);
                setAdhoc((a) => ({ ...a, invoiceId: "" }));
              }}
            >
              <option value="">Select learner…</option>
              {(students ?? []).map((s: any) => (
                <option key={s.id} value={s.id}>
                  {s.full_name} ({s.admission_no})
                </option>
              ))}
            </Select>
          </Field>

          {/* Manual invoice ID override */}
          <Field label="Invoice ID (override)">
            <Input
              placeholder="Leave blank to use learner's current invoice"
              value={adhoc.invoiceId}
              onChange={(e) => setAdhoc({ ...adhoc, invoiceId: e.target.value })}
            />
          </Field>

          {/* Step 2 – pick a charge type */}
          <Field label="Charge type">
            <Select
              value={adhoc.preset}
              onChange={(e) => setAdhoc({ ...adhoc, preset: e.target.value, customDesc: "" })}
            >
              <option value="">Select charge type…</option>
              {ADHOC_PRESETS.map((p) => (
                <option key={p.value} value={p.value}>
                  {p.label}
                </option>
              ))}
            </Select>
          </Field>

          {/* Step 3 – custom description (only when "other" is chosen) */}
          {adhoc.preset === "other" && (
            <Field label="Specify description">
              <Input
                placeholder="e.g. Science fair materials"
                value={adhoc.customDesc}
                onChange={(e) => setAdhoc({ ...adhoc, customDesc: e.target.value })}
              />
            </Field>
          )}

          {/* Step 4 – amount */}
          <Field label="Amount (KES)">
            <Input
              type="number"
              min="0"
              placeholder="0"
              value={adhoc.amount}
              onChange={(e) => setAdhoc({ ...adhoc, amount: e.target.value })}
            />
          </Field>
        </div>

        {/* Summary line */}
        {resolvedDesc && adhoc.amount && resolvedInvoiceId && (
          <p className="mt-3 text-xs text-slate-500">
            Will add <span className="font-medium text-slate-700">{kes(adhoc.amount)}</span> for{" "}
            <span className="font-medium text-slate-700">"{resolvedDesc}"</span> to invoice{" "}
            <span className="font-medium text-slate-700">#{resolvedInvoiceId}</span>.
          </p>
        )}

        <div className="mt-4">
          <Button
            onClick={handleAddCharge}
            disabled={add.isPending || !resolvedInvoiceId || !resolvedDesc || !adhoc.amount}
          >
            {add.isPending ? "Adding…" : "Add charge"}
          </Button>
        </div>
      </Card>

      {/* ── Invoices table ── */}
      <Table headers={["ID", "Learner", "Term", "Due", "Paid", "Balance", "Status"]}>
        {(data ?? []).map((inv: any) => (
          <tr key={inv.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 text-slate-500">#{inv.id}</td>
            <td className="px-4 py-3">
              <span className="font-medium">{inv.student_name}</span>
              <span className="ml-1 text-xs text-slate-500">{inv.admission_no}</span>
            </td>
            <td className="px-4 py-3">{inv.term_label}</td>
            <td className="px-4 py-3">{kes(inv.total_due)}</td>
            <td className="px-4 py-3">{kes(inv.total_paid)}</td>
            <td className={`px-4 py-3 font-semibold ${Number(inv.balance) > 0 ? "text-red-600" : "text-green-600"}`}>
              {kes(inv.balance)}
            </td>
            <td className="px-4 py-3">
              <span
                className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                  inv.status === "paid"
                    ? "bg-green-100 text-green-700"
                    : inv.status === "partial"
                    ? "bg-yellow-100 text-yellow-700"
                    : inv.status === "void"
                    ? "bg-slate-100 text-slate-500"
                    : "bg-red-100 text-red-700"
                }`}
              >
                {inv.status}
              </span>
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
