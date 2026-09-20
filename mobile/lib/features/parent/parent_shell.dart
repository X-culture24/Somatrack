import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';

import '../../../core/providers/auth_provider.dart';
import '../../../core/theme/app_theme.dart';
import 'parent_home_page.dart';
import 'wards_page.dart';
import 'announcements_page.dart';
import 'fees_page.dart';
import 'profile_page.dart';

class ParentShell extends StatefulWidget {
  final Widget child;
  const ParentShell({super.key, required this.child});

  @override
  State<ParentShell> createState() => _ParentShellState();
}

class _ParentShellState extends State<ParentShell> {
  int _index = 0;

  final _pages = const <Widget>[
    ParentHomePage(),
    WardsPage(),
    AnnouncementsPage(),
    FeesPage(),
    ParentProfilePage(),
  ];

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final user = auth.user;
    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            const Icon(Icons.school, size: 22),
            const SizedBox(width: 8),
            Text(_titles[_index]),
          ],
        ),
        actions: [
          IconButton(
            onPressed: () {},
            icon: const Icon(Icons.notifications_outlined),
            tooltip: 'Notifications',
          ),
        ],
      ),
      body: IndexedStack(index: _index, children: _pages),
      drawer: Drawer(
        child: SafeArea(
          child: Column(
            children: [
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(18),
                color: AppTheme.primary,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    CircleAvatar(
                      radius: 28,
                      backgroundColor: Colors.white.withOpacity(0.2),
                      child: Text(
                        user?.initials ?? 'P',
                        style: const TextStyle(
                            color: Colors.white,
                            fontSize: 20,
                            fontWeight: FontWeight.w800),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      user?.fullName ?? 'Parent',
                      style: const TextStyle(
                          color: Colors.white,
                          fontSize: 16,
                          fontWeight: FontWeight.w700),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      user?.email ?? '',
                      style: TextStyle(
                          color: Colors.white.withOpacity(0.8), fontSize: 13),
                    ),
                    const SizedBox(height: 4),
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: AppTheme.accent,
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: const Text(
                        'PARENT PORTAL',
                        style: TextStyle(
                            fontSize: 10,
                            fontWeight: FontWeight.w800,
                            color: AppTheme.primaryDark),
                      ),
                    ),
                  ],
                ),
              ),
              ..._buildDrawerTiles(),
              const Spacer(),
              const Divider(),
              ListTile(
                leading: const Icon(Icons.logout, color: AppTheme.danger),
                title: const Text('Sign Out',
                    style: TextStyle(color: AppTheme.danger)),
                onTap: () async {
                  Navigator.pop(context);
                  await context.read<AuthProvider>().logout();
                  if (mounted) context.go('/login');
                },
              ),
              const SizedBox(height: 10),
            ],
          ),
        ),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (i) => setState(() => _index = i),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.home_outlined), selectedIcon: Icon(Icons.home), label: 'Home'),
          NavigationDestination(icon: Icon(Icons.family_restroom_outlined), selectedIcon: Icon(Icons.family_restroom), label: 'Children'),
          NavigationDestination(icon: Icon(Icons.campaign_outlined), selectedIcon: Icon(Icons.campaign), label: 'News'),
          NavigationDestination(icon: Icon(Icons.receipt_long_outlined), selectedIcon: Icon(Icons.receipt_long), label: 'Fees'),
          NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: 'Profile'),
        ],
      ),
    );
  }

  List<ListTile> _buildDrawerTiles() {
    return [
      ListTile(
        leading: const Icon(Icons.dashboard_outlined),
        title: const Text('Dashboard'),
        onTap: () {
          Navigator.pop(context);
          setState(() => _index = 0);
        },
      ),
      ListTile(
        leading: const Icon(Icons.menu_book_outlined),
        title: const Text('Academic Progress'),
        onTap: () => Navigator.pop(context),
      ),
      ListTile(
        leading: const Icon(Icons.directions_bus_outlined),
        title: const Text('Transport'),
        onTap: () => Navigator.pop(context),
      ),
      ListTile(
        leading: const Icon(Icons.menu_book_sharp),
        title: const Text('Library'),
        onTap: () => Navigator.pop(context),
      ),
      ListTile(
        leading: const Icon(Icons.chat_bubble_outline),
        title: const Text('Messages'),
        onTap: () => Navigator.pop(context),
      ),
    ];
  }
}

const _titles = [
  "St. Mary's Kabete",
  'My Children',
  'Announcements',
  'Fee Statements',
  'Profile',
];
