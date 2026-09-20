import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { payrollApi } from "../../api/payroll";
import { Button, Card, Field, Input, PageHeader, Select, Stat, Table, kes, useConfirm } from "../../components/ui";

const MONTHS = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];

export function PayrollPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [selectedRun, setSelectedRun] = useState<number | null>(null);
  const [newRun, setNewRun] = useState({ month: String(new Date().getMonth() + 1), year: String(new Date().getFullYear()) });

  const { data: runs } = useQuery({ queryKey: ["payroll-runs"], queryFn: () => payrollApi.runs() });
  const { data: entries } = useQuery({
    queryKey: ["payroll-entries", selectedRun],
    enabled: Boolean(selectedRun),
    queryFn: () => payrollApi.entries({ payroll: String(selectedRun) }),
  });

  const create = useMutation({
    mutationFn: () => payrollApi.createRun(newRun),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["payroll-runs"] }),
  });
  const generate = useMutation({
    mutationFn: (id: number) => payrollApi.generate(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["payroll-runs"] });
      qc.invalidateQueries({ queryKey: ["payroll-entries", selectedRun] });
    },
  });
  const approve = useMutation({
    mutationFn: (id: number) => payrollApi.approve(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["payroll-runs"] }),
  });
  const markPaid = useMutation({
    mutationFn: (id: number) => payrollApi.markPaid(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["payroll-runs"] });
      qc.invalidateQueries({ queryKey: ["payroll-entries", selectedRun] });
    },
  });

  async function handleCreate() {
    const ok = await confirm({
      title: "Create payroll run",
      message: `Create payroll for ${MONTHS[Number(newRun.month) - 1]} ${newRun.year}?`,
      confirmLabel: "Create",
    });
    if (ok) create.mutate();
  }

  const currentRun = (runs as any[] ?? []).find((r: any) => r.id === selectedRun);

  async function handleGenerate() {
    if (!currentRun) return;
    const ok = await confirm({
      title: "Generate payslips",
      message: `Build payslips for every active staff member for ${MONTHS[currentRun.month - 1]} ${currentRun.year}? This replaces any existing entries on this run.`,
      confirmLabel: "Generate",
    });
    if (ok) generate.mutate(currentRun.id);
  }
  async function handleApprove() {
    if (!currentRun) return;
    const ok = await confirm({
      title: "Approve payroll",
      message: `Approve the ${MONTHS[currentRun.month - 1]} ${currentRun.year} payroll? PAYE must be filled in per payslip before approving.`,
      confirmLabel: "Approve",
    });
    if (ok) approve.mutate(currentRun.id);
  }
  async function handleMarkPaid() {
    if (!currentRun) return;
    const ok = await confirm({
      title: "Mark payroll as paid",
      message: `Mark the ${MONTHS[currentRun.month - 1]} ${currentRun.year} payroll as paid? This confirms bank/M-Pesa disbursement has gone out.`,
      confirmLabel: "Mark paid",
    });
    if (ok) markPaid.mutate(currentRun.id);
  }

  return (
    <div className="space-y-5">
      {dialog}
      <PageHeader title="Payroll" subtitle="Monthly staff payroll — gross, statutory deductions, and net pay." />

      {/* Create new run */}
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New payroll run</h3>
        <div className="flex flex-wrap gap-3 items-end">
          <Field label="Month">
            <Select value={newRun.month} onChange={(e) => setNewRun({ ...newRun, month: e.target.value })} className="w-32">
              {MONTHS.map((m, i) => <option key={m} value={i + 1}>{m}</option>)}
            </Select>
          </Field>
          <Field label="Year">
            <Input type="number" value={newRun.year} onChange={(e) => setNewRun({ ...newRun, year: e.target.value })} className="w-24" />
          </Field>
          <Button onClick={handleCreate} disabled={create.isPending}>Create run</Button>
        </div>
      </Card>

      {/* Runs table */}
      <Table headers={["Period", "Status", "Gross", "PAYE", "NSSF+NHIF", "Net", ""]}>
        {(runs as any[] ?? []).map((r: any) => (
          <tr key={r.id} className={`hover:bg-slate-50 cursor-pointer ${selectedRun === r.id ? "bg-[#4a9eca]/5" : ""}`}
            onClick={() => setSelectedRun(r.id === selectedRun ? null : r.id)}>
            <td className="px-4 py-3 font-medium">{MONTHS[r.month - 1]} {r.year}</td>
            <td className="px-4 py-3">
              <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                r.status === "paid" ? "bg-green-100 text-green-700" :
                r.status === "approved" ? "bg-blue-100 text-blue-700" :
                "bg-slate-100 text-slate-600"}`}>{r.status}</span>
            </td>
            <td className="px-4 py-3">{kes(r.total_gross)}</td>
            <td className="px-4 py-3">{kes(r.total_paye)}</td>
            <td className="px-4 py-3">{kes(Number(r.total_nssf) + Number(r.total_nhif))}</td>
            <td className="px-4 py-3 font-semibold text-[#1b365d]">{kes(r.total_net)}</td>
            <td className="px-4 py-3 text-xs text-[#4a9eca]">{selectedRun === r.id ? "▲ hide" : "▼ entries"}</td>
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
            <div className="flex gap-2">
              {currentRun?.status === "draft" && (
                <Button variant="ghost" onClick={handleGenerate} disabled={generate.isPending}>
                  {(entries as any[]).length > 0 ? "Regenerate payslips" : "Generate payslips"}
                </Button>
              )}
              {(currentRun?.status === "draft" || currentRun?.status === "reviewed") && (entries as any[]).length > 0 && (
                <Button onClick={handleApprove} disabled={approve.isPending}>Approve payroll</Button>
              )}
              {currentRun?.status === "approved" && (
                <Button onClick={handleMarkPaid} disabled={markPaid.isPending}>Mark as paid</Button>
              )}
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4 mb-4">
            <Stat label="Staff on payroll" value={(entries as any[]).length} />
            <Stat label="Total gross" value={kes((entries as any[]).reduce((s, e) => s + Number(e.gross_pay), 0))} />
            <Stat label="Total deductions" value={kes((entries as any[]).reduce((s, e) => s + Number(e.total_deductions), 0))} />
            <Stat label="Total net" value={kes((entries as any[]).reduce((s, e) => s + Number(e.net_pay), 0))} />
          </div>
          <Table headers={["Staff no.", "Name", "Basic", "Gross", "PAYE", "NSSF", "NHIF", "Other deductions", "Net", "Paid"]}>
            {(entries as any[]).map((e: any) => (
              <tr key={e.id} className="hover:bg-slate-50">
                <td className="px-3 py-2 text-xs">{e.staff_no}</td>
                <td className="px-3 py-2 font-medium">{e.staff_name}</td>
                <td className="px-3 py-2">{kes(e.basic_salary)}</td>
                <td className="px-3 py-2 font-medium">{kes(e.gross_pay)}</td>
                <td className="px-3 py-2 text-red-600">{kes(e.paye)}</td>
                <td className="px-3 py-2 text-red-600">{kes(e.nssf)}</td>
                <td className="px-3 py-2 text-red-600">{kes(e.nhif)}</td>
                <td className="px-3 py-2 text-red-600">{kes(Number(e.sacco)+Number(e.helb)+Number(e.other_deductions))}</td>
                <td className="px-3 py-2 font-bold text-[#1b365d]">{kes(e.net_pay)}</td>
                <td className="px-3 py-2">{e.paid ? "✅" : "⏳"}</td>
              </tr>
            ))}
          </Table>
        </div>
      )}
    </div>
  );
}
