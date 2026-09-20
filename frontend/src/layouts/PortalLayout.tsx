import { useState } from "react";
import { NavLink, Navigate, Outlet, useNavigate } from "react-router-dom";
import { BookOpen, GraduationCap, Landmark, LayoutDashboard, LogOut, Menu, Users, X } from "lucide-react";
import { useAuth, type User } from "../store/auth";
import { openNova } from "../api/nova";
import { cn } from "../components/ui";
import type { ReactNode } from "react";

const nav: Record<string, { to: string; label: string; section?: string }[]> = {
  admin: [
    { to: "/admin", label: "Overview" },
    // Students
    { to: "/admin/students", label: "Students" },
    { to: "/admin/admissions", label: "Admissions" },
    // School
    { to: "/admin/users", label: "Users" },
    { to: "/admin/classes", label: "Classes" },
    { to: "/admin/school-setup", label: "School setup" },
    { to: "/admin/staff", label: "Staff" },
    { to: "/admin/payroll", label: "Payroll" },
    // Academic
    { to: "/admin/curriculum", label: "Curriculum" },
    { to: "/admin/grade-requirements", label: "Grade requirements" },
    { to: "/admin/announcements", label: "Announcements" },
    // Operations
    { to: "/admin/transport", label: "Transport" },
    { to: "/admin/library", label: "Library" },
    { to: "/admin/inventory", label: "Inventory" },
    { to: "/admin/reports", label: "Reports" },
    { to: "/nova", label: "Nova LMS" },
  ],
  finance: [
    { to: "/finance", label: "Overview" },
    { to: "/finance/invoices", label: "Invoices" },
    { to: "/finance/payments", label: "Payments" },
    { to: "/finance/unreconciled", label: "Unreconciled" },
    { to: "/finance/fees", label: "Fee structures" },
    { to: "/finance/mpesa-test", label: "M-Pesa test" },
  ],
  teacher: [
    { to: "/teacher", label: "Overview" },
    { to: "/teacher/attendance", label: "Attendance" },
    { to: "/teacher/marks", label: "Marks & gradebook" },
    { to: "/teacher/lesson-plans", label: "Lesson plans" },
    { to: "/teacher/timetable", label: "Timetable" },
    { to: "/nova", label: "Nova LMS" },
  ],
  parent: [
    { to: "/parent", label: "Family" },
    { to: "/parent/fees", label: "Fees & receipts" },
    { to: "/parent/progress", label: "Progress" },
    { to: "/nova", label: "Nova LMS" },
  ],
  student: [
    { to: "/student", label: "Home" },
    { to: "/nova", label: "Nova campus" },
    { to: "/nova/my-work", label: "My work" },
  ],
};

function PortalIcon({ portal }: { portal: User["portal"] }) {
  if (portal === "finance") return <Landmark size={14} />;
  if (portal === "teacher") return <GraduationCap size={14} />;
  if (portal === "parent") return <Users size={14} />;
  return <LayoutDashboard size={14} />;
}

function Sidebar({ items, user, onSignOut, onNova, onClose }: {
  items: { to: string; label: string; section?: string }[];
  user: User | null;
  onSignOut: () => void;
  onNova: () => void;
  onClose?: () => void;
}) {
  return (
    <div className="flex h-full flex-col bg-[#1b365d] text-white">
      {/* Brand */}
      <div className="border-b border-white/10 px-5 py-5">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-[10px] font-bold tracking-[0.25em] text-[#4a9eca] uppercase">ACK St. Mary's</p>
            <p className="mt-0.5 text-base font-bold">Kabete SMS</p>
          </div>
          {onClose && (
            <button className="rounded-lg p-1 text-white/60 hover:bg-white/10" onClick={onClose}>
              <X size={18} />
            </button>
          )}
        </div>
        {/* Portal badge */}
        <div className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-[#4a9eca]/20 px-3 py-1 text-xs text-[#4a9eca] font-medium border border-[#4a9eca]/30">
          <PortalIcon portal={user?.portal ?? "admin"} />
          {user?.portal} portal
        </div>
      </div>

      {/* Nav links */}
      <nav className="flex-1 space-y-0.5 overflow-y-auto p-3">
        {items.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to.split("/").length <= 2}
            onClick={(event) => {
              if (item.to.startsWith("/nova")) { event.preventDefault(); onNova(); onClose?.(); return; }
              onClose?.();
            }}
            className={({ isActive }) =>
              cn(
                "block rounded-lg px-3 py-2 text-sm transition",
                isActive
                  ? "bg-[#4a9eca] text-white font-semibold"
                  : "text-white/75 hover:bg-white/10 hover:text-white",
              )
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>

      {/* User + sign out */}
      <div className="border-t border-white/10 p-3">
        <div className="mb-2 px-2">
          <p className="text-sm font-medium text-white">{user?.full_name}</p>
          <p className="text-xs text-white/50">{user?.email}</p>
        </div>
        <button
          className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-white/70 hover:bg-white/10 hover:text-white transition"
          onClick={onSignOut}
        >
          <LogOut size={16} /> Sign out
        </button>
      </div>
    </div>
  );
}

export function PortalLayout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [mobileOpen, setMobileOpen] = useState(false);
  const items = nav[user?.portal ?? "admin"] ?? nav.admin;

  function handleSignOut() {
    logout();
    navigate("/login");
  }

  async function handleOpenNova() {
    try {
      await openNova();
    } catch {
      // The user stays in the Portal and can retry; credentials are never put
      // in a navigation URL.
      window.alert("NOVA could not be opened. Please sign in again and retry.");
    }
  }

  return (
    <div className="min-h-screen bg-[#f0f4f8]">
      {/* Desktop sidebar */}
      <aside className="fixed inset-y-0 left-0 hidden w-64 md:block">
        <Sidebar items={items} user={user ?? null} onSignOut={handleSignOut} onNova={handleOpenNova} />
      </aside>

      {/* Mobile sidebar overlay */}
      {mobileOpen && (
        <div className="fixed inset-0 z-40 md:hidden">
          <div className="absolute inset-0 bg-black/50" onClick={() => setMobileOpen(false)} />
          <aside className="absolute inset-y-0 left-0 w-72 z-50">
            <Sidebar items={items} user={user ?? null} onSignOut={handleSignOut} onNova={handleOpenNova} onClose={() => setMobileOpen(false)} />
          </aside>
        </div>
      )}

      {/* Main content */}
      <div className="md:pl-64">
        {/* Top header */}
        <header className="sticky top-0 z-30 flex items-center justify-between border-b border-slate-200 bg-white px-4 py-3 shadow-sm md:px-6">
          <div className="flex items-center gap-3">
            {/* Mobile menu button */}
            <button
              className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 md:hidden"
              onClick={() => setMobileOpen(true)}
            >
              <Menu size={20} />
            </button>
            <div className="flex items-center gap-2 text-[#1b365d]">
              <BookOpen size={18} className="text-[#4a9eca]" />
              <span className="font-semibold text-sm">Nova · School Management</span>
            </div>
          </div>
          <div className="hidden sm:block text-right text-sm">
            <p className="font-semibold text-[#1b365d]">{user?.full_name}</p>
            <p className="text-xs text-slate-400">{user?.email}</p>
          </div>
        </header>

        <main className="p-4 md:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const user = useAuth((s) => s.user);
  if (!user) return <Navigate to="/login" replace />;
  return children;
}
