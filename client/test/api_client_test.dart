import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:productionflow/core/api_client.dart';
import 'package:productionflow/core/backend.dart';

/// Answers requests from a script instead of the network, and remembers what
/// was asked.
class ScriptedAdapter implements HttpClientAdapter {
  ScriptedAdapter(this.answer);

  /// Returns (status, JSON body) for a request. Throw to simulate no network.
  final (int, Object?) Function(RequestOptions request) answer;

  final List<RequestOptions> requests = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    final (status, body) = answer(options);
    return ResponseBody.fromString(
      jsonEncode(body),
      status,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

Map<String, dynamic> projectJson(String id) => {
  'id': id,
  'name': 'Project $id',
  'my_role': 'crew',
  'my_participant_id': 'pa-$id',
  'invite_code': null,
  'invite_url': null,
  'created_at': '2026-09-01T09:00:00Z',
};

void main() {
  late int signOuts;
  String? token;

  RealApi apiWith(ScriptedAdapter adapter) {
    final dio = Dio(BaseOptions(baseUrl: 'http://api.test'))..httpClientAdapter = adapter;
    return RealApi(
      baseUrl: 'http://api.test',
      token: () => token,
      onUnauthorized: () => signOuts++,
      dio: dio,
    );
  }

  setUp(() {
    signOuts = 0;
    token = 'secret-token';
  });

  test('sends the token on authenticated calls', () async {
    final adapter = ScriptedAdapter((request) => (200, projectJson('p1')));
    await apiWith(adapter).project('p1');

    expect(adapter.requests.single.path, '/api/projects/p1');
    expect(adapter.requests.single.headers['Authorization'], 'Bearer secret-token');
  });

  test('does not send the token to login, register or the public invite lookup', () async {
    final adapter = ScriptedAdapter((request) {
      if (request.path.startsWith('/api/invites/')) {
        return (200, {'project_id': 'p1', 'project_name': 'Dead Air'});
      }
      return (
        200,
        {
          'token': 't',
          'expires_at': '2026-11-09T09:00:00Z',
          'user': {'id': 'u1', 'email': 'a@b.kz', 'display_name': 'A'},
        },
      );
    });
    final api = apiWith(adapter);

    await api.login(email: 'a@b.kz', password: 'correct horse');
    await api.register(email: 'a@b.kz', password: 'correct horse', displayName: 'A');
    await api.invite('abc');

    for (final request in adapter.requests) {
      expect(request.headers.containsKey('Authorization'), isFalse, reason: request.path);
    }
  });

  test('turns the error envelope into an ApiException with fields', () async {
    final adapter = ScriptedAdapter(
      (request) => (
        422,
        {
          'error': {
            'code': 'validation_failed',
            'message': 'some fields are invalid',
            'fields': {'call_time': 'must be HH:MM, 24-hour'},
          },
        },
      ),
    );

    await expectLater(
      apiWith(adapter).createDay('p1', shootDate: '2026-10-04', callTime: '7am', location: ''),
      throwsA(
        isA<ApiException>()
            .having((e) => e.code, 'code', 'validation_failed')
            .having((e) => e.message, 'message', 'some fields are invalid')
            .having((e) => e.fields, 'fields', {'call_time': 'must be HH:MM, 24-hour'}),
      ),
    );
    expect(signOuts, 0);
  });

  test('a rejected token logs the user out', () async {
    final adapter = ScriptedAdapter(
      (request) => (
        401,
        {
          'error': {'code': 'unauthorized', 'message': 'log in again'},
        },
      ),
    );

    await expectLater(
      apiWith(adapter).projects(),
      throwsA(isA<ApiException>().having((e) => e.code, 'code', 'unauthorized')),
    );
    expect(signOuts, 1);
  });

  // A wrong password is also a 401, but it is not a dead session.
  test('a wrong password at login does not log anyone out', () async {
    final adapter = ScriptedAdapter(
      (request) => (
        401,
        {
          'error': {'code': 'invalid_credentials', 'message': 'wrong email or password'},
        },
      ),
    );

    await expectLater(
      apiWith(adapter).login(email: 'a@b.kz', password: 'nope'),
      throwsA(isA<ApiException>().having((e) => e.code, 'code', 'invalid_credentials')),
    );
    expect(signOuts, 0);
  });

  test('no answer at all is a network error', () async {
    final adapter = ScriptedAdapter((request) => throw Exception('connection refused'));

    await expectLater(
      apiWith(adapter).projects(),
      throwsA(isA<ApiException>().having((e) => e.code, 'code', ApiException.networkCode)),
    );
  });

  test('an error that is not our envelope becomes a generic one', () async {
    final adapter = ScriptedAdapter((request) => (502, 'Bad Gateway'));

    await expectLater(
      apiWith(adapter).projects(),
      throwsA(isA<ApiException>().having((e) => e.code, 'code', 'internal')),
    );
  });

  test('lists follow next_cursor until the last page', () async {
    final adapter = ScriptedAdapter((request) {
      final cursor = request.queryParameters['cursor'];
      if (cursor == null) {
        return (
          200,
          {
            'data': [projectJson('p1'), projectJson('p2')],
            'next_cursor': 'page-2',
          },
        );
      }
      return (
        200,
        {
          'data': [projectJson('p3')],
          'next_cursor': null,
        },
      );
    });

    final projects = await apiWith(adapter).projects();

    expect(projects.map((p) => p.id), ['p1', 'p2', 'p3']);
    expect(adapter.requests, hasLength(2));
    expect(adapter.requests.first.queryParameters.containsKey('cursor'), isFalse);
    expect(adapter.requests.last.queryParameters['cursor'], 'page-2');
  });

  test('today sends the date it was given', () async {
    final adapter = ScriptedAdapter(
      (request) => (200, {'date': null, 'is_today': false, 'days': <dynamic>[]}),
    );
    await apiWith(adapter).today('2026-11-02');

    expect(adapter.requests.single.path, '/api/me/today');
    expect(adapter.requests.single.queryParameters['date'], '2026-11-02');
  });

  test('a move preview sends only the fields that change', () async {
    final adapter = ScriptedAdapter(
      (request) => (
        200,
        {
          'preview_token': 'tok',
          'expires_at': '2026-11-02T09:10:00Z',
          'change': {
            'kind': 'move_shooting_day',
            'shooting_day_id': 'd1',
            'diff': {
              'location': {'from': 'Kok-Tobe', 'to': 'Studio B'},
            },
          },
          'affected': {'scenes': [], 'participants': []},
          'will_notify_count': 0,
        },
      ),
    );

    final preview = await apiWith(adapter).previewMoveDay('d1', location: 'Studio B');

    final request = adapter.requests.single;
    expect(request.method, 'POST');
    expect(request.path, '/api/changes/preview');
    expect(request.data, {
      'kind': 'move_shooting_day',
      'shooting_day_id': 'd1',
      'to': {'location': 'Studio B'},
    });
    expect(preview.diff['location']?.to, 'Studio B');
  });

  test('taking a scene off its day sends an explicit null', () async {
    final adapter = ScriptedAdapter(
      (request) => (
        200,
        {
          'id': 's1',
          'project_id': 'p1',
          'shooting_day_id': null,
          'number': '14',
          'description': 'Kitchen',
          'participants': [],
        },
      ),
    );

    final scene = await apiWith(adapter).moveScene('s1', null);

    final request = adapter.requests.single;
    expect(request.method, 'PATCH');
    expect(request.data, {'shooting_day_id': null});
    expect(scene.shootingDayId, isNull);
  });
}
