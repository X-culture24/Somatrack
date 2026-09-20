import { api, unwrap } from "./client";

export async function openNova() {
  const { data } = await api.post("/auth/nova/handoff/");
  window.location.assign(data.url);
}

export const novaApi = {
  courses: (params?: Record<string, string>) =>
    api.get("/nova/courses/", { params }).then((r) => unwrap(r.data)),
  course: (id: string) =>
    api.get(`/nova/courses/${id}/`).then((r) => r.data),
  materials: (params?: Record<string, string>) =>
    api.get("/nova/materials/", { params }).then((r) => unwrap(r.data)),
  assignments: (params?: Record<string, string>) =>
    api.get("/nova/assignments/", { params }).then((r) => unwrap(r.data)),
  assignment: (id: string) =>
    api.get(`/nova/assignments/${id}/`).then((r) => r.data),
  submitAssignment: (data: FormData) =>
    api.post("/nova/submissions/", data, { headers: { "Content-Type": "multipart/form-data" } }).then((r) => r.data),
  mySubmissions: (params?: Record<string, string>) =>
    api.get("/nova/submissions/", { params }).then((r) => unwrap(r.data)),
  discussions: (params?: Record<string, string>) =>
    api.get("/nova/discussions/", { params }).then((r) => unwrap(r.data)),
  postDiscussion: (data: unknown) =>
    api.post("/nova/discussions/", data).then((r) => r.data),
  quizzes: (params?: Record<string, string>) =>
    api.get("/nova/quizzes/", { params }).then((r) => unwrap(r.data)),
  quiz: (id: string) =>
    api.get(`/nova/quizzes/${id}/`).then((r) => r.data),
  submitQuiz: (id: string, data: unknown) =>
    api.post(`/nova/quizzes/${id}/attempt/`, data).then((r) => r.data),
  myAttempts: (params?: Record<string, string>) =>
    api.get("/nova/attempts/", { params }).then((r) => unwrap(r.data)),
  gradeSubmission: (subId: number, data: { score: string; feedback: string }) =>
    api.post(`/nova/submissions/${subId}/grade/`, data).then((r) => r.data),
  downloadSubmission: (subId: number) =>
    api.get(`/nova/submissions/${subId}/download/`, { responseType: "blob" }).then((r) => {
      const url = URL.createObjectURL(r.data);
      window.open(url, "_blank", "noopener,noreferrer");
      window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
    }),
  courseSubmissions: (courseId: string) =>
    api.get("/nova/submissions/", { params: { assignment__course: courseId } }).then((r) => unwrap(r.data)),
  gradeAttempt: (attemptId: number, data: { score: string; feedback: string }) =>
    api.post(`/nova/attempts/${attemptId}/grade/`, data).then((r) => r.data),
};

export type QuizQuestion = {
  id: number;
  quiz: number;
  order: number;
  prompt: string;
  kind: string;
  choices: string[];
  matching_pairs: { left: string; right: string }[];
  correct_answer: string;
  correct_answers: string[];
  explanation: string;
  points: string;
  is_required: boolean;
};

export const questionsApi = {
  list: (quizId: number | string) =>
    api.get("/nova/questions/", { params: { quiz: quizId } }).then((r) => unwrap<QuizQuestion>(r.data)),
  create: (data: Partial<QuizQuestion>) =>
    api.post("/nova/questions/", data).then((r) => r.data),
  remove: (id: number) => api.delete(`/nova/questions/${id}/`).then((r) => r.data),
};
