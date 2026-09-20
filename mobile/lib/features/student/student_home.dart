import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import '../../core/providers/auth_provider.dart';
import '../../core/theme/app_theme.dart';
import '../../shared/widgets.dart';

class StudentHomePage extends StatefulWidget {
  const StudentHomePage({super.key});

  @override
  State<StudentHomePage> createState() => _StudentHomePageState();
}

class _StudentHomePageState extends State<StudentHomePage> {
  int _idx = 0;

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final user = auth.user;
    final today = DateFormat('EEEE, MMM d, yyyy').format(DateTime.now());
    return Scaffold(
      appBar: AppBar(
        title: Text(user?.role.displayName == 'Student'
            ? 'My Dashboard'
            : "St. Mary's Kabete"),
      ),
      body: IndexedStack(
        index: _idx,
        children: [
          ListView(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Hi, ${user?.firstName ?? 'Student'} 👋',
                            style: Theme.of(context)
                                .textTheme
                                .titleLarge
                                ?.copyWith(fontWeight: FontWeight.w800),
                          ),
                          const SizedBox(height: 2),
                          Text(today,
                              style: TextStyle(
                                  color: Colors.grey.shade600, fontSize: 13)),
                        ],
                      ),
                    ),
                    AvatarCircle(
                        initials: user?.initials ?? 'S',
                        name: user?.fullName,
                        radius: 24),
                  ],
                ),
              ),
              const SectionHeader(title: 'Attendance This Term'),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Card(
                  color: AppTheme.primary,
                  child: Padding(
                    padding: const EdgeInsets.all(18),
                    child: Row(
                      children: [
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: const [
                              Text(
                                '94.2%',
                                style: TextStyle(
                                    color: Colors.white,
                                    fontSize: 40,
                                    fontWeight: FontWeight.w900),
                              ),
                              SizedBox(height: 4),
                              Text(
                                'Attendance rate',
                                style: TextStyle(
                                    color: Colors.white70, fontSize: 13),
                              ),
                            ],
                          ),
                        ),
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.end,
                          children: [
                            _mini('Present', 48, Colors.white),
                            _mini('Late', 3, AppTheme.accent),
                            _mini('Absent', 2, Colors.redAccent),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              const SectionHeader(title: 'Today\'s Timetable'),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Card(
                  child: Padding(
                    padding: const EdgeInsets.all(14),
                    child: Column(
                      children: [
                        _tt(0, '08:00 - 09:10', 'Mathematics', 'Mr. Kamau'),
                        const Divider(),
                        _tt(1, '09:20 - 10:30', 'English', 'Mrs. Wanjiru'),
                        const Divider(),
                        _tt(2, '11:00 - 12:10', 'Chemistry', 'Mr. Otieno'),
                        const Divider(),
                        _tt(3, '14:00 - 15:10', 'History', 'Mr. Kimani'),
                      ],
                    ),
                  ),
                ),
              ),
              const SectionHeader(title: 'Nova LMS'),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Row(
                  children: const [
                    Expanded(child: _Box(Icons.assignment_outlined, 'Assignments', 2, AppTheme.primary)),
                    SizedBox(width: 10),
                    Expanded(child: _Box(Icons.quiz_outlined, 'Quizzes', 1, AppTheme.warning)),
                    SizedBox(width: 10),
                    Expanded(child: _Box(Icons.menu_book_outlined, 'Materials', 5, AppTheme.info)),
                  ],
                ),
              ),
              const SizedBox(height: 30),
            ],
          ),
          const Center(child: Text('Assignments')),
          const Center(child: Text('Progress')),
          const Center(child: Text('Profile')),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _idx,
        onDestinationSelected: (i) => setState(() => _idx = i),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.home_outlined), selectedIcon: Icon(Icons.home), label: 'Home'),
          NavigationDestination(icon: Icon(Icons.assignment_outlined), selectedIcon: Icon(Icons.assignment), label: 'Work'),
          NavigationDestination(icon: Icon(Icons.bar_chart_outlined), selectedIcon: Icon(Icons.bar_chart), label: 'Progress'),
          NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: 'Profile'),
        ],
      ),
    );
  }

  Widget _mini(String label, int v, Color c) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        children: [
          Text('$v',
              style: TextStyle(color: c, fontWeight: FontWeight.w800)),
          const SizedBox(width: 6),
          Text(label, style: const TextStyle(color: Colors.white70, fontSize: 12)),
        ],
      ),
    );
  }

  Widget _tt(int i, String time, String subj, String teacher) {
    final colors = [
      AppTheme.primary,
      AppTheme.info,
      AppTheme.warning,
      Colors.purple,
      Colors.teal
    ];
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Container(
            width: 4,
            height: 38,
            decoration: BoxDecoration(
                color: colors[i % colors.length],
                borderRadius: BorderRadius.circular(2)),
          ),
          const SizedBox(width: 12),
          SizedBox(
            width: 90,
            child: Text(time,
                style: TextStyle(
                    color: Colors.grey.shade600,
                    fontWeight: FontWeight.w600,
                    fontSize: 12)),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(subj,
                    style: const TextStyle(fontWeight: FontWeight.w700)),
                const SizedBox(height: 2),
                Text(teacher,
                    style: TextStyle(color: Colors.grey.shade600, fontSize: 12)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _Box extends StatelessWidget {
  final IconData icon;
  final String label;
  final int count;
  final Color color;
  const _Box(this.icon, this.label, this.count, this.color);

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: color.withOpacity(0.15),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Icon(icon, color: color, size: 22),
            ),
            const SizedBox(height: 10),
            Text(label,
                style: const TextStyle(
                    fontSize: 12, fontWeight: FontWeight.w700)),
            const SizedBox(height: 2),
            Text('$count pending',
                style: TextStyle(color: Colors.grey.shade600, fontSize: 11)),
          ],
        ),
      ),
    );
  }
}
