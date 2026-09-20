import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';
import 'teacher_attendance_take.dart';

class TeacherClassesPage extends StatefulWidget {
  const TeacherClassesPage({super.key});

  @override
  State<TeacherClassesPage> createState() => _TeacherClassesPageState();
}

class _TeacherClassesPageState extends State<TeacherClassesPage> {
  List<ClassGroup> _classes = [];
  bool _loading = true;
  String _q = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    _classes = await context.read<AttendanceProvider>().fetchMyClasses(force: true);
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final filtered = _q.isEmpty
        ? _classes
        : _classes.where((c) => c.name.toLowerCase().contains(_q.toLowerCase())).toList();
    return Scaffold(
      body: RefreshIndicator(
        onRefresh: _load,
        child: CustomScrollView(
          slivers: [
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 6),
                child: TextField(
                  onChanged: (v) => setState(() => _q = v),
                  decoration: const InputDecoration(
                    hintText: 'Search classes…',
                    prefixIcon: Icon(Icons.search),
                  ),
                ),
              ),
            ),
            if (_loading)
              const SliverFillRemaining(
                  child: Center(child: CircularProgressIndicator()))
            else if (filtered.isEmpty)
              const SliverFillRemaining(
                child: EmptyState(
                  message: 'No classes assigned for this teacher account.',
                  icon: Icons.class_outlined,
                ),
              )
            else
              SliverList(
                delegate: SliverChildBuilderDelegate(
                  (c, i) => _tile(filtered[i]),
                  childCount: filtered.length,
                ),
              ),
            const SliverToBoxAdapter(child: SizedBox(height: 30)),
          ],
        ),
      ),
    );
  }

  Widget _tile(ClassGroup c) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Card(
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () => Navigator.of(context).push(MaterialPageRoute(
            builder: (_) => TeacherAttendanceTakePage(classId: c.id),
          )),
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                    color: AppTheme.primary.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: const Icon(Icons.class_, size: 30, color: AppTheme.primary),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        c.name,
                        style: Theme.of(context).textTheme.titleMedium?.copyWith(
                              fontWeight: FontWeight.w800,
                            ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'Grade ${c.grade} · ${c.streamName} Stream',
                        style: TextStyle(color: Colors.grey.shade600, fontSize: 13),
                      ),
                    ],
                  ),
                ),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    TextButton.icon(
                      onPressed: () {},
                      icon: const Icon(Icons.bar_chart, size: 18),
                      label: const Text('Roster'),
                    ),
                    OutlinedButton.icon(
                      onPressed: () => Navigator.of(context).push(MaterialPageRoute(
                        builder: (_) =>
                            TeacherAttendanceTakePage(classId: c.id),
                      )),
                      style: OutlinedButton.styleFrom(
                          foregroundColor: AppTheme.primary,
                          side: const BorderSide(color: AppTheme.primary)),
                      icon: const Icon(Icons.fact_check, size: 16),
                      label: const Text('Take Attendance',
                          style: TextStyle(fontWeight: FontWeight.w700)),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
