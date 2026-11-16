import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/dates.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/routes.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';
import '../shooting_day/add_day_dialog.dart';

/// One production: its shooting days, in order (US-04). A producer also gets
/// the invite link (US-02) and the button to add a day; crew and cast see the
/// same schedule without either.
class ProjectPage extends ConsumerWidget {
  const ProjectPage({super.key, required this.projectId});

  final String projectId;

  Future<void> _addDay(BuildContext context, WidgetRef ref) async {
    final day = await showDialog<ShootingDay>(
      context: context,
      builder: (context) => AddDayDialog(projectId: projectId),
    );
    if (day == null || !context.mounted) {
      return;
    }
    // The new day belongs in this list, and it may be today's.
    ref.invalidate(daysProvider(projectId));
    ref.invalidate(todayProvider);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final project = ref.watch(projectProvider(projectId));

    return Scaffold(
      appBar: AppBar(title: Text(project.value?.name ?? 'Project')),
      floatingActionButton: project.value?.iAmProducer == true
          ? FloatingActionButton.extended(
              onPressed: () => _addDay(context, ref),
              icon: const Icon(Icons.add_rounded),
              label: const Text('Add shooting day'),
            )
          : null,
      body: AsyncView<Project>(
        value: project,
        onRetry: () => ref.invalidate(projectProvider(projectId)),
        builder: (data) => _ProjectBody(project: data),
      ),
    );
  }
}

class _ProjectBody extends ConsumerWidget {
  const _ProjectBody({required this.project});

  final Project project;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final days = ref.watch(daysProvider(project.id));
    final inviteCode = project.inviteCode;

    return PageBody(
      onRefresh: () async {
        try {
          ref.invalidate(daysProvider(project.id));
          await ref.read(daysProvider(project.id).future);
        } on Object {
          // The failed state is shown below, in place of the list.
        }
      },
      children: [
        Eyebrow('You are ${project.myRole}'),
        const SizedBox(height: AppTheme.space2),
        Text(project.name, style: theme.textTheme.headlineLarge),
        const SizedBox(height: AppTheme.space5),

        if (inviteCode != null) ...[
          _InviteCard(link: inviteLink(inviteCode, project.inviteUrl)),
          const SizedBox(height: AppTheme.space6),
        ],

        Text('Shooting days', style: theme.textTheme.headlineSmall),
        const SizedBox(height: AppTheme.space4),

        // The days load separately from the project, so the heading and the
        // invite link are already on screen while the list arrives.
        days.when(
          loading: () => const Padding(
            padding: EdgeInsets.symmetric(vertical: AppTheme.space6),
            child: Center(child: CircularProgressIndicator()),
          ),
          error: (error, _) => Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              FormError(describeError(error)),
              const SizedBox(height: AppTheme.space3),
              OutlinedButton(
                onPressed: () => ref.invalidate(daysProvider(project.id)),
                child: const Text('Try again'),
              ),
            ],
          ),
          data: (list) {
            if (list.isEmpty) {
              return Text(
                project.iAmProducer
                    ? 'No shooting days yet. Add the first one — everything '
                          'else in the app hangs off a day.'
                    : 'No shooting days yet. They show up here as soon as the '
                          'producer adds one.',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              );
            }
            return Column(
              children: [
                for (final day in list) ...[
                  _DayCard(day: day),
                  const SizedBox(height: AppTheme.space3),
                ],
              ],
            );
          },
        ),

        // Room for the floating button.
        const SizedBox(height: AppTheme.space8),
      ],
    );
  }
}

/// The link a producer sends to the crew. In a browser it is built from the
/// address the app is open at, so it is right on localhost and in production
/// alike; on a phone there is no such address, so the server's is used.
String inviteLink(String code, String? fromServer) {
  if (kIsWeb) {
    return '${Uri.base.origin}${Routes.join(code)}';
  }
  return fromServer ?? Routes.join(code);
}

class _InviteCard extends StatelessWidget {
  const _InviteCard({required this.link});

  final String link;

  Future<void> _copy(BuildContext context) async {
    await Clipboard.setData(ClipboardData(text: link));
    if (!context.mounted) {
      return;
    }
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Invite link copied')),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.space5),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Invite your crew', style: theme.textTheme.titleMedium),
            const SizedBox(height: AppTheme.space2),
            Text(
              'Anyone with this link can join as crew or cast. Send it in the '
              'chat you already use.',
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: AppTheme.space4),
            SelectableText(link, style: theme.textTheme.bodyMedium),
            const SizedBox(height: AppTheme.space4),
            OutlinedButton.icon(
              onPressed: () => _copy(context),
              icon: const Icon(Icons.copy_rounded, size: 18),
              label: const Text('Copy link'),
            ),
          ],
        ),
      ),
    );
  }
}

class _DayCard extends StatelessWidget {
  const _DayCard({required this.day});

  final ShootingDay day;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final scenes = switch (day.sceneCount) {
      0 => 'No scenes yet',
      1 => '1 scene',
      final n => '$n scenes',
    };

    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => context.go(Routes.day(day.projectId, day.id)),
        child: Padding(
          padding: const EdgeInsets.all(AppTheme.space5),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '${formatShootDate(day.shootDate)} · call ${day.callTime}',
                      style: theme.textTheme.titleMedium,
                    ),
                    const SizedBox(height: AppTheme.space1),
                    Text(
                      day.location.isEmpty ? 'Location not set yet' : day.location,
                      style: theme.textTheme.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
                    ),
                    const SizedBox(height: AppTheme.space1),
                    Text(scenes, style: theme.textTheme.bodySmall),
                  ],
                ),
              ),
              Icon(Icons.chevron_right_rounded, color: scheme.onSurfaceVariant),
            ],
          ),
        ),
      ),
    );
  }
}
