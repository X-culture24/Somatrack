import { api, unwrap } from "./client";

// internal/api/teacher.go — classes this logged-in teacher is homeroom or
// subject teacher for (class_groups.class_teacher_id / class_subjects.teacher_id).
export const teacherApi = {
  myClasses: () => api.get("/teacher/my-classes/").then((r) => unwrap(r.data)),
};

export type Assignment = { class_id: string; subject_id: string };

// /teachers/ — every teacher with the classes and subjects they teach.
// Assigning a subject here is what lets that teacher enter its marks and
// write its lesson plans.
export const teachersApi = {
  list: (params?: Record<string, string>) => api.get("/teachers/", { params }).then((r) => unwrap(r.data)),
  get: (id: string) => api.get(`/teachers/${id}/`).then((r) => r.data),
  // Replaces the teacher's full set; a 409 lists subjects another teacher
  // holds — resend with reassign: true to move them.
  replaceAssignments: (id: string, assignments: Assignment[], reassign = false) =>
    api.put(`/teachers/${id}/assignments/`, { assignments, reassign }).then((r) => r.data),
  // Adds to the teacher's set; same 409 / reassign contract as above.
  addAssignments: (id: string, assignments: Assignment[], reassign = false) =>
    api.post(`/teachers/${id}/assignments/`, { assignments, reassign }).then((r) => r.data),
  removeAssignment: (id: string, assignmentId: string) =>
    api.delete(`/teachers/${id}/assignments/${assignmentId}/`),
};
