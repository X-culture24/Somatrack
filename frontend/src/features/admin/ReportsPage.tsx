import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { academicsApi } from "../../api/academics";
import { Card, Field, PageHeader, Select, Stat, kes } from "../../components/ui";

export function ReportsPage() {
  const [classId, setClassId] = useState("");
  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: feeReport } = useQuery({ queryKey: ["report-fees"], queryFn: schoolApi.reportFeeCollection });
  const { data: attendanceReport } = useQuery({ queryKey: ["report-attendance"], queryFn: () => schoolApi.reportAttendance() });
  const { data: perfReport, isLoading: perfLoading } = useQuery({
    queryKey: ["report-perf", classId],
    enabled: Boolean(classId),
    queryFn: () => schoolApi.reportClassPerformance(classId),
  });

  return (
    <div className="space-y-6">
      <PageHeader title="Reports & analytics" subtitle="Fee collection, attendance rates, and academic performance." />

      {/* Fee collection */}
      <section>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Fee collection — current term</h3>
        {feeReport ? (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Stat label="Total billed" value={kes(feeReport.total_billed)} />
            <Stat label="Total collected" value={kes(feeReport.total_collected)} />
            <Stat label="Outstanding" value={kes(feeReport.total_balance)} />
            <Stat label="Collection rate" value={`${feeReport.collection_rate}%`} />
          </div>
        ) : <p className="text-sm text-slate-400">Loading…</p>}

        {feeReport?.invoices_by_status && (
          <Card className="mt-3">
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Invoices by status</p>
            <div className="flex flex-wrap gap-3">
              {Object.entries(feeReport.invoices_by_status).map(([status, count]: any) => (
                <div key={status} className="rounded-xl border border-slate-200 px-4 py-2 text-center">
                  <p className="text-xl font-bold text-[#1b365d]">{count}</p>
                  <p className="text-xs text-slate-500 capitalize">{status}</p>
                </div>
              ))}
            </div>
          </Card>
        )}
      </section>

      {/* Attendance summary */}
      <section>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Attendance — current term</h3>
        {attendanceReport ? (
          <div className="grid gap-3 sm:grid-cols-3">
            <Stat label="Total records" value={attendanceReport.total_records} />
            <Stat label="Present rate" value={`${attendanceReport.present_rate}%`} />
            <Card>
              <p className="text-xs uppercase tracking-wide text-slate-500 mb-2">By status</p>
              {Object.entries(attendanceReport.by_status ?? {}).map(([s, c]: any) => (
                <div key={s} className="flex justify-between text-sm py-0.5">
                  <span className="capitalize text-slate-600">{s}</span>
                  <span className="font-semibold text-[#1b365d]">{c}</span>
                </div>
              ))}
            </Card>
          </div>
        ) : <p className="text-sm text-slate-400">Loading…</p>}
      </section>

      {/* Class performance */}
      <section>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Class performance</h3>
        <div className="mb-4 w-56">
          <Field label="Select class">
            <Select value={classId} onChange={(e) => setClassId(e.target.value)}>
              <option value="">Select class…</option>
              {(classes as any[] ?? []).map((c: any) => (
                <option key={c.id} value={c.id}>{c.grade} {c.stream_name}</option>
              ))}
            </Select>
          </Field>
        </div>

        {classId && perfLoading && <p className="text-sm text-slate-400">Loading…</p>}

        {classId && perfReport && (
          <div className="space-y-2">
            {(perfReport as any[]).map((a: any) => (
              <Card key={a.assessment_id} className="flex items-center justify-between gap-4">
                <div>
                  <p className="font-medium text-[#1b365d]">{a.title}</p>
                  <p className="text-xs text-slate-500">{a.subject} · {a.kind} · max {a.max_score}</p>
                </div>
                <div className="text-right">
                  <p className="text-2xl font-bold text-[#1b365d]">{a.average_score}</p>
                  <p className="text-xs text-slate-400">{a.submissions} submissions</p>
                </div>
              </Card>
            ))}
            {(perfReport as any[]).length === 0 && (
              <p className="text-sm text-slate-400">No assessments recorded for this class yet.</p>
            )}
          </div>
        )}
      </section>
    </div>
  );
}
