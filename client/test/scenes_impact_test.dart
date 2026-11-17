import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:productionflow/core/backend.dart';
import 'package:productionflow/core/models.dart';

import 'support/fake_backend.dart';
import 'support/pump.dart';

// Release 2, screen by screen: people added by name, scenes on days, who is
// needed where (US-06), and the impact preview before a day moves (US-07).

Finder textField(String label) => find.widgetWithText(TextField, label);
Finder filled(String label) => find.widgetWithText(FilledButton, label);

Future<void> tapAndSettle(WidgetTester tester, Finder finder) async {
  await tester.tap(finder);
  await tester.pumpAndSettle();
}

void main() {
  group('people (US-06)', () {
    testWidgets('a producer adds someone by name, and they show as having no app', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      backend.addPerson(project.id, 'Madina', Roles.producer);
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app/projects/${project.id}');

      expect(find.text('Madina · producer'), findsOneWidget);

      await tapAndSettle(tester, find.text('Add a person'));
      await tester.enterText(textField('Name'), 'Ruslan');
      await tapAndSettle(tester, find.text('Crew'));
      await tapAndSettle(tester, filled('Add person'));

      final ruslan = backend.people.last;
      expect(ruslan.displayName, 'Ruslan');
      expect(ruslan.role, Roles.crew);
      expect(ruslan.email, isNull);
      expect(find.text('Ruslan · crew · no app'), findsOneWidget);
    });

    testWidgets('crew see the people but cannot add anyone', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air', role: Roles.crew);
      backend.addPerson(project.id, 'Madina', Roles.producer);
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app/projects/${project.id}');

      expect(find.text('Madina · producer'), findsOneWidget);
      expect(find.text('Add a person'), findsNothing);
      expect(find.text('Add scene'), findsNothing);
    });
  });

  group('scenes (US-06)', () {
    testWidgets('an unscheduled scene is put on a day from the project page', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      final day = backend.addDay(project.id, '2026-11-02', '07:30', 'Kok-Tobe');
      backend.addScene(project.id, '14');
      await pumpApp(tester, backend: backend, session: madinaSession, location: '/app/projects/${project.id}');

      expect(find.text('Scene 14'), findsOneWidget);

      await tapAndSettle(tester, find.text('Put on a day'));
      await tapAndSettle(tester, find.text('Mon 2 Nov · call 07:30 · Kok-Tobe'));

      expect(backend.sceneList.single.shootingDayId, day.id);
      expect(find.text('Every scene is on a shooting day.'), findsOneWidget);
    });

    testWidgets('a producer builds a day on its call sheet', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      final day = backend.addDay(project.id, '2026-11-02', '07:30', 'Kok-Tobe');
      backend.addPerson(project.id, 'Aisha', Roles.cast);
      backend.addPerson(project.id, 'Daniyar', Roles.crew);
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}/days/${day.id}',
      );

      expect(find.text('No scenes are attached to this day yet.'), findsOneWidget);

      // A new scene, straight onto this day.
      await tapAndSettle(tester, find.text('Add scene'));
      await tester.enterText(textField('Scene number'), '12A');
      await tester.enterText(textField('What happens'), 'Rooftop argument, dusk');
      await tapAndSettle(tester, filled('Add scene'));

      expect(backend.sceneList.single.shootingDayId, day.id);
      expect(find.text('Rooftop argument, dusk'), findsOneWidget);
      expect(find.text('Nobody attached yet'), findsOneWidget);

      // Who is needed in it.
      await tapAndSettle(tester, find.byTooltip('Scene 12A'));
      await tapAndSettle(tester, find.text('Who is needed'));
      await tapAndSettle(tester, find.text('Aisha'));
      await tapAndSettle(tester, filled('Save'));

      expect(find.text('Aisha · cast'), findsOneWidget);

      // And who is on set whatever is shot.
      await tapAndSettle(tester, find.text('Choose people'));
      await tapAndSettle(tester, find.text('Daniyar'));
      await tapAndSettle(tester, filled('Save'));

      expect(find.text('Daniyar · crew'), findsOneWidget);
      expect(backend.onSetAllDay[day.id], hasLength(1));
    });

    testWidgets('a taken scene number is shown on the field', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air');
      final day = backend.addDay(project.id, '2026-11-02', '07:30', 'Kok-Tobe');
      backend.addScene(project.id, '12A');
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}/days/${day.id}',
      );

      await tapAndSettle(tester, find.text('Add scene'));
      await tester.enterText(textField('Scene number'), '12A');
      await tapAndSettle(tester, filled('Add scene'));

      expect(find.text('There is already a scene 12A'), findsOneWidget);
      expect(backend.sceneList, hasLength(1));
    });

    testWidgets('crew read the call sheet without any of the producer controls', (tester) async {
      usePhoneSurface(tester);
      final backend = FakeBackend();
      final project = backend.addProject('Dead Air', role: Roles.crew);
      final day = backend.addDay(project.id, '2026-11-02', '07:30', 'Kok-Tobe');
      final aisha = backend.addPerson(project.id, 'Aisha', Roles.cast);
      backend.addScene(project.id, '12A', dayId: day.id, peopleIds: [aisha.id]);
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}/days/${day.id}',
      );

      expect(find.text('Aisha · cast'), findsOneWidget);
      expect(find.text('Add scene'), findsNothing);
      expect(find.text('Move this day'), findsNothing);
      expect(find.text('Choose people'), findsNothing);
      expect(find.byTooltip('Scene 12A'), findsNothing);
    });
  });

  group('the impact preview (US-07)', () {
    late FakeBackend backend;
    late Project project;
    late ShootingDay day;

    Future<void> openCallSheet(WidgetTester tester) async {
      usePhoneSurface(tester);
      await pumpApp(
        tester,
        backend: backend,
        session: madinaSession,
        location: '/app/projects/${project.id}/days/${day.id}',
      );
    }

    setUp(() {
      backend = FakeBackend();
      project = backend.addProject('Dead Air');
      day = backend.addDay(project.id, '2026-11-02', '07:30', 'Kok-Tobe');
    });

    testWidgets('moving a day shows who and what it affects, and changes nothing', (tester) async {
      backend.previewAnswer = const ChangePreview(
        previewToken: 'token',
        shootingDayId: 'd',
        diff: {'location': FieldChange(from: 'Kok-Tobe', to: 'Studio B')},
        scenes: [SceneRef(id: 's1', number: '12A', description: 'Rooftop argument, dusk')],
        participants: [
          AffectedParticipant(
            id: 'p1',
            displayName: 'Daniyar',
            role: Roles.crew,
            hasAccount: true,
            onSetAllDay: true,
            sceneNumbers: ['12A'],
          ),
          AffectedParticipant(
            id: 'p2',
            displayName: 'Ruslan',
            role: Roles.crew,
            hasAccount: false,
            onSetAllDay: false,
            sceneNumbers: ['12A'],
          ),
        ],
        willNotifyCount: 1,
      );
      await openCallSheet(tester);

      await tapAndSettle(tester, find.text('Move this day'));
      await tester.enterText(textField('Location'), 'Studio B');
      await tapAndSettle(tester, filled('See who this affects'));

      // Only the field that changed is sent.
      expect(backend.previewRequests.single, {
        'day': day.id,
        'shoot_date': null,
        'call_time': null,
        'location': 'Studio B',
      });

      expect(find.text('Who this affects'), findsOneWidget);
      expect(find.text('Kok-Tobe  →  Studio B'), findsOneWidget);
      expect(find.text('Affects 1 scene and 2 people. 1 person would be notified.'), findsOneWidget);
      expect(find.text('Daniyar · crew'), findsOneWidget);
      expect(find.text('on set all day and in scene 12A'), findsOneWidget);
      expect(find.text('Ruslan · crew'), findsOneWidget);
      expect(find.text('no app'), findsOneWidget);

      // Confirming is US-08: the button is there, switched off.
      final confirm = tester.widget<FilledButton>(filled('Confirm change'));
      expect(confirm.onPressed, isNull);
      expect(backend.dayList.single.location, 'Kok-Tobe');
    });

    testWidgets('an empty day can be previewed and says nobody needs telling', (tester) async {
      backend.previewAnswer = const ChangePreview(
        previewToken: 'token',
        shootingDayId: 'd',
        diff: {'location': FieldChange(from: 'Kok-Tobe', to: 'Studio B')},
        scenes: [],
        participants: [],
        willNotifyCount: 0,
      );
      await openCallSheet(tester);

      await tapAndSettle(tester, find.text('Move this day'));
      await tester.enterText(textField('Location'), 'Studio B');
      await tapAndSettle(tester, filled('See who this affects'));

      expect(backend.previewRequests, hasLength(1));
      expect(find.text('Affects 0 scenes and 0 people.'), findsOneWidget);
      expect(find.text('Nobody is attached to this day, so nobody needs to be told.'), findsOneWidget);
    });

    testWidgets('nothing changed on the form is caught before asking the server', (tester) async {
      await openCallSheet(tester);

      await tapAndSettle(tester, find.text('Move this day'));
      await tapAndSettle(tester, filled('See who this affects'));

      expect(find.text('Change the date, the call time or the location first.'), findsOneWidget);
      expect(backend.previewRequests, isEmpty);
    });

    testWidgets("the server's reason is shown in the dialog", (tester) async {
      backend.previewError = const ApiException(code: 'forbidden', message: 'only a producer of this project can do that');
      await openCallSheet(tester);

      await tapAndSettle(tester, find.text('Move this day'));
      await tester.enterText(textField('Location'), 'Studio B');
      await tapAndSettle(tester, filled('See who this affects'));

      expect(find.text('only a producer of this project can do that'), findsOneWidget);
      expect(find.text('Who this affects'), findsNothing);
    });
  });
}
