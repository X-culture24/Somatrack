import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:table_calendar/table_calendar.dart';

import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';

class TeacherTimetablePage extends StatefulWidget {
  const TeacherTimetablePage({super.key});

  @override
  State<TeacherTimetablePage> createState() => _TeacherTimetablePageState();
}

class _TeacherTimetablePageState extends State<TeacherTimetablePage> {
  DateTime _focused = DateTime.now();
  DateTime? _selected;

  final Map<int, List<_Slot>> _weekly = {
    1: [
      _slot(8, 0, 9, 10, 'Mathematics', 'Form 4W', Room: 'L1'),
      _slot(9, 20, 10, 30, 'Chemistry', 'Form 3E', Room: 'Lab 2'),
      _slot(11, 0, 12, 10, 'Physics', 'Form 4W', Room: 'Lab 1'),
      _slot(14, 0, 15, 10, 'Mathematics', 'Form 3E', Room: 'L3'),
    ],
    2: [
      _slot(8, 0, 9, 10, 'Biology', 'Form 4W', Room: 'Lab 3'),
      _slot(9, 20, 10, 30, 'Mathematics', 'Form 2N', Room: 'L2'),
      _slot(11, 0, 12, 10, 'Chemistry', 'Form 4W', Room: 'Lab 2'),
      _slot(14, 0, 15, 10, 'Physics', 'Form 3E', Room: 'Lab 1'),
    ],
    3: [
      _slot(8, 0, 9, 10, 'Mathematics', 'Form 4W', Room: 'L1'),
      _slot(10, 0, 11, 10, 'Physics', 'Form 2N', Room: 'Lab 1'),
      _slot(11, 20, 12, 30, 'Biology', 'Form 3E', Room: 'Lab 3'),
      _slot(14, 0, 15, 10, 'Chemistry', 'Form 4W', Room: 'Lab 2'),
    ],
    4: [
      _slot(8, 40, 9, 50, 'Chemistry', 'Form 4W', Room: 'Lab 2'),
      _slot(10, 0, 11, 10, 'Mathematics', 'Form 4W', Room: 'L1'),
      _slot(11, 20, 12, 30, 'Biology', 'Form 4W', Room: 'Lab 3'),
      _slot(14, 0, 15, 10, 'Physics', 'Form 2N', Room: 'Lab 1'),
    ],
    5: [
      _slot(8, 0, 9, 10, 'Physics', 'Form 4W', Room: 'Lab 1'),
      _slot(9, 20, 10, 30, 'Chemistry', 'Form 3E', Room: 'Lab 2'),
      _slot(11, 0, 12, 10, 'Mathematics', 'Form 3E', Room: 'L3'),
      _slot(14, 0, 15, 10, 'Remedial', 'Form 4W', Room: 'L1'),
    ],
  };

  @override
  void initState() {
    super.initState();
    _selected = DateTime.now();
  }

  @override
  Widget build(BuildContext context) {
    final dow = _selected!.weekday;
    final slots = _weekly[dow] ?? [];
    final todayFmt =
        DateFormat('EEEE, MMM d, yyyy').format(_selected ?? DateTime.now());
    return Scaffold(
      body: ListView(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 0),
            child: Card(
              child: TableCalendar(
                focusedDay: _focused,
                firstDay: DateTime.now().subtract(const Duration(days: 120)),
                lastDay: DateTime.now().add(const Duration(days: 120)),
                selectedDayPredicate: (d) => isSameDay(_selected, d),
                onDaySelected: (s, f) {
                  setState(() {
                    _selected = s;
                    _focused = f;
                  });
                },
                headerStyle: const HeaderStyle(
                  formatButtonVisible: false,
                  titleCentered: true,
                  headerPadding: EdgeInsets.symmetric(vertical: 6),
                  titleTextStyle:
                      TextStyle(fontWeight: FontWeight.w700, fontSize: 15),
                ),
                calendarStyle: CalendarStyle(
                  todayDecoration: BoxDecoration(
                    color: AppTheme.accent,
                    shape: BoxShape.circle,
                  ),
                  todayTextStyle: const TextStyle(
                      color: AppTheme.primaryDark, fontWeight: FontWeight.w800),
                  selectedDecoration: const BoxDecoration(
                    color: AppTheme.primary,
                    shape: BoxShape.circle,
                  ),
                  selectedTextStyle: const TextStyle(
                      color: Colors.white, fontWeight: FontWeight.w700),
                  weekendTextStyle: TextStyle(color: Colors.grey.shade600),
                ),
                availableGestures: AvailableGestures.horizontalSwipe,
              ),
            ),
          ),
          const SizedBox(height: 10),
          SectionHeader(title: todayFmt),
          if (dow > 5 || slots.isEmpty)
            const Padding(
              padding: EdgeInsets.all(30),
              child: EmptyState(
                message: 'No scheduled lessons for this day.',
                icon: Icons.event_available_outlined,
              ),
            )
          else
            ...slots.map(_slotTile),
          const SizedBox(height: 30),
        ],
      ),
    );
  }

  Widget _slotTile(_Slot s) {
    final colors = [
      AppTheme.primary,
      AppTheme.info,
      AppTheme.warning,
      Colors.purple,
      Colors.teal,
      Colors.brown,
    ];
    final idx = s.subject.codeUnitAt(0) % colors.length;
    final c = colors[idx];
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 5),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Row(
            children: [
              Container(
                width: 5,
                height: 60,
                decoration: BoxDecoration(
                  color: c,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            s.subject,
                            style: Theme.of(context)
                                .textTheme
                                .titleMedium
                                ?.copyWith(fontWeight: FontWeight.w800),
                          ),
                        ),
                        Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 8, vertical: 3),
                          decoration: BoxDecoration(
                              color: c.withOpacity(0.12),
                              borderRadius: BorderRadius.circular(6)),
                          child: Text(
                            s.className,
                            style: TextStyle(
                                color: c,
                                fontSize: 12,
                                fontWeight: FontWeight.w700),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        Icon(Icons.access_time,
                            size: 15, color: Colors.grey.shade600),
                        const SizedBox(width: 6),
                        Text(
                          '${s.timeFrom} — ${s.timeTo}',
                          style: TextStyle(
                              color: Colors.grey.shade700,
                              fontWeight: FontWeight.w600),
                        ),
                        const SizedBox(width: 14),
                        Icon(Icons.room_outlined,
                            size: 15, color: Colors.grey.shade600),
                        const SizedBox(width: 6),
                        Text(s.Room,
                            style: TextStyle(color: Colors.grey.shade700)),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Slot {
  final String timeFrom;
  final String timeTo;
  final String subject;
  final String className;
  final String Room;
  _Slot(this.timeFrom, this.timeTo, this.subject, this.className, {required this.Room});
}

_Slot _slot(int h1, int m1, int h2, int m2, String subj, String cls,
    {required String Room}) {
  String fmt(int h, int m) =>
      '${h.toString().padLeft(2, '0')}:${m.toString().padLeft(2, '0')}';
  return _Slot(fmt(h1, m1), fmt(h2, m2), subj, cls, Room: Room);
}
