import 'package:productionflow/core/backend.dart';
import 'package:productionflow/core/models.dart';

/// An in-memory [Backend] for widget tests. Each test fills in the lists it
/// needs and reads back what the screens did.
///
/// It keeps the rules the screens rely on — wrong password is
/// `invalid_credentials`, an unknown id is `*_not_found`, creating something
/// makes it show up in the list — and nothing more. The real rules are tested
/// in the Go backend.
class FakeBackend implements Backend {
  /// email to (password, user). Add an entry to make a login succeed.
  final Map<String, (String, User)> accounts = {};

  final List<Project> projectList = [];
  final List<ShootingDay> dayList = [];

  /// Call sheets a test fills in by hand. A day without one here gets a sheet
  /// built from [dayList], [sceneList], [onSetAllDay] and [people].
  final Map<String, CallSheet> sheets = {};

  /// Everyone in every project, for the people and impact screens.
  final List<Participant> people = [];
  final List<Scene> sceneList = [];

  /// Day id to the ids of the people on set all day.
  final Map<String, List<String>> onSetAllDay = {};

  /// What `previewMoveDay` answers, and the requests it was asked.
  ChangePreview? previewAnswer;
  ApiException? previewError;
  final List<Map<String, String?>> previewRequests = [];

  /// What `today()` answers. Set it per test.
  Today todayAnswer = const Today(date: null, isToday: false, days: []);

  /// The date the Today screen last asked about.
  String? askedForDate;

  /// If set, register() fails with this instead of creating an account.
  ApiException? registerError;

  /// Roles passed to joinInvite, in order.
  final List<String> joinedAs = [];

  int _next = 0;
  String _id(String prefix) => '$prefix-${++_next}';

  static const User madina = User(id: 'u-madina', email: 'madina@example.kz', displayName: 'Madina');

  /// A project where the caller is the producer.
  Project addProject(String name, {String role = Roles.producer}) {
    final id = _id('project');
    final isProducer = role == Roles.producer;
    final project = Project(
      id: id,
      name: name,
      myRole: role,
      myParticipantId: _id('participant'),
      inviteCode: isProducer ? 'code-$id' : null,
      inviteUrl: isProducer ? 'https://example.test/join/code-$id' : null,
    );
    projectList.add(project);
    return project;
  }

  Participant addPerson(String projectId, String name, String role, {bool hasAccount = true}) {
    final person = Participant(
      id: _id('person'),
      projectId: projectId,
      displayName: name,
      role: role,
      hasAccount: hasAccount,
    );
    people.add(person);
    return person;
  }

  Scene addScene(String projectId, String number, {String? dayId, List<String> peopleIds = const []}) {
    final scene = Scene(
      id: _id('scene'),
      projectId: projectId,
      shootingDayId: dayId,
      number: number,
      description: 'Scene $number',
      participants: _refs(peopleIds),
    );
    sceneList.add(scene);
    return scene;
  }

  ShootingDay addDay(String projectId, String date, String callTime, String location) {
    final day = ShootingDay(
      id: _id('day'),
      projectId: projectId,
      shootDate: date,
      callTime: callTime,
      location: location,
      sceneCount: 0,
    );
    dayList.add(day);
    return day;
  }

  @override
  Future<AuthResult> register({
    required String email,
    required String password,
    required String displayName,
  }) async {
    final error = registerError;
    if (error != null) {
      throw error;
    }
    final user = User(id: _id('user'), email: email, displayName: displayName);
    accounts[email] = (password, user);
    return AuthResult(token: 'token-${user.id}', user: user);
  }

  @override
  Future<AuthResult> login({required String email, required String password}) async {
    final account = accounts[email];
    if (account == null || account.$1 != password) {
      throw const ApiException(code: 'invalid_credentials', message: 'wrong email or password');
    }
    return AuthResult(token: 'token-${account.$2.id}', user: account.$2);
  }

  @override
  Future<List<Project>> projects() async => projectList.reversed.toList();

  @override
  Future<Project> createProject(String name) async => addProject(name);

  @override
  Future<Project> project(String projectId) async {
    for (final project in projectList) {
      if (project.id == projectId) {
        return project;
      }
    }
    throw const ApiException(code: 'project_not_found', message: 'no such project');
  }

  @override
  Future<Invite> invite(String code) async {
    for (final project in projectList) {
      if ('code-${project.id}' == code) {
        return Invite(projectId: project.id, projectName: project.name);
      }
    }
    throw const ApiException(code: 'invite_not_found', message: 'this invite link does not work');
  }

  @override
  Future<Participant> joinInvite(String code, String role) async {
    final invited = await invite(code);
    joinedAs.add(role);
    return Participant(
      id: _id('participant'),
      projectId: invited.projectId,
      displayName: 'Me',
      role: role,
      hasAccount: true,
    );
  }

  @override
  Future<List<ShootingDay>> days(String projectId) async {
    final list = dayList.where((day) => day.projectId == projectId).toList();
    list.sort((a, b) => '${a.shootDate} ${a.callTime}'.compareTo('${b.shootDate} ${b.callTime}'));
    return list;
  }

  @override
  Future<ShootingDay> createDay(
    String projectId, {
    required String shootDate,
    required String callTime,
    required String location,
  }) async {
    return addDay(projectId, shootDate, callTime, location);
  }

  @override
  Future<CallSheet> callSheet(String dayId) async {
    final sheet = sheets[dayId];
    if (sheet != null) {
      return sheet;
    }
    final day = _day(dayId);
    final parent = await project(day.projectId);
    return CallSheet(
      id: day.id,
      projectId: parent.id,
      projectName: parent.name,
      myRole: parent.myRole,
      shootDate: day.shootDate,
      callTime: day.callTime,
      location: day.location,
      onSetAllDay: _refs(onSetAllDay[day.id] ?? const []),
      scenes: [
        for (final scene in sceneList)
          if (scene.shootingDayId == day.id)
            CallSheetScene(
              id: scene.id,
              number: scene.number,
              description: scene.description,
              participants: scene.participants,
            ),
      ],
    );
  }

  @override
  Future<Today> today(String date) async {
    askedForDate = date;
    return todayAnswer;
  }

  // --- people and scenes ---

  @override
  Future<List<Participant>> participants(String projectId) async {
    await project(projectId);
    return people.where((p) => p.projectId == projectId).toList();
  }

  @override
  Future<Participant> addParticipant(
    String projectId, {
    required String displayName,
    required String role,
    String? email,
  }) async {
    final person = Participant(
      id: _id('person'),
      projectId: projectId,
      displayName: displayName,
      role: role,
      hasAccount: false,
      email: email,
    );
    people.add(person);
    return person;
  }

  @override
  Future<List<ParticipantRef>> setDayParticipants(String dayId, List<String> participantIds) async {
    _day(dayId);
    onSetAllDay[dayId] = [...participantIds];
    return _refs(participantIds);
  }

  @override
  Future<List<Scene>> scenes(String projectId) async {
    return sceneList.where((s) => s.projectId == projectId).toList();
  }

  @override
  Future<Scene> createScene(
    String projectId, {
    required String number,
    required String description,
    String? shootingDayId,
  }) async {
    if (sceneList.any((s) => s.projectId == projectId && s.number == number)) {
      throw const ApiException(code: 'scene_number_taken', message: 'scene number taken');
    }
    final scene = Scene(
      id: _id('scene'),
      projectId: projectId,
      shootingDayId: shootingDayId,
      number: number,
      description: description,
      participants: const [],
    );
    sceneList.add(scene);
    return scene;
  }

  @override
  Future<Scene> moveScene(String sceneId, String? shootingDayId) async {
    return _replaceScene(sceneId, (old) => Scene(
      id: old.id,
      projectId: old.projectId,
      shootingDayId: shootingDayId,
      number: old.number,
      description: old.description,
      participants: old.participants,
    ));
  }

  @override
  Future<Scene> setSceneParticipants(String sceneId, List<String> participantIds) async {
    return _replaceScene(sceneId, (old) => Scene(
      id: old.id,
      projectId: old.projectId,
      shootingDayId: old.shootingDayId,
      number: old.number,
      description: old.description,
      participants: _refs(participantIds),
    ));
  }

  @override
  Future<ChangePreview> previewMoveDay(
    String dayId, {
    String? shootDate,
    String? callTime,
    String? location,
  }) async {
    previewRequests.add({
      'day': dayId,
      'shoot_date': shootDate,
      'call_time': callTime,
      'location': location,
    });
    final error = previewError;
    if (error != null) {
      throw error;
    }
    final answer = previewAnswer;
    if (answer == null) {
      throw StateError('set previewAnswer in the test first');
    }
    return answer;
  }

  // --- helpers ---

  ShootingDay _day(String dayId) {
    for (final day in dayList) {
      if (day.id == dayId) {
        return day;
      }
    }
    throw const ApiException(code: 'day_not_found', message: 'no such shooting day');
  }

  List<ParticipantRef> _refs(List<String> ids) => [
    for (final person in people)
      if (ids.contains(person.id))
        ParticipantRef(id: person.id, displayName: person.displayName, role: person.role),
  ];

  Scene _replaceScene(String sceneId, Scene Function(Scene old) change) {
    final index = sceneList.indexWhere((s) => s.id == sceneId);
    if (index < 0) {
      throw const ApiException(code: 'scene_not_found', message: 'no such scene');
    }
    return sceneList[index] = change(sceneList[index]);
  }
}
