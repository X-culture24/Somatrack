import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { PortalLayout, RequireAuth } from "./layouts/PortalLayout";

// UI demo / public landing
import { DemoOne } from "./components/ui/demo";

// Auth
import { LoginPage } from "./features/auth/LoginPage";

// Public — reached from QR codes and SMS links
import { VerifyPage } from "./features/public/VerifyPage";
import { ShortLinkPage } from "./features/public/ShortLinkPage";

// Admin
import { AdminHome } from "./features/admin/AdminHome";
import { StudentsPage } from "./features/admin/StudentsPage";
import { StudentDetailPage } from "./features/admin/StudentDetailPage";
import { AdmissionsPage } from "./features/admin/AdmissionsPage";
import { UsersPage } from "./features/admin/UsersPage";
import { ClassesPage } from "./features/admin/ClassesPage";
import { SchoolSetupPage } from "./features/admin/SchoolSetupPage";
import { StaffPage } from "./features/admin/StaffPage";
import { InventoryPage } from "./features/admin/InventoryPage";
import { CurriculumPage } from "./features/admin/CurriculumPage";
import { AnnouncementsPage } from "./features/admin/AnnouncementsPage";
import { GradeRequirementsPage } from "./features/admin/GradeRequirementsPage";
import { LibraryManagementPage } from "./features/admin/LibraryManagementPage";
import { TransportManagementPage } from "./features/admin/TransportManagementPage";
import { ReportsPage } from "./features/admin/ReportsPage";
import { AlumniStudentsPage } from "./features/admin/AlumniStudentsPage";
import { SiblingsPage } from "./features/admin/SiblingsPage";
import { MoveToNextClassPage } from "./features/admin/MoveToNextClassPage";
import { FeeArrearsPage } from "./features/admin/FeeArrearsPage";
import { HeadteacherDashboardPage } from "./features/admin/HeadteacherDashboardPage";
import { ExamManagementPage } from "./features/admin/ExamManagementPage";
import { LessonPlanReviewPage } from "./features/admin/LessonPlanReviewPage";
import { ResultsOverviewPage } from "./features/admin/ResultsOverviewPage";
import { ClassTeacherAssignmentPage } from "./features/admin/ClassTeacherAssignmentPage";

// Finance
import { FinanceHome } from "./features/finance/FinanceHome";
import { InvoicesPage } from "./features/finance/InvoicesPage";
import { PaymentsPage } from "./features/finance/PaymentsPage";
import { UnreconciledPage } from "./features/finance/UnreconciledPage";
import { FeeStructuresPage } from "./features/finance/FeeStructuresPage";
import { MpesaTestPage } from "./features/finance/MpesaTestPage";
import { StkPushPage } from "./features/finance/StkPushPage";
import { StudentLedgerPage } from "./features/finance/StudentLedgerPage";

// Staff
import { PayrollPage } from "./features/staff/PayrollPage";

// School records — generic pages over the declarative resource engine
import { RecordsIndex } from "./features/records/RecordsIndex";
import { ResourcePage } from "./features/records/ResourcePage";

// Teacher — web portal only (the Flutter app is no longer the client for this)
import { TeacherHome } from "./features/teacher/TeacherHome";
import { AttendancePage } from "./features/teacher/AttendancePage";
import { MarksEntryPage } from "./features/teacher/MarksEntryPage";
import { CBCMarksheetPage } from "./features/teacher/CBCMarksheetPage";
import { LessonPlanPage } from "./features/teacher/LessonPlanPage";

// Parent
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
          <Route path="/" element={<DemoOne />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/verify" element={<VerifyPage />} />
          <Route path="/r" element={<ShortLinkPage />} />

          {/* Main portal — all roles including parent */}
          <Route element={<RequireAuth><PortalLayout /></RequireAuth>}>

            {/* Parent */}
            <Route path="/parent" element={<ParentHome />} />
            <Route path="/parent/fees" element={<ParentFeesPage />} />
            <Route path="/parent/progress" element={<ParentProgressPage />} />

            {/* Student */}
            <Route path="/student" element={<StudentHome />} />

            {/* School records (any staff role; the API decides what each can see) */}
            <Route path="/records" element={<RecordsIndex />} />
            <Route path="/records/:name" element={<ResourcePage />} />

            {/* Teacher */}
            <Route path="/teacher" element={<TeacherHome />} />
            <Route path="/teacher/attendance" element={<AttendancePage />} />
            <Route path="/teacher/marks" element={<MarksEntryPage />} />
            <Route path="/teacher/lesson-plans" element={<LessonPlanPage />} />
            <Route path="/teacher/cbc-marksheet" element={<CBCMarksheetPage />} />

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
            <Route path="/admin/alumni" element={<AlumniStudentsPage />} />
            <Route path="/admin/siblings" element={<SiblingsPage />} />
            <Route path="/admin/promote" element={<MoveToNextClassPage />} />
            <Route path="/admin/fee-arrears" element={<FeeArrearsPage />} />
            <Route path="/admin/headteacher" element={<HeadteacherDashboardPage />} />
            <Route path="/admin/exam-management" element={<ExamManagementPage />} />
            <Route path="/admin/teacher-assignment" element={<ClassTeacherAssignmentPage />} />
            <Route path="/admin/lesson-plans" element={<LessonPlanReviewPage />} />
            <Route path="/admin/results" element={<ResultsOverviewPage />} />

            {/* Finance */}
            <Route path="/finance" element={<FinanceHome />} />
            <Route path="/finance/invoices" element={<InvoicesPage />} />
            <Route path="/finance/payments" element={<PaymentsPage />} />
            <Route path="/finance/unreconciled" element={<UnreconciledPage />} />
            <Route path="/finance/fees" element={<FeeStructuresPage />} />
            <Route path="/finance/mpesa-test" element={<MpesaTestPage />} />
            <Route path="/finance/stk-push" element={<StkPushPage />} />
            <Route path="/finance/ledger" element={<StudentLedgerPage />} />

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

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default function App() {
  return import.meta.env.VITE_APP_SURFACE === "nova" ? <NovaApp /> : <PortalApp />;
}
