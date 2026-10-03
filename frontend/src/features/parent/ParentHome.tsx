import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router-dom";
import {
  CreditCard, TrendingUp, CalendarCheck, Bell, Smartphone,
  Users, GraduationCap, BookOpen, Award, ChevronRight, Send,
} from "lucide-react";
import { feePaymentApi } from "../../api/finance";
import { schoolApi } from "../../api/school";
import { kes } from "../../components/ui";
import { Card, CardContent } from "../../components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";
import { BorderBeam } from "../../components/ui/border-beam";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";

function AttendancePill({ present, absent, late, days }: { present: number; absent: number; late: number; days: number }) {
  if (days === 0) return <span className="text-xs text-slate-400">No records yet</span>;
  const pct = Math.round(((present + late) / days) * 100);
  const color = pct >= 90 ? "bg-green-100 text-green-700 border-green-200" : pct >= 75 ? "bg-amber-100 text-amber-700 border-amber-200" : "bg-red-100 text-red-700 border-red-200";
  return (
    <div className="flex flex-wrap gap-1.5 mt-2">
      <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold border ${color}`}>{pct}% attendance</span>
      <span className="rounded-full bg-green-50 text-green-700 border border-green-100 px-2.5 py-0.5 text-[11px] font-semibold">{present} present</span>
      {late > 0 && <span className="rounded-full bg-amber-50 text-amber-700 border border-amber-100 px-2.5 py-0.5 text-[11px] font-semibold">{late} late</span>}
      {absent > 0 && <span className="rounded-full bg-red-50 text-red-700 border border-red-100 px-2.5 py-0.5 text-[11px] font-semibold">{absent} absent</span>}
    </div>
  );
}

function StatCard({
  label, value, icon: Icon, color, featured,
}: { label: string; value: string | number; icon: any; color: string; featured?: boolean }) {
  return (
    <Card className="relative overflow-hidden hover:shadow-lg transition-shadow duration-300">
      {featured && <BorderBeam size={200} duration={14} colorFrom="#1b365d" colorTo="#4a9eca" />}
      <CardContent className="p-4">
        <div className="flex items-center gap-3">
          <div
            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl shadow-sm"
            style={{ backgroundColor: `${color}15` }}
          >
            <Icon size={18} style={{ color }} strokeWidth={2.25} />
          </div>
          <div className="min-w-0">
            <p className="text-[11px] uppercase tracking-wide text-muted-foreground font-semibold">{label}</p>
            <p className="mt-0.5 text-xl font-bold" style={{ color }}>{value}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function ParentHome() {
  const { data, isLoading } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const kids: any[] = data?.children ?? [];
  const announcements: any[] = data?.announcements ?? [];

  return (
    <div className="space-y-6 pb-8">
      <Card className="relative overflow-hidden border-0 bg-gradient-to-r from-[#1b365d] via-[#2c5282] to-[#4a9eca] text-white shadow-lg">
        <div className="pointer-events-none absolute -top-20 -right-16 h-56 w-56 rounded-full bg-white/10 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-24 -left-10 h-56 w-56 rounded-full bg-[#4a9eca]/30 blur-3xl" />
        <CardContent className="relative p-5">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-white/15 backdrop-blur border border-white/20">
                <Users size={22} />
              </div>
              <div>
                <div className="text-xs uppercase tracking-wider text-white/70 font-semibold">Family Dashboard</div>
                <div className="text-xl font-bold mt-0.5">Welcome back 👋</div>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <div className="rounded-full bg-white/10 border border-white/20 px-3.5 py-1.5 text-xs backdrop-blur font-medium flex items-center gap-1.5">
                <Users size={12} /> {kids.length} {kids.length === 1 ? "ward" : "wards"}
              </div>
              {data?.term && (
                <div className="rounded-full bg-white/10 border border-white/20 px-3.5 py-1.5 text-xs backdrop-blur font-medium">
                  {data.term}
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {isLoading && (
        <div className="space-y-3">
          {[1, 2].map((i) => (
            <div key={i} className="h-40 animate-pulse rounded-2xl bg-slate-200" />
          ))}
        </div>
      )}

      {!isLoading && kids.length === 0 && (
        <Card className="border-dashed border-slate-300 bg-white/60">
          <CardContent className="p-8 text-center">
            <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-100">
              <Users size={24} className="text-slate-400" />
            </div>
            <p className="text-sm font-medium text-slate-600">No learners linked to your account yet.</p>
            <p className="mt-1 text-xs text-slate-400">Contact the school office to link your child.</p>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Active Wards"
          value={kids.length}
          icon={GraduationCap}
          color="#1b365d"
          featured={true}
        />
        <StatCard
          label="Total Balance"
          value={kes(kids.reduce((s: number, k: any) => s + Number(k.fee_balance || 0), 0))}
          icon={CreditCard}
          color="#c62828"
          featured={true}
        />
        <StatCard
          label="Announcements"
          value={announcements.length}
          icon={Bell}
          color="#4a9eca"
        />
        <StatCard
          label="Avg. Attendance"
          value={kids.length > 0 ? `${Math.round(kids.map((k: any) => {
            const a = k.attendance;
            return a && a.days > 0 ? ((a.present + a.late) / a.days) * 100 : 0;
          }).reduce((a: number, b: number) => a + b, 0) / kids.length)}%` : "—"}
          icon={CalendarCheck}
          color="#2c5282"
        />
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Link
          to="/parent/fees"
          className="group col-span-1 sm:col-span-2 flex flex-col items-center justify-center gap-2 rounded-2xl border border-slate-200/70 bg-gradient-to-br from-[#1b365d] to-[#2c5282] p-5 shadow-sm hover:shadow-lg hover:scale-[1.02] transition-all duration-200 text-center"
        >
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-white/15">
            <CreditCard size={22} className="text-white" strokeWidth={2.25} />
          </div>
          <div>
            <span className="text-sm font-bold text-white">Fee statements</span>
            <p className="text-[11px] text-white/70 mt-0.5">View &amp; pay fees</p>
          </div>
        </Link>
        <Link
          to="/parent/progress"
          className="group col-span-1 sm:col-span-2 flex flex-col items-center justify-center gap-2 rounded-2xl border border-slate-200/70 bg-gradient-to-br from-[#4a9eca] to-[#3d8bc0] p-5 shadow-sm hover:shadow-lg hover:scale-[1.02] transition-all duration-200 text-center"
        >
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-white/15">
            <TrendingUp size={22} className="text-white" strokeWidth={2.25} />
          </div>
          <div>
            <span className="text-sm font-bold text-white">Progress &amp; results</span>
            <p className="text-[11px] text-white/70 mt-0.5">Academics &amp; attendance</p>
          </div>
        </Link>
      </div>

      <Tabs defaultValue="children" className="space-y-5">
        <TabsList className="h-11 grid w-full max-w-sm grid-cols-2 rounded-xl bg-slate-100/80 p-1 shadow-inner">
          <TabsTrigger value="children" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">My Children</TabsTrigger>
          <TabsTrigger value="news" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">School News</TabsTrigger>
        </TabsList>

        <TabsContent value="children" className="space-y-4">
          {kids.map((c: any) => (
            <ChildCard key={c.student_id} child={c} />
          ))}
        </TabsContent>

        <TabsContent value="news" className="space-y-4">
          {announcements.length === 0 ? (
            <Card>
              <CardContent className="py-10 text-center text-muted-foreground text-sm">
                No announcements yet.
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-3">
              {announcements.map((a: any) => (
                <Card key={a.id} className="overflow-hidden hover:shadow-md transition-shadow">
                  <CardContent className="p-4">
                    <div className="flex items-start gap-3">
                      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-[#1b365d]/10">
                        <Bell size={16} className="text-[#1b365d]" />
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-bold text-[#1b365d]">{a.title}</p>
                        <p className="mt-1 text-xs text-slate-500 leading-relaxed">{a.body}</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

function ChildCard({ child: c }: { child: any }) {
  const balance = Number(c.fee_balance);
  const [amount, setAmount] = useState(balance > 0 ? String(balance) : "");
  const [result, setResult] = useState<{ ok: boolean; msg: string } | null>(null);

  const mpesa = useMutation({
    mutationFn: () =>
      feePaymentApi.initiateMpesa({
        student_id: c.student_id,
        phone: "0725926141",
        amount: Number(amount),
      }),
    onSuccess: () => setResult({ ok: true, msg: "STK push sent to 0725926141 — check your phone and enter your M-Pesa PIN." }),
    onError: (err: any) => setResult({ ok: false, msg: err?.response?.data?.message ?? "Payment request failed. Please try again." }),
  });

  const initials = c.full_name?.split(" ").map((n: string) => n[0]).slice(0, 2).join("").toUpperCase() ?? "👶";

  return (
    <Card className="relative overflow-hidden hover:shadow-lg transition-shadow duration-300 border-slate-200/70">
      <div className="pointer-events-none absolute top-0 right-0 h-32 w-32 rounded-bl-full opacity-50"
           style={{ background: balance === 0 ? "linear-gradient(135deg, #86efac33, transparent)" : "linear-gradient(135deg, #fca5a533, transparent)" }} />
      <CardContent className="relative p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-3 min-w-0">
            <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-[#1b365d] to-[#4a9eca] text-white font-bold shadow-md">
              {initials}
            </div>
            <div className="min-w-0">
              <p className="text-[11px] text-muted-foreground font-mono font-medium">{c.admission_no}</p>
              <h2 className="mt-0.5 text-lg font-bold text-[#1b365d] leading-tight truncate">{c.full_name}</h2>
              <p className="text-sm text-slate-500 flex items-center gap-1.5 mt-0.5">
                <BookOpen size={13} className="text-slate-400" /> {c.class_name}
              </p>
            </div>
          </div>
          <span className={`mt-1 rounded-full px-3 py-1 text-[11px] font-bold shrink-0 border ${balance === 0 ? "bg-green-100 text-green-700 border-green-200" : "bg-red-100 text-red-700 border-red-200"}`}>
            {balance === 0 ? "✓ Fees Paid" : "Balance Due"}
          </span>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          <div className="rounded-xl border border-slate-100 bg-gradient-to-br from-slate-50/80 to-white p-3">
            <p className="text-[11px] uppercase tracking-wide text-slate-500 font-semibold">Term Balance</p>
            <p className="text-xl font-bold mt-1 text-[#1b365d]">{kes(c.fee_balance)}</p>
            <p className="mt-1 text-[10px] text-slate-400 leading-tight">
              Paybill 247247 · Acc 137101#{String(c.full_name).replace(/\s+/g, "").toUpperCase()}
            </p>
          </div>

          <div className="sm:col-span-2 rounded-xl border border-slate-200/70 bg-white p-3 shadow-sm">
            <p className="mb-2 text-[11px] font-bold uppercase tracking-wide text-slate-600 flex items-center gap-1">
              <Smartphone size={13} className="text-[#4CAF50]" /> Pay via M-Pesa
            </p>
            <div className="flex flex-wrap items-center gap-2">
              <div className="relative flex-1 min-w-[160px]">
                <Label htmlFor={`amt-${c.student_id}`} className="sr-only">Amount</Label>
                <div className="absolute left-3 top-1/2 -translate-y-1/2 text-xs font-semibold text-slate-400">KES</div>
                <Input
                  id={`amt-${c.student_id}`}
                  type="number"
                  min="1"
                  placeholder="Amount"
                  value={amount}
                  onChange={(e) => { setAmount(e.target.value); setResult(null); }}
                  className="h-9 pl-10 text-sm"
                />
              </div>
              <Button
                onClick={() => { setResult(null); mpesa.mutate(); }}
                disabled={mpesa.isPending || !amount || Number(amount) < 1}
                size="sm"
                className="h-9 bg-[#4CAF50] hover:bg-[#43a047] text-white shrink-0"
              >
                <Send size={14} className="mr-1" />
                {mpesa.isPending ? "Sending…" : "Send STK"}
              </Button>
            </div>
            <p className="mt-1.5 text-[10px] text-slate-400">Prompt will be sent to 0725926141</p>
            {result && (
              <p className={`mt-2 rounded-lg px-3 py-1.5 text-[11px] font-medium ${result.ok ? "bg-green-50 text-green-700 border border-green-100" : "bg-red-50 text-red-700 border border-red-100"}`}>
                {result.msg}
              </p>
            )}
          </div>
        </div>

        <div className="mt-4 border-t border-slate-100 pt-4">
          <div className="flex items-center justify-between mb-1">
            <p className="text-xs font-bold uppercase tracking-wide text-slate-500 flex items-center gap-1">
              <CalendarCheck size={12} /> Attendance — last 30 days
            </p>
            <Link to="/parent/progress" className="text-[11px] text-[#4a9eca] hover:text-[#1b365d] font-semibold flex items-center gap-0.5">
              View progress <ChevronRight size={12} />
            </Link>
          </div>
          <AttendancePill
            present={c.attendance?.present ?? 0}
            absent={c.attendance?.absent ?? 0}
            late={c.attendance?.late ?? 0}
            days={c.attendance?.days ?? 0}
          />
        </div>

        {c.latest_mark && (
          <div className="mt-3 rounded-xl bg-gradient-to-r from-[#e3f2fd]/70 to-white p-3 border border-[#4a9eca]/15">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-2 min-w-0">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-[#4a9eca]/15">
                  <Award size={14} className="text-[#1b365d]" />
                </div>
                <div className="min-w-0">
                  <p className="text-[11px] font-bold text-[#1b365d]">Latest grade</p>
                  <p className="text-xs text-slate-600 truncate">{c.latest_mark.subject}</p>
                </div>
              </div>
              <div className="text-right shrink-0">
                <p className="text-lg font-bold text-[#1b365d] leading-none">
                  {c.latest_mark.score}{c.latest_mark.competency_level ? ` <span class="text-xs text-slate-500 font-semibold">(${c.latest_mark.competency_level})</span>` : ""}
                </p>
                <Link to="/parent/progress" className="text-[10px] text-[#4a9eca] font-semibold hover:underline">
                  See all →
                </Link>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
