import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { unreconciledApi } from "../../api/finance";
import { studentsApi } from "../../api/students";
import { Button, PageHeader, Select, Table, kes, useConfirm } from "../../components/ui";

export function UnreconciledPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();

  const { data } = useQuery({ queryKey: ["unrec"], queryFn: unreconciledApi.list });
  const { data: students } = useQuery({ queryKey: ["students", ""], queryFn: () => studentsApi.list() });
  const [link, setLink] = useState<Record<number, string>>({});

  const mutate = useMutation({
    mutationFn: ({ id, student }: { id: number; student: string }) => unreconciledApi.link(id, student),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["unrec"] }),
  });

  async function handleLink(payment: any) {
    const studentId = link[payment.id];
    if (!studentId) return;
    const student = (students as any[] ?? []).find((s: any) => String(s.id) === studentId);
    const ok = await confirm({
      title: "Link payment to learner",
      message: `Link ${kes(payment.amount)} (receipt: ${payment.mpesa_receipt_no || payment.paybill_account_ref}) to ${student?.full_name ?? "selected learner"}? This will post the payment to their invoice.`,
      confirmLabel: "Link & post",
    });
    if (ok) mutate.mutate({ id: payment.id, student: studentId });
  }

  return (
    <div>
      {dialog}
      <PageHeader
        title="Unreconciled M-Pesa"
        subtitle="Fuzzy match failed — link to a learner. Money is never dropped."
      />
      {(data ?? []).length === 0 && (
        <p className="rounded-2xl border border-slate-200 bg-white px-4 py-8 text-center text-sm text-slate-500">
          No unreconciled payments. All M-Pesa receipts matched.
        </p>
      )}
      {(data ?? []).length > 0 && (
        <Table headers={["Amount", "Account ref", "Receipt", "Notes", "Assign to learner", ""]}>
          {(data ?? []).map((p: any) => (
            <tr key={p.id} className="hover:bg-slate-50">
              <td className="px-4 py-3 font-medium">{kes(p.amount)}</td>
              <td className="px-4 py-3 text-xs">{p.paybill_account_ref}</td>
              <td className="px-4 py-3 text-xs">{p.mpesa_receipt_no || "—"}</td>
              <td className="px-4 py-3 text-xs text-slate-500 max-w-[160px] truncate">{p.notes}</td>
              <td className="px-4 py-3 min-w-[180px]">
                <Select
                  value={link[p.id] ?? ""}
                  onChange={(e) => setLink({ ...link, [p.id]: e.target.value })}
                >
                  <option value="">Select learner</option>
                  {(students ?? []).map((s: any) => (
                    <option key={s.id} value={s.id}>{s.full_name}</option>
                  ))}
                </Select>
              </td>
              <td className="px-4 py-3">
                <Button
                  variant="gold"
                  disabled={!link[p.id] || mutate.isPending}
                  onClick={() => handleLink(p)}
                >
                  Link
                </Button>
              </td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  );
}
