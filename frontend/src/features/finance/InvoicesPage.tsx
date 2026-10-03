import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { downloadFile } from "../../api/client";
import { invoicesApi } from "../../api/finance";
import { studentsApi } from "../../api/students";
import { Badge, Button, Card, Field, PageHeader, Select, Table, kes } from "../../components/ui";

const STATUS_VARIANT: Record<string, "success" | "warning" | "danger" | "default"> = {
  paid: "success",
  partial: "warning",
  unpaid: "danger",
  overdue: "danger",
  cancelled: "default",
  refunded: "default",
};

export function InvoicesPage() {
  const [studentId, setStudentId] = useState("");
  const { data: students } = useQuery({ queryKey: ["students", ""], queryFn: () => studentsApi.list() });
  const { data: invoices, dataUpdatedAt, refetch, isFetching } = useQuery({
    queryKey: ["invoices", studentId],
    queryFn: () => invoicesApi.list(studentId ? { student_id: studentId } : undefined),
    refetchInterval: 5000, // demo-friendly: balance updates show up within a few seconds of a payment landing
  });

  const studentsById = useMemo(() => {
    const m = new Map<string, any>();
    for (const s of (students ?? []) as any[]) m.set(s.id, s);
    return m;
  }, [students]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Invoices"
        subtitle="Auto-refreshes every 5s — fire a payment from Simulate Payment and watch the balance update here."
      />

      <Card>
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="w-64">
            <Field label="Filter by learner">
              <Select value={studentId} onChange={(e) => setStudentId(e.target.value)}>
                <option value="">All learners</option>
                {(students ?? []).map((s: any) => (
                  <option key={s.id} value={s.id}>{s.full_name} ({s.admission_no})</option>
                ))}
              </Select>
            </Field>
          </div>
          <p className="text-xs text-slate-400">
            {isFetching ? "Refreshing…" : `Last updated ${new Date(dataUpdatedAt).toLocaleTimeString()}`}
            {" · "}
            <button className="underline hover:text-slate-600" onClick={() => refetch()}>refresh now</button>
          </p>
        </div>
      </Card>

      <Table headers={["Invoice No.", "Learner", "Term", "Total due", "Paid", "Balance", "Status", ""]}>
        {(invoices ?? []).length === 0 && (
          <tr><td colSpan={8} className="px-4 py-8 text-center text-sm text-slate-400">
            No invoices yet — generate billing for a term from the Finance home page.
          </td></tr>
        )}
        {(invoices ?? []).map((inv: any) => {
          const student = studentsById.get(inv.student_id);
          return (
            <tr key={inv.id} className="hover:bg-slate-50">
              <td className="px-4 py-3 font-mono text-xs text-slate-500">{inv.invoice_no}</td>
              <td className="px-4 py-3">
                <span className="font-medium">{student?.full_name ?? inv.student_id}</span>
                {student && <span className="ml-1 text-xs text-slate-500">{student.admission_no}</span>}
              </td>
              <td className="px-4 py-3">{inv.term_name || "—"}</td>
              <td className="px-4 py-3">{kes(inv.total_due)}</td>
              <td className="px-4 py-3">{kes(inv.total_paid)}</td>
              <td className={`px-4 py-3 font-semibold ${Number(inv.balance) > 0 ? "text-red-600" : "text-green-600"}`}>
                {kes(inv.balance)}
              </td>
              <td className="px-4 py-3">
                <Badge variant={STATUS_VARIANT[inv.status] ?? "default"}>{inv.status}</Badge>
              </td>
              <td className="px-4 py-3">
                <Button
                  variant="ghost"
                  onClick={() => downloadFile(`/invoices/${inv.id}/pdf/`, `invoice-${inv.invoice_no}.pdf`)}
                >
                  Download PDF
                </Button>
              </td>
            </tr>
          );
        })}
      </Table>
    </div>
  );
}
