import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:productionflow/core/backend.dart';
import 'package:productionflow/core/models.dart';
import 'package:productionflow/features/auth/auth_pages.dart';
import 'package:productionflow/features/invite/join_page.dart';
import 'package:productionflow/features/landing/landing_page.dart';
import 'package:productionflow/features/projects/project_page.dart';
import 'package:productionflow/features/projects/projects_page.dart';
import 'package:productionflow/features/shooting_day/call_sheet_page.dart';
import 'package:productionflow/features/today/today_page.dart';

import 'support/fake_backend.dart';
import 'support/pump.dart';

// The R1 flow, screen by screen, against a fake backend: land, register or log
// in, see today, create a project, invite, join, add a day, open a call sheet.

Finder field(String label) => find.widgetWithText(TextFormField, label);

Future<void> logIn(WidgetTester tester, String email, String password) async {
  await tester.enterText(field('Email'), email);
  await tester.enterText(field('Password'), password);
  await tester.tap(find.widgetWithText(FilledButton, 'Log in'));
  await tester.pumpAndSettle();
}

void main() {
  group('landing', () {
    testWidgets('shows the pitch and exactly one primary call to action', (tester) async {
      await pumpApp(tester, backend: FakeBackend());

      expect(find.text('ProductionFlow'), findsOneWidget);
      expect(find.textContaining('A shooting day moves.'), findsOneWidget);
      // One FilledButton on the page. If this starts failing, the page has
      // grown a competing call to action.
      expect(find.byType(FilledButton), findsOneWidget);
    });

    testWidgets('the call to action leads a visitor to log in', (tester) async {
      await pumpApp(tester, backend: FakeBackend());

      final cta = find.text('Open ProductionFlow');
      await tester.ensureVisible(cta);
      await tester.pumpAndSettle();
      await tester.tap(cta);
      await tester.pumpAndSettle();

      expect(find.byType(LandingPage), findsNothing);
      expect(find.byType(LoginPage), findsOneWidget);
    });

    testWidgets('the call to action leads someone logged in straight to Today', (tester) async {
      await pumpApp(tester, backend: FakeBackend(), session: madinaSession);

      final cta = find.text('Open ProductionFlow');
      await tester.ensureVisible(cta);
      await tester.pumpAndSettle();
      await tester.tap(cta);
      await tester.pumpAndSettle();

      expect(find.byType(TodayPage), findsOneWidget);
    });
  });

  group('logging in (US-01)', () {
    testWidgets('a logged-out visitor cannot open the app, and returns after login', (
      tester,
    ) async {
      final backend = FakeBackend()..accounts['madina@example.kz'] = ('correct horse', FakeBackend.madina);

      await pumpApp(tester, backend: backend, location: '/app/projects');
      expect(find.byType(LoginPage), findsOneWidget);
      expect(find.byType(ProjectsPage), findsNothing);

      await logIn(tester, 'madina@example.kz', 'correct horse');

      // Back where they were heading, not dumped on the default screen.
      expect(find.byType(ProjectsPage), findsOneWidget);
    });

    testWidgets('a wrong password shows the reason and stays on the form', (tester) async {
      final backend = FakeBackend()..accounts['madina@example.kz'] = ('correct horse', FakeBackend.madina);
      await pumpApp(tester, backend: backend, location: '/login');

      await logIn(tester, 'madina@example.kz', 'wrong horse');

      expect(find.text('wrong email or password'), findsOneWidget);
      expect(find.byType(LoginPage), findsOneWidget);
      // The button is usable again for a second try.
      expect(tester.widget<FilledButton>(find.byType(FilledButton)).onPressed, isNotNull);
    });

    testWidgets('an empty form is caught before anything is sent', (tester) async {
      await pumpApp(tester, backend: FakeBackend(), location: '/login');

      await tester.tap(find.widgetWithText(FilledButton, 'Log in'));
      await tester.pumpAndSettle();

      expect(find.text('Enter your email'), findsOneWidget);
      expect(find.text('Enter your password'), findsOneWidget);
    });

    testWidgets('registering creates the account and opens Today', (tester) async {
      final backend = FakeBackend();
      await pumpApp(tester, backend: backend, location: '/register');

      await tester.enterText(field('Your name'), 'Aisha');
      await tester.enterText(field('Email'), 'aisha@example.kz');
      await tester.enterText(field('Password'), 'correct horse');
      await tester.tap(find.widgetWithText(FilledButton, 'Create account'));
      await tester.pumpAndSettle();

      expect(backend.accounts.containsKey('aisha@example.kz'), isTrue);
      expect(find.byType(TodayPage), findsOneWidget);
    });

    testWidgets('a short new password is caught on the phone', (tester) async {
      final backend = FakeBackend();
      await pumpApp(tester, backend: backend, location: '/register');

      await tester.enterText(field('Your name'), 'Aisha');
      await tester.enterText(field('Email'), 'aisha@example.kz');
      await tester.enterText(field('Password'), 'short');
      await tester.tap(find.widgetWithText(FilledButton, 'Create account'));
      await tester.pumpAndSettle();

      expect(find.text('Use at least 8 characters'), findsOneWidget);
      expect(backend.accounts, isEmpty);
    });

    testWidgets('what the server rejects is shown under the field it is about', (tester) async {
      final backend = FakeBackend()
        ..registerError = const ApiException(
          code: 'validation_failed',
          message: 'some fields are invalid',
          fields: {'email': 'must be a valid email address'},
        );
      await pumpApp(tester, backend: backend, location: '/register');

      await tester.enterText(field('Your name'), 'Aisha');
      await tester.enterText(field('Email'), 'aisha-at-example');
      await tester.enterText(field('Password'), 'correct horse');
      await tester.tap(find.widgetWithText(FilledButton, 'Create account'));
      await tester.pumpAndSettle();

      expect(find.text('must be a valid email address'), findsOneWidget);
      expect(find.byType(RegisterPage), findsOneWidget);
    });

    testWidgets('logging out returns to the login screen', (tester) async {
      await pumpApp(tester, backend: FakeBackend(), session: madinaSession, location: '/app');

      await tester.tap(find.byTooltip('Account'));
      await tester.pumpAndSettle();
      expect(find.text('madina@example.kz'), findsOneWidget);
      await tester.tap(find.text('Log out'));
      await tester.pumpAndSettle();

      expect(find.byType(LoginPage), findsOneWidget);
    });
  });

  group('today (US-05)', () {
    AgendaDay agendaDay({
      required String id,
      required String date,
      required String callTime,
      List<SceneRef> scenes = const [],
      bool allDay = false,
    }) {
      return AgendaDay(
        id: id,
        projectId: 'p1',
        projectName: 'Dead Air',
        shootDate: date,
        callTime: callTime,
        location: 'Kok-Tobe, upper lot',
        myScenes: scenes,
        iAmOnSetAllDay: allDay,
      );
    }

    testWidgets("shows today's call time, location and my scenes on the first screen", (
      tester,
    ) async {
      final backend = FakeBackend()
        ..todayAnswer = Today(
          date: '2026-11-02',
          isToday: true,
          days: [
            agendaDay(
              id: 'd1',
              date: '2026-11-02',
              callTime: '07:30',
              scenes: const [
                SceneRef(id: 's1', number: '12A', description: 'Rooftop argument, dusk'),
              ],
            ),
          ],
        );
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app');

      // The phone's date is what the server is asked about.
      expect(backend.askedForDate, '2026-11-02');

      expect(find.text('SHOOTING TODAY'), findsOneWidget);
      expect(find.text('Mon 2 Nov'), findsOneWidget);
      expect(find.text('07:30'), findsOneWidget);
      expect(find.text('Kok-Tobe, upper lot'), findsOneWidget);
      expect(find.text('Dead Air'), findsOneWidget);
      expect(find.text('12A'), findsOneWidget);
      expect(find.text('Rooftop argument, dusk'), findsOneWidget);
    });

    testWidgets('with nothing today, shows the next shooting day instead of an empty screen', (
      tester,
    ) async {
      final backend = FakeBackend()
        ..todayAnswer = Today(
          date: '2026-11-10',
          isToday: false,
          days: [agendaDay(id: 'd2', date: '2026-11-10', callTime: '09:00', allDay: true)],
        );
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app');

      expect(find.text('NOTHING TODAY · NEXT SHOOTING DAY'), findsOneWidget);
      expect(find.text('Tue 10 Nov'), findsOneWidget);
      expect(find.text('09:00'), findsOneWidget);
      expect(find.text('You are on set all day'), findsOneWidget);
    });

    testWidgets('with nothing ahead at all, says so and points to projects', (tester) async {
      await pumpApp(tester, backend: FakeBackend(), session: madinaSession, location: '/app');

      expect(find.text('Nothing scheduled'), findsOneWidget);
      await tester.tap(find.text('Open projects'));
      await tester.pumpAndSettle();
      expect(find.byType(ProjectsPage), findsOneWidget);
    });

    testWidgets('tapping a day opens its call sheet', (tester) async {
      final backend = FakeBackend()
        ..todayAnswer = Today(
          date: '2026-11-02',
          isToday: true,
          days: [agendaDay(id: 'd1', date: '2026-11-02', callTime: '07:30')],
        )
        ..sheets['d1'] = const CallSheet(
          id: 'd1',
          projectId: 'p1',
          projectName: 'Dead Air',
          myRole: Roles.crew,
          shootDate: '2026-11-02',
          callTime: '07:30',
          location: 'Kok-Tobe, upper lot',
          onSetAllDay: [],
          scenes: [],
        );
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app');

      await tester.tap(find.text('07:30'));
      await tester.pumpAndSettle();

      expect(find.byType(CallSheetPage), findsOneWidget);
    });
  });

  group('navigation', () {
    testWidgets('a phone gets the bottom bar', (tester) async {
      await pumpApp(tester, backend: FakeBackend(), session: madinaSession, location: '/app');

      expect(find.byType(NavigationBar), findsOneWidget);
      expect(find.byType(NavigationRail), findsNothing);

      await tester.tap(find.widgetWithText(NavigationDestination, 'Projects'));
      await tester.pumpAndSettle();
      expect(find.byType(ProjectsPage), findsOneWidget);
    });

    testWidgets('a desktop gets the rail', (tester) async {
      useDesktopSurface(tester);
      await pumpApp(tester, backend: FakeBackend(), session: madinaSession, location: '/app');

      expect(find.byType(NavigationRail), findsOneWidget);
      expect(find.byType(NavigationBar), findsNothing);
    });
  });

  group('projects (US-02)', () {
    testWidgets('with no projects, explains the two ways in', (tester) async {
      await pumpApp(
        tester,
        backend: FakeBackend(),
        session: madinaSession,
        location: '/app/projects',
      );

      expect(find.text('No projects yet'), findsOneWidget);
      expect(find.text('New project'), findsOneWidget);
      expect(find.text('Join with invite'), findsOneWidget);
    });

    testWidgets('creating a project opens it, with the invite link ready to send', (tester) async {
      final backend = FakeBackend();
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app/projects');

      await tester.tap(find.text('New project'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'Short film A');
      await tester.tap(find.widgetWithText(FilledButton, 'Create'));
      await tester.pumpAndSettle();

      expect(backend.projectList.single.name, 'Short film A');
      expect(find.byType(ProjectPage), findsOneWidget);
      expect(find.text('Invite your crew'), findsOneWidget);
      expect(find.text(backend.projectList.single.inviteUrl!), findsOneWidget);
    });

    testWidgets('a project needs a name', (tester) async {
      final backend = FakeBackend();
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app/projects');

      await tester.tap(find.text('New project'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(FilledButton, 'Create'));
      await tester.pumpAndSettle();

      expect(find.text('Give the project a name'), findsOneWidget);
      expect(backend.projectList, isEmpty);
    });

    testWidgets('lists my projects and opens one', (tester) async {
      final backend = FakeBackend();
      backend.addProject('Dead Air');
      backend.addProject('Second film', role: Roles.crew);
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app/projects');

      expect(find.text('Dead Air'), findsOneWidget);
      expect(find.text('You are crew'), findsOneWidget);

      await tester.tap(find.text('Dead Air'));
      await tester.pumpAndSettle();
      expect(find.byType(ProjectPage), findsOneWidget);
    });
  });

  group('shooting days (US-04)', () {
    testWidgets('a producer adds a day and it appears in the schedule', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}',
      );

      expect(find.textContaining('No shooting days yet'), findsOneWidget);

      await tester.tap(find.text('Add shooting day'));
      await tester.pumpAndSettle();

      // The date picker opens on today (2 November in tests); OK accepts it.
      await tester.tap(find.text('Pick a date'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('OK'));
      await tester.pumpAndSettle();

      await tester.enterText(find.widgetWithText(TextField, 'Location'), 'Studio 3');
      await tester.tap(find.widgetWithText(FilledButton, 'Add day'));
      await tester.pumpAndSettle();

      final created = backend.dayList.single;
      expect(created.shootDate, '2026-11-02');
      expect(created.callTime, '07:00');
      expect(created.location, 'Studio 3');

      expect(find.text('Mon 2 Nov · call 07:00'), findsOneWidget);
      expect(find.text('Studio 3'), findsOneWidget);
    });

    testWidgets('a day cannot be added without a date', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}',
      );

      await tester.tap(find.text('Add shooting day'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(FilledButton, 'Add day'));
      await tester.pumpAndSettle();

      expect(find.text('Pick a date'), findsNWidgets(2)); // the field, and the error under it
      expect(backend.dayList, isEmpty);
    });

    testWidgets('crew see the schedule but cannot add days or see the invite', (tester) async {
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air', role: Roles.crew);
      backend.addDay(project.id, '2026-11-20', '07:00', 'Studio 3');
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}',
      );

      expect(find.text('Fri 20 Nov · call 07:00'), findsOneWidget);
      expect(find.text('Add shooting day'), findsNothing);
      expect(find.text('Invite your crew'), findsNothing);
    });

    testWidgets('a project I am not in shows an error, not a blank page', (tester) async {
      await pumpApp(
        tester,
        backend: FakeBackend(),
        session: madinaSession,
        location: '/app/projects/not-mine',
      );

      expect(find.text('Could not load this'), findsOneWidget);
      expect(find.text('no such project'), findsOneWidget);
    });
  });

  group('call sheet (US-05)', () {
    testWidgets('shows when, where, the scenes and who is needed', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air', role: Roles.cast);
      backend.sheets['d1'] = CallSheet(
        id: 'd1',
        projectId: project.id,
        projectName: 'Dead Air',
        myRole: Roles.cast,
        shootDate: '2026-10-04',
        callTime: '07:30',
        location: 'Kok-Tobe, upper lot',
        onSetAllDay: const [ParticipantRef(id: 'pa2', displayName: 'Daniyar', role: 'crew')],
        scenes: const [
          CallSheetScene(
            id: 's1',
            number: '12A',
            description: 'Rooftop argument, dusk',
            participants: [ParticipantRef(id: 'pa1', displayName: 'Aisha', role: 'cast')],
          ),
          CallSheetScene(
            id: 's2',
            number: '13',
            description: 'Insert: the note on the table',
            participants: [],
          ),
        ],
      );
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}/days/d1',
      );

      expect(find.text('DEAD AIR'), findsOneWidget);
      expect(find.text('Sun 4 Oct'), findsOneWidget);
      expect(find.text('07:30'), findsOneWidget);
      expect(find.text('Kok-Tobe, upper lot'), findsOneWidget);
      expect(find.text('On set all day'), findsOneWidget);
      expect(find.text('Daniyar · crew'), findsOneWidget);
      expect(find.text('12A'), findsOneWidget);
      expect(find.text('Aisha · cast'), findsOneWidget);
      expect(find.text('Nobody attached yet'), findsOneWidget);
    });

    testWidgets('a day with nothing attached says so plainly', (tester) async {
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      backend.sheets['d1'] = CallSheet(
        id: 'd1',
        projectId: project.id,
        projectName: 'Dead Air',
        myRole: Roles.producer,
        shootDate: '2026-10-04',
        callTime: '07:30',
        location: '',
        onSetAllDay: const [],
        scenes: const [],
      );
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}/days/d1',
      );

      expect(find.text('Location not set yet'), findsOneWidget);
      expect(find.text('No scenes are attached to this day yet.'), findsOneWidget);
      expect(find.text('On set all day'), findsNothing);
    });
  });

  group('joining by invite (US-03)', () {
    testWidgets('someone logged in picks a role and lands in the project', (tester) async {
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/join/code-${project.id}',
      );

      expect(find.text('Dead Air'), findsOneWidget);

      await tester.tap(find.text('Cast'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(FilledButton, 'Join project'));
      await tester.pumpAndSettle();

      expect(backend.joinedAs, ['cast']);
      expect(find.byType(ProjectPage), findsOneWidget);
    });

    testWidgets('a visitor sees the project name, logs in, and comes back to join', (
      tester,
    ) async {
      final backend = FakeBackend()..accounts['madina@example.kz'] = ('correct horse', FakeBackend.madina);
      final project = backend.addProject('Dead Air');
      await pumpApp(tester, backend: backend, location: '/join/code-${project.id}');

      // Public: the name is visible before logging in. Joining is not offered yet.
      expect(find.text('Dead Air'), findsOneWidget);
      expect(find.text('Join project'), findsNothing);

      await tester.tap(find.text('I already have an account'));
      await tester.pumpAndSettle();
      expect(find.byType(LoginPage), findsOneWidget);

      await logIn(tester, 'madina@example.kz', 'correct horse');

      expect(find.byType(JoinPage), findsOneWidget);
      expect(find.text('Join project'), findsOneWidget);
    });

    testWidgets('a broken link says so', (tester) async {
      await pumpApp(tester, backend: FakeBackend(), location: '/join/nope');

      expect(find.text('This invite link does not work'), findsOneWidget);
    });

    testWidgets('a pasted link in "Join with invite" opens the join screen', (tester) async {
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app/projects');

      await tester.tap(find.text('Join with invite'));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(TextField),
        'https://productionflow.pages.dev/join/code-${project.id}',
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Continue'));
      await tester.pumpAndSettle();

      expect(find.byType(JoinPage), findsOneWidget);
    });
  });
}
