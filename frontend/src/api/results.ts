import { api, downloadFile } from "./client";

// Assessments, marks entry and the results trail (internal/api/results.go).

export type Assessment = {
  id: string;
  name: string;
  assessment_type: string;
  class_id: string;
  class_name: string;
  subject_id: string;
  subject_name: string;
  term_id: string;
  term_name: string;
  exam_thread_id: string | null;
  exam_thread_name: string;
  total_marks: number;
  weight: number;
  assessment_date: string | null;
  marks_count: number;
  is_locked: boolean;
};

export type MarkRow = {
  student_id: string;
  full_name: string;
  admission_no: string;
  score: number | null;
  percentage: number | null;
  grade_letter: string;
  competency_level: string;
  comments: string;
};

export type SubjectResult = {
  subject_id: string;
  subject_name: string;
  percentage: number;
  grade: string;
  remarks: string;
  assessments: number;
};

export type TermResult = {
  term_id: string;
  term_name: string;
  academic_year: string;
  start_date: string;
  class_id: string;
  class_name: string;
  subjects: SubjectResult[];
  mean: number;
  mean_grade: string;
};

export type ResultsHistory = {
  student_id: string;
  full_name: string;
  admission_no: string;
  terms: TermResult[];
};

export const ASSESSMENT_TYPES = [
  ["cat1", "CAT 1"], ["cat2", "CAT 2"], ["cat3", "CAT 3"], ["midterm", "Mid-term"],
  ["endterm", "End of term"], ["mock", "Mock"], ["assignment", "Assignment"],
  ["project", "Project"], ["practical", "Practical"], ["homework", "Homework"], ["other", "Other"],
] as const;

export const COMPETENCY = [
  ["EE", "Exceeding expectations"], ["ME", "Meeting expectations"],
  ["AE", "Approaching expectations"], ["BE", "Below expectations"],
] as const;

export const resultsApi = {
  assessments: (params: Record<string, string>) =>
    api.get<{ results: Assessment[] }>("/assessments/", { params }).then((r) => r.data.results),
  createAssessment: (data: Record<string, unknown>) =>
    api.post<Assessment>("/assessments/", data).then((r) => r.data),
  marksheet: (id: string) =>
    api.get<{ assessment: Assessment; results: MarkRow[] }>(`/assessments/${id}/marksheet/`).then((r) => r.data),
  saveMarks: (id: string, marks: { student_id: string; score: number | null; competency_level: string; comments: string }[]) =>
    api.put<{ saved: number; cleared: number }>(`/assessments/${id}/marks/`, { marks }).then((r) => r.data),
  history: (studentId: string) =>
    api.get<ResultsHistory>(`/results/students/${studentId}/history/`).then((r) => r.data),
  downloadTranscript: (studentId: string, admissionNo: string) =>
    downloadFile(`/results/students/${studentId}/transcript/`, `academic-record-${admissionNo}.pdf`),
  downloadReportCard: (studentId: string, termId: string, filename: string) =>
    downloadFile(`/documents/cbc-report/?student_id=${studentId}&term_id=${termId}`, filename),
};
