import { api, unwrap } from "./client";

export const academicsApi = {
  years: () => api.get("/academic-years/").then((r) => unwrap(r.data)),
  createYear: (data: unknown) => api.post("/academic-years/", data).then((r) => r.data),
  updateYear: (id: string, data: unknown) => api.patch(`/academic-years/${id}/`, data).then((r) => r.data),
  terms: () => api.get("/terms/").then((r) => unwrap(r.data)),
  createTerm: (data: unknown) => api.post("/terms/", data).then((r) => r.data),
  streams: () => api.get("/streams/").then((r) => unwrap(r.data)),
  createStream: (data: unknown) => api.post("/streams/", data).then((r) => r.data),
  classes: (params?: Record<string, string>) =>
    api.get("/classes/", { params }).then((r) => unwrap(r.data)),
  createClass: (data: unknown) => api.post("/classes/", data).then((r) => r.data),
  updateClass: (id: string, data: unknown) =>
    api.patch(`/classes/${id}/`, data).then((r) => r.data),
  subjects: (params?: Record<string, string>) =>
    api.get("/subjects/", { params }).then((r) => unwrap(r.data)),
  createSubject: (data: unknown) => api.post("/subjects/", data).then((r) => r.data),
  assessments: (params?: Record<string, string>) =>
    api.get("/assessments/", { params }).then((r) => unwrap(r.data)),
  createAssessment: (data: unknown) =>
    api.post("/assessments/", data).then((r) => r.data),
  marks: (params?: Record<string, string>) =>
    api.get("/marks/", { params }).then((r) => unwrap(r.data)),
  saveMark: (data: unknown) =>
    api.post("/marks/", data).then((r) => r.data),
  timetable: (params?: Record<string, string>) =>
    api.get("/timetable/", { params }).then((r) => unwrap(r.data)),
  attendance: (params?: Record<string, string>) =>
    api.get("/attendance/", { params }).then((r) => unwrap(r.data)),
  saveAttendance: (data: unknown) =>
    api.post("/attendance/", data).then((r) => r.data),
  lessonPlans: (params?: Record<string, string>) =>
    api.get("/lesson-plans/", { params }).then((r) => unwrap(r.data)),
  createLessonPlan: (data: unknown) =>
    api.post("/lesson-plans/", data).then((r) => r.data),
  updateLessonPlan: (id: string, data: unknown) =>
    api.patch(`/lesson-plans/${id}/`, data).then((r) => r.data),
  reviewLessonPlan: (id: string, status: "approved" | "reviewed" | "returned", comment: string) =>
    api.post(`/lesson-plans/${id}/review/`, { status, comment }).then((r) => r.data),
  lessonPlanCoverage: (params?: Record<string, string>) =>
    api.get("/lesson-plans/coverage/", { params }).then((r) => r.data),
  classPerformance: (classId: string, termId: string) =>
    api.get("/results/class-performance/", { params: { class_id: classId, term_id: termId } }).then((r) => r.data),
};
