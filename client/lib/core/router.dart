import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/app_shell/app_shell.dart';
import '../features/auth/auth_pages.dart';
import '../features/invite/join_page.dart';
import '../features/landing/landing_page.dart';
import '../features/projects/project_page.dart';
import '../features/projects/projects_page.dart';
import '../features/shooting_day/call_sheet_page.dart';
import '../features/today/today_page.dart';
import 'routes.dart';
import 'session.dart';

/// The router. It re-checks its redirect whenever the session changes, which
/// is the whole login flow: sign in, and the redirect moves you into the app;
/// sign out (or have the server reject the token), and it moves you out.
///
///   /                              landing (public)
///   /login, /register              public
///   /join/:code                    invite link (public; joining needs login)
///   /app                           today
///   /app/projects                  my projects
///   /app/projects/:id              one project: days, invite link
///   /app/projects/:id/days/:dayId  the call sheet
final routerProvider = Provider<GoRouter>((ref) {
  // go_router wants a Listenable to know when to re-run redirect; this turns
  // "the session provider changed" into one.
  final sessionChanged = ValueNotifier<int>(0);
  ref.listen(sessionProvider, (_, _) => sessionChanged.value++);

  final router = GoRouter(
    initialLocation: Routes.landing,
    refreshListenable: sessionChanged,
    redirect: (context, state) {
      final loggedIn = ref.read(sessionProvider) != null;
      final path = state.uri.path;
      final from = safeFrom(state.uri.queryParameters['from']);

      final inApp = path == Routes.today || path.startsWith('${Routes.today}/');
      if (inApp && !loggedIn) {
        return Routes.withFrom(Routes.login, state.uri.toString());
      }

      final onAuthPage = path == Routes.login || path == Routes.register;
      if (onAuthPage && loggedIn) {
        return from ?? Routes.today;
      }

      return null;
    },
    routes: [
      GoRoute(path: Routes.landing, builder: (context, state) => const LandingPage()),
      GoRoute(
        path: Routes.login,
        builder: (context, state) =>
            LoginPage(from: safeFrom(state.uri.queryParameters['from'])),
      ),
      GoRoute(
        path: Routes.register,
        builder: (context, state) =>
            RegisterPage(from: safeFrom(state.uri.queryParameters['from'])),
      ),
      GoRoute(
        path: '/join/:code',
        builder: (context, state) => JoinPage(code: state.pathParameters['code']!),
      ),

      // Everything under /app shares the navigation bar or rail.
      ShellRoute(
        builder: (context, state, child) => AppShell(path: state.uri.path, child: child),
        routes: [
          GoRoute(path: Routes.today, builder: (context, state) => const TodayPage()),
          GoRoute(
            path: Routes.projects,
            builder: (context, state) => const ProjectsPage(),
            routes: [
              GoRoute(
                path: ':projectId',
                builder: (context, state) =>
                    ProjectPage(projectId: state.pathParameters['projectId']!),
                routes: [
                  GoRoute(
                    path: 'days/:dayId',
                    builder: (context, state) => CallSheetPage(
                      projectId: state.pathParameters['projectId']!,
                      dayId: state.pathParameters['dayId']!,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
    ],
    errorBuilder: (context, state) => _RouteNotFound(location: state.uri.toString()),
  );

  ref.onDispose(() {
    router.dispose();
    sessionChanged.dispose();
  });
  return router;
});

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
                onPressed: () => context.go(Routes.landing),
                child: const Text('Back to start'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
