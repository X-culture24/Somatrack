import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../core/providers/auth_provider.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';
import 'child_attendance_page.dart';

class ParentHomePage extends StatefulWidget {
  const ParentHomePage({super.key});

  @override
  State<ParentHomePage> createState() => _ParentHomePageState();
}

class _ParentHomePageState extends State<ParentHomePage> {
  List<Student> _wards = [];
  List<Announcement> _announcements = [];
  DailyAttendanceSummary? _todaySummary;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    final att = context.read<AttendanceProvider>();
    _wards = await att.myWards();
    _announcements = await att.announcements();
    _todaySummary = await att.dailySummary();
    if (mounted) {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final today = DateFormat('EEEE, MMM d, yyyy').format(DateTime.now());
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
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
                        'Welcome back, ${auth.user?.firstName ?? 'Guardian'}',
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                              fontWeight: FontWeight.w800,
                            ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        today,
                        style: TextStyle(color: Colors.grey.shade600, fontSize: 13),
                      ),
                    ],
                  ),
                ),
                AvatarCircle(
                  initials: auth.user?.initials ?? 'P',
                  name: auth.user?.fullName,
                  radius: 24,
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          _buildTodayAttendance(),
          const SizedBox(height: 8),
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
                  title: 'My Children',
                  value: '${_wards.length}',
                  icon: Icons.family_restroom,
                  color: AppTheme.primary,
                ),
                StatCard(
                  title: 'Unpaid Fees',
                  value: 'KES ${_feesBalance().toStringAsFixed(0)}',
                  icon: Icons.payments,
                  color: AppTheme.danger,
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          SectionHeader(
            title: 'My Children',
            actionLabel: 'View all',
            onAction: () {},
          ),
          if (_wards.isEmpty)
            const Padding(
              padding: EdgeInsets.all(16),
              child: EmptyState(message: 'No children linked to this account'),
            )
          else
            ..._wards.take(3).map(_buildWardTile),
          const SectionHeader(title: 'Latest Announcements'),
          if (_announcements.isEmpty)
            const Padding(
              padding: EdgeInsets.all(16),
              child: EmptyState(
                message: 'No recent announcements from the school',
                icon: Icons.campaign_outlined,
              ),
            )
          else
            ..._announcements.take(3).map(_buildAnnouncementTile),
          const SizedBox(height: 30),
        ],
      ),
    );
  }

  double _feesBalance() {
    return 0.0;
  }

  Widget _buildTodayAttendance() {
    final sum = _todaySummary;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Card(
        color: AppTheme.primary,
        child: Padding(
          padding: const EdgeInsets.all(18),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: const [
                  Icon(Icons.today, color: Colors.white70, size: 18),
                  SizedBox(width: 6),
                  Text(
                    'TODAY\'S ATTENDANCE',
                    style: TextStyle(
                        color: Colors.white70,
                        fontWeight: FontWeight.w700,
                        fontSize: 12,
                        letterSpacing: 0.5),
                  ),
                ],
              ),
              const SizedBox(height: 14),
              sum == null
                  ? const Text('No data', style: TextStyle(color: Colors.white))
                  : Row(
                      children: [
                        Expanded(
                          child: Text(
                            '${sum.attendancePct.toStringAsFixed(0)}%',
                            style: const TextStyle(
                              color: Colors.white,
                              fontSize: 44,
                              fontWeight: FontWeight.w900,
                            ),
                          ),
                        ),
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.end,
                          children: [
                            _miniStat('Present', sum.presentCount, Colors.white),
                            _miniStat('Late', sum.lateCount, AppTheme.accent),
                            _miniStat('Absent', sum.absentCount, Colors.redAccent),
                          ],
                        ),
                      ],
                    ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _miniStat(String label, int value, Color color) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '$value',
            style: TextStyle(
                color: color, fontWeight: FontWeight.w800, fontSize: 14),
          ),
          const SizedBox(width: 6),
          Text(label,
              style: const TextStyle(color: Colors.white70, fontSize: 12)),
        ],
      ),
    );
  }

  Widget _buildWardTile(Student s) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Card(
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () {
            Navigator.of(context).push(
              MaterialPageRoute(
                builder: (_) => ChildAttendancePage(student: s),
              ),
            );
          },
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Row(
              children: [
                AvatarCircle(
                  initials: s.initials,
                  name: s.fullName,
                  radius: 26,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        s.fullName,
                        style: Theme.of(context).textTheme.titleMedium?.copyWith(
                              fontWeight: FontWeight.w700,
                            ),
                      ),
                      const SizedBox(height: 4),
                      Row(
                        children: [
                          Icon(Icons.badge_outlined,
                              size: 14, color: Colors.grey.shade600),
                          const SizedBox(width: 4),
                          Text(
                            s.admissionNo,
                            style: TextStyle(
                                color: Colors.grey.shade600, fontSize: 13),
                          ),
                          const SizedBox(width: 12),
                          if (s.className != null)
                            Icon(Icons.class_outlined,
                                size: 14, color: Colors.grey.shade600),
                          if (s.className != null) const SizedBox(width: 4),
                          if (s.className != null)
                            Text(
                              s.className!,
                              style: TextStyle(
                                  color: Colors.grey.shade600, fontSize: 13),
                            ),
                        ],
                      ),
                    ],
                  ),
                ),
                Icon(Icons.chevron_right, color: Colors.grey.shade400),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildAnnouncementTile(Announcement a) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      a.title,
                      style: Theme.of(context).textTheme.titleSmall?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                    ),
                  ),
                  StatusChip.priority(a.priority),
                ],
              ),
              const SizedBox(height: 6),
              Text(
                a.body,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: Colors.grey.shade700, height: 1.4),
              ),
              if (a.createdAt != null) ...[
                const SizedBox(height: 8),
                Text(
                  DateFormat('MMM d, h:mm a').format(a.createdAt!),
                  style: TextStyle(color: Colors.grey.shade500, fontSize: 12),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
