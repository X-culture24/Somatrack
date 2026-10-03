import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';

class TeacherLessonPlansPage extends StatefulWidget {
  const TeacherLessonPlansPage({super.key});

  @override
  State<TeacherLessonPlansPage> createState() => _TeacherLessonPlansPageState();
}

class _TeacherLessonPlansPageState extends State<TeacherLessonPlansPage> {
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
      appBar: AppBar(title: const Text('Lesson Plans')),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () async {
          final created = await Navigator.of(context).push<bool>(
            MaterialPageRoute(builder: (_) => const NewLessonPlanPage()),
          );
          if (created == true) _load();
        },
        icon: const Icon(Icons.add),
        label: const Text('New plan'),
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _plans.isEmpty
                ? ListView(
                    children: const [
                      EmptyState(
                        message: 'No lesson plans yet — tap "New plan" to add your first one.',
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
              Row(
                children: [
                  Expanded(
                    child: Text(
                      p.topic,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
                    ),
                  ),
                  StatusChip(
                    label: p.status,
                    color: p.status == 'approved' ? AppTheme.success.withOpacity(0.12) : Colors.grey.shade100,
                    textColor: p.status == 'approved' ? AppTheme.success : Colors.grey.shade700,
                  ),
                ],
              ),
              if (p.subTopic.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text(p.subTopic, style: TextStyle(color: Colors.grey.shade600, fontSize: 13)),
              ],
              const SizedBox(height: 8),
              Wrap(
                spacing: 12,
                runSpacing: 4,
                children: [
                  _chip(Icons.class_outlined, p.className),
                  _chip(Icons.menu_book_outlined, p.subjectName),
                  _chip(Icons.calendar_today_outlined, p.lessonDate),
                  _chip(Icons.event_repeat, 'Week ${p.weekNumber}'),
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

class NewLessonPlanPage extends StatefulWidget {
  const NewLessonPlanPage({super.key});

  @override
  State<NewLessonPlanPage> createState() => _NewLessonPlanPageState();
}

class _NewLessonPlanPageState extends State<NewLessonPlanPage> {
  List<ClassGroup> _classes = [];
  List<Subject> _subjects = [];
  bool _loadingOptions = true;
  bool _saving = false;
  String? _error;

  String? _classId;
  String? _subjectId;
  final _weekCtrl = TextEditingController(text: '1');
  DateTime _date = DateTime.now();
  final _topicCtrl = TextEditingController();
  final _subTopicCtrl = TextEditingController();
  final _objectivesCtrl = TextEditingController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadOptions());
  }

  Future<void> _loadOptions() async {
    final att = context.read<AttendanceProvider>();
    _classes = await att.fetchMyClasses();
    _subjects = await att.fetchSubjects();
    if (mounted) setState(() => _loadingOptions = false);
  }

  Future<void> _save() async {
    if (_classId == null || _subjectId == null || _topicCtrl.text.trim().isEmpty) {
      setState(() => _error = 'Class, subject, and topic are required.');
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await context.read<AttendanceProvider>().createLessonPlan(
            classId: _classId!,
            subjectId: _subjectId!,
            weekNumber: int.tryParse(_weekCtrl.text) ?? 1,
            lessonDate: DateFormat('yyyy-MM-dd').format(_date),
            topic: _topicCtrl.text.trim(),
            subTopic: _subTopicCtrl.text.trim(),
            learningObjectives: _objectivesCtrl.text.trim(),
          );
      if (mounted) Navigator.of(context).pop(true);
    } catch (e) {
      setState(() => _error = 'Could not save this lesson plan: $e');
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  void dispose() {
    _weekCtrl.dispose();
    _topicCtrl.dispose();
    _subTopicCtrl.dispose();
    _objectivesCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('New Lesson Plan')),
      body: _loadingOptions
          ? const Center(child: CircularProgressIndicator())
          : _classes.isEmpty
              ? const EmptyState(
                  message: 'You are not assigned to any class yet — ask an admin to assign you before planning a lesson.',
                  icon: Icons.class_outlined,
                )
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    if (_error != null) ...[
                      Container(
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: AppTheme.danger.withOpacity(0.1),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Text(_error!, style: const TextStyle(color: AppTheme.danger)),
                      ),
                      const SizedBox(height: 12),
                    ],
                    DropdownButtonFormField<String>(
                      initialValue: _classId,
                      decoration: const InputDecoration(labelText: 'Class'),
                      items: _classes
                          .map((c) => DropdownMenuItem(value: c.id, child: Text(c.name)))
                          .toList(),
                      onChanged: (v) => setState(() => _classId = v),
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      initialValue: _subjectId,
                      decoration: const InputDecoration(labelText: 'Subject'),
                      items: _subjects
                          .map((s) => DropdownMenuItem(value: s.id, child: Text(s.name)))
                          .toList(),
                      onChanged: (v) => setState(() => _subjectId = v),
                    ),
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        Expanded(
                          child: TextField(
                            controller: _weekCtrl,
                            keyboardType: TextInputType.number,
                            decoration: const InputDecoration(labelText: 'Week no.'),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          flex: 2,
                          child: InkWell(
                            onTap: () async {
                              final picked = await showDatePicker(
                                context: context,
                                initialDate: _date,
                                firstDate: DateTime.now().subtract(const Duration(days: 365)),
                                lastDate: DateTime.now().add(const Duration(days: 365)),
                              );
                              if (picked != null) setState(() => _date = picked);
                            },
                            child: InputDecorator(
                              decoration: const InputDecoration(labelText: 'Lesson date'),
                              child: Text(DateFormat('yyyy-MM-dd').format(_date)),
                            ),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _topicCtrl,
                      decoration: const InputDecoration(labelText: 'Topic'),
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _subTopicCtrl,
                      decoration: const InputDecoration(labelText: 'Sub-topic'),
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _objectivesCtrl,
                      maxLines: 3,
                      decoration: const InputDecoration(labelText: 'Learning objectives'),
                    ),
                    const SizedBox(height: 20),
                    FilledButton(
                      onPressed: _saving ? null : _save,
                      child: _saving
                          ? const SizedBox(
                              width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                          : const Text('Save plan'),
                    ),
                  ],
                ),
    );
  }
}
