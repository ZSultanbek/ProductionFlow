import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/backend.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';

/// Adds someone to the project by name (US-06): the actor who will never
/// install the app, the gaffer you only know by phone. They still count in
/// every impact preview. Pops with the new [Participant].
class AddPersonDialog extends ConsumerStatefulWidget {
  const AddPersonDialog({super.key, required this.projectId});

  final String projectId;

  @override
  ConsumerState<AddPersonDialog> createState() => _AddPersonDialogState();
}

class _AddPersonDialogState extends ConsumerState<AddPersonDialog> {
  final _name = TextEditingController();
  final _email = TextEditingController();
  String _role = Roles.cast;

  bool _busy = false;
  String? _error;
  Map<String, String> _fieldErrors = const {};

  @override
  void dispose() {
    _name.dispose();
    _email.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_busy) {
      return;
    }
    final name = _name.text.trim();
    if (name.isEmpty) {
      setState(() => _fieldErrors = const {'display_name': 'Enter a name'});
      return;
    }
    final email = _email.text.trim();
    setState(() {
      _busy = true;
      _error = null;
      _fieldErrors = const {};
    });
    try {
      final person = await ref.read(backendProvider).addParticipant(
        widget.projectId,
        displayName: name,
        role: _role,
        email: email.isEmpty ? null : email,
      );
      if (mounted) {
        Navigator.of(context).pop(person);
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
    final theme = Theme.of(context);
    return AlertDialog(
      title: const Text('Add a person'),
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
              controller: _name,
              autofocus: true,
              decoration: InputDecoration(
                labelText: 'Name',
                errorText: _fieldErrors['display_name'],
              ),
              textCapitalization: TextCapitalization.words,
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: AppTheme.space4),
            SegmentedButton<String>(
              segments: const [
                ButtonSegment(value: Roles.cast, label: Text('Cast')),
                ButtonSegment(value: Roles.crew, label: Text('Crew')),
              ],
              selected: {_role},
              onSelectionChanged: _busy ? null : (picked) => setState(() => _role = picked.first),
            ),
            if (_fieldErrors['role'] != null) ...[
              const SizedBox(height: AppTheme.space2),
              Text(
                _fieldErrors['role']!,
                style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.error),
              ),
            ],
            const SizedBox(height: AppTheme.space4),
            TextField(
              controller: _email,
              decoration: InputDecoration(
                labelText: 'Email',
                helperText: 'Optional. If they sign up with it later, it links them to this entry.',
                helperMaxLines: 2,
                errorText: _fieldErrors['email'],
              ),
              keyboardType: TextInputType.emailAddress,
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
        BusyButton(busy: _busy, onPressed: _submit, label: 'Add person'),
      ],
    );
  }
}
