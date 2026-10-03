import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  PieChart, Pie, Cell, LineChart, Line,
} from "recharts";
import { schoolApi } from "../../api/school";
import { academicsApi } from "../../api/academics";
import { api } from "../../api/client";
import { Card, Field, Input, PageHeader, Select, Stat, Table, kes } from "../../components/ui";

function todayISO() { return new Date().toISOString().slice(0, 10); }
function daysAgoISO(n: number) {
  const d = new Date(); d.setDate(d.getDate() - n); return d.toISOString().slice(0, 10);
}

const LEVEL_COLORS: Record<string, string> = {
  EE: "#16a34a",
  ME: "#2563eb",
  AE: "#f59e0b",
  BE: "#dc2626",
};
const LEVEL_LABELS: Record<string, string> = {
  EE: "Exceeds Expectation",
  ME: "Meets Expectation",
  AE: "Approaches Expectation",
  BE: "Below Expectation",
};

function unwrap(d: unknown) {
  if (d && typeof d === "object") {
    if ("results" in (d as Record<string, unknown>)) return (d as { results: unknown }).results;
    if ("data" in (d as Record<string, unknown>)) return (d as { data: unknown }).data;
  }
  return d;
}

const cbcApi = {
  distribution: (term_id?: string) =>
    api.get("/cbc/analytics/distribution/", { params: term_id ? { term_id } : {} })
      .then(r => unwrap(r.data) as { level: string; count: number; percent: number }[]),
};

function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h3 className="mb-3 font-semibold text-[#1b365d] text-base">{children}</h3>;
}

export function ReportsPage() {
  const [classId, setClassId] = useState("");
  const [termId, setTermId] = useState("");
  const [cbcTermId, setCbcTermId] = useState("");
  const [startDate, setStartDate] = useState(daysAgoISO(30));
  const [endDate, setEndDate] = useState(todayISO());
  const [activeTab, setActiveTab] = useState<"chart" | "table">("chart");
  const [attTab, setAttTab] = useState<"chart" | "table">("chart");

  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: terms } = useQuery({ queryKey: ["terms"], queryFn: () => academicsApi.terms() });

  const { data: feeReport } = useQuery({ queryKey: ["report-fees"], queryFn: schoolApi.reportFeeCollection });
  const { data: attendanceReport, isLoading: attLoading } = useQuery({
    queryKey: ["report-attendance", startDate, endDate],
    queryFn: () => schoolApi.reportAttendance({ start_date: startDate, end_date: endDate }),
    enabled: Boolean(startDate && endDate),
  });
  const { data: perfReport, isLoading: perfLoading } = useQuery({
    queryKey: ["report-perf", termId, classId],
    enabled: Boolean(termId && classId),
    queryFn: () => schoolApi.reportClassPerformance(termId, classId),
  });
  const { data: cbcDist = [] } = useQuery({
    queryKey: ["cbc-dist", cbcTermId],
    queryFn: () => cbcApi.distribution(cbcTermId || undefined),
  });

  const fees = (feeReport ?? []) as any[];
  const totalBilled = fees.reduce((s, f) => s + Number(f.expected_amount ?? 0), 0);
  const totalCollected = fees.reduce((s, f) => s + Number(f.collected_amount ?? 0), 0);
  const totalOutstanding = fees.reduce((s, f) => s + Number(f.outstanding ?? 0), 0);
  const collectionRate = totalBilled > 0 ? Math.round((totalCollected / totalBilled) * 100) : 0;

  const feeChartData = fees.map((f) => ({
    name: (f.class_name || "Unassigned").replace("Grade ", "Gr "),
    Billed: Math.round(Number(f.expected_amount ?? 0)),
    Collected: Math.round(Number(f.collected_amount ?? 0)),
    Outstanding: Math.round(Number(f.outstanding ?? 0)),
  }));

  const attendance = (attendanceReport ?? []) as any[];
  const totalExpected = attendance.reduce((s, a) => s + Number(a.expected_days ?? 0), 0);
  const totalPresent = attendance.reduce((s, a) => s + Number(a.present_days ?? 0), 0);
  const presentRate = totalExpected > 0 ? Math.round((totalPresent / totalExpected) * 100) : 0;

  const attChartData = attendance.map((a) => ({
    name: (a.student_name as string).split(" ")[0],
    fullName: a.student_name,
    Rate: Math.round(Number(a.attendance_pct ?? 0)),
    Present: Number(a.present_days ?? 0),
    Absent: Number(a.absent_days ?? 0),
  }));

  const perfData = ((perfReport as any[]) ?? []).map((s) => ({
    name: (s.subject_name as string).replace("Mathematics", "Maths").replace("English Language", "English"),
    Mean: parseFloat(Number(s.mean_score ?? 0).toFixed(1)),
    PassPct: parseFloat(Number(s.pass_pct ?? 0).toFixed(0)),
  }));

  const cbcDistData = (cbcDist as { level: string; count: number; percent: number }[]);
  const cbcTotal = cbcDistData.reduce((s, d) => s + d.count, 0);

  function TabToggle({ value, onChange }: { value: "chart" | "table"; onChange: (v: "chart" | "table") => void }) {
    return (
      <div className="flex border rounded overflow-hidden text-xs font-medium mb-3 w-fit">
        {(["chart", "table"] as const).map(v => (
          <button key={v} onClick={() => onChange(v)}
            className={`px-3 py-1 ${value === v ? "bg-[#1b365d] text-white" : "bg-white text-gray-500 hover:bg-gray-50"}`}>
            {v === "chart" ? "Chart" : "Table"}
          </button>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-8 pb-8">
      <PageHeader title="Reports & Analytics" subtitle="Fee collection, attendance rates, academic performance, and CBC competency." />

      {/* ── Fee collection ─────────────────────────────────────────────── */}
      <section>
        <SectionTitle>Fee Collection — by Class</SectionTitle>
        <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Stat label="Total billed" value={kes(totalBilled)} />
          <Stat label="Total collected" value={kes(totalCollected)} />
          <Stat label="Outstanding" value={kes(totalOutstanding)} />
          <Stat label="Collection rate" value={`${collectionRate}%`} />
        </div>

        {fees.length === 0 ? (
          <p className="text-sm text-slate-400">No invoices generated yet.</p>
        ) : (
          <>
            <TabToggle value={activeTab} onChange={setActiveTab} />
            {activeTab === "chart" ? (
              <Card className="p-4">
                <ResponsiveContainer width="100%" height={260}>
                  <BarChart data={feeChartData} margin={{ top: 4, right: 12, left: 0, bottom: 4 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                    <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                    <YAxis tickFormatter={v => `${(v / 1000).toFixed(0)}k`} tick={{ fontSize: 11 }} />
                    <Tooltip formatter={(v: number) => kes(v)} />
                    <Legend iconSize={10} wrapperStyle={{ fontSize: 12 }} />
                    <Bar dataKey="Billed" fill="#93c5fd" radius={[3, 3, 0, 0]} />
                    <Bar dataKey="Collected" fill="#1b365d" radius={[3, 3, 0, 0]} />
                    <Bar dataKey="Outstanding" fill="#fca5a5" radius={[3, 3, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </Card>
            ) : (
              <Table headers={["Class", "Students", "Billed", "Collected", "Outstanding", "Collection %"]}>
                {fees.map((f) => (
                  <tr key={f.class_id ?? f.class_name}>
                    <td className="px-4 py-2 font-medium text-[#1b365d]">{f.class_name || "Unassigned"}</td>
                    <td className="px-4 py-2">{f.total_students}</td>
                    <td className="px-4 py-2">{kes(f.expected_amount)}</td>
                    <td className="px-4 py-2 text-green-600">{kes(f.collected_amount)}</td>
                    <td className={`px-4 py-2 ${Number(f.outstanding) > 0 ? "text-red-600" : "text-slate-500"}`}>{kes(f.outstanding)}</td>
                    <td className="px-4 py-2">{Number(f.collection_pct ?? 0).toFixed(0)}%</td>
                  </tr>
                ))}
              </Table>
            )}
          </>
        )}
      </section>

      {/* ── Attendance ─────────────────────────────────────────────────── */}
      <section>
        <SectionTitle>Attendance</SectionTitle>
        <div className="mb-4 flex flex-wrap gap-3 items-end">
          <Field label="From"><Input type="date" value={startDate} onChange={e => setStartDate(e.target.value)} /></Field>
          <Field label="To"><Input type="date" value={endDate} onChange={e => setEndDate(e.target.value)} /></Field>
          <div className="flex gap-3">
            <Stat label="Weekday-days (per student)" value={attendance[0]?.expected_days ?? 0} />
            <Stat label="Overall present rate" value={`${presentRate}%`} />
          </div>
        </div>

        {attLoading ? (
          <p className="text-sm text-slate-400">Loading…</p>
        ) : attendance.length === 0 ? (
          <p className="text-sm text-slate-400">No active students found.</p>
        ) : (
          <>
            <TabToggle value={attTab} onChange={setAttTab} />
            {attTab === "chart" ? (
              <Card className="p-4">
                <ResponsiveContainer width="100%" height={240}>
                  <BarChart data={attChartData} layout="vertical" margin={{ top: 4, right: 40, left: 30, bottom: 4 }}>
                    <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="#f0f0f0" />
                    <XAxis type="number" domain={[0, 100]} tickFormatter={v => `${v}%`} tick={{ fontSize: 11 }} />
                    <YAxis type="category" dataKey="name" tick={{ fontSize: 12 }} width={55} />
                    <Tooltip formatter={(v: number) => `${v}%`} labelFormatter={(_, p) => p[0]?.payload?.fullName ?? ""} />
                    <Bar dataKey="Rate" name="Attendance %" fill="#1b365d" radius={[0, 3, 3, 0]}>
                      {attChartData.map((entry, i) => (
                        <Cell key={i} fill={entry.Rate >= 80 ? "#16a34a" : entry.Rate >= 60 ? "#f59e0b" : "#dc2626"} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </Card>
            ) : (
              <Table headers={["Student", "Class", "Present", "Absent", "Expected", "Rate"]}>
                {attendance.map((a) => (
                  <tr key={a.student_id}>
                    <td className="px-4 py-2 font-medium text-[#1b365d]">{a.student_name}</td>
                    <td className="px-4 py-2 text-sm text-slate-600">{a.class_name || "—"}</td>
                    <td className="px-4 py-2 text-green-600">{a.present_days}</td>
                    <td className="px-4 py-2 text-red-600">{a.absent_days}</td>
                    <td className="px-4 py-2">{a.expected_days}</td>
                    <td className="px-4 py-2">{Number(a.attendance_pct ?? 0).toFixed(0)}%</td>
                  </tr>
                ))}
              </Table>
            )}
          </>
        )}
        <p className="mt-2 text-xs text-slate-400">
          Based on fingerprint-scanned daily attendance — a student with no scans on record shows as neither present nor absent.
        </p>
      </section>

      {/* ── Class performance ──────────────────────────────────────────── */}
      <section>
        <SectionTitle>Class Performance — by Subject</SectionTitle>
        <div className="mb-4 grid gap-3 sm:grid-cols-2 md:w-96">
          <Field label="Term">
            <Select value={termId} onChange={e => setTermId(e.target.value)}>
              <option value="">Select term…</option>
              {((terms as any[]) ?? []).map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
            </Select>
          </Field>
          <Field label="Class">
            <Select value={classId} onChange={e => setClassId(e.target.value)}>
              <option value="">Select class…</option>
              {((classes as any[]) ?? []).map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </Field>
        </div>

        {termId && classId && perfLoading && <p className="text-sm text-slate-400">Loading…</p>}

        {termId && classId && perfData.length > 0 && (
          <Card className="p-4">
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={perfData} margin={{ top: 4, right: 12, left: 0, bottom: 16 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="name" tick={{ fontSize: 11 }} angle={-15} textAnchor="end" />
                <YAxis domain={[0, 100]} tick={{ fontSize: 11 }} />
                <Tooltip />
                <Legend iconSize={10} wrapperStyle={{ fontSize: 12 }} />
                <Bar dataKey="Mean" name="Mean Score" fill="#1b365d" radius={[3, 3, 0, 0]} />
                <Bar dataKey="PassPct" name="Pass %" fill="#86efac" radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        )}
        {termId && classId && !perfLoading && perfData.length === 0 && (
          <p className="text-sm text-slate-400">No marks recorded for this class/term yet.</p>
        )}
        {(!termId || !classId) && (
          <p className="text-sm text-slate-400">Select a term and class to see subject-level performance.</p>
        )}
      </section>

      {/* ── CBC Competency Distribution ────────────────────────────────── */}
      <section>
        <SectionTitle>CBC Competency Distribution</SectionTitle>
        <div className="mb-4 w-56">
          <Field label="Filter by term (optional)">
            <Select value={cbcTermId} onChange={e => setCbcTermId(e.target.value)}>
              <option value="">All terms</option>
              {((terms as any[]) ?? []).map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
            </Select>
          </Field>
        </div>

        {cbcTotal === 0 ? (
          <p className="text-sm text-slate-400">No CBC marks recorded yet.</p>
        ) : (
          <div className="flex flex-col md:flex-row gap-6 items-start">
            <Card className="p-4 flex-shrink-0" style={{ width: 280 }}>
              <PieChart width={248} height={248}>
                <Pie
                  data={cbcDistData}
                  cx="50%" cy="50%"
                  innerRadius={60} outerRadius={100}
                  dataKey="count"
                  nameKey="level"
                  paddingAngle={2}
                >
                  {cbcDistData.map((entry) => (
                    <Cell key={entry.level} fill={LEVEL_COLORS[entry.level] ?? "#cbd5e1"} />
                  ))}
                </Pie>
                <Tooltip formatter={(v: number, name: string) => [`${v} marks`, name]} />
              </PieChart>
            </Card>
            <div className="flex-1 space-y-2">
              {cbcDistData.map((entry) => (
                <div key={entry.level} className="flex items-center gap-3">
                  <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ background: LEVEL_COLORS[entry.level] }} />
                  <div className="flex-1">
                    <div className="flex justify-between text-sm font-medium mb-0.5">
                      <span><strong>{entry.level}</strong> — {LEVEL_LABELS[entry.level]}</span>
                      <span className="text-slate-500">{entry.count} ({entry.percent.toFixed(0)}%)</span>
                    </div>
                    <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                      <div
                        className="h-full rounded-full"
                        style={{ width: `${entry.percent}%`, background: LEVEL_COLORS[entry.level] }}
                      />
                    </div>
                  </div>
                </div>
              ))}
              <p className="text-xs text-slate-400 pt-1">Total marks: {cbcTotal}</p>
            </div>
          </div>
        )}
      </section>
    </div>
  );
}
