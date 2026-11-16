import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/backend.dart';
import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/routes.dart';
import '../../core/session.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';

// Log in and Create account (US-01). The two screens are the same form with
// one extra field, so they share everything below.
//
// Neither screen navigates after success. Signing in changes the session, the
// router notices, and its redirect moves on — to `from` if the user was on
// their way somewhere (an invite link, say), otherwise to Today.

class LoginPage extends StatelessWidget {
  const LoginPage({super.key, this.from});

  /// Where the user was heading when they were asked to log in.
  final String? from;

  @override
  Widget build(BuildContext context) => _AuthForm(isRegister: false, from: from);
}

class RegisterPage extends StatelessWidget {
  const RegisterPage({super.key, this.from});

  final String? from;

  @override
  Widget build(BuildContext context) => _AuthForm(isRegister: true, from: from);
}

class _AuthForm extends ConsumerStatefulWidget {
  const _AuthForm({required this.isRegister, required this.from});

  final bool isRegister;
  final String? from;

  @override
  ConsumerState<_AuthForm> createState() => _AuthFormState();
}

class _AuthFormState extends ConsumerState<_AuthForm> {
  final _formKey = GlobalKey<FormState>();
  final _name = TextEditingController();
  final _email = TextEditingController();
  final _password = TextEditingController();

  bool _busy = false;

  /// A problem with the whole form: wrong password, no connection.
  String? _error;

  /// Problems the server found with single fields, shown under each one.
  Map<String, String> _fieldErrors = const {};

  @override
  void dispose() {
    _name.dispose();
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_busy || !_formKey.currentState!.validate()) {
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
      _fieldErrors = const {};
    });

    final backend = ref.read(backendProvider);
    try {
      final AuthResult result;
      if (widget.isRegister) {
        result = await backend.register(
          email: _email.text,
          password: _password.text,
          displayName: _name.text,
        );
      } else {
        result = await backend.login(email: _email.text, password: _password.text);
      }
      // The router takes it from here; this screen is about to be replaced.
      await ref.read(sessionProvider.notifier).signIn(result);
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
    final isRegister = widget.isRegister;

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(AppTheme.space5),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 400),
              child: Form(
                key: _formKey,
                child: AutofillGroup(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Align(
                        alignment: Alignment.centerLeft,
                        child: TextButton(
                          onPressed: () => context.go(Routes.landing),
                          child: const Text('ProductionFlow'),
                        ),
                      ),
                      const SizedBox(height: AppTheme.space5),
                      Text(
                        isRegister ? 'Create your account' : 'Welcome back',
                        style: theme.textTheme.headlineLarge,
                      ),
                      const SizedBox(height: AppTheme.space3),
                      Text(
                        isRegister
                            ? 'One account for every production you work on.'
                            : 'Log in to see where you need to be.',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                      const SizedBox(height: AppTheme.space6),

                      if (_error != null) ...[
                        FormError(_error!),
                        const SizedBox(height: AppTheme.space4),
                      ],

                      if (isRegister) ...[
                        TextFormField(
                          controller: _name,
                          decoration: InputDecoration(
                            labelText: 'Your name',
                            helperText: 'How the crew will see you on a call sheet',
                            errorText: _fieldErrors['display_name'],
                          ),
                          textCapitalization: TextCapitalization.words,
                          textInputAction: TextInputAction.next,
                          autofillHints: const [AutofillHints.name],
                          validator: (value) =>
                              (value ?? '').trim().isEmpty ? 'Enter your name' : null,
                        ),
                        const SizedBox(height: AppTheme.space4),
                      ],

                      TextFormField(
                        controller: _email,
                        decoration: InputDecoration(
                          labelText: 'Email',
                          errorText: _fieldErrors['email'],
                        ),
                        keyboardType: TextInputType.emailAddress,
                        textInputAction: TextInputAction.next,
                        autocorrect: false,
                        autofillHints: const [AutofillHints.email],
                        validator: (value) =>
                            (value ?? '').trim().isEmpty ? 'Enter your email' : null,
                      ),
                      const SizedBox(height: AppTheme.space4),

                      TextFormField(
                        controller: _password,
                        decoration: InputDecoration(
                          labelText: 'Password',
                          helperText: isRegister ? 'At least 8 characters' : null,
                          errorText: _fieldErrors['password'],
                        ),
                        obscureText: true,
                        textInputAction: TextInputAction.done,
                        autofillHints: [
                          isRegister ? AutofillHints.newPassword : AutofillHints.password,
                        ],
                        onFieldSubmitted: (_) => _submit(),
                        validator: (value) {
                          final password = value ?? '';
                          if (password.isEmpty) {
                            return 'Enter your password';
                          }
                          // Only a new password is held to the length rule. At
                          // login, whatever they typed is for the server to judge.
                          if (isRegister && password.length < 8) {
                            return 'Use at least 8 characters';
                          }
                          return null;
                        },
                      ),
                      const SizedBox(height: AppTheme.space5),

                      BusyButton(
                        busy: _busy,
                        onPressed: _submit,
                        label: isRegister ? 'Create account' : 'Log in',
                      ),
                      const SizedBox(height: AppTheme.space3),

                      TextButton(
                        onPressed: _busy
                            ? null
                            : () => context.go(
                                Routes.withFrom(
                                  isRegister ? Routes.login : Routes.register,
                                  widget.from,
                                ),
                              ),
                        child: Text(
                          isRegister
                              ? 'I already have an account'
                              : 'New here? Create an account',
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
