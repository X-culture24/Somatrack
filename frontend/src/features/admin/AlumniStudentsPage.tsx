import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { studentsApi } from "../../api/students";
import { Input, PageHeader, Table, kes } from "../../components/ui";
import { useState } from "react";

export function AlumniStudentsPage() {
  const navigate = useNavigate();
  const [q, setQ] = useState("");
  const { data, isLoading } = useQuery({
    queryKey: ["alumni-students", q],
    queryFn: () => studentsApi.list({ alumni: "true", search: q }),
  });

  return (
    <div className="p-6">
      <PageHeader title="Alumni Students" subtitle="Graduated or inactive students" />

      <div className="mb-4 flex items-center gap-3">
        <Input
          className="max-w-sm"
          placeholder="Search name or admission no."
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <span className="text-sm text-slate-500">
          {data ? `${data.length} alumni found` : ""}
        </span>
      </div>

      {isLoading ? (
        <p className="py-8 text-center text-sm text-slate-500">Loading alumni…</p>
      ) : (
        <Table headers={["#", "Adm No.", "Name", "Gender", "Class", "Status", "Parent", "Phone", ""]}>
          {(data ?? []).map((s: any, i: number) => (
            <tr key={s.id} className="hover:bg-slate-50">
              <td className="px-3 py-2.5 text-slate-400 text-sm">{i + 1}</td>
              <td className="px-3 py-2.5 font-medium text-[#1b365d] text-sm">{s.admission_no}</td>
              <td className="px-3 py-2.5 font-medium text-sm">{s.full_name}</td>
              <td className="px-3 py-2.5 text-sm capitalize text-slate-600">{s.gender}</td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.class_name || "—"}</td>
              <td className="px-3 py-2.5">
                <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${
                  s.status === "graduated" ? "bg-blue-100 text-blue-700"
                  : s.status === "transferred" ? "bg-amber-100 text-amber-700"
                  : s.status === "withdrawn" ? "bg-red-100 text-red-700"
                  : "bg-slate-100 text-slate-600"}`}>
                  {s.status}
                </span>
              </td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.parent_name || "—"}</td>
              <td className="px-3 py-2.5 text-sm text-slate-600">{s.parent_phone || "—"}</td>
              <td className="px-3 py-2.5">
                <button
                  onClick={() => navigate(`/admin/students/${s.id}`)}
                  className="rounded bg-[#1b365d] px-2.5 py-1 text-xs font-semibold text-white hover:bg-[#152a49]"
                >
                  View
                </button>
              </td>
            </tr>
          ))}
        </Table>
      )}

      {!isLoading && (data ?? []).length === 0 && (
        <div className="mt-8 rounded-lg border border-dashed border-slate-200 py-12 text-center">
          <p className="text-sm text-slate-400">No alumni found.</p>
          <p className="mt-1 text-xs text-slate-300">
            Students marked as graduated or inactive will appear here.
          </p>
        </div>
      )}
    </div>
  );
}
