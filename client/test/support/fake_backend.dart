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
  final Map<String, CallSheet> sheets = {};

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
    if (sheet == null) {
      throw const ApiException(code: 'day_not_found', message: 'no such shooting day');
    }
    return sheet;
  }

  @override
  Future<Today> today(String date) async {
    askedForDate = date;
    return todayAnswer;
  }
}
