import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { ResultsTrail } from "../results/ResultsTrail";

type Child = { student_id: string; full_name: string; admission_no: string; class_name?: string };

// Results for every term each child has been at the school, newest first.
export function ParentProgressPage() {
  const { data, isLoading } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const kids: Child[] = data?.children ?? [];
  const [picked, setPicked] = useState<string>("");
  const current = kids.find((k) => k.student_id === picked) ?? kids[0];

  return (
    <div className="p-4">
      <h1 className="text-xl font-semibold text-[#1b365d]">Academic progress</h1>
      <p className="mt-0.5 text-xs text-slate-500">Results for every term, with the class your child was in at the time.</p>

      {isLoading && <div className="mt-4 h-24 animate-pulse rounded-xl bg-slate-200" />}
      {!isLoading && kids.length === 0 && (
        <p className="py-8 text-center text-sm text-slate-400">No children are linked to your account yet. Contact the school office.</p>
      )}

      {kids.length > 1 && (
        <div className="mt-4 flex gap-2 overflow-x-auto">
          {kids.map((k) => (
            <button
              key={k.student_id}
              onClick={() => setPicked(k.student_id)}
              className={`shrink-0 rounded-full px-3 py-1.5 text-sm font-medium ${
                current?.student_id === k.student_id ? "bg-[#1b365d] text-white" : "bg-slate-100 text-slate-600"
              }`}
            >
              {k.full_name.split(" ")[0]}
            </button>
          ))}
        </div>
      )}

      {current && (
        <div className="mt-4">
          <p className="mb-3 text-sm font-medium text-slate-700">
            {current.full_name} <span className="text-slate-400">· {current.admission_no}{current.class_name ? ` · ${current.class_name}` : ""}</span>
          </p>
          <ResultsTrail studentId={current.student_id} />
        </div>
      )}
    </div>
  );
}
