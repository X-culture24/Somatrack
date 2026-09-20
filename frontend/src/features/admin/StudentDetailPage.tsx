import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { studentsApi } from "../../api/students";
import { Card, PageHeader, kes } from "../../components/ui";

export function StudentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: student, isLoading } = useQuery({
    queryKey: ["student", id],
    queryFn: () => studentsApi.get(Number(id)),
  });

  if (isLoading) return <div className="p-8 text-center text-slate-400">Loading…</div>;
  if (!student) return null;

  return (
    <div className="space-y-5 max-w-2xl">
      <PageHeader title={student.full_name} subtitle={`Admission No: ${student.admission_no}`} />
      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">Profile</p>
          <dl className="space-y-1.5 text-sm">
            <div className="flex justify-between"><dt className="text-slate-500">Class</dt><dd className="font-medium">{student.class_label}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Gender</dt><dd className="font-medium">{student.gender === "M" ? "Male" : "Female"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Status</dt><dd className="font-medium capitalize">{student.status}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Enrolled</dt><dd className="font-medium">{student.enrollment_date ?? "—"}</dd></div>
          </dl>
        </Card>
        <Card>
          <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">Fees</p>
          <p className="text-3xl font-bold text-[#1b365d]">{kes(student.fee_balance)}</p>
          <p className="text-xs text-slate-400 mt-1">Current term balance</p>
        </Card>
      </div>
    </div>
  );
}
