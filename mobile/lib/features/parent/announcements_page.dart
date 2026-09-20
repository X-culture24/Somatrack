import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';

class AnnouncementsPage extends StatefulWidget {
  const AnnouncementsPage({super.key});

  @override
  State<AnnouncementsPage> createState() => _AnnouncementsPageState();
}

class _AnnouncementsPageState extends State<AnnouncementsPage> {
  List<Announcement> _items = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    _items = await context.read<AttendanceProvider>().announcements(force: true);
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: RefreshIndicator(
        onRefresh: _load,
        child: CustomScrollView(
          slivers: [
            const SliverToBoxAdapter(child: SizedBox(height: 16)),
            if (_loading)
              const SliverFillRemaining(
                  child: Center(child: CircularProgressIndicator()))
            else if (_items.isEmpty)
              const SliverFillRemaining(
                child: EmptyState(
                  message: 'No announcements yet. Check back later!',
                  icon: Icons.campaign_outlined,
                ),
              )
            else
              SliverList(
                delegate: SliverChildBuilderDelegate(
                  (c, i) => _tile(_items[i]),
                  childCount: _items.length,
                ),
              ),
            const SliverToBoxAdapter(child: SizedBox(height: 30)),
          ],
        ),
      ),
    );
  }

  Widget _tile(Announcement a) {
    final isHigh = a.priority == 'high' || a.priority == 'urgent';
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Card(
        color: isHigh ? AppTheme.danger.withOpacity(0.04) : null,
        shape: isHigh
            ? RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
                side: BorderSide(color: AppTheme.danger.withOpacity(0.4), width: 1.2),
              )
            : null,
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () => _open(a),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(
                      isHigh ? Icons.priority_high : Icons.info_outline,
                      color: isHigh ? AppTheme.danger : AppTheme.info,
                      size: 20,
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Text(
                        a.title,
                        style:
                            Theme.of(context).textTheme.titleMedium?.copyWith(
                                  fontWeight: FontWeight.w800,
                                ),
                      ),
                    ),
                    StatusChip.priority(a.priority),
                  ],
                ),
                const SizedBox(height: 10),
                Text(
                  a.body,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(color: Colors.grey.shade700, height: 1.45),
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Icon(Icons.access_time,
                        size: 14, color: Colors.grey.shade500),
                    const SizedBox(width: 4),
                    Text(
                      a.createdAt != null
                          ? DateFormat('MMM d, y · h:mm a').format(a.createdAt!)
                          : '',
                      style: TextStyle(color: Colors.grey.shade500, fontSize: 12),
                    ),
                    const Spacer(),
                    Text(
                      'Audience: ${a.audience}',
                      style: TextStyle(color: Colors.grey.shade500, fontSize: 12),
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

  void _open(Announcement a) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => DraggableScrollableSheet(
        initialChildSize: 0.7,
        maxChildSize: 0.92,
        minChildSize: 0.4,
        expand: false,
        builder: (_, scrollCtrl) => ListView(
          controller: scrollCtrl,
          padding: const EdgeInsets.all(20),
          children: [
            Center(
              child: Container(
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: Colors.grey.shade300,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
            const SizedBox(height: 20),
            Row(
              children: [
                Expanded(
                  child: Text(
                    a.title,
                    style: Theme.of(context).textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.w800,
                        ),
                  ),
                ),
                StatusChip.priority(a.priority),
              ],
            ),
            const SizedBox(height: 8),
            if (a.createdAt != null)
              Text(
                DateFormat('EEEE, MMMM d, yyyy · h:mm a').format(a.createdAt!),
                style: TextStyle(color: Colors.grey.shade500),
              ),
            const SizedBox(height: 18),
            const Divider(),
            const SizedBox(height: 12),
            Text(
              a.body,
              style: const TextStyle(fontSize: 15, height: 1.6),
            ),
          ],
        ),
      ),
    );
  }
}
