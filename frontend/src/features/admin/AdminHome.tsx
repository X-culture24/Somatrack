import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  UserPlus, CreditCard, LayoutGrid, MessageSquare, Wallet,
  Receipt, CalendarCheck, BookOpen, Users, Truck,
  Library, Package, BarChart2, Eye, RefreshCw,
  TrendingUp, TrendingDown, Activity, Bell,
} from "lucide-react";
import { schoolApi } from "../../api/school";
import { kes } from "../../components/ui";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../../components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";
import { BorderBeam } from "../../components/ui/border-beam";
import { Button } from "../../components/ui/button";
import { FeeChartsSection } from "./FeeChartsSection";

const QUICK_ACTIONS = [
  { to: "/admin/admissions",   label: "New Student",   icon: UserPlus,       bg: "#e3f2fd", color: "#1565c0" },
  { to: "/finance/stk-push",   label: "New Payment",   icon: CreditCard,     bg: "#dbeafe", color: "#1b365d" },
  { to: "/admin/classes",      label: "Classes",       icon: LayoutGrid,     bg: "#eff6fb", color: "#2c5282" },
  { to: "/admin/announcements",label: "Send SMS",      icon: MessageSquare,  bg: "#e0f2fe", color: "#0369a1" },
  { to: "/admin/payroll",      label: "Payroll",       icon: Wallet,         bg: "#e3f2fd", color: "#2c5282" },
  { to: "/admin/reports",      label: "Reports",       icon: BarChart2,      bg: "#eef5fb", color: "#1b365d" },
  { to: "/teacher/attendance", label: "Roll Call",     icon: CalendarCheck,  bg: "#dbeafe", color: "#2c5282" },
  { to: "/teacher/marks",      label: "Exams",         icon: BookOpen,       bg: "#e0f2fe", color: "#3d5a80" },
  { to: "/finance/unreconciled",label: "Fee Arrears",  icon: Receipt,        bg: "#eff6fb", color: "#455a64" },
  { to: "/admin/transport",    label: "Transport",     icon: Truck,          bg: "#eff6fb", color: "#3d5a80" },
  { to: "/admin/library",      label: "Library",       icon: Library,        bg: "#dbeafe", color: "#2c5282" },
  { to: "/admin/inventory",    label: "Inventory",     icon: Package,        bg: "#e3f2fd", color: "#1b365d" },
];

function StatCard({
  label,
  value,
  icon: Icon,
  color,
  trend,
  featured = false,
}: {
  label: string;
  value: string | number;
  icon: any;
  color: string;
  trend?: { value: number; up: boolean };
  featured?: boolean;
}) {
  return (
    <Card className="relative overflow-hidden hover:shadow-lg transition-shadow duration-300">
      {featured && (
        <BorderBeam
          size={250}
          duration={12}
          colorFrom="#4a9eca"
          colorTo="#1b365d"
        />
      )}
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide">{label}</p>
            <p className="mt-1.5 text-2xl font-bold" style={{ color }}>{value}</p>
            {trend && (
              <div className="mt-1.5 flex items-center gap-1 text-xs">
                {trend.up ? (
                  <TrendingUp size={12} className="text-green-600" />
                ) : (
                  <TrendingDown size={12} className="text-red-600" />
                )}
                <span className={trend.up ? "text-green-600 font-medium" : "text-red-600 font-medium"}>
                  {Math.abs(trend.value)}%
                </span>
                <span className="text-muted-foreground">vs last term</span>
              </div>
            )}
          </div>
          <div
            className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl"
            style={{ backgroundColor: `${color}15` }}
          >
            <Icon size={22} style={{ color }} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function DonutChart({ collected, outstanding }: { collected: number; outstanding: number }) {
  const total = collected + outstanding;
  if (total <= 0) return <p className="text-sm text-slate-400 text-center py-8">No data yet.</p>;
  const pct = Math.round((collected / total) * 100);
  const collectedDeg = Math.round((collected / total) * 360);

  return (
    <div className="flex flex-col items-center">
      <div
        className="relative flex items-center justify-center rounded-full shadow-inner"
        style={{
          width: 160, height: 160,
          background: `conic-gradient(#2c5282 0deg ${collectedDeg}deg, #6b8aab ${collectedDeg}deg 360deg)`,
        }}
      >
        <div className="absolute inset-0 m-auto rounded-full bg-white shadow-inner" style={{ width: 100, height: 100 }} />
        <div className="relative z-10 text-center">
          <div className="text-2xl font-bold text-[#1b365d]">{pct}%</div>
          <div className="text-xs text-muted-foreground font-medium">collected</div>
        </div>
      </div>
      <div className="mt-5 grid grid-cols-2 gap-4 text-xs w-full">
        <div className="flex items-center gap-2 rounded-lg bg-slate-50 p-2">
          <div className="h-3.5 w-3.5 rounded-sm bg-[#2c5282] shadow-sm" />
          <div>
            <div className="font-semibold text-slate-700">Collected</div>
            <div className="text-muted-foreground">{kes(collected)}</div>
          </div>
        </div>
        <div className="flex items-center gap-2 rounded-lg bg-slate-50 p-2">
          <div className="h-3.5 w-3.5 rounded-sm bg-[#6b8aab] shadow-sm" />
          <div>
            <div className="font-semibold text-slate-700">Outstanding</div>
            <div className="text-muted-foreground">{kes(outstanding)}</div>
          </div>
        </div>
      </div>
    </div>
  );
}

export function AdminHome() {
  const { data, dataUpdatedAt } = useQuery({
    queryKey: ["dashboard"],
    queryFn: schoolApi.dashboard,
    refetchInterval: 15000,
  });

  const recentStudents: any[] = data?.recent_students ?? [];
  const recentPayments: any[] = data?.recent_payments ?? [];
  const totalCollected: number = data?.total_collected ?? 0;
  const outstanding: number = data?.outstanding_fees_total ?? 0;
  const announcements: any[] = data?.announcements ?? [];

  return (
    <div className="space-y-6 pb-8">
      {data?.term && (
        <Card className="relative overflow-hidden border-0 bg-gradient-to-r from-[#1b365d] via-[#2c5282] to-[#4a9eca] text-white shadow-lg">
          <div className="pointer-events-none absolute -top-16 -right-16 h-48 w-48 rounded-full bg-white/10 blur-2xl" />
          <div className="pointer-events-none absolute -bottom-20 -left-10 h-52 w-52 rounded-full bg-[#4a9eca]/30 blur-3xl" />
          <CardContent className="relative p-5">
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-white/15 backdrop-blur border border-white/20">
                  <Activity className="animate-pulse" size={22} />
                </div>
                <div>
                  <div className="text-xs uppercase tracking-wider text-white/70 font-semibold">Current Term</div>
                  <div className="text-2xl font-bold mt-0.5">{data.term}</div>
                </div>
              </div>
              <div className="flex items-center gap-5">
                <div className="hidden sm:flex items-center gap-2 rounded-full bg-white/10 border border-white/20 px-3 py-1.5 text-xs backdrop-blur">
                  <span className="h-2 w-2 rounded-full bg-[#4a9eca] animate-pulse shadow-[0_0_10px_rgba(74,158,202,0.8)]" />
                  Live Dashboard
                </div>
                {dataUpdatedAt && (
                  <div className="text-right text-xs text-white/80">
                    <div>Updated</div>
                    <div className="font-semibold">{new Date(dataUpdatedAt).toLocaleTimeString()}</div>
                  </div>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Card className="relative overflow-hidden">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg font-semibold text-[#1b365d]">Quick Actions</CardTitle>
              <CardDescription className="mt-0.5 text-xs">Jump to frequently used modules</CardDescription>
            </div>
            <Button variant="outline" size="sm" asChild className="border-slate-200 text-xs">
              <Link to="/admin/modules">All modules →</Link>
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6">
            {QUICK_ACTIONS.map(({ to, label, icon: Icon, bg, color }) => (
              <Link
                key={to}
                to={to}
                className="group flex flex-col items-center gap-2 rounded-2xl border border-slate-200/70 bg-gradient-to-b from-white to-slate-50/50 p-3 shadow-sm hover:shadow-md hover:scale-[1.03] hover:border-[#4a9eca]/50 transition-all duration-200 text-center"
              >
                <div
                  className="flex h-12 w-12 items-center justify-center rounded-2xl shadow-sm group-hover:scale-110 group-hover:shadow-md transition-all duration-200"
                  style={{ background: bg }}
                >
                  <Icon size={22} style={{ color }} strokeWidth={2.25} />
                </div>
                <span className="text-[11px] font-semibold text-slate-700 leading-tight group-hover:text-[#1b365d]">{label}</span>
              </Link>
            ))}
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Total Students"
          value={data?.student_count ?? "—"}
          icon={Users}
          color="#1b365d"
          trend={{ value: 8, up: true }}
          featured={true}
        />
        <StatCard
          label="Teaching Staff"
          value={data?.teacher_count ?? "—"}
          icon={BookOpen}
          color="#2c5282"
          trend={{ value: 3, up: true }}
        />
        <StatCard
          label="Active Classes"
          value={data?.class_count ?? "—"}
          icon={LayoutGrid}
          color="#4a9eca"
        />
        <StatCard
          label="Outstanding Fees"
          value={kes(data?.outstanding_fees_total)}
          icon={Receipt}
          color="#c62828"
          trend={{ value: 12, up: false }}
          featured={true}
        />
      </div>

      <Tabs defaultValue="recent" className="space-y-5">
        <TabsList className="h-11 grid w-full max-w-lg grid-cols-3 rounded-xl bg-slate-100/80 p-1 shadow-inner">
          <TabsTrigger value="recent" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Recent Activity</TabsTrigger>
          <TabsTrigger value="finance" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Finance</TabsTrigger>
          <TabsTrigger value="reports" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Reports</TabsTrigger>
        </TabsList>

        <TabsContent value="recent" className="space-y-5">
          <div className="grid gap-4 lg:grid-cols-2">
            <Card className="overflow-hidden border-slate-200/70">
              <CardHeader className="bg-gradient-to-r from-[#37474f] to-[#455a64] text-white py-4">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <Users size={16} strokeWidth={2.25} /> Recently Registered Students
                  </CardTitle>
                  <Link to="/admin/students" className="text-xs text-white/70 hover:text-white flex items-center gap-1 font-medium transition-colors">
                    <RefreshCw size={12} /> View all
                  </Link>
                </div>
              </CardHeader>
              <CardContent className="p-0">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-slate-50/80 text-xs text-muted-foreground">
                        <th className="px-4 py-2.5 text-left font-semibold">Name</th>
                        <th className="px-4 py-2.5 text-left font-semibold">ADM No.</th>
                        <th className="px-4 py-2.5 text-left font-semibold">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {recentStudents.length === 0 ? (
                        <tr><td colSpan={3} className="px-4 py-10 text-center text-muted-foreground">No students yet.</td></tr>
                      ) : recentStudents.slice(0, 5).map((s: any, i: number) => (
                        <tr key={i} className="border-b border-slate-100 hover:bg-slate-50/60 transition-colors">
                          <td className="px-4 py-3">
                            <div className="flex items-center gap-2.5">
                              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#1b365d]/10 text-[11px] font-bold text-[#1b365d]">
                                {s.full_name?.split(" ").map((n: string) => n[0]).slice(0, 2).join("") ?? "??"}
                              </div>
                              <span className="font-semibold text-[#1b365d]">{s.full_name}</span>
                            </div>
                          </td>
                          <td className="px-4 py-3 text-muted-foreground font-mono text-xs">{s.admission_no}</td>
                          <td className="px-4 py-3">
                            <Link
                              to={`/admin/students/${s.id}`}
                              className="inline-flex items-center gap-1 rounded-lg bg-[#4a9eca]/10 px-2.5 py-1 text-xs font-semibold text-[#1b6a9c] hover:bg-[#4a9eca]/20 transition-colors"
                            >
                              <Eye size={12} /> View
                            </Link>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </CardContent>
            </Card>

            <Card className="overflow-hidden border-slate-200/70">
              <CardHeader className="bg-gradient-to-r from-[#37474f] to-[#455a64] text-white py-4">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <CreditCard size={16} strokeWidth={2.25} /> Recent Fee Payments
                  </CardTitle>
                  <Link to="/finance/payments" className="text-xs text-white/70 hover:text-white flex items-center gap-1 font-medium transition-colors">
                    <RefreshCw size={12} /> View all
                  </Link>
                </div>
              </CardHeader>
              <CardContent className="p-0">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-slate-50/80 text-xs text-muted-foreground">
                        <th className="px-4 py-2.5 text-left font-semibold">Student</th>
                        <th className="px-4 py-2.5 text-right font-semibold">Amount</th>
                        <th className="px-4 py-2.5 text-left font-semibold">Method</th>
                      </tr>
                    </thead>
                    <tbody>
                      {recentPayments.length === 0 ? (
                        <tr><td colSpan={3} className="px-4 py-10 text-center text-muted-foreground">No payments yet.</td></tr>
                      ) : recentPayments.slice(0, 5).map((p: any, i: number) => (
                        <tr key={i} className="border-b border-slate-100 hover:bg-slate-50/60 transition-colors">
                          <td className="px-4 py-3 font-semibold text-[#1b365d]">{p.student_name}</td>
                          <td className="px-4 py-3 text-right font-bold text-green-700">
                            {kes(p.amount)}
                          </td>
                          <td className="px-4 py-3">
                            <span className="rounded-full bg-blue-50 border border-blue-100 px-2.5 py-0.5 text-[11px] font-semibold text-blue-700 capitalize">
                              {String(p.payment_method).replace("_", " ")}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </CardContent>
            </Card>
          </div>

          {announcements.length > 0 && (
            <Card className="overflow-hidden border-[#4a9eca]/20 bg-gradient-to-br from-[#e3f2fd]/50 to-white">
              <CardHeader className="pb-3">
                <CardTitle className="text-base flex items-center gap-2 text-[#1b365d]">
                  <Bell size={18} className="text-[#2c5282]" /> Latest Announcements
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                  {announcements.slice(0, 6).map((a: any) => (
                    <div key={a.id} className="rounded-xl border border-[#4a9eca]/20 bg-white p-3 shadow-sm hover:shadow-md transition-shadow">
                      <p className="text-sm font-semibold text-[#1b365d] line-clamp-1">{a.title}</p>
                      <p className="mt-1 line-clamp-2 text-xs text-slate-500">{a.body}</p>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="finance" className="space-y-5">
          <div className="grid gap-4 lg:grid-cols-3">
            <Card className="lg:col-span-2">
              <CardHeader>
                <CardTitle className="text-base">Fee Collection Trend</CardTitle>
                <CardDescription className="text-xs">Per-term collection vs target</CardDescription>
              </CardHeader>
              <CardContent>
                <FeeChartsSection />
              </CardContent>
            </Card>

            <Card className="relative overflow-hidden">
              <div className="absolute top-0 right-0 h-24 w-24 rounded-bl-full bg-gradient-to-br from-[#4a9eca]/12 to-transparent opacity-80" />
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <BarChart2 size={18} className="text-[#1b365d]" />
                  Cashflow Summary
                </CardTitle>
                <CardDescription className="text-xs">{data?.term ?? "Current Term"}</CardDescription>
              </CardHeader>
              <CardContent className="relative">
                <DonutChart collected={totalCollected} outstanding={outstanding} />
                <div className="mt-6 space-y-2.5 border-t pt-4">
                  <div className="flex justify-between items-center rounded-lg bg-[#e3f2fd] px-3 py-2">
                    <span className="text-sm text-slate-600">Total Collected</span>
                    <span className="font-bold text-[#1b365d]">{kes(totalCollected)}</span>
                  </div>
                  <div className="flex justify-between items-center rounded-lg bg-[#eef5fb] px-3 py-2">
                    <span className="text-sm text-slate-600">Outstanding</span>
                    <span className="font-bold text-[#2c5282]">{kes(outstanding)}</span>
                  </div>
                  {data?.unreconciled_payment_count > 0 && (
                    <div className="flex justify-between items-center rounded-lg bg-[#dbeafe] px-3 py-2 border-t-0">
                      <span className="text-sm text-slate-600">Unreconciled</span>
                      <span className="font-bold text-[#3d5a80]">{data.unreconciled_payment_count}</span>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="reports">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Reports &amp; Analytics</CardTitle>
              <CardDescription className="text-xs">Quick access to commonly used reports</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {[
                  { label: "Attendance Report", to: "/admin/reports", icon: CalendarCheck, color: "#2c5282" },
                  { label: "Fee Collection", to: "/admin/reports", icon: Receipt, color: "#1565c0" },
                  { label: "Academic Performance", to: "/admin/reports", icon: BarChart2, color: "#3d5a80" },
                  { label: "Student Demographics", to: "/admin/reports", icon: Users, color: "#4a9eca" },
                  { label: "Staff Summary", to: "/admin/reports", icon: BookOpen, color: "#2c5282" },
                  { label: "Transport Utilization", to: "/admin/reports", icon: Truck, color: "#455a64" },
                ].map(({ label, to, icon: Icon, color }) => (
                  <Link
                    key={to + label}
                    to={to}
                    className="group flex items-center gap-3 rounded-2xl border border-slate-200/70 bg-white p-4 hover:bg-slate-50 hover:shadow-lg hover:border-[#4a9eca]/40 transition-all duration-200"
                  >
                    <div
                      className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl group-hover:scale-110 transition-transform duration-200 shadow-sm"
                      style={{ backgroundColor: `${color}15` }}
                    >
                      <Icon size={20} style={{ color }} strokeWidth={2} />
                    </div>
                    <div className="min-w-0">
                      <span className="text-sm font-semibold text-slate-800 group-hover:text-[#1b365d] transition-colors">{label}</span>
                      <p className="text-[11px] text-muted-foreground mt-0.5">Open report →</p>
                    </div>
                  </Link>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
