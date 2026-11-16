import 'dart:convert';

import 'package:flutter/services.dart' show rootBundle;

import '../backend.dart';
import '../dates.dart';
import '../models.dart';

/// A [Backend] that needs no server: it answers from the fixture JSON in
/// `fixtures/` and remembers what you add until the app is closed.
///
///   flutter run --dart-define=USE_MOCK=true
///
/// Any email and password log in, as Madina, the producer of "Dead Air". It is
/// there for working on screens without the Go server running, and as a way to
/// demo the app if the real server is asleep.
///
/// The fixtures are written to match docs/api.md exactly — same keys, same
/// formats — and are parsed by the same fromJson as real responses. When the
/// contract changes, they change with it.
class MockBackend implements Backend {
  MockBackend({required this.now});

  /// What time it is. Injected so the fixture days can be moved to "today".
  final DateTime Function() now;

  static const String _dir = 'lib/core/mock/fixtures';

  // Loaded once, on first use.
  Future<void>? _loading;
  late AuthResult _auth;
  final List<Project> _projects = [];
  final List<Participant> _participants = [];
  final List<ShootingDay> _days = [];
  final List<_MockScene> _scenes = [];

  /// Day id to the people on set all day.
  final Map<String, List<String>> _onSetAllDay = {};

  int _created = 0;

  Future<void> _ready() => _loading ??= _load();

  Future<void> _load() async {
    _auth = AuthResult.fromJson(await _read('auth.json'));

    final projects = await _read('project.json');
    for (final item in projects['data'] as List<dynamic>) {
      _projects.add(Project.fromJson(item as Map<String, dynamic>));
    }

    final participants = await _read('participants.json');
    for (final item in participants['data'] as List<dynamic>) {
      _participants.add(Participant.fromJson(item as Map<String, dynamic>));
    }

    // The fixture dates are fixed, and by now in the past. Move them so the
    // first day is today, the second tomorrow and the third next week —
    // otherwise the Today screen would have nothing to show.
    final today = now();
    const offsets = [0, 1, 5];
    final days = await _read('shooting_days.json');
    var index = 0;
    for (final item in days['data'] as List<dynamic>) {
      final day = ShootingDay.fromJson(item as Map<String, dynamic>);
      final offset = offsets[index % offsets.length] + 7 * (index ~/ offsets.length);
      final moved = DateTime(today.year, today.month, today.day + offset);
      _days.add(
        ShootingDay(
          id: day.id,
          projectId: day.projectId,
          shootDate: dateKey(moved),
          callTime: day.callTime,
          location: day.location,
          sceneCount: day.sceneCount,
        ),
      );
      index++;
    }

    final scenes = await _read('scenes.json');
    for (final item in scenes['data'] as List<dynamic>) {
      final json = item as Map<String, dynamic>;
      _scenes.add(
        _MockScene(
          dayId: json['shooting_day_id'] as String?,
          scene: CallSheetScene.fromJson(json),
        ),
      );
    }

    // Madina, the logged-in producer, is on set for the whole first day.
    if (_days.isNotEmpty && _projects.isNotEmpty) {
      _onSetAllDay[_days.first.id] = [_projects.first.myParticipantId];
    }
  }

  Future<Map<String, dynamic>> _read(String fileName) async {
    final raw = await rootBundle.loadString('$_dir/$fileName');
    return jsonDecode(raw) as Map<String, dynamic>;
  }

  // --- auth ---

  @override
  Future<AuthResult> login({required String email, required String password}) async {
    await _ready();
    return _auth;
  }

  @override
  Future<AuthResult> register({
    required String email,
    required String password,
    required String displayName,
  }) async {
    await _ready();
    return _auth;
  }

  // --- projects and invites ---

  @override
  Future<List<Project>> projects() async {
    await _ready();
    // Newest first, like the real API.
    return _projects.reversed.toList();
  }

  @override
  Future<Project> createProject(String name) async {
    await _ready();
    _created++;
    final code = 'mockinvite$_created';
    final project = Project(
      id: 'mock-project-$_created',
      name: name,
      myRole: Roles.producer,
      myParticipantId: 'mock-participant-$_created',
      inviteCode: code,
      inviteUrl: 'http://localhost:3000/join/$code',
    );
    _projects.add(project);
    return project;
  }

  @override
  Future<Project> project(String projectId) async {
    await _ready();
    for (final project in _projects) {
      if (project.id == projectId) {
        return project;
      }
    }
    throw const ApiException(code: 'project_not_found', message: 'no such project');
  }

  @override
  Future<Invite> invite(String code) async {
    await _ready();
    for (final project in _projects) {
      if (project.inviteCode == code) {
        return Invite(projectId: project.id, projectName: project.name);
      }
    }
    throw const ApiException(
      code: 'invite_not_found',
      message: 'this invite link does not work',
    );
  }

  @override
  Future<Participant> joinInvite(String code, String role) async {
    final invited = await invite(code);
    final me = await project(invited.projectId);
    // Already a member of every mock project, so joining changes nothing —
    // which is also what the real API does for a member.
    return Participant(
      id: me.myParticipantId,
      projectId: me.id,
      displayName: _auth.user.displayName,
      email: _auth.user.email,
      role: me.myRole,
      hasAccount: true,
    );
  }

  // --- shooting days ---

  @override
  Future<List<ShootingDay>> days(String projectId) async {
    await project(projectId);
    final list = [
      for (final day in _days)
        if (day.projectId == projectId) _withSceneCount(day),
    ];
    list.sort(_inShootingOrder);
    return list;
  }

  @override
  Future<ShootingDay> createDay(
    String projectId, {
    required String shootDate,
    required String callTime,
    required String location,
  }) async {
    await project(projectId);
    _created++;
    final day = ShootingDay(
      id: 'mock-day-$_created',
      projectId: projectId,
      shootDate: shootDate,
      callTime: callTime,
      location: location,
      sceneCount: 0,
    );
    _days.add(day);
    return day;
  }

  @override
  Future<CallSheet> callSheet(String dayId) async {
    await _ready();
    for (final day in _days) {
      if (day.id != dayId) {
        continue;
      }
      final parent = await project(day.projectId);
      final onSet = _onSetAllDay[day.id] ?? const <String>[];
      return CallSheet(
        id: day.id,
        projectId: parent.id,
        projectName: parent.name,
        myRole: parent.myRole,
        shootDate: day.shootDate,
        callTime: day.callTime,
        location: day.location,
        onSetAllDay: [
          for (final person in _participants)
            if (onSet.contains(person.id))
              ParticipantRef(id: person.id, displayName: person.displayName, role: person.role),
        ],
        scenes: [
          for (final entry in _scenes)
            if (entry.dayId == day.id) entry.scene,
        ],
      );
    }
    throw const ApiException(code: 'day_not_found', message: 'no such shooting day');
  }

  @override
  Future<Today> today(String date) async {
    await _ready();

    // The same rule as the server: the given date if anything shoots on it,
    // otherwise the next date that has something.
    String? target;
    for (final day in _days) {
      final upcoming = day.shootDate.compareTo(date) >= 0;
      if (upcoming && (target == null || day.shootDate.compareTo(target) < 0)) {
        target = day.shootDate;
      }
    }
    if (target == null) {
      return const Today(date: null, isToday: false, days: []);
    }

    final onDate = [
      for (final day in _days)
        if (day.shootDate == target) day,
    ];
    onDate.sort(_inShootingOrder);

    final agenda = <AgendaDay>[];
    for (final day in onDate) {
      final parent = await project(day.projectId);
      final me = parent.myParticipantId;
      agenda.add(
        AgendaDay(
          id: day.id,
          projectId: parent.id,
          projectName: parent.name,
          shootDate: day.shootDate,
          callTime: day.callTime,
          location: day.location,
          myScenes: [
            for (final entry in _scenes)
              if (entry.dayId == day.id && entry.scene.participants.any((p) => p.id == me))
                SceneRef(
                  id: entry.scene.id,
                  number: entry.scene.number,
                  description: entry.scene.description,
                ),
          ],
          iAmOnSetAllDay: (_onSetAllDay[day.id] ?? const <String>[]).contains(me),
        ),
      );
    }
    return Today(date: target, isToday: target == date, days: agenda);
  }

  // --- helpers ---

  ShootingDay _withSceneCount(ShootingDay day) {
    return ShootingDay(
      id: day.id,
      projectId: day.projectId,
      shootDate: day.shootDate,
      callTime: day.callTime,
      location: day.location,
      sceneCount: _scenes.where((entry) => entry.dayId == day.id).length,
    );
  }

  /// By date, then call time. Both are zero-padded strings, so comparing them
  /// as text sorts them correctly.
  static int _inShootingOrder(ShootingDay a, ShootingDay b) {
    final byDate = a.shootDate.compareTo(b.shootDate);
    return byDate != 0 ? byDate : a.callTime.compareTo(b.callTime);
  }
}

class _MockScene {
  const _MockScene({required this.dayId, required this.scene});

  /// Null for an unscheduled scene.
  final String? dayId;
  final CallSheetScene scene;
}
