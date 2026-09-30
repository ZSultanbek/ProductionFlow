import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:productionflow/features/app_shell/app_shell.dart';
import 'package:productionflow/features/landing/landing_page.dart';
import 'package:productionflow/main.dart';

void main() {
  /// The default test surface is 800x600, which is below AppTheme.breakpoint and
  /// therefore the phone layout. Tests that want the desktop layout ask for it.
  void useDesktopSurface(WidgetTester tester) {
    tester.view.physicalSize = const Size(1440, 1000);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
  }

  testWidgets('landing page shows the pitch and exactly one primary CTA', (tester) async {
    await tester.pumpWidget(const ProductionFlowApp());
    await tester.pumpAndSettle();

    expect(find.text('ProductionFlow'), findsOneWidget);
    expect(find.textContaining('A shooting day moves.'), findsOneWidget);

    // One FilledButton on the page. Everything else is a quieter affordance —
    // if this starts failing, the page has grown a competing call to action.
    expect(find.byType(FilledButton), findsOneWidget);
    expect(find.text('Open ProductionFlow'), findsOneWidget);
  });

  testWidgets('phone layout: CTA is reachable by scrolling and opens the shell', (tester) async {
    await tester.pumpWidget(const ProductionFlowApp());
    await tester.pumpAndSettle();

    final cta = find.text('Open ProductionFlow');
    await tester.ensureVisible(cta);
    await tester.pumpAndSettle();

    await tester.tap(cta);
    await tester.pumpAndSettle();

    expect(find.byType(LandingPage), findsNothing);
    expect(find.byType(AppShell), findsOneWidget);
    // Phone gets the bottom bar, not the rail.
    expect(find.byType(NavigationBar), findsOneWidget);
    expect(find.byType(NavigationRail), findsNothing);
  });

  testWidgets('desktop layout: shell uses the rail and switches destination', (tester) async {
    useDesktopSurface(tester);

    await tester.pumpWidget(const ProductionFlowApp());
    await tester.pumpAndSettle();

    await tester.tap(find.text('Open ProductionFlow'));
    await tester.pumpAndSettle();

    expect(find.byType(NavigationRail), findsOneWidget);
    expect(find.byType(NavigationBar), findsNothing);
    expect(find.text('No shooting days yet'), findsOneWidget);

    await tester.tap(find.text('Changes'));
    await tester.pumpAndSettle();

    expect(find.text('Nothing has changed yet'), findsOneWidget);
  });
}
