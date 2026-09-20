import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { Card, Field, PageHeader, Select } from "../../components/ui";

const GRADES = [
  "Playgroup", "Pre-Primary 1", "Pre-Primary 2",
  "Grade 1", "Grade 2", "Grade 3",
  "Grade 4", "Grade 5", "Grade 6",
  "Grade 7", "Grade 8", "Grade 9",
];

const AREA_COLORS: Record<string, string> = {
  Languages: "bg-blue-100 text-blue-700",
  STEM: "bg-green-100 text-green-700",
  Humanities: "bg-amber-100 text-amber-700",
  Arts: "bg-purple-100 text-purple-700",
  Wellbeing: "bg-teal-100 text-teal-700",
  General: "bg-slate-100 text-slate-600",
};

export function CurriculumPage() {
  const [selectedGrade, setSelectedGrade] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["curriculum", selectedGrade],
    queryFn: () => schoolApi.curriculum(selectedGrade || undefined),
  });

  const rows: any[] = data ?? [];

  return (
    <div className="space-y-6">
      <PageHeader
        title="CBC Curriculum"
        subtitle="Subject assignments per grade and learning area for the current academic year."
      />

      <div className="w-56">
        <Field label="Filter by grade">
          <Select value={selectedGrade} onChange={(e) => setSelectedGrade(e.target.value)}>
            <option value="">All grades</option>
            {GRADES.map((g) => <option key={g} value={g}>{g}</option>)}
          </Select>
        </Field>
      </div>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => <div key={i} className="h-32 animate-pulse rounded-2xl bg-slate-200" />)}
        </div>
      )}

      {rows.map((gradeRow: any) => (
        <Card key={gradeRow.grade}>
          <div className="mb-3 flex items-center gap-3">
            <h2 className="text-base font-semibold text-[#1b365d]">{gradeRow.grade}</h2>
            <span className="rounded-full bg-[#1b365d]/10 px-2 py-0.5 text-xs font-medium text-[#1b365d]">
              Uniform band: {gradeRow.band}
            </span>
          </div>
          <div className="space-y-3">
            {gradeRow.learning_areas.map((la: any) => (
              <div key={la.area}>
                <p className="mb-1.5 text-xs font-semibold uppercase tracking-wider text-slate-500">
                  {la.area}
                </p>
                <div className="flex flex-wrap gap-2">
                  {la.subjects.map((s: any) => (
                    <div
                      key={s.id}
                      className={`rounded-xl px-3 py-1.5 text-xs font-medium ${AREA_COLORS[la.area] ?? AREA_COLORS.General}`}
                      title={s.teacher ? `Teacher: ${s.teacher}` : "No teacher assigned"}
                    >
                      <span>{s.name}</span>
                      {s.teacher && (
                        <span className="ml-1.5 opacity-70">· {s.teacher.split(" ")[0]}</span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </Card>
      ))}

      {!isLoading && rows.length === 0 && (
        <p className="rounded-2xl border border-slate-200 bg-white py-12 text-center text-sm text-slate-400">
          No curriculum data found. Run the seed command or assign subjects to classes.
        </p>
      )}
    </div>
  );
}
