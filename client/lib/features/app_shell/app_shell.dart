import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme.dart';
import '../landing/landing_page.dart';

/// The product shell: navigation and nothing else yet.
///
/// The four destinations are the four things a crew member opens the app to do.
/// Each one's body is an empty state today; a feature folder replaces one body
/// at a time (`features/shooting_day/`, `features/change_feed/`, …).
///
/// Navigation is a rail on desktop and a bar on a phone — the same destinations,
/// placed where the hand or the cursor already is.
class AppShell extends StatefulWidget {
  const AppShell({super.key});

  static const String path = '/app';

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  int _index = 0;

  static const List<_Destination> _destinations = [
    _Destination(
      label: 'Days',
      icon: Icons.calendar_today_outlined,
      selectedIcon: Icons.calendar_today_rounded,
      emptyTitle: 'No shooting days yet',
      emptyBody:
          'A shooting day carries its call time, location, scenes and the people '
          'needed. Everything else in the app hangs off one.',
    ),
    _Destination(
      label: 'Scenes',
      icon: Icons.movie_outlined,
      selectedIcon: Icons.movie_rounded,
      emptyTitle: 'No scenes yet',
      emptyBody:
          'Scenes can sit unscheduled until they are attached to a day, so a '
          'breakdown does not have to be finished to be useful.',
    ),
    _Destination(
      label: 'Changes',
      icon: Icons.history_toggle_off_outlined,
      selectedIcon: Icons.history_rounded,
      emptyTitle: 'Nothing has changed yet',
      emptyBody:
          'Every confirmed change is appended here with who made it and what '
          'moved. The log is never edited or deleted.',
    ),
    _Destination(
      label: 'People',
      icon: Icons.people_outline_rounded,
      selectedIcon: Icons.people_rounded,
      emptyTitle: 'No one on the crew yet',
      emptyBody:
          'Producer, crew or cast. Who is needed in which scene is what makes '
          'the impact view possible.',
    ),
  ];

  @override
  Widget build(BuildContext context) {
    final destination = _destinations[_index];

    return LayoutBuilder(
      builder: (context, constraints) {
        final isWide = constraints.maxWidth >= AppTheme.breakpoint;

        final body = Scaffold(
          appBar: AppBar(
            title: Text(destination.label),
            leading: IconButton(
              icon: const Icon(Icons.arrow_back_rounded),
              tooltip: 'Back to start',
              onPressed: () => context.go(LandingPage.path),
            ),
          ),
          body: _EmptyState(
            icon: destination.icon,
            title: destination.emptyTitle,
            body: destination.emptyBody,
          ),
          bottomNavigationBar: isWide
              ? null
              : NavigationBar(
                  selectedIndex: _index,
                  onDestinationSelected: _select,
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

        if (!isWide) return body;

        return Scaffold(
          body: Row(
            children: [
              NavigationRail(
                selectedIndex: _index,
                onDestinationSelected: _select,
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
              Expanded(child: body),
            ],
          ),
        );
      },
    );
  }

  void _select(int index) => setState(() => _index = index);
}

class _Destination {
  const _Destination({
    required this.label,
    required this.icon,
    required this.selectedIcon,
    required this.emptyTitle,
    required this.emptyBody,
  });

  final String label;
  final IconData icon;
  final IconData selectedIcon;
  final String emptyTitle;
  final String emptyBody;
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.icon, required this.title, required this.body});

  final IconData icon;
  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 420),
        child: Padding(
          padding: const EdgeInsets.all(AppTheme.space6),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(icon, size: 24, color: theme.colorScheme.onSurfaceVariant),
              const SizedBox(height: AppTheme.space5),
              Text(title, style: theme.textTheme.headlineSmall),
              const SizedBox(height: AppTheme.space3),
              Text(
                body,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
