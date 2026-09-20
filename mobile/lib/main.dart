import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';

import 'core/providers/auth_provider.dart';
import 'core/providers/attendance_provider.dart';
import 'core/theme/app_theme.dart';
import 'core/routes/app_router.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const StMarysApp());
}

class StMarysApp extends StatelessWidget {
  const StMarysApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => AuthProvider()..init()),
        ChangeNotifierProxyProvider<AuthProvider, AttendanceProvider>(
          create: (_) => AttendanceProvider(),
          update: (_, auth, previous) {
            return (previous ?? AttendanceProvider())..updateAuth(auth);
          },
        ),
      ],
      child: const AppShell(),
    );
  }
}

class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  late GoRouter _router;

  @override
  void initState() {
    super.initState();
    _router = AppRouter.create(context.read<AuthProvider>());
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'ACK St. Mary\'s Kabete',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      routerConfig: _router,
    );
  }
}
