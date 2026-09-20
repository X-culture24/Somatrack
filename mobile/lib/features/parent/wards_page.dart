import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../shared/widgets.dart';
import 'child_attendance_page.dart';

class WardsPage extends StatefulWidget {
  const WardsPage({super.key});

  @override
  State<WardsPage> createState() => _WardsPageState();
}

class _WardsPageState extends State<WardsPage> {
  List<Student> _list = [];
  bool _loading = true;
  String _q = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    _list = await context.read<AttendanceProvider>().myWards(force: true);
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final filtered = _q.isEmpty
        ? _list
        : _list
            .where((s) =>
                s.fullName.toLowerCase().contains(_q.toLowerCase()) ||
                s.admissionNo.toLowerCase().contains(_q.toLowerCase()))
            .toList();
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
                    hintText: 'Search by name or admission #',
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
                  message: 'No children linked to this guardian account',
                  icon: Icons.family_restroom_outlined,
                ),
              )
            else
              SliverList(
                delegate: SliverChildBuilderDelegate(
                  (c, i) => _buildTile(filtered[i]),
                  childCount: filtered.length,
                ),
              ),
            const SliverToBoxAdapter(child: SizedBox(height: 30)),
          ],
        ),
      ),
    );
  }

  Widget _buildTile(Student s) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
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
                Hero(
                  tag: 'avatar-${s.id}',
                  child: AvatarCircle(
                    initials: s.initials,
                    name: s.fullName,
                    radius: 28,
                  ),
                ),
                const SizedBox(width: 14),
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
                      const SizedBox(height: 6),
                      Wrap(
                        spacing: 10,
                        runSpacing: 4,
                        children: [
                          _chip(Icons.badge_outlined, s.admissionNo),
                          if (s.className != null)
                            _chip(Icons.class_outlined, s.className!),
                          _chip(
                            s.status == 'active' ? Icons.check_circle : Icons.cancel,
                            s.status.toUpperCase(),
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

  Widget _chip(IconData icon, String label) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 14, color: Colors.grey.shade600),
        const SizedBox(width: 4),
        Text(
          label,
          style: TextStyle(color: Colors.grey.shade700, fontSize: 12.5),
        ),
      ],
    );
  }
}
