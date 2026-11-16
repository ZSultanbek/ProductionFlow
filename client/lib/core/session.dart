import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'models.dart';

/// Being logged in: a token and the account it belongs to.
class Session {
  const Session({required this.token, required this.user});

  final String token;
  final User user;
}

/// Where the session is kept between app launches.
///
/// An interface so tests can use memory, and so the storage can be swapped
/// (for the Android Keystore, say) without touching anything else.
abstract class SessionStore {
  /// Synchronous on purpose: the router needs to know at the very first frame
  /// whether someone is logged in.
  Session? read();
  Future<void> write(Session session);
  Future<void> clear();
}

/// Keeps the session in shared_preferences: app-private storage on Android,
/// localStorage in the browser.
///
/// The token is not encrypted at rest. That is an accepted risk for this MVP —
/// it expires after 7 days and unlocks a shooting schedule, not a bank account —
/// and it is recorded in docs/architecture.md.
class PrefsSessionStore implements SessionStore {
  PrefsSessionStore(this._prefs);

  final SharedPreferences _prefs;
  static const String _key = 'session';

  @override
  Session? read() {
    final raw = _prefs.getString(_key);
    if (raw == null) {
      return null;
    }
    try {
      final json = jsonDecode(raw) as Map<String, dynamic>;
      return Session(
        token: json['token'] as String,
        user: User.fromJson(json['user'] as Map<String, dynamic>),
      );
    } on Object {
      // Written by an older build, or corrupted: treat as logged out.
      return null;
    }
  }

  @override
  Future<void> write(Session session) async {
    final raw = jsonEncode({'token': session.token, 'user': session.user.toJson()});
    await _prefs.setString(_key, raw);
  }

  @override
  Future<void> clear() async {
    await _prefs.remove(_key);
  }
}

/// A [SessionStore] that forgets everything when the app closes. For tests.
class MemorySessionStore implements SessionStore {
  MemorySessionStore([this._session]);

  Session? _session;

  @override
  Session? read() => _session;

  @override
  Future<void> write(Session session) async => _session = session;

  @override
  Future<void> clear() async => _session = null;
}

/// Overridden in main() with the real store, and in tests with a memory one.
final sessionStoreProvider = Provider<SessionStore>(
  (ref) => throw UnimplementedError('sessionStoreProvider must be overridden'),
);

/// The current session, or null when logged out. The router watches this to
/// decide between the app and the login screen.
final sessionProvider = NotifierProvider<SessionController, Session?>(SessionController.new);

class SessionController extends Notifier<Session?> {
  @override
  Session? build() => ref.read(sessionStoreProvider).read();

  /// Called after a successful login or registration.
  Future<void> signIn(AuthResult result) async {
    final session = Session(token: result.token, user: result.user);
    state = session;
    await ref.read(sessionStoreProvider).write(session);
  }

  /// Called by the Log out button, and by the API client when the server
  /// rejects the token. Safe to call when already logged out.
  Future<void> signOut() async {
    if (state == null) {
      return;
    }
    state = null;
    await ref.read(sessionStoreProvider).clear();
  }
}
