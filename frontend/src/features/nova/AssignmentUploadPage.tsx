import { useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { FileText, Paperclip, Upload, X } from "lucide-react";
import { novaApi } from "../../api/nova";
import { Button, Card, useConfirm } from "../../components/ui";

const MAX_MB = 10;

export function AssignmentUploadPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { confirm, dialog } = useConfirm();
  const fileRef = useRef<HTMLInputElement>(null);

  const { data: assignment, isLoading } = useQuery({
    queryKey: ["assignment", id],
    queryFn: () => novaApi.assignment(id!),
  });

  const [file, setFile] = useState<File | null>(null);
  const [text, setText] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const isLate = assignment?.due_at && new Date() > new Date(assignment.due_at);

  function handleFilePick(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    if (!f) return;
    if (f.size > MAX_MB * 1024 * 1024) {
      setError(`File too large. Maximum is ${MAX_MB} MB.`);
      return;
    }
    setError("");
    setFile(f);
  }

  async function handleSubmit() {
    if (!file && !text.trim()) {
      setError("Please attach a file or write a text response.");
      return;
    }
    const ok = await confirm({
      title: "Submit assignment",
      message: isLate
        ? "This submission is past the due date and will be marked as late. Submit anyway?"
        : `Submit "${assignment?.title}" now? You cannot resubmit after this.`,
      confirmLabel: "Submit",
      variant: isLate ? "danger" : "primary",
    });
    if (!ok) return;

    setSubmitting(true);
    setError("");
    try {
      const fd = new FormData();
      fd.append("assignment", id!);
      if (file) fd.append("file", file);
      if (text.trim()) fd.append("text_response", text.trim());

      const result = await novaApi.submitAssignment(fd);
      navigate(`/nova/assignment/${id}/submitted`, { state: { ...result, assignment } });
    } catch (err: any) {
      setError(err?.response?.data?.detail ?? "Submission failed. Please try again.");
      setSubmitting(false);
    }
  }

  if (isLoading) return <div className="p-8 text-center text-slate-400">Loading…</div>;
  if (!assignment) return null;

  return (
    <div className="mx-auto max-w-xl space-y-5 p-6">
      {dialog}

      {/* Header */}
      <div className="rounded-2xl bg-[#1b365d] p-5 text-white">
        <p className="text-xs tracking-widest text-[#4a9eca] uppercase">Assignment / CAT</p>
        <h1 className="mt-1 text-lg font-semibold">{assignment.title}</h1>
        <div className="mt-2 flex gap-4 text-sm text-white/70">
          <span>Due: {new Date(assignment.due_at).toLocaleString()}</span>
          <span>Max: {assignment.max_score} marks</span>
        </div>
        {isLate && (
          <span className="mt-2 inline-block rounded-full bg-red-500 px-2 py-0.5 text-xs font-semibold">
            Past due — late submission
          </span>
        )}
      </div>

      {/* Instructions */}
      {assignment.instructions && (
        <Card>
          <h3 className="mb-1 font-semibold text-[#1b365d]">Instructions</h3>
          <p className="text-sm text-slate-600 whitespace-pre-line">{assignment.instructions}</p>
        </Card>
      )}

      {/* Text response */}
      <Card>
        <h3 className="mb-2 font-semibold text-[#1b365d]">Written response (optional)</h3>
        <textarea
          rows={5}
          className="w-full rounded-xl border border-slate-300 px-3 py-2 text-sm outline-none focus:ring-2 ring-[#4a9eca] resize-none"
          placeholder="Type your answer here…"
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
      </Card>

      {/* File upload */}
      <Card>
        <h3 className="mb-2 font-semibold text-[#1b365d]">Attach file (PDF, image, Word)</h3>
        <input
          ref={fileRef}
          type="file"
          accept=".pdf,.doc,.docx,.png,.jpg,.jpeg"
          className="hidden"
          onChange={handleFilePick}
        />
        {!file ? (
          <button
            onClick={() => fileRef.current?.click()}
            className="flex w-full flex-col items-center gap-2 rounded-xl border-2 border-dashed border-slate-300 py-8 text-slate-400 hover:border-[#1b365d] hover:text-[#1b365d] transition"
          >
            <Upload size={28} />
            <p className="text-sm">Click to select file</p>
            <p className="text-xs">Max {MAX_MB} MB · PDF, Word, PNG, JPG</p>
          </button>
        ) : (
          <div className="flex items-center justify-between rounded-xl border border-[#4a9eca] bg-blue-50 px-4 py-3">
            <div className="flex items-center gap-3">
              <Paperclip size={18} className="text-[#4a9eca]" />
              <div>
                <p className="text-sm font-medium text-slate-800">{file.name}</p>
                <p className="text-xs text-slate-400">{(file.size / 1024 / 1024).toFixed(2)} MB</p>
              </div>
            </div>
            <button onClick={() => setFile(null)} className="text-slate-400 hover:text-red-500">
              <X size={18} />
            </button>
          </div>
        )}
      </Card>

      {error && <p className="rounded-xl bg-red-50 px-4 py-2 text-sm text-red-700">{error}</p>}

      <Button
        className="w-full"
        onClick={handleSubmit}
        disabled={submitting || (!file && !text.trim())}
      >
        <FileText size={16} />
        {submitting ? "Submitting…" : "Submit assignment"}
      </Button>
    </div>
  );
}
