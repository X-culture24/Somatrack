import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { BrowserRouter, Navigate, Route, Routes, useNavigate, useSearchParams } from "react-router-dom";
import { api } from "./api/client";
import { RequireAuth } from "./layouts/PortalLayout";
import { useAuth } from "./store/auth";
import { NovaHome } from "./features/nova/NovaHome";
import { NovaCoursePage } from "./features/nova/NovaCoursePage";
import { QuizInstructionsPage } from "./features/nova/QuizInstructionsPage";
import { QuizAttemptPage } from "./features/nova/QuizAttemptPage";
import { QuizResultPage } from "./features/nova/QuizResultPage";
import { AssignmentUploadPage } from "./features/nova/AssignmentUploadPage";
import { SubmissionConfirmedPage } from "./features/nova/SubmissionConfirmedPage";
import { MyWorkPage } from "./features/nova/MyWorkPage";
import { TeacherNovaPage } from "./features/nova/TeacherNovaPage";
import { GradeSubmissionsPage } from "./features/nova/GradeSubmissionsPage";

const client = new QueryClient();

function Callback() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState("");
  const code = params.get("code");
  useEffect(() => {
    if (!code) { setError("The NOVA sign-in link is missing."); return; }
    api.post("/auth/nova/exchange/", { code })
      .then(({ data }) => { useAuth.setState({ access: data.access, refresh: data.refresh, user: data.user }); navigate("/nova", { replace: true }); })
      .catch(() => setError("The NOVA sign-in link is invalid or expired. Return to the Portal and try again."));
  }, [code, navigate]);
  return <div className="p-8 text-center text-slate-600">{error || "Opening NOVA securely…"}</div>;
}

export function NovaApp() {
  return <QueryClientProvider client={client}><BrowserRouter><Routes>
    <Route path="/auth/callback" element={<Callback />} />
    <Route element={<RequireAuth><div className="min-h-screen bg-[#f0f4f8]"><main className="p-4 md:p-6"><Routes>
      <Route path="/nova" element={<NovaHome />} />
      <Route path="/nova/courses/:id" element={<NovaCoursePage />} />
      <Route path="/nova/quiz/:id/instructions" element={<QuizInstructionsPage />} />
      <Route path="/nova/quiz/:id/attempt" element={<QuizAttemptPage />} />
      <Route path="/nova/quiz/:id/result" element={<QuizResultPage />} />
      <Route path="/nova/assignment/:id" element={<AssignmentUploadPage />} />
      <Route path="/nova/assignment/:id/submitted" element={<SubmissionConfirmedPage />} />
      <Route path="/nova/my-work" element={<MyWorkPage />} />
      <Route path="/nova/teacher/:id" element={<TeacherNovaPage />} />
      <Route path="/nova/teacher/:id/submissions" element={<GradeSubmissionsPage />} />
      <Route path="*" element={<Navigate to="/nova" replace />} />
    </Routes></main></div></RequireAuth>} />
    <Route path="*" element={<Navigate to="/auth/callback" replace />} />
  </Routes></BrowserRouter></QueryClientProvider>;
}
