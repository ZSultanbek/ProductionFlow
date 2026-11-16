import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'backend.dart';
import 'models.dart';
import 'theme.dart';

// Small pieces of UI shared by more than one feature.

/// What to tell the user when [error] happened.
String describeError(Object error) {
  if (error is ApiException) {
    return error.message;
  }
  return 'Something went wrong. Try again.';
}

/// The body of an ordinary screen: scrolls, and keeps its content in a
/// readable column instead of stretching across a wide monitor.
class PageBody extends StatelessWidget {
  const PageBody({super.key, required this.children, this.onRefresh});

  final List<Widget> children;

  /// If set, pulling down on a phone reloads the screen.
  final Future<void> Function()? onRefresh;

  static const double maxWidth = 720;

  @override
  Widget build(BuildContext context) {
    final list = ListView(
      // Always scrollable, so pull-to-refresh works on a short page too.
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.symmetric(vertical: AppTheme.space5),
      children: [
        for (final child in children)
          Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: maxWidth),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: AppTheme.space5),
                child: child,
              ),
            ),
          ),
      ],
    );

    final refresh = onRefresh;
    if (refresh == null) {
      return list;
    }
    return RefreshIndicator(onRefresh: refresh, child: list);
  }
}

/// Shows the three states of something being loaded: waiting, failed, there.
///
/// While reloading after a change, the old data stays on screen instead of
/// flashing back to a spinner.
class AsyncView<T> extends StatelessWidget {
  const AsyncView({
    super.key,
    required this.value,
    required this.onRetry,
    required this.builder,
  });

  final AsyncValue<T> value;
  final VoidCallback onRetry;
  final Widget Function(T data) builder;

  @override
  Widget build(BuildContext context) {
    return value.when(
      data: builder,
      loading: () => const SlowLoader(),
      error: (error, _) => ErrorState(message: describeError(error), onRetry: onRetry),
    );
  }
}

/// A spinner that explains itself if the wait gets long.
///
/// The free server sleeps when idle and takes about a minute to wake. Without
/// the explanation, that minute looks like a broken app.
class SlowLoader extends StatefulWidget {
  const SlowLoader({super.key});

  @override
  State<SlowLoader> createState() => _SlowLoaderState();
}

class _SlowLoaderState extends State<SlowLoader> {
  Timer? _timer;
  bool _slow = false;

  @override
  void initState() {
    super.initState();
    _timer = Timer(const Duration(seconds: 5), () {
      if (mounted) {
        setState(() => _slow = true);
      }
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppTheme.space6),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const SizedBox(
              width: 28,
              height: 28,
              child: CircularProgressIndicator(strokeWidth: 3),
            ),
            if (_slow) ...[
              const SizedBox(height: AppTheme.space5),
              Text(
                'Waking up the server…',
                style: theme.textTheme.titleMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: AppTheme.space2),
              Text(
                'It sleeps when nobody is using it. This can take up to a minute.',
                style: theme.textTheme.bodySmall,
                textAlign: TextAlign.center,
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// A screen-sized message: nothing here yet, or something failed.
class MessageState extends StatelessWidget {
  const MessageState({
    super.key,
    required this.icon,
    required this.title,
    required this.body,
    this.action,
  });

  final IconData icon;
  final String title;
  final String body;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: SingleChildScrollView(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Padding(
            padding: const EdgeInsets.all(AppTheme.space6),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(icon, size: 24, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(height: AppTheme.space5),
                Text(title, style: theme.textTheme.headlineSmall),
                const SizedBox(height: AppTheme.space3),
                Text(
                  body,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                if (action != null) ...[
                  const SizedBox(height: AppTheme.space5),
                  action!,
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class ErrorState extends StatelessWidget {
  const ErrorState({super.key, required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return MessageState(
      icon: Icons.cloud_off_rounded,
      title: 'Could not load this',
      body: message,
      action: OutlinedButton(onPressed: onRetry, child: const Text('Try again')),
    );
  }
}

/// The small all-caps label that sits above a heading.
class Eyebrow extends StatelessWidget {
  const Eyebrow(this.text, {super.key});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Text(
      text.toUpperCase(),
      style: theme.textTheme.labelSmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
    );
  }
}

/// An error shown inside a form, above the fields.
class FormError extends StatelessWidget {
  const FormError(this.message, {super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppTheme.space4),
      decoration: BoxDecoration(
        color: scheme.errorContainer,
        borderRadius: BorderRadius.circular(AppTheme.radiusControl),
      ),
      child: Text(
        message,
        style: theme.textTheme.bodyMedium?.copyWith(color: scheme.onErrorContainer),
      ),
    );
  }
}

/// A person on a call sheet: "Aisha · cast".
class PersonChip extends StatelessWidget {
  const PersonChip(this.person, {super.key});

  final ParticipantRef person;

  @override
  Widget build(BuildContext context) {
    return Chip(
      label: Text('${person.displayName} · ${person.role}'),
      visualDensity: VisualDensity.compact,
    );
  }
}

/// A filled button that shows a spinner and ignores taps while [busy].
class BusyButton extends StatelessWidget {
  const BusyButton({
    super.key,
    required this.busy,
    required this.onPressed,
    required this.label,
  });

  final bool busy;
  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return FilledButton(
      onPressed: busy ? null : onPressed,
      child: busy
          ? const SizedBox(
              width: 20,
              height: 20,
              child: CircularProgressIndicator(strokeWidth: 2.5),
            )
          : Text(label),
    );
  }
}
