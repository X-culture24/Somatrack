import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { studentsApi } from "../../api/students";
import { Input, PageHeader, Table, kes } from "../../components/ui";

export function StudentsPage() {
  const [q, setQ] = useState("");
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ["students", q],
    queryFn: () => studentsApi.list({ search: q }),
  });

  return (
    <div>
      <PageHeader
        title="Students"
        subtitle="Profiles, class placement, transport, and fee balance."
      />
      <Input
        className="mb-4 max-w-sm"
        placeholder="Search name or admission no."
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      {isLoading ? (
        <p className="py-8 text-center text-sm text-slate-500">Loading students…</p>
      ) : (
        <Table headers={["Adm no.", "Name", "Class", "Status", "Balance"]}>
          {(data ?? []).map((s: any) => (
            <tr
              key={s.id}
              className="cursor-pointer hover:bg-slate-50"
              onClick={() => navigate(`/admin/students/${s.id}`)}
            >
              <td className="px-4 py-3 font-medium text-[#1b365d]">{s.admission_no}</td>
              <td className="px-4 py-3">{s.full_name}</td>
              <td className="px-4 py-3">{s.class_label}</td>
              <td className="px-4 py-3">
                <span
                  className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${
                    s.status === "active"
                      ? "bg-green-100 text-green-700"
                      : s.status === "transferred"
                      ? "bg-amber-100 text-amber-700"
                      : s.status === "withdrawn"
                      ? "bg-red-100 text-red-700"
                      : "bg-slate-100 text-slate-600"
                  }`}
                >
                  {s.status}
                </span>
              </td>
              <td
                className={`px-4 py-3 font-medium ${
                  Number(s.fee_balance) > 0 ? "text-red-600" : "text-green-600"
                }`}
              >
                {kes(s.fee_balance)}
              </td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  );
}
