import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { feeStructuresApi } from "../../api/finance";
import { Card, Field, PageHeader, Select, Stat, Table, kes } from "../../components/ui";

const GRADES = [
  "Playgroup", "Pre-Primary 1", "Pre-Primary 2",
  "Grade 1", "Grade 2", "Grade 3",
  "Grade 4", "Grade 5", "Grade 6",
  "Grade 7", "Grade 8", "Grade 9",
];

const GRADE_BAND: Record<string, string> = {
  "Playgroup": "Kinder", "Pre-Primary 1": "Kinder", "Pre-Primary 2": "Kinder",
  "Grade 1": "1-3", "Grade 2": "1-3", "Grade 3": "1-3",
  "Grade 4": "4-6", "Grade 5": "4-6", "Grade 6": "4-6",
  "Grade 7": "JSS", "Grade 8": "JSS", "Grade 9": "JSS",
};

export function FeeStructuresPage() {
  const [selectedGrade, setSelectedGrade] = useState("");
  const { data: all } = useQuery({ queryKey: ["fees"], queryFn: () => feeStructuresApi.list() });

  const rows = (all as any[] ?? []);
  const filtered = selectedGrade
    ? rows.filter((f: any) => f.grade === selectedGrade)
    : rows;

  // Compute per-grade annual totals for the selected grade
  const gradeFees = selectedGrade
    ? rows.filter((f: any) => f.grade === selectedGrade).sort((a: any, b: any) => a.term_number - b.term_number)
    : [];

  const annualNew = gradeFees.reduce(
    (s: number, f: any) => s + Number(f.admission_fee) + Number(f.school_fees) + Number(f.diary_progress_book), 0
  );
  const annualContinuing = gradeFees.reduce(
    (s: number, f: any) => s + Number(f.school_fees) + Number(f.diary_progress_book), 0
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title="Fee structures 2026"
        subtitle="Per-grade, per-term breakdown. Admission billed Term 1 only for new intake."
      />

      {/* Grade selector */}
      <Card>
        <div className="flex flex-wrap items-end gap-4">
          <div className="w-56">
            <Field label="Filter by grade">
              <Select value={selectedGrade} onChange={(e) => setSelectedGrade(e.target.value)}>
                <option value="">All grades</option>
                {GRADES.map((g) => <option key={g} value={g}>{g}</option>)}
              </Select>
            </Field>
          </div>
          {selectedGrade && (
            <div className="text-sm text-slate-500">
              Uniform band: <span className="font-semibold text-[#1b365d]">{GRADE_BAND[selectedGrade] ?? "—"}</span>
            </div>
          )}
        </div>

        {/* Annual summary for selected grade */}
        {selectedGrade && gradeFees.length > 0 && (
          <div className="mt-4 grid gap-3 sm:grid-cols-3">
            <Stat label="Annual total (new intake)" value={kes(annualNew)} />
            <Stat label="Annual total (continuing)" value={kes(annualContinuing)} />
            <Stat label="Uniform band" value={GRADE_BAND[selectedGrade] ?? "—"} />
          </div>
        )}
      </Card>

      {/* Detailed table */}
      <Table headers={["Grade", "Term", "Admission fee", "School fees", "Diary & book", "New-intake total", "Continuing total"]}>
        {filtered.map((f: any) => {
          const newTotal = Number(f.admission_fee) + Number(f.school_fees) + Number(f.diary_progress_book);
          const contTotal = Number(f.school_fees) + Number(f.diary_progress_book);
          return (
            <tr key={f.id} className="hover:bg-slate-50">
              <td className="px-4 py-3 font-medium text-[#1b365d]">{f.grade}</td>
              <td className="px-4 py-3">
                <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium">Term {f.term_number}</span>
              </td>
              <td className="px-4 py-3">
                {Number(f.admission_fee) > 0 ? kes(f.admission_fee) : <span className="text-slate-400">Free</span>}
              </td>
              <td className="px-4 py-3">{kes(f.school_fees)}</td>
              <td className="px-4 py-3">
                {Number(f.diary_progress_book) > 0 ? kes(f.diary_progress_book) : <span className="text-slate-400">—</span>}
              </td>
              <td className="px-4 py-3 font-semibold text-[#1b365d]">{kes(newTotal)}</td>
              <td className="px-4 py-3 text-slate-600">{kes(contTotal)}</td>
            </tr>
          );
        })}
      </Table>

      {/* Notes */}
      <Card>
        <h3 className="font-semibold text-[#1b365d]">Notes</h3>
        <ul className="mt-2 space-y-1 text-sm text-slate-600 list-disc list-inside">
          <li>Admission fee is billed once — Term 1 only, for new intake learners.</li>
          <li>Diary &amp; progress book is billed once at Term 1 only.</li>
          <li>Playgroup admission is <strong>free</strong>.</li>
          <li>Transport charges are billed separately based on route (monthly × 3).</li>
          <li>Uniform pricing varies by band: Kinder, 1-3, 4-6, JSS.</li>
          <li>Pay via M-Pesa Paybill <strong>247247</strong>, account <strong>137101#CHILDNAME</strong>.</li>
        </ul>
      </Card>
    </div>
  );
}
