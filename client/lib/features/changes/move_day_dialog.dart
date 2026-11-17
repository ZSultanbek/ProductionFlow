import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/backend.dart';
import '../../core/dates.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';
import '../shooting_day/add_day_dialog.dart' show PickerField;

/// Step one of moving a day (US-07): pick the new date, call time or place.
/// "See who this affects" asks the server for the impact preview, which writes
/// nothing; the dialog pops with the [ChangePreview] for the next screen.
///
/// This is the only way the app changes a day's date, time or location — there
/// is deliberately no plain edit form for them (CLAUDE.md).
class MoveDayDialog extends ConsumerStatefulWidget {
  const MoveDayDialog({super.key, required this.sheet});

  /// The day as it is now. The fields start from its values.
  final CallSheet sheet;

  @override
  ConsumerState<MoveDayDialog> createState() => _MoveDayDialogState();
}

class _MoveDayDialogState extends ConsumerState<MoveDayDialog> {
  late String _shootDate = widget.sheet.shootDate;
  late String _callTime = widget.sheet.callTime;
  late final _location = TextEditingController(text: widget.sheet.location);

  bool _busy = false;
  String? _error;
  Map<String, String> _fieldErrors = const {};

  @override
  void dispose() {
    _location.dispose();
    super.dispose();
  }

  Future<void> _pickDate() async {
    final parts = _shootDate.split('-').map(int.parse).toList();
    // A calendar date for the picker to start on; no time zone is involved.
    final current = DateTime(parts[0], parts[1], parts[2]);
    final picked = await showDatePicker(
      context: context,
      initialDate: current,
      firstDate: DateTime(current.year - 1),
      lastDate: DateTime(current.year + 3),
    );
    if (picked != null && mounted) {
      setState(() => _shootDate = dateKey(picked));
    }
  }

  Future<void> _pickTime() async {
    final parts = _callTime.split(':');
    final picked = await showTimePicker(
      context: context,
      initialTime: TimeOfDay(hour: int.parse(parts[0]), minute: int.parse(parts[1])),
    );
    if (picked != null && mounted) {
      setState(() => _callTime = callTimeKey(picked.hour, picked.minute));
    }
  }

  Future<void> _preview() async {
    if (_busy) {
      return;
    }
    final sheet = widget.sheet;
    final location = _location.text.trim();
    // Only what actually changes is sent, so the preview's diff matches what
    // the producer did on this form.
    final shootDate = _shootDate == sheet.shootDate ? null : _shootDate;
    final callTime = _callTime == sheet.callTime ? null : _callTime;
    final newLocation = location == sheet.location ? null : location;
    if (shootDate == null && callTime == null && newLocation == null) {
      setState(() => _error = 'Change the date, the call time or the location first.');
      return;
    }

    setState(() {
      _busy = true;
      _error = null;
      _fieldErrors = const {};
    });
    try {
      final preview = await ref.read(backendProvider).previewMoveDay(
        sheet.id,
        shootDate: shootDate,
        callTime: callTime,
        location: newLocation,
      );
      if (mounted) {
        Navigator.of(context).pop(preview);
      }
    } on Object catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _busy = false;
        if (error is ApiException && error.fields.isNotEmpty) {
          // The server names them "to.shoot_date" and so on.
          _fieldErrors = {
            for (final entry in error.fields.entries) entry.key.replaceFirst('to.', ''): entry.value,
          };
        } else {
          _error = describeError(error);
        }
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Move this day'),
      scrollable: true,
      content: SizedBox(
        width: 360,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Nothing changes yet. Next you will see exactly who and what this '
              'affects, before anything is confirmed.',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: AppTheme.space4),
            if (_error != null) ...[
              FormError(_error!),
              const SizedBox(height: AppTheme.space4),
            ],
            PickerField(
              label: 'Date',
              value: formatShootDate(_shootDate),
              icon: Icons.calendar_today_outlined,
              errorText: _fieldErrors['shoot_date'],
              onTap: _busy ? null : _pickDate,
            ),
            const SizedBox(height: AppTheme.space4),
            PickerField(
              label: 'Call time',
              value: _callTime,
              icon: Icons.schedule_rounded,
              errorText: _fieldErrors['call_time'],
              onTap: _busy ? null : _pickTime,
            ),
            const SizedBox(height: AppTheme.space4),
            TextField(
              controller: _location,
              decoration: InputDecoration(
                labelText: 'Location',
                errorText: _fieldErrors['location'],
              ),
              textCapitalization: TextCapitalization.sentences,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _preview(),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        BusyButton(busy: _busy, onPressed: _preview, label: 'See who this affects'),
      ],
    );
  }
}
