import { api, unwrap } from "./client";

export const invoicesApi = {
  list: (params?: Record<string, string>) =>
    api.get("/invoices/", { params }).then((r) => unwrap(r.data)),
  generateAll: () => api.post("/invoices/generate_all/").then((r) => r.data),
  addAdhoc: (id: number, data: unknown) =>
    api.post(`/invoices/${id}/add_adhoc/`, data).then((r) => r.data),
  statement: (id: number) =>
    api.get(`/invoices/${id}/statement/`).then((r) => r.data),
};

export const paymentsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/payments/", { params }).then((r) => unwrap(r.data)),
  manual: (data: unknown) =>
    api.post("/finance/manual-payment/", data).then((r) => r.data),
  reconciliation: () =>
    api.get("/finance/reconciliation/").then((r) => r.data),
  sendReminders: () =>
    api.post("/finance/send-reminders/").then((r) => r.data),
};

export const unreconciledApi = {
  list: () => api.get("/unreconciled-payments/").then((r) => unwrap(r.data)),
  link: (id: number, student: string) =>
    api.post(`/unreconciled-payments/${id}/link/`, { student }).then((r) => r.data),
};

export const feeStructuresApi = {
  list: (params?: Record<string, string>) =>
    api.get("/fee-structures/", { params }).then((r) => unwrap(r.data)),
};
