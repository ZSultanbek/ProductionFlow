import 'package:flutter/material.dart';

import '../../core/dates.dart';
import '../../core/models.dart';

// Two small pickers for moving a scene. They only choose; the caller sends the
// request, so these stay free of any backend code.

/// Which day to put a scene on. Pops with the day's id, or nothing.
Future<String?> pickDay(BuildContext context, List<ShootingDay> days) {
  return showDialog<String>(
    context: context,
    builder: (context) => SimpleDialog(
      title: const Text('Put on which day?'),
      children: [
        if (days.isEmpty)
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 24, vertical: 8),
            child: Text('Add a shooting day first.'),
          ),
        for (final day in days)
          SimpleDialogOption(
            onPressed: () => Navigator.of(context).pop(day.id),
            child: Text(
              '${formatShootDate(day.shootDate)} · call ${day.callTime}'
              '${day.location.isEmpty ? '' : ' · ${day.location}'}',
            ),
          ),
      ],
    ),
  );
}

/// Which unscheduled scene to put on a day. Pops with the scene's id.
Future<String?> pickScene(BuildContext context, List<Scene> scenes) {
  return showDialog<String>(
    context: context,
    builder: (context) => SimpleDialog(
      title: const Text('Which scene?'),
      children: [
        for (final scene in scenes)
          SimpleDialogOption(
            onPressed: () => Navigator.of(context).pop(scene.id),
            child: Text(
              scene.description.isEmpty ? scene.number : '${scene.number} — ${scene.description}',
            ),
          ),
      ],
    ),
  );
}
