import { useState } from "react";
import { NavLink, Navigate, Outlet, useNavigate } from "react-router-dom";
import {
  BookOpen, CreditCard, Users, User, Settings, Building2,
  HelpCircle, Home, Menu, LogOut, ChevronDown, X,
  Sparkles, GraduationCap, Wallet, UsersRound, Rocket,
  ExternalLink,
} from "lucide-react";
import { useAuth, type User as AuthUser } from "../store/auth";
import { openNova } from "../api/nova";
import { BorderBeam } from "../components/ui/border-beam";
import { cn } from "../lib/utils";
import type { ReactNode } from "react";

// ── Nav structure ─────────────────────────────────────────────────────────────

type NavItem    = { to: string; label: string; nova?: true };
type NavSection = { label: string; icon: React.ElementType; items: NavItem[] };
type PortalNav  = { home: string; homeLabel: string; sections: NavSection[] };

const portalNav: Record<string, PortalNav> = {
  admin: {
    home: "/admin", homeLabel: "Home",
    sections: [
      { label: "Students", icon: Users, items: [
        { to: "/admin/students",    label: "All Students" },
        { to: "/admin/admissions",  label: "Admissions" },
        { to: "/admin/promote",     label: "Move to Next Class" },
        { to: "/admin/siblings",    label: "Siblings" },
        { to: "/admin/alumni",      label: "Alumni Students" },
      ]},
      { label: "School", icon: Building2, items: [
        { to: "/admin/users",       label: "Users" },
        { to: "/admin/classes",     label: "Classes" },
        { to: "/admin/staff",       label: "Staff" },
        { to: "/admin/school-setup",label: "School setup" },
      ]},
      { label: "Finance", icon: CreditCard, items: [
        { to: "/admin/fee-arrears", label: "Fee Arrears" },
        { to: "/admin/payroll",     label: "Payroll" },
        { to: "/admin/reports",     label: "Reports" },
        { to: "/finance",           label: "Finance desk" },
      ]},
      { label: "Academic", icon: BookOpen, items: [
        { to: "/admin/curriculum",          label: "Curriculum" },
        { to: "/admin/grade-requirements",  label: "Grade requirements" },
        { to: "/admin/exam-management",     label: "Exam Management" },
        { to: "/admin/results",             label: "Results & mark list" },
        { to: "/admin/teacher-assignment",  label: "Teacher Assignment" },
        { to: "/admin/lesson-plans",        label: "Lesson plans (review)" },
        { to: "/admin/headteacher",         label: "CBC Reports (HT)" },
        { to: "/teacher/marks",             label: "Marks entry" },
        { to: "/admin/announcements",       label: "Announcements" },
      ]},
      { label: "Operations", icon: Settings, items: [
        { to: "/admin/transport",  label: "Transport" },
        { to: "/admin/library",    label: "Library" },
        { to: "/admin/inventory",  label: "Inventory" },
        { to: "/records",          label: "School records" },
        { to: "/nova",             label: "Nova LMS", nova: true },
      ]},
      { label: "Help", icon: HelpCircle, items: [
        { to: "/admin/help",       label: "Help & support" },
      ]},
    ],
  },

  finance: {
    home: "/finance", homeLabel: "Home",
    sections: [
      { label: "Finance", icon: CreditCard, items: [
        { to: "/finance/invoices",     label: "Invoices" },
        { to: "/finance/payments",     label: "Payments" },
        { to: "/finance/ledger",       label: "Student Ledger" },
        { to: "/admin/fee-arrears",    label: "Fee Arrears" },
        { to: "/finance/unreconciled", label: "Unreconciled" },
        { to: "/finance/fees",         label: "Fee structures" },
        { to: "/finance/stk-push",     label: "STK Push / Ad-hoc" },
        { to: "/finance/mpesa-test",   label: "Simulate payment" },
      ]},
      { label: "Self Service", icon: User, items: [
        { to: "/records",              label: "School records" },
      ]},
      { label: "FAQ", icon: HelpCircle, items: [
        { to: "/finance/help",         label: "Help & support" },
      ]},
    ],
  },

  teacher: {
    home: "/teacher", homeLabel: "Home",
    sections: [
      { label: "Academics", icon: BookOpen, items: [
        { to: "/teacher/attendance",     label: "Attendance" },
        { to: "/teacher/marks",          label: "Marks & gradebook" },
        { to: "/teacher/cbc-marksheet",  label: "CBC marksheet" },
        { to: "/teacher/lesson-plans",   label: "Lesson plans" },
        { to: "/admin/results",          label: "Results & rankings" },
      ]},
      { label: "Self Service", icon: User, items: [
        { to: "/records",              label: "School records" },
        { to: "/nova",                 label: "Nova LMS", nova: true },
      ]},
      { label: "FAQ", icon: HelpCircle, items: [
        { to: "/teacher/help",         label: "Help & support" },
      ]},
    ],
  },

  parent: {
    home: "/parent", homeLabel: "Home",
    sections: [
      { label: "Academics", icon: BookOpen, items: [
        { to: "/parent/progress",      label: "Progress & results" },
      ]},
      { label: "Finance", icon: CreditCard, items: [
        { to: "/parent/fees",          label: "Fees & receipts" },
      ]},
      { label: "Self Service", icon: User, items: [
        { to: "/nova",                 label: "Nova campus", nova: true },
      ]},
      { label: "FAQ", icon: HelpCircle, items: [
        { to: "/parent/help",          label: "Help & support" },
      ]},
    ],
  },

  student: {
    home: "/student", homeLabel: "Home",
    sections: [
      { label: "Self Service", icon: User, items: [
        { to: "/nova",          label: "Nova campus",  nova: true },
        { to: "/nova/my-work",  label: "My work",      nova: true },
      ]},
      { label: "FAQ", icon: HelpCircle, items: [
        { to: "/student/help",  label: "Help & support" },
      ]},
    ],
  },
};

// ── Role-tinted accent helpers ─────────────────────────────────────────────────

const ROLE_THEME: Record<string, { accent: string; badge: string; Icon: React.ElementType; label: string }> = {
  admin:   { accent: "#4a9eca", badge: "bg-[#4a9eca]/15 text-[#cfe9f8] border-[#4a9eca]/30", Icon: Rocket,        label: "Admin Portal"   },
  finance: { accent: "#5aa8d6", badge: "bg-[#5aa8d6]/15 text-[#cfe9f8] border-[#5aa8d6]/30", Icon: Wallet,        label: "Finance Portal"  },
  teacher: { accent: "#6fb4dc", badge: "bg-[#6fb4dc]/15 text-[#d9eef9] border-[#6fb4dc]/30", Icon: GraduationCap, label: "Teacher Portal"  },
  parent:  { accent: "#3d8bc0", badge: "bg-[#3d8bc0]/15 text-[#cfe9f8] border-[#3d8bc0]/30", Icon: UsersRound,    label: "Parent Portal"   },
  student: { accent: "#2a7bb3", badge: "bg-[#2a7bb3]/15 text-[#c5e2f5] border-[#2a7bb3]/30", Icon: Sparkles,      label: "Student Portal"  },
};
const SIDEBAR_GRADIENT = "from-[#1b365d] via-[#244a7b] to-[#2c5a96]";

function roleOf(portal?: string) {
  return ROLE_THEME[portal ?? "admin"] ?? ROLE_THEME.admin;
}

function initials(name?: string) {
  if (!name) return "SM";
  const parts = name.trim().split(/\s+/).slice(0, 2);
  return parts.map((p) => p[0]?.toUpperCase() ?? "").join("") || "SM";
}

// ── Sidebar ───────────────────────────────────────────────────────────────────

function Sidebar({
  pnav, user, onSignOut, onNova, onClose,
}: {
  pnav: PortalNav;
  user: AuthUser | null;
  onSignOut: () => void;
  onNova: () => void;
  onClose?: () => void;
}) {
  const [open, setOpen] = useState<Record<string, boolean>>(() =>
    Object.fromEntries(pnav.sections.map((s) => [s.label, true]))
  );
  const toggle = (label: string) =>
    setOpen((prev) => ({ ...prev, [label]: !prev[label] }));

  const theme = roleOf(user?.portal);
  const RoleIcon = theme.Icon;

  return (
    <div className={cn(
      "relative flex h-full flex-col overflow-hidden",
      "bg-gradient-to-b",
      SIDEBAR_GRADIENT,
      "text-white",
    )}>
      {/* decorative blur blobs */}
      <div className="pointer-events-none absolute -top-24 -left-16 h-56 w-56 rounded-full bg-white/8 blur-3xl" />
      <div className="pointer-events-none absolute top-1/3 -right-20 h-60 w-60 rounded-full"
        style={{ backgroundColor: `${theme.accent}1a` }}
      />
      <div className="pointer-events-none absolute bottom-10 -left-10 h-44 w-44 rounded-full bg-white/5 blur-3xl" />

      {/* Brand card */}
      <div className="relative px-4 pt-4 pb-3">
        <div className="relative overflow-hidden rounded-2xl border border-white/18 bg-white/8 p-3 backdrop-blur-sm">
          <BorderBeam
            size={160}
            duration={16}
            delay={1.5}
            borderWidth={1.5}
            colorFrom={theme.accent}
            colorTo="#ffffff"
          />
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white/12 border border-white/22 backdrop-blur-sm">
              <span className="text-lg font-black tracking-tighter" style={{ color: theme.accent }}>M</span>
            </div>
            <div className="min-w-0">
              <p className="text-[10px] font-bold tracking-[0.28em] uppercase text-white/75">
                ACK St. Mary's Kabete
              </p>
              <p className="text-[15px] font-bold leading-tight truncate">{theme.label}</p>
            </div>
            {onClose && (
              <button
                onClick={onClose}
                className="ml-auto rounded-lg p-1.5 text-white/70 hover:text-white hover:bg-white/10 transition-colors"
                aria-label="Close navigation"
              >
                <X size={16} />
              </button>
            )}
          </div>
          <div className="mt-3 flex items-center gap-2">
            <span className={cn(
              "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[10px] font-semibold",
              theme.badge,
            )}>
              <RoleIcon size={11} />
              {user?.portal === "admin"   && "Super-user access"}
              {user?.portal === "finance" && "Finance & billing"}
              {user?.portal === "teacher" && "CBC Academics"}
              {user?.portal === "parent"  && "Parent & guardian"}
              {user?.portal === "student" && "Nova learner"}
              {!user?.portal && "Welcome"}
            </span>
            <span className="text-[10px] text-white/55 font-semibold ml-auto">v1.3</span>
          </div>
        </div>
      </div>

      {/* Scrollable nav */}
      <nav className="relative flex-1 overflow-y-auto px-3 pb-2 pt-1">
        {/* Home link */}
        <NavLink
          to={pnav.home}
          end
          onClick={onClose}
          className={({ isActive }) => cn(
            "group relative flex items-center gap-3 rounded-xl px-3 py-2.5 text-[14px] transition-all duration-200",
            isActive
              ? "bg-white/14 text-white font-semibold shadow-inner ring-1 ring-white/22"
              : "text-white/85 hover:text-white hover:bg-white/9",
          )}
        >
          {({ isActive }) => (
            <>
              <span className={cn(
                "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg transition-colors",
                isActive ? "bg-white/14" : "bg-white/6 group-hover:bg-white/10",
              )}>
                <Home size={16} style={{ color: isActive ? theme.accent : undefined }} strokeWidth={2.25} />
              </span>
              <span className="flex-1 font-medium">{pnav.homeLabel}</span>
              {isActive && (
                <span
                  className="h-2 w-2 shrink-0 rounded-full"
                  style={{ backgroundColor: theme.accent, boxShadow: `0 0 10px ${theme.accent}` }}
                />
              )}
            </>
          )}
        </NavLink>

        {/* Section divider */}
        <div className="my-3 h-px bg-white/12 mx-1" />

        {/* Collapsible sections */}
        {pnav.sections.map((section) => {
          const Icon = section.icon;
          const isExpanded = open[section.label] ?? true;
          return (
            <div key={section.label} className="mb-0.5">
              {/* Section header */}
              <button
                onClick={() => toggle(section.label)}
                className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-[13.5px] transition-colors hover:bg-white/9"
              >
                <div className="flex items-center gap-3">
                  <span className={cn(
                    "flex h-7 w-7 shrink-0 items-center justify-center rounded-lg",
                    isExpanded ? "bg-white/11" : "bg-white/6",
                  )}>
                    <Icon size={14} strokeWidth={2.25} style={{ color: isExpanded ? theme.accent : "rgba(255,255,255,0.7)" }} />
                  </span>
                  <span
                    className={cn(
                      "text-[12.5px] font-bold tracking-wide uppercase",
                      isExpanded ? "text-white" : "text-white/72",
                    )}
                  >
                    {section.label}
                  </span>
                </div>
                <span className={cn(
                  "text-white/55 transition-transform duration-200",
                  isExpanded ? "rotate-0" : "-rotate-90",
                )}>
                  <ChevronDown size={14} strokeWidth={2.25} />
                </span>
              </button>

              {/* Sub-items */}
              <div
                className={cn(
                  "grid overflow-hidden transition-all duration-200 ease-out",
                  isExpanded ? "grid-rows-[1fr] opacity-100 mt-1" : "grid-rows-[0fr] opacity-0",
                )}
              >
                <div className="min-h-0">
                  <div className="ml-3 border-l border-white/14 pl-2 space-y-0.5 py-0.5">
                    {section.items.map((item) => (
                      item.nova ? (
                        <button
                          key={item.to}
                          onClick={(e) => { e.preventDefault(); onNova(); onClose?.(); }}
                          className="group flex w-full items-center justify-between rounded-lg px-3 py-2 text-[13.5px] text-white/80 hover:text-white hover:bg-white/9 transition-colors"
                        >
                          <span className="flex items-center gap-2 font-medium">
                            <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-white/35 group-hover:bg-white/70" />
                            {item.label}
                            <ExternalLink size={11} className="opacity-70" />
                          </span>
                          <span className="text-[9px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-md bg-[#4a9eca]/22 text-[#cfe9f8] border border-[#4a9eca]/30">
                            Nova
                          </span>
                        </button>
                      ) : (
                        <NavLink
                          key={item.to}
                          to={item.to}
                          end={item.to.split("/").length <= 2}
                          onClick={onClose}
                          className={({ isActive }) => cn(
                            "group relative flex items-center gap-2 rounded-lg px-3 py-2 text-[13.5px] transition-all duration-150",
                            isActive
                              ? "text-white font-semibold bg-white/11"
                              : "text-white/82 hover:text-white hover:bg-white/6",
                          )}
                        >
                          {({ isActive }) => (
                            <>
                              <span
                                className={cn(
                                  "h-1.5 w-1.5 shrink-0 rounded-full transition-colors",
                                  isActive ? "" : "bg-white/35 group-hover:bg-white/60",
                                )}
                                style={isActive ? { backgroundColor: theme.accent, boxShadow: `0 0 8px ${theme.accent}` } : undefined}
                              />
                              <span className="flex-1 truncate font-medium">{item.label}</span>
                            </>
                          )}
                        </NavLink>
                      )
                    ))}
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </nav>

      {/* User card footer */}
      <div className="relative px-3 pb-2 pt-2">
        <div className="rounded-2xl border border-white/13 bg-white/6 backdrop-blur-sm p-3">
          <div className="flex items-center gap-2.5">
            <div
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl font-bold text-[13px] border border-white/22"
              style={{ background: `linear-gradient(135deg, ${theme.accent}4d, ${theme.accent}1a)` }}
            >
              {initials(user?.full_name)}
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-[13.5px] font-semibold text-white truncate leading-tight">{user?.full_name ?? "Signed in"}</p>
              <p className="text-[11.5px] text-white/65 truncate leading-tight mt-0.5">{user?.email ?? ""}</p>
            </div>
            <button
              onClick={onSignOut}
              className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-white/70 hover:text-white hover:bg-white/10 transition-colors"
              title="Sign out"
            >
              <LogOut size={15} strokeWidth={2} />
            </button>
          </div>
        </div>
      </div>

      {/* Copyright strip */}
      <div className="relative px-3 pb-3 pt-1 text-center">
        <p className="text-[10.5px] text-white/45 font-semibold tracking-wide">
          {new Date().getFullYear()} &copy; St. Mary's Kabete SMS
        </p>
      </div>
    </div>
  );
}

// ── Layout shell ──────────────────────────────────────────────────────────────

export function PortalLayout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [mobileOpen, setMobileOpen] = useState(false);

  const pnav = portalNav[user?.portal ?? "admin"] ?? portalNav.admin;
  const theme = roleOf(user?.portal);

  function handleSignOut() { logout(); navigate("/login"); }

  async function handleOpenNova() {
    try { await openNova(); }
    catch { window.alert("Nova could not be opened. Please sign in again and retry."); }
  }

  return (
    <div className="min-h-screen flex bg-[#f0f4f8]">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex flex-col w-64 flex-shrink-0 fixed inset-y-0 left-0 z-30 shadow-[0_0_0_1px_rgba(27,54,93,0.06)]">
        <Sidebar pnav={pnav} user={user ?? null} onSignOut={handleSignOut} onNova={handleOpenNova} />
      </aside>

      {/* Mobile sidebar overlay */}
      {mobileOpen && (
        <div className="fixed inset-0 z-40 md:hidden">
          <div className="absolute inset-0 bg-[#1b365d]/40 backdrop-blur-sm" onClick={() => setMobileOpen(false)} />
          <aside className="absolute inset-y-0 left-0 w-64 z-50 shadow-2xl">
            <Sidebar pnav={pnav} user={user ?? null} onSignOut={handleSignOut} onNova={handleOpenNova} onClose={() => setMobileOpen(false)} />
          </aside>
        </div>
      )}

      {/* Main area */}
      <div className="flex-1 flex flex-col md:ml-64 min-w-0">
        {/* Top bar */}
        <header className="sticky top-0 z-20 flex items-center gap-3 px-4 sm:px-5 py-2.5 bg-white/80 backdrop-blur border-b border-slate-200/60">
          <button
            className="rounded-xl p-2 text-slate-600 hover:text-[#1b365d] hover:bg-slate-100 md:hidden transition-colors"
            onClick={() => setMobileOpen(true)}
            aria-label="Open navigation"
          >
            <Menu size={18} strokeWidth={2} />
          </button>
          <button
            className="hidden md:flex rounded-xl p-2 text-slate-400 hover:text-[#1b365d] hover:bg-slate-100 transition-colors"
            onClick={() => setMobileOpen(true)}
            aria-label="Toggle navigation"
          >
            <Menu size={17} strokeWidth={2} />
          </button>

          {/* breadcrumb / term pill */}
          <div className="hidden sm:flex items-center gap-2 rounded-full bg-slate-100 px-3 py-1 border border-slate-200/70">
            <span
              className="h-2 w-2 rounded-full animate-pulse"
              style={{ backgroundColor: theme.accent }}
            />
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-500">
              {theme.label.replace(" Portal", "")}
            </span>
          </div>

          <div className="ml-auto flex items-center gap-2 sm:gap-3">
            <span className="hidden md:block text-xs text-slate-500 font-medium truncate max-w-[200px]">
              {user?.full_name}
            </span>
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl font-bold text-xs text-white border"
              style={{ background: `linear-gradient(135deg, ${theme.accent}, #1b365d)` }}
              title={user?.full_name}
            >
              {initials(user?.full_name)}
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 p-4 md:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

// ── Auth guard ────────────────────────────────────────────────────────────────

export function RequireAuth({ children }: { children: ReactNode }) {
  const user = useAuth((s) => s.user);
  if (!user) return <Navigate to="/login" replace />;
  return children;
}
