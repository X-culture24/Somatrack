import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Save, ChevronDown, ChevronUp } from "lucide-react";
import { academicsApi } from "../../api/academics";
import { cbcApi, type SaveMarkEntry } from "../../api/cbc";
import { Button, Card, PageHeader, Select } from "../../components/ui";

const LEVELS = ["", "EE", "ME", "AE", "BE"] as const;
const LEVEL_COLORS: Record<string, string> = {
  EE: "bg-green-100 text-green-800 border-green-300",
  ME: "bg-blue-100 text-blue-800 border-blue-300",
  AE: "bg-amber-100 text-amber-800 border-amber-300",
  BE: "bg-red-100 text-red-800 border-red-300",
  "": "bg-gray-50 text-gray-400 border-gray-200",
};

export function CBCMarksheetPage() {
  const [classId, setClassId] = useState("");
  const [termId, setTermId] = useState("");
  const [expandedStudent, setExpandedStudent] = useState<string | null>(null);
  const [pending, setPending] = useState<Record<string, Record<string, { level: string; reflection: string }>>>({});
  // pending[studentId][assessmentId] = {level, reflection}

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: terms } = useQuery({ queryKey: ["terms"], queryFn: () => academicsApi.terms() });

  const { data: marksheet, isLoading } = useQuery({
    queryKey: ["cbc-marksheet", classId, termId],
    queryFn: () => cbcApi.marksheet(classId, termId),
    enabled: !!classId && !!termId,
  });

  const qc = useQueryClient();
  const save = useMutation({
    mutationFn: (entries: SaveMarkEntry[]) => cbcApi.saveMarks(entries),
    onSuccess: () => {
      setPending({});
      qc.invalidateQueries({ queryKey: ["cbc-marksheet", classId, termId] });
    },
  });

  function getLevel(studentId: string, assessmentId: string, fallback: string) {
    return pending[studentId]?.[assessmentId]?.level ?? fallback;
  }
  function getReflection(studentId: string, assessmentId: string, fallback: string) {
    return pending[studentId]?.[assessmentId]?.reflection ?? fallback;
  }
  function setLevel(studentId: string, assessmentId: string, level: string) {
    setPending((p) => ({
      ...p,
      [studentId]: {
        ...(p[studentId] ?? {}),
        [assessmentId]: {
          level,
          reflection: p[studentId]?.[assessmentId]?.reflection ?? "",
        },
      },
    }));
  }
  function setReflection(studentId: string, assessmentId: string, reflection: string) {
    setPending((p) => ({
      ...p,
      [studentId]: {
        ...(p[studentId] ?? {}),
        [assessmentId]: {
          level: p[studentId]?.[assessmentId]?.level ?? "",
          reflection,
        },
      },
    }));
  }

  function handleSaveAll() {
    if (!marksheet) return;
    const entries: SaveMarkEntry[] = [];
    for (const st of marksheet.students) {
      for (const sub of st.subjects) {
        for (const cat of ["cat1", "cat2", "cat3"] as const) {
          const entry = sub[cat];
          if (!entry.assessment_id) continue;
          const level = getLevel(st.student_id, entry.assessment_id, entry.level);
          const reflection = getReflection(st.student_id, entry.assessment_id, entry.reflection);
          if (level || reflection) {
            entries.push({ assessment_id: entry.assessment_id, student_id: st.student_id, level, reflection });
          }
        }
      }
    }
    save.mutate(entries);
  }

  const hasPending = Object.keys(pending).length > 0;

  return (
    <div className="p-6 space-y-4">
      <PageHeader title="CBC Marksheet Entry" subtitle="Enter competency levels per student and subject" />

      <Card className="p-4">
        <div className="flex gap-4 flex-wrap">
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium mb-1">Class</label>
            <Select value={classId} onChange={(e) => { setClassId(e.target.value); setPending({}); }}>
              <option value="">Select class…</option>
              {(classes as { id: string; name: string }[] | undefined)?.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </Select>
          </div>
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium mb-1">Term</label>
            <Select value={termId} onChange={(e) => { setTermId(e.target.value); setPending({}); }}>
              <option value="">Select term…</option>
              {(terms as { id: string; name: string; year: number }[] | undefined)?.map((t) => (
                <option key={t.id} value={t.id}>{t.name} {t.year}</option>
              ))}
            </Select>
          </div>
          {hasPending && (
            <div className="flex items-end">
              <Button onClick={handleSaveAll} disabled={save.isPending}>
                <Save size={14} className="mr-1" />
                {save.isPending ? "Saving…" : `Save changes`}
              </Button>
            </div>
          )}
        </div>
      </Card>

      {isLoading && <p className="text-gray-500 text-sm">Loading marksheet…</p>}

      {marksheet && (
        <div className="space-y-2">
          <div className="flex items-center justify-between mb-2">
            <p className="text-sm text-gray-600">{marksheet.students?.length ?? 0} students · {marksheet.subjects?.length ?? 0} subjects</p>
            <div className="flex gap-2 text-xs">
              {["EE", "ME", "AE", "BE"].map((l) => (
                <span key={l} className={`px-2 py-0.5 rounded border ${LEVEL_COLORS[l]}`}>{l}</span>
              ))}
            </div>
          </div>

          {marksheet.students?.map((st) => {
            const isOpen = expandedStudent === st.student_id;
            return (
              <Card key={st.student_id} className="overflow-hidden">
                <button
                  className="w-full flex items-center justify-between p-3 text-left hover:bg-gray-50"
                  onClick={() => setExpandedStudent(isOpen ? null : st.student_id)}
                >
                  <span className="font-medium">{st.full_name}</span>
                  <div className="flex items-center gap-3">
                    <span className="text-xs text-gray-400">{st.admission_no}</span>
                    {isOpen ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                  </div>
                </button>

                {isOpen && (
                  <div className="border-t overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="text-left p-2 font-medium">Subject</th>
                          <th className="text-center p-2 font-medium">Ass 1</th>
                          <th className="text-center p-2 font-medium">Ass 2</th>
                          <th className="text-center p-2 font-medium">Ass 3</th>
                          <th className="text-left p-2 font-medium">Reflection</th>
                        </tr>
                      </thead>
                      <tbody>
                        {st.subjects?.map((sub) => (
                          <tr key={sub.subject_id} className="border-t">
                            <td className="p-2 font-medium text-gray-700">{sub.subject_name}</td>
                            {(["cat1", "cat2", "cat3"] as const).map((cat) => {
                              const entry = sub[cat];
                              const level = getLevel(st.student_id, entry.assessment_id, entry.level);
                              return (
                                <td key={cat} className="p-2 text-center">
                                  <select
                                    value={level}
                                    onChange={(e) => setLevel(st.student_id, entry.assessment_id, e.target.value)}
                                    className={`text-xs font-semibold px-2 py-1 rounded border cursor-pointer ${LEVEL_COLORS[level]}`}
                                    disabled={!entry.assessment_id}
                                  >
                                    {LEVELS.map((l) => <option key={l} value={l}>{l || "–"}</option>)}
                                  </select>
                                </td>
                              );
                            })}
                            <td className="p-2">
                              <input
                                type="text"
                                placeholder="Short comment…"
                                value={getReflection(st.student_id, sub.cat1.assessment_id, sub.cat1.reflection)}
                                onChange={(e) => setReflection(st.student_id, sub.cat1.assessment_id, e.target.value)}
                                className="w-full text-xs border rounded px-2 py-1"
                              />
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </Card>
            );
          })}

          {hasPending && (
            <div className="sticky bottom-4 flex justify-end">
              <Button onClick={handleSaveAll} disabled={save.isPending} className="shadow-lg">
                <Save size={14} className="mr-1" />
                {save.isPending ? "Saving…" : "Save all changes"}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
