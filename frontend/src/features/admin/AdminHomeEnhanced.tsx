import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  UserPlus, CreditCard, LayoutGrid, MessageSquare, Wallet,
  Receipt, CalendarCheck, BookOpen, Users, Truck,
  Library, Package, BarChart2, Eye, RefreshCw,
  TrendingUp, TrendingDown, Activity,
} from "lucide-react";
import { schoolApi } from "../../api/school";
import { kes } from "../../components/ui";
import { Card, CardHeader, CardTitle, CardContent } from "../../components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";
import { BorderBeam } from "../../components/ui/border-beam";
import { FeeChartsSection } from "./FeeChartsSection";

// ── Quick actions (matches mock-up grid) ──────────────────────────────────────
const QUICK_ACTIONS = [
  { to: "/admin/admissions",   label: "New Student",   icon: UserPlus,       bg: "#e8f5e9", color: "#2e7d32" },
  { to: "/finance/stk-push",   label: "New Payment",   icon: CreditCard,     bg: "#e3f2fd", color: "#1565c0" },
  { to: "/admin/classes",      label: "Classes",       icon: LayoutGrid,     bg: "#fff3e0", color: "#e65100" },
  { to: "/admin/announcements",label: "Send SMS",      icon: MessageSquare,  bg: "#f3e5f5", color: "#6a1b9a" },
  { to: "/admin/payroll",      label: "Payroll",       icon: Wallet,         bg: "#e8f5e9", color: "#2e7d32" },
  { to: "/admin/reports",      label: "Reports",       icon: BarChart2,      bg: "#fff8e1", color: "#f57f17" },
  { to: "/teacher/attendance", label: "Roll Call",     icon: CalendarCheck,  bg: "#e3f2fd", color: "#1565c0" },
  { to: "/teacher/marks",      label: "Exams",         icon: BookOpen,       bg: "#fce4ec", color: "#880e4f" },
  { to: "/finance/unreconciled",label: "Fee Arrears",  icon: Receipt,        bg: "#fff3e0", color: "#bf360c" },
  { to: "/admin/transport",    label: "Transport",     icon: Truck,          bg: "#f3e5f5", color: "#4a148c" },
  { to: "/admin/library",      label: "Library",       icon: Library,        bg: "#e0f7fa", color: "#006064" },
  { to: "/admin/inventory",    label: "Inventory",     icon: Package,        bg: "#e8eaf6", color: "#1a237e" },
];

// ── Enhanced stat card with BorderBeam animation ──────────────────────────────
function StatCard({ 
  label, 
  value, 
  icon: Icon, 
  color, 
  trend, 
  featured = false 
}: { 
  label: string; 
  value: string | number; 
  icon: any; 
  color: string; 
  trend?: { value: number; up: boolean }; 
  featured?: boolean;
}) {
  return (
    <Card className="relative overflow-hidden hover:shadow-lg transition-shadow">
      {featured && (
        <BorderBeam 
          size={250}
          duration={12}
          colorFrom="#4a9eca"
          colorTo="#1b365d"
        />
      )}
      <CardContent className="p-4">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide">{label}</p>
            <p className="mt-1 text-2xl font-bold" style={{ color }}>{value}</p>
            {trend && (
              <div className="mt-1 flex items-center gap-1 text-xs">
                {trend.up ? (
                  <TrendingUp size={12} className="text-green-600" />
                ) : (
                  <TrendingDown size={12} className="text-red-600" />
                )}
                <span className={trend.up ? "text-green-600" : "text-red-600"}>
                  {Math.abs(trend.value)}%
                </span>
                <span className="text-muted-foreground">vs last term</span>
              </div>
            )}
          </div>
          <div 
            className="flex h-12 w-12 items-center justify-center rounded-full" 
            style={{ backgroundColor: `${color}15` }}
          >
            <Icon size={24} style={{ color }} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// ── CSS donut chart ───────────────────────────────────────────────────────────
function DonutChart({ collected, outstanding }: { collected: number; outstanding: number }) {
  const total = collected + outstanding;
  if (total <= 0) return <p className="text-sm text-slate-400 text-center py-8">No data yet.</p>;
  const pct = Math.round((collected / total) * 100);
  const collectedDeg = Math.round((collected / total) * 360);
  
  return (
    <div className="flex flex-col items-center">
      <div
        className="relative flex items-center justify-center rounded-full"
        style={{
          width: 140, height: 140,
          background: `conic-gradient(#2e7d32 0deg ${collectedDeg}deg, #f97316 ${collectedDeg}deg 360deg)`,
        }}
      >
        <div className="absolute inset-0 m-auto rounded-full bg-white" style={{ width: 90, height: 90 }} />
        <div className="relative z-10 text-center">
          <div className="text-2xl font-bold text-[#1b365d]">{pct}%</div>
          <div className="text-xs text-muted-foreground">collected</div>
        </div>
      </div>
      <div className="mt-4 grid grid-cols-2 gap-3 text-xs">
        <div className="flex items-center gap-1.5">
          <div className="h-3 w-3 rounded-sm bg-green-600" />
          <div>
            <div className="font-medium text-slate-700">Collected</div>
            <div className="text-muted-foreground">{kes(collected)}</div>
          </div>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="h-3 w-3 rounded-sm bg-orange-500" />
          <div>
            <div className="font-medium text-slate-700">Outstanding</div>
            <div className="text-muted-foreground">{kes(outstanding)}</div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Main component ────────────────────────────────────────────────────────────
export function AdminHomeEnhanced() {
  const { data, dataUpdatedAt } = useQuery({
    queryKey: ["dashboard"],
    queryFn: schoolApi.dashboard,
    refetchInterval: 15000,
  });

  const recentStudents: any[] = data?.recent_students ?? [];
  const recentPayments: any[] = data?.recent_payments ?? [];
  const totalCollected: number = data?.total_collected ?? 0;
  const outstanding: number = data?.outstanding_fees_total ?? 0;

  return (
    <div className="space-y-6">
      {/* Term banner with live indicator */}
      {data?.term && (
        <Card className="bg-gradient-to-r from-[#1b365d] to-[#4a9eca] border-0">
          <CardContent className="p-4">
            <div className="flex items-center justify-between text-white">
              <div className="flex items-center gap-3">
                <Activity className="animate-pulse" size={20} />
                <div>
                  <div className="text-sm opacity-90">Current Term</div>
                  <div className="text-lg font-semibold">{data.term}</div>
                </div>
              </div>
              {dataUpdatedAt && (
                <div className="text-right text-xs opacity-75">
                  <div>Live Dashboard</div>
                  <div>Updated {new Date(dataUpdatedAt).toLocaleTimeString()}</div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Quick actions grid - Enhanced with hover effects */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Quick Actions</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-6">
            {QUICK_ACTIONS.map(({ to, label, icon: Icon, bg, color }) => (
              <Link
                key={to}
                to={to}
                className="group flex flex-col items-center gap-2 rounded-xl border border-slate-200 bg-white p-3 shadow-sm hover:shadow-md hover:scale-105 transition-all text-center"
              >
                <div 
                  className="flex h-12 w-12 items-center justify-center rounded-xl group-hover:scale-110 transition-transform" 
                  style={{ background: bg }}
                >
                  <Icon size={22} style={{ color }} />
                </div>
                <span className="text-xs font-medium text-slate-600 leading-tight">{label}</span>
              </Link>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Stats grid with featured cards */}
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
          color="#2e7d32"
          trend={{ value: 3, up: true }}
        />
        <StatCard
          label="Active Classes"
          value={data?.class_count ?? "—"}
          icon={LayoutGrid}
          color="#e65100"
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

      {/* Tabbed content area */}
      <Tabs defaultValue="recent" className="space-y-4">
        <TabsList className="grid w-full max-w-md grid-cols-3">
          <TabsTrigger value="recent">Recent Activity</TabsTrigger>
          <TabsTrigger value="finance">Finance</TabsTrigger>
          <TabsTrigger value="reports">Reports</TabsTrigger>
        </TabsList>

        <TabsContent value="recent" className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-2">
            {/* Recently Registered Students */}
            <Card>
              <CardHeader className="bg-[#37474f] text-white rounded-t-lg">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <Users size={16} /> Recently Registered Students
                  </CardTitle>
                  <Link to="/admin/students" className="text-xs hover:underline flex items-center gap-1">
                    <RefreshCw size={12} /> View all
                  </Link>
                </div>
              </CardHeader>
              <CardContent className="p-0">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-slate-50 text-xs text-muted-foreground">
                        <th className="px-4 py-2 text-left font-medium">Name</th>
                        <th className="px-4 py-2 text-left font-medium">ADM No.</th>
                        <th className="px-4 py-2 text-left font-medium">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {recentStudents.length === 0 ? (
                        <tr><td colSpan={3} className="px-4 py-8 text-center text-muted-foreground">No students yet.</td></tr>
                      ) : recentStudents.slice(0, 5).map((s: any, i: number) => (
                        <tr key={i} className="border-b hover:bg-slate-50/50 transition-colors">
                          <td className="px-4 py-3 font-medium text-[#1b365d]">{s.full_name}</td>
                          <td className="px-4 py-3 text-muted-foreground">{s.admission_no}</td>
                          <td className="px-4 py-3">
                            <Link 
                              to={`/admin/students/${s.id}`} 
                              className="inline-flex items-center gap-1 rounded bg-[#4a9eca] px-2 py-1 text-xs text-white hover:bg-[#1b365d] transition-colors"
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

            {/* Recent Fee Payments */}
            <Card>
              <CardHeader className="bg-[#37474f] text-white rounded-t-lg">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <CreditCard size={16} /> Recent Fee Payments
                  </CardTitle>
                  <Link to="/finance/payments" className="text-xs hover:underline flex items-center gap-1">
                    <RefreshCw size={12} /> View all
                  </Link>
                </div>
              </CardHeader>
              <CardContent className="p-0">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-slate-50 text-xs text-muted-foreground">
                        <th className="px-4 py-2 text-left font-medium">Student</th>
                        <th className="px-4 py-2 text-right font-medium">Amount</th>
                        <th className="px-4 py-2 text-left font-medium">Method</th>
                      </tr>
                    </thead>
                    <tbody>
                      {recentPayments.length === 0 ? (
                        <tr><td colSpan={3} className="px-4 py-8 text-center text-muted-foreground">No payments yet.</td></tr>
                      ) : recentPayments.slice(0, 5).map((p: any, i: number) => (
                        <tr key={i} className="border-b hover:bg-slate-50/50 transition-colors">
                          <td className="px-4 py-3 font-medium text-[#1b365d]">{p.student_name}</td>
                          <td className="px-4 py-3 text-right font-semibold text-green-700">
                            KES {Number(p.amount).toLocaleString("en-KE", { minimumFractionDigits: 2 })}
                          </td>
                          <td className="px-4 py-3">
                            <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 capitalize">
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
        </TabsContent>

        <TabsContent value="finance" className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-3">
            <Card className="lg:col-span-2">
              <CardHeader>
                <CardTitle>Fee Collection Trend</CardTitle>
              </CardHeader>
              <CardContent>
                <FeeChartsSection />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <BarChart2 size={18} />
                  Cashflow Summary
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="mb-4 text-xs font-medium text-center text-muted-foreground">
                  {data?.term ?? "Current Term"}
                </p>
                <DonutChart collected={totalCollected} outstanding={outstanding} />
                <div className="mt-6 space-y-3 border-t pt-4">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Total Collected</span>
                    <span className="font-semibold text-green-700">{kes(totalCollected)}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Outstanding</span>
                    <span className="font-semibold text-red-600">{kes(outstanding)}</span>
                  </div>
                  {data?.unreconciled_payment_count > 0 && (
                    <div className="flex justify-between text-sm border-t pt-2">
                      <span className="text-muted-foreground">Unreconciled</span>
                      <span className="font-semibold text-orange-600">{data.unreconciled_payment_count}</span>
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
              <CardTitle>Reports & Analytics</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {[
                  { label: "Attendance Report", to: "/admin/reports/attendance", icon: CalendarCheck, color: "#2e7d32" },
                  { label: "Fee Collection", to: "/admin/reports/fees", icon: Receipt, color: "#1565c0" },
                  { label: "Academic Performance", to: "/admin/reports/performance", icon: BarChart2, color: "#e65100" },
                  { label: "Student Demographics", to: "/admin/reports/demographics", icon: Users, color: "#6a1b9a" },
                  { label: "Staff Summary", to: "/admin/reports/staff", icon: BookOpen, color: "#00838f" },
                  { label: "Transport Utilization", to: "/admin/reports/transport", icon: Truck, color: "#4a148c" },
                ].map(({ label, to, icon: Icon, color }) => (
                  <Link
                    key={to}
                    to={to}
                    className="flex items-center gap-3 rounded-lg border p-4 hover:bg-slate-50 hover:shadow-md transition-all group"
                  >
                    <div 
                      className="flex h-10 w-10 items-center justify-center rounded-lg group-hover:scale-110 transition-transform" 
                      style={{ backgroundColor: `${color}15` }}
                    >
                      <Icon size={20} style={{ color }} />
                    </div>
                    <span className="text-sm font-medium text-slate-700">{label}</span>
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
