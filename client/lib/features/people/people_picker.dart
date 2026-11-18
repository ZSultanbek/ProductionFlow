import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';

/// Picks who is needed for something — a scene, or the whole day — from the
/// project's people. [onSave] does the actual request; the dialog stays open
/// and shows the error if it fails, and pops with `true` once it succeeds.
class PeoplePickerDialog extends ConsumerStatefulWidget {
  const PeoplePickerDialog({
    super.key,
    required this.projectId,
    required this.title,
    required this.selected,
    required this.onSave,
  });

  final String projectId;
  final String title;

  /// Ids ticked when the dialog opens.
  final Set<String> selected;
  final Future<void> Function(List<String> participantIds) onSave;

  @override
  ConsumerState<PeoplePickerDialog> createState() => _PeoplePickerDialogState();
}

class _PeoplePickerDialogState extends ConsumerState<PeoplePickerDialog> {
  late final Set<String> _picked = {...widget.selected};
  bool _busy = false;
  String? _error;

  Future<void> _save(List<Participant> people) async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      // In list order, so the request reads the same as the screen.
      await widget.onSave([
        for (final person in people)
          if (_picked.contains(person.id)) person.id,
      ]);
      if (mounted) {
        Navigator.of(context).pop(true);
      }
    } on Object catch (error) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = describeError(error);
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final people = ref.watch(participantsProvider(widget.projectId));

    return AlertDialog(
      title: Text(widget.title),
      scrollable: true,
      content: SizedBox(
        width: 380,
        child: people.when(
          loading: () => const Padding(
            padding: EdgeInsets.all(AppTheme.space6),
            child: Center(child: CircularProgressIndicator()),
          ),
          error: (error, _) => FormError(describeError(error)),
          data: (list) => Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (_error != null) ...[
                FormError(_error!),
                const SizedBox(height: AppTheme.space3),
              ],
              if (list.isEmpty)
                Text('Nobody is in this project yet.', style: theme.textTheme.bodyMedium),
              for (final person in list)
                CheckboxListTile(
                  value: _picked.contains(person.id),
                  onChanged: _busy
                      ? null
                      : (on) => setState(() {
                          if (on == true) {
                            _picked.add(person.id);
                          } else {
                            _picked.remove(person.id);
                          }
                        }),
                  title: Text(person.displayName),
                  subtitle: Text(person.hasAccount ? person.role : '${person.role} · no app'),
                  contentPadding: EdgeInsets.zero,
                  controlAffinity: ListTileControlAffinity.leading,
                ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        BusyButton(
          busy: _busy,
          onPressed: () {
            final list = people.value;
            if (list != null) {
              _save(list);
            }
          },
          label: 'Save',
        ),
      ],
    );
  }
}
