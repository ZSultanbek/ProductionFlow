import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'api_client.dart';
import 'backend.dart';
import 'dates.dart';
import 'mock/mock_api.dart';
import 'models.dart';
import 'session.dart';

/// Run with fixture data and no server:
///
///   flutter run --dart-define=USE_MOCK=true
const bool useMock = bool.fromEnvironment('USE_MOCK');

/// The [Backend] every screen talks to. Tests override it with a fake.
final backendProvider = Provider<Backend>((ref) {
  if (useMock) {
    return MockBackend(now: ref.read(clockProvider));
  }
  return RealApi(
    baseUrl: resolveApiBaseUrl(),
    // Read at request time, not captured here: a login or logout must take
    // effect on the very next call.
    token: () => ref.read(sessionProvider)?.token,
    onUnauthorized: () => ref.read(sessionProvider.notifier).signOut(),
  );
});

/// What time it is. A provider so tests can pin "today" to a known date.
final clockProvider = Provider<DateTime Function()>((ref) => DateTime.now);

/// The id of whoever is logged in. Data providers watch it so that nothing
/// loaded for one account is ever shown to the next one on the same device.
final _userIdProvider = Provider<String?>((ref) => ref.watch(sessionProvider)?.user.id);

// The data each screen shows. All of them are autoDispose: when the last screen
// showing something closes, it is dropped and fetched fresh next time. After a
// change, the screen that made it calls ref.invalidate on what it affected.

final projectsProvider = FutureProvider.autoDispose<List<Project>>((ref) {
  ref.watch(_userIdProvider);
  return ref.watch(backendProvider).projects();
});

final projectProvider = FutureProvider.autoDispose.family<Project, String>((ref, projectId) {
  ref.watch(_userIdProvider);
  return ref.watch(backendProvider).project(projectId);
});

final daysProvider = FutureProvider.autoDispose.family<List<ShootingDay>, String>((ref, projectId) {
  ref.watch(_userIdProvider);
  return ref.watch(backendProvider).days(projectId);
});

final callSheetProvider = FutureProvider.autoDispose.family<CallSheet, String>((ref, dayId) {
  ref.watch(_userIdProvider);
  return ref.watch(backendProvider).callSheet(dayId);
});

final participantsProvider = FutureProvider.autoDispose.family<List<Participant>, String>((
  ref,
  projectId,
) {
  ref.watch(_userIdProvider);
  return ref.watch(backendProvider).participants(projectId);
});

/// Every scene of a project, on a day or not.
final scenesProvider = FutureProvider.autoDispose.family<List<Scene>, String>((ref, projectId) {
  ref.watch(_userIdProvider);
  return ref.watch(backendProvider).scenes(projectId);
});

/// Today's days for the logged-in user. "Today" is the phone's local date.
final todayProvider = FutureProvider.autoDispose<Today>((ref) {
  ref.watch(_userIdProvider);
  final today = dateKey(ref.watch(clockProvider)());
  return ref.watch(backendProvider).today(today);
});

/// Public: what an invite link points at.
final inviteProvider = FutureProvider.autoDispose.family<Invite, String>((ref, code) {
  return ref.watch(backendProvider).invite(code);
});
