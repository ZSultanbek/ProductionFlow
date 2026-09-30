import 'dart:convert';

import 'package:flutter/services.dart' show rootBundle;

/// Reads the fixture JSON in `fixtures/`, so the client can be built and
/// demoed with no Go server running.
///
/// The fixtures are hand-written to match `docs/api.md` exactly — same keys,
/// same date and time formats, same `{"data": [...], "next_cursor": null}`
/// envelope. When the contract changes, these change with it; that is the point
/// of them. Swapping a screen from mock to real should mean changing which
/// object it asks, and nothing else.
///
/// Nothing calls this yet.
class MockApi {
  const MockApi();

  static const String _dir = 'lib/core/mock/fixtures';

  /// `GET /api/projects`
  Future<Map<String, dynamic>> projects() => _load('project.json');

  /// `GET /api/days?project_id=…`
  Future<Map<String, dynamic>> shootingDays() => _load('shooting_days.json');

  /// Scenes for the project, including one with a null `shooting_day_id`.
  Future<Map<String, dynamic>> scenes() => _load('scenes.json');

  /// Participants, one per role plus a second crew member with no email.
  Future<Map<String, dynamic>> participants() => _load('participants.json');

  /// `POST /api/changes/preview` — moving day `a111…` two days later.
  Future<Map<String, dynamic>> changePreview() => _load('change_preview.json');

  /// The append-only change feed.
  Future<Map<String, dynamic>> changeFeed() => _load('change_feed.json');

  Future<Map<String, dynamic>> _load(String fileName) async {
    final raw = await rootBundle.loadString('$_dir/$fileName');
    return jsonDecode(raw) as Map<String, dynamic>;
  }
}
