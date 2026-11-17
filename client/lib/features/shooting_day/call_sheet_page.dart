import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/backend.dart';
import '../../core/dates.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';
import '../changes/impact_preview_page.dart';
import '../changes/move_day_dialog.dart';
import '../people/people_picker.dart';
import '../scenes/add_scene_dialog.dart';
import '../scenes/scene_dialogs.dart';

/// The call sheet for one shooting day (US-05): when, where, which scenes and
/// who is needed. The same page for everyone on the project — it replaces the
/// three versions of the spreadsheet.
///
/// A producer also builds the day here (US-06): adds scenes, says who is
/// needed in each and who is on set all day, and moves the day — which first
/// shows who the move affects (US-07).
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
            if (data.myRole == Roles.producer) ...[
              const SizedBox(height: AppTheme.space4),
              Align(
                alignment: Alignment.centerLeft,
                child: OutlinedButton.icon(
                  onPressed: () => _moveDay(context, data),
                  icon: const Icon(Icons.edit_calendar_outlined, size: 18),
                  label: const Text('Move this day'),
                ),
              ),
            ],
            const SizedBox(height: AppTheme.space6),
            if (data.onSetAllDay.isNotEmpty || data.myRole == Roles.producer) ...[
              _OnSetAllDay(sheet: data, projectId: projectId),
              const SizedBox(height: AppTheme.space6),
            ],
            _Scenes(sheet: data, projectId: projectId),
            const SizedBox(height: AppTheme.space7),
          ],
        ),
      ),
    );
  }

  /// Move day → preview → the impact screen. Nothing is written on the way.
  Future<void> _moveDay(BuildContext context, CallSheet sheet) async {
    final preview = await showDialog<ChangePreview>(
      context: context,
      builder: (context) => MoveDayDialog(sheet: sheet),
    );
    if (preview == null || !context.mounted) {
      return;
    }
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        fullscreenDialog: true,
        builder: (context) => ImpactPreviewPage(sheet: sheet, preview: preview),
      ),
    );
  }
}

/// After a producer changes what is on a day, everything that shows it is
/// stale: this sheet, the project's scene list and day cards (scene counts),
/// and Today.
void _refreshDay(WidgetRef ref, CallSheet sheet) {
  ref.invalidate(callSheetProvider(sheet.id));
  ref.invalidate(scenesProvider(sheet.projectId));
  ref.invalidate(daysProvider(sheet.projectId));
  ref.invalidate(todayProvider);
}

/// Runs one change and reports a failure at the bottom of the screen.
Future<void> _change(
  BuildContext context,
  WidgetRef ref,
  CallSheet sheet,
  Future<void> Function(Backend backend) request,
) async {
  try {
    await request(ref.read(backendProvider));
    _refreshDay(ref, sheet);
  } on Object catch (error) {
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(describeError(error))));
    }
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

class _OnSetAllDay extends ConsumerWidget {
  const _OnSetAllDay({required this.sheet, required this.projectId});

  final CallSheet sheet;
  final String projectId;

  Future<void> _edit(BuildContext context, WidgetRef ref) async {
    final saved = await showDialog<bool>(
      context: context,
      builder: (context) => PeoplePickerDialog(
        projectId: projectId,
        title: 'On set all day',
        selected: {for (final person in sheet.onSetAllDay) person.id},
        onSave: (ids) => ref.read(backendProvider).setDayParticipants(sheet.id, ids),
      ),
    );
    if (saved == true) {
      _refreshDay(ref, sheet);
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final iAmProducer = sheet.myRole == Roles.producer;
    final people = sheet.onSetAllDay;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('On set all day', style: theme.textTheme.headlineSmall),
        const SizedBox(height: AppTheme.space3),
        if (people.isEmpty)
          Text(
            'Nobody yet — the people needed whatever is shot: sound, the gaffer.',
            style: theme.textTheme.bodyMedium?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
        Wrap(
          spacing: AppTheme.space2,
          runSpacing: AppTheme.space2,
          children: [for (final person in people) PersonChip(person)],
        ),
        if (iAmProducer) ...[
          const SizedBox(height: AppTheme.space2),
          TextButton.icon(
            onPressed: () => _edit(context, ref),
            icon: const Icon(Icons.group_add_outlined, size: 18),
            label: const Text('Choose people'),
          ),
        ],
      ],
    );
  }
}

class _Scenes extends ConsumerWidget {
  const _Scenes({required this.sheet, required this.projectId});

  final CallSheet sheet;
  final String projectId;

  Future<void> _add(BuildContext context, WidgetRef ref) async {
    final scene = await showDialog<Scene>(
      context: context,
      builder: (context) => AddSceneDialog(projectId: projectId, dayId: sheet.id),
    );
    if (scene != null) {
      _refreshDay(ref, sheet);
    }
  }

  Future<void> _addUnscheduled(BuildContext context, WidgetRef ref, List<Scene> loose) async {
    final sceneId = await pickScene(context, loose);
    if (sceneId == null || !context.mounted) {
      return;
    }
    await _change(context, ref, sheet, (backend) => backend.moveScene(sceneId, sheet.id));
  }

  Future<void> _choosePeople(BuildContext context, WidgetRef ref, CallSheetScene scene) async {
    final saved = await showDialog<bool>(
      context: context,
      builder: (context) => PeoplePickerDialog(
        projectId: projectId,
        title: 'Who is needed in ${scene.number}',
        selected: {for (final person in scene.participants) person.id},
        onSave: (ids) => ref.read(backendProvider).setSceneParticipants(scene.id, ids),
      ),
    );
    if (saved == true) {
      _refreshDay(ref, sheet);
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final scenes = sheet.scenes;
    final iAmProducer = sheet.myRole == Roles.producer;

    // Scenes not on any day yet, which a producer can put on this one.
    final loose = iAmProducer
        ? [
            for (final scene in ref.watch(scenesProvider(projectId)).value ?? const <Scene>[])
              if (scene.shootingDayId == null) scene,
          ]
        : const <Scene>[];

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
        if (iAmProducer) ...[
          Wrap(
            spacing: AppTheme.space2,
            runSpacing: AppTheme.space2,
            children: [
              TextButton.icon(
                onPressed: () => _add(context, ref),
                icon: const Icon(Icons.add_rounded, size: 18),
                label: const Text('Add scene'),
              ),
              if (loose.isNotEmpty)
                TextButton.icon(
                  onPressed: () => _addUnscheduled(context, ref, loose),
                  icon: const Icon(Icons.playlist_add_rounded, size: 18),
                  label: Text('Add an unscheduled scene (${loose.length})'),
                ),
            ],
          ),
          const SizedBox(height: AppTheme.space3),
        ],
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
                      if (iAmProducer)
                        PopupMenuButton<String>(
                          tooltip: 'Scene ${scene.number}',
                          onSelected: (action) {
                            if (action == 'people') {
                              _choosePeople(context, ref, scene);
                            } else {
                              _change(context, ref, sheet, (backend) => backend.moveScene(scene.id, null));
                            }
                          },
                          itemBuilder: (context) => const [
                            PopupMenuItem(value: 'people', child: Text('Who is needed')),
                            PopupMenuItem(value: 'unschedule', child: Text('Take off this day')),
                          ],
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
