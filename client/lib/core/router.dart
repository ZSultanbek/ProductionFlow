import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../features/app_shell/app_shell.dart';
import '../features/landing/landing_page.dart';

/// Two routes for now.
///
///   `/`      the landing page — what this is, and one way in.
///   `/app`   the product shell. Features get nested under here as they land,
///            so add child routes rather than more top-level ones.
///
/// A function rather than a top-level `final`: a GoRouter holds navigation
/// state, and a shared instance leaks the current location from one widget test
/// into the next.
GoRouter createAppRouter() => GoRouter(
  initialLocation: LandingPage.path,
  routes: <RouteBase>[
    GoRoute(
      path: LandingPage.path,
      name: 'landing',
      builder: (context, state) => const LandingPage(),
    ),
    GoRoute(
      path: AppShell.path,
      name: 'app',
      builder: (context, state) => const AppShell(),
    ),
  ],
  errorBuilder: (context, state) => _RouteNotFound(location: state.uri.toString()),
);

class _RouteNotFound extends StatelessWidget {
  const _RouteNotFound({required this.location});

  final String location;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('No such page', style: theme.textTheme.headlineMedium),
              const SizedBox(height: 8),
              Text(location, style: theme.textTheme.bodySmall),
              const SizedBox(height: 24),
              FilledButton(
                onPressed: () => context.go(LandingPage.path),
                child: const Text('Back to start'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
