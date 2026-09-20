import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { Badge, Button, Card, Field, Input, PageHeader, Select, Stat, Table, useConfirm } from "../../components/ui";

const CATEGORIES = ["textbook", "storybook", "stationery", "ict", "furniture", "asset"];

export function InventoryPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({ queryKey: ["inventory"], queryFn: schoolApi.inventory });

  const [form, setForm] = useState({
    name: "", category: "stationery", quantity: "0", reorder_level: "5", location: "", supplier: "",
  });
  const [adjustingId, setAdjustingId] = useState<number | null>(null);
  const [adjustQty, setAdjustQty] = useState("");

  const rows = (data as any[] ?? []);
  const needReorder = rows.filter((r) => Number(r.quantity) <= Number(r.reorder_level)).length;

  const create = useMutation({
    mutationFn: () => schoolApi.createInventoryItem(form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["inventory"] });
      setForm({ name: "", category: "stationery", quantity: "0", reorder_level: "5", location: "", supplier: "" });
    },
  });
  const adjust = useMutation({
    mutationFn: (id: number) => schoolApi.updateInventoryItem(id, { quantity: adjustQty }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["inventory"] }); setAdjustingId(null); },
  });

  async function handleCreate() {
    const ok = await confirm({
      title: "Add inventory item",
      message: `Add "${form.name}" (${form.quantity} units) to inventory?`,
      confirmLabel: "Add item",
    });
    if (ok) create.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Inventory" subtitle="Monitor school supplies, stock levels, and reorder pressure." />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat label="Stock items" value={rows.length} />
        <Stat label="Need reorder" value={needReorder} />
        <Stat label="Total units" value={rows.reduce((s, r) => s + Number(r.quantity ?? 0), 0)} />
        <Stat label="Categories" value={new Set(rows.map((r) => r.category)).size} />
      </div>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New item</h3>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="Item name"><Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Field>
          <Field label="Category">
            <Select value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })}>
              {CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
            </Select>
          </Field>
          <Field label="Quantity"><Input type="number" value={form.quantity} onChange={(e) => setForm({ ...form, quantity: e.target.value })} /></Field>
          <Field label="Reorder level"><Input type="number" value={form.reorder_level} onChange={(e) => setForm({ ...form, reorder_level: e.target.value })} /></Field>
          <Field label="Location"><Input value={form.location} onChange={(e) => setForm({ ...form, location: e.target.value })} /></Field>
          <Field label="Supplier"><Input value={form.supplier} onChange={(e) => setForm({ ...form, supplier: e.target.value })} /></Field>
        </div>
        <Button className="mt-4" onClick={handleCreate} disabled={create.isPending || !form.name}>
          {create.isPending ? "Adding…" : "Add item"}
        </Button>
      </Card>

      <Table headers={["Item", "Category", "In stock", "Reorder level", "Status", ""]}>
        {rows.map((r: any) => {
          const low = Number(r.quantity) <= Number(r.reorder_level);
          return (
            <tr key={r.id} className="hover:bg-slate-50">
              <td className="px-4 py-3 font-medium">{r.name}</td>
              <td className="px-4 py-3 capitalize">{r.category || "-"}</td>
              <td className="px-4 py-3">
                {adjustingId === r.id ? (
                  <div className="flex items-center gap-2">
                    <Input className="w-20" type="number" value={adjustQty} onChange={(e) => setAdjustQty(e.target.value)} />
                    <Button onClick={() => adjust.mutate(r.id)} disabled={adjust.isPending}>Save</Button>
                    <Button variant="ghost" onClick={() => setAdjustingId(null)}>Cancel</Button>
                  </div>
                ) : r.quantity}
              </td>
              <td className="px-4 py-3">{r.reorder_level}</td>
              <td className="px-4 py-3"><Badge variant={low ? "danger" : "success"}>{low ? "Reorder" : "Healthy"}</Badge></td>
              <td className="px-4 py-3">
                {adjustingId !== r.id && (
                  <Button variant="ghost" onClick={() => { setAdjustingId(r.id); setAdjustQty(String(r.quantity)); }}>
                    Adjust stock
                  </Button>
                )}
              </td>
            </tr>
          );
        })}
      </Table>
    </div>
  );
}
