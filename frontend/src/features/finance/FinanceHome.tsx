import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router-dom";
import { billingApi, paymentsApi } from "../../api/finance";
import { academicsApi } from "../../api/academics";
import { schoolApi } from "../../api/school";
import { kes, useConfirm } from "../../components/ui";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "../../components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";
import { BorderBeam } from "../../components/ui/border-beam";
import { Button } from "../../components/ui/button";
import { Label } from "../../components/ui/label";
import {
  DollarSign, Users, AlertTriangle, FileText, TrendingUp,
  CreditCard, Receipt, Send, RefreshCw, Sparkles,
  Wallet, BarChart3, PiggyBank, ChevronRight,
} from "lucide-react";

function StatCard({
  label, value, icon: Icon, color, trend, featured,
}: {
  label: string;
  value: string | number;
  icon: any;
  color: string;
  trend?: { value: number; up: boolean; label: string };
  featured?: boolean;
}) {
  return (
    <Card className={`relative overflow-hidden hover:shadow-lg transition-all duration-300 hover:-translate-y-0.5`}>
      {featured && (
        <BorderBeam size={220} duration={14} colorFrom="#4a9eca" colorTo="#1b365d" delay={0.5} />
      )}
      <CardContent className="p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-[11px] uppercase tracking-wide text-muted-foreground font-semibold">{label}</p>
            <p className="mt-1.5 text-2xl font-bold truncate" style={{ color }}>{value}</p>
            {trend && (
              <div className="mt-1.5 flex items-center gap-1 text-xs flex-wrap">
                <span className={trend.up ? "text-green-600 font-semibold" : "text-red-600 font-semibold"}>
                  {trend.up ? "↑" : "↓"} {Math.abs(trend.value)}%
                </span>
                <span className="text-muted-foreground">{trend.label}</span>
              </div>
            )}
          </div>
          <div
            className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl shadow-sm"
            style={{ backgroundColor: `${color}15` }}
          >
            <Icon size={22} style={{ color }} strokeWidth={2.25} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function FinanceHome() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data: dash } = useQuery({ queryKey: ["dashboard"], queryFn: schoolApi.dashboard });
  const { data: recentPayments } = useQuery({ queryKey: ["payments", "recent"], queryFn: () => paymentsApi.list() });
  const { data: terms } = useQuery({ queryKey: ["terms"], queryFn: academicsApi.terms });
  const [termId, setTermId] = useState("");
  const [genResult, setGenResult] = useState<{ generated: number; skipped: unknown[] } | null>(null);
  const [reminderResult, setReminderResult] = useState<{ sent: number } | null>(null);

  const generate = useMutation({
    mutationFn: () => billingApi.generate(termId),
    onSuccess: (result) => {
      setGenResult({ generated: result.generated, skipped: result.skipped ?? [] });
      qc.invalidateQueries({ queryKey: ["invoices"] });
    },
  });
  const remind = useMutation({
    mutationFn: paymentsApi.sendReminders,
    onSuccess: (result) => setReminderResult(result as { sent: number }),
  });

  async function handleGenerate() {
    if (!termId) return;
    const ok = await confirm({
      title: "Generate term invoices",
      message: "This will create or refresh invoices for every active learner in the selected term, from the fee structures set up for their grade. Continue?",
      confirmLabel: "Generate invoices",
    });
    if (ok) { setGenResult(null); generate.mutate(); }
  }

  async function handleRemind() {
    const ok = await confirm({
      title: "Send fee reminders",
      message: "This will send a fee reminder to every guardian with an outstanding balance.",
      confirmLabel: "Send reminders",
    });
    if (ok) remind.mutate();
  }

  const sortedPayments = [...((recentPayments as any[] ?? []))]
    .sort((a: any, b: any) => new Date(b.payment_date).getTime() - new Date(a.payment_date).getTime());

  const totalCollected: number = dash?.total_collected ?? 0;
  const outstanding: number = dash?.outstanding_fees_total ?? 0;
  const collectionRate = totalCollected + outstanding > 0
    ? Math.round((totalCollected / (totalCollected + outstanding)) * 100)
    : 0;

  return (
    <div className="space-y-6 pb-8">
      {dialog}

      <Card className="relative overflow-hidden border-0 bg-gradient-to-r from-[#1b365d] via-[#2c5282] to-[#4a9eca] text-white shadow-lg">
        <div className="pointer-events-none absolute -top-16 -right-16 h-48 w-48 rounded-full bg-white/10 blur-2xl" />
        <div className="pointer-events-none absolute -bottom-24 -left-10 h-52 w-52 rounded-full bg-[#4a9eca]/25 blur-3xl" />
        <CardContent className="relative p-5">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-white/15 backdrop-blur border border-white/20">
                <Wallet size={22} />
              </div>
              <div>
                <div className="text-xs uppercase tracking-wider text-white/70 font-semibold">Finance Portal</div>
                <div className="text-xl font-bold mt-0.5">
                  {dash?.term ?? "Fee Management"}
                </div>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <div className="hidden sm:flex items-center gap-2 rounded-full bg-white/10 border border-white/20 px-3.5 py-1.5 text-xs backdrop-blur font-medium">
                <Sparkles size={12} />
                Paybill 247247
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleRemind}
                disabled={remind.isPending}
                className="bg-white/10 border-white/30 text-white hover:bg-white/20 hover:text-white border backdrop-blur shrink-0"
              >
                <Send size={14} className="mr-1.5" />
                {remind.isPending ? "Sending…" : "Fee reminders"}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {reminderResult && (
        <Card className="border-[#4a9eca]/25 bg-gradient-to-r from-[#e3f2fd]/80 to-white">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[#4a9eca]/15">
                <Send size={18} className="text-[#1b365d]" />
              </div>
              <p className="text-sm font-semibold text-[#1b365d]">
                ✓ Sent {reminderResult.sent} fee reminder{reminderResult.sent === 1 ? "" : "s"}.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Active Students"
          value={dash?.student_count ?? "—"}
          icon={Users}
          color="#1b365d"
          featured={true}
        />
        <StatCard
          label="Open Admissions"
          value={dash?.open_admissions_count ?? "—"}
          icon={FileText}
          color="#4a9eca"
          trend={{ value: 2, up: true, label: "this week" }}
        />
        <StatCard
          label="Outstanding Fees"
          value={kes(dash?.outstanding_fees_total)}
          icon={AlertTriangle}
          color="#c62828"
          trend={{ value: 5, up: false, label: "vs last week" }}
          featured={true}
        />
        <StatCard
          label="Payments to Review"
          value={dash?.unreconciled_payment_count ?? "—"}
          icon={RefreshCw}
          color="#3d5a80"
        />
      </div>

      <Card className="relative overflow-hidden">
        <div className="pointer-events-none absolute top-0 right-0 h-40 w-40 rounded-bl-full bg-gradient-to-bl from-[#4a9eca]/15 to-transparent" />
        <CardContent className="relative p-5">
          <div className="flex flex-wrap items-center justify-between gap-6">
            <div className="flex-1 min-w-[240px]">
              <p className="text-xs uppercase tracking-wider text-slate-500 font-semibold">Collection Rate</p>
              <div className="flex items-end gap-2 mt-1">
                <p className="text-4xl font-black text-[#1b365d]">{collectionRate}%</p>
                <p className="text-xs text-muted-foreground mb-1.5">of term target</p>
              </div>
              <div className="mt-3 h-3 w-full max-w-md bg-slate-100 rounded-full overflow-hidden shadow-inner">
                <div
                  className="h-full bg-gradient-to-r from-[#1b365d] via-[#2c5282] to-[#4a9eca] rounded-full transition-all duration-1000 shadow-sm"
                  style={{ width: `${Math.min(100, collectionRate)}%` }}
                />
              </div>
              <div className="mt-2 flex flex-wrap gap-x-5 gap-y-1 text-xs">
                <span className="text-green-700 font-semibold flex items-center gap-1">
                  <TrendingUp size={12} /> Collected: {kes(totalCollected)}
                </span>
                <span className="text-red-600 font-semibold">
                  Outstanding: {kes(outstanding)}
                </span>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3 min-w-[280px]">
              <Link to="/finance/payments" className="group">
                <Card className="h-full hover:shadow-md hover:border-[#4a9eca]/40 transition-all duration-200 cursor-pointer border-slate-200/70">
                  <CardContent className="p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[#e3f2fd] mb-2 group-hover:scale-110 transition-transform">
                      <CreditCard size={18} className="text-[#1b365d]" />
                    </div>
                    <p className="text-sm font-bold text-slate-800">All Payments</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5 flex items-center gap-0.5">
                      View list <ChevronRight size={11} />
                    </p>
                  </CardContent>
                </Card>
              </Link>
              <Link to="/finance/invoices" className="group">
                <Card className="h-full hover:shadow-md hover:border-[#4a9eca]/40 transition-all duration-200 cursor-pointer border-slate-200/70">
                  <CardContent className="p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 mb-2 group-hover:scale-110 transition-transform">
                      <Receipt size={18} className="text-blue-700" />
                    </div>
                    <p className="text-sm font-bold text-slate-800">Invoices</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5 flex items-center gap-0.5">
                      Manage bills <ChevronRight size={11} />
                    </p>
                  </CardContent>
                </Card>
              </Link>
              <Link to="/finance/stk-push" className="group">
                <Card className="h-full hover:shadow-md hover:border-[#4a9eca]/40 transition-all duration-200 cursor-pointer border-slate-200/70">
                  <CardContent className="p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[#4CAF50]/15 mb-2 group-hover:scale-110 transition-transform">
                      <DollarSign size={18} className="text-[#4CAF50]" />
                    </div>
                    <p className="text-sm font-bold text-slate-800">STK Push</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5 flex items-center gap-0.5">
                      Request payment <ChevronRight size={11} />
                    </p>
                  </CardContent>
                </Card>
              </Link>
              <Link to="/finance/unreconciled" className="group">
                <Card className="h-full hover:shadow-md hover:border-[#4a9eca]/40 transition-all duration-200 cursor-pointer border-slate-200/70">
                  <CardContent className="p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[#dbeafe] mb-2 group-hover:scale-110 transition-transform">
                      <BarChart3 size={18} className="text-[#2c5282]" />
                    </div>
                    <p className="text-sm font-bold text-slate-800">Unreconciled</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5 flex items-center gap-0.5">
                      Match items <ChevronRight size={11} />
                    </p>
                  </CardContent>
                </Card>
              </Link>
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="billing" className="space-y-5">
        <TabsList className="h-11 grid w-full max-w-lg grid-cols-3 rounded-xl bg-slate-100/80 p-1 shadow-inner">
          <TabsTrigger value="billing" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Billing</TabsTrigger>
          <TabsTrigger value="payments" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Recent Payments</TabsTrigger>
          <TabsTrigger value="settings" className="rounded-lg font-medium text-sm data-[state=active]:shadow-sm data-[state=active]:bg-white">Settings</TabsTrigger>
        </TabsList>

        <TabsContent value="billing" className="space-y-4">
          <Card className="overflow-hidden border-slate-200/70">
            <CardHeader className="py-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <CardTitle className="text-base text-[#1b365d] flex items-center gap-2">
                    <PiggyBank size={18} /> Generate Term Billing
                  </CardTitle>
                  <CardDescription className="text-xs mt-0.5 max-w-2xl">
                    Creates/refreshes every active student&apos;s invoice for the selected term from the fee
                    structure matching their grade. Set up fee structures first.
                  </CardDescription>
                </div>
                <Button variant="outline" size="sm" asChild className="border-slate-200">
                  <Link to="/finance/fee-structures">Fee structures →</Link>
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap items-end gap-3">
                <div className="w-full sm:w-64">
                  <Label htmlFor="term-select" className="mb-1.5 text-xs font-semibold text-slate-600">
                    Term
                  </Label>
                  <select
                    id="term-select"
                    value={termId}
                    onChange={(e) => setTermId(e.target.value)}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  >
                    <option value="">Select term…</option>
                    {(terms ?? []).map((t: any) => (
                      <option key={t.id} value={t.id}>{t.name}</option>
                    ))}
                  </select>
                </div>
                <div className="flex gap-2">
                  <Button
                    onClick={handleGenerate}
                    disabled={generate.isPending || !termId}
                    className="bg-[#1b365d] hover:bg-[#152a49]"
                  >
                    <FileText size={15} className="mr-1.5" />
                    {generate.isPending ? "Generating…" : "Generate billing"}
                  </Button>
                </div>
              </div>
              {genResult && (
                <div className="mt-4 rounded-xl bg-gradient-to-r from-[#4a9eca]/10 to-transparent border border-[#4a9eca]/20 px-4 py-3">
                  <p className="text-sm font-semibold text-slate-800">
                    ✓ Generated {genResult.generated} invoice{genResult.generated === 1 ? "" : "s"}.
                  </p>
                  {genResult.skipped.length > 0 && (
                    <p className="text-xs text-muted-foreground mt-1">
                      {genResult.skipped.length} student(s) skipped (no fee structure or no class assigned).
                    </p>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="payments" className="space-y-4">
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
              {sortedPayments.length === 0 ? (
                <div className="py-10 text-center text-muted-foreground text-sm px-4">
                  No payments recorded yet.
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-slate-50/80 text-xs text-muted-foreground">
                        <th className="px-4 py-2.5 text-left font-semibold">Payer / Student</th>
                        <th className="px-4 py-2.5 text-right font-semibold">Amount</th>
                        <th className="px-4 py-2.5 text-left font-semibold">Method</th>
                        <th className="px-4 py-2.5 text-right font-semibold">Date</th>
                      </tr>
                    </thead>
                    <tbody>
                      {sortedPayments.slice(0, 8).map((p: any) => (
                        <tr key={p.id} className="border-b border-slate-100 hover:bg-slate-50/60 transition-colors">
                          <td className="px-4 py-3">
                            <p className="font-semibold text-[#1b365d] text-sm truncate max-w-[240px]">
                              {p.payer_name || "Unmatched"}
                            </p>
                            {p.payment_method && (
                              <p className="text-[11px] text-muted-foreground">
                                Receipt #{p.id?.slice(0, 8) ?? "—"}
                              </p>
                            )}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <span className="font-bold text-green-700">{kes(p.amount)}</span>
                          </td>
                          <td className="px-4 py-3">
                            <span className="rounded-full bg-blue-50 border border-blue-100 px-2.5 py-0.5 text-[11px] font-semibold text-blue-700 uppercase">
                              {p.payment_method}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-right text-xs text-muted-foreground font-mono">
                            {new Date(p.payment_date).toLocaleDateString()}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="settings" className="space-y-4">
          <Card className="overflow-hidden">
            <CardHeader className="py-4">
              <CardTitle className="text-base text-[#1b365d]">Fee Listener &amp; Webhooks</CardTitle>
              <CardDescription className="text-xs">
                Automated payment processing pipeline
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="rounded-xl bg-gradient-to-r from-[#e3f2fd]/60 via-white to-[#f0f4f8]/60 border border-[#4a9eca]/15 p-4">
                <div className="flex items-start gap-3">
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[#4a9eca]/15 mt-0.5">
                    <Sparkles size={16} className="text-[#1b365d]" />
                  </div>
                  <div>
                    <p className="text-sm font-bold text-slate-800">M-Pesa C2B Webhook Active</p>
                    <p className="mt-1 text-xs text-slate-600 leading-relaxed">
                      Safaricom&apos;s M-Pesa C2B webhook posts to{" "}
                      <code className="rounded bg-white px-1.5 py-0.5 border border-slate-200 text-[10px] font-mono text-[#1b365d]">
                        /api/webhooks/payments/confirmation
                      </code>{" "}
                      on the payment service. It matches the payment&apos;s account reference to a learner,
                      posts the payment, and updates the open term invoice atomically — the same pipeline
                      the &quot;Simulate Payment&quot; tool exercises for demos.
                    </p>
                  </div>
                </div>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-xl border border-slate-200/70 p-4 bg-white/60">
                  <p className="text-[11px] uppercase tracking-wider text-slate-500 font-semibold mb-1">Paybill Number</p>
                  <p className="font-bold text-lg text-[#1b365d] font-mono">247247</p>
                </div>
                <div className="rounded-xl border border-slate-200/70 p-4 bg-white/60">
                  <p className="text-[11px] uppercase tracking-wider text-slate-500 font-semibold mb-1">Equity Bank Account</p>
                  <p className="font-bold text-lg text-[#1b365d] font-mono break-all">1370263402101</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
