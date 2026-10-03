import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { Badge, Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

const PRIORITY_VARIANT: Record<string, "success" | "warning" | "danger" | "default"> = {
  low: "default", normal: "default", high: "warning", urgent: "danger",
};

export function AnnouncementsPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({
    queryKey: ["announcements"],
    queryFn: () => schoolApi.announcements(),
  });
  const [form, setForm] = useState({ title: "", content: "", audience: "all", priority: "normal" });

  const create = useMutation({
    mutationFn: () => schoolApi.createAnnouncement(form),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["announcements"] }); setForm({ title: "", content: "", audience: "all", priority: "normal" }); },
  });
  const remove = useMutation({
    mutationFn: (id: string) => schoolApi.deleteAnnouncement(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["announcements"] }),
  });

  async function handleCreate() {
    const ok = await confirm({ title: "Post announcement", message: `Post "${form.title}" to ${form.audience}?`, confirmLabel: "Post" });
    if (ok) create.mutate();
  }

  async function handleDelete(a: any) {
    const ok = await confirm({
      title: "Delete announcement",
      message: `Delete "${a.title}"? This cannot be undone.`,
      confirmLabel: "Delete",
      variant: "danger",
    });
    if (ok) remove.mutate(a.id);
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Announcements" />
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New announcement</h3>
        <div className="grid gap-3">
          <Field label="Title"><Input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
          <Field label="Message">
            <textarea rows={3} className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca]"
              value={form.content} onChange={(e) => setForm({ ...form, content: e.target.value })} />
          </Field>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Audience">
              <Select value={form.audience} onChange={(e) => setForm({ ...form, audience: e.target.value })}>
                <option value="all">Whole school</option>
                <option value="staff">Staff</option>
                <option value="parents">Parents</option>
                <option value="students">Students</option>
                <option value="teachers">Teachers</option>
                <option value="finance">Finance</option>
                <option value="administration">Administration</option>
              </Select>
            </Field>
            <Field label="Priority">
              <Select value={form.priority} onChange={(e) => setForm({ ...form, priority: e.target.value })}>
                <option value="low">Low</option>
                <option value="normal">Normal</option>
                <option value="high">High</option>
                <option value="urgent">Urgent</option>
              </Select>
            </Field>
          </div>
        </div>
        <Button className="mt-3" onClick={handleCreate} disabled={create.isPending || !form.title || !form.content}>Post announcement</Button>
      </Card>
      <Table headers={["Title", "Audience", "Priority", "Published", "Posted by", ""]}>
        {(data ?? []).length === 0 && (
          <tr><td colSpan={6} className="px-4 py-8 text-center text-sm text-slate-400">No announcements yet.</td></tr>
        )}
        {(data ?? []).map((a: any) => (
          <tr key={a.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 font-medium">{a.title}</td>
            <td className="px-4 py-3 capitalize">{a.audience}</td>
            <td className="px-4 py-3"><Badge variant={PRIORITY_VARIANT[a.priority] ?? "default"}>{a.priority}</Badge></td>
            <td className="px-4 py-3 text-xs text-slate-500">{new Date(a.publish_date).toLocaleDateString()}</td>
            <td className="px-4 py-3 text-xs text-slate-500">{a.created_by_name || "—"}</td>
            <td className="px-4 py-3">
              <Button variant="ghost" onClick={() => handleDelete(a)} disabled={remove.isPending}>
                Delete
              </Button>
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
