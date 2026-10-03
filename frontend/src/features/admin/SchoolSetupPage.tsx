import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";
import { ModulePage } from "./ModulePage";

const SUBJECT_LEVELS = [
  { value: "KG", label: "Pre-School (PP1/PP2/Playgroup)" },
  { value: "LP", label: "Lower Primary (Grade 1-3)" },
  { value: "UP", label: "Upper Primary (Grade 4-6)" },
  { value: "JSS", label: "Junior Secondary (Grade 7-9)" },
  { value: "ALL", label: "All Levels" },
];

export function SchoolSetupPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data: years } = useQuery({ queryKey: ["academic-years"], queryFn: academicsApi.years });
  const { data: terms } = useQuery({ queryKey: ["terms"], queryFn: academicsApi.terms });
  const { data: streams } = useQuery({ queryKey: ["streams"], queryFn: academicsApi.streams });
  const { data: subjects } = useQuery({ queryKey: ["subjects"], queryFn: () => academicsApi.subjects() });

  const EMPTY_YEAR = { name: "", start_date: "", end_date: "", is_active: false };
  const [yearForm, setYearForm] = useState(EMPTY_YEAR);
  const [editingYearId, setEditingYearId] = useState<string | null>(null);
  const [termForm, setTermForm] = useState({ academic_year: "", number: "1", start_date: "", end_date: "", is_current: false });
  const [streamName, setStreamName] = useState("");
  const [subjectForm, setSubjectForm] = useState({ name: "", code: "", level: "ALL", learning_area: "" });

  const createYear = useMutation({
    mutationFn: () => academicsApi.createYear(yearForm),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["academic-years"] }); setYearForm(EMPTY_YEAR); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not create — this name may already exist."),
  });
  const updateYear = useMutation({
    mutationFn: () => academicsApi.updateYear(editingYearId!, yearForm),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["academic-years"] }); setYearForm(EMPTY_YEAR); setEditingYearId(null); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not save changes."),
  });

  function startEditYear(y: any) {
    setEditingYearId(y.id);
    setYearForm({ name: y.name, start_date: y.start_date, end_date: y.end_date, is_active: y.is_active });
  }
  function cancelEditYear() {
    setEditingYearId(null);
    setYearForm(EMPTY_YEAR);
  }
  const createTerm = useMutation({
    mutationFn: () => academicsApi.createTerm(termForm),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["terms"] }); setTermForm({ ...termForm, start_date: "", end_date: "" }); },
  });
  const createStream = useMutation({
    mutationFn: () => academicsApi.createStream({ name: streamName }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["streams"] }); setStreamName(""); },
  });
  const createSubject = useMutation({
    mutationFn: () => academicsApi.createSubject(subjectForm),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["subjects"] }); setSubjectForm({ name: "", code: "", level: "ALL", learning_area: "" }); },
  });

  async function handleSaveYear() {
    const isEdit = Boolean(editingYearId);
    const ok = await confirm({
      title: isEdit ? "Save academic year changes" : "Create academic year",
      message: isEdit ? `Save changes to "${yearForm.name}"?` : `Create academic year "${yearForm.name}"?`,
      confirmLabel: isEdit ? "Save changes" : "Create",
    });
    if (ok) (isEdit ? updateYear : createYear).mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="School setup" subtitle="Academic years, terms, streams, and subjects that power the school year." />

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-semibold text-[#1b365d]">Academic years</h3>
          {editingYearId && <Button variant="ghost" onClick={cancelEditYear}>Cancel edit</Button>}
        </div>
        <p className="mb-3 text-xs text-slate-500">
          Academic years are preloaded — click "Edit" on a row below to change its dates or active status. The form below is only for adding a genuinely new year.
        </p>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Name"><Input placeholder="2027" value={yearForm.name} onChange={(e) => setYearForm({ ...yearForm, name: e.target.value })} /></Field>
          <Field label="Starts"><Input type="date" value={yearForm.start_date} onChange={(e) => setYearForm({ ...yearForm, start_date: e.target.value })} /></Field>
          <Field label="Ends"><Input type="date" value={yearForm.end_date} onChange={(e) => setYearForm({ ...yearForm, end_date: e.target.value })} /></Field>
          <Field label="Active year">
            <Select value={yearForm.is_active ? "yes" : "no"} onChange={(e) => setYearForm({ ...yearForm, is_active: e.target.value === "yes" })}>
              <option value="no">No</option>
              <option value="yes">Yes — make active</option>
            </Select>
          </Field>
        </div>
        <Button className="mt-3" onClick={handleSaveYear} disabled={createYear.isPending || updateYear.isPending || !yearForm.name}>
          {createYear.isPending || updateYear.isPending ? "Saving…" : editingYearId ? "Save changes" : "Create year"}
        </Button>
        <Table headers={["Academic year", "Starts", "Ends", "Active", ""]}>
          {(years as any[] ?? []).map((y: any) => (
            <tr key={y.id} className={editingYearId === y.id ? "bg-[#4a9eca]/5" : ""}>
              <td className="px-4 py-2">{y.name}</td>
              <td className="px-4 py-2">{y.start_date}</td>
              <td className="px-4 py-2">{y.end_date}</td>
              <td className="px-4 py-2">{y.is_active ? "Active" : ""}</td>
              <td className="px-4 py-2"><Button variant="ghost" onClick={() => startEditYear(y)}>Edit</Button></td>
            </tr>
          ))}
        </Table>
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Terms</h3>
        <div className="grid gap-3 md:grid-cols-5">
          <Field label="Academic year">
            <Select value={termForm.academic_year} onChange={(e) => setTermForm({ ...termForm, academic_year: e.target.value })}>
              <option value="">Select year…</option>
              {(years as any[] ?? []).map((y: any) => <option key={y.id} value={y.id}>{y.name}</option>)}
            </Select>
          </Field>
          <Field label="Term">
            <Select value={termForm.number} onChange={(e) => setTermForm({ ...termForm, number: e.target.value })}>
              <option value="1">Term 1</option><option value="2">Term 2</option><option value="3">Term 3</option>
            </Select>
          </Field>
          <Field label="Starts"><Input type="date" value={termForm.start_date} onChange={(e) => setTermForm({ ...termForm, start_date: e.target.value })} /></Field>
          <Field label="Ends"><Input type="date" value={termForm.end_date} onChange={(e) => setTermForm({ ...termForm, end_date: e.target.value })} /></Field>
          <Field label="Current term">
            <Select value={termForm.is_current ? "yes" : "no"} onChange={(e) => setTermForm({ ...termForm, is_current: e.target.value === "yes" })}>
              <option value="no">No</option>
              <option value="yes">Yes — make current</option>
            </Select>
          </Field>
        </div>
        <Button className="mt-3" onClick={() => createTerm.mutate()} disabled={createTerm.isPending || !termForm.academic_year}>
          {createTerm.isPending ? "Creating…" : "Create term"}
        </Button>
        <Table headers={["Term", "Starts", "Ends", "Current"]}>
          {(terms as any[] ?? []).map((t: any) => (
            <tr key={t.id}><td className="px-4 py-2">Term {t.number}</td><td className="px-4 py-2">{t.start_date}</td><td className="px-4 py-2">{t.end_date}</td><td className="px-4 py-2">{t.is_current ? "Current" : ""}</td></tr>
          ))}
        </Table>
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Streams</h3>
        <div className="flex flex-wrap items-end gap-3">
          <Field label="Stream name"><Input placeholder="e.g. Blue" className="w-48" value={streamName} onChange={(e) => setStreamName(e.target.value)} /></Field>
          <Button onClick={() => createStream.mutate()} disabled={createStream.isPending || !streamName}>Add stream</Button>
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          {(streams as any[] ?? []).map((s: any) => (
            <span key={s.id} className="rounded-full bg-[#4a9eca]/10 px-3 py-1 text-xs font-medium text-[#1b365d]">{s.name}</span>
          ))}
        </div>
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Subjects / learning areas</h3>
        <div className="grid gap-3 md:grid-cols-4">
          <Field label="Name"><Input value={subjectForm.name} onChange={(e) => setSubjectForm({ ...subjectForm, name: e.target.value })} /></Field>
          <Field label="Code"><Input value={subjectForm.code} onChange={(e) => setSubjectForm({ ...subjectForm, code: e.target.value })} /></Field>
          <Field label="Level">
            <Select value={subjectForm.level} onChange={(e) => setSubjectForm({ ...subjectForm, level: e.target.value })}>
              {SUBJECT_LEVELS.map((l) => <option key={l.value} value={l.value}>{l.label}</option>)}
            </Select>
          </Field>
          <Field label="Learning area"><Input value={subjectForm.learning_area} onChange={(e) => setSubjectForm({ ...subjectForm, learning_area: e.target.value })} /></Field>
        </div>
        <Button className="mt-3" onClick={() => createSubject.mutate()} disabled={createSubject.isPending || !subjectForm.name || !subjectForm.code}>
          {createSubject.isPending ? "Creating…" : "Add subject"}
        </Button>
        <Table headers={["Code", "Subject", "Level", "Learning area"]}>
          {(subjects as any[] ?? []).map((s: any) => (
            <tr key={s.id}><td className="px-4 py-2 font-mono text-xs">{s.code}</td><td className="px-4 py-2">{s.name}</td><td className="px-4 py-2">{s.level}</td><td className="px-4 py-2">{s.learning_area || "-"}</td></tr>
          ))}
        </Table>
      </Card>
    </div>
  );
}

export function TimetablePage() {
  // day_of_week is 1 (Monday) … 7 (Sunday).
  const DAYS = ["", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
  return <ModulePage config={{
    title: "Timetable",
    subtitle: "See the teaching day across classes, subjects, and staff.",
    endpoint: "/timetable/",
    queryKey: "timetable",
    columns: [
      { label: "Day", value: (r) => DAYS[r.day_of_week] ?? r.day_of_week },
      { label: "Time", value: (r) => `${r.start_time} - ${r.end_time}` },
      { label: "Class", value: (r) => r.class_id_label },
      { label: "Subject", value: (r) => r.subject_id_label },
      { label: "Teacher", value: (r) => r.teacher_id_label || "-" },
    ],
    stats: [{ label: "Lessons", value: (rows) => rows.length }],
  }} />;
}
