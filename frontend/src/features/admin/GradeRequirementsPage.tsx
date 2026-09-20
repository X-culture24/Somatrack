import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { BookOpen, Package } from "lucide-react";
import { schoolApi } from "../../api/school";
import { Card, Field, PageHeader, Select } from "../../components/ui";

const GRADES = [
  "Playgroup", "Pre-Primary 1", "Pre-Primary 2",
  "Grade 1", "Grade 2", "Grade 3",
  "Grade 4", "Grade 5", "Grade 6",
  "Grade 7", "Grade 8", "Grade 9",
];

export function GradeRequirementsPage() {
  const [grade, setGrade] = useState("Grade 4");

  const { data, isLoading } = useQuery({
    queryKey: ["grade-req", grade],
    queryFn: () => schoolApi.gradeRequirements(grade),
    select: (rows: any[]) => rows[0] ?? null,
  });

  return (
    <div className="space-y-5">
      <PageHeader
        title="Class requirements & textbooks"
        subtitle="Official stationery list and textbooks per grade — from the 2026 fee structure document."
      />

      <div className="w-56">
        <Field label="Select grade">
          <Select value={grade} onChange={(e) => setGrade(e.target.value)}>
            {GRADES.map((g) => <option key={g} value={g}>{g}</option>)}
          </Select>
        </Field>
      </div>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2].map((i) => <div key={i} className="h-40 animate-pulse rounded-2xl bg-slate-200" />)}
        </div>
      )}

      {data && (
        <div className="grid gap-5 lg:grid-cols-2">
          {/* Class requirements */}
          <Card>
            <div className="mb-3 flex items-center gap-2 text-[#1b365d]">
              <Package size={18} />
              <h3 className="font-semibold">Class requirements ({data.class_requirements?.length ?? 0} items)</h3>
            </div>
            {data.class_requirements?.length > 0 ? (
              <ul className="space-y-1.5">
                {data.class_requirements.map((item: string, i: number) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-slate-700">
                    <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[#1b365d]/10 text-xs font-semibold text-[#1b365d]">
                      {i + 1}
                    </span>
                    {item}
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-slate-400">No specific requirements listed.</p>
            )}
          </Card>

          {/* Textbooks */}
          <Card>
            <div className="mb-3 flex items-center gap-2 text-[#1b365d]">
              <BookOpen size={18} />
              <h3 className="font-semibold">Textbooks ({data.textbooks?.length ?? 0} titles)</h3>
            </div>
            {data.textbooks?.length > 0 ? (
              <ul className="space-y-1.5">
                {data.textbooks.map((book: string, i: number) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-slate-700">
                    <span className="mt-0.5 text-[#4a9eca]">📗</span>
                    {book}
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-slate-400">No textbooks listed for this grade.</p>
            )}
          </Card>
        </div>
      )}

      {!isLoading && !data && (
        <p className="rounded-2xl border border-slate-200 bg-white py-10 text-center text-sm text-slate-400">
          No requirements found for {grade}. Run the seed command to populate data.
        </p>
      )}
    </div>
  );
}
