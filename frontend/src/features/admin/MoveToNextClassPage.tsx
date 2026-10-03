import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { academicsApi } from "../../api/academics";
import { studentsApi } from "../../api/students";
import { Button, PageHeader, Select } from "../../components/ui";

export function MoveToNextClassPage() {
  const qc = useQueryClient();
  const [selectedClass, setSelectedClass] = useState("");
  const [targetClass, setTargetClass] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const { data: classes } = useQuery({
    queryKey: ["classes"],
    queryFn: () => academicsApi.classes(),
  });
  const { data: students, isLoading } = useQuery({
    queryKey: ["students-for-promote", selectedClass],
    queryFn: () => studentsApi.list(selectedClass ? { class_id: selectedClass } : {}),
    enabled: true,
  });

  const promote = useMutation({
    mutationFn: () =>
      studentsApi.promote({
        student_ids: [...selected],
        target_class_id: targetClass,
      }),
    onSuccess: () => {
      setSelected(new Set());
      qc.invalidateQueries({ queryKey: ["students-for-promote"] });
      alert(`${selected.size} students promoted successfully.`);
    },
  });

  const allIds = (students ?? []).map((s: any) => s.id as string);
  const allSelected = allIds.length > 0 && allIds.every((id) => selected.has(id));

  function toggleAll() {
    if (allSelected) {
      setSelected(new Set());
    } else {
      setSelected(new Set(allIds));
    }
  }

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }

  return (
    <div className="p-6">
      <PageHeader title="Move Students to Next Class" subtitle="Select students and a target class, then promote." />

      <div className="mb-5 flex flex-wrap items-end gap-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Filter by current class</label>
          <Select value={selectedClass} onChange={(e) => { setSelectedClass(e.target.value); setSelected(new Set()); }} className="w-52">
            <option value="">All classes</option>
            {(classes ?? []).map((c: any) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </Select>
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Promote to class</label>
          <Select value={targetClass} onChange={(e) => setTargetClass(e.target.value)} className="w-52">
            <option value="">Select target class…</option>
            {(classes ?? []).map((c: any) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </Select>
        </div>
        <Button
          onClick={() => promote.mutate()}
          disabled={selected.size === 0 || !targetClass || promote.isPending}
          className="bg-green-700 hover:bg-green-800"
        >
          {promote.isPending ? "Promoting…" : `Promote ${selected.size} student${selected.size === 1 ? "" : "s"}`}
        </Button>
        <span className="ml-auto text-sm text-slate-500">
          {selected.size} selected of {(students ?? []).length}
        </span>
      </div>

      {isLoading ? (
        <p className="py-8 text-center text-sm text-slate-500">Loading students…</p>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-slate-200">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-[#37474f] text-white text-xs uppercase tracking-wide">
                <th className="px-4 py-3 text-left w-10">
                  <input type="checkbox" checked={allSelected} onChange={toggleAll}
                    className="rounded border-slate-400" />
                </th>
                <th className="px-4 py-3 text-left">#</th>
                <th className="px-4 py-3 text-left">Student Name</th>
                <th className="px-4 py-3 text-left">Adm No.</th>
                <th className="px-4 py-3 text-left">Current Class</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {(students ?? []).map((s: any, i: number) => (
                <tr key={s.id} className={`hover:bg-slate-50 ${selected.has(s.id) ? "bg-blue-50" : ""}`}>
                  <td className="px-4 py-2.5">
                    <input type="checkbox" checked={selected.has(s.id)} onChange={() => toggle(s.id)}
                      className="rounded border-slate-300" />
                  </td>
                  <td className="px-4 py-2.5 text-slate-400">{i + 1}</td>
                  <td className="px-4 py-2.5 font-medium text-slate-800">{s.full_name}</td>
                  <td className="px-4 py-2.5 text-slate-600">{s.admission_no}</td>
                  <td className="px-4 py-2.5 text-slate-600">{s.class_name || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {(students ?? []).length === 0 && (
            <div className="py-10 text-center text-sm text-slate-400">No students found.</div>
          )}
        </div>
      )}

      {/* Right panel: class summary */}
    </div>
  );
}
