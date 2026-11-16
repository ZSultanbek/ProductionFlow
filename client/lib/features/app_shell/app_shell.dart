import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/routes.dart';
import '../../core/session.dart';
import '../../core/theme.dart';

/// The frame around every screen under /app: the two places a crew member
/// goes, as a bottom bar on a phone and a rail on a desktop — the same
/// destinations, put where the thumb or the cursor already is.
///
/// The router builds this once and swaps [child] as the address changes.
class AppShell extends StatelessWidget {
  const AppShell({super.key, required this.path, required this.child});

  /// The current address, used to highlight the right destination.
  final String path;
  final Widget child;

  static const List<_Destination> _destinations = [
    _Destination(
      label: 'Today',
      icon: Icons.wb_sunny_outlined,
      selectedIcon: Icons.wb_sunny_rounded,
      route: Routes.today,
    ),
    _Destination(
      label: 'Projects',
      icon: Icons.movie_outlined,
      selectedIcon: Icons.movie_rounded,
      route: Routes.projects,
    ),
  ];

  @override
  Widget build(BuildContext context) {
    // A project page and a call sheet both live under Projects.
    final index = path.startsWith(Routes.projects) ? 1 : 0;

    void select(int i) => context.go(_destinations[i].route);

    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < AppTheme.breakpoint) {
          return Scaffold(
            body: child,
            bottomNavigationBar: NavigationBar(
              selectedIndex: index,
              onDestinationSelected: select,
              destinations: [
                for (final d in _destinations)
                  NavigationDestination(
                    icon: Icon(d.icon),
                    selectedIcon: Icon(d.selectedIcon),
                    label: d.label,
                  ),
              ],
            ),
          );
        }

        return Scaffold(
          body: Row(
            children: [
              NavigationRail(
                selectedIndex: index,
                onDestinationSelected: select,
                labelType: NavigationRailLabelType.all,
                groupAlignment: -0.85,
                destinations: [
                  for (final d in _destinations)
                    NavigationRailDestination(
                      icon: Icon(d.icon),
                      selectedIcon: Icon(d.selectedIcon),
                      label: Text(d.label),
                      padding: const EdgeInsets.symmetric(vertical: AppTheme.space2),
                    ),
                ],
              ),
              VerticalDivider(
                width: 1,
                thickness: 1,
                color: Theme.of(context).colorScheme.outlineVariant,
              ),
              Expanded(child: child),
            ],
          ),
        );
      },
    );
  }
}

class _Destination {
  const _Destination({
    required this.label,
    required this.icon,
    required this.selectedIcon,
    required this.route,
  });

  final String label;
  final IconData icon;
  final IconData selectedIcon;
  final String route;
}

/// The account button in the top bar of the main screens: who is logged in,
/// and the way out.
class AccountButton extends ConsumerWidget {
  const AccountButton({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(sessionProvider)?.user;
    if (user == null) {
      return const SizedBox.shrink();
    }
    final theme = Theme.of(context);

    return PopupMenuButton<String>(
      tooltip: 'Account',
      icon: const Icon(Icons.account_circle_outlined),
      onSelected: (_) => ref.read(sessionProvider.notifier).signOut(),
      itemBuilder: (context) => [
        PopupMenuItem<String>(
          enabled: false,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(user.displayName, style: theme.textTheme.titleSmall),
              Text(user.email, style: theme.textTheme.bodySmall),
            ],
          ),
        ),
        const PopupMenuDivider(),
        const PopupMenuItem<String>(value: 'logout', child: Text('Log out')),
      ],
    );
  }
}
