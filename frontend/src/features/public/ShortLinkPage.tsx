import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../../api/client";
import { Card } from "../../components/ui";
import { useAuth } from "../../store/auth";

// Survives the trip through /login when a parent taps an SMS link while
// signed out. sessionStorage: gone when the tab closes, never sent anywhere.
export const PENDING_LINK_KEY = "stmarys-pending-link";

// Landing page for SMS links: …/r#<code>. The code is in the fragment so it
// never reaches server logs, and resolving it needs a login — a forwarded
// SMS on its own opens nothing.
export function ShortLinkPage() {
  const user = useAuth((s) => s.user);
  const navigate = useNavigate();
  const [error, setError] = useState("");

  useEffect(() => {
    const fromHash = window.location.hash.slice(1);
    if (fromHash) {
      sessionStorage.setItem(PENDING_LINK_KEY, fromHash);
      // Drop the code from the address bar and browser history.
      window.history.replaceState(null, "", window.location.pathname);
    }
    const code = sessionStorage.getItem(PENDING_LINK_KEY);
    if (!code) {
      setError("This link is incomplete. Open it again from the SMS.");
      return;
    }
    if (!user) {
      navigate("/login", { replace: true });
      return;
    }
    api
      .post<{ target: string }>("/links/resolve/", { code })
      .then((r) => {
        sessionStorage.removeItem(PENDING_LINK_KEY);
        navigate(r.data.target, { replace: true });
      })
      .catch((e) => {
        sessionStorage.removeItem(PENDING_LINK_KEY);
        const status = e?.response?.status;
        setError(
          status === 403
            ? "This link belongs to a different parent account. Sign in with the account the school has on record."
            : status === 404
              ? "This link has expired or is no longer valid. Your receipts are always under Fees in the portal."
              : "Could not open this link. Try again shortly.",
        );
      });
  }, [user, navigate]);

  return (
    <div className="flex min-h-screen items-start justify-center bg-[#f6f1e7] px-4 py-10">
      <Card className="w-full max-w-md text-center">
        {error ? (
          <>
            <p className="text-sm text-slate-600">{error}</p>
            <button className="mt-4 text-sm font-semibold text-[#1b365d] underline" onClick={() => navigate("/login")}>
              Go to the portal
            </button>
          </>
        ) : (
          <p className="text-sm text-slate-500">Opening…</p>
        )}
      </Card>
    </div>
  );
}
