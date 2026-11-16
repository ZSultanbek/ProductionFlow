import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:productionflow/core/providers.dart';
import 'package:productionflow/core/router.dart';
import 'package:productionflow/core/session.dart';
import 'package:productionflow/main.dart';

import 'fake_backend.dart';

/// "Now" in every widget test: Monday 2 November 2026, nine in the morning.
final DateTime testNow = DateTime(2026, 11, 2, 9);

const Session madinaSession = Session(token: 'token-u-madina', user: FakeBackend.madina);

/// Starts the whole app against [backend], logged in as [session] (or logged
/// out), and optionally opens [location] the way a pasted link would.
Future<void> pumpApp(
  WidgetTester tester, {
  required FakeBackend backend,
  Session? session,
  String? location,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      retry: (retryCount, error) => null,
      overrides: [
        backendProvider.overrideWithValue(backend),
        sessionStoreProvider.overrideWithValue(MemorySessionStore(session)),
        clockProvider.overrideWithValue(() => testNow),
      ],
      child: const ProductionFlowApp(),
    ),
  );
  await tester.pumpAndSettle();

  if (location != null) {
    go(tester, location);
    await tester.pumpAndSettle();
  }
}

/// Navigates like typing an address into the browser.
void go(WidgetTester tester, String location) {
  final container = ProviderScope.containerOf(tester.element(find.byType(ProductionFlowApp)));
  container.read(routerProvider).go(location);
}

/// The default test surface is 800x600, which is below AppTheme.breakpoint and
/// therefore the phone layout. Tests that want the desktop layout ask for it.
void useDesktopSurface(WidgetTester tester) {
  tester.view.physicalSize = const Size(1440, 1000);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);
}

/// A tall phone, so a whole screen fits without scrolling to find a button.
void usePhoneSurface(WidgetTester tester) {
  tester.view.physicalSize = const Size(420, 1400);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);
}
