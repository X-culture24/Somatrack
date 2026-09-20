import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';
import 'package:fl_chart/fl_chart.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';
import '../../../shared/attendance_time_chart.dart';

class TeacherMyAttendancePage extends StatefulWidget {
  const TeacherMyAttendancePage({super.key});

  @override
  State<TeacherMyAttendancePage> createState() => _TeacherMyAttendancePageState();
}

class _TeacherMyAttendancePageState extends State<TeacherMyAttendancePage> {
  StaffAttendanceRange? _range;
  DateTime _start = DateTime.now().subtract(const Duration(days: 30));
  DateTime _end = DateTime.now();
  bool _loading = true;
  String _error = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = '';
    });
    final att = context.read<AttendanceProvider>();
    final res = await att.myStaffAttendance(start: _start, end: _end, force: true);
    if (!mounted) return;
    if (res == null && att.error != null) {
      _error = att.error!;
    }
    setState(() {
      _range = res;
      _loading = false;
    });
  }

  Future<void> _pickRange() async {
    final picked = await showDateRangePicker(
      context: context,
      firstDate: DateTime.now().subtract(const Duration(days: 365)),
      lastDate: DateTime.now(),
      initialDateRange: DateTimeRange(start: _start, end: _end),
    );
    if (picked != null) {
      setState(() {
        _start = picked.start;
        _end = picked.end;
      });
      _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    final range = _range;
    return Scaffold(
      appBar: AppBar(title: const Text('My Attendance')),
      body: ListView(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 4),
            child: Row(
              children: [
                Text(
                  '${DateFormat('MMM d').format(_start)} — ${DateFormat('MMM d, y').format(_end)}',
                  style: const TextStyle(fontWeight: FontWeight.w700),
                ),
                const Spacer(),
                TextButton.icon(
                  onPressed: _pickRange,
                  icon: const Icon(Icons.date_range, size: 18),
                  label: const Text('Change'),
                ),
              ],
            ),
          ),
          _loading
              ? const Padding(
                  padding: EdgeInsets.all(40),
                  child: Center(child: CircularProgressIndicator()),
                )
              : range == null
                  ? Padding(
                      padding: const EdgeInsets.all(40),
                      child: Center(
                        child: Text(_error.isEmpty
                            ? 'No attendance data'
                            : 'Error: $_error'),
                      ),
                    )
                  : _buildSummary(range),
          if (!_loading && range != null && range.events.isNotEmpty) ...[
            const SectionHeader(title: 'Clock-in Time vs Date'),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Card(
                child: Padding(
                  padding: const EdgeInsets.all(14),
                  child: AttendanceTimeChart(
                    title: 'Clock-in Trend',
                    points: range.events
                        .map((e) => AttendanceTimePoint(
                              date: DateTime.tryParse(e.date) ?? DateTime.now(),
                              minutesSinceMidnight:
                                  AttendanceTimePoint.parseTimeOfDay(e.timeIn),
                              status: e.status,
                            ))
                        .toList(),
                  ),
                ),
              ),
            ),
          ],
          const SectionHeader(title: 'Daily Log'),
          if (_loading)
            const Padding(
              padding: EdgeInsets.all(30),
              child: Center(child: CircularProgressIndicator()),
            )
          else if (range == null || range.events.isEmpty)
            const EmptyState(message: 'No attendance records for this period')
          else
            ...range.events.map((e) => Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                  child: Card(
                    child: ListTile(
                      title: Text(
                        DateFormat('EEEE, MMM d, yyyy')
                            .format(DateTime.tryParse(e.date) ?? DateTime.now()),
                        style: const TextStyle(fontWeight: FontWeight.w600),
                      ),
                      subtitle: Row(
                        children: [
                          Icon(Icons.login, size: 14, color: Colors.grey.shade600),
                          const SizedBox(width: 4),
                          Text(e.timeIn ?? '--:--'),
                          const SizedBox(width: 14),
                          Icon(Icons.logout, size: 14, color: Colors.grey.shade600),
                          const SizedBox(width: 4),
                          Text(e.timeOut ?? '--:--'),
                        ],
                      ),
                      trailing: StatusChip.attendance(e.status),
                    ),
                  ),
                )),
          const SizedBox(height: 30),
        ],
      ),
    );
  }

  Widget _buildSummary(StaffAttendanceRange r) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        children: [
          SizedBox(
            height: 180,
            child: PieChart(
              PieChartData(
                sections: [
                  PieChartSectionData(
                    value: r.presentDays.toDouble(),
                    color: AppTheme.success,
                    title: '${r.presentDays}',
                    radius: 38,
                    titleStyle: const TextStyle(
                        color: Colors.white, fontWeight: FontWeight.w800, fontSize: 16),
                  ),
                  PieChartSectionData(
                    value: r.lateDays.toDouble(),
                    color: AppTheme.warning,
                    title: '${r.lateDays}',
                    radius: 38,
                    titleStyle: const TextStyle(
                        color: Colors.white, fontWeight: FontWeight.w800, fontSize: 16),
                  ),
                  PieChartSectionData(
                    value: r.absentDays.toDouble(),
                    color: AppTheme.danger,
                    title: '${r.absentDays}',
                    radius: 38,
                    titleStyle: const TextStyle(
                        color: Colors.white, fontWeight: FontWeight.w800, fontSize: 16),
                  ),
                ],
                centerSpaceRadius: 40,
              ),
            ),
          ),
          const SizedBox(height: 8),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              _legend('Present', AppTheme.success),
              const SizedBox(width: 18),
              _legend('Late', AppTheme.warning),
              const SizedBox(width: 18),
              _legend('Absent', AppTheme.danger),
            ],
          ),
          const SizedBox(height: 14),
          Container(
            padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 18),
            decoration: BoxDecoration(
              color: AppTheme.primary.withOpacity(0.08),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              children: [
                Text(
                  '${r.attendancePct.toStringAsFixed(1)}%',
                  style: const TextStyle(
                    fontSize: 30,
                    fontWeight: FontWeight.w900,
                    color: AppTheme.primary,
                  ),
                ),
                const SizedBox(width: 12),
                const Expanded(
                  child: Text(
                    'Overall attendance rate for this period',
                    style: TextStyle(height: 1.3, fontSize: 13),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          GridView.count(
            crossAxisCount: 4,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            childAspectRatio: 1.1,
            crossAxisSpacing: 8,
            children: [
              _stat('Days', r.expectedDays, AppTheme.primary),
              _stat('Present', r.presentDays, AppTheme.success),
              _stat('Late', r.lateDays, AppTheme.warning),
              _stat('Absent', r.absentDays, AppTheme.danger),
            ],
          ),
        ],
      ),
    );
  }

  Widget _stat(String label, int v, Color color) {
    return Card(
      margin: EdgeInsets.zero,
      color: color.withOpacity(0.08),
      elevation: 0,
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text('$v',
              style:
                  TextStyle(color: color, fontSize: 22, fontWeight: FontWeight.w800)),
          const SizedBox(height: 2),
          Text(label, style: TextStyle(color: Colors.grey.shade700, fontSize: 12)),
        ],
      ),
    );
  }

  Widget _legend(String label, Color color) {
    return Row(
      children: [
        Container(
          width: 12,
          height: 12,
          decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(3)),
        ),
        const SizedBox(width: 6),
        Text(label, style: const TextStyle(fontSize: 12)),
      ],
    );
  }
}
