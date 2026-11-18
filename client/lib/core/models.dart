// The shapes in docs/api.md, as Dart classes.
//
// fromJson is written by hand rather than generated: there are a dozen small
// classes, and hand-written parsing needs no build step and no generated files
// to keep in sync. When the contract changes, this file changes with it.
//
// shoot_date ("2026-10-04") and call_time ("07:30") stay Strings everywhere.
// They are wall-clock values, not instants — parsing them into DateTime invites
// a toLocal() somewhere that silently moves a call time by five hours. Format
// them for display with core/dates.dart and never convert them.

class User {
  const User({required this.id, required this.email, required this.displayName});

  factory User.fromJson(Map<String, dynamic> json) => User(
    id: json['id'] as String,
    email: json['email'] as String,
    displayName: json['display_name'] as String,
  );

  final String id;
  final String email;
  final String displayName;

  Map<String, dynamic> toJson() => {'id': id, 'email': email, 'display_name': displayName};
}

/// What register and login return: a token and who it belongs to.
class AuthResult {
  const AuthResult({required this.token, required this.user});

  factory AuthResult.fromJson(Map<String, dynamic> json) => AuthResult(
    token: json['token'] as String,
    user: User.fromJson(json['user'] as Map<String, dynamic>),
  );

  final String token;
  final User user;
}

/// The three roles. Kept as the API's strings; these constants avoid typos.
abstract final class Roles {
  static const String producer = 'producer';
  static const String crew = 'crew';
  static const String cast = 'cast';
}

class Project {
  const Project({
    required this.id,
    required this.name,
    required this.myRole,
    required this.myParticipantId,
    this.inviteCode,
    this.inviteUrl,
  });

  factory Project.fromJson(Map<String, dynamic> json) => Project(
    id: json['id'] as String,
    name: json['name'] as String,
    myRole: json['my_role'] as String,
    myParticipantId: json['my_participant_id'] as String,
    inviteCode: json['invite_code'] as String?,
    inviteUrl: json['invite_url'] as String?,
  );

  final String id;
  final String name;
  final String myRole;
  final String myParticipantId;

  /// Null unless I am the producer: only producers may hand out the invite.
  final String? inviteCode;
  final String? inviteUrl;

  bool get iAmProducer => myRole == Roles.producer;
}

/// What an invite link points at, readable before logging in.
class Invite {
  const Invite({required this.projectId, required this.projectName});

  factory Invite.fromJson(Map<String, dynamic> json) => Invite(
    projectId: json['project_id'] as String,
    projectName: json['project_name'] as String,
  );

  final String projectId;
  final String projectName;
}

class Participant {
  const Participant({
    required this.id,
    required this.projectId,
    required this.displayName,
    required this.role,
    required this.hasAccount,
    this.email,
  });

  factory Participant.fromJson(Map<String, dynamic> json) => Participant(
    id: json['id'] as String,
    projectId: json['project_id'] as String,
    displayName: json['display_name'] as String,
    role: json['role'] as String,
    hasAccount: json['has_account'] as bool,
    email: json['email'] as String?,
  );

  final String id;
  final String projectId;
  final String displayName;
  final String role;
  final bool hasAccount;
  final String? email;
}

/// The short form of a participant shown inside other things.
class ParticipantRef {
  const ParticipantRef({required this.id, required this.displayName, required this.role});

  factory ParticipantRef.fromJson(Map<String, dynamic> json) => ParticipantRef(
    id: json['id'] as String,
    displayName: json['display_name'] as String,
    role: json['role'] as String,
  );

  final String id;
  final String displayName;
  final String role;
}

/// The short form of a scene.
class SceneRef {
  const SceneRef({required this.id, required this.number, required this.description});

  factory SceneRef.fromJson(Map<String, dynamic> json) => SceneRef(
    id: json['id'] as String,
    number: json['number'] as String,
    description: json['description'] as String,
  );

  final String id;
  final String number;
  final String description;
}

class ShootingDay {
  const ShootingDay({
    required this.id,
    required this.projectId,
    required this.shootDate,
    required this.callTime,
    required this.location,
    required this.sceneCount,
  });

  factory ShootingDay.fromJson(Map<String, dynamic> json) => ShootingDay(
    id: json['id'] as String,
    projectId: json['project_id'] as String,
    shootDate: json['shoot_date'] as String,
    callTime: json['call_time'] as String,
    location: json['location'] as String,
    sceneCount: json['scene_count'] as int,
  );

  final String id;
  final String projectId;

  /// YYYY-MM-DD. A String on purpose — see the note at the top of this file.
  final String shootDate;

  /// HH:MM, 24-hour. A String on purpose.
  final String callTime;
  final String location;
  final int sceneCount;
}

class CallSheetScene {
  const CallSheetScene({
    required this.id,
    required this.number,
    required this.description,
    required this.participants,
  });

  factory CallSheetScene.fromJson(Map<String, dynamic> json) => CallSheetScene(
    id: json['id'] as String,
    number: json['number'] as String,
    description: json['description'] as String,
    participants: _list(json['participants'], ParticipantRef.fromJson),
  );

  final String id;
  final String number;
  final String description;
  final List<ParticipantRef> participants;
}

/// One shooting day as the whole crew sees it: GET /api/days/{day_id}.
class CallSheet {
  const CallSheet({
    required this.id,
    required this.projectId,
    required this.projectName,
    required this.myRole,
    required this.shootDate,
    required this.callTime,
    required this.location,
    required this.onSetAllDay,
    required this.scenes,
  });

  factory CallSheet.fromJson(Map<String, dynamic> json) {
    final project = json['project'] as Map<String, dynamic>;
    return CallSheet(
      id: json['id'] as String,
      projectId: project['id'] as String,
      projectName: project['name'] as String,
      myRole: project['my_role'] as String,
      shootDate: json['shoot_date'] as String,
      callTime: json['call_time'] as String,
      location: json['location'] as String,
      onSetAllDay: _list(json['on_set_all_day'], ParticipantRef.fromJson),
      scenes: _list(json['scenes'], CallSheetScene.fromJson),
    );
  }

  final String id;
  final String projectId;
  final String projectName;
  final String myRole;
  final String shootDate;
  final String callTime;
  final String location;

  /// People on set for the whole day, whichever scenes are shot.
  final List<ParticipantRef> onSetAllDay;
  final List<CallSheetScene> scenes;
}

/// A scene of the script (US-06). [shootingDayId] is null while the scene is
/// not on any day yet.
class Scene {
  const Scene({
    required this.id,
    required this.projectId,
    required this.shootingDayId,
    required this.number,
    required this.description,
    required this.participants,
  });

  factory Scene.fromJson(Map<String, dynamic> json) => Scene(
    id: json['id'] as String,
    projectId: json['project_id'] as String,
    shootingDayId: json['shooting_day_id'] as String?,
    number: json['number'] as String,
    description: json['description'] as String,
    participants: _list(json['participants'], ParticipantRef.fromJson),
  );

  final String id;
  final String projectId;
  final String? shootingDayId;
  final String number;
  final String description;
  final List<ParticipantRef> participants;
}

/// One field of a change: what it was and what it would become.
class FieldChange {
  const FieldChange({required this.from, required this.to});

  factory FieldChange.fromJson(Map<String, dynamic> json) =>
      FieldChange(from: json['from'] as String?, to: json['to'] as String?);

  final String? from;
  final String? to;
}

/// Someone a change reaches, and every reason why (US-07).
class AffectedParticipant {
  const AffectedParticipant({
    required this.id,
    required this.displayName,
    required this.role,
    required this.hasAccount,
    required this.onSetAllDay,
    required this.sceneNumbers,
  });

  factory AffectedParticipant.fromJson(Map<String, dynamic> json) {
    final via = json['via'] as Map<String, dynamic>;
    return AffectedParticipant(
      id: json['id'] as String,
      displayName: json['display_name'] as String,
      role: json['role'] as String,
      hasAccount: json['has_account'] as bool,
      onSetAllDay: via['day'] as bool,
      sceneNumbers: [for (final n in via['scene_numbers'] as List<dynamic>) n as String],
    );
  }

  final String id;
  final String displayName;
  final String role;

  /// False for someone added by name: they will not get a notification, so
  /// the producer has to tell them.
  final bool hasAccount;
  final bool onSetAllDay;
  final List<String> sceneNumbers;
}

/// POST /api/changes/preview: who and what moving a day would affect. Nothing
/// has changed yet when the app has one of these.
class ChangePreview {
  const ChangePreview({
    required this.previewToken,
    required this.shootingDayId,
    required this.diff,
    required this.scenes,
    required this.participants,
    required this.willNotifyCount,
  });

  factory ChangePreview.fromJson(Map<String, dynamic> json) {
    final change = json['change'] as Map<String, dynamic>;
    final affected = json['affected'] as Map<String, dynamic>;
    return ChangePreview(
      previewToken: json['preview_token'] as String,
      shootingDayId: change['shooting_day_id'] as String,
      diff: {
        for (final entry in (change['diff'] as Map<String, dynamic>).entries)
          entry.key: FieldChange.fromJson(entry.value as Map<String, dynamic>),
      },
      scenes: _list(affected['scenes'], SceneRef.fromJson),
      participants: _list(affected['participants'], AffectedParticipant.fromJson),
      willNotifyCount: json['will_notify_count'] as int,
    );
  }

  /// What confirming will send back (US-08). Opaque: never look inside.
  final String previewToken;
  final String shootingDayId;

  /// Only the fields that change: `shoot_date`, `call_time`, `location`.
  final Map<String, FieldChange> diff;
  final List<SceneRef> scenes;
  final List<AffectedParticipant> participants;
  final int willNotifyCount;
}

/// A shooting day as seen by one person: which scenes they are in.
class AgendaDay {
  const AgendaDay({
    required this.id,
    required this.projectId,
    required this.projectName,
    required this.shootDate,
    required this.callTime,
    required this.location,
    required this.myScenes,
    required this.iAmOnSetAllDay,
  });

  factory AgendaDay.fromJson(Map<String, dynamic> json) {
    final project = json['project'] as Map<String, dynamic>;
    return AgendaDay(
      id: json['id'] as String,
      projectId: project['id'] as String,
      projectName: project['name'] as String,
      shootDate: json['shoot_date'] as String,
      callTime: json['call_time'] as String,
      location: json['location'] as String,
      myScenes: _list(json['my_scenes'], SceneRef.fromJson),
      iAmOnSetAllDay: json['i_am_on_set_all_day'] as bool,
    );
  }

  final String id;
  final String projectId;
  final String projectName;
  final String shootDate;
  final String callTime;
  final String location;
  final List<SceneRef> myScenes;
  final bool iAmOnSetAllDay;
}

/// GET /api/me/today: the days on the requested date, or on the next date that
/// has any. [date] is null when nothing is scheduled from here on.
class Today {
  const Today({required this.date, required this.isToday, required this.days});

  factory Today.fromJson(Map<String, dynamic> json) => Today(
    date: json['date'] as String?,
    isToday: json['is_today'] as bool,
    days: _list(json['days'], AgendaDay.fromJson),
  );

  final String? date;
  final bool isToday;
  final List<AgendaDay> days;
}

List<T> _list<T>(Object? raw, T Function(Map<String, dynamic>) parse) {
  return [for (final item in raw as List<dynamic>) parse(item as Map<String, dynamic>)];
}
