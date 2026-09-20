import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { novaApi, questionsApi } from "../../api/nova";
import { api } from "../../api/client";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

const QUESTION_KINDS: { value: string; label: string; needsAnswer: boolean; needsChoices: boolean }[] = [
  { value: "mcq", label: "Multiple choice", needsAnswer: true, needsChoices: true },
  { value: "multiselect", label: "Multi-select", needsAnswer: true, needsChoices: true },
  { value: "truefalse", label: "True / False", needsAnswer: true, needsChoices: false },
  { value: "short", label: "Short answer", needsAnswer: true, needsChoices: false },
  { value: "fill", label: "Fill in the blank", needsAnswer: true, needsChoices: false },
  { value: "numeric", label: "Numeric", needsAnswer: true, needsChoices: false },
  { value: "long", label: "Long answer / essay (manually graded)", needsAnswer: false, needsChoices: false },
  { value: "match", label: "Matching (manually graded)", needsAnswer: false, needsChoices: false },
  { value: "file", label: "File upload (manually graded)", needsAnswer: false, needsChoices: false },
];

function QuestionBuilder({ quizId, quizTitle }: { quizId: number; quizTitle: string }) {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data: questions } = useQuery({
    queryKey: ["quiz-questions", quizId],
    queryFn: () => questionsApi.list(quizId),
  });
  const [form, setForm] = useState({ prompt: "", kind: "mcq", choices: "", correct_answer: "", points: "1" });
  const kindMeta = QUESTION_KINDS.find((k) => k.value === form.kind)!;

  const addQuestion = useMutation({
    mutationFn: () =>
      questionsApi.create({
        quiz: quizId,
        prompt: form.prompt,
        kind: form.kind,
        points: form.points,
        choices: kindMeta.needsChoices
          ? form.choices.split(",").map((c) => c.trim()).filter(Boolean)
          : [],
        correct_answer: kindMeta.needsAnswer && form.kind !== "multiselect" ? form.correct_answer : "",
        correct_answers:
          form.kind === "multiselect"
            ? form.correct_answer.split(",").map((c) => c.trim()).filter(Boolean)
            : [],
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["quiz-questions", quizId] });
      qc.invalidateQueries({ queryKey: ["quizzes"] });
      setForm({ prompt: "", kind: "mcq", choices: "", correct_answer: "", points: "1" });
    },
  });

  const removeQuestion = useMutation({
    mutationFn: (id: number) => questionsApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["quiz-questions", quizId] }),
  });

  async function handleRemove(id: number, prompt: string) {
    const ok = await confirm({ title: "Remove question", message: `Remove "${prompt}" from this quiz?`, confirmLabel: "Remove" });
    if (ok) removeQuestion.mutate(id);
  }

  return (
    <Card className="border-2 border-[#4a9eca]/30">
      {dialog}
      <h3 className="mb-1 font-semibold text-[#1b365d]">Questions — {quizTitle}</h3>
      <p className="mb-3 text-xs text-slate-400">
        Multiple choice, multi-select, true/false, short, fill-in, and numeric answers are graded automatically.
        Essay, matching, and file questions are flagged for you to grade by hand after a student submits.
      </p>

      {(questions ?? []).length > 0 && (
        <ul className="mb-4 space-y-2">
          {(questions ?? []).map((q) => (
            <li key={q.id} className="flex items-start justify-between gap-3 rounded-lg border border-slate-200 px-3 py-2 text-sm">
              <div>
                <span className="font-medium text-slate-700">{q.prompt}</span>
                <span className="ml-2 text-xs text-slate-400">
                  {QUESTION_KINDS.find((k) => k.value === q.kind)?.label ?? q.kind} · {q.points} pt{Number(q.points) === 1 ? "" : "s"}
                </span>
              </div>
              <button className="shrink-0 text-xs text-red-500 hover:underline" onClick={() => handleRemove(q.id, q.prompt)}>
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Question prompt">
          <Input value={form.prompt} onChange={(e) => setForm({ ...form, prompt: e.target.value })} />
        </Field>
        <Field label="Type">
          <Select value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}>
            {QUESTION_KINDS.map((k) => (
              <option key={k.value} value={k.value}>{k.label}</option>
            ))}
          </Select>
        </Field>
        {kindMeta.needsChoices && (
          <Field label="Choices (comma separated)">
            <Input value={form.choices} onChange={(e) => setForm({ ...form, choices: e.target.value })} placeholder="e.g. 12, 14, 16, 18" />
          </Field>
        )}
        {kindMeta.needsAnswer && (
          <Field label={form.kind === "multiselect" ? "Correct answers (comma separated)" : "Correct answer"}>
            <Input value={form.correct_answer} onChange={(e) => setForm({ ...form, correct_answer: e.target.value })} />
          </Field>
        )}
        <Field label="Points">
          <Input type="number" min="1" value={form.points} onChange={(e) => setForm({ ...form, points: e.target.value })} />
        </Field>
      </div>
      <Button className="mt-3" onClick={() => addQuestion.mutate()} disabled={addQuestion.isPending || !form.prompt}>
        Add question
      </Button>
    </Card>
  );
}

export function TeacherNovaPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();

  const { data: course } = useQuery({ queryKey: ["course", id], queryFn: () => novaApi.course(id!) });
  const { data: materials } = useQuery({ queryKey: ["mats", id], queryFn: () => novaApi.materials({ course: id! }) });
  const { data: quizzes } = useQuery({ queryKey: ["quizzes", id], queryFn: () => novaApi.quizzes({ course: id! }) });

  const [matForm, setMatForm] = useState({ title: "", strand: "", description: "" });
  const [asgForm, setAsgForm] = useState({ title: "", instructions: "", due_at: "", max_score: "100" });
  const [quizForm, setQuizForm] = useState({ title: "", instructions: "", duration_minutes: "30", due_at: "" });
  const [expandedQuizId, setExpandedQuizId] = useState<number | null>(null);

  const addMat = useMutation({
    mutationFn: () => api.post("/nova/materials/", { ...matForm, course: id }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["mats", id] }); setMatForm({ title: "", strand: "", description: "" }); },
  });
  const addAsg = useMutation({
    mutationFn: () => api.post("/nova/assignments/", { ...asgForm, course: id }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["asg", id] }); setAsgForm({ title: "", instructions: "", due_at: "", max_score: "100" }); },
  });
  const addQuiz = useMutation({
    mutationFn: () => api.post("/nova/quizzes/", { ...quizForm, course: id }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["quizzes", id] }); setQuizForm({ title: "", instructions: "", duration_minutes: "30", due_at: "" }); },
  });

  async function handleAddMat() {
    const ok = await confirm({ title: "Add material", message: `Add "${matForm.title}" to this course?`, confirmLabel: "Add" });
    if (ok) addMat.mutate();
  }
  async function handleAddAsg() {
    const ok = await confirm({ title: "Create assignment", message: `Create assignment "${asgForm.title}" due ${asgForm.due_at}?`, confirmLabel: "Create" });
    if (ok) addAsg.mutate();
  }
  async function handleAddQuiz() {
    const ok = await confirm({ title: "Create quiz", message: `Create quiz "${quizForm.title}" (${quizForm.duration_minutes} min)?`, confirmLabel: "Create" });
    if (ok) addQuiz.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <div className="flex items-center justify-between">
        <PageHeader title={course?.title ?? "Manage course"} subtitle="Add content, assignments, and quizzes." />
        <Button variant="ghost" onClick={() => navigate(`/nova/teacher/${id}/submissions`)}>
          Grade submissions →
        </Button>
      </div>

      {/* Learning materials */}
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Add learning material</h3>
        <div className="grid gap-3 sm:grid-cols-3">
          <Field label="Title"><Input value={matForm.title} onChange={(e) => setMatForm({ ...matForm, title: e.target.value })} /></Field>
          <Field label="CBC strand"><Input value={matForm.strand} onChange={(e) => setMatForm({ ...matForm, strand: e.target.value })} placeholder="e.g. Numbers" /></Field>
          <Field label="Notes"><Input value={matForm.description} onChange={(e) => setMatForm({ ...matForm, description: e.target.value })} /></Field>
        </div>
        <Button className="mt-3" onClick={handleAddMat} disabled={addMat.isPending || !matForm.title}>Add material</Button>
      </Card>
      {(materials as any[] ?? []).length > 0 && (
        <Table headers={["Title", "Strand", "Notes"]}>
          {(materials as any[]).map((m: any) => (
            <tr key={m.id}><td className="px-4 py-3">{m.title}</td><td className="px-4 py-3">{m.strand}</td><td className="px-4 py-3">{m.description}</td></tr>
          ))}
        </Table>
      )}

      {/* Assignments */}
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Create assignment / CAT</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Title"><Input value={asgForm.title} onChange={(e) => setAsgForm({ ...asgForm, title: e.target.value })} /></Field>
          <Field label="Due date & time"><Input type="datetime-local" value={asgForm.due_at} onChange={(e) => setAsgForm({ ...asgForm, due_at: e.target.value })} /></Field>
          <Field label="Max marks"><Input type="number" value={asgForm.max_score} onChange={(e) => setAsgForm({ ...asgForm, max_score: e.target.value })} /></Field>
          <Field label="Instructions">
            <textarea rows={2} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]"
              value={asgForm.instructions} onChange={(e) => setAsgForm({ ...asgForm, instructions: e.target.value })} />
          </Field>
        </div>
        <Button className="mt-3" onClick={handleAddAsg} disabled={addAsg.isPending || !asgForm.title || !asgForm.due_at}>Create assignment</Button>
      </Card>

      {/* Quizzes */}
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Create quiz</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Title"><Input value={quizForm.title} onChange={(e) => setQuizForm({ ...quizForm, title: e.target.value })} /></Field>
          <Field label="Duration (minutes)"><Input type="number" min="5" value={quizForm.duration_minutes} onChange={(e) => setQuizForm({ ...quizForm, duration_minutes: e.target.value })} /></Field>
          <Field label="Due date (optional)"><Input type="datetime-local" value={quizForm.due_at} onChange={(e) => setQuizForm({ ...quizForm, due_at: e.target.value })} /></Field>
          <Field label="Instructions"><Input value={quizForm.instructions} onChange={(e) => setQuizForm({ ...quizForm, instructions: e.target.value })} /></Field>
        </div>
        <Button className="mt-3" onClick={handleAddQuiz} disabled={addQuiz.isPending || !quizForm.title}>Create quiz</Button>
      </Card>

      {(quizzes as any[] ?? []).length > 0 && (
        <Table headers={["Quiz", "Duration", "Due", "Questions", ""]}>
          {(quizzes as any[]).map((q: any) => (
            <tr key={q.id}>
              <td className="px-4 py-3">{q.title}</td>
              <td className="px-4 py-3">{q.duration_minutes}m</td>
              <td className="px-4 py-3">{q.due_at ? new Date(q.due_at).toLocaleDateString() : "—"}</td>
              <td className="px-4 py-3">{q.question_count ?? 0}</td>
              <td className="px-4 py-3">
                <button
                  className="text-xs font-medium text-[#4a9eca] hover:underline"
                  onClick={() => setExpandedQuizId(expandedQuizId === q.id ? null : q.id)}
                >
                  {expandedQuizId === q.id ? "Hide questions" : "Manage questions"}
                </button>
              </td>
            </tr>
          ))}
        </Table>
      )}

      {expandedQuizId != null && (
        <QuestionBuilder
          quizId={expandedQuizId}
          quizTitle={(quizzes as any[] ?? []).find((q: any) => q.id === expandedQuizId)?.title ?? ""}
        />
      )}
    </div>
  );
}
