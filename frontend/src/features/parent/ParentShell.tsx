import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { BookOpen, CreditCard, Home, LogOut, TrendingUp } from "lucide-react";
import { useAuth } from "../../store/auth";
import { cn } from "../../components/ui";
import { openNova } from "../../api/nova";

const tabs = [
  { to: "/parent", label: "Home", icon: Home },
  { to: "/parent/fees", label: "Fees", icon: CreditCard },
  { to: "/parent/progress", label: "Progress", icon: TrendingUp },
  { to: "/nova", label: "Nova", icon: BookOpen },
];

export function ParentShell() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [installPrompt, setInstallPrompt] = useState<any>(null);
  const [showBanner, setShowBanner] = useState(false);

  useEffect(() => {
    const handler = (e: Event) => {
      e.preventDefault();
      setInstallPrompt(e);
      setShowBanner(true);
    };
    window.addEventListener("beforeinstallprompt", handler);
    return () => window.removeEventListener("beforeinstallprompt", handler);
  }, []);

  async function handleInstall() {
    if (!installPrompt) return;
    installPrompt.prompt();
    await installPrompt.userChoice;
    setInstallPrompt(null);
    setShowBanner(false);
  }

  return (
    <div className="flex min-h-screen flex-col bg-[#f0f4f8]">

      {/* PWA install banner */}
      {showBanner && (
        <div className="flex items-center justify-between bg-[#1b365d] px-4 py-2 text-sm text-white">
          <span>Add St. Mary's to your home screen for quick access.</span>
          <div className="flex gap-3">
            <button className="font-semibold text-[#4a9eca]" onClick={handleInstall}>Install</button>
            <button className="text-white/60" onClick={() => setShowBanner(false)}>✕</button>
          </div>
        </div>
      )}

      {/* Top bar */}
      <header className="sticky top-0 z-10 flex items-center justify-between border-b border-slate-200 bg-white px-4 py-3 shadow-sm">
        <div>
          <p className="text-[10px] font-bold tracking-[0.25em] text-[#4a9eca] uppercase">ACK St. Mary's</p>
          <p className="text-sm font-bold text-[#1b365d]">Parent portal</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="hidden text-sm text-slate-600 sm:block">{user?.full_name}</span>
          <button
            className="rounded-full p-1.5 text-slate-500 hover:bg-slate-100"
            onClick={() => { logout(); navigate("/login"); }}
            title="Sign out"
          >
            <LogOut size={16} />
          </button>
        </div>
      </header>

      {/* Page content */}
      <main className="flex-1 overflow-y-auto pb-20">
        <Outlet />
      </main>

      {/* Bottom tab bar */}
      <nav className="fixed inset-x-0 bottom-0 z-10 flex border-t border-slate-200 bg-white">
        {tabs.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            onClick={(event) => { if (to === "/nova") { event.preventDefault(); openNova(); } }}
            end={to === "/parent"}
            className={({ isActive }) =>
              cn(
                "flex flex-1 flex-col items-center gap-0.5 py-2 text-xs transition",
                isActive ? "text-[#1b365d] font-semibold" : "text-slate-400 hover:text-slate-600",
              )
            }
          >
            {({ isActive }) => (
              <>
                <span className={cn("rounded-xl p-1.5", isActive && "bg-[#4a9eca]/15")}>
                  <Icon size={20} strokeWidth={isActive ? 2.5 : 1.8} color={isActive ? "#1b365d" : undefined} />
                </span>
                {label}
              </>
            )}
          </NavLink>
        ))}
      </nav>
    </div>
  );
}
