import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";
import { useState } from "react";
import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

// ── Colour tokens (school: navy / white / light-blue) ─────────────────────────
// Navy:       #1b365d  — sidebars, headers, primary buttons, headings
// Light blue: #4a9eca  — accents, active states, focus rings, badges
// White:      #ffffff  — card backgrounds, inputs
// Sky bg:     #f0f4f8  — page background, light sections

export function cn(...inputs: Array<string | false | null | undefined>) {
  return twMerge(clsx(inputs));
}

export function Card({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("rounded-2xl border border-slate-200 bg-white p-5 shadow-sm", className)}>
      {children}
    </div>
  );
}

export function Button({
  children,
  className,
  variant = "primary",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "primary" | "ghost" | "accent" | "gold" }) {
  const styles: Record<string, string> = {
    primary: "bg-[#1b365d] text-white hover:bg-[#152a49]",
    accent:  "bg-[#4a9eca] text-white hover:bg-[#3a8bb8]",
    ghost:   "border border-slate-300 bg-white text-slate-700 hover:bg-slate-50",
    // gold kept as alias to accent for backward compat
    gold:    "bg-[#4a9eca] text-white hover:bg-[#3a8bb8]",
  };
  return (
    <button
      className={cn(
        "inline-flex items-center gap-1.5 rounded-lg px-4 py-2 text-sm font-semibold transition disabled:opacity-50",
        styles[variant] ?? styles.primary,
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={cn(
        "w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none",
        "focus:border-[#4a9eca] focus:ring-2 focus:ring-[#4a9eca]/20",
        props.className,
      )}
    />
  );
}

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={cn(
        "w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none",
        "focus:border-[#4a9eca] focus:ring-2 focus:ring-[#4a9eca]/20",
        props.className,
      )}
    />
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block space-y-1 text-sm">
      <span className="font-medium text-slate-600">{label}</span>
      {children}
    </label>
  );
}

export function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <Card>
      <p className="text-xs uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-bold text-[#1b365d]">{value}</p>
    </Card>
  );
}

export function PageHeader({
  title,
  subtitle,
  actions,
}: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="text-2xl font-semibold text-[#1b365d]">{title}</h1>
        {subtitle ? <p className="mt-1 text-sm text-slate-500">{subtitle}</p> : null}
      </div>
      {actions}
    </div>
  );
}

export function Table({ headers, children }: { headers: string[]; children: ReactNode }) {
  return (
    <div className="overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm">
      <table className="min-w-full text-left text-sm">
        <thead>
          <tr className="bg-[#1b365d] text-white">
            {headers.map((h) => (
              <th key={h} className="px-4 py-3 font-medium tracking-wide">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">{children}</tbody>
      </table>
    </div>
  );
}

export function Badge({
  children,
  variant = "default",
}: {
  children: ReactNode;
  variant?: "default" | "success" | "warning" | "danger" | "blue" | "navy";
}) {
  const styles: Record<string, string> = {
    default: "bg-slate-100 text-slate-600",
    success: "bg-green-100 text-green-700",
    warning: "bg-yellow-100 text-yellow-700",
    danger:  "bg-red-100 text-red-700",
    blue:    "bg-[#4a9eca]/15 text-[#1b6a9c]",
    navy:    "bg-[#1b365d] text-white",
  };
  return (
    <span className={cn("rounded-full px-2 py-0.5 text-xs font-medium", styles[variant])}>
      {children}
    </span>
  );
}

export function kes(value: string | number | undefined | null) {
  const n = Number(value ?? 0);
  return `KES ${n.toLocaleString("en-KE")}`;
}

// ── Confirm Dialog ─────────────────────────────────────────────────────────────

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "danger" | "primary";
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({
  open, title, message,
  confirmLabel = "Confirm", cancelLabel = "Cancel",
  variant = "primary", onConfirm, onCancel,
}: ConfirmDialogProps) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl">
        <h3 className="text-base font-semibold text-[#1b365d]">{title}</h3>
        <p className="mt-2 text-sm text-slate-600">{message}</p>
        <div className="mt-5 flex justify-end gap-3">
          <button
            className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50"
            onClick={onCancel}
          >
            {cancelLabel}
          </button>
          <button
            className={cn(
              "rounded-lg px-4 py-2 text-sm font-semibold text-white transition",
              variant === "danger"
                ? "bg-red-600 hover:bg-red-700"
                : "bg-[#1b365d] hover:bg-[#152a49]",
            )}
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── useConfirm hook ────────────────────────────────────────────────────────────

type ConfirmOptions = {
  title: string;
  message: string;
  confirmLabel?: string;
  variant?: "danger" | "primary";
};

export function useConfirm() {
  const [state, setState] = useState<(ConfirmOptions & { resolve: (v: boolean) => void }) | null>(null);

  function confirm(opts: ConfirmOptions): Promise<boolean> {
    return new Promise((resolve) => setState({ ...opts, resolve }));
  }

  function handleConfirm() { state?.resolve(true); setState(null); }
  function handleCancel()  { state?.resolve(false); setState(null); }

  const dialog = state ? (
    <ConfirmDialog
      open
      title={state.title}
      message={state.message}
      confirmLabel={state.confirmLabel}
      variant={state.variant}
      onConfirm={handleConfirm}
      onCancel={handleCancel}
    />
  ) : null;

  return { confirm, dialog };
}
