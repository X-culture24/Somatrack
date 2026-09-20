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
  updateStaff: (id: number, data: unknown) =>
    api.patch(`/staff/${id}/`, data).then((r) => r.data),
  departments: () => api.get("/departments/").then((r) => unwrap(r.data)),
  createDepartment: (data: unknown) => api.post("/departments/", data).then((r) => r.data),
  users: (params?: Record<string, string>) =>
    api.get("/users/", { params }).then((r) => unwrap(r.data)),
  createUser: (data: unknown) => api.post("/users/", data).then((r) => r.data),
  updateUser: (id: number, data: unknown) =>
    api.patch(`/users/${id}/`, data).then((r) => r.data),
  announcements: () => api.get("/announcements/").then((r) => unwrap(r.data)),
  createAnnouncement: (data: unknown) => api.post("/announcements/", data).then((r) => r.data),
  updateAnnouncement: (id: number, data: unknown) =>
    api.patch(`/announcements/${id}/`, data).then((r) => r.data),
  deleteAnnouncement: (id: number) => api.delete(`/announcements/${id}/`).then((r) => r.data),
  transport: () => api.get("/transport-routes/").then((r) => unwrap(r.data)),
  createTransportRoute: (data: unknown) => api.post("/transport-routes/", data).then((r) => r.data),
  updateTransportRoute: (id: number, data: unknown) =>
    api.patch(`/transport-routes/${id}/`, data).then((r) => r.data),
  transportAttendance: (params?: Record<string, string>) =>
    api.get("/transport-attendance/", { params }).then((r) => unwrap(r.data)),
  vehicles: (params?: Record<string, string>) =>
    api.get("/vehicles/", { params }).then((r) => unwrap(r.data)),
  createVehicle: (data: unknown) => api.post("/vehicles/", data).then((r) => r.data),
  updateVehicle: (id: number, data: unknown) =>
    api.patch(`/vehicles/${id}/`, data).then((r) => r.data),
  library: () => api.get("/library/books/").then((r) => unwrap(r.data)),
  libraryLoans: (params?: Record<string, string>) =>
    api.get("/library/loans/", { params }).then((r) => unwrap(r.data)),
  returnBook: (loanId: number) =>
    api.post(`/library/loans/${loanId}/return_book/`).then((r) => r.data),
  renewLoan: (loanId: number, extraDays = 7) =>
    api.post(`/library/loans/${loanId}/renew/`, { extra_days: extraDays }).then((r) => r.data),
  overdueLoans: () =>
    api.get("/library/loans/overdue/").then((r) => r.data),
  inventory: () => api.get("/inventory/").then((r) => unwrap(r.data)),
  createInventoryItem: (data: unknown) => api.post("/inventory/", data).then((r) => r.data),
  updateInventoryItem: (id: number, data: unknown) =>
    api.patch(`/inventory/${id}/`, data).then((r) => r.data),
  dashboard: () => api.get("/dashboard/").then((r) => r.data),
  // Reports
  reportAttendance: (params?: Record<string, string>) =>
    api.get("/reports/attendance/", { params }).then((r) => r.data),
  reportGrades: (params?: Record<string, string>) =>
    api.get("/reports/grades/", { params }).then((r) => r.data),
  reportFeeCollection: () =>
    api.get("/reports/fee-collection/").then((r) => r.data),
  reportClassPerformance: (classId: string) =>
    api.get("/reports/class-performance/", { params: { class_group: classId } }).then((r) => r.data),
};
