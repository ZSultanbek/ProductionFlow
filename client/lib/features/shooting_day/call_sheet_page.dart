import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/dates.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';

/// The call sheet for one shooting day (US-05): when, where, which scenes and
/// who is needed. The same page for everyone on the project — it replaces the
/// three versions of the spreadsheet.
class CallSheetPage extends ConsumerWidget {
  const CallSheetPage({super.key, required this.projectId, required this.dayId});

  /// Part of the address so the back arrow leads to the project. The day
  /// itself is looked up by [dayId] alone.
  final String projectId;
  final String dayId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final sheet = ref.watch(callSheetProvider(dayId));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Call sheet'),
        actions: [
          IconButton(
            tooltip: 'Refresh',
            icon: const Icon(Icons.refresh_rounded),
            onPressed: () => ref.invalidate(callSheetProvider(dayId)),
          ),
          const SizedBox(width: AppTheme.space2),
        ],
      ),
      body: AsyncView<CallSheet>(
        value: sheet,
        onRetry: () => ref.invalidate(callSheetProvider(dayId)),
        builder: (data) => PageBody(
          onRefresh: () async {
            try {
              ref.invalidate(callSheetProvider(dayId));
              await ref.read(callSheetProvider(dayId).future);
            } on Object {
              // The failed state is shown by AsyncView.
            }
          },
          children: [
            _Header(sheet: data),
            const SizedBox(height: AppTheme.space6),
            if (data.onSetAllDay.isNotEmpty) ...[
              _OnSetAllDay(people: data.onSetAllDay),
              const SizedBox(height: AppTheme.space6),
            ],
            _Scenes(scenes: data.scenes),
            const SizedBox(height: AppTheme.space7),
          ],
        ),
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.sheet});

  final CallSheet sheet;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Eyebrow(sheet.projectName),
        const SizedBox(height: AppTheme.space2),
        Text(formatShootDate(sheet.shootDate), style: theme.textTheme.headlineLarge),
        const SizedBox(height: AppTheme.space5),
        Text('Call time', style: theme.textTheme.bodySmall),
        // The accent, used once on this page: the call time.
        Text(
          sheet.callTime,
          style: theme.textTheme.displayMedium?.copyWith(color: scheme.tertiary),
        ),
        const SizedBox(height: AppTheme.space4),
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(Icons.place_outlined, size: 20, color: scheme.onSurfaceVariant),
            const SizedBox(width: AppTheme.space2),
            Expanded(
              child: Text(
                sheet.location.isEmpty ? 'Location not set yet' : sheet.location,
                style: theme.textTheme.titleMedium,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _OnSetAllDay extends StatelessWidget {
  const _OnSetAllDay({required this.people});

  final List<ParticipantRef> people;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('On set all day', style: theme.textTheme.headlineSmall),
        const SizedBox(height: AppTheme.space3),
        Wrap(
          spacing: AppTheme.space2,
          runSpacing: AppTheme.space2,
          children: [for (final person in people) PersonChip(person)],
        ),
      ],
    );
  }
}

class _Scenes extends StatelessWidget {
  const _Scenes({required this.scenes});

  final List<CallSheetScene> scenes;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Scenes', style: theme.textTheme.headlineSmall),
        const SizedBox(height: AppTheme.space3),
        if (scenes.isEmpty)
          Text(
            'No scenes are attached to this day yet.',
            style: theme.textTheme.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
          ),
        for (final scene in scenes) ...[
          Card(
            child: Padding(
              padding: const EdgeInsets.all(AppTheme.space5),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      SizedBox(
                        width: 56,
                        child: Text(scene.number, style: theme.textTheme.titleLarge),
                      ),
                      Expanded(
                        child: Text(scene.description, style: theme.textTheme.bodyLarge),
                      ),
                    ],
                  ),
                  const SizedBox(height: AppTheme.space4),
                  if (scene.participants.isEmpty)
                    Text('Nobody attached yet', style: theme.textTheme.bodySmall)
                  else
                    Wrap(
                      spacing: AppTheme.space2,
                      runSpacing: AppTheme.space2,
                      children: [for (final person in scene.participants) PersonChip(person)],
                    ),
                ],
              ),
            ),
          ),
          const SizedBox(height: AppTheme.space3),
        ],
      ],
    );
  }
}
