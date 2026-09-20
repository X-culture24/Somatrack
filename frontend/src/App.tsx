import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { PortalLayout, RequireAuth } from "./layouts/PortalLayout";

// Auth
import { LoginPage } from "./features/auth/LoginPage";

// Admin
import { AdminHome } from "./features/admin/AdminHome";
import { StudentsPage } from "./features/admin/StudentsPage";
import { StudentDetailPage } from "./features/admin/StudentDetailPage";
import { AdmissionsPage } from "./features/admin/AdmissionsPage";
import { UsersPage } from "./features/admin/UsersPage";
import { ClassesPage } from "./features/admin/ClassesPage";
import { SchoolSetupPage, TimetablePage } from "./features/admin/SchoolSetupPage";
import { StaffPage } from "./features/admin/StaffPage";
import { InventoryPage } from "./features/admin/InventoryPage";
import { CurriculumPage } from "./features/admin/CurriculumPage";
import { AnnouncementsPage } from "./features/admin/AnnouncementsPage";
import { GradeRequirementsPage } from "./features/admin/GradeRequirementsPage";
import { LibraryManagementPage } from "./features/admin/LibraryManagementPage";
import { TransportManagementPage } from "./features/admin/TransportManagementPage";
import { ReportsPage } from "./features/admin/ReportsPage";

// Finance
import { FinanceHome } from "./features/finance/FinanceHome";
import { InvoicesPage } from "./features/finance/InvoicesPage";
import { PaymentsPage } from "./features/finance/PaymentsPage";
import { UnreconciledPage } from "./features/finance/UnreconciledPage";
import { FeeStructuresPage } from "./features/finance/FeeStructuresPage";
import { MpesaTestPage } from "./features/finance/MpesaTestPage";

// Teacher
import { TeacherHome } from "./features/teacher/TeacherHome";
import { AttendancePage } from "./features/teacher/AttendancePage";
import { MarksEntryPage } from "./features/teacher/MarksEntryPage";
import { LessonPlanPage } from "./features/teacher/LessonPlanPage";

// Staff
import { PayrollPage } from "./features/staff/PayrollPage";

// Parent (PWA shell)
import { ParentShell } from "./features/parent/ParentShell";
import { ParentHome } from "./features/parent/ParentHome";
import { ParentFeesPage } from "./features/parent/ParentFeesPage";
import { ParentProgressPage } from "./features/parent/ParentProgressPage";

// Student
import { StudentHome } from "./features/student/StudentHome";

// Nova LMS
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
import { NovaApp } from "./NovaApp";

const client = new QueryClient();

export function PortalApp() {
  return (
    <QueryClientProvider client={client}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />

          {/* Parent portal — mobile-first PWA shell */}
          <Route element={<RequireAuth><ParentShell /></RequireAuth>}>
            <Route path="/parent" element={<ParentHome />} />
            <Route path="/parent/fees" element={<ParentFeesPage />} />
            <Route path="/parent/progress" element={<ParentProgressPage />} />
          </Route>

          {/* Main portal */}
          <Route element={<RequireAuth><PortalLayout /></RequireAuth>}>

            {/* Student */}
            <Route path="/student" element={<StudentHome />} />

            {/* Admin */}
            <Route path="/admin" element={<AdminHome />} />
            <Route path="/admin/students" element={<StudentsPage />} />
            <Route path="/admin/students/:id" element={<StudentDetailPage />} />
            <Route path="/admin/admissions" element={<AdmissionsPage />} />
            <Route path="/admin/users" element={<UsersPage />} />
            <Route path="/admin/classes" element={<ClassesPage />} />
            <Route path="/admin/school-setup" element={<SchoolSetupPage />} />
            <Route path="/admin/staff" element={<StaffPage />} />
            <Route path="/admin/announcements" element={<AnnouncementsPage />} />
            <Route path="/admin/transport" element={<TransportManagementPage />} />
            <Route path="/admin/curriculum" element={<CurriculumPage />} />
            <Route path="/admin/grade-requirements" element={<GradeRequirementsPage />} />
            <Route path="/admin/library" element={<LibraryManagementPage />} />
            <Route path="/admin/inventory" element={<InventoryPage />} />
            <Route path="/admin/reports" element={<ReportsPage />} />
            <Route path="/admin/payroll" element={<PayrollPage />} />

            {/* Finance */}
            <Route path="/finance" element={<FinanceHome />} />
            <Route path="/finance/invoices" element={<InvoicesPage />} />
            <Route path="/finance/payments" element={<PaymentsPage />} />
            <Route path="/finance/unreconciled" element={<UnreconciledPage />} />
            <Route path="/finance/fees" element={<FeeStructuresPage />} />
            <Route path="/finance/mpesa-test" element={<MpesaTestPage />} />

            {/* Teacher */}
            <Route path="/teacher" element={<TeacherHome />} />
            <Route path="/teacher/attendance" element={<AttendancePage />} />
            <Route path="/teacher/marks" element={<MarksEntryPage />} />
            <Route path="/teacher/lesson-plans" element={<LessonPlanPage />} />
            <Route path="/teacher/timetable" element={<TimetablePage />} />

            {/* Nova LMS — shared across teacher, student, admin */}
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

          </Route>

          <Route path="/" element={<Navigate to="/login" replace />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default function App() {
  return import.meta.env.VITE_APP_SURFACE === "nova" ? <NovaApp /> : <PortalApp />;
}
