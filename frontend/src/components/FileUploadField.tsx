import { useState } from "react";
import { type UploadCategory, uploadsApi } from "../api/uploads";
import { Field } from "./ui";

// A file input that uploads immediately on selection and stores the
// server-returned path in the parent form's state. Kept generic so it
// covers every registration document (student, guardian and staff).
export function FileUploadField({
  label, category, value, onUploaded,
}: { label: string; category: UploadCategory; value: string; onUploaded: (path: string) => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      const { path } = await uploadsApi.upload(category, file);
      onUploaded(path);
    } catch {
      setError("Upload failed — must be a JPEG, PNG, or PDF under 10MB.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Field label={label}>
      <input
        type="file"
        accept="image/jpeg,image/png,application/pdf"
        onChange={handleChange}
        disabled={busy}
        className="w-full text-sm text-slate-600 file:mr-3 file:rounded-lg file:border-0 file:bg-[#1b365d] file:px-3 file:py-1.5 file:text-sm file:font-semibold file:text-white hover:file:bg-[#152a49]"
      />
      {busy && <p className="mt-1 text-xs text-slate-400">Uploading…</p>}
      {!busy && value && <p className="mt-1 text-xs text-green-600">✓ Uploaded</p>}
      {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
    </Field>
  );
}
