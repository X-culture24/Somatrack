import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/providers/auth_provider.dart';
import '../../../core/theme/app_theme.dart';

class TeacherProfilePage extends StatelessWidget {
  const TeacherProfilePage({super.key});

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final user = auth.user;
    return Scaffold(
      body: ListView(
        children: [
          Container(
            padding: const EdgeInsets.fromLTRB(16, 30, 16, 24),
            color: AppTheme.primary,
            child: Column(
              children: [
                CircleAvatar(
                  radius: 44,
                  backgroundColor: Colors.white.withOpacity(0.2),
                  child: Text(
                    user?.initials ?? 'T',
                    style: const TextStyle(
                        color: Colors.white,
                        fontSize: 32,
                        fontWeight: FontWeight.w800),
                  ),
                ),
                const SizedBox(height: 14),
                Text(
                  user?.fullName ?? 'Staff Member',
                  style: const TextStyle(
                      color: Colors.white,
                      fontSize: 20,
                      fontWeight: FontWeight.w800),
                ),
                const SizedBox(height: 4),
                Text(user?.email ?? '',
                    style: TextStyle(color: Colors.white.withOpacity(0.85))),
                const SizedBox(height: 10),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 5),
                  decoration: BoxDecoration(
                    color: AppTheme.accent,
                    borderRadius: BorderRadius.circular(5),
                  ),
                  child: Text(
                    user?.role.displayName ?? 'STAFF',
                    style: const TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.w800,
                        color: AppTheme.primaryDark),
                  ),
                ),
              ],
            ),
          ),
          _section('Staff Information'),
          _item(Icons.badge_outlined, 'TSC No.', '—'),
          _item(Icons.work_outline, 'Department', '—'),
          _item(Icons.class_outlined, 'Classes Assigned', ''),
          _item(Icons.phone_android_outlined, 'Mobile', user?.phoneNumber ?? '—'),
          _item(Icons.fingerprint, 'My Attendance',
              () => context.push('/teacher/attendance')),
          const SizedBox(height: 8),
          _section('Account'),
          _item(Icons.security_outlined, 'Change Password', () {}),
          _item(Icons.notifications_active_outlined, 'Notification Preferences', () {}),
          _item(Icons.language_outlined, 'Language / Locale', () {}),
          const SizedBox(height: 8),
          _section('Support'),
          _item(Icons.help_outline, 'Help & Documentation', () {}),
          _item(Icons.bug_report_outlined, 'Report an Issue', () {}),
          _item(Icons.info_outline, 'About the App', () {}),
          const SizedBox(height: 20),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: OutlinedButton.icon(
              onPressed: () async {
                await context.read<AuthProvider>().logout();
                if (context.mounted) context.go('/login');
              },
              style: OutlinedButton.styleFrom(
                foregroundColor: AppTheme.danger,
                side: const BorderSide(color: AppTheme.danger),
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(10)),
              ),
              icon: const Icon(Icons.logout),
              label: const Text('SIGN OUT',
                  style: TextStyle(fontWeight: FontWeight.w700)),
            ),
          ),
          const SizedBox(height: 26),
          const Center(
            child: Text(
              'ACK St. Mary\'s Kabete · Staff Portal v1.0',
              style: TextStyle(color: Colors.grey, fontSize: 12),
            ),
          ),
          const SizedBox(height: 20),
        ],
      ),
    );
  }

  Widget _section(String title) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 14, 16, 4),
      child: Text(
        title.toUpperCase(),
        style: TextStyle(
            fontSize: 11,
            color: Colors.grey.shade500,
            fontWeight: FontWeight.w800,
            letterSpacing: 0.5),
      ),
    );
  }

  Widget _item(IconData icon, String title, Object? trailing) {
    final VoidCallback? onTap =
        trailing is VoidCallback ? trailing : null;
    final String? value = trailing is String ? trailing : null;
    return ListTile(
      leading: Icon(icon, color: AppTheme.primary),
      title: Text(title, style: const TextStyle(fontWeight: FontWeight.w600)),
      trailing: onTap != null
          ? Icon(Icons.chevron_right, color: Colors.grey.shade400)
          : (value != null
              ? Text(value,
                  style: TextStyle(
                      color: Colors.grey.shade600,
                      fontWeight: FontWeight.w500))
              : null),
      onTap: onTap,
    );
  }
}
