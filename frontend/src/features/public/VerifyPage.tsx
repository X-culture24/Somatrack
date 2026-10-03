import { useEffect, useState } from "react";
import { CheckCircle2, ShieldAlert, XCircle } from "lucide-react";
import { api } from "../../api/client";
import { Card, kes } from "../../components/ui";

type Result = {
  valid: boolean;
  reason?: string;
  document?: string;
  school?: string;
  number?: string;
  amount?: number;
  date?: string;
  student_initials?: string;
  status?: string;
};

// Public page behind the QR code on school documents: …/verify#<token>.
// The token rides in the URL fragment, which browsers never send to a
// server, so it stays out of access logs; it's POSTed to the API instead.
export function VerifyPage() {
  const [result, setResult] = useState<Result | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const token = window.location.hash.slice(1);
    if (!token) {
      setError("No code found. Scan the QR code on the document again.");
      return;
    }
    api
      .post<Result>("/verify/", { token })
      .then((r) => setResult(r.data))
      .catch((e) =>
        setError(
          e?.response?.status === 429
            ? "Too many checks from this device. Wait a minute and try again."
            : "Could not reach the school's verification service. Try again shortly.",
        ),
      );
  }, []);

  return (
    <div className="flex min-h-screen items-start justify-center bg-[#f6f1e7] px-4 py-10">
      <div className="w-full max-w-md">
        <p className="mb-1 text-center text-[10px] font-bold tracking-[0.25em] text-[#4a9eca] uppercase">
          ACK St. Mary's School Kabete
        </p>
        <h1 className="mb-6 text-center text-xl font-bold text-[#1b365d]">Document verification</h1>

        <Card>
          {!result && !error && <p className="text-center text-sm text-slate-500">Checking…</p>}

          {error && (
            <div className="flex flex-col items-center gap-2 text-center">
              <ShieldAlert className="text-amber-500" size={40} />
              <p className="text-sm text-slate-600">{error}</p>
            </div>
          )}

          {result && (
            <div className="flex flex-col items-center gap-3 text-center">
              {result.valid ? (
                <CheckCircle2 className="text-emerald-600" size={48} />
              ) : (
                <XCircle className="text-red-600" size={48} />
              )}
              <p className={`text-lg font-bold ${result.valid ? "text-emerald-700" : "text-red-700"}`}>
                {result.valid ? "Genuine document" : "Not valid"}
              </p>
              {result.reason && <p className="text-sm text-slate-600">{result.reason}</p>}

              {result.number && (
                <dl className="mt-2 w-full divide-y divide-slate-100 rounded-lg border border-slate-200 text-left text-sm">
                  {[
                    ["Document", result.document],
                    ["Number", result.number],
                    ["Amount", result.amount !== undefined ? kes(result.amount) : undefined],
                    ["Date", result.date],
                    ["Student", result.student_initials],
                  ]
                    .filter(([, v]) => v)
                    .map(([k, v]) => (
                      <div key={k} className="flex justify-between gap-4 px-3 py-2">
                        <dt className="text-slate-500">{k}</dt>
                        <dd className="font-medium text-slate-800">{v}</dd>
                      </div>
                    ))}
                </dl>
              )}
              <p className="mt-2 text-xs text-slate-400">
                Compare these details with the printed document. If anything differs, contact the school bursar.
              </p>
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}
