import 'package:dio/dio.dart';

/// Where the Go API lives during development. Override at build time:
/// `flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080`
/// (10.0.2.2 is how the Android emulator reaches the host machine).
const String apiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:8080',
);

/// The configured Dio instance.
///
/// Nothing calls this yet — the screens read fixtures from `core/mock/`. It is
/// here so the first person to wire a real endpoint has the base URL, timeouts
/// and the error envelope already handled in one place.
Dio createApiClient() {
  final dio = Dio(
    BaseOptions(
      baseUrl: apiBaseUrl,
      connectTimeout: const Duration(seconds: 5),
      receiveTimeout: const Duration(seconds: 10),
      contentType: 'application/json',
      // Let non-2xx through to the interceptor instead of throwing first, so the
      // error envelope can be read before an exception is raised.
      validateStatus: (status) => status != null && status < 500,
    ),
  );

  dio.interceptors.add(
    InterceptorsWrapper(
      onError: (error, handler) {
        handler.next(error.copyWith(message: describeApiError(error)));
      },
    ),
  );

  return dio;
}

/// Pulls the human-readable half out of the API's error envelope:
/// `{"error": {"code": "day_not_found", "message": "..."}}`.
///
/// Falls back to Dio's own message when the body is not an envelope — which
/// happens for timeouts and for a server that is simply not running.
String describeApiError(DioException error) {
  final body = error.response?.data;
  if (body is Map && body['error'] is Map) {
    final envelope = body['error'] as Map;
    final message = envelope['message'];
    if (message is String && message.isNotEmpty) return message;
  }
  return error.message ?? 'Network error';
}

/// The machine-readable half, for code that needs to branch on the failure.
String? apiErrorCode(DioException error) {
  final body = error.response?.data;
  if (body is Map && body['error'] is Map) {
    final code = (body['error'] as Map)['code'];
    if (code is String) return code;
  }
  return null;
}
