import { api, unwrap } from "./client";

export const schoolApi = {
  profile: () => api.get("/school/profile/").then((r) => r.data),
  updateProfile: (data: unknown) =>
    api.put("/school/profile/", data).then((r) => r.data),
  curriculum: (grade?: string) =>
    api.get("/school/curriculum/", { params: grade ? { grade } : {} }).then((r) => r.data as any[]),
  gradeRequirements: (grade?: string) =>
    api.get("/grade-requirements/", { params: grade ? { grade } : {} }).then((r) => unwrap(r.data)),
  staff: (params?: Record<string, string>) =>
    api.get("/staff/", { params }).then((r) => unwrap(r.data)),
  createStaff: (data: unknown) => api.post("/staff/", data).then((r) => r.data),
  updateStaff: (id: string, data: unknown) =>
    api.patch(`/staff/${id}/`, data).then((r) => r.data),
  departments: () => api.get("/departments/").then((r) => unwrap(r.data)),
  createDepartment: (data: unknown) => api.post("/departments/", data).then((r) => r.data),
  users: (params?: Record<string, string>) =>
    api.get("/users/", { params }).then((r) => unwrap(r.data)),
  // Returns { user, temporary_password? } — the temp password (when the
  // caller left password blank) is only ever shown this once, since the
  // backend only stores its bcrypt hash from here on.
  createUser: (data: unknown) =>
    api.post("/users/", data).then((r) => r.data as { user: any; temporary_password?: string }),
  updateUser: (id: string, data: unknown) =>
    api.patch(`/users/${id}/`, data).then((r) => r.data),
  deleteUser: (id: string) => api.delete(`/users/${id}/`).then((r) => r.data),
  announcements: (audience?: string) =>
    api.get("/announcements/", { params: audience ? { audience } : {} }).then((r) => unwrap(r.data)),
  createAnnouncement: (data: unknown) => api.post("/announcements/", data).then((r) => r.data),
  updateAnnouncement: (id: string, data: unknown) =>
    api.patch(`/announcements/${id}/`, data).then((r) => r.data),
  deleteAnnouncement: (id: string) => api.delete(`/announcements/${id}/`).then((r) => r.data),
  transport: () => api.get("/transport-routes/").then((r) => unwrap(r.data)),
  createTransportRoute: (data: unknown) => api.post("/transport-routes/", data).then((r) => r.data),
  updateTransportRoute: (id: string, data: unknown) =>
    api.patch(`/transport-routes/${id}/`, data).then((r) => r.data),
  transportAttendance: (params?: Record<string, string>) =>
    api.get("/transport-attendance/", { params }).then((r) => unwrap(r.data)),
  vehicles: (params?: Record<string, string>) =>
    api.get("/vehicles/", { params }).then((r) => unwrap(r.data)),
  createVehicle: (data: unknown) => api.post("/vehicles/", data).then((r) => r.data),
  updateVehicle: (id: string, data: unknown) =>
    api.patch(`/vehicles/${id}/`, data).then((r) => r.data),
  deleteVehicle: (id: string) => api.delete(`/vehicles/${id}/`).then((r) => r.data),
  library: () => api.get("/library/books/").then((r) => unwrap(r.data)),
  createBook: (data: unknown) => api.post("/library/books/", data).then((r) => r.data),
  updateBook: (id: string, data: unknown) => api.patch(`/library/books/${id}/`, data).then((r) => r.data),
  deleteBook: (id: string) => api.delete(`/library/books/${id}/`).then((r) => r.data),
  libraryLoans: (params?: Record<string, string>) =>
    api.get("/library/loans/", { params }).then((r) => unwrap(r.data)),
  returnBook: (loanId: string) =>
    api.post(`/library/loans/${loanId}/return/`).then((r) => r.data),
  renewLoan: (loanId: string, extraDays = 7) =>
    api.post(`/library/loans/${loanId}/renew/`, { extra_days: extraDays }).then((r) => r.data),
  overdueLoans: () =>
    api.get("/library/loans/overdue/").then((r) => unwrap(r.data)),
  inventory: () => api.get("/inventory/").then((r) => unwrap(r.data)),
  createInventoryItem: (data: unknown) => api.post("/inventory/", data).then((r) => r.data),
  updateInventoryItem: (id: string, data: unknown) =>
    api.patch(`/inventory/${id}/`, data).then((r) => r.data),
  dashboard: () => api.get("/dashboard/").then((r) => r.data),
  // Reports — internal/api/reports.go (real, proxies to report-svc)
  reportAttendance: (params: { start_date: string; end_date: string; class_id?: string }) =>
    api.get("/reports/attendance/", { params }).then((r) => r.data as any[]),
  reportGrades: (params: { term_id: string; class_id?: string; subject_id?: string }) =>
    api.get("/reports/grades/", { params }).then((r) => r.data as any[]),
  reportFeeCollection: () =>
    api.get("/reports/fee-collection/").then((r) => r.data as any[]),
  reportClassPerformance: (termId: string, classId: string) =>
    api.get("/reports/class-performance/", { params: { term_id: termId, class_id: classId } }).then((r) => r.data as any[]),
};
