import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { studentsApi } from "../../api/students";
import { PageHeader } from "../../components/ui";

export function SiblingsPage() {
  const navigate = useNavigate();
  const { data, isLoading } = useQuery({
    queryKey: ["siblings"],
    queryFn: () => studentsApi.siblings(),
  });

  return (
    <div className="p-6">
      <PageHeader
        title="Siblings"
        subtitle="Parents / guardians with more than one enrolled child"
      />

      {isLoading && (
        <p className="py-8 text-center text-sm text-slate-500">Loading…</p>
      )}

      {!isLoading && (data ?? []).length === 0 && (
        <div className="mt-8 rounded-lg border border-dashed border-slate-200 py-12 text-center">
          <p className="text-sm text-slate-400">No siblings found.</p>
          <p className="mt-1 text-xs text-slate-300">Guardians linked to 2+ active students appear here.</p>
        </div>
      )}

      <div className="mt-4 space-y-4">
        {(data ?? []).map((group: any, i: number) => (
          <div key={group.guardian_id} className="rounded-lg border border-slate-200 bg-white shadow-sm">
            <div className="flex items-center gap-3 border-b border-slate-100 bg-slate-50 px-4 py-3 rounded-t-lg">
              <span className="flex h-7 w-7 items-center justify-center rounded-full bg-[#1b365d] text-xs font-bold text-white">
                {i + 1}
              </span>
              <div>
                <span className="font-semibold text-slate-800">{group.full_name}</span>
                <span className="ml-3 text-sm text-slate-500">{group.phone}</span>
              </div>
              <span className="ml-auto rounded-full bg-[#1b365d]/10 px-2.5 py-0.5 text-xs font-medium text-[#1b365d]">
                {group.students?.length ?? 0} children
              </span>
            </div>
            <div className="divide-y divide-slate-50">
              {(group.students ?? []).map((s: any) => (
                <div key={s.student_id} className="flex items-center gap-4 px-4 py-2.5">
                  <span className="text-sm text-slate-500 w-12 shrink-0">{s.admission_no}</span>
                  <span className="text-sm font-medium text-slate-800 flex-1">{s.full_name}</span>
                  <span className="text-sm text-slate-400">{s.class_name || "—"}</span>
                  <button
                    onClick={() => navigate(`/admin/students/${s.student_id}`)}
                    className="rounded bg-[#1b365d] px-2.5 py-1 text-xs font-semibold text-white hover:bg-[#152a49]"
                  >
                    View
                  </button>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
