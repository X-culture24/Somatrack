import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { api } from "../../api/client";
import { unwrap } from "../../api/client";
import { Card, PageHeader, Stat, Table } from "../../components/ui";

type Column = { label: string; value: (row: any) => ReactNode };
type ModuleConfig = {
  title: string;
  subtitle: string;
  endpoint: string;
  queryKey: string;
  columns: Column[];
  stats?: { label: string; value: (rows: any[]) => string | number }[];
};

export function ModulePage({ config }: { config: ModuleConfig }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: [config.queryKey],
    queryFn: () => api.get(config.endpoint).then((r) => unwrap(r.data)),
  });
  const rows = data ?? [];
  return (
    <div className="space-y-6">
      <PageHeader title={config.title} subtitle={config.subtitle} />
      {config.stats ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {config.stats.map((s) => <Stat key={s.label} label={s.label} value={s.value(rows)} />)}
        </div>
      ) : null}
      {isLoading ? <Card><p className="text-sm text-slate-600">Loading…</p></Card> : null}
      {isError ? <Card><p className="text-sm text-red-700">Could not load. Check the API and try again.</p></Card> : null}
      {!isLoading && !isError ? (
        <Table headers={config.columns.map((c) => c.label)}>
          {rows.length ? rows.map((row: any) => (
            <tr key={row.id} className="hover:bg-slate-50">
              {config.columns.map((c) => <td key={c.label} className="px-4 py-3">{c.value(row)}</td>)}
            </tr>
          )) : (
            <tr><td className="px-4 py-8 text-center text-slate-500" colSpan={config.columns.length}>No records yet.</td></tr>
          )}
        </Table>
      ) : null}
    </div>
  );
}
