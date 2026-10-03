import { api } from "./client";

function unwrap(d: { data?: unknown; results?: unknown } | unknown) {
  if (d && typeof d === "object") {
    if ("data" in (d as Record<string, unknown>)) return (d as { data: unknown }).data;
    if ("results" in (d as Record<string, unknown>)) return (d as { results: unknown }).results;
  }
  return d;
}

export interface MarkEntry {
  assessment_id: string;
  level: string;
  reflection: string;
}

export interface SubjectEntry {
  subject_id: string;
  subject_name: string;
  learning_area: string;
  cat1: MarkEntry;
  cat2: MarkEntry;
  cat3: MarkEntry;
}

export interface CBCStudentRow {
  student_id: string;
  full_name: string;
  admission_no: string;
  subjects: SubjectEntry[];
  signoff: Record<string, string> | null;
}

export interface CBCMarksheetResponse {
  class_id: string;
  term_id: string;
  students: CBCStudentRow[];
  subjects: { id: string; name: string; code: string; learning_area: string }[];
}

export interface SaveMarkEntry {
  assessment_id: string;
  student_id: string;
  level: string;
  reflection: string;
}

export interface SignoffPayload {
  student_id: string;
  term_id: string;
  class_id: string;
  teacher_comment: string;
  general_performance_level: string;
  headteacher_comment: string;
  headteacher_signed: boolean;
  opening_date: string;
  closing_date: string;
}

export const cbcApi = {
  marksheet: (class_id: string, term_id: string) =>
    api
      .get<CBCMarksheetResponse>("/cbc/marksheet/", { params: { class_id, term_id } })
      .then((r) => r.data),

  saveMarks: (entries: SaveMarkEntry[]) =>
    api.put("/cbc/marksheet/", { entries }).then((r) => r.data),

  classSummary: (class_id: string, term_id: string) =>
    api
      .get("/cbc/class-summary/", { params: { class_id, term_id } })
      .then((r) => unwrap(r.data)),

  listSignoffs: (class_id: string, term_id: string) =>
    api
      .get("/cbc/signoffs/", { params: { class_id, term_id } })
      .then((r) => unwrap(r.data)),

  upsertSignoff: (data: SignoffPayload) =>
    api.post("/cbc/signoffs/", data).then((r) => r.data),
};
