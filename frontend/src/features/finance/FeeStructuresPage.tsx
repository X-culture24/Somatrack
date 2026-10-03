import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { feeStructuresApi, GRADE_LABELS, type FeeStructure } from "../../api/finance";
import { academicsApi } from "../../api/academics";
import { Button, Card, Field, Input, PageHeader, Select, Table, kes, useConfirm } from "../../components/ui";

const COMPONENTS: { key: keyof typeof EMPTY_FORM; label: string }[] = [
  { key: "admission_fee", label: "Admission (new intake, Term 1 only)" },
  { key: "tuition_fee", label: "Tuition / school fees" },
  { key: "diary_book_fee", label: "Diary & progress book (Term 1 only)" },
  { key: "boarding_fee", label: "Boarding" },
  { key: "transport_fee", label: "Transport (flat, if not using per-route pricing)" },
  { key: "medical_fee", label: "Medical" },
  { key: "activity_fee", label: "Activity" },
  { key: "exam_fee", label: "Exam" },
  { key: "library_fee", label: "Library" },
  { key: "computer_fee", label: "Computer" },
  { key: "uniform_fee", label: "Uniform" },
  { key: "other_fee", label: "Other" },
];

const EMPTY_FORM = {
  academic_year_id: "", grade: "", term_number: "1",
  admission_fee: "0", tuition_fee: "0", diary_book_fee: "0",
  boarding_fee: "0", transport_fee: "0", medical_fee: "0",
  activity_fee: "0", exam_fee: "0", library_fee: "0", computer_fee: "0",
  uniform_fee: "0", other_fee: "0", notes: "",
};

function gradeLabel(grade: number): string {
  return GRADE_LABELS[grade] ?? `Grade ${grade}`;
}

function toForm(f: FeeStructure) {
  return {
    academic_year_id: f.academic_year_id, grade: String(f.grade), term_number: String(f.term_number),
    admission_fee: String(f.admission_fee), tuition_fee: String(f.tuition_fee), diary_book_fee: String(f.diary_book_fee),
    boarding_fee: String(f.boarding_fee), transport_fee: String(f.transport_fee), medical_fee: String(f.medical_fee),
    activity_fee: String(f.activity_fee), exam_fee: String(f.exam_fee), library_fee: String(f.library_fee),
    computer_fee: String(f.computer_fee), uniform_fee: String(f.uniform_fee), other_fee: String(f.other_fee),
    notes: f.notes ?? "",
  };
}

export function FeeStructuresPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data: rows } = useQuery({ queryKey: ["fees"], queryFn: () => feeStructuresApi.list() });
  const { data: years } = useQuery({ queryKey: ["academic-years"], queryFn: academicsApi.years });
  const [form, setForm] = useState(EMPTY_FORM);
  const [editingId, setEditingId] = useState<string | null>(null);

  function payload() {
    return {
      academic_year_id: form.academic_year_id,
      grade: Number(form.grade),
      term_number: Number(form.term_number),
      admission_fee: Number(form.admission_fee),
      tuition_fee: Number(form.tuition_fee),
      diary_book_fee: Number(form.diary_book_fee),
      boarding_fee: Number(form.boarding_fee),
      transport_fee: Number(form.transport_fee),
      medical_fee: Number(form.medical_fee),
      activity_fee: Number(form.activity_fee),
      exam_fee: Number(form.exam_fee),
      library_fee: Number(form.library_fee),
      computer_fee: Number(form.computer_fee),
      uniform_fee: Number(form.uniform_fee),
      other_fee: Number(form.other_fee),
      notes: form.notes,
    };
  }

  const create = useMutation({
    mutationFn: () => feeStructuresApi.create(payload()),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["fees"] });
      setForm(EMPTY_FORM);
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not save — this grade/term may already have a fee structure for this year.");
    },
  });
  const update = useMutation({
    mutationFn: () => feeStructuresApi.update(editingId!, payload()),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["fees"] });
      setForm(EMPTY_FORM);
      setEditingId(null);
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not save changes.");
    },
  });

  function startEdit(f: FeeStructure) {
    setEditingId(f.id);
    setForm(toForm(f));
  }
  function cancelEdit() {
    setEditingId(null);
    setForm(EMPTY_FORM);
  }

  async function handleSave() {
    const isEdit = Boolean(editingId);
    const ok = await confirm({
      title: isEdit ? "Save fee structure changes" : "Create fee structure",
      message: isEdit
        ? `Save changes to ${gradeLabel(Number(form.grade))} Term ${form.term_number}? Existing invoices won't update until billing is regenerated.`
        : `Create the ${gradeLabel(Number(form.grade))} Term ${form.term_number} fee structure? Run "Generate billing" afterwards to apply it to students.`,
      confirmLabel: isEdit ? "Save changes" : "Create",
    });
    if (ok) (isEdit ? update : create).mutate();
  }

  const sorted = [...(rows ?? [])].sort((a, b) => a.grade - b.grade || a.term_number - b.term_number);

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader
        title="Fee structures"
        subtitle="One row per (academic year, grade, term) — Playgroup through Grade 9, matching the school's real fee schedule. Click a row to edit it. Generate billing from the Finance home page after changes."
      />

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-semibold text-[#1b365d]">{editingId ? "Edit fee structure" : "New fee structure"}</h3>
          {editingId && (
            <Button variant="ghost" onClick={cancelEdit}>Cancel edit</Button>
          )}
        </div>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Academic year">
            <Select value={form.academic_year_id} onChange={(e) => setForm({ ...form, academic_year_id: e.target.value })}>
              <option value="">Select…</option>
              {(years ?? []).map((y: any) => <option key={y.id} value={y.id}>{y.name}</option>)}
            </Select>
          </Field>
          <Field label="Grade / level">
            <Select value={form.grade} onChange={(e) => setForm({ ...form, grade: e.target.value })}>
              <option value="">Select…</option>
              {Object.entries(GRADE_LABELS).map(([g, label]) => (
                <option key={g} value={g}>{label}</option>
              ))}
            </Select>
          </Field>
          <Field label="Term number">
            <Select value={form.term_number} onChange={(e) => setForm({ ...form, term_number: e.target.value })}>
              <option value="1">Term 1</option>
              <option value="2">Term 2</option>
              <option value="3">Term 3</option>
            </Select>
          </Field>
          <Field label="Notes">
            <Input value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} placeholder="Optional" />
          </Field>
          {COMPONENTS.map((c) => (
            <Field key={c.key} label={`${c.label} (KES)`}>
              <Input type="number" min="0" value={form[c.key]} onChange={(e) => setForm({ ...form, [c.key]: e.target.value })} />
            </Field>
          ))}
        </div>
        <div className="mt-4 flex gap-2">
          <Button
            onClick={handleSave}
            disabled={create.isPending || update.isPending || !form.academic_year_id || form.grade === ""}
          >
            {create.isPending || update.isPending ? "Saving…" : editingId ? "Save changes" : "Create fee structure"}
          </Button>
        </div>
      </Card>

      <Table headers={["Level", "Term", "Admission", "Tuition", "Diary/book", "Total (day)", "Total (boarding)", ""]}>
        {sorted.length === 0 && (
          <tr><td colSpan={8} className="px-4 py-8 text-center text-sm text-slate-400">No fee structures yet.</td></tr>
        )}
        {sorted.map((f) => (
          <tr key={f.id} className={`hover:bg-slate-50 ${editingId === f.id ? "bg-[#4a9eca]/5" : ""}`}>
            <td className="px-4 py-3 font-medium text-[#1b365d]">{gradeLabel(f.grade)}</td>
            <td className="px-4 py-3">
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium">Term {f.term_number}</span>
            </td>
            <td className="px-4 py-3">{f.admission_fee > 0 ? kes(f.admission_fee) : <span className="text-slate-400">—</span>}</td>
            <td className="px-4 py-3">{kes(f.tuition_fee)}</td>
            <td className="px-4 py-3">{f.diary_book_fee > 0 ? kes(f.diary_book_fee) : <span className="text-slate-400">—</span>}</td>
            <td className="px-4 py-3 font-semibold text-[#1b365d]">{kes(f.total_day)}</td>
            <td className="px-4 py-3 text-slate-600">{kes(f.total_boarding)}</td>
            <td className="px-4 py-3">
              <Button variant="ghost" onClick={() => startEdit(f)}>Edit</Button>
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
