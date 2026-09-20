import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { Button, Card, Field, Input, PageHeader, Select, Table, useConfirm } from "../../components/ui";

const STAFF_ROLES = [
  "HEAD_TEACHER", "DEPUTY_HEAD_TEACHER", "ACADEMIC_COORDINATOR", "CLASS_TEACHER", "SUBJECT_TEACHER",
  "LIBRARIAN", "TRANSPORT_COORDINATOR", "ADMISSIONS_OFFICER", "RECEPTIONIST", "FINANCE_OFFICER",
];

const EMPLOYMENT_TYPES = ["permanent", "contract", "parttime", "temporary", "board"];

export function StaffPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const { data: staff } = useQuery({ queryKey: ["staff"], queryFn: () => schoolApi.staff() });
  const { data: departments } = useQuery({ queryKey: ["departments"], queryFn: schoolApi.departments });

  const [form, setForm] = useState({
    first_name: "", last_name: "", email: "", role: "SUBJECT_TEACHER", phone: "",
    staff_no: "", department: "", job_title: "", employment_type: "permanent", basic_salary: "0",
  });
  const [deptName, setDeptName] = useState("");
  const [editingId, setEditingId] = useState<number | null>(null);
  const [statusChoice, setStatusChoice] = useState("active");

  const createStaff = useMutation({
    mutationFn: async () => {
      const user = await schoolApi.createUser({
        first_name: form.first_name, last_name: form.last_name, email: form.email,
        role: form.role, phone: form.phone,
      });
      return schoolApi.createStaff({
        user: user.id, staff_no: form.staff_no, department: form.department || null,
        job_title: form.job_title, employment_type: form.employment_type, basic_salary: form.basic_salary,
        phone: form.phone,
      });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["staff"] });
      qc.invalidateQueries({ queryKey: ["users"] });
      setForm({ ...form, first_name: "", last_name: "", email: "", staff_no: "", job_title: "" });
    },
    onError: (err: any) => {
      window.alert(
        err?.response?.data?.email?.[0] ?? err?.response?.data?.staff_no?.[0] ?? err?.response?.data?.detail
        ?? "Could not create the staff record. Check the email and staff number are unique.",
      );
    },
  });
  const createDept = useMutation({
    mutationFn: () => schoolApi.createDepartment({ name: deptName }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["departments"] }); setDeptName(""); },
  });
  const updateStatus = useMutation({
    mutationFn: (id: number) => schoolApi.updateStaff(id, { status: statusChoice }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["staff"] }); setEditingId(null); },
  });

  async function handleCreate() {
    const ok = await confirm({
      title: "Create staff record",
      message: `Create a staff record and login for ${form.first_name} ${form.last_name} (${form.role.replaceAll("_", " ").toLowerCase()})?`,
      confirmLabel: "Create staff",
    });
    if (ok) createStaff.mutate();
  }

  return (
    <div className="space-y-6">
      {dialog}
      <PageHeader title="Staff directory" subtitle="Teaching and support staff, departments, and school responsibilities." />

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New staff member</h3>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="First name"><Input value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></Field>
          <Field label="Last name"><Input value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></Field>
          <Field label="Email"><Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
          <Field label="Phone"><Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
          <Field label="Staff no."><Input value={form.staff_no} onChange={(e) => setForm({ ...form, staff_no: e.target.value })} /></Field>
          <Field label="Role">
            <Select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}>
              {STAFF_ROLES.map((r) => <option key={r} value={r}>{r.replaceAll("_", " ")}</option>)}
            </Select>
          </Field>
          <Field label="Department">
            <Select value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })}>
              <option value="">None</option>
              {(departments as any[] ?? []).map((d: any) => <option key={d.id} value={d.id}>{d.name}</option>)}
            </Select>
          </Field>
          <Field label="Job title"><Input value={form.job_title} onChange={(e) => setForm({ ...form, job_title: e.target.value })} /></Field>
          <Field label="Employment type">
            <Select value={form.employment_type} onChange={(e) => setForm({ ...form, employment_type: e.target.value })}>
              {EMPLOYMENT_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
            </Select>
          </Field>
          <Field label="Basic salary (KES)"><Input type="number" value={form.basic_salary} onChange={(e) => setForm({ ...form, basic_salary: e.target.value })} /></Field>
        </div>
        <Button
          className="mt-4"
          onClick={handleCreate}
          disabled={createStaff.isPending || !form.first_name || !form.email || !form.staff_no}
        >
          {createStaff.isPending ? "Creating…" : "Create staff record"}
        </Button>
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Departments</h3>
        <div className="flex flex-wrap items-end gap-3">
          <Field label="New department name">
            <Input value={deptName} onChange={(e) => setDeptName(e.target.value)} className="w-56" />
          </Field>
          <Button onClick={() => createDept.mutate()} disabled={createDept.isPending || !deptName}>Add department</Button>
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          {(departments as any[] ?? []).map((d: any) => (
            <span key={d.id} className="rounded-full bg-[#4a9eca]/10 px-3 py-1 text-xs font-medium text-[#1b365d]">{d.name}</span>
          ))}
        </div>
      </Card>

      <Table headers={["Staff no.", "Name", "Role", "Department", "TSC no.", "Status", ""]}>
        {(staff as any[] ?? []).map((s: any) => (
          <tr key={s.id} className="hover:bg-slate-50">
            <td className="px-4 py-3">{s.staff_no}</td>
            <td className="px-4 py-3 font-medium">{s.full_name ?? "-"}</td>
            <td className="px-4 py-3">{s.job_title || "-"}</td>
            <td className="px-4 py-3">{s.department_name || "-"}</td>
            <td className="px-4 py-3">{s.tsc_number || "-"}</td>
            <td className="px-4 py-3">
              {editingId === s.id ? (
                <div className="flex items-center gap-2">
                  <Select className="w-36" value={statusChoice} onChange={(e) => setStatusChoice(e.target.value)}>
                    <option value="active">Active</option>
                    <option value="on_leave">On leave</option>
                    <option value="suspended">Suspended</option>
                    <option value="resigned">Resigned</option>
                    <option value="terminated">Terminated</option>
                    <option value="retired">Retired</option>
                  </Select>
                  <Button onClick={() => updateStatus.mutate(s.id)} disabled={updateStatus.isPending}>Save</Button>
                </div>
              ) : (
                <span className="capitalize">{(s.status ?? "active").replaceAll("_", " ")}</span>
              )}
            </td>
            <td className="px-4 py-3">
              {editingId !== s.id && (
                <Button variant="ghost" onClick={() => { setEditingId(s.id); setStatusChoice(s.status ?? "active"); }}>
                  Change status
                </Button>
              )}
            </td>
          </tr>
        ))}
      </Table>
    </div>
  );
}
