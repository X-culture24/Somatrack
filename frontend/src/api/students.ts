import { api, unwrap } from "./client";

export const studentsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/students/", { params }).then((r) => unwrap(r.data)),
  get: (id: number) => api.get(`/students/${id}/`).then((r) => r.data),
  create: (data: unknown) => api.post("/students/", data).then((r) => r.data),
  update: (id: number, data: unknown) =>
    api.patch(`/students/${id}/`, data).then((r) => r.data),
  withdraw: (id: number) =>
    api.post(`/students/${id}/withdraw/`).then((r) => r.data),
  generateInvoice: (id: number) =>
    api.post(`/students/${id}/generate_invoice/`).then((r) => r.data),
};

export const guardiansApi = {
  list: (params?: Record<string, string>) =>
    api.get("/guardians/", { params }).then((r) => unwrap(r.data)),
};

export const admissionsApi = {
  list: () => api.get("/admissions/").then((r) => unwrap(r.data)),
  create: (data: unknown) => api.post("/admissions/", data).then((r) => r.data),
  update: (id: number, data: unknown) =>
    api.patch(`/admissions/${id}/`, data).then((r) => r.data),
  enroll: (id: number, data: unknown) =>
    api.post(`/admissions/${id}/enroll/`, data).then((r) => r.data),
};
