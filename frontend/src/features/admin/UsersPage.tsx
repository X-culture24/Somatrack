import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { Badge, Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

const ROLES = [
  "SYSTEM_ADMIN", "SCHOOL_DIRECTOR", "FINANCE_OFFICER", "HEAD_TEACHER", "DEPUTY_HEAD_TEACHER",
  "ACADEMIC_COORDINATOR", "CLASS_TEACHER", "SUBJECT_TEACHER", "LIBRARIAN", "TRANSPORT_COORDINATOR",
  "ADMISSIONS_OFFICER", "RECEPTIONIST", "PARENT", "STUDENT",
];

export function UsersPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data } = useQuery({ queryKey: ["users"], queryFn: () => schoolApi.users() });
  const [form, setForm] = useState({
    first_name: "", last_name: "", email: "", role: "CLASS_TEACHER", phone: "", password: "",
  });

  const create = useMutation({
    mutationFn: () => schoolApi.createUser(form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      setForm({ first_name: "", last_name: "", email: "", role: "CLASS_TEACHER", phone: "", password: "" });
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.email?.[0] ?? err?.response?.data?.detail ?? "Could not create the account. Check the email is unique.");
    },
  });
  const toggleActive = useMutation({
    mutationFn: (u: any) => schoolApi.updateUser(u.id, { is_active: !u.is_active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["users"] }),
  });

  async function handleCreate() {
    const ok = await confirm({
      title: "Create account",
      message: `Create a ${form.role.replaceAll("_", " ").toLowerCase()} account for ${form.first_name} ${form.last_name} (${form.email})?`,
      confirmLabel: "Create account",
    });
    if (ok) create.mutate();
  }

  async function handleToggle(u: any) {
    const ok = await confirm({
      title: u.is_active ? "Deactivate account" : "Activate account",
      message: `${u.is_active ? "Deactivate" : "Activate"} the account for ${u.full_name}?`,
      confirmLabel: u.is_active ? "Deactivate" : "Activate",
      variant: u.is_active ? "danger" : "primary",
    });
    if (ok) toggleActive.mutate(u);
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Users & roles" subtitle="RBAC accounts for all fourteen school roles." />

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New account</h3>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="First name">
            <Input value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} />
          </Field>
          <Field label="Last name">
            <Input value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} />
          </Field>
          <Field label="Email">
            <Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
          </Field>
          <Field label="Role">
            <Select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
              {ROLES.map((r) => <option key={r} value={r}>{r.replaceAll("_", " ")}</option>)}
            </Select>
          </Field>
          <Field label="Phone">
            <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          </Field>
          <Field label="Temporary password">
            <Input
              type="text"
              placeholder="Leave blank to auto-generate"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
            />
          </Field>
        </div>
        <Button
          className="mt-4"
          onClick={handleCreate}
          disabled={create.isPending || !form.first_name || !form.email}
        >
          {create.isPending ? "Creating…" : "Create account"}
        </Button>
      </Card>

      <Table headers={["Name", "Email", "Role", "Portal", "Status", ""]}>
        {(data ?? []).map((u: any) => (
          <tr key={u.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 font-medium">{u.full_name}</td>
            <td className="px-4 py-3">{u.email}</td>
            <td className="px-4 py-3">{u.role}</td>
            <td className="px-4 py-3 capitalize">{u.portal}</td>
            <td className="px-4 py-3">
              <Badge variant={u.is_active ? "success" : "danger"}>{u.is_active ? "Active" : "Inactive"}</Badge>
            </td>
            <td className="px-4 py-3">
              <Button variant="ghost" onClick={() => handleToggle(u)} disabled={toggleActive.isPending}>
                {u.is_active ? "Deactivate" : "Activate"}
              </Button>
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
