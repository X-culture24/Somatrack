import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { schoolApi } from "../../api/school";
import { studentsApi } from "../../api/students";
import { Button, Card, Field, Input, PageHeader, Select, Stat, Table, kes, useConfirm } from "../../components/ui";

export function TransportManagementPage() {
  const qc = useQueryClient();
  const { confirm, dialog } = useConfirm();
  const [routeId, setRouteId] = useState("");

  const { data: routes } = useQuery({ queryKey: ["transport-routes"], queryFn: schoolApi.transport });
  const { data: vehicles } = useQuery({ queryKey: ["vehicles"], queryFn: () => schoolApi.vehicles() });
  const { data: allStudents } = useQuery({ queryKey: ["students", ""], queryFn: () => studentsApi.list() });

  const [routeForm, setRouteForm] = useState({ name: "", area: "", monthly_charge: "" });
  const [vehicleForm, setVehicleForm] = useState({ registration: "", driver_name: "", driver_phone: "", capacity: "14", route: "" });
  const [alertRouteId, setAlertRouteId] = useState("");
  const [alertMessage, setAlertMessage] = useState("");

  const createRoute = useMutation({
    mutationFn: () => schoolApi.createTransportRoute(routeForm),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["transport-routes"] }); setRouteForm({ name: "", area: "", monthly_charge: "" }); },
  });
  const createVehicle = useMutation({
    mutationFn: () => schoolApi.createVehicle({ ...vehicleForm, route: vehicleForm.route || null }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["vehicles"] }); setVehicleForm({ registration: "", driver_name: "", driver_phone: "", capacity: "14", route: "" }); },
  });
  const sendAlert = useMutation({
    mutationFn: () => {
      const route = (routes as any[] ?? []).find((r: any) => String(r.id) === alertRouteId);
      return schoolApi.createAnnouncement({
        title: `Transport alert — ${route?.name ?? "route"}`,
        body: alertMessage,
        audience: "parents",
      });
    },
    onSuccess: () => { setAlertMessage(""); setAlertRouteId(""); },
  });

  async function handleCreateRoute() {
    const ok = await confirm({ title: "Create route", message: `Create route "${routeForm.name}" at KES ${routeForm.monthly_charge}/month?`, confirmLabel: "Create route" });
    if (ok) createRoute.mutate();
  }
  async function handleCreateVehicle() {
    const ok = await confirm({ title: "Register vehicle", message: `Register vehicle ${vehicleForm.registration}?`, confirmLabel: "Register" });
    if (ok) createVehicle.mutate();
  }
  async function handleSendAlert() {
    const route = (routes as any[] ?? []).find((r: any) => String(r.id) === alertRouteId);
    const ok = await confirm({
      title: "Send transport alert",
      message: `Post this alert (concerning ${route?.name ?? "the selected route"}) to all parents?`,
      confirmLabel: "Send alert",
    });
    if (ok) sendAlert.mutate();
  }

  const routeStudents = routeId
    ? (allStudents as any[] ?? []).filter((s: any) => String(s.transport_route) === routeId)
    : [];

  const totalStudentsOnTransport = (allStudents as any[] ?? []).filter((s: any) => s.transport_route).length;
  const monthlyRevenue = (routes as any[] ?? []).reduce((sum: number, r: any) => {
    const count = (allStudents as any[] ?? []).filter((s: any) => String(s.transport_route) === String(r.id)).length;
    return sum + count * Number(r.monthly_charge);
  }, 0);

  return (
    <div className="space-y-5">
      {dialog}
      <PageHeader
        title="Transport management"
        subtitle="Routes, vehicles, student allocations, and monthly charges."
      />

      <div className="grid gap-3 sm:grid-cols-4">
        <Stat label="Active routes" value={(routes as any[] ?? []).filter((r: any) => r.is_active).length} />
        <Stat label="Vehicles" value={(vehicles as any[] ?? []).length} />
        <Stat label="Students on transport" value={totalStudentsOnTransport} />
        <Stat label="Monthly revenue" value={kes(monthlyRevenue)} />
      </div>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">New route</h3>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="Route name"><Input value={routeForm.name} onChange={(e) => setRouteForm({ ...routeForm, name: e.target.value })} /></Field>
          <Field label="Area"><Input value={routeForm.area} onChange={(e) => setRouteForm({ ...routeForm, area: e.target.value })} /></Field>
          <Field label="Monthly charge (KES)"><Input type="number" value={routeForm.monthly_charge} onChange={(e) => setRouteForm({ ...routeForm, monthly_charge: e.target.value })} /></Field>
        </div>
        <Button className="mt-3" onClick={handleCreateRoute} disabled={createRoute.isPending || !routeForm.name || !routeForm.monthly_charge}>
          {createRoute.isPending ? "Creating…" : "Create route"}
        </Button>
      </Card>

      {/* Routes with student counts */}
      <Table headers={["Route", "Monthly charge", "Term charge", "Students", "Vehicles"]}>
        {(routes as any[] ?? []).map((r: any) => {
          const count = (allStudents as any[] ?? []).filter((s: any) => String(s.transport_route) === String(r.id)).length;
          const routeVehicles = (vehicles as any[] ?? []).filter((v: any) => String(v.route) === String(r.id));
          return (
            <tr key={r.id}
              className={`hover:bg-slate-50 cursor-pointer ${routeId === String(r.id) ? "bg-[#4a9eca]/5" : ""}`}
              onClick={() => setRouteId(routeId === String(r.id) ? "" : String(r.id))}>
              <td className="px-4 py-3 font-medium">
                <div>{r.name}</div>
                {r.area && <div className="text-xs text-slate-400">{r.area}</div>}
              </td>
              <td className="px-4 py-3">{kes(r.monthly_charge)}</td>
              <td className="px-4 py-3 text-slate-600">{kes(Number(r.monthly_charge) * 3)}</td>
              <td className="px-4 py-3">
                <span className="rounded-full bg-[#4a9eca]/15 text-[#1b365d] px-2 py-0.5 text-xs font-medium">{count}</span>
              </td>
              <td className="px-4 py-3 text-xs text-slate-500">
                {routeVehicles.map((v: any) => v.registration).join(", ") || "—"}
              </td>
            </tr>
          );
        })}
      </Table>

      {/* Students on selected route */}
      {routeId && (
        <Card>
          <h3 className="mb-3 font-semibold text-[#1b365d]">
            Students on route — {(routes as any[] ?? []).find((r: any) => String(r.id) === routeId)?.name}
          </h3>
          {routeStudents.length === 0 ? (
            <p className="text-sm text-slate-400">No students assigned to this route.</p>
          ) : (
            <Table headers={["Adm no.", "Name", "Class"]}>
              {routeStudents.map((s: any) => (
                <tr key={s.id} className="hover:bg-slate-50">
                  <td className="px-4 py-2 text-xs font-mono">{s.admission_no}</td>
                  <td className="px-4 py-2 font-medium">{s.full_name}</td>
                  <td className="px-4 py-2 text-sm text-slate-600">{s.class_label}</td>
                </tr>
              ))}
            </Table>
          )}
        </Card>
      )}

      {/* Vehicles */}
      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Register vehicle</h3>
        <div className="grid gap-3 md:grid-cols-5">
          <Field label="Registration"><Input value={vehicleForm.registration} onChange={(e) => setVehicleForm({ ...vehicleForm, registration: e.target.value })} /></Field>
          <Field label="Driver name"><Input value={vehicleForm.driver_name} onChange={(e) => setVehicleForm({ ...vehicleForm, driver_name: e.target.value })} /></Field>
          <Field label="Driver phone"><Input value={vehicleForm.driver_phone} onChange={(e) => setVehicleForm({ ...vehicleForm, driver_phone: e.target.value })} /></Field>
          <Field label="Capacity"><Input type="number" value={vehicleForm.capacity} onChange={(e) => setVehicleForm({ ...vehicleForm, capacity: e.target.value })} /></Field>
          <Field label="Route">
            <Select value={vehicleForm.route} onChange={(e) => setVehicleForm({ ...vehicleForm, route: e.target.value })}>
              <option value="">Unassigned</option>
              {(routes as any[] ?? []).map((r: any) => <option key={r.id} value={r.id}>{r.name}</option>)}
            </Select>
          </Field>
        </div>
        <Button className="mt-3" onClick={handleCreateVehicle} disabled={createVehicle.isPending || !vehicleForm.registration || !vehicleForm.driver_name}>
          {createVehicle.isPending ? "Registering…" : "Register vehicle"}
        </Button>
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Send transport alert</h3>
        <p className="mb-3 text-xs text-slate-500">Posts an announcement to all parents (e.g. delay, route change, breakdown).</p>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="Concerning route">
            <Select value={alertRouteId} onChange={(e) => setAlertRouteId(e.target.value)}>
              <option value="">Select route…</option>
              {(routes as any[] ?? []).map((r: any) => <option key={r.id} value={r.id}>{r.name}</option>)}
            </Select>
          </Field>
          <div className="md:col-span-2">
            <Field label="Message">
              <Input value={alertMessage} onChange={(e) => setAlertMessage(e.target.value)} placeholder="e.g. Running 20 minutes late this afternoon due to traffic." />
            </Field>
          </div>
        </div>
        <Button className="mt-3" onClick={handleSendAlert} disabled={sendAlert.isPending || !alertRouteId || !alertMessage}>
          {sendAlert.isPending ? "Sending…" : "Send alert"}
        </Button>
      </Card>

      <h3 className="font-semibold text-[#1b365d]">Vehicles</h3>
      <Table headers={["Registration", "Type", "Driver", "Capacity", "Route", "Status"]}>
        {(vehicles as any[] ?? []).map((v: any) => (
          <tr key={v.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 font-mono font-medium">{v.registration}</td>
            <td className="px-4 py-3 text-sm capitalize">{v.vehicle_type}</td>
            <td className="px-4 py-3">{v.driver_name} {v.driver_phone && <span className="text-xs text-slate-400">· {v.driver_phone}</span>}</td>
            <td className="px-4 py-3">{v.capacity}</td>
            <td className="px-4 py-3 text-sm text-slate-600">
              {(routes as any[] ?? []).find((r: any) => String(r.id) === String(v.route))?.name ?? "—"}
            </td>
            <td className="px-4 py-3">
              <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${v.status === "active" ? "bg-green-100 text-green-700" : "bg-orange-100 text-orange-700"}`}>
                {v.status}
              </span>
            </td>
          </tr>
        ))}
        {(vehicles as any[] ?? []).length === 0 && (
          <tr><td colSpan={6} className="px-4 py-8 text-center text-slate-400">No vehicles registered yet.</td></tr>
        )}
      </Table>
    </div>
  );
}
