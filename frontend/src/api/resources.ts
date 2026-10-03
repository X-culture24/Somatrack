import { api } from "./client";

// Client for the declarative resource engine (internal/resource). Every
// record-keeping form — visitors, discipline, hostels, expenses… — is served
// the same way, so one set of calls and one generic page cover them all.

export type FieldType =
  | "text" | "longtext" | "int" | "decimal" | "bool" | "date" | "datetime"
  | "time" | "enum" | "ref" | "phone" | "email";

export type ResourceField = {
  name: string;
  label: string;
  type: FieldType;
  required?: boolean;
  options?: string[];
  ref?: { resource: string };
  read_only?: boolean;
  filterable?: boolean;
  searchable?: boolean;
  in_list?: boolean;
  section?: string;
  help?: string;
  max_length?: number;
  min?: number;
  max?: number;
};

export type ResourceSchema = {
  name: string;
  title: string;
  module: string;
  fields: ResourceField[];
  can_write: boolean;
};

export type ResourceSummary = { name: string; title: string; module: string; can_write: boolean };

export type Row = Record<string, any> & { id: string };

export type Page = { results: Row[]; count: number; page: number; page_size: number };

export type Option = { id: string; label: string };

export const resourcesApi = {
  list: () =>
    api.get<{ results: ResourceSummary[] }>("/meta/resources/").then((r) => r.data.results),
  schema: (name: string) =>
    api.get<ResourceSchema>(`/meta/resources/${name}/`).then((r) => r.data),
  options: (name: string, field: string, params: { search?: string; id?: string }) =>
    api.get<{ results: Option[] }>(`/meta/resources/${name}/options/${field}/`, { params }).then((r) => r.data.results),
  rows: (name: string, params: Record<string, string>) =>
    api.get<Page>(`/${name}/`, { params }).then((r) => r.data),
  create: (name: string, data: Record<string, unknown>) =>
    api.post<Row>(`/${name}/`, data).then((r) => r.data),
  update: (name: string, id: string, data: Record<string, unknown>) =>
    api.patch<Row>(`/${name}/${id}/`, data).then((r) => r.data),
  remove: (name: string, id: string) => api.delete(`/${name}/${id}/`),
};

/** Turns an API error into { message, fields } for a form to show. */
export function formErrors(e: any): { message: string; fields: Record<string, string> } {
  const d = e?.response?.data;
  return {
    message: d?.message || "Something went wrong. Try again.",
    fields: d?.fields ?? {},
  };
}

/** "walk_in" → "Walk in" for enum values shown to people. */
export function humanize(v: string) {
  const s = v.replace(/_/g, " ");
  return s.charAt(0).toUpperCase() + s.slice(1);
}
