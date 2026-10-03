import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { payrollApi } from "../../api/payroll";
import { Button, Card, Field, Input, PageHeader, Select, Stat, Table, kes, useConfirm } from "../../components/ui";

const MONTHS = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];

const DISBURSE_LABEL: Record<string, string> = {
  pending: "Disburse",
  failed: "Retry disburse",
  initiated: "In progress…",
  success: "Paid",
};

export function PayrollPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [selectedRun, setSelectedRun] = useState<string | null>(null);
  const [newRun, setNewRun] = useState({ month: String(new Date().getMonth() + 1), year: String(new Date().getFullYear()) });
  const [disbursingAll, setDisbursingAll] = useState(false);

  const { data: runs } = useQuery({ queryKey: ["payroll-runs"], queryFn: payrollApi.runs });
  const { data: entries, refetch: refetchEntries } = useQuery({
    queryKey: ["payroll-entries", selectedRun],
    enabled: Boolean(selectedRun),
    queryFn: () => payrollApi.entries(selectedRun!),
  });

  const create = useMutation({
    mutationFn: () => payrollApi.createRun({ month: Number(newRun.month), year: Number(newRun.year) }),
    onSuccess: ({ run }) => {
      qc.invalidateQueries({ queryKey: ["payroll-runs"] });
      setSelectedRun(run.id);
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not create this payroll run — one may already exist for that month/year.");
    },
  });

  const disburse = useMutation({
    mutationFn: (entryId: string) => payrollApi.disburse(entryId),
    onSuccess: () => refetchEntries(),
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Disbursement failed to initiate.");
    },
  });

  const disburseEquity = useMutation({
    mutationFn: (entryId: string) => payrollApi.disburseEquity(entryId),
    onSuccess: () => refetchEntries(),
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Equity EFT disbursement failed to initiate.");
    },
  });

  async function handleCreate() {
    const ok = await confirm({
      title: "Run payroll",
      message: `Compute gross/net pay for every active staff member for ${MONTHS[Number(newRun.month) - 1]} ${newRun.year}?`,
      confirmLabel: "Run payroll",
    });
    if (ok) create.mutate();
  }

  async function handleDisburseOne(entryId: string, name: string) {
    const ok = await confirm({
      title: "Disburse salary",
      message: `Send this net pay to ${name} via M-Pesa now? This cannot be undone.`,
      confirmLabel: "Disburse",
    });
    if (ok) disburse.mutate(entryId);
  }

  async function handleDisburseOneEquity(entryId: string, name: string) {
    const ok = await confirm({
      title: "Equity EFT disbursement",
      message: `Send this net pay to ${name} via Equity Bank EFT to their registered bank account? This cannot be undone.`,
      confirmLabel: "Disburse via Equity",
    });
    if (ok) disburseEquity.mutate(entryId);
  }

  async function handleDisburseAll() {
    const pending = ((entries as any[]) ?? []).filter((e) => e.disbursement_status === "pending" || e.disbursement_status === "failed");
    if (pending.length === 0) return;
    const ok = await confirm({
      title: "Disburse to everyone",
      message: `Send net pay to all ${pending.length} staff member(s) awaiting payment via M-Pesa? This triggers a real payout for each and cannot be undone.`,
      confirmLabel: `Disburse to ${pending.length}`,
    });
    if (!ok) return;
    setDisbursingAll(true);
    for (const e of pending) {
      try {
        await payrollApi.disburse(e.id);
      } catch {
        // one failure shouldn't stop the rest — each entry's own status reflects the outcome
      }
    }
    setDisbursingAll(false);
    refetchEntries();
  }

  const currentRun = (runs as any[] ?? []).find((r: any) => r.id === selectedRun);
  const pendingCount = ((entries as any[]) ?? []).filter((e) => e.disbursement_status === "pending" || e.disbursement_status === "failed").length;

  return (
    <div className="space-y-5">
      {dialog}
      <PageHeader title="Payroll" subtitle="Monthly staff payroll — gross, statutory deductions, and net pay disbursement via M-Pesa or Equity Bank EFT." />

      {/* Create new run */}
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Run payroll for a month</h3>
        <p className="mb-3 text-xs text-slate-500">
          This immediately computes gross/net pay for every active staff member — there's no separate "generate" step.
        </p>
        <div className="flex flex-wrap gap-3 items-end">
          <Field label="Month">
            <Select value={newRun.month} onChange={(e) => setNewRun({ ...newRun, month: e.target.value })} className="w-32">
              {MONTHS.map((m, i) => <option key={m} value={i + 1}>{m}</option>)}
            </Select>
          </Field>
          <Field label="Year">
            <Input type="number" value={newRun.year} onChange={(e) => setNewRun({ ...newRun, year: e.target.value })} className="w-24" />
          </Field>
          <Button onClick={handleCreate} disabled={create.isPending}>{create.isPending ? "Running…" : "Run payroll"}</Button>
        </div>
      </Card>

      {/* Runs table */}
      <Table headers={["Period", "Staff", "Total net pay", ""]}>
        {(runs as any[] ?? []).length === 0 && (
          <tr><td colSpan={4} className="px-4 py-8 text-center text-sm text-slate-400">No payroll runs yet.</td></tr>
        )}
        {(runs as any[] ?? []).map((r: any) => (
          <tr key={r.id} className={`hover:bg-slate-50 cursor-pointer ${selectedRun === r.id ? "bg-[#4a9eca]/5" : ""}`}
            onClick={() => setSelectedRun(r.id === selectedRun ? null : r.id)}>
            <td className="px-4 py-3 font-medium">{MONTHS[r.month - 1]} {r.year}</td>
            <td className="px-4 py-3">{r.staff_count}</td>
            <td className="px-4 py-3 font-semibold text-[#1b365d]">{kes(r.total_net_pay)}</td>
            <td className="px-4 py-3 text-xs text-[#4a9eca]">{selectedRun === r.id ? "▲ hide" : "▼ payslips"}</td>
          </tr>
        ))}
      </Table>

      {/* Payslip entries for selected run */}
      {selectedRun && entries && (
        <div>
          <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
            <h3 className="font-semibold text-[#1b365d]">
              Payslips — {currentRun ? `${MONTHS[currentRun.month - 1]} ${currentRun.year}` : ""}
            </h3>
            {pendingCount > 0 && (
              <Button onClick={handleDisburseAll} disabled={disbursingAll}>
                {disbursingAll ? "Disbursing…" : `Disburse to all (${pendingCount})`}
              </Button>
            )}
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4 mb-4">
            <Stat label="Staff on payroll" value={(entries as any[]).length} />
            <Stat label="Total gross" value={kes((entries as any[]).reduce((s, e) => s + Number(e.gross_pay), 0))} />
            <Stat label="Total deductions" value={kes((entries as any[]).reduce((s, e) => s + Number(e.deductions), 0))} />
            <Stat label="Total net" value={kes((entries as any[]).reduce((s, e) => s + Number(e.net_pay), 0))} />
          </div>
          {(entries as any[]).length === 0 ? (
            <p className="rounded-2xl border border-slate-200 bg-white py-8 text-center text-sm text-slate-400">
              No active staff members yet — add one on the Staff page first.
            </p>
          ) : (
            <Table headers={["Employee no.", "Name", "Basic", "Gross", "PAYE", "NSSF", "NHIF", "Deductions", "Net", "Disbursement", ""]}>
              {(entries as any[]).map((e: any) => (
                <tr key={e.id} className="hover:bg-slate-50">
                  <td className="px-3 py-2 text-xs">{e.employee_no}</td>
                  <td className="px-3 py-2 font-medium">{e.staff_name}</td>
                  <td className="px-3 py-2">{kes(e.basic_salary)}</td>
                  <td className="px-3 py-2 font-medium">{kes(e.gross_pay)}</td>
                  <td className="px-3 py-2 text-red-600">{kes(e.paye)}</td>
                  <td className="px-3 py-2 text-red-600">{kes(e.nssf)}</td>
                  <td className="px-3 py-2 text-red-600">{kes(e.nhif)}</td>
                  <td className="px-3 py-2 text-red-600">{kes(e.deductions)}</td>
                  <td className="px-3 py-2 font-bold text-[#1b365d]">{kes(e.net_pay)}</td>
                  <td className="px-3 py-2">
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                      e.disbursement_status === "success" ? "bg-green-100 text-green-700" :
                      e.disbursement_status === "failed" ? "bg-red-100 text-red-700" :
                      e.disbursement_status === "initiated" ? "bg-blue-100 text-blue-700" :
                      "bg-slate-100 text-slate-600"}`}>
                      {e.disbursement_status}
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    {(e.disbursement_status === "pending" || e.disbursement_status === "failed") && (
                      <div className="flex flex-col gap-1">
                        <Button variant="ghost" onClick={() => handleDisburseOne(e.id, e.staff_name)} disabled={disburse.isPending || disburseEquity.isPending} title="Pay via M-Pesa B2C">
                          {DISBURSE_LABEL[e.disbursement_status]} (M-Pesa)
                        </Button>
                        {e.bank_account_no && (
                          <Button variant="ghost" onClick={() => handleDisburseOneEquity(e.id, e.staff_name)} disabled={disburse.isPending || disburseEquity.isPending} title="Pay via Equity Bank EFT">
                            {DISBURSE_LABEL[e.disbursement_status]} (Equity EFT)
                          </Button>
                        )}
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </Table>
          )}
        </div>
      )}
    </div>
  );
}
