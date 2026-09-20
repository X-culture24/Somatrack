import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';

class TeacherAttendanceTakePage extends StatefulWidget {
  final String? classId;
  final DateTime? date;
  const TeacherAttendanceTakePage({super.key, this.classId, this.date});

  @override
  State<TeacherAttendanceTakePage> createState() =>
      _TeacherAttendanceTakePageState();
}

class _TeacherAttendanceTakePageState
    extends State<TeacherAttendanceTakePage> {
  String? _selectedClass;
  late DateTime _selectedDate;
  List<ClassGroup> _classes = [];
  List<ClassRosterAttendance> _roster = [];
  final Map<String, String> _statuses = {};
  bool _loading = false;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _selectedDate = widget.date ?? DateTime.now();
    WidgetsBinding.instance.addPostFrameCallback((_) => _init());
  }

  Future<void> _init() async {
    final att = context.read<AttendanceProvider>();
    _classes = await att.fetchMyClasses();
    if (_classes.isNotEmpty) {
      setState(() => _selectedClass = widget.classId ?? _classes.first.id);
      await _loadRoster();
    } else {
      if (mounted) setState(() {});
    }
  }

  Future<void> _loadRoster() async {
    if (_selectedClass == null) return;
    setState(() => _loading = true);
    final att = context.read<AttendanceProvider>();
    final list = await att.classRoster(
      classId: _selectedClass!,
      date: _selectedDate,
      force: true,
    );
    for (final r in list) {
      if (!_statuses.containsKey(r.studentId)) {
        String s = 'present';
        if (r.absentDays > 0) {
          s = 'absent';
        } else if (r.lateDays > 0) {
          s = 'late';
        }
        _statuses[r.studentId] = s;
      }
    }
    setState(() {
      _roster = list;
      _loading = false;
    });
  }

  Future<void> _pickDate() async {
    final d = await showDatePicker(
      context: context,
      firstDate: DateTime.now().subtract(const Duration(days: 60)),
      lastDate: DateTime.now().add(const Duration(days: 1)),
      initialDate: _selectedDate,
    );
    if (d != null && mounted) {
      setState(() => _selectedDate = d);
      _loadRoster();
    }
  }

  Future<void> _saveAttendance() async {
    setState(() => _saving = true);
    await Future.delayed(const Duration(seconds: 1));
    if (!mounted) return;
    setState(() => _saving = false);
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('Attendance saved successfully'),
        backgroundColor: AppTheme.success,
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  void _markAll(String status) {
    setState(() {
      for (final r in _roster) {
        _statuses[r.studentId] = status;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final dateStr = DateFormat('EEEE, MMM d, yyyy').format(_selectedDate);
    final cls = _classes.cast<ClassGroup?>().firstWhere(
          (c) => c?.id == _selectedClass,
          orElse: () => null,
        );
    final stats = _computeStats();
    return Scaffold(
      floatingActionButton: _roster.isNotEmpty
          ? FloatingActionButton.extended(
              onPressed: _saving ? null : _saveAttendance,
              backgroundColor: AppTheme.primary,
              icon: _saving
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white))
                  : const Icon(Icons.save_alt),
              label: Text(_saving ? 'SAVING…' : 'SAVE ATTENDANCE',
                  style: const TextStyle(fontWeight: FontWeight.w800)),
            )
          : null,
      body: CustomScrollView(
        slivers: [
          SliverToBoxAdapter(
            child: Container(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 16),
              color: AppTheme.primary.withOpacity(0.06),
              child: Column(
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: DropdownButtonFormField<String>(
                          value: _selectedClass,
                          decoration: const InputDecoration(
                            labelText: 'Select Class',
                            prefixIcon: Icon(Icons.class_),
                          ),
                          items: _classes
                              .map((c) => DropdownMenuItem(
                                    value: c.id,
                                    child: Text(c.name),
                                  ))
                              .toList(),
                          onChanged: (v) {
                            setState(() => _selectedClass = v);
                            _loadRoster();
                          },
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: InkWell(
                          onTap: _pickDate,
                          borderRadius: BorderRadius.circular(10),
                          child: InputDecorator(
                            decoration: const InputDecoration(
                              labelText: 'Date',
                              prefixIcon: Icon(Icons.today),
                              suffixIcon: Icon(Icons.keyboard_arrow_down),
                            ),
                            child: Text(dateStr),
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  if (cls != null)
                    Row(
                      children: [
                        Expanded(
                          child: Card(
                            color: AppTheme.success.withOpacity(0.12),
                            elevation: 0,
                            child: Padding(
                              padding: const EdgeInsets.all(12),
                              child: Column(
                                children: [
                                  Text('${stats.$1}',
                                      style: const TextStyle(
                                          fontSize: 22,
                                          fontWeight: FontWeight.w900,
                                          color: AppTheme.success)),
                                  const Text('Present',
                                      style: TextStyle(fontSize: 12)),
                                ],
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Card(
                            color: AppTheme.warning.withOpacity(0.15),
                            elevation: 0,
                            child: Padding(
                              padding: const EdgeInsets.all(12),
                              child: Column(
                                children: [
                                  Text('${stats.$2}',
                                      style: const TextStyle(
                                          fontSize: 22,
                                          fontWeight: FontWeight.w900,
                                          color: AppTheme.warning)),
                                  const Text('Late',
                                      style: TextStyle(fontSize: 12)),
                                ],
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Card(
                            color: AppTheme.danger.withOpacity(0.1),
                            elevation: 0,
                            child: Padding(
                              padding: const EdgeInsets.all(12),
                              child: Column(
                                children: [
                                  Text('${stats.$3}',
                                      style: const TextStyle(
                                          fontSize: 22,
                                          fontWeight: FontWeight.w900,
                                          color: AppTheme.danger)),
                                  const Text('Absent',
                                      style: TextStyle(fontSize: 12)),
                                ],
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Card(
                            color: AppTheme.primary.withOpacity(0.1),
                            elevation: 0,
                            child: Padding(
                              padding: const EdgeInsets.all(12),
                              child: Column(
                                children: [
                                  Text('${_roster.length}',
                                      style: const TextStyle(
                                          fontSize: 22,
                                          fontWeight: FontWeight.w900,
                                          color: AppTheme.primary)),
                                  const Text('Total',
                                      style: TextStyle(fontSize: 12)),
                                ],
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () => _markAll('present'),
                          icon: const Icon(Icons.check_circle_outline, size: 18),
                          label: const Text('Mark All Present'),
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () => _markAll('absent'),
                          style: OutlinedButton.styleFrom(
                              foregroundColor: AppTheme.danger,
                              side:
                                  const BorderSide(color: AppTheme.danger)),
                          icon: const Icon(Icons.cancel_outlined, size: 18),
                          label: const Text('Mark All Absent'),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
          const SliverToBoxAdapter(
            child: SectionHeader(title: 'Students · Tap to change status'),
          ),
          if (_loading)
            const SliverFillRemaining(
                child: Center(child: CircularProgressIndicator()))
          else if (_roster.isEmpty)
            const SliverFillRemaining(
              child: EmptyState(
                message:
                    'No students in this class roster. Please select a different class.',
                icon: Icons.people_outline,
              ),
            )
          else
            SliverList.separated(
              itemCount: _roster.length,
              separatorBuilder: (_, __) =>
                  Divider(height: 1, color: Colors.grey.shade200),
              itemBuilder: (c, i) {
                final r = _roster[i];
                final s = _statuses[r.studentId] ?? 'present';
                return _studentTile(r, s);
              },
            ),
          const SliverToBoxAdapter(child: SizedBox(height: 90)),
        ],
      ),
    );
  }

  (int, int, int) _computeStats() {
    int p = 0, l = 0, a = 0;
    for (final e in _statuses.values) {
      if (e == 'present') {
        p++;
      } else if (e == 'late') {
        l++;
      } else if (e == 'absent') {
        a++;
      }
    }
    return (p, l, a);
  }

  Widget _studentTile(ClassRosterAttendance r, String status) {
    return Material(
      color: Colors.white,
      child: InkWell(
        onTap: () {
          setState(() {
            if (status == 'present') {
              _statuses[r.studentId] = 'late';
            } else if (status == 'late') {
              _statuses[r.studentId] = 'absent';
            } else {
              _statuses[r.studentId] = 'present';
            }
          });
        },
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          child: Row(
            children: [
              AvatarCircle(
                  initials: (r.studentName.isNotEmpty
                      ? r.studentName[0]
                      : '?'),
                  name: r.studentName,
                  radius: 22),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(r.studentName,
                        style: Theme.of(context)
                            .textTheme
                            .titleSmall
                            ?.copyWith(fontWeight: FontWeight.w700)),
                    const SizedBox(height: 3),
                    Text('Adm #${r.admissionNo}',
                        style:
                            TextStyle(color: Colors.grey.shade600, fontSize: 12)),
                  ],
                ),
              ),
              SizedBox(
                width: 90,
                child: Align(
                  alignment: Alignment.centerRight,
                  child: StatusChip.attendance(status),
                ),
              ),
              const SizedBox(width: 8),
              const Icon(Icons.touch_app_outlined,
                  size: 18, color: Colors.grey),
            ],
          ),
        ),
      ),
    );
  }
}
