import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/models.dart';
import '../../core/providers.dart';
import '../../core/routes.dart';
import '../../core/theme.dart';
import '../../core/widgets.dart';
import '../app_shell/app_shell.dart';

/// Every production I am part of, and the two ways to get into one (US-02,
/// US-03): start my own, or use an invite from a producer.
class ProjectsPage extends ConsumerWidget {
  const ProjectsPage({super.key});

  Future<void> _create(BuildContext context, WidgetRef ref) async {
    final project = await showDialog<Project>(
      context: context,
      builder: (context) => const _CreateProjectDialog(),
    );
    if (project == null || !context.mounted) {
      return;
    }
    ref.invalidate(projectsProvider);
    // Straight into the new project: the next thing a producer wants is the
    // invite link and the first shooting day, and both are on that screen.
    context.go(Routes.project(project.id));
  }

  Future<void> _joinWithCode(BuildContext context) async {
    final code = await showDialog<String>(
      context: context,
      builder: (context) => const _InviteCodeDialog(),
    );
    if (code == null || !context.mounted) {
      return;
    }
    context.go(Routes.join(code));
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final projects = ref.watch(projectsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Projects'),
        actions: [
          TextButton(
            onPressed: () => _joinWithCode(context),
            child: const Text('Join with invite'),
          ),
          const AccountButton(),
          const SizedBox(width: AppTheme.space2),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _create(context, ref),
        icon: const Icon(Icons.add_rounded),
        label: const Text('New project'),
      ),
      body: AsyncView<List<Project>>(
        value: projects,
        onRetry: () => ref.invalidate(projectsProvider),
        builder: (list) {
          if (list.isEmpty) {
            return const MessageState(
              icon: Icons.movie_outlined,
              title: 'No projects yet',
              body:
                  'Start a new project if you are producing. If someone else '
                  'is, ask them for the invite link and open it — or paste it '
                  'under "Join with invite".',
            );
          }
          return PageBody(
            onRefresh: () async {
              try {
                ref.invalidate(projectsProvider);
                await ref.read(projectsProvider.future);
              } on Object {
                // The failed state is shown by AsyncView.
              }
            },
            children: [
              for (final project in list) ...[
                _ProjectCard(project: project),
                const SizedBox(height: AppTheme.space3),
              ],
              // Room for the floating button, so it never covers the last card.
              const SizedBox(height: AppTheme.space8),
            ],
          );
        },
      ),
    );
  }
}

class _ProjectCard extends StatelessWidget {
  const _ProjectCard({required this.project});

  final Project project;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => context.go(Routes.project(project.id)),
        child: Padding(
          padding: const EdgeInsets.all(AppTheme.space5),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(project.name, style: theme.textTheme.titleLarge),
                    const SizedBox(height: AppTheme.space1),
                    Text('You are ${project.myRole}', style: theme.textTheme.bodySmall),
                  ],
                ),
              ),
              Icon(Icons.chevron_right_rounded, color: theme.colorScheme.onSurfaceVariant),
            ],
          ),
        ),
      ),
    );
  }
}

/// Asks for a name and creates the project. Pops with the new [Project], or
/// with nothing if cancelled.
class _CreateProjectDialog extends ConsumerStatefulWidget {
  const _CreateProjectDialog();

  @override
  ConsumerState<_CreateProjectDialog> createState() => _CreateProjectDialogState();
}

class _CreateProjectDialogState extends ConsumerState<_CreateProjectDialog> {
  final _name = TextEditingController();
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final name = _name.text.trim();
    if (_busy) {
      return;
    }
    if (name.isEmpty) {
      setState(() => _error = 'Give the project a name');
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final project = await ref.read(backendProvider).createProject(name);
      if (mounted) {
        Navigator.of(context).pop(project);
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
    return AlertDialog(
      title: const Text('New project'),
      content: SizedBox(
        width: 360,
        child: TextField(
          controller: _name,
          autofocus: true,
          decoration: InputDecoration(
            labelText: 'Working title',
            helperText: 'You will be its producer',
            errorText: _error,
          ),
          textCapitalization: TextCapitalization.sentences,
          textInputAction: TextInputAction.done,
          onSubmitted: (_) => _submit(),
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        BusyButton(busy: _busy, onPressed: _submit, label: 'Create'),
      ],
    );
  }
}

/// Takes an invite link or a bare code and pops with the code.
///
/// On Android an invite link opens in the browser, not in the installed app,
/// so there has to be a way to bring the link in by hand.
class _InviteCodeDialog extends StatefulWidget {
  const _InviteCodeDialog();

  @override
  State<_InviteCodeDialog> createState() => _InviteCodeDialogState();
}

class _InviteCodeDialogState extends State<_InviteCodeDialog> {
  final _input = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _input.dispose();
    super.dispose();
  }

  void _submit() {
    final code = inviteCodeFrom(_input.text);
    if (code == null) {
      setState(() => _error = 'Paste the invite link you were sent');
      return;
    }
    Navigator.of(context).pop(code);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Join with invite'),
      content: SizedBox(
        width: 360,
        child: TextField(
          controller: _input,
          autofocus: true,
          decoration: InputDecoration(labelText: 'Invite link or code', errorText: _error),
          autocorrect: false,
          textInputAction: TextInputAction.done,
          onSubmitted: (_) => _submit(),
        ),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('Cancel')),
        FilledButton(onPressed: _submit, child: const Text('Continue')),
      ],
    );
  }
}

/// Pulls the invite code out of whatever was pasted: a full link
/// (https://…/join/3f9a0c2e), or just the code. Null if there is nothing usable.
String? inviteCodeFrom(String pasted) {
  final text = pasted.trim();
  if (text.isEmpty) {
    return null;
  }
  // The code is the last part of the link's path; a bare code has no slashes
  // and is its own last part.
  final withoutQuery = text.split('?').first.split('#').first;
  final parts = withoutQuery.split('/').where((part) => part.isNotEmpty).toList();
  if (parts.isEmpty) {
    return null;
  }
  final code = parts.last;
  final looksLikeCode = RegExp(r'^[A-Za-z0-9_-]+$').hasMatch(code);
  return looksLikeCode ? code : null;
}
