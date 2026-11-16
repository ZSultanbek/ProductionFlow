import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:productionflow/core/models.dart';

// The fixtures are written to match docs/api.md. Parsing them with the real
// fromJson is the check that the models and the contract agree.

Map<String, dynamic> fixture(String name) {
  final raw = File('lib/core/mock/fixtures/$name').readAsStringSync();
  return jsonDecode(raw) as Map<String, dynamic>;
}

List<Map<String, dynamic>> dataOf(String name) {
  return [for (final item in fixture(name)['data'] as List<dynamic>) item as Map<String, dynamic>];
}

void main() {
  test('auth fixture parses', () {
    final auth = AuthResult.fromJson(fixture('auth.json'));
    expect(auth.token, isNotEmpty);
    expect(auth.user.displayName, 'Madina');
    expect(auth.user.email, 'madina@example.kz');
  });

  test('a user survives being stored and read back', () {
    const user = User(id: 'u1', email: 'a@b.kz', displayName: 'Aisha');
    final copy = User.fromJson(jsonDecode(jsonEncode(user.toJson())) as Map<String, dynamic>);
    expect(copy.id, 'u1');
    expect(copy.email, 'a@b.kz');
    expect(copy.displayName, 'Aisha');
  });

  test('project fixture parses, with the invite a producer sees', () {
    final project = Project.fromJson(dataOf('project.json').single);
    expect(project.name, 'Dead Air');
    expect(project.iAmProducer, isTrue);
    expect(project.inviteCode, isNotNull);
    expect(project.inviteUrl, endsWith('/join/${project.inviteCode}'));
  });

  test('a project seen by crew has no invite', () {
    final project = Project.fromJson({
      'id': 'p1',
      'name': 'Dead Air',
      'my_role': 'crew',
      'my_participant_id': 'pa1',
      'invite_code': null,
      'invite_url': null,
      'created_at': '2026-09-01T09:00:00Z',
    });
    expect(project.iAmProducer, isFalse);
    expect(project.inviteCode, isNull);
  });

  test('participants fixture parses, including one without an account or email', () {
    final people = dataOf('participants.json').map(Participant.fromJson).toList();
    expect(people, hasLength(4));
    final ruslan = people.firstWhere((p) => p.displayName == 'Ruslan');
    expect(ruslan.hasAccount, isFalse);
    expect(ruslan.email, isNull);
  });

  test('shooting days keep date and call time as the exact strings sent', () {
    final days = dataOf('shooting_days.json').map(ShootingDay.fromJson).toList();
    expect(days, hasLength(3));
    expect(days.first.shootDate, '2026-10-04');
    expect(days.first.callTime, '07:30');
    expect(days.first.sceneCount, 3);
  });

  test('scenes fixture parses as call sheet scenes', () {
    final scenes = dataOf('scenes.json').map(CallSheetScene.fromJson).toList();
    final first = scenes.firstWhere((s) => s.number == '12A');
    expect(first.participants.map((p) => p.displayName), ['Daniyar', 'Aisha']);
    final unscheduled = scenes.firstWhere((s) => s.number == '21');
    expect(unscheduled.participants, isEmpty);
  });

  // The two shapes below are copied from the examples in docs/api.md.

  test('call sheet parses', () {
    final sheet = CallSheet.fromJson({
      'id': 'd1',
      'project': {'id': 'p1', 'name': 'Dead Air', 'my_role': 'crew'},
      'shoot_date': '2026-10-04',
      'call_time': '07:30',
      'location': 'Kok-Tobe, upper lot',
      'updated_at': '2026-10-01T12:00:00Z',
      'on_set_all_day': [
        {'id': 'pa2', 'display_name': 'Daniyar', 'role': 'crew'},
      ],
      'scenes': [
        {
          'id': 's1',
          'number': '12A',
          'description': 'Rooftop argument, dusk',
          'participants': [
            {'id': 'pa1', 'display_name': 'Aisha', 'role': 'cast'},
          ],
        },
      ],
    });
    expect(sheet.projectName, 'Dead Air');
    expect(sheet.myRole, 'crew');
    expect(sheet.callTime, '07:30');
    expect(sheet.onSetAllDay.single.displayName, 'Daniyar');
    expect(sheet.scenes.single.participants.single.role, 'cast');
  });

  test('today parses, with and without a date', () {
    final today = Today.fromJson({
      'date': '2026-11-02',
      'is_today': true,
      'days': [
        {
          'id': 'd1',
          'project': {'id': 'p1', 'name': 'Dead Air'},
          'shoot_date': '2026-11-02',
          'call_time': '07:30',
          'location': 'Kok-Tobe, upper lot',
          'my_scenes': [
            {'id': 's1', 'number': '12A', 'description': 'Rooftop argument, dusk'},
          ],
          'i_am_on_set_all_day': false,
        },
      ],
    });
    expect(today.isToday, isTrue);
    expect(today.days.single.myScenes.single.number, '12A');
    expect(today.days.single.projectId, 'p1');

    final nothing = Today.fromJson({'date': null, 'is_today': false, 'days': <dynamic>[]});
    expect(nothing.date, isNull);
    expect(nothing.days, isEmpty);
  });
}
