import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle, Circle, Download } from "lucide-react";
import { academicsApi } from "../../api/academics";
import { cbcApi, type SignoffPayload } from "../../api/cbc";
import { Button, Card, PageHeader, Select } from "../../components/ui";

const PERF_LEVELS = ["Exceeds Expectations", "Meets Expectations", "Approaches Expectations", "Below Expectations"];

interface SignoffRow {
  id: string;
  student_id: string;
  full_name: string;
  admission_no: string;
  teacher_comment: string;
  general_performance_level: string;
  headteacher_comment: string;
  headteacher_signed: boolean;
  opening_date: string;
  closing_date: string;
}

export function HeadteacherDashboardPage() {
  const [classId, setClassId] = useState("");
  const [termId, setTermId] = useState("");
  const [editing, setEditing] = useState<string | null>(null);
  const [form, setForm] = useState<Partial<SignoffPayload>>({});

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: terms } = useQuery({ queryKey: ["terms"], queryFn: () => academicsApi.terms() });

  const { data: signoffs, isLoading } = useQuery({
    queryKey: ["cbc-signoffs", classId, termId],
    queryFn: () => cbcApi.listSignoffs(classId, termId),
    enabled: !!classId && !!termId,
  });

  const qc = useQueryClient();
  const saveSignoff = useMutation({
    mutationFn: (data: SignoffPayload) => cbcApi.upsertSignoff(data),
    onSuccess: () => {
      setEditing(null);
      setForm({});
      qc.invalidateQueries({ queryKey: ["cbc-signoffs", classId, termId] });
    },
  });

  function startEdit(row: SignoffRow) {
    setEditing(row.student_id);
    setForm({
      student_id: row.student_id,
      term_id: termId,
      class_id: classId,
      teacher_comment: row.teacher_comment,
      general_performance_level: row.general_performance_level,
      headteacher_comment: row.headteacher_comment,
      headteacher_signed: row.headteacher_signed,
      opening_date: row.opening_date,
      closing_date: row.closing_date,
    });
  }

  function handleSave() {
    if (!form.student_id || !termId || !classId) return;
    saveSignoff.mutate({ ...form, student_id: form.student_id!, term_id: termId, class_id: classId } as SignoffPayload);
  }

  const rows = (signoffs as SignoffRow[] | undefined) ?? [];
  const signedCount = rows.filter((r) => r.headteacher_signed).length;

  return (
    <div className="p-6 space-y-4">
      <PageHeader
        title="Headteacher — CBC Reports"
        subtitle="Review teacher assessments and sign off on student reports"
      />

      <Card className="p-4">
        <div className="flex gap-4 flex-wrap">
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium mb-1">Class</label>
            <Select value={classId} onChange={(e) => setClassId(e.target.value)}>
              <option value="">Select class…</option>
              {(classes as { id: string; name: string }[] | undefined)?.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </Select>
          </div>
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium mb-1">Term</label>
            <Select value={termId} onChange={(e) => setTermId(e.target.value)}>
              <option value="">Select term…</option>
              {(terms as { id: string; name: string; year: number }[] | undefined)?.map((t) => (
                <option key={t.id} value={t.id}>{t.name} {t.year}</option>
              ))}
            </Select>
          </div>
        </div>
      </Card>

      {rows.length > 0 && (
        <div className="flex gap-4 text-sm">
          <div className="bg-green-50 border border-green-200 rounded px-3 py-2">
            <span className="font-semibold text-green-700">{signedCount}</span>
            <span className="text-green-600"> / {rows.length} signed off</span>
          </div>
          <div className="bg-amber-50 border border-amber-200 rounded px-3 py-2">
            <span className="font-semibold text-amber-700">{rows.length - signedCount}</span>
            <span className="text-amber-600"> pending</span>
          </div>
        </div>
      )}

      {isLoading && <p className="text-gray-500 text-sm">Loading…</p>}

      {!isLoading && classId && termId && rows.length === 0 && (
        <Card className="p-8 text-center text-gray-500">
          No students found in this class for the selected term.
        </Card>
      )}

      <div className="space-y-3">
        {rows.map((row) => (
          <Card key={row.student_id} className="p-4">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-2">
                {row.headteacher_signed ? (
                  <CheckCircle size={18} className="text-green-500 shrink-0" />
                ) : (
                  <Circle size={18} className="text-gray-300 shrink-0" />
                )}
                <div>
                  <p className="font-medium">{row.full_name}</p>
                  <p className="text-xs text-gray-400">{row.admission_no}</p>
                  {row.general_performance_level && (
                    <p className="text-xs text-gray-500 mt-0.5">Performance: {row.general_performance_level}</p>
                  )}
                  {row.teacher_comment && (
                    <p className="text-xs text-gray-500 mt-0.5">Teacher: "{row.teacher_comment}"</p>
                  )}
                  {!row.id && (
                    <p className="text-xs text-amber-500 mt-0.5">No comments yet</p>
                  )}
                </div>
              </div>
              <div className="flex gap-2 shrink-0">
                <Button variant="outline" size="sm" onClick={() => startEdit(row)}>
                  {row.headteacher_signed ? "Edit" : "Sign Off"}
                </Button>
                <a
                  href={`/api/documents/cbc-report/?student_id=${row.student_id}&term_id=${termId}`}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button variant="outline" size="sm">
                    <Download size={14} />
                  </Button>
                </a>
              </div>
            </div>

            {editing === row.student_id && (
              <div className="mt-4 border-t pt-4 space-y-3">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium mb-1">Teacher's Comment (editable by HT)</label>
                    <textarea
                      rows={2}
                      value={form.teacher_comment ?? ""}
                      onChange={(e) => setForm((f) => ({ ...f, teacher_comment: e.target.value }))}
                      className="w-full text-sm border rounded px-2 py-1"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium mb-1">Headteacher's Comment</label>
                    <textarea
                      rows={2}
                      value={form.headteacher_comment ?? ""}
                      onChange={(e) => setForm((f) => ({ ...f, headteacher_comment: e.target.value }))}
                      className="w-full text-sm border rounded px-2 py-1"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium mb-1">General Performance Level</label>
                    <select
                      value={form.general_performance_level ?? ""}
                      onChange={(e) => setForm((f) => ({ ...f, general_performance_level: e.target.value }))}
                      className="w-full text-sm border rounded px-2 py-1"
                    >
                      <option value="">Select…</option>
                      {PERF_LEVELS.map((l) => <option key={l} value={l}>{l}</option>)}
                    </select>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="block text-xs font-medium mb-1">Opening Date</label>
                      <input
                        type="text"
                        placeholder="e.g. 4th Jan 2025"
                        value={form.opening_date ?? ""}
                        onChange={(e) => setForm((f) => ({ ...f, opening_date: e.target.value }))}
                        className="w-full text-sm border rounded px-2 py-1"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium mb-1">Closing Date</label>
                      <input
                        type="text"
                        placeholder="e.g. 28th Mar 2025"
                        value={form.closing_date ?? ""}
                        onChange={(e) => setForm((f) => ({ ...f, closing_date: e.target.value }))}
                        className="w-full text-sm border rounded px-2 py-1"
                      />
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={form.headteacher_signed ?? false}
                      onChange={(e) => setForm((f) => ({ ...f, headteacher_signed: e.target.checked }))}
                      className="w-4 h-4"
                    />
                    <span className="text-sm font-medium">Sign off this report</span>
                  </label>
                  <div className="flex gap-2 ml-auto">
                    <Button variant="outline" size="sm" onClick={() => setEditing(null)}>Cancel</Button>
                    <Button size="sm" onClick={handleSave} disabled={saveSignoff.isPending}>
                      {saveSignoff.isPending ? "Saving…" : "Save"}
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}
