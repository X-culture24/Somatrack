import { useEffect, useMemo, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { ChevronLeft, ChevronRight, Pencil, Plus, Trash2, X } from "lucide-react";
import {
  formErrors, humanize, resourcesApi,
  type ResourceField, type ResourceSchema, type Row,
} from "../../api/resources";
import { Badge, Button, Card, Field, Input, PageHeader, Select, Table, kes, useConfirm } from "../../components/ui";

// One page for every declarative resource: the table, filters, search and
// add/edit form are all built from the schema the API serves, so a new
// register needs a spec on the server and nothing here.

const PAGE_SIZE = 25;

export function ResourcePage() {
  const { name = "" } = useParams();
  const { data: schema, isLoading, isError } = useQuery({
    queryKey: ["resource-schema", name],
    queryFn: () => resourcesApi.schema(name),
  });

  if (isLoading) return <Card><p className="text-sm text-slate-500">Loading…</p></Card>;
  if (isError || !schema) return <Card><p className="text-sm text-red-700">This register does not exist or you can't open it.</p></Card>;
  return <ResourceView key={name} schema={schema} />;
}

function ResourceView({ schema }: { schema: ResourceSchema }) {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [editing, setEditing] = useState<Row | "new" | null>(null);

  const listFields = schema.fields.filter((f) => f.in_list);
  const filterFields = schema.fields.filter(
    (f) => f.filterable && (f.type === "enum" || f.type === "bool" || (f.type === "ref" && schema.can_write)),
  );
  const searchable = schema.fields.some((f) => f.searchable);

  const params = useMemo(() => {
    const p: Record<string, string> = { page: String(page), page_size: String(PAGE_SIZE) };
    if (search) p.search = search;
    for (const [k, v] of Object.entries(filters)) if (v) p[k] = v;
    return p;
  }, [page, search, filters]);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["resource-rows", schema.name, params],
    queryFn: () => resourcesApi.rows(schema.name, params),
  });

  const remove = useMutation({
    mutationFn: (id: string) => resourcesApi.remove(schema.name, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["resource-rows", schema.name] }),
  });

  async function onDelete(row: Row) {
    const ok = await confirm({
      title: `Delete this ${schema.title.toLowerCase()}?`,
      message: "This can't be undone.",
      confirmLabel: "Delete",
      variant: "danger",
    });
    if (ok) remove.mutate(row.id);
  }

  const total = data?.count ?? 0;
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div>
      {dialog}
      <Link to="/records" className="mb-2 inline-flex items-center gap-1 text-xs text-slate-500 hover:text-slate-700">
        <ChevronLeft size={14} /> {schema.module}
      </Link>
      <PageHeader
        title={schema.title}
        subtitle={`${total} record${total === 1 ? "" : "s"}`}
        actions={schema.can_write && (
          <Button onClick={() => setEditing("new")}><Plus size={16} /> Add {schema.title.toLowerCase()}</Button>
        )}
      />

      <div className="mb-4 flex flex-wrap items-end gap-3">
        {searchable && (
          <div className="w-full sm:w-64">
            <Input placeholder="Search…" value={search} onChange={(e) => { setSearch(e.target.value); setPage(1); }} />
          </div>
        )}
        {filterFields.map((f) => (
          <div key={f.name} className="w-full sm:w-48">
            <FilterInput schema={schema} field={f} value={filters[f.name] ?? ""}
              onChange={(v) => { setFilters((s) => ({ ...s, [f.name]: v })); setPage(1); }} />
          </div>
        ))}
      </div>

      {remove.isError && <p className="mb-3 text-sm text-red-700">{formErrors(remove.error).message}</p>}
      {isError && <Card><p className="text-sm text-red-700">Could not load records.</p></Card>}

      {!isError && (
        <Table headers={[...listFields.map((f) => f.label), ...(schema.can_write ? [""] : [])]}>
          {isLoading ? (
            <tr><td className="px-4 py-8 text-center text-slate-500" colSpan={listFields.length + 1}>Loading…</td></tr>
          ) : data?.results.length ? data.results.map((row) => (
            <tr key={row.id} className="hover:bg-slate-50">
              {listFields.map((f) => <td key={f.name} className="px-4 py-3">{renderCell(f, row)}</td>)}
              {schema.can_write && (
                <td className="px-4 py-3 text-right whitespace-nowrap">
                  <button className="rounded p-1.5 text-slate-500 hover:bg-slate-100" title="Edit" onClick={() => setEditing(row)}><Pencil size={15} /></button>
                  <button className="rounded p-1.5 text-red-500 hover:bg-red-50" title="Delete" onClick={() => onDelete(row)}><Trash2 size={15} /></button>
                </td>
              )}
            </tr>
          )) : (
            <tr><td className="px-4 py-8 text-center text-slate-500" colSpan={listFields.length + 1}>No records yet.</td></tr>
          )}
        </Table>
      )}

      {pages > 1 && (
        <div className="mt-3 flex items-center justify-end gap-2 text-sm text-slate-600">
          <Button variant="ghost" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}><ChevronLeft size={14} /></Button>
          Page {page} of {pages}
          <Button variant="ghost" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}><ChevronRight size={14} /></Button>
        </div>
      )}

      {editing && (
        <RecordForm schema={schema} row={editing === "new" ? null : editing} onClose={() => setEditing(null)} />
      )}
    </div>
  );
}

function renderCell(f: ResourceField, row: Row) {
  const v = row[f.name];
  if (v === null || v === undefined || v === "") return <span className="text-slate-300">—</span>;
  switch (f.type) {
    case "ref": return row[`${f.name}_label`] ?? v;
    case "bool": return v ? <Badge variant="success">Yes</Badge> : <Badge>No</Badge>;
    case "enum": return <Badge variant="blue">{humanize(String(v))}</Badge>;
    case "decimal": return f.label.includes("KES") ? kes(v) : Number(v).toLocaleString("en-KE");
    case "datetime": return new Date(v).toLocaleString("en-KE", { dateStyle: "medium", timeStyle: "short" });
    case "longtext": return <span className="line-clamp-2 max-w-xs">{v}</span>;
    default: return String(v);
  }
}

function FilterInput({ schema, field, value, onChange }: {
  schema: ResourceSchema; field: ResourceField; value: string; onChange: (v: string) => void;
}) {
  if (field.type === "ref") {
    return <RefSelect schema={schema} field={field} value={value} onChange={onChange} placeholder={`All ${field.label.toLowerCase()}s`} />;
  }
  const opts = field.type === "bool" ? ["true", "false"] : field.options ?? [];
  return (
    <Select value={value} onChange={(e) => onChange(e.target.value)} aria-label={field.label}>
      <option value="">{field.label}: all</option>
      {opts.map((o) => <option key={o} value={o}>{field.type === "bool" ? (o === "true" ? "Yes" : "No") : humanize(o)}</option>)}
    </Select>
  );
}

// ---------------------------------------------------------------- Form

type Values = Record<string, string | boolean>;

function toFormValue(f: ResourceField, v: any): string | boolean {
  if (f.type === "bool") return Boolean(v);
  if (v === null || v === undefined) return "";
  if (f.type === "datetime") {
    const d = new Date(v);
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }
  return String(v);
}

function RecordForm({ schema, row, onClose }: { schema: ResourceSchema; row: Row | null; onClose: () => void }) {
  const qc = useQueryClient();
  const writable = schema.fields.filter((f) => !f.read_only);
  const initial = useMemo(() => {
    const v: Values = {};
    for (const f of writable) v[f.name] = toFormValue(f, row?.[f.name]);
    return v;
  }, [row]); // eslint-disable-line react-hooks/exhaustive-deps
  const [values, setValues] = useState<Values>(initial);
  const [errors, setErrors] = useState<{ message: string; fields: Record<string, string> } | null>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const save = useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      row ? resourcesApi.update(schema.name, row.id, payload) : resourcesApi.create(schema.name, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["resource-rows", schema.name] });
      onClose();
    },
    onError: (e) => setErrors(formErrors(e)),
  });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    const payload: Record<string, unknown> = {};
    for (const f of writable) {
      const v = values[f.name];
      if (row && v === initial[f.name]) continue; // edits send only what changed
      if (v === "") {
        if (!row) continue; // new record: let the server default apply
        payload[f.name] = ["text", "longtext", "phone", "email"].includes(f.type) ? "" : null;
        continue;
      }
      payload[f.name] = v;
    }
    if (row && Object.keys(payload).length === 0) { onClose(); return; }
    setErrors(null);
    save.mutate(payload);
  }

  // Group by section, keeping the order fields first appear in.
  const sections: [string, ResourceField[]][] = [];
  for (const f of writable) {
    const key = f.section ?? "";
    const s = sections.find(([k]) => k === key);
    if (s) s[1].push(f); else sections.push([key, [f]]);
  }

  return (
    <div className="fixed inset-0 z-40 flex justify-end bg-black/30" onClick={onClose}>
      <form onSubmit={onSubmit} onClick={(e) => e.stopPropagation()}
        className="flex h-full w-full max-w-xl flex-col bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
          <h2 className="text-lg font-semibold text-[#1b365d]">{row ? "Edit" : "Add"} {schema.title.toLowerCase()}</h2>
          <button type="button" className="rounded p-1 text-slate-500 hover:bg-slate-100" onClick={onClose} aria-label="Close"><X size={18} /></button>
        </div>
        <div className="flex-1 space-y-6 overflow-y-auto px-5 py-4">
          {errors && <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">{errors.message}</p>}
          {sections.map(([title, fields]) => (
            <fieldset key={title} className="space-y-4">
              {title && <legend className="mb-1 text-xs font-bold tracking-wide text-slate-500 uppercase">{title}</legend>}
              {fields.map((f) => (
                <div key={f.name}>
                  <FieldInput schema={schema} field={f} value={values[f.name]}
                    onChange={(v) => setValues((s) => ({ ...s, [f.name]: v }))} />
                  {errors?.fields[f.name] && <p className="mt-1 text-xs text-red-600">{errors.fields[f.name]}</p>}
                  {f.help && <p className="mt-1 text-xs text-slate-400">{f.help}</p>}
                </div>
              ))}
            </fieldset>
          ))}
        </div>
        <div className="flex justify-end gap-3 border-t border-slate-200 px-5 py-4">
          <Button type="button" variant="ghost" onClick={onClose}>Cancel</Button>
          <Button type="submit" disabled={save.isPending}>{save.isPending ? "Saving…" : "Save"}</Button>
        </div>
      </form>
    </div>
  );
}

function FieldInput({ schema, field: f, value, onChange }: {
  schema: ResourceSchema; field: ResourceField; value: string | boolean; onChange: (v: string | boolean) => void;
}) {
  const label = f.label + (f.required ? " *" : "");
  const common = { required: f.required, value: value as string, onChange: (e: { target: { value: string } }) => onChange(e.target.value) };

  switch (f.type) {
    case "bool":
      return (
        <label className="flex items-center gap-2 text-sm text-slate-700">
          <input type="checkbox" checked={Boolean(value)} onChange={(e) => onChange(e.target.checked)} className="h-4 w-4 rounded border-slate-300" />
          {f.label}
        </label>
      );
    case "enum":
      return (
        <Field label={label}>
          <Select {...common}>
            <option value="">{f.required ? "Choose…" : "—"}</option>
            {f.options!.map((o) => <option key={o} value={o}>{humanize(o)}</option>)}
          </Select>
        </Field>
      );
    case "ref":
      return (
        <Field label={label}>
          <RefSelect schema={schema} field={f} value={value as string} onChange={onChange} required={f.required} />
        </Field>
      );
    case "longtext":
      return (
        <Field label={label}>
          <textarea {...common} rows={3} maxLength={f.max_length}
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-[#4a9eca] focus:ring-2 focus:ring-[#4a9eca]/20" />
        </Field>
      );
    default: {
      const type = { int: "number", decimal: "number", date: "date", datetime: "datetime-local", time: "time", phone: "tel", email: "email" }[f.type as string] ?? "text";
      return (
        <Field label={label}>
          <Input {...common} type={type} maxLength={f.max_length} min={f.min} max={f.max}
            step={f.type === "decimal" ? "0.01" : f.type === "int" ? "1" : undefined} />
        </Field>
      );
    }
  }
}

// Searchable dropdown for reference fields, fed by the engine's options
// endpoint so labels match the table's display.
function RefSelect({ schema, field, value, onChange, required, placeholder }: {
  schema: ResourceSchema; field: ResourceField; value: string; onChange: (v: string) => void;
  required?: boolean; placeholder?: string;
}) {
  const [search, setSearch] = useState("");
  const { data: options = [] } = useQuery({
    queryKey: ["ref-options", schema.name, field.name, search],
    queryFn: () => resourcesApi.options(schema.name, field.name, search ? { search } : {}),
    enabled: schema.can_write,
  });
  // Make sure the current value's label shows even if it isn't in the first 50.
  const { data: current } = useQuery({
    queryKey: ["ref-option", schema.name, field.name, value],
    queryFn: () => resourcesApi.options(schema.name, field.name, { id: value }),
    enabled: schema.can_write && !!value && !options.some((o) => o.id === value),
  });
  const all = [...(current ?? []), ...options.filter((o) => !current?.some((c) => c.id === o.id))];

  return (
    <div className="space-y-1">
      <Input placeholder={`Search ${field.label.toLowerCase()}…`} value={search} onChange={(e) => setSearch(e.target.value)} />
      <Select value={value} onChange={(e) => onChange(e.target.value)} required={required} aria-label={field.label}>
        <option value="">{placeholder ?? (required ? "Choose…" : "—")}</option>
        {all.map((o) => <option key={o.id} value={o.id}>{o.label}</option>)}
      </Select>
    </div>
  );
}
