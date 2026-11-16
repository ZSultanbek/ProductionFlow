/// Every address in the app, in one place.
abstract final class Routes {
  static const String landing = '/';
  static const String login = '/login';
  static const String register = '/register';
  static const String today = '/app';
  static const String projects = '/app/projects';

  static String join(String code) => '/join/$code';
  static String project(String projectId) => '/app/projects/$projectId';
  static String day(String projectId, String dayId) => '/app/projects/$projectId/days/$dayId';

  /// The login or register address that returns to [from] afterwards.
  static String withFrom(String path, String? from) {
    if (from == null) {
      return path;
    }
    return Uri(path: path, queryParameters: {'from': from}).toString();
  }
}

/// Where to go after logging in. Only addresses inside this app are accepted:
/// a `from` that came in through a link must not be able to send someone to
/// another site.
String? safeFrom(String? from) {
  if (from == null || !from.startsWith('/') || from.startsWith('//')) {
    return null;
  }
  return from;
}
