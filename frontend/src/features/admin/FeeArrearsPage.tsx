import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { academicsApi } from "../../api/academics";
import { studentsApi } from "../../api/students";
import { Input, PageHeader, Select, Table, kes } from "../../components/ui";

export function FeeArrearsPage() {
  const navigate = useNavigate();
  const [classFilter, setClassFilter] = useState("");
  const [q, setQ] = useState("");

  const { data: classes } = useQuery({
    queryKey: ["classes"],
    queryFn: () => academicsApi.classes(),
  });
  const { data, isLoading } = useQuery({
    queryKey: ["fee-arrears", classFilter],
    queryFn: () => studentsApi.feeArrears(classFilter ? { class_id: classFilter } : {}),
  });

  const filtered = (data ?? []).filter((s: any) =>
    !q || s.full_name?.toLowerCase().includes(q.toLowerCase()) || s.admission_no?.includes(q)
  );

  const total = filtered.reduce((sum: number, s: any) => sum + (s.outstanding ?? 0), 0);

  return (
    <div className="p-6">
      <PageHeader
        title="Fee Arrears"
        subtitle="Students with outstanding fee balance"
      />

      <div className="mb-5 flex flex-wrap items-end gap-3">
        <Input
          className="max-w-xs"
          placeholder="Search name or adm no."
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <Select value={classFilter} onChange={(e) => setClassFilter(e.target.value)} className="w-48">
          <option value="">All classes</option>
          {(classes ?? []).map((c: any) => (
            <option key={c.id} value={c.id}>{c.name}</option>
          ))}
        </Select>
        <div className="ml-auto flex items-center gap-4 rounded-lg border border-red-200 bg-red-50 px-4 py-2">
          <span className="text-sm text-red-700 font-medium">Total outstanding:</span>
          <span className="text-lg font-bold text-red-700">{kes(total)}</span>
        </div>
      </div>

      {isLoading ? (
        <p className="py-8 text-center text-sm text-slate-500">Loading fee arrears…</p>
      ) : (
        <Table headers={["#", "Adm No.", "Name", "Class", "Parent", "Parent Phone", "Outstanding", ""]}>
          {filtered.map((s: any, i: number) => (
            <tr key={s.student_id} className="hover:bg-slate-50">
              <td className="px-3 py-2.5 text-slate-400 text-sm">{i + 1}</td>
              <td className="px-3 py-2.5 font-medium text-[#1b365d] text-sm">{s.admission_no}</td>
              <td className="px-3 py-2.5 font-medium text-sm">{s.full_name}</td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.class_name || "—"}</td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.parent_name || "—"}</td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.parent_phone || "—"}</td>
              <td className="px-3 py-2.5 text-sm font-bold text-red-600">{kes(s.outstanding)}</td>
              <td className="px-3 py-2.5">
                <button
                  onClick={() => navigate(`/admin/students/${s.student_id}`)}
                  className="rounded bg-[#1b365d] px-2.5 py-1 text-xs font-semibold text-white hover:bg-[#152a49]"
                >
                  Profile
                </button>
              </td>
            </tr>
          ))}
        </Table>
      )}

      {!isLoading && filtered.length === 0 && (
        <div className="mt-8 rounded-lg border border-dashed border-slate-200 py-12 text-center">
          <p className="text-sm text-slate-400">No fee arrears found.</p>
          <p className="mt-1 text-xs text-slate-300">All students are up to date with their fees.</p>
        </div>
      )}
    </div>
  );
}
