import 'models.dart';

/// Everything the app asks the server for: one method per endpoint in
/// docs/api.md that the client uses so far.
///
/// Two implementations: `RealApi` (core/api_client.dart) talks to the Go server
/// over HTTP, `MockBackend` (core/mock/mock_api.dart) answers from fixture data
/// so the app runs with no server at all. Screens only ever see this interface,
/// so swapping one for the other changes nothing above this line.
///
/// Every method throws [ApiException] on failure and nothing else.
abstract class Backend {
  Future<AuthResult> register({
    required String email,
    required String password,
    required String displayName,
  });

  Future<AuthResult> login({required String email, required String password});

  /// The caller's projects, newest first.
  Future<List<Project>> projects();

  Future<Project> createProject(String name);

  Future<Project> project(String projectId);

  /// Public: works without being logged in.
  Future<Invite> invite(String code);

  /// [role] is `crew` or `cast`. Safe to call twice.
  Future<Participant> joinInvite(String code, String role);

  /// A project's days in shooting order.
  Future<List<ShootingDay>> days(String projectId);

  Future<ShootingDay> createDay(
    String projectId, {
    required String shootDate,
    required String callTime,
    required String location,
  });

  Future<CallSheet> callSheet(String dayId);

  /// [date] is the phone's local date, YYYY-MM-DD. The server never guesses it.
  Future<Today> today(String date);
}

/// The one error type screens see. It mirrors the API's error envelope:
/// a machine-readable [code] to branch on and a human-readable [message] to
/// show. Branch on [code], never on the HTTP status.
class ApiException implements Exception {
  const ApiException({
    required this.code,
    required this.message,
    this.fields = const {},
  });

  /// Not from the server: the request never got an answer.
  const ApiException.network()
    : code = networkCode,
      message = 'Could not reach the server. Check your connection and try again.',
      fields = const {};

  static const String networkCode = 'network';

  final String code;
  final String message;

  /// Set on `validation_failed`: field name to what is wrong with it.
  final Map<String, String> fields;

  @override
  String toString() => 'ApiException($code: $message)';
}
