import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'core/router.dart';
import 'core/session.dart';
import 'core/theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Real paths on the web (/join/abc123, not /#/join/abc123), so an invite
  // link looks like a link and survives being pasted into a chat. Does nothing
  // on Android.
  usePathUrlStrategy();

  // Loaded before the first frame so the router knows straight away whether
  // someone is logged in, instead of flashing the login screen at them.
  final prefs = await SharedPreferences.getInstance();

  runApp(
    ProviderScope(
      // Riverpod retries a failed provider on its own by default. Here a
      // failure is shown to the user with a "Try again" button instead: a 404
      // does not get better by asking ten more times.
      retry: (retryCount, error) => null,
      overrides: [sessionStoreProvider.overrideWithValue(PrefsSessionStore(prefs))],
      child: const ProductionFlowApp(),
    ),
  );
}

class ProductionFlowApp extends ConsumerWidget {
  const ProductionFlowApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      title: 'ProductionFlow',
      debugShowCheckedModeBanner: false,
      routerConfig: ref.watch(routerProvider),
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      // Follow the device. Both palettes are designed, so neither is a fallback.
      themeMode: ThemeMode.system,
    );
  }
}
