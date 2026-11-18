import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/backend.dart';
import '../../core/dates.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';

/// Adds a shooting day to a project (US-04): a date, a call time, a location.
/// Pops with the new [ShootingDay], or with nothing if cancelled.
///
/// This is the only place a day's date and call time are simply typed in.
/// Once a day exists, changing them goes through the impact preview — by then
/// there may be people attached who need to know.
class AddDayDialog extends ConsumerStatefulWidget {
  const AddDayDialog({super.key, required this.projectId});

  final String projectId;

  @override
  ConsumerState<AddDayDialog> createState() => _AddDayDialogState();
}

class _AddDayDialogState extends ConsumerState<AddDayDialog> {
  final _location = TextEditingController();

  /// Kept as the API's strings from the moment they are picked, so nothing in
  /// between can shift them: "2026-10-04" and "07:30".
  String? _shootDate;
  String _callTime = '07:00';

  bool _busy = false;
  String? _error;
  Map<String, String> _fieldErrors = const {};

  @override
  void dispose() {
    _location.dispose();
    super.dispose();
  }

  Future<void> _pickDate() async {
    final now = ref.read(clockProvider)();
    final picked = await showDatePicker(
      context: context,
      initialDate: now,
      // Past dates are allowed: crews also log days after the fact.
      firstDate: DateTime(now.year - 1),
      lastDate: DateTime(now.year + 3),
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

  Future<void> _submit() async {
    if (_busy) {
      return;
    }
    final shootDate = _shootDate;
    if (shootDate == null) {
      setState(() => _fieldErrors = const {'shoot_date': 'Pick a date'});
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
      _fieldErrors = const {};
    });
    try {
      final ShootingDay day = await ref.read(backendProvider).createDay(
        widget.projectId,
        shootDate: shootDate,
        callTime: _callTime,
        location: _location.text.trim(),
      );
      if (mounted) {
        Navigator.of(context).pop(day);
      }
    } on Object catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _busy = false;
        if (error is ApiException && error.fields.isNotEmpty) {
          _fieldErrors = error.fields;
        } else {
          _error = describeError(error);
        }
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final shootDate = _shootDate;

    return AlertDialog(
      title: const Text('Add shooting day'),
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
            PickerField(
              label: 'Date',
              value: shootDate == null ? 'Pick a date' : formatShootDate(shootDate),
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
                helperText: 'Optional — you can add it later',
                errorText: _fieldErrors['location'],
              ),
              textCapitalization: TextCapitalization.sentences,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _submit(),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        BusyButton(busy: _busy, onPressed: _submit, label: 'Add day'),
      ],
    );
  }
}

/// Looks like a text field, opens a picker when tapped. Shared with the move
/// dialog (features/changes), so a date is picked the same way in both.
class PickerField extends StatelessWidget {
  const PickerField({
    super.key,
    required this.label,
    required this.value,
    required this.icon,
    required this.onTap,
    this.errorText,
  });

  final String label;
  final String value;
  final IconData icon;
  final VoidCallback? onTap;
  final String? errorText;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(AppTheme.radiusControl),
      child: InputDecorator(
        decoration: InputDecoration(
          labelText: label,
          errorText: errorText,
          suffixIcon: Icon(icon, size: 20),
        ),
        child: Text(value, style: Theme.of(context).textTheme.bodyLarge),
      ),
    );
  }
}
