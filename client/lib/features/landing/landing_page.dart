import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/routes.dart';
import '../../core/theme.dart';

/// The public face of the product: what it does for a crew, and one way in.
///
/// Two deliberate layouts, not one layout that shrinks. Above
/// [AppTheme.breakpoint] the hero is two columns with the impact card beside the
/// pitch; below it the page is a single column and the impact card comes after
/// the call to action, because on a phone the words have to land first.
class LandingPage extends StatelessWidget {
  const LandingPage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final isWide = constraints.maxWidth >= AppTheme.breakpoint;
            return SingleChildScrollView(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: AppTheme.maxContentWidth),
                  child: Padding(
                    padding: EdgeInsets.symmetric(
                      horizontal: isWide ? AppTheme.space7 : AppTheme.space5,
                      vertical: isWide ? AppTheme.space6 : AppTheme.space5,
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        const _TopBar(),
                        SizedBox(height: isWide ? AppTheme.space8 : AppTheme.space7),
                        _Hero(isWide: isWide),
                        SizedBox(height: isWide ? AppTheme.space8 : AppTheme.space7),
                        _HowItWorks(isWide: isWide),
                        SizedBox(height: isWide ? AppTheme.space8 : AppTheme.space7),
                        const _Footer(),
                      ],
                    ),
                  ),
                ),
              ),
            );
          },
        ),
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  const _TopBar();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        // Expanded, so on a narrow phone the name gives way before the row
        // overflows.
        const Expanded(child: _Wordmark()),
        // Goes to the app; the router sends anyone not logged in to the login
        // screen first.
        TextButton(
          onPressed: () => context.go(Routes.today),
          child: Text('Log in', style: theme.textTheme.labelLarge),
        ),
      ],
    );
  }
}

class _Wordmark extends StatelessWidget {
  const _Wordmark();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        // A flat mark, not an icon in a tinted box: 8px of solid primary.
        Container(
          width: 10,
          height: 10,
          decoration: BoxDecoration(
            color: theme.colorScheme.primary,
            borderRadius: BorderRadius.circular(3),
          ),
        ),
        const SizedBox(width: AppTheme.space3),
        Flexible(
          child: Text(
            'ProductionFlow',
            style: theme.textTheme.headlineSmall,
            maxLines: 1,
            softWrap: false,
            overflow: TextOverflow.fade,
          ),
        ),
      ],
    );
  }
}

class _Hero extends StatelessWidget {
  const _Hero({required this.isWide});

  final bool isWide;

  @override
  Widget build(BuildContext context) {
    if (!isWide) {
      return const Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _Pitch(isWide: false),
          SizedBox(height: AppTheme.space6),
          ImpactPreviewCard(),
        ],
      );
    }

    return const Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(flex: 6, child: _Pitch(isWide: true)),
        SizedBox(width: AppTheme.space8),
        Expanded(flex: 5, child: ImpactPreviewCard()),
      ],
    );
  }
}

class _Pitch extends StatelessWidget {
  const _Pitch({required this.isWide});

  final bool isWide;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'FOR SMALL FILM CREWS',
          style: theme.textTheme.labelSmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
        ),
        const SizedBox(height: AppTheme.space4),
        Text(
          'A shooting day moves.\nKnow who that hits.',
          style: isWide ? theme.textTheme.displayMedium : theme.textTheme.displaySmall,
        ),
        const SizedBox(height: AppTheme.space5),
        ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Text(
            'Your schedule lives in a group chat and three versions of a '
            'spreadsheet. ProductionFlow attaches every scene, every person and '
            'every file to a shooting day — so when the day changes, you see '
            'exactly which scenes and which people are affected before you '
            'confirm it, and only they get told.',
            style: theme.textTheme.bodyLarge?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
        ),
        const SizedBox(height: AppTheme.space6),
        // One call to action on the page. Full width on a phone, natural width
        // on a desktop, where a stretched button reads as a banner.
        Align(
          alignment: Alignment.centerLeft,
          child: SizedBox(
            width: isWide ? null : double.infinity,
            child: FilledButton.icon(
              onPressed: () => context.go(Routes.today),
              icon: const Icon(Icons.arrow_forward_rounded, size: 20),
              iconAlignment: IconAlignment.end,
              label: const Text('Open ProductionFlow'),
            ),
          ),
        ),
        const SizedBox(height: AppTheme.space4),
        Text(
          'Free for small crews. Web and Android.',
          style: theme.textTheme.bodySmall,
        ),
      ],
    );
  }
}

/// The product in one picture: a change, and what it touches.
///
/// The only place on this page that uses [ColorScheme.tertiary] — the accent
/// marks the single idea the product is built around.
class ImpactPreviewCard extends StatelessWidget {
  const ImpactPreviewCard({super.key});

  static const List<(String, String)> _scenes = [
    ('12A', 'Rooftop argument, dusk'),
    ('12B', 'Stairwell, she leaves'),
    ('13', 'Insert: the note on the table'),
  ];

  static const List<(String, String)> _people = [
    ('Aisha', 'cast'),
    ('Daniyar', 'crew'),
    ('Ruslan', 'crew'),
  ];

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.space5),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    'Day 4 · Kok-Tobe',
                    style: theme.textTheme.titleMedium,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                const SizedBox(width: AppTheme.space3),
                Text('preview', style: theme.textTheme.bodySmall),
              ],
            ),
            const SizedBox(height: AppTheme.space4),

            // The change itself, as a diff. Struck-through old value, new value
            // in full contrast — no colour needed to read it.
            Container(
              padding: const EdgeInsets.symmetric(
                horizontal: AppTheme.space4,
                vertical: AppTheme.space3,
              ),
              decoration: BoxDecoration(
                color: scheme.surfaceContainerHigh,
                borderRadius: BorderRadius.circular(AppTheme.radiusControl),
              ),
              child: Row(
                children: [
                  Text(
                    '4 Oct, 07:30',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      decoration: TextDecoration.lineThrough,
                      color: scheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(width: AppTheme.space3),
                  Icon(Icons.east_rounded, size: 16, color: scheme.onSurfaceVariant),
                  const SizedBox(width: AppTheme.space3),
                  Expanded(
                    child: Text('6 Oct, 08:00', style: theme.textTheme.titleSmall),
                  ),
                ],
              ),
            ),

            const Divider(),

            // The accent, used once.
            Container(
              padding: const EdgeInsets.symmetric(
                horizontal: AppTheme.space3,
                vertical: AppTheme.space1 + 2,
              ),
              decoration: BoxDecoration(
                color: scheme.tertiaryContainer,
                borderRadius: BorderRadius.circular(AppTheme.radiusControl),
              ),
              child: Text(
                'AFFECTS 3 SCENES · 3 PEOPLE',
                style: theme.textTheme.labelSmall?.copyWith(
                  color: scheme.onTertiaryContainer,
                ),
              ),
            ),
            const SizedBox(height: AppTheme.space4),

            for (final (number, description) in _scenes)
              Padding(
                padding: const EdgeInsets.only(bottom: AppTheme.space3),
                child: Row(
                  children: [
                    SizedBox(
                      width: 42,
                      child: Text(number, style: theme.textTheme.titleSmall),
                    ),
                    Expanded(
                      child: Text(
                        description,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: scheme.onSurfaceVariant,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
              ),

            const SizedBox(height: AppTheme.space2),
            Wrap(
              spacing: AppTheme.space2,
              runSpacing: AppTheme.space2,
              children: [
                for (final (name, role) in _people)
                  Chip(
                    label: Text('$name · $role'),
                    visualDensity: VisualDensity.compact,
                  ),
              ],
            ),
            const SizedBox(height: AppTheme.space5),
            Text(
              'Nothing is written until you confirm. Only these three get the '
              'notification.',
              style: theme.textTheme.bodySmall,
            ),
          ],
        ),
      ),
    );
  }
}

class _HowItWorks extends StatelessWidget {
  const _HowItWorks({required this.isWide});

  final bool isWide;

  static const List<(IconData, String, String)> _points = [
    (
      Icons.event_note_rounded,
      'The day is the unit',
      'Call time, location, scenes, who is needed, which script version. One '
          'page per shooting day instead of four places to check.',
    ),
    (
      Icons.podcasts_rounded,
      'Changes are previewed',
      'Move a day or reassign a scene and read the blast radius first: which '
          'scenes, which people, which materials. Then commit it.',
    ),
    (
      Icons.notifications_active_rounded,
      'Only the affected are told',
      'No 40-message group chat where everyone skims. The three people whose '
          'day actually changed are the three people notified.',
    ),
  ];

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    final heading = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('How it works', style: theme.textTheme.headlineLarge),
        const SizedBox(height: AppTheme.space5),
      ],
    );

    final cards = [
      for (final (icon, title, body) in _points)
        _PointCard(icon: icon, title: title, body: body),
    ];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        heading,
        if (isWide)
          // IntrinsicHeight so the three cards share a height and their bottom
          // edges line up. A bare stretch would ask for infinite height inside
          // the scroll view.
          IntrinsicHeight(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                for (var i = 0; i < cards.length; i++) ...[
                  if (i > 0) const SizedBox(width: AppTheme.space4),
                  Expanded(child: cards[i]),
                ],
              ],
            ),
          )
        else
          Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              for (var i = 0; i < cards.length; i++) ...[
                if (i > 0) const SizedBox(height: AppTheme.space4),
                cards[i],
              ],
            ],
          ),
      ],
    );
  }
}

class _PointCard extends StatelessWidget {
  const _PointCard({required this.icon, required this.title, required this.body});

  final IconData icon;
  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.space5),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, size: 22, color: theme.colorScheme.primary),
            const SizedBox(height: AppTheme.space4),
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
    );
  }
}

class _Footer extends StatelessWidget {
  const _Footer();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Divider(color: theme.colorScheme.outlineVariant),
        const SizedBox(height: AppTheme.space4),
        Text(
          'KBTU · IT Project Management · Fall 2026',
          style: theme.textTheme.bodySmall,
        ),
      ],
    );
  }
}
