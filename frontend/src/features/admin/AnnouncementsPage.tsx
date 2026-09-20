import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

export function AnnouncementsPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({
    queryKey: ["announcements"],
    queryFn: schoolApi.announcements,
  });
  const [form, setForm] = useState({ title: "", body: "", audience: "school" });

  const create = useMutation({
    mutationFn: () => schoolApi.createAnnouncement(form),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["announcements"] }); setForm({ title: "", body: "", audience: "school" }); },
  });
  const togglePin = useMutation({
    mutationFn: (a: any) => schoolApi.updateAnnouncement(a.id, { is_pinned: !a.is_pinned }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["announcements"] }),
  });
  const remove = useMutation({
    mutationFn: (id: number) => schoolApi.deleteAnnouncement(id),
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
              value={form.body} onChange={(e) => setForm({ ...form, body: e.target.value })} />
          </Field>
          <Field label="Audience">
            <Select value={form.audience} onChange={(e) => setForm({ ...form, audience: e.target.value })}>
              <option value="school">Whole school</option>
              <option value="parents">Parents</option>
              <option value="staff">Staff</option>
            </Select>
          </Field>
        </div>
        <Button className="mt-3" onClick={handleCreate} disabled={create.isPending || !form.title}>Post announcement</Button>
      </Card>
      <Table headers={["Title", "Audience", "Published", "Pinned", ""]}>
        {(data ?? []).map((a: any) => (
          <tr key={a.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 font-medium">{a.title}</td>
            <td className="px-4 py-3 capitalize">{a.audience}</td>
            <td className="px-4 py-3 text-xs text-slate-500">{new Date(a.published_at).toLocaleDateString()}</td>
            <td className="px-4 py-3">{a.is_pinned ? "📌" : ""}</td>
            <td className="px-4 py-3 flex gap-2">
              <Button variant="ghost" onClick={() => togglePin.mutate(a)} disabled={togglePin.isPending}>
                {a.is_pinned ? "Unpin" : "Pin"}
              </Button>
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
