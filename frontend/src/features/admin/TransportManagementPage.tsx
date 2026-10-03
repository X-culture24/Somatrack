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

  const EMPTY_ROUTE = { route_name: "", route_code: "", start_location: "", end_location: "", fare: "", driver_name: "", driver_phone: "" };
  const [routeForm, setRouteForm] = useState(EMPTY_ROUTE);
  const [editingRouteId, setEditingRouteId] = useState<string | null>(null);
  const EMPTY_VEHICLE = { plate_no: "", vehicle_type: "bus", driver_name: "", driver_phone: "", capacity: "40", route_id: "" };
  const [vehicleForm, setVehicleForm] = useState(EMPTY_VEHICLE);
  const [editingVehicleId, setEditingVehicleId] = useState<string | null>(null);
  const [alertRouteId, setAlertRouteId] = useState("");
  const [alertMessage, setAlertMessage] = useState("");

  const routePayload = () => ({ ...routeForm, fare: Number(routeForm.fare) || 0 });

  const createRoute = useMutation({
    mutationFn: () => schoolApi.createTransportRoute(routePayload()),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["transport-routes"] }); setRouteForm(EMPTY_ROUTE); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not create route — the name/code may already exist."),
  });
  const updateRoute = useMutation({
    mutationFn: () => schoolApi.updateTransportRoute(editingRouteId!, routePayload()),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["transport-routes"] }); setRouteForm(EMPTY_ROUTE); setEditingRouteId(null); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not save changes."),
  });

  function startEditRoute(r: any) {
    setEditingRouteId(r.id);
    setRouteForm({
      route_name: r.route_name, route_code: r.route_code, start_location: r.start_location,
      end_location: r.end_location, fare: String(r.fare), driver_name: r.driver_name ?? "", driver_phone: r.driver_phone ?? "",
    });
  }
  function cancelEditRoute() {
    setEditingRouteId(null);
    setRouteForm(EMPTY_ROUTE);
  }
  const vehiclePayload = () => ({
    plate_no: vehicleForm.plate_no,
    vehicle_type: vehicleForm.vehicle_type,
    driver_name: vehicleForm.driver_name,
    driver_phone: vehicleForm.driver_phone,
    capacity: Number(vehicleForm.capacity) || 0,
    route_id: vehicleForm.route_id || null,
  });
  const createVehicle = useMutation({
    mutationFn: () => schoolApi.createVehicle(vehiclePayload()),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["vehicles"] }); setVehicleForm(EMPTY_VEHICLE); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not register this vehicle — the plate number may already exist."),
  });
  const updateVehicle = useMutation({
    mutationFn: () => schoolApi.updateVehicle(editingVehicleId!, vehiclePayload()),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["vehicles"] }); setVehicleForm(EMPTY_VEHICLE); setEditingVehicleId(null); },
    onError: (err: any) => window.alert(err?.response?.data?.message ?? "Could not save changes."),
  });
  function startEditVehicle(v: any) {
    setEditingVehicleId(v.id);
    setVehicleForm({
      plate_no: v.plate_no, vehicle_type: v.vehicle_type ?? "bus", driver_name: v.driver_name ?? "",
      driver_phone: v.driver_phone ?? "", capacity: String(v.capacity ?? 40), route_id: v.route_id ?? "",
    });
  }
  function cancelEditVehicle() {
    setEditingVehicleId(null);
    setVehicleForm(EMPTY_VEHICLE);
  }
  const sendAlert = useMutation({
    mutationFn: () => {
      const route = (routes as any[] ?? []).find((r: any) => String(r.id) === alertRouteId);
      return schoolApi.createAnnouncement({
        title: `Transport alert — ${route?.name ?? "route"}`,
        content: alertMessage,
        audience: "parents",
      });
    },
    onSuccess: () => { setAlertMessage(""); setAlertRouteId(""); },
  });

  async function handleSaveRoute() {
    const isEdit = Boolean(editingRouteId);
    const ok = await confirm({
      title: isEdit ? "Save route changes" : "Create route",
      message: isEdit
        ? `Save changes to "${routeForm.route_name}"?`
        : `Create route "${routeForm.route_name}" at KES ${routeForm.fare}/month?`,
      confirmLabel: isEdit ? "Save changes" : "Create route",
    });
    if (ok) (isEdit ? updateRoute : createRoute).mutate();
  }
  async function handleSaveVehicle() {
    const isEdit = Boolean(editingVehicleId);
    const ok = await confirm({
      title: isEdit ? "Save vehicle changes" : "Register vehicle",
      message: isEdit ? `Save changes to ${vehicleForm.plate_no}?` : `Register vehicle ${vehicleForm.plate_no}?`,
      confirmLabel: isEdit ? "Save changes" : "Register",
    });
    if (ok) (isEdit ? updateVehicle : createVehicle).mutate();
  }
  async function handleSendAlert() {
    const route = (routes as any[] ?? []).find((r: any) => String(r.id) === alertRouteId);
    const ok = await confirm({
      title: "Send transport alert",
      message: `Post this alert (concerning ${route?.route_name ?? "the selected route"}) to all parents?`,
      confirmLabel: "Send alert",
    });
    if (ok) sendAlert.mutate();
  }

  const routeStudents = routeId
    ? (allStudents as any[] ?? []).filter((s: any) => String(s.transport_route_id) === routeId)
    : [];

  const totalStudentsOnTransport = (allStudents as any[] ?? []).filter((s: any) => s.uses_transport && s.transport_route_id).length;
  const monthlyRevenue = (routes as any[] ?? []).reduce((sum: number, r: any) => {
    const count = (allStudents as any[] ?? []).filter((s: any) => String(s.transport_route_id) === String(r.id)).length;
    return sum + count * Number(r.fare);
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
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-semibold text-[#1b365d]">{editingRouteId ? "Edit route" : "New route"}</h3>
          {editingRouteId && <Button variant="ghost" onClick={cancelEditRoute}>Cancel edit</Button>}
        </div>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="Route name"><Input value={routeForm.route_name} onChange={(e) => setRouteForm({ ...routeForm, route_name: e.target.value })} /></Field>
          <Field label="Route code"><Input value={routeForm.route_code} onChange={(e) => setRouteForm({ ...routeForm, route_code: e.target.value })} placeholder="e.g. R1" /></Field>
          <Field label="Fare (KES/month)"><Input type="number" value={routeForm.fare} onChange={(e) => setRouteForm({ ...routeForm, fare: e.target.value })} /></Field>
          <Field label="Start location"><Input value={routeForm.start_location} onChange={(e) => setRouteForm({ ...routeForm, start_location: e.target.value })} /></Field>
          <Field label="End location"><Input value={routeForm.end_location} onChange={(e) => setRouteForm({ ...routeForm, end_location: e.target.value })} /></Field>
          <Field label="Driver name"><Input value={routeForm.driver_name} onChange={(e) => setRouteForm({ ...routeForm, driver_name: e.target.value })} /></Field>
          <Field label="Driver phone"><Input value={routeForm.driver_phone} onChange={(e) => setRouteForm({ ...routeForm, driver_phone: e.target.value })} /></Field>
        </div>
        <Button
          className="mt-3"
          onClick={handleSaveRoute}
          disabled={createRoute.isPending || updateRoute.isPending || !routeForm.route_name || !routeForm.fare}
        >
          {createRoute.isPending || updateRoute.isPending ? "Saving…" : editingRouteId ? "Save changes" : "Create route"}
        </Button>
      </Card>

      {/* Routes with student counts */}
      <Table headers={["Route", "Fare", "Term charge", "Students", "Vehicles", ""]}>
        {(routes as any[] ?? []).map((r: any) => {
          const count = (allStudents as any[] ?? []).filter((s: any) => String(s.transport_route_id) === String(r.id)).length;
          const routeVehicles = (vehicles as any[] ?? []).filter((v: any) => String(v.route_id) === String(r.id));
          return (
            <tr key={r.id}
              className={`hover:bg-slate-50 cursor-pointer ${routeId === String(r.id) ? "bg-[#4a9eca]/5" : ""}`}
              onClick={() => setRouteId(routeId === String(r.id) ? "" : String(r.id))}>
              <td className="px-4 py-3 font-medium">
                <div>{r.route_name}</div>
                <div className="text-xs text-slate-400">{r.start_location} → {r.end_location}</div>
              </td>
              <td className="px-4 py-3">{kes(r.fare)}</td>
              <td className="px-4 py-3 text-slate-600">{kes(Number(r.fare) * 3)}</td>
              <td className="px-4 py-3">
                <span className="rounded-full bg-[#4a9eca]/15 text-[#1b365d] px-2 py-0.5 text-xs font-medium">{count}</span>
              </td>
              <td className="px-4 py-3 text-xs text-slate-500">
                {routeVehicles.map((v: any) => v.registration).join(", ") || "—"}
              </td>
              <td className="px-4 py-3">
                <Button variant="ghost" onClick={(e) => { e.stopPropagation(); startEditRoute(r); }}>Edit</Button>
              </td>
            </tr>
          );
        })}
      </Table>

      {/* Students on selected route */}
      {routeId && (
        <Card>
          <h3 className="mb-3 font-semibold text-[#1b365d]">
            Students on route — {(routes as any[] ?? []).find((r: any) => String(r.id) === routeId)?.route_name}
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
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-semibold text-[#1b365d]">{editingVehicleId ? "Edit vehicle" : "Register vehicle"}</h3>
          {editingVehicleId && <Button variant="ghost" onClick={cancelEditVehicle}>Cancel edit</Button>}
        </div>
        <div className="grid gap-3 md:grid-cols-5">
          <Field label="Plate no."><Input value={vehicleForm.plate_no} onChange={(e) => setVehicleForm({ ...vehicleForm, plate_no: e.target.value })} /></Field>
          <Field label="Type">
            <Select value={vehicleForm.vehicle_type} onChange={(e) => setVehicleForm({ ...vehicleForm, vehicle_type: e.target.value })}>
              <option value="bus">Bus</option>
              <option value="van">Van</option>
              <option value="car">Car</option>
              <option value="other">Other</option>
            </Select>
          </Field>
          <Field label="Driver name"><Input value={vehicleForm.driver_name} onChange={(e) => setVehicleForm({ ...vehicleForm, driver_name: e.target.value })} /></Field>
          <Field label="Driver phone"><Input value={vehicleForm.driver_phone} onChange={(e) => setVehicleForm({ ...vehicleForm, driver_phone: e.target.value })} /></Field>
          <Field label="Capacity"><Input type="number" value={vehicleForm.capacity} onChange={(e) => setVehicleForm({ ...vehicleForm, capacity: e.target.value })} /></Field>
          <Field label="Route">
            <Select value={vehicleForm.route_id} onChange={(e) => setVehicleForm({ ...vehicleForm, route_id: e.target.value })}>
              <option value="">Unassigned</option>
              {(routes as any[] ?? []).map((r: any) => <option key={r.id} value={r.id}>{r.route_name}</option>)}
            </Select>
          </Field>
        </div>
        <Button
          className="mt-3"
          onClick={handleSaveVehicle}
          disabled={createVehicle.isPending || updateVehicle.isPending || !vehicleForm.plate_no || !vehicleForm.driver_name}
        >
          {createVehicle.isPending || updateVehicle.isPending ? "Saving…" : editingVehicleId ? "Save changes" : "Register vehicle"}
        </Button>
      </Card>

      <Card>
        <h3 className="mb-3 font-semibold text-[#1b365d]">Send transport alert</h3>
        <p className="mb-3 text-xs text-slate-500">Posts an announcement to all parents (e.g. delay, route change, breakdown).</p>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="Concerning route">
            <Select value={alertRouteId} onChange={(e) => setAlertRouteId(e.target.value)}>
              <option value="">Select route…</option>
              {(routes as any[] ?? []).map((r: any) => <option key={r.id} value={r.id}>{r.route_name}</option>)}
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
      <Table headers={["Plate no.", "Type", "Driver", "Capacity", "Route", "Status", ""]}>
        {(vehicles as any[] ?? []).map((v: any) => (
          <tr key={v.id} className="hover:bg-slate-50">
            <td className="px-4 py-3 font-mono font-medium">{v.plate_no}</td>
            <td className="px-4 py-3 text-sm capitalize">{v.vehicle_type}</td>
            <td className="px-4 py-3">{v.driver_name} {v.driver_phone && <span className="text-xs text-slate-400">· {v.driver_phone}</span>}</td>
            <td className="px-4 py-3">{v.capacity}</td>
            <td className="px-4 py-3 text-sm text-slate-600">{v.route_name || "—"}</td>
            <td className="px-4 py-3">
              <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${v.status === "active" ? "bg-green-100 text-green-700" : "bg-orange-100 text-orange-700"}`}>
                {v.status}
              </span>
            </td>
            <td className="px-4 py-3">
              <Button variant="ghost" onClick={() => startEditVehicle(v)}>Edit</Button>
            </td>
          </tr>
        ))}
        {(vehicles as any[] ?? []).length === 0 && (
          <tr><td colSpan={7} className="px-4 py-8 text-center text-slate-400">No vehicles registered yet.</td></tr>
        )}
      </Table>
    </div>
  );
}
