import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useParams } from "react-router-dom";
import { academicsApi } from "../../api/academics";
import { transportApi } from "../../api/clubs";
import { guardiansApi, studentsApi, transfersApi } from "../../api/students";
import { Badge, Button, Card, Field, Input, PageHeader, Select, kes, useConfirm } from "../../components/ui";
import { ResultsTrail } from "../results/ResultsTrail";

const PARENTAL_STATUS_LABEL: Record<string, string> = {
  both_alive: "Both parents alive", father_deceased: "Single orphan (father)",
  mother_deceased: "Single orphan (mother)", both_deceased: "Total orphan",
};

export function StudentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState<Record<string, any>>({});
  const [transferring, setTransferring] = useState(false);
  const [transferForm, setTransferForm] = useState({ leaving: false, to_class_id: "", reason: "" });
  const [linkingGuardian, setLinkingGuardian] = useState(false);
  const emptyGuardianForm = {
    first_name: "", last_name: "", phone: "", email: "", relation: "mother", is_primary: false,
  };
  const [guardianForm, setGuardianForm] = useState(emptyGuardianForm);
  const [creatingLoginFor, setCreatingLoginFor] = useState<string | null>(null);

  const { data: student, isLoading } = useQuery({
    queryKey: ["student", id],
    queryFn: () => studentsApi.get(id!),
    enabled: !!id,
  });
  const { data: classes } = useQuery({ queryKey: ["classes"], queryFn: () => academicsApi.classes() });
  const { data: routes } = useQuery({ queryKey: ["transport-routes"], queryFn: () => transportApi.routes() });

  const update = useMutation({
    mutationFn: () =>
      studentsApi.update(id!, {
        first_name: form.first_name,
        middle_name: form.middle_name,
        last_name: form.last_name,
        gender: form.gender,
        date_of_birth: form.date_of_birth || null,
        phone: form.phone,
        email: form.email,
        current_class_id: form.current_class_id || null,
        boarding_status: form.boarding_status,
        status: form.status,
        blood_group: form.blood_group,
        allergies: form.allergies,
        chronic_conditions: form.chronic_conditions,
        residential_address: form.residential_address,
        uses_transport: form.uses_transport,
        transport_route_id: form.uses_transport ? (form.transport_route_id || null) : null,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["student", id] });
      qc.invalidateQueries({ queryKey: ["students"] });
      setEditing(false);
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not save changes.");
    },
  });

  const { data: transferHistory } = useQuery({
    queryKey: ["student-transfers", id],
    queryFn: () => transfersApi.list(id!),
    enabled: !!id,
  });

  const transfer = useMutation({
    mutationFn: () =>
      transfersApi.create({
        student_id: id!,
        to_class_id: transferForm.leaving ? null : (transferForm.to_class_id || null),
        reason: transferForm.reason,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["student", id] });
      qc.invalidateQueries({ queryKey: ["students"] });
      qc.invalidateQueries({ queryKey: ["student-transfers", id] });
      setTransferring(false);
      setTransferForm({ leaving: false, to_class_id: "", reason: "" });
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not record this transfer.");
    },
  });

  async function handleTransfer() {
    const destLabel = transferForm.leaving
      ? "leave the school (marked transferred out, deactivated)"
      : `move to "${(classes as any[] ?? []).find((c: any) => c.id === transferForm.to_class_id)?.name ?? "the selected class"}"`;
    const ok = await confirm({
      title: "Transfer student",
      message: `${student.full_name} will ${destLabel}. This updates their class immediately and is recorded in transfer history. Continue?`,
      confirmLabel: "Transfer",
    });
    if (ok) transfer.mutate();
  }

  const linkGuardian = useMutation({
    mutationFn: () =>
      studentsApi.linkGuardian(id!, {
        first_name: guardianForm.first_name,
        last_name: guardianForm.last_name,
        phone: guardianForm.phone,
        email: guardianForm.email,
        relation: guardianForm.relation,
        is_primary: guardianForm.is_primary,
        has_custody: true,
        contact_priority: guardianForm.is_primary ? 1 : 2,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["student", id] });
      setLinkingGuardian(false);
      setGuardianForm(emptyGuardianForm);
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not link this guardian — check the phone/email aren't already used by someone else.");
    },
  });

  const createLogin = useMutation({
    mutationFn: (guardianId: string) => guardiansApi.createLogin(guardianId),
    onSuccess: ({ temporary_password }) => {
      qc.invalidateQueries({ queryKey: ["student", id] });
      window.alert(`Parent login created. One-time temporary password: ${temporary_password}\n\nShare this with the parent directly — it will not be shown again.`);
    },
    onError: (err: any) => {
      window.alert(err?.response?.data?.message ?? "Could not create a login for this guardian.");
    },
    onSettled: () => setCreatingLoginFor(null),
  });

  async function handleLinkGuardian() {
    const ok = await confirm({
      title: "Link guardian",
      message: `Link ${guardianForm.first_name} ${guardianForm.last_name} as ${student.full_name}'s ${guardianForm.relation}?`,
      confirmLabel: "Link guardian",
    });
    if (ok) linkGuardian.mutate();
  }

  async function handleCreateLogin(guardianId: string, name: string) {
    const ok = await confirm({
      title: "Create parent login",
      message: `Create a parent-portal login for ${name}? This lets them log into the app and see ${student.full_name}'s fees, receipts, and class work.`,
      confirmLabel: "Create login",
    });
    if (ok) {
      setCreatingLoginFor(guardianId);
      createLogin.mutate(guardianId);
    }
  }

  function startEdit() {
    setForm({
      first_name: student.first_name ?? "",
      middle_name: student.middle_name ?? "",
      last_name: student.last_name ?? "",
      gender: student.gender ?? "female",
      date_of_birth: student.date_of_birth ?? "",
      phone: student.phone ?? "",
      email: student.email ?? "",
      current_class_id: student.current_class_id ?? "",
      boarding_status: student.boarding_status ?? "day",
      status: student.status ?? "active",
      blood_group: student.blood_group ?? "",
      allergies: student.allergies ?? "",
      chronic_conditions: student.chronic_conditions ?? "",
      residential_address: student.residential_address ?? "",
      uses_transport: !!student.uses_transport,
      transport_route_id: student.transport_route_id ?? "",
    });
    setEditing(true);
  }

  if (isLoading) return <div className="p-8 text-center text-slate-400">Loading…</div>;
  if (!student) return null;

  const guardians: any[] = student.guardians ?? [];
  const clubs: any[] = student.clubs ?? [];
  const fees = student.fee_summary ?? { total_due: 0, total_paid: 0, total_waivers: 0, balance: 0 };
  const balance = Number(student.fee_balance ?? fees.balance ?? 0);
  const isCredit = balance < 0;
  const isSettled = balance === 0 && fees.total_due > 0;

  return (
    <div className="space-y-5 max-w-2xl">
      {dialog}
      <PageHeader
        title={student.full_name}
        subtitle={`Admission No: ${student.admission_no}`}
        actions={
          editing ? (
            <div className="flex gap-2">
              <Button variant="ghost" onClick={() => setEditing(false)}>Cancel</Button>
              <Button onClick={() => update.mutate()} disabled={update.isPending}>Save</Button>
            </div>
          ) : (
            <div className="flex gap-2">
              <Button variant="ghost" onClick={() => setTransferring((v) => !v)}>
                {transferring ? "Close" : "Transfer"}
              </Button>
              <Button onClick={startEdit}>Edit</Button>
            </div>
          )
        }
      />

      {transferring && (
        <Card>
          <h3 className="mb-3 font-semibold text-[#1b365d]">Transfer student</h3>
          <p className="mb-3 text-xs text-slate-500">
            Currently in <span className="font-medium text-slate-700">
              {(classes as any[] ?? []).find((c: any) => c.id === student.current_class_id)?.name ?? "no class"}
            </span>. Moving to a new class re-runs billing for the new grade immediately; leaving the school marks
            the student "transferred" and deactivates their record.
          </p>
          <div className="grid gap-3 md:grid-cols-2">
            <Field label="Destination">
              <Select
                value={transferForm.leaving ? "__leaving__" : transferForm.to_class_id}
                onChange={(e) =>
                  e.target.value === "__leaving__"
                    ? setTransferForm({ ...transferForm, leaving: true, to_class_id: "" })
                    : setTransferForm({ ...transferForm, leaving: false, to_class_id: e.target.value })
                }
              >
                <option value="">Select a class…</option>
                {(classes as any[] ?? []).filter((c: any) => c.id !== student.current_class_id).map((c: any) => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
                <option value="__leaving__">Leaving the school entirely</option>
              </Select>
            </Field>
            <Field label="Reason">
              <Input
                value={transferForm.reason}
                onChange={(e) => setTransferForm({ ...transferForm, reason: e.target.value })}
                placeholder="e.g. Stream balancing, relocating, sibling's school"
              />
            </Field>
          </div>
          <Button
            className="mt-3"
            onClick={handleTransfer}
            disabled={transfer.isPending || (!transferForm.leaving && !transferForm.to_class_id)}
          >
            {transfer.isPending ? "Saving…" : "Confirm transfer"}
          </Button>
        </Card>
      )}

      {editing && (
        <Card>
          <h3 className="mb-3 font-semibold text-[#1b365d]">Edit student</h3>
          <div className="grid gap-3 md:grid-cols-3">
            <Field label="First name"><Input value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></Field>
            <Field label="Middle name"><Input value={form.middle_name} onChange={(e) => setForm({ ...form, middle_name: e.target.value })} /></Field>
            <Field label="Last name"><Input value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></Field>
            <Field label="Gender">
              <Select value={form.gender} onChange={(e) => setForm({ ...form, gender: e.target.value })}>
                <option value="female">Female</option>
                <option value="male">Male</option>
                <option value="other">Other</option>
              </Select>
            </Field>
            <Field label="Date of birth"><Input type="date" value={form.date_of_birth} onChange={(e) => setForm({ ...form, date_of_birth: e.target.value })} /></Field>
            <Field label="Phone"><Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
            <Field label="Email"><Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
            <Field label="Class">
              <Select value={form.current_class_id} onChange={(e) => setForm({ ...form, current_class_id: e.target.value })}>
                <option value="">Unassigned</option>
                {(classes as any[] ?? []).map((c: any) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </Select>
            </Field>
            <Field label="Boarding status">
              <Select value={form.boarding_status} onChange={(e) => setForm({ ...form, boarding_status: e.target.value })}>
                <option value="day">Day</option>
                <option value="boarding">Boarding</option>
              </Select>
            </Field>
            <Field label="Status">
              <Select value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
                <option value="transferred">Transferred</option>
                <option value="withdrawn">Withdrawn</option>
                <option value="suspended">Suspended</option>
                <option value="graduated">Graduated</option>
              </Select>
            </Field>
            <Field label="Residential address"><Input value={form.residential_address} onChange={(e) => setForm({ ...form, residential_address: e.target.value })} /></Field>
            <Field label="Blood group"><Input value={form.blood_group} onChange={(e) => setForm({ ...form, blood_group: e.target.value })} /></Field>
            <Field label="Allergies"><Input value={form.allergies} onChange={(e) => setForm({ ...form, allergies: e.target.value })} /></Field>
            <Field label="Chronic conditions"><Input value={form.chronic_conditions} onChange={(e) => setForm({ ...form, chronic_conditions: e.target.value })} /></Field>
            <Field label="Uses school transport">
              <label className="flex items-center gap-2 pt-2 text-sm">
                <input type="checkbox" checked={form.uses_transport} onChange={(e) => setForm({ ...form, uses_transport: e.target.checked })} />
                Yes
              </label>
            </Field>
            {form.uses_transport && (
              <Field label="Route">
                <Select value={form.transport_route_id} onChange={(e) => setForm({ ...form, transport_route_id: e.target.value })}>
                  <option value="">Select route…</option>
                  {(routes as any[] ?? []).map((rt: any) => (
                    <option key={rt.id} value={rt.id}>{rt.route_name}</option>
                  ))}
                </Select>
              </Field>
            )}
          </div>
        </Card>
      )}

      <Card className={isCredit ? "border-2 border-green-300" : balance > 0 ? "border-2 border-red-300" : undefined}>
        <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">Fee balance</p>
        <div className="flex items-baseline justify-between">
          <span className={`text-2xl font-bold ${isCredit || isSettled ? "text-green-600" : balance > 0 ? "text-red-600" : "text-slate-700"}`}>
            {kes(Math.abs(balance))}
          </span>
          <Badge variant={isCredit ? "success" : isSettled ? "success" : balance > 0 ? "danger" : "default"}>
            {isCredit ? "Credit (overpaid)" : isSettled ? "Fully paid" : balance > 0 ? "Owing" : "No invoice yet"}
          </Badge>
        </div>
        <dl className="mt-3 grid grid-cols-3 gap-2 text-center text-xs">
          <div>
            <dt className="text-slate-400">Billed</dt>
            <dd className="font-medium text-slate-700">{kes(fees.total_due)}</dd>
          </div>
          <div>
            <dt className="text-slate-400">Paid</dt>
            <dd className="font-medium text-green-600">{kes(fees.total_paid)}</dd>
          </div>
          <div>
            <dt className="text-slate-400">Waivers</dt>
            <dd className="font-medium text-slate-700">{kes(fees.total_waivers)}</dd>
          </div>
        </dl>
      </Card>

      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">Profile</p>
          <dl className="space-y-1.5 text-sm">
            <div className="flex justify-between"><dt className="text-slate-500">Gender</dt><dd className="font-medium capitalize">{student.gender}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Date of birth</dt><dd className="font-medium">{student.date_of_birth ?? "—"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Boarding status</dt><dd className="font-medium capitalize">{student.boarding_status}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Status</dt><dd className="font-medium capitalize">{student.status}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Admitted</dt><dd className="font-medium">{student.admission_date ?? "—"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">KEMIS no.</dt><dd className="font-medium">{student.kemis_number || "—"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Assessment no.</dt><dd className="font-medium">{student.assessment_number || "—"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Transport</dt><dd className="font-medium">{student.uses_transport ? "Uses school transport" : "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">UPI</dt><dd className="text-right font-medium">{student.upi || "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Birth cert. no.</dt><dd className="text-right font-medium">{student.birth_certificate_no || "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Student status</dt><dd className="text-right font-medium">{PARENTAL_STATUS_LABEL[student.parental_status] ?? "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Religion</dt><dd className="text-right font-medium">{student.religion || "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Special need</dt><dd className="text-right font-medium">{student.has_special_needs ? (student.special_needs || "Yes") : "No"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Former school</dt><dd className="text-right font-medium">{student.former_school || "—"}{student.kcpe_marks != null ? ` (entry ${student.kcpe_marks})` : ""}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Citizenship</dt><dd className="text-right font-medium">{student.citizenship || "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Home county</dt><dd className="text-right font-medium">{student.home_county || "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Residence</dt><dd className="text-right font-medium">{[student.residential_address, student.sub_county, student.residence_county].filter(Boolean).join(", ") || "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Photo consent</dt><dd className="text-right font-medium">{student.photo_consent ? "Given" : "Not given"}</dd></div>
          </dl>
        </Card>
        <Card>
          <p className="text-xs text-slate-500 uppercase tracking-wide mb-2">Medical</p>
          <dl className="space-y-1.5 text-sm">
            <div className="flex justify-between"><dt className="text-slate-500">Blood group</dt><dd className="font-medium">{student.blood_group || "—"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Allergies</dt><dd className="font-medium">{student.allergies || "—"}</dd></div>
            <div className="flex justify-between"><dt className="text-slate-500">Chronic conditions</dt><dd className="font-medium">{student.chronic_conditions || "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Doctor</dt><dd className="text-right font-medium">{[student.doctor_name, student.doctor_phone].filter(Boolean).join(" · ") || "—"}</dd></div>
            <div className="flex justify-between gap-3"><dt className="text-slate-500">Preferred hospital</dt><dd className="text-right font-medium">{student.preferred_hospital || "—"}</dd></div>
          </dl>
          {student.has_scholarship && (
            <>
              <p className="mt-4 text-xs text-slate-500 uppercase tracking-wide mb-2">Scholarship / sponsorship</p>
              <dl className="space-y-1.5 text-sm">
                <div className="flex justify-between gap-3"><dt className="text-slate-500">Type</dt><dd className="text-right font-medium"><span className="capitalize">{student.scholarship_type || "—"}</span></dd></div>
                <div className="flex justify-between gap-3"><dt className="text-slate-500">Sponsor</dt><dd className="text-right font-medium">{student.sponsor_name || "—"}</dd></div>
                <div className="flex justify-between gap-3"><dt className="text-slate-500">Contact</dt><dd className="text-right font-medium">{[student.sponsor_contact_person, student.sponsor_phone, student.sponsor_contact_details].filter(Boolean).join(" · ") || "—"}</dd></div>
                <div className="flex justify-between gap-3"><dt className="text-slate-500">Location</dt><dd className="text-right font-medium">{student.sponsor_location || "—"}</dd></div>
              </dl>
            </>
          )}
          <p className="mt-4 text-xs text-slate-500 uppercase tracking-wide mb-2">Documents on file</p>
          <div className="flex flex-wrap gap-1.5">
            {student.photo_path ? <Badge variant="success">Photo</Badge> : <Badge>No photo</Badge>}
            {student.birth_cert_document_path ? <Badge variant="success">Birth cert.</Badge> : <Badge>No birth cert.</Badge>}
          </div>
        </Card>
      </div>

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Guardians</p>
          <Button variant="ghost" onClick={() => setLinkingGuardian((v) => !v)}>
            {linkingGuardian ? "Cancel" : "+ Link guardian"}
          </Button>
        </div>

        {linkingGuardian && (
          <div className="mb-4 rounded-lg border border-slate-200 p-3">
            <div className="grid gap-2 sm:grid-cols-2">
              <Field label="First name"><Input value={guardianForm.first_name} onChange={(e) => setGuardianForm({ ...guardianForm, first_name: e.target.value })} /></Field>
              <Field label="Last name"><Input value={guardianForm.last_name} onChange={(e) => setGuardianForm({ ...guardianForm, last_name: e.target.value })} /></Field>
              <Field label="Phone"><Input value={guardianForm.phone} onChange={(e) => setGuardianForm({ ...guardianForm, phone: e.target.value })} /></Field>
              <Field label="Email (needed for a login)"><Input type="email" value={guardianForm.email} onChange={(e) => setGuardianForm({ ...guardianForm, email: e.target.value })} /></Field>
              <Field label="Relation">
                <Select value={guardianForm.relation} onChange={(e) => setGuardianForm({ ...guardianForm, relation: e.target.value })}>
                  <option value="mother">Mother</option>
                  <option value="father">Father</option>
                  <option value="guardian">Guardian</option>
                  <option value="sibling">Sibling</option>
                  <option value="other">Other</option>
                </Select>
              </Field>
              <Field label="Primary contact">
                <label className="flex items-center gap-2 pt-2 text-sm">
                  <input type="checkbox" checked={guardianForm.is_primary} onChange={(e) => setGuardianForm({ ...guardianForm, is_primary: e.target.checked })} />
                  Yes
                </label>
              </Field>
            </div>
            <p className="mt-2 text-xs text-slate-400">
              This links a new guardian record. If they already exist as a guardian for a sibling, add them via the Guardians page instead to avoid a duplicate.
            </p>
            <Button
              className="mt-3"
              onClick={handleLinkGuardian}
              disabled={linkGuardian.isPending || !guardianForm.first_name || !guardianForm.last_name || !guardianForm.phone}
            >
              {linkGuardian.isPending ? "Linking…" : "Link guardian"}
            </Button>
          </div>
        )}

        {guardians.length === 0 ? (
          <p className="text-sm text-slate-400">No guardian on file yet.</p>
        ) : (
          <ul className="space-y-2">
            {guardians.map((g) => (
              <li key={g.guardian_id} className="flex items-center justify-between rounded-lg border border-slate-100 px-3 py-2 text-sm">
                <div>
                  <p className="font-medium text-[#1b365d]">{g.first_name} {g.last_name}</p>
                  <p className="text-slate-500">{g.phone} {g.email ? `· ${g.email}` : ""}</p>
                </div>
                <div className="flex items-center gap-1.5">
                  <Badge variant="blue">{g.relation}</Badge>
                  {g.is_primary && <Badge variant="navy">Primary</Badge>}
                  {g.has_login ? (
                    <Badge variant="success">Has app login</Badge>
                  ) : (
                    <Button
                      variant="ghost"
                      onClick={() => handleCreateLogin(g.guardian_id, `${g.first_name} ${g.last_name}`)}
                      disabled={creatingLoginFor === g.guardian_id}
                    >
                      {creatingLoginFor === g.guardian_id ? "Creating…" : "Create login"}
                    </Button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Card>
        <p className="text-xs text-slate-500 uppercase tracking-wide mb-3">Co-curricular</p>
        {clubs.length === 0 ? (
          <p className="text-sm text-slate-400">Not enrolled in any club yet.</p>
        ) : (
          <div className="flex flex-wrap gap-1.5">
            {clubs.map((c) => <Badge key={c.club_id} variant="blue">{c.name}</Badge>)}
          </div>
        )}
      </Card>

      <Card>
        <p className="text-xs text-slate-500 uppercase tracking-wide mb-3">Transfer history</p>
        {(transferHistory as any[] ?? []).length === 0 ? (
          <p className="text-sm text-slate-400">No transfers on record.</p>
        ) : (
          <ul className="space-y-2">
            {(transferHistory as any[]).map((t: any) => (
              <li key={t.id} className="rounded-lg border border-slate-100 px-3 py-2 text-sm">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-[#1b365d]">
                    {t.from_class_name || "—"} → {t.to_class_name || "Left the school"}
                  </span>
                  <span className="text-xs text-slate-400">{t.transfer_date}</span>
                </div>
                {t.reason && <p className="mt-1 text-slate-500">{t.reason}</p>}
                {t.approved_by && <p className="mt-1 text-xs text-slate-400">Approved by {t.approved_by}</p>}
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Results trail</h3>
        <ResultsTrail studentId={id!} />
      </Card>
    </div>
  );
}
