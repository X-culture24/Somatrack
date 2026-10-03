import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Download } from "lucide-react";
import { resultsApi, type TermResult } from "../../api/results";
import { Button } from "../../components/ui";

// A student's results for every term since they joined, newest first. Each
// term is labelled with the class the student was in *that* term, so the
// trail reads correctly after promotions. Shared by the parent portal and
// the staff student profile.

const GRADE_COLOR: Record<string, string> = {
  EE: "bg-green-100 text-green-700",
  ME: "bg-blue-100 text-blue-700",
  AE: "bg-yellow-100 text-yellow-700",
  BE: "bg-red-100 text-red-700",
};

function GradeChip({ grade }: { grade: string }) {
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${GRADE_COLOR[grade] ?? "bg-slate-100 text-slate-600"}`}>
      {grade || "—"}
    </span>
  );
}

// Mean change vs the previous term, e.g. "▲ 4.5".
function Trend({ now, before }: { now: number; before?: number }) {
  if (before === undefined) return null;
  const d = Math.round((now - before) * 10) / 10;
  if (d === 0) return <span className="text-xs text-slate-400">no change</span>;
  return (
    <span className={`text-xs font-medium ${d > 0 ? "text-green-600" : "text-red-600"}`}>
      {d > 0 ? "▲" : "▼"} {Math.abs(d)}
    </span>
  );
}

export function ResultsTrail({ studentId }: { studentId: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["results-history", studentId],
    queryFn: () => resultsApi.history(studentId),
  });
  const [downloading, setDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState("");

  if (isLoading) {
    return <div className="space-y-2">{[1, 2].map((i) => <div key={i} className="h-24 animate-pulse rounded-xl bg-slate-200" />)}</div>;
  }
  if (isError || !data) {
    return <p className="py-6 text-center text-sm text-red-600">Could not load results.</p>;
  }

  // Oldest-first from the API; show newest first, trend against the term before.
  const terms: (TermResult & { prev?: number })[] = data.terms.map((t, i) => ({ ...t, prev: data.terms[i - 1]?.mean }));
  terms.reverse();

  async function downloadReportCard(t: TermResult) {
    setDownloadError("");
    try {
      await resultsApi.downloadReportCard(data!.student_id, t.term_id, `report-card-${data!.admission_no}-${t.term_name}-${t.academic_year}.pdf`);
    } catch {
      setDownloadError(`Could not download the ${t.term_name} report card. Try again.`);
    }
  }

  async function download() {
    setDownloading(true);
    setDownloadError("");
    try {
      await resultsApi.downloadTranscript(data!.student_id, data!.admission_no);
    } catch {
      setDownloadError("Could not download the record. Try again.");
    } finally {
      setDownloading(false);
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-slate-600">
          {terms.length ? `${terms.length} term${terms.length === 1 ? "" : "s"} on record` : "No results recorded yet."}
        </p>
        {terms.length > 0 && (
          <Button variant="ghost" onClick={download} disabled={downloading}>
            <Download size={15} /> {downloading ? "Preparing…" : "Download academic record (PDF)"}
          </Button>
        )}
      </div>
      {downloadError && <p className="text-sm text-red-600">{downloadError}</p>}

      <ol className="relative space-y-3 border-l-2 border-slate-200 pl-4">
        {terms.map((t) => (
          <li key={`${t.term_id}-${t.class_id}`} className="relative">
            <span className="absolute top-4 -left-[23px] h-3 w-3 rounded-full border-2 border-white bg-[#4a9eca]" />
            <div className="rounded-xl border border-slate-200 bg-white p-3 shadow-sm">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <p className="font-semibold text-[#1b365d]">{t.term_name} {t.academic_year}</p>
                  <p className="text-xs text-slate-500">{t.class_name}</p>
                </div>
                <div className="text-right">
                  <p className="text-lg font-bold text-[#1b365d]">{t.mean.toFixed(1)}% <GradeChip grade={t.mean_grade} /></p>
                  <Trend now={t.mean} before={t.prev} />
                  <button
                    type="button"
                    onClick={() => downloadReportCard(t)}
                    className="mt-1 flex items-center gap-1 text-xs font-medium text-[#4a9eca] hover:underline"
                  >
                    <Download size={12} /> Report card
                  </button>
                </div>
              </div>
              <table className="mt-2 w-full text-sm">
                <tbody className="divide-y divide-slate-100">
                  {t.subjects.map((s) => (
                    <tr key={s.subject_id}>
                      <td className="py-1.5 text-slate-700">{s.subject_name}</td>
                      <td className="py-1.5 text-right tabular-nums text-slate-600">{s.percentage.toFixed(1)}%</td>
                      <td className="w-14 py-1.5 text-right"><GradeChip grade={s.grade} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </li>
        ))}
      </ol>
    </div>
  );
}
