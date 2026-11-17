import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'backend.dart';
import 'models.dart';

/// Where the Go API lives. Set at build time:
///
///   flutter run --dart-define=API_BASE_URL=http://192.168.1.20:8080
///
/// An empty value means "same origin": the web build served by the compose
/// nginx calls /api/... on its own host, and nginx proxies it to the API.
String resolveApiBaseUrl() {
  if (const bool.hasEnvironment('API_BASE_URL')) {
    return const String.fromEnvironment('API_BASE_URL');
  }
  // Nothing configured: assume a Go server running on the developer's machine.
  // From the Android emulator, the host machine is 10.0.2.2, not localhost.
  if (!kIsWeb && defaultTargetPlatform == TargetPlatform.android) {
    return 'http://10.0.2.2:8080';
  }
  return 'http://localhost:8080';
}

/// The real [Backend]: HTTP calls to the Go server.
class RealApi implements Backend {
  /// [token] is asked for before every request, so a login or logout takes
  /// effect immediately. [onUnauthorized] is called when the server says the
  /// token is no longer good; the app answers by logging out.
  RealApi({
    required String baseUrl,
    required this.token,
    required this.onUnauthorized,
    Dio? dio,
  }) : _dio =
           dio ??
           Dio(
             BaseOptions(
               baseUrl: baseUrl,
               // Generous on purpose. The free Render instance sleeps after 15
               // idle minutes and takes about a minute to wake; a short timeout
               // would turn every first request of the day into an error.
               connectTimeout: const Duration(seconds: 75),
               receiveTimeout: const Duration(seconds: 75),
               contentType: 'application/json',
             ),
           );

  final Dio _dio;
  final String? Function() token;
  final void Function() onUnauthorized;

  // --- auth ---

  @override
  Future<AuthResult> register({
    required String email,
    required String password,
    required String displayName,
  }) async {
    final body = await _send('POST', '/api/auth/register', auth: false, data: {
      'email': email,
      'password': password,
      'display_name': displayName,
    });
    return AuthResult.fromJson(body);
  }

  @override
  Future<AuthResult> login({required String email, required String password}) async {
    final body = await _send('POST', '/api/auth/login', auth: false, data: {
      'email': email,
      'password': password,
    });
    return AuthResult.fromJson(body);
  }

  // --- projects and invites ---

  @override
  Future<List<Project>> projects() => _all('/api/projects', Project.fromJson);

  @override
  Future<Project> createProject(String name) async {
    return Project.fromJson(await _send('POST', '/api/projects', data: {'name': name}));
  }

  @override
  Future<Project> project(String projectId) async {
    return Project.fromJson(await _send('GET', '/api/projects/$projectId'));
  }

  @override
  Future<Invite> invite(String code) async {
    return Invite.fromJson(await _send('GET', '/api/invites/$code', auth: false));
  }

  @override
  Future<Participant> joinInvite(String code, String role) async {
    return Participant.fromJson(
      await _send('POST', '/api/invites/$code/join', data: {'role': role}),
    );
  }

  // --- shooting days ---

  @override
  Future<List<ShootingDay>> days(String projectId) {
    return _all('/api/projects/$projectId/days', ShootingDay.fromJson);
  }

  @override
  Future<ShootingDay> createDay(
    String projectId, {
    required String shootDate,
    required String callTime,
    required String location,
  }) async {
    final body = await _send('POST', '/api/projects/$projectId/days', data: {
      'shoot_date': shootDate,
      'call_time': callTime,
      'location': location,
    });
    return ShootingDay.fromJson(body);
  }

  @override
  Future<CallSheet> callSheet(String dayId) async {
    return CallSheet.fromJson(await _send('GET', '/api/days/$dayId'));
  }

  @override
  Future<Today> today(String date) async {
    return Today.fromJson(await _send('GET', '/api/me/today', query: {'date': date}));
  }

  // --- people and scenes ---

  @override
  Future<List<Participant>> participants(String projectId) {
    return _all('/api/projects/$projectId/participants', Participant.fromJson);
  }

  @override
  Future<Participant> addParticipant(
    String projectId, {
    required String displayName,
    required String role,
    String? email,
  }) async {
    final body = await _send('POST', '/api/projects/$projectId/participants', data: {
      'display_name': displayName,
      'role': role,
      'email': email,
    });
    return Participant.fromJson(body);
  }

  @override
  Future<List<ParticipantRef>> setDayParticipants(String dayId, List<String> participantIds) async {
    final body = await _send('PUT', '/api/days/$dayId/participants', data: {
      'participant_ids': participantIds,
    });
    return [
      for (final item in body['on_set_all_day'] as List<dynamic>)
        ParticipantRef.fromJson(item as Map<String, dynamic>),
    ];
  }

  @override
  Future<List<Scene>> scenes(String projectId) {
    return _all('/api/projects/$projectId/scenes', Scene.fromJson);
  }

  @override
  Future<Scene> createScene(
    String projectId, {
    required String number,
    required String description,
    String? shootingDayId,
  }) async {
    final body = await _send('POST', '/api/projects/$projectId/scenes', data: {
      'number': number,
      'description': description,
      'shooting_day_id': shootingDayId,
    });
    return Scene.fromJson(body);
  }

  @override
  Future<Scene> moveScene(String sceneId, String? shootingDayId) async {
    // null is sent as JSON null, which the API reads as "take it off its day".
    final body = await _send('PATCH', '/api/scenes/$sceneId', data: {
      'shooting_day_id': shootingDayId,
    });
    return Scene.fromJson(body);
  }

  @override
  Future<Scene> setSceneParticipants(String sceneId, List<String> participantIds) async {
    final body = await _send('PUT', '/api/scenes/$sceneId/participants', data: {
      'participant_ids': participantIds,
    });
    return Scene.fromJson(body);
  }

  // --- the impact view ---

  @override
  Future<ChangePreview> previewMoveDay(
    String dayId, {
    String? shootDate,
    String? callTime,
    String? location,
  }) async {
    final body = await _send('POST', '/api/changes/preview', data: {
      'kind': 'move_shooting_day',
      'shooting_day_id': dayId,
      'to': {
        'shoot_date': ?shootDate,
        'call_time': ?callTime,
        'location': ?location,
      },
    });
    return ChangePreview.fromJson(body);
  }

  // --- plumbing ---

  /// One request. Returns the decoded JSON object, or throws [ApiException].
  Future<Map<String, dynamic>> _send(
    String method,
    String path, {
    Map<String, dynamic>? data,
    Map<String, dynamic>? query,
    bool auth = true,
  }) async {
    final bearer = auth ? token() : null;
    try {
      final response = await _dio.request<dynamic>(
        path,
        data: data,
        queryParameters: query,
        options: Options(
          method: method,
          headers: {if (bearer != null) 'Authorization': 'Bearer $bearer'},
        ),
      );
      final body = response.data;
      if (body is Map<String, dynamic>) {
        return body;
      }
      throw const ApiException(code: 'internal', message: 'The server sent an unexpected reply.');
    } on DioException catch (error) {
      final failure = _toApiException(error);
      // Only an authenticated call can discover that the session has ended.
      // A wrong password at login is `invalid_credentials`, not this.
      if (auth && failure.code == 'unauthorized') {
        onUnauthorized();
      }
      throw failure;
    }
  }

  /// Every page of a list endpoint, following `next_cursor` until it is null.
  Future<List<T>> _all<T>(String path, T Function(Map<String, dynamic>) parse) async {
    final items = <T>[];
    String? cursor;
    do {
      final body = await _send('GET', path, query: {
        'limit': 100,
        'cursor': ?cursor,
      });
      for (final item in body['data'] as List<dynamic>) {
        items.add(parse(item as Map<String, dynamic>));
      }
      cursor = body['next_cursor'] as String?;
    } while (cursor != null);
    return items;
  }
}

/// Turns whatever went wrong into the one error type screens understand.
ApiException _toApiException(DioException error) {
  final response = error.response;
  if (response == null) {
    // No answer at all: offline, DNS, timeout, server down.
    return const ApiException.network();
  }

  // The API's envelope: {"error": {"code": ..., "message": ..., "fields": ...}}
  final body = response.data;
  if (body is Map && body['error'] is Map) {
    final envelope = body['error'] as Map;
    final fields = <String, String>{};
    final rawFields = envelope['fields'];
    if (rawFields is Map) {
      for (final entry in rawFields.entries) {
        fields['${entry.key}'] = '${entry.value}';
      }
    }
    return ApiException(
      code: '${envelope['code']}',
      message: '${envelope['message']}',
      fields: fields,
    );
  }

  // An answer that is not our envelope — a proxy's error page, for instance.
  return const ApiException(
    code: 'internal',
    message: 'The server had a problem. Try again in a moment.',
  );
}
