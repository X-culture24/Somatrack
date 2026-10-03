import { api, unwrap } from "./client";

// Real routes (internal/api/finance_proxy.go): POST /payroll-runs/ both
// creates the run AND generates every active staff member's payslip in one
// call — there is no separate create/generate/approve/mark-paid workflow,
// and entries are listed under the run (/payroll-runs/{id}/entries/), not
// a flat /payroll-entries/?payroll=X the old Django-shaped client assumed.
export const payrollApi = {
  runs: () => api.get("/payroll-runs/").then((r) => unwrap(r.data)),
  createRun: (data: { month: number; year: number }) =>
    api.post("/payroll-runs/", data).then((r) => r.data as { run: any; entries: any[] }),
  entries: (runId: string) =>
    api.get(`/payroll-runs/${runId}/entries/`).then((r) => unwrap(r.data)),
  disburse: (entryId: string) =>
    api.post(`/payroll-entries/${entryId}/disburse/`).then((r) => r.data),
  disburseEquity: (entryId: string) =>
    api.post(`/payroll-entries/${entryId}/disburse/equity/`).then((r) => r.data),
};
