import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../core/providers/auth_provider.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';
import 'teacher_attendance_take.dart';

class TeacherHomePage extends StatefulWidget {
  const TeacherHomePage({super.key});

  @override
  State<TeacherHomePage> createState() => _TeacherHomePageState();
}

class _TeacherHomePageState extends State<TeacherHomePage> {
  List<ClassGroup> _classes = [];
  DailyAttendanceSummary? _todaySummary;
  List<Announcement> _announcements = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    final att = context.read<AttendanceProvider>();
    _classes = await att.fetchMyClasses(force: true);
    _todaySummary = await att.dailySummary();
    _announcements = await att.announcements();
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final today = DateFormat('EEEE, MMM d, yyyy').format(DateTime.now());
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Good day, ${auth.user?.firstName ?? 'Teacher'}',
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                              fontWeight: FontWeight.w800,
                            ),
                      ),
                      const SizedBox(height: 2),
                      Text(today,
                          style:
                              TextStyle(color: Colors.grey.shade600, fontSize: 13)),
                    ],
                  ),
                ),
                AvatarCircle(
                    initials: auth.user?.initials ?? 'T',
                    name: auth.user?.fullName,
                    radius: 24),
              ],
            ),
          ),
          const SizedBox(height: 12),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: GridView.count(
              crossAxisCount: 2,
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              crossAxisSpacing: 10,
              mainAxisSpacing: 10,
              childAspectRatio: 1.35,
              children: [
                StatCard(
                  title: 'My Classes',
                  value: '${_classes.length}',
                  icon: Icons.class_,
                  color: AppTheme.primary,
                ),
                StatCard(
                  title: 'Attendance Today',
                  value: _todaySummary == null
                      ? '--%'
                      : '${_todaySummary!.attendancePct.toStringAsFixed(0)}%',
                  icon: Icons.how_to_reg,
                  color: AppTheme.info,
                  percent: _todaySummary?.attendancePct,
                ),
              ],
            ),
          ),
          const SizedBox(height: 14),
          _quickActions(),
          const SectionHeader(title: 'Today\'s Attendance Summary'),
          _todaySummary == null
              ? const Padding(
                  padding: EdgeInsets.all(20),
                  child: Center(child: CircularProgressIndicator()),
                )
              : Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: Card(
                    child: Padding(
                      padding: const EdgeInsets.all(18),
                      child: Row(
                        children: [
                          Expanded(
                            child: _bigStat(
                                'Present',
                                _todaySummary!.presentCount,
                                AppTheme.success),
                          ),
                          Expanded(
                            child: _bigStat('Late', _todaySummary!.lateCount,
                                AppTheme.warning),
                          ),
                          Expanded(
                            child: _bigStat('Absent', _todaySummary!.absentCount,
                                AppTheme.danger),
                          ),
                          Expanded(
                            child: _bigStat('Total',
                                _todaySummary!.totalExpected, AppTheme.primary),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
          const SectionHeader(
              title: 'My Classes', actionLabel: 'View all'),
          if (_loading)
            const Padding(
              padding: EdgeInsets.all(30),
              child: Center(child: CircularProgressIndicator()),
            )
          else if (_classes.isEmpty)
            const Padding(
              padding: EdgeInsets.all(20),
              child: EmptyState(
                  message: 'No classes assigned yet',
                  icon: Icons.class_outlined),
            )
          else
            ..._classes.take(5).map(_classCard),
          const SectionHeader(title: 'Latest Announcements'),
          if (_announcements.isEmpty)
            const Padding(
                padding: EdgeInsets.all(20),
                child: EmptyState(
                    message: 'No announcements', icon: Icons.campaign_outlined))
          else
            ..._announcements.take(3).map((a) => Padding(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                  child: Card(
                    child: Padding(
                      padding: const EdgeInsets.all(14),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Expanded(
                                  child: Text(a.title,
                                      style: Theme.of(context)
                                          .textTheme
                                          .titleSmall
                                          ?.copyWith(
                                              fontWeight: FontWeight.w700))),
                              StatusChip.priority(a.priority),
                            ],
                          ),
                          const SizedBox(height: 6),
                          Text(a.body,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                  color: Colors.grey.shade700, height: 1.4)),
                        ],
                      ),
                    ),
                  ),
                )),
          const SizedBox(height: 30),
        ],
      ),
    );
  }

  Widget _bigStat(String label, int v, Color color) {
    return Column(
      children: [
        Text(
          '$v',
          style: TextStyle(
              color: color, fontWeight: FontWeight.w900, fontSize: 24),
        ),
        const SizedBox(height: 2),
        Text(label,
            style: TextStyle(color: Colors.grey.shade600, fontSize: 12)),
      ],
    );
  }

  Widget _quickActions() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              'Quick Actions',
              style: Theme.of(context)
                  .textTheme
                  .titleMedium
                  ?.copyWith(fontWeight: FontWeight.w700),
            ),
          ),
          Row(
            children: [
              Expanded(
                  child: _action(
                      Icons.fact_check_outlined, 'Take\nAttendance',
                      AppTheme.primary, () {
                Navigator.of(context).push(MaterialPageRoute(
                    builder: (_) => const TeacherAttendanceTakePage()));
              })),
              const SizedBox(width: 10),
              Expanded(
                  child: _action(Icons.grade_outlined, 'Marks\nEntry',
                      AppTheme.info, () {})),
              const SizedBox(width: 10),
              Expanded(
                  child: _action(Icons.assignment_outlined, 'New\nAssignment',
                      AppTheme.warning, () {})),
              const SizedBox(width: 10),
              Expanded(
                  child: _action(Icons.menu_book_outlined, 'Lesson\nPlan',
                      Colors.purple, () {})),
            ],
          ),
        ],
      ),
    );
  }

  Widget _action(IconData icon, String label, Color color, VoidCallback onTap) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 14),
        decoration: BoxDecoration(
          color: color.withOpacity(0.08),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          children: [
            Icon(icon, color: color, size: 28),
            const SizedBox(height: 6),
            Text(
              label,
              textAlign: TextAlign.center,
              style: TextStyle(
                  fontSize: 12,
                  color: color,
                  fontWeight: FontWeight.w700,
                  height: 1.2),
            ),
          ],
        ),
      ),
    );
  }

  Widget _classCard(ClassGroup c) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Card(
        child: ListTile(
          contentPadding: const EdgeInsets.all(14),
          leading: Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppTheme.primary.withOpacity(0.12),
              borderRadius: BorderRadius.circular(10),
            ),
            child:
                const Icon(Icons.class_, size: 26, color: AppTheme.primary),
          ),
          title: Text(
            c.name,
            style: Theme.of(context)
                .textTheme
                .titleMedium
                ?.copyWith(fontWeight: FontWeight.w700),
          ),
          subtitle: Text('Grade ${c.grade} · ${c.streamName} Stream'),
          trailing: Icon(Icons.chevron_right, color: Colors.grey.shade400),
          onTap: () {
            Navigator.of(context).push(
              MaterialPageRoute(
                builder: (_) => TeacherAttendanceTakePage(classId: c.id),
              ),
            );
          },
        ),
      ),
    );
  }
}
