import { api } from "./client";

export type UploadCategory =
  | "birth_cert" | "guardian_id" | "photo"
  | "staff_id" | "certificate" | "appointment_letter" | "good_conduct";

// POST /api/uploads/ — multipart. Returns the server-generated relative
// path (store this on the student/guardian record) and a servable url.
export const uploadsApi = {
  upload: (category: UploadCategory, file: File) => {
    const form = new FormData();
    form.append("category", category);
    form.append("file", file);
    return api
      .post<{ path: string; url: string }>("/uploads/", form, {
        headers: { "Content-Type": "multipart/form-data" },
      })
      .then((r) => r.data);
  },
};
