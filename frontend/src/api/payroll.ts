import { api, unwrap } from "./client";

export const payrollApi = {
  runs: (params?: Record<string, string>) =>
    api.get("/payroll-runs/", { params }).then((r) => unwrap(r.data)),
  run: (id: number) =>
    api.get(`/payroll-runs/${id}/`).then((r) => r.data),
  createRun: (data: unknown) =>
    api.post("/payroll-runs/", data).then((r) => r.data),
  generate: (id: number) =>
    api.post(`/payroll-runs/${id}/generate/`).then((r) => r.data),
  approve: (id: number) =>
    api.post(`/payroll-runs/${id}/approve/`).then((r) => r.data),
  markPaid: (id: number) =>
    api.post(`/payroll-runs/${id}/mark_paid/`).then((r) => r.data),
  entries: (params?: Record<string, string>) =>
    api.get("/payroll-entries/", { params }).then((r) => unwrap(r.data)),
  deductions: (params?: Record<string, string>) =>
    api.get("/staff-deductions/", { params }).then((r) => unwrap(r.data)),
};
