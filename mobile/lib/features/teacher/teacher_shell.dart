import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/providers/auth_provider.dart';
import '../../../core/theme/app_theme.dart';
import 'teacher_home_page.dart';
import 'teacher_classes_page.dart';
import 'teacher_attendance_take.dart';
import 'teacher_timetable_page.dart';
import 'teacher_profile_page.dart';

class TeacherShell extends StatefulWidget {
  final Widget child;
  const TeacherShell({super.key, required this.child});

  @override
  State<TeacherShell> createState() => _TeacherShellState();
}

class _TeacherShellState extends State<TeacherShell> {
  int _index = 0;

  final _pages = const [
    TeacherHomePage(),
    TeacherClassesPage(),
    TeacherAttendanceTakePage(),
    TeacherTimetablePage(),
    TeacherProfilePage(),
  ];

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final user = auth.user;
    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            const Icon(Icons.school, size: 22),
            const SizedBox(width: 8),
            Text(_titles[_index]),
          ],
        ),
        actions: [
          IconButton(
            onPressed: () {},
            icon: const Icon(Icons.notifications_outlined),
            tooltip: 'Notifications',
          ),
        ],
      ),
      body: IndexedStack(index: _index, children: _pages),
      drawer: Drawer(
        child: SafeArea(
          child: Column(
            children: [
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(18),
                color: AppTheme.primary,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    CircleAvatar(
                      radius: 28,
                      backgroundColor: Colors.white.withOpacity(0.2),
                      child: Text(
                        user?.initials ?? 'T',
                        style: const TextStyle(
                            color: Colors.white,
                            fontSize: 20,
                            fontWeight: FontWeight.w800),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      user?.fullName ?? 'Teacher',
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 16,
                          fontWeight: FontWeight.w700),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      user?.email ?? '',
                      style: TextStyle(
                          color: Colors.white.withOpacity(0.8), fontSize: 13),
                    ),
                    const SizedBox(height: 4),
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: AppTheme.accent,
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(
                        (user?.role.displayName ?? 'STAFF').toUpperCase(),
                        style: const TextStyle(
                            fontSize: 10,
                            fontWeight: FontWeight.w800,
                            color: AppTheme.primaryDark),
                      ),
                    ),
                  ],
                ),
              ),
              ..._drawerTiles(),
              const Spacer(),
              const Divider(),
              ListTile(
                leading: const Icon(Icons.logout, color: AppTheme.danger),
                title: const Text('Sign Out',
                    style: TextStyle(color: AppTheme.danger)),
                onTap: () async {
                  Navigator.pop(context);
                  await context.read<AuthProvider>().logout();
                  if (mounted) context.go('/login');
                },
              ),
              const SizedBox(height: 10),
            ],
          ),
        ),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (i) => setState(() => _index = i),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.home_outlined), selectedIcon: Icon(Icons.home), label: 'Home'),
          NavigationDestination(icon: Icon(Icons.class_outlined), selectedIcon: Icon(Icons.class_), label: 'Classes'),
          NavigationDestination(icon: Icon(Icons.fact_check_outlined), selectedIcon: Icon(Icons.fact_check), label: 'Attendance'),
          NavigationDestination(icon: Icon(Icons.calendar_month_outlined), selectedIcon: Icon(Icons.calendar_month), label: 'Timetable'),
          NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: 'Profile'),
        ],
      ),
    );
  }

  List<ListTile> _drawerTiles() {
    return [
      ListTile(
        leading: const Icon(Icons.dashboard_outlined),
        title: const Text('Dashboard'),
        onTap: () {
          Navigator.pop(context);
          setState(() => _index = 0);
        },
      ),
      ListTile(
        leading: const Icon(Icons.assignment_outlined),
        title: const Text('Assignments / Nova'),
        onTap: () => Navigator.pop(context),
      ),
      ListTile(
        leading: const Icon(Icons.grade_outlined),
        title: const Text('Marks Entry'),
        onTap: () => Navigator.pop(context),
      ),
      ListTile(
        leading: const Icon(Icons.menu_book_outlined),
        title: const Text('Lesson Plans'),
        onTap: () => Navigator.pop(context),
      ),
      ListTile(
        leading: const Icon(Icons.local_library_outlined),
        title: const Text('Library'),
        onTap: () => Navigator.pop(context),
      ),
      ListTile(
        leading: const Icon(Icons.chat_bubble_outline),
        title: const Text('Messages'),
        onTap: () => Navigator.pop(context),
      ),
      ListTile(
        leading: const Icon(Icons.assessment_outlined),
        title: const Text('Reports'),
        onTap: () => Navigator.pop(context),
      ),
    ];
  }
}

const _titles = [
  "St. Mary's Kabete",
  'My Classes',
  'Take Attendance',
  'Timetable',
  'Profile',
];
