import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../core/providers/auth_provider.dart';
import '../../core/api/api_client.dart';
import '../../core/theme/app_theme.dart';

/// Shared by both the teacher and parent profile screens — was a no-op
/// "Change Password" menu item on each before AuthProvider.changePassword
/// (and the real backend behind it) existed.
Future<void> showChangePasswordDialog(BuildContext context) async {
  final currentCtrl = TextEditingController();
  final newCtrl = TextEditingController();
  final confirmCtrl = TextEditingController();
  String? error;

  await showDialog(
    context: context,
    builder: (dialogContext) => StatefulBuilder(
      builder: (dialogContext, setState) => AlertDialog(
        title: const Text('Change Password'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (error != null) ...[
              Text(error!, style: const TextStyle(color: AppTheme.danger, fontSize: 13)),
              const SizedBox(height: 10),
            ],
            TextField(
              controller: currentCtrl,
              obscureText: true,
              decoration: const InputDecoration(labelText: 'Current password'),
            ),
            const SizedBox(height: 10),
            TextField(
              controller: newCtrl,
              obscureText: true,
              decoration: const InputDecoration(labelText: 'New password (min 8 characters)'),
            ),
            const SizedBox(height: 10),
            TextField(
              controller: confirmCtrl,
              obscureText: true,
              decoration: const InputDecoration(labelText: 'Confirm new password'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(dialogContext), child: const Text('Cancel')),
          FilledButton(
            onPressed: () async {
              if (newCtrl.text != confirmCtrl.text) {
                setState(() => error = "New passwords don't match.");
                return;
              }
              if (newCtrl.text.length < 8) {
                setState(() => error = 'New password must be at least 8 characters.');
                return;
              }
              try {
                await dialogContext.read<AuthProvider>().changePassword(
                      currentPassword: currentCtrl.text,
                      newPassword: newCtrl.text,
                    );
                if (dialogContext.mounted) {
                  Navigator.pop(dialogContext);
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Password changed successfully.'), backgroundColor: AppTheme.success),
                  );
                }
              } on ApiException catch (e) {
                setState(() => error = e.message);
              } catch (e) {
                setState(() => error = 'Could not change password: $e');
              }
            },
            child: const Text('Change Password'),
          ),
        ],
      ),
    ),
  );
}

class StatusChip extends StatelessWidget {
  final String label;
  final Color? color;
  final Color? textColor;
  final IconData? icon;

  const StatusChip({
    super.key,
    required this.label,
    this.color,
    this.textColor,
    this.icon,
  });

  factory StatusChip.attendance(String status, {double fontSize = 12}) {
    switch (status) {
      case 'present':
        return StatusChip(
          label: 'Present',
          color: AppTheme.success.withOpacity(0.12),
          textColor: AppTheme.success,
          icon: Icons.check_circle,
        );
      case 'late':
        return StatusChip(
          label: 'Late',
          color: AppTheme.warning.withOpacity(0.15),
          textColor: AppTheme.warning,
          icon: Icons.schedule,
        );
      case 'absent':
        return StatusChip(
          label: 'Absent',
          color: AppTheme.danger.withOpacity(0.12),
          textColor: AppTheme.danger,
          icon: Icons.cancel,
        );
      default:
        return StatusChip(
          label: status,
          color: Colors.grey.shade100,
          textColor: Colors.grey.shade700,
        );
    }
  }

  factory StatusChip.invoice(String status) {
    switch (status) {
      case 'paid':
      case 'full':
        return StatusChip(
          label: 'Paid',
          color: AppTheme.success.withOpacity(0.12),
          textColor: AppTheme.success,
          icon: Icons.check_circle,
        );
      case 'partial':
        return StatusChip(
          label: 'Partial',
          color: AppTheme.warning.withOpacity(0.15),
          textColor: AppTheme.warning,
          icon: Icons.payments,
        );
      case 'unpaid':
      default:
        return StatusChip(
          label: 'Unpaid',
          color: AppTheme.danger.withOpacity(0.12),
          textColor: AppTheme.danger,
          icon: Icons.warning_amber,
        );
    }
  }

  factory StatusChip.priority(String priority) {
    switch (priority) {
      case 'high':
      case 'urgent':
        return StatusChip(
          label: 'High',
          color: AppTheme.danger.withOpacity(0.12),
          textColor: AppTheme.danger,
        );
      case 'normal':
      default:
        return StatusChip(
          label: 'Normal',
          color: AppTheme.info.withOpacity(0.12),
          textColor: AppTheme.info,
        );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color ?? Colors.grey.shade100,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 14, color: textColor ?? Colors.grey.shade700),
            const SizedBox(width: 4),
          ],
          Text(
            label,
            style: TextStyle(
              fontWeight: FontWeight.w600,
              color: textColor ?? Colors.grey.shade700,
            ),
          ),
        ],
      ),
    );
  }
}

class StatCard extends StatelessWidget {
  final String title;
  final String value;
  final IconData icon;
  final Color color;
  final double? percent;

  const StatCard({
    super.key,
    required this.title,
    required this.value,
    required this.icon,
    required this.color,
    this.percent,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: color.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Icon(icon, size: 20, color: color),
                ),
                const Spacer(),
                if (percent != null)
                  Text(
                    '${percent!.toStringAsFixed(0)}%',
                    style: TextStyle(
                      color: color,
                      fontWeight: FontWeight.w700,
                      fontSize: 13,
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 12),
            Text(
              value,
              style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w800,
                  ),
            ),
            const SizedBox(height: 4),
            Text(
              title,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Colors.grey.shade600,
                    fontWeight: FontWeight.w500,
                  ),
            ),
          ],
        ),
      ),
    );
  }
}

class AvatarCircle extends StatelessWidget {
  final String initials;
  final String? name;
  final double radius;
  final Color? backgroundColor;

  const AvatarCircle({
    super.key,
    required this.initials,
    this.name,
    this.radius = 22,
    this.backgroundColor,
  });

  @override
  Widget build(BuildContext context) {
    final colors = [
      AppTheme.primary,
      AppTheme.info,
      AppTheme.warning,
      AppTheme.danger,
      Colors.purple,
      Colors.teal,
    ];
    int idx = 0;
    if (name != null && name!.isNotEmpty) {
      idx = name!.codeUnitAt(0) % colors.length;
    }
    final bg = backgroundColor ?? colors[idx];
    return CircleAvatar(
      radius: radius,
      backgroundColor: bg,
      child: Text(
        initials,
        style: TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.w700,
          fontSize: radius * 0.8,
        ),
      ),
    );
  }
}

class SectionHeader extends StatelessWidget {
  final String title;
  final String? actionLabel;
  final VoidCallback? onAction;

  const SectionHeader({
    super.key,
    required this.title,
    this.actionLabel,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          Text(
            title,
            style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
          ),
          const Spacer(),
          if (actionLabel != null && onAction != null)
            TextButton(
              onPressed: onAction,
              child: Text(actionLabel!),
            ),
        ],
      ),
    );
  }
}

class EmptyState extends StatelessWidget {
  final String message;
  final IconData icon;

  const EmptyState({
    super.key,
    required this.message,
    this.icon = Icons.inbox_outlined,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(40),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 56, color: Colors.grey.shade400),
            const SizedBox(height: 14),
            Text(
              message,
              textAlign: TextAlign.center,
              style: TextStyle(color: Colors.grey.shade600, fontSize: 14),
            ),
          ],
        ),
      ),
    );
  }
}
