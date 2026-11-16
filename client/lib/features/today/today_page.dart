import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/dates.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/routes.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';
import '../app_shell/app_shell.dart';

/// The first screen after opening the app (US-05): where do I need to be, and
/// when. Today's shooting days across every project I am in — or, if today is
/// free, the next day that is not. Never an empty screen while something is
/// still ahead.
class TodayPage extends ConsumerWidget {
  const TodayPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final today = ref.watch(todayProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Today'),
        actions: [
          IconButton(
            tooltip: 'Refresh',
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () => ref.invalidate(todayProvider),
          ),
          const AccountButton(),
          const SizedBox(width: AppTheme.space2),
        ],
      ),
      body: AsyncView<Today>(
        value: today,
        onRetry: () => ref.invalidate(todayProvider),
        builder: (data) {
          final date = data.date;
          if (date == null) {
            return MessageState(
              icon: Icons.event_available_outlined,
              title: 'Nothing scheduled',
              body:
                  'None of your projects has a shooting day coming up. When a '
                  'producer adds one, it shows here.',
              action: OutlinedButton(
                onPressed: () => context.go(Routes.projects),
                child: const Text('Open projects'),
              ),
            );
          }
          return PageBody(
            onRefresh: () async {
              try {
                // Completes when the reload does, which is what tells the
                // pull-to-refresh spinner to go away.
                ref.invalidate(todayProvider);
                await ref.read(todayProvider.future);
              } on Object {
                // Nothing to do here: the failed state is shown by AsyncView.
              }
            },
            children: [
              _DateHeader(date: date, isToday: data.isToday),
              const SizedBox(height: AppTheme.space5),
              for (final day in data.days) ...[
                _AgendaCard(day: day),
                const SizedBox(height: AppTheme.space4),
              ],
            ],
          );
        },
      ),
    );
  }
}

class _DateHeader extends StatelessWidget {
  const _DateHeader({required this.date, required this.isToday});

  final String date;
  final bool isToday;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Eyebrow(isToday ? 'Shooting today' : 'Nothing today · next shooting day'),
        const SizedBox(height: AppTheme.space2),
        Text(formatShootDate(date), style: theme.textTheme.headlineLarge),
      ],
    );
  }
}

/// One shooting day, as it concerns me: the call time first and largest,
/// because it is the one thing nobody can afford to get wrong.
class _AgendaCard extends StatelessWidget {
  const _AgendaCard({required this.day});

  final AgendaDay day;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => context.go(Routes.day(day.projectId, day.id)),
        child: Padding(
          padding: const EdgeInsets.all(AppTheme.space5),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Call time',
                          style: theme.textTheme.bodySmall,
                        ),
                        // The accent, used once per card: the call time.
                        Text(
                          day.callTime,
                          style: theme.textTheme.displaySmall?.copyWith(color: scheme.tertiary),
                        ),
                      ],
                    ),
                  ),
                  Icon(Icons.chevron_right_rounded, color: scheme.onSurfaceVariant),
                ],
              ),
              const SizedBox(height: AppTheme.space3),
              Text(
                day.location.isEmpty ? 'Location not set yet' : day.location,
                style: theme.textTheme.titleMedium,
              ),
              const SizedBox(height: AppTheme.space1),
              Text(day.projectName, style: theme.textTheme.bodySmall),

              if (day.iAmOnSetAllDay || day.myScenes.isNotEmpty) ...[
                const Divider(),
                if (day.iAmOnSetAllDay)
                  Padding(
                    padding: const EdgeInsets.only(bottom: AppTheme.space3),
                    child: Text('You are on set all day', style: theme.textTheme.titleSmall),
                  ),
                if (day.myScenes.isNotEmpty) ...[
                  Text('Your scenes', style: theme.textTheme.bodySmall),
                  const SizedBox(height: AppTheme.space2),
                  for (final scene in day.myScenes)
                    Padding(
                      padding: const EdgeInsets.only(bottom: AppTheme.space2),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          SizedBox(
                            width: 48,
                            child: Text(scene.number, style: theme.textTheme.titleSmall),
                          ),
                          Expanded(
                            child: Text(
                              scene.description,
                              style: theme.textTheme.bodyMedium?.copyWith(
                                color: scheme.onSurfaceVariant,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                ],
              ],
            ],
          ),
        ),
      ),
    );
  }
}
