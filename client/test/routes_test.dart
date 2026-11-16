import 'package:flutter_test/flutter_test.dart';
import 'package:productionflow/core/routes.dart';
import 'package:productionflow/features/projects/projects_page.dart';

void main() {
  group('safeFrom', () {
    test('accepts addresses inside the app', () {
      expect(safeFrom('/app/projects'), '/app/projects');
      expect(safeFrom('/join/abc123'), '/join/abc123');
    });

    // `from` arrives in a link, so it must not be able to send a freshly
    // logged-in user to someone else's site.
    test('rejects anything that could leave the app', () {
      expect(safeFrom(null), isNull);
      expect(safeFrom(''), isNull);
      expect(safeFrom('https://evil.example'), isNull);
      expect(safeFrom('//evil.example'), isNull);
      expect(safeFrom('app'), isNull);
    });
  });

  test('withFrom carries the destination through login', () {
    expect(Routes.withFrom(Routes.login, null), '/login');
    final address = Uri.parse(Routes.withFrom(Routes.login, '/join/abc?x=1'));
    expect(address.path, '/login');
    expect(address.queryParameters['from'], '/join/abc?x=1');
  });

  group('inviteCodeFrom', () {
    test('takes the code out of a pasted link', () {
      expect(inviteCodeFrom('https://productionflow.pages.dev/join/3f9a0c2e'), '3f9a0c2e');
      expect(inviteCodeFrom('  http://localhost:3000/join/3f9a0c2e/  '), '3f9a0c2e');
      expect(inviteCodeFrom('https://productionflow.pages.dev/join/3f9a0c2e?utm=x'), '3f9a0c2e');
    });

    test('takes a bare code as it is', () {
      expect(inviteCodeFrom('3f9a0c2e7b1d4e8fa6c5b2d1e0f9a8b7'), '3f9a0c2e7b1d4e8fa6c5b2d1e0f9a8b7');
    });

    test('gives up on things that are not a code', () {
      expect(inviteCodeFrom(''), isNull);
      expect(inviteCodeFrom('   '), isNull);
      expect(inviteCodeFrom('see you on set!'), isNull);
    });
  });
}
