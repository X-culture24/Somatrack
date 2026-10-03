import { useQuery } from "@tanstack/react-query";
import { feeAnalyticsApi, GRADE_LABELS } from "../../api/finance";
import { Card, kes } from "../../components/ui";

const METHOD_COLORS: Record<string, string> = {
  mpesa: "#1b365d", cash: "#4a9eca", bank: "#c9a24b", cheque: "#8b8b8b",
};
const STATUS_COLORS: Record<string, string> = {
  paid: "#16a34a", partial: "#c9a24b", unpaid: "#dc2626", overdue: "#dc2626",
  cancelled: "#94a3b8", refunded: "#94a3b8",
};

function TwoBarRow({ label, billed, collected, max }: { label: string; billed: number; collected: number; max: number }) {
  const billedPct = max > 0 ? Math.min(100, (Math.max(billed, 0) / max) * 100) : 0;
  const collectedPct = max > 0 ? Math.min(100, (Math.max(collected, 0) / max) * 100) : 0;
  return (
    <div className="mb-3 last:mb-0">
      <div className="mb-1 flex items-baseline justify-between text-xs">
        <span className="font-medium text-slate-700">{label}</span>
        <span className="text-slate-400">
          Billed {kes(billed)} · Collected {kes(collected)}
        </span>
      </div>
      <div className="mb-0.5 h-2 w-full overflow-hidden rounded-full bg-slate-100">
        <div className="h-full rounded-full bg-[#1b365d]" style={{ width: `${billedPct}%` }} />
      </div>
      <div className="h-2 w-full overflow-hidden rounded-full bg-slate-100">
        <div className="h-full rounded-full bg-[#4a9eca]" style={{ width: `${collectedPct}%` }} />
      </div>
    </div>
  );
}

function ShareBars({ rows, colors }: { rows: { label: string; value: number }[]; colors: Record<string, string> }) {
  const total = rows.reduce((sum, r) => sum + r.value, 0);
  if (rows.length === 0 || total <= 0) {
    return <p className="text-sm text-slate-400">No data yet.</p>;
  }
  return (
    <div>
      <div className="mb-3 flex h-3 w-full overflow-hidden rounded-full bg-slate-100">
        {rows.map((r) => (
          <div
            key={r.label}
            style={{ width: `${(r.value / total) * 100}%`, background: colors[r.label.toLowerCase()] ?? "#94a3b8" }}
            title={`${r.label}: ${kes(r.value)}`}
          />
        ))}
      </div>
      <ul className="space-y-1.5 text-sm">
        {rows.map((r) => (
          <li key={r.label} className="flex items-center justify-between">
            <span className="flex items-center gap-2 capitalize text-slate-600">
              <span className="h-2.5 w-2.5 rounded-full" style={{ background: colors[r.label.toLowerCase()] ?? "#94a3b8" }} />
              {r.label}
            </span>
            <span className="font-medium text-slate-800">
              {kes(r.value)} <span className="text-xs text-slate-400">({((r.value / total) * 100).toFixed(0)}%)</span>
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function FeeChartsSection() {
  const { data, isLoading } = useQuery({ queryKey: ["fee-analytics"], queryFn: feeAnalyticsApi.get });

  if (isLoading) return null;
  if (!data) return null;

  const byTerm = data.by_term ?? [];
  const byGrade = [...(data.by_grade ?? [])].sort((a, b) => a.grade - b.grade);
  const byStatus = data.by_status ?? [];
  const byMethod = data.payment_methods ?? [];

  const termMax = Math.max(1, ...byTerm.map((t) => Math.max(t.billed, t.collected)));
  const gradeMax = Math.max(1, ...byGrade.map((g) => Math.max(g.billed, g.collected)));

  const totalBilled = byTerm.reduce((s, t) => s + t.billed, 0);
  const totalCollected = byTerm.reduce((s, t) => s + t.collected, 0);
  const collectionRate = totalBilled > 0 ? Math.round((totalCollected / totalBilled) * 100) : 0;

  return (
    <div className="mt-6 grid gap-4 lg:grid-cols-2">
      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-semibold text-[#1b365d]">Collections by term</h3>
          <span className="text-xs font-medium text-slate-500">{collectionRate}% collected overall</span>
        </div>
        <div className="mb-2 flex items-center gap-4 text-xs text-slate-500">
          <span className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-[#1b365d]" /> Billed</span>
          <span className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-[#4a9eca]" /> Collected</span>
        </div>
        {byTerm.length === 0 ? (
          <p className="text-sm text-slate-400">No invoices generated yet.</p>
        ) : (
          byTerm.map((t) => <TwoBarRow key={t.term_id} label={t.label} billed={t.billed} collected={t.collected} max={termMax} />)
        )}
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Billed vs collected by grade</h3>
        <div className="mb-2 flex items-center gap-4 text-xs text-slate-500">
          <span className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-[#1b365d]" /> Billed</span>
          <span className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-[#4a9eca]" /> Collected</span>
        </div>
        {byGrade.length === 0 ? (
          <p className="text-sm text-slate-400">No invoices generated yet.</p>
        ) : (
          byGrade.map((g) => (
            <TwoBarRow key={g.grade} label={GRADE_LABELS[g.grade] ?? `Grade ${g.grade}`} billed={g.billed} collected={g.collected} max={gradeMax} />
          ))
        )}
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Invoice status mix</h3>
        <ShareBars rows={byStatus.map((s) => ({ label: s.status, value: s.total }))} colors={STATUS_COLORS} />
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Payments by method</h3>
        <ShareBars rows={byMethod.map((m) => ({ label: m.method, value: m.total }))} colors={METHOD_COLORS} />
      </Card>
    </div>
  );
}
