import { api, unwrap } from "./client";

// IDs are UUID strings throughout the real backend, not the auto-increment
// integers this file used to assume.
export const studentsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/students/", { params }).then((r) => unwrap(r.data)),
  get: (id: string) => api.get(`/students/${id}/`).then((r) => r.data),
  create: (data: unknown) => api.post("/students/", data).then((r) => r.data),
  update: (id: string, data: unknown) =>
    api.patch(`/students/${id}/`, data).then((r) => r.data),
  linkGuardian: (studentId: string, data: unknown) =>
    api.post(`/students/${studentId}/guardians/`, data).then((r) => r.data),
  unlinkGuardian: (studentId: string, guardianId: string) =>
    api.delete(`/students/${studentId}/guardians/${guardianId}/`).then((r) => r.data),
  promote: (data: { student_ids: string[]; target_class_id: string }) =>
    api.post("/students/promote/", data).then((r) => r.data),
  feeArrears: (params?: Record<string, string>) =>
    api.get("/students/fee-arrears/", { params }).then((r) => unwrap(r.data)),
  siblings: () =>
    api.get("/guardians/siblings/").then((r) => unwrap(r.data)),
  ledger: (studentId: string) =>
    api.get("/student-ledger/", { params: { student_id: studentId } }).then((r) => r.data),
};

export const transfersApi = {
  list: (studentId?: string) =>
    api.get("/student-transfers/", { params: studentId ? { student_id: studentId } : {} }).then((r) => unwrap(r.data)),
  create: (data: { student_id: string; to_class_id?: string | null; reason?: string; transfer_date?: string }) =>
    api.post("/student-transfers/", data).then((r) => r.data),
};

export const guardiansApi = {
  list: (params?: Record<string, string>) =>
    api.get("/guardians/", { params }).then((r) => unwrap(r.data)),
  get: (id: string) => api.get(`/guardians/${id}/`).then((r) => r.data),
  create: (data: unknown) => api.post("/guardians/", data).then((r) => r.data),
  createLogin: (id: string) =>
    api.post(`/guardians/${id}/create-login/`).then((r) => r.data as { user_id: string; temporary_password: string }),
};

// admission_applications field names/status values match the real DB schema
// exactly (see internal/api/admissions.go) — the previous version of this
// file (pupil_first_name, seeking_grade as a string like "Grade 1",
// status: "approved"/"declined"/"enrolled") didn't correspond to any actual
// column or CHECK constraint on the table.
export const admissionsApi = {
  list: (params?: Record<string, string>) =>
    api.get("/admissions/", { params }).then((r) => unwrap(r.data)),
  get: (id: string) => api.get(`/admissions/${id}/`).then((r) => r.data),
  create: (data: unknown) => api.post("/admissions/", data).then((r) => r.data),
  update: (id: string, data: unknown) =>
    api.patch(`/admissions/${id}/`, data).then((r) => r.data),
  enroll: (id: string, data: { admission_no: string; class_id?: string; stream_id?: string }) =>
    api.post(`/admissions/${id}/enroll/`, data).then((r) => r.data),
};
