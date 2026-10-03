import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../shared/widgets.dart';

/// Shows what's being taught in a parent's children's class(es) — the
/// closest real "what work is my child doing" data available today
/// (lesson plans). Scoped server-side (internal/api/lesson_plans.go) to
/// classes the caller's own children are actually enrolled in.
class ClassWorkPage extends StatefulWidget {
  const ClassWorkPage({super.key});

  @override
  State<ClassWorkPage> createState() => _ClassWorkPageState();
}

class _ClassWorkPageState extends State<ClassWorkPage> {
  List<LessonPlan> _plans = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    _plans = await context.read<AttendanceProvider>().fetchLessonPlans(force: true);
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Class Work')),
      body: RefreshIndicator(
        onRefresh: _load,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _plans.isEmpty
                ? ListView(
                    children: const [
                      EmptyState(
                        message: "No lessons recorded for your child's class yet.",
                        icon: Icons.menu_book_outlined,
                      ),
                    ],
                  )
                : ListView.builder(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    itemCount: _plans.length,
                    itemBuilder: (c, i) => _tile(_plans[i]),
                  ),
      ),
    );
  }

  Widget _tile(LessonPlan p) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(p.topic, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
              if (p.subTopic.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text(p.subTopic, style: TextStyle(color: Colors.grey.shade600, fontSize: 13)),
              ],
              if (p.learningObjectives.isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(p.learningObjectives, style: TextStyle(color: Colors.grey.shade700, height: 1.4)),
              ],
              const SizedBox(height: 8),
              Wrap(
                spacing: 12,
                runSpacing: 4,
                children: [
                  _chip(Icons.class_outlined, p.className),
                  _chip(Icons.menu_book_outlined, p.subjectName),
                  _chip(Icons.calendar_today_outlined, p.lessonDate),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _chip(IconData icon, String label) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 14, color: Colors.grey.shade600),
        const SizedBox(width: 4),
        Text(label, style: TextStyle(color: Colors.grey.shade700, fontSize: 12.5)),
      ],
    );
  }
}
