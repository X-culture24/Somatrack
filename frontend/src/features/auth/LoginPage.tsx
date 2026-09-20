import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { portalHome, useAuth } from "../../store/auth";
import { Button, Card, Field, Input } from "../../components/ui";

const demos = [
  ["Administrator", "admin@stmaryskabete.ac.ke", "Admin@2026"],
  ["Finance officer", "finance@stmaryskabete.ac.ke", "Finance@2026"],
  ["Class teacher", "teacher@stmaryskabete.ac.ke", "Teacher@2026"],
  ["Parent", "parent@stmaryskabete.ac.ke", "Parent@2026"],
];

export function LoginPage() {
  const login = useAuth((s) => s.login);
  const navigate = useNavigate();
  const [email, setEmail] = useState("admin@stmaryskabete.ac.ke");
  const [password, setPassword] = useState("Admin@2026");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const user = await login(email, password);
      navigate(portalHome(user.portal));
    } catch {
      setError("Could not sign in. Check email, password, and that the API is running.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="grid min-h-screen md:grid-cols-2">
      {/* Left — navy brand panel */}
      <div className="hidden flex-col justify-between bg-[#1b365d] p-10 text-white md:flex">
        <div>
          {/* Logo mark */}
          <div className="mb-6 inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-white/10">
            <span className="text-2xl font-black text-[#4a9eca]">M</span>
          </div>
          <p className="text-xs font-bold tracking-[0.3em] text-[#4a9eca] uppercase mb-3">
            CBC · Playgroup – Grade 9
          </p>
          <h1 className="text-4xl font-bold leading-tight">
            ACK St. Mary's<br />School Kabete
          </h1>
          <p className="mt-4 max-w-sm text-white/70 text-sm leading-relaxed">
            Integrated school management — student records, M-Pesa fee collection,
            CBC academics, and the Nova digital learning campus.
          </p>
          {/* Feature pills */}
          <div className="mt-6 flex flex-wrap gap-2">
            {["Finance & M-Pesa", "Nova LMS", "CBC Academics", "Parent Portal"].map((f) => (
              <span key={f} className="rounded-full bg-white/10 border border-white/20 px-3 py-1 text-xs text-white/80">
                {f}
              </span>
            ))}
          </div>
        </div>
        <p className="text-sm text-white/40">
          P.O BOX 29190-00625, Nairobi · 0746714946
        </p>
      </div>

      {/* Right — login form */}
      <div className="flex items-center justify-center bg-[#f0f4f8] p-6">
        <div className="w-full max-w-md space-y-5">
          {/* Mobile logo */}
          <div className="flex items-center gap-3 md:hidden">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[#1b365d]">
              <span className="text-lg font-black text-[#4a9eca]">M</span>
            </div>
            <div>
              <p className="text-xs font-bold tracking-widest text-[#4a9eca] uppercase">ACK St. Mary's</p>
              <p className="text-sm font-bold text-[#1b365d]">Kabete SMS</p>
            </div>
          </div>

          <Card>
            <h2 className="text-xl font-bold text-[#1b365d]">Sign in</h2>
            <p className="mt-1 text-sm text-slate-500">Use your school account. Portals open by role.</p>

            <form className="mt-5 space-y-4" onSubmit={onSubmit}>
              <Field label="Email">
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  placeholder="you@stmaryskabete.ac.ke"
                />
              </Field>
              <Field label="Password">
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </Field>
              {error && (
                <p className="rounded-lg bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">
                  {error}
                </p>
              )}
              <Button type="submit" disabled={busy} className="w-full justify-center">
                {busy ? "Signing in…" : "Enter portal →"}
              </Button>
            </form>
          </Card>

          {/* Demo accounts */}
          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Demo accounts</p>
            <div className="space-y-1.5">
              {demos.map(([label, mail, pw]) => (
                <button
                  key={mail}
                  type="button"
                  className="flex w-full items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-left text-sm transition hover:border-[#4a9eca] hover:bg-[#4a9eca]/5"
                  onClick={() => { setEmail(mail); setPassword(pw); }}
                >
                  <span className="font-semibold text-[#1b365d]">{label}</span>
                  <span className="text-xs text-slate-400">{mail.split("@")[0]}</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
