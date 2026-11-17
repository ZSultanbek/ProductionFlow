import 'package:flutter/material.dart';

import '../../core/dates.dart';
import '../../core/models.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';

/// Step two of moving a day (US-07): who and what the change would affect,
/// before anything is confirmed. This is the screen the product exists for.
///
/// Nothing has been changed when this screen is open. Confirming — applying
/// the change and telling the people on this list — is US-08, so the button
/// is here but switched off until that lands.
class ImpactPreviewPage extends StatelessWidget {
  const ImpactPreviewPage({super.key, required this.sheet, required this.preview});

  /// The day as it is now.
  final CallSheet sheet;
  final ChangePreview preview;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final people = preview.participants;
    final withoutApp = people.where((p) => !p.hasAccount).length;

    return Scaffold(
      appBar: AppBar(title: const Text('Who this affects')),
      body: PageBody(
        children: [
          Eyebrow(sheet.projectName),
          const SizedBox(height: AppTheme.space2),
          Text('Move ${formatShootDate(sheet.shootDate)}', style: theme.textTheme.headlineLarge),
          const SizedBox(height: AppTheme.space5),

          for (final entry in preview.diff.entries) ...[
            _DiffRow(field: entry.key, change: entry.value),
            const SizedBox(height: AppTheme.space2),
          ],
          const SizedBox(height: AppTheme.space4),

          Card(
            color: scheme.secondaryContainer,
            child: Padding(
              padding: const EdgeInsets.all(AppTheme.space5),
              child: Text(
                _summary(preview.scenes.length, people.length, preview.willNotifyCount),
                style: theme.textTheme.titleMedium?.copyWith(color: scheme.onSecondaryContainer),
              ),
            ),
          ),
          const SizedBox(height: AppTheme.space6),

          Text('Scenes', style: theme.textTheme.headlineSmall),
          const SizedBox(height: AppTheme.space3),
          if (preview.scenes.isEmpty)
            Text('No scenes are on this day.', style: theme.textTheme.bodyMedium),
          for (final scene in preview.scenes)
            ListTile(
              contentPadding: EdgeInsets.zero,
              leading: SizedBox(
                width: 48,
                child: Text(scene.number, style: theme.textTheme.titleLarge),
              ),
              title: Text(scene.description.isEmpty ? 'Scene ${scene.number}' : scene.description),
            ),
          const SizedBox(height: AppTheme.space6),

          Text('People', style: theme.textTheme.headlineSmall),
          const SizedBox(height: AppTheme.space3),
          if (people.isEmpty)
            Text(
              'Nobody is attached to this day, so nobody needs to be told.',
              style: theme.textTheme.bodyMedium,
            ),
          for (final person in people) _PersonRow(person: person),
          if (withoutApp > 0) ...[
            const SizedBox(height: AppTheme.space3),
            Text(
              withoutApp == 1
                  ? '1 person has no app — they will not get a notification, so tell them yourself.'
                  : '$withoutApp people have no app — they will not get a notification, so tell them yourself.',
              style: theme.textTheme.bodySmall,
            ),
          ],
          const SizedBox(height: AppTheme.space7),

          // US-08 turns this on. Until then the preview is look-only, and it
          // says so rather than pretending to confirm.
          const FilledButton(onPressed: null, child: Text('Confirm change')),
          const SizedBox(height: AppTheme.space2),
          Text(
            'Confirming is coming in the next update. Nothing has been changed.',
            style: theme.textTheme.bodySmall,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: AppTheme.space3),
          OutlinedButton(
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('Back to the call sheet'),
          ),
          const SizedBox(height: AppTheme.space7),
        ],
      ),
    );
  }
}

String _summary(int scenes, int people, int notify) {
  String count(int n, String one, String many) => n == 1 ? '1 $one' : '$n $many';
  final affects = 'Affects ${count(scenes, 'scene', 'scenes')} and ${count(people, 'person', 'people')}.';
  if (people == 0) {
    return affects;
  }
  return '$affects ${count(notify, 'person', 'people')} would be notified.';
}

class _DiffRow extends StatelessWidget {
  const _DiffRow({required this.field, required this.change});

  final String field;
  final FieldChange change;

  String _show(String? value) {
    if (value == null || value.isEmpty) {
      return 'not set';
    }
    return field == 'shoot_date' ? formatShootDate(value) : value;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final label = switch (field) {
      'shoot_date' => 'Date',
      'call_time' => 'Call time',
      'location' => 'Location',
      _ => field,
    };
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(width: 96, child: Text(label, style: theme.textTheme.bodySmall)),
        Expanded(
          child: Text(
            '${_show(change.from)}  →  ${_show(change.to)}',
            style: theme.textTheme.titleMedium,
          ),
        ),
      ],
    );
  }
}

class _PersonRow extends StatelessWidget {
  const _PersonRow({required this.person});

  final AffectedParticipant person;

  @override
  Widget build(BuildContext context) {
    final reasons = [
      if (person.onSetAllDay) 'on set all day',
      if (person.sceneNumbers.isNotEmpty)
        person.sceneNumbers.length == 1
            ? 'in scene ${person.sceneNumbers.single}'
            : 'in scenes ${person.sceneNumbers.join(', ')}',
    ];
    return ListTile(
      contentPadding: EdgeInsets.zero,
      title: Text('${person.displayName} · ${person.role}'),
      subtitle: Text(reasons.join(' and ')),
      trailing: person.hasAccount
          ? null
          : const Chip(label: Text('no app'), visualDensity: VisualDensity.compact),
    );
  }
}
