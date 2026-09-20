import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../providers/auth_provider.dart';
import '../../features/auth/login_page.dart';
import '../../features/parent/parent_shell.dart';
import '../../features/parent/parent_home_page.dart';
import '../../features/teacher/teacher_shell.dart';
import '../../features/teacher/teacher_home_page.dart';
import '../../features/teacher/teacher_my_attendance_page.dart';
import '../../features/student/student_home.dart';

class AppRouter {
  static GoRouter create(AuthProvider auth) {
    return GoRouter(
      initialLocation: '/',
      refreshListenable: auth,
      redirect: (context, state) {
        final loggingIn = state.matchedLocation == '/login';
        if (auth.isLoading) return null;
        final authenticated = auth.isAuthenticated;
        if (!authenticated && !loggingIn) return '/login';
        if (authenticated && loggingIn) {
          if (auth.isTeacher || auth.isStaff) return '/teacher';
          if (auth.isParent) return '/parent';
          if (auth.isStudent) return '/student';
          return '/teacher';
        }
        return null;
      },
      routes: [
        GoRoute(
          path: '/',
          redirect: (c, s) => '/login',
        ),
        GoRoute(
          path: '/login',
          builder: (_, __) => const LoginPage(),
        ),
        ShellRoute(
          builder: (_, state, child) => ParentShell(child: child),
          routes: [
            GoRoute(
              path: '/parent',
              name: 'parent-home',
              builder: (_, __) => const ParentHomePage(),
            ),
          ],
        ),
        ShellRoute(
          builder: (_, state, child) => TeacherShell(child: child),
          routes: [
            GoRoute(
              path: '/teacher',
              name: 'teacher-home',
              builder: (_, __) => const TeacherHomePage(),
            ),
          ],
        ),
        GoRoute(
          path: '/teacher/attendance',
          name: 'teacher-my-attendance',
          builder: (_, __) => const TeacherMyAttendancePage(),
        ),
        GoRoute(
          path: '/student',
          builder: (_, __) => const StudentHomePage(),
        ),
      ],
      errorBuilder: (context, state) => Scaffold(
        body: Center(child: Text('Route not found: ${state.error}')),
      ),
    );
  }
}
