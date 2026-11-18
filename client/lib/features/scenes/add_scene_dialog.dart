import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/backend.dart';
import '../../core/dates.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';

/// Adds a scene (US-06): its number as the script has it ("12A"), a line on
/// what it is, and which day it is shot on — or none yet. Pops with the new
/// [Scene].
///
/// Opened from a call sheet, [dayId] is that day and cannot be changed here.
/// Opened from the project, the producer picks a day or leaves it unscheduled.
class AddSceneDialog extends ConsumerStatefulWidget {
  const AddSceneDialog({super.key, required this.projectId, this.dayId});

  final String projectId;
  final String? dayId;

  @override
  ConsumerState<AddSceneDialog> createState() => _AddSceneDialogState();
}

class _AddSceneDialogState extends ConsumerState<AddSceneDialog> {
  final _number = TextEditingController();
  final _description = TextEditingController();
  late String? _dayId = widget.dayId;

  bool _busy = false;
  String? _error;
  Map<String, String> _fieldErrors = const {};

  @override
  void dispose() {
    _number.dispose();
    _description.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_busy) {
      return;
    }
    final number = _number.text.trim();
    if (number.isEmpty) {
      setState(() => _fieldErrors = const {'number': 'Enter the scene number'});
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
      _fieldErrors = const {};
    });
    try {
      final scene = await ref.read(backendProvider).createScene(
        widget.projectId,
        number: number,
        description: _description.text.trim(),
        shootingDayId: _dayId,
      );
      if (mounted) {
        Navigator.of(context).pop(scene);
      }
    } on Object catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _busy = false;
        if (error is ApiException && error.code == 'scene_number_taken') {
          _fieldErrors = {'number': 'There is already a scene $number'};
        } else if (error is ApiException && error.fields.isNotEmpty) {
          _fieldErrors = error.fields;
        } else {
          _error = describeError(error);
        }
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Add scene'),
      scrollable: true,
      content: SizedBox(
        width: 360,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (_error != null) ...[
              FormError(_error!),
              const SizedBox(height: AppTheme.space4),
            ],
            TextField(
              controller: _number,
              autofocus: true,
              decoration: InputDecoration(
                labelText: 'Scene number',
                hintText: 'e.g. 12A',
                errorText: _fieldErrors['number'],
              ),
              textCapitalization: TextCapitalization.characters,
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: AppTheme.space4),
            TextField(
              controller: _description,
              decoration: InputDecoration(
                labelText: 'What happens',
                hintText: 'e.g. Rooftop argument, dusk',
                errorText: _fieldErrors['description'],
              ),
              textCapitalization: TextCapitalization.sentences,
              textInputAction: widget.dayId == null ? TextInputAction.next : TextInputAction.done,
              onSubmitted: widget.dayId == null ? null : (_) => _submit(),
            ),
            if (widget.dayId == null) ...[
              const SizedBox(height: AppTheme.space4),
              _DayChoice(
                projectId: widget.projectId,
                value: _dayId,
                errorText: _fieldErrors['shooting_day_id'],
                onChanged: _busy ? null : (dayId) => setState(() => _dayId = dayId),
              ),
            ],
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        BusyButton(busy: _busy, onPressed: _submit, label: 'Add scene'),
      ],
    );
  }
}

/// "Not on a day yet", or one of the project's days.
class _DayChoice extends ConsumerWidget {
  const _DayChoice({
    required this.projectId,
    required this.value,
    required this.onChanged,
    this.errorText,
  });

  final String projectId;
  final String? value;
  final ValueChanged<String?>? onChanged;
  final String? errorText;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final days = ref.watch(daysProvider(projectId)).value ?? const <ShootingDay>[];
    return DropdownButtonFormField<String?>(
      initialValue: value,
      decoration: InputDecoration(labelText: 'Shooting day', errorText: errorText),
      items: [
        const DropdownMenuItem<String?>(child: Text('Not on a day yet')),
        for (final day in days)
          DropdownMenuItem<String?>(
            value: day.id,
            child: Text('${formatShootDate(day.shootDate)} · ${day.callTime}'),
          ),
      ],
      onChanged: onChanged,
    );
  }
}
