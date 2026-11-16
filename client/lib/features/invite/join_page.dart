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

/// Where an invite link lands (US-03): /join/:code.
///
/// Anyone can open it and see which project they are being invited to. Joining
/// needs an account, so a logged-out visitor is sent to log in or register and
/// comes straight back here afterwards.
class JoinPage extends ConsumerWidget {
  const JoinPage({super.key, required this.code});

  final String code;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final invite = ref.watch(inviteProvider(code));

    return Scaffold(
      body: SafeArea(
        child: invite.when(
          loading: () => const SlowLoader(),
          error: (error, _) {
            if (error is ApiException && error.code == 'invite_not_found') {
              return MessageState(
                icon: Icons.link_off_rounded,
                title: 'This invite link does not work',
                body:
                    'It may have been copied incompletely. Ask your producer to '
                    'send it again.',
                action: OutlinedButton(
                  onPressed: () => context.go(Routes.landing),
                  child: const Text('Back to start'),
                ),
              );
            }
            return ErrorState(
              message: describeError(error),
              onRetry: () => ref.invalidate(inviteProvider(code)),
            );
          },
          data: (data) => _JoinBody(code: code, invite: data),
        ),
      ),
    );
  }
}

class _JoinBody extends ConsumerStatefulWidget {
  const _JoinBody({required this.code, required this.invite});

  final String code;
  final Invite invite;

  @override
  ConsumerState<_JoinBody> createState() => _JoinBodyState();
}

class _JoinBodyState extends ConsumerState<_JoinBody> {
  String _role = Roles.crew;
  bool _busy = false;
  String? _error;

  Future<void> _join() async {
    if (_busy) {
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final participant = await ref.read(backendProvider).joinInvite(widget.code, _role);
      if (!mounted) {
        return;
      }
      // The project list and today's days both just changed.
      ref.invalidate(projectsProvider);
      ref.invalidate(todayProvider);
      context.go(Routes.project(participant.projectId));
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
    final loggedIn = ref.watch(sessionProvider) != null;
    final here = Routes.join(widget.code);

    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(AppTheme.space5),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Eyebrow('You are invited to join'),
              const SizedBox(height: AppTheme.space3),
              Text(widget.invite.projectName, style: theme.textTheme.displaySmall),
              const SizedBox(height: AppTheme.space5),

              if (!loggedIn) ...[
                Text(
                  'Log in or create an account to join. It takes a minute, and '
                  'you come straight back here.',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: AppTheme.space5),
                FilledButton(
                  onPressed: () => context.go(Routes.withFrom(Routes.register, here)),
                  child: const Text('Create an account'),
                ),
                const SizedBox(height: AppTheme.space3),
                OutlinedButton(
                  onPressed: () => context.go(Routes.withFrom(Routes.login, here)),
                  child: const Text('I already have an account'),
                ),
              ] else ...[
                Text('What do you do on this production?', style: theme.textTheme.titleMedium),
                const SizedBox(height: AppTheme.space3),
                SegmentedButton<String>(
                  segments: const [
                    ButtonSegment(
                      value: Roles.crew,
                      label: Text('Crew'),
                      icon: Icon(Icons.videocam_outlined),
                    ),
                    ButtonSegment(
                      value: Roles.cast,
                      label: Text('Cast'),
                      icon: Icon(Icons.theater_comedy_outlined),
                    ),
                  ],
                  selected: {_role},
                  onSelectionChanged: _busy
                      ? null
                      : (selection) => setState(() => _role = selection.first),
                ),
                const SizedBox(height: AppTheme.space2),
                Text(
                  'Camera, sound, light, art, production — that is crew. '
                  'In front of the camera is cast.',
                  style: theme.textTheme.bodySmall,
                ),
                const SizedBox(height: AppTheme.space5),
                if (_error != null) ...[
                  FormError(_error!),
                  const SizedBox(height: AppTheme.space4),
                ],
                BusyButton(busy: _busy, onPressed: _join, label: 'Join project'),
                const SizedBox(height: AppTheme.space3),
                TextButton(
                  onPressed: _busy ? null : () => context.go(Routes.today),
                  child: const Text('Not now'),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
