import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { resourcesApi, type ResourceSummary } from "../../api/resources";
import { Card, PageHeader } from "../../components/ui";

// Module order mirrors the school's left rail (see mock-up/).
const MODULE_ORDER = ["Students", "Boarding", "Front office", "Accounts", "Procurement", "Academics", "Inventory", "People", "Communication"];

export function RecordsIndex() {
  const { data, isLoading, isError } = useQuery({ queryKey: ["resources"], queryFn: resourcesApi.list });

  const groups = new Map<string, ResourceSummary[]>();
  for (const r of data ?? []) {
    groups.set(r.module, [...(groups.get(r.module) ?? []), r]);
  }
  const modules = [...groups.keys()].sort(
    (a, b) => (MODULE_ORDER.indexOf(a) + 1 || 99) - (MODULE_ORDER.indexOf(b) + 1 || 99),
  );

  return (
    <div>
      <PageHeader title="School records" subtitle="Every register the school keeps, grouped by department." />
      {isLoading && <Card><p className="text-sm text-slate-500">Loading…</p></Card>}
      {isError && <Card><p className="text-sm text-red-700">Could not load the list of records.</p></Card>}
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {modules.map((m) => (
          <Card key={m}>
            <h2 className="mb-3 text-sm font-bold tracking-wide text-[#1b365d] uppercase">{m}</h2>
            <ul className="space-y-1">
              {groups.get(m)!.map((r) => (
                <li key={r.name}>
                  <Link to={`/records/${r.name}`} className="flex items-center justify-between rounded-lg px-2 py-1.5 text-sm text-slate-700 hover:bg-slate-50">
                    <span>{r.title}</span>
                    {!r.can_write && <span className="text-xs text-slate-400">view only</span>}
                  </Link>
                </li>
              ))}
            </ul>
          </Card>
        ))}
      </div>
    </div>
  );
}
