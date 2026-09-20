import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// PC-DEF-020. The native Public Mode toggle is the authority for the dashboard
/// listener, and the only way it can be is by always saying what it decided.
///
/// The service used to add `-public` when the toggle was on and add nothing
/// when it was off. To the backend those are not opposites: a supplied flag is
/// a decision, and silence means "consult launcher-config.json", whose `public`
/// field the dashboard's own Config page can set to true. So a user who enabled
/// LAN access, saved that page and then switched the toggle off got a loopback
/// listener until the service next started and a wildcard one after, with the
/// toggle still reading OFF.
///
/// These are static source assertions. They need no device, and each names the
/// regression it exists to catch.
void main() {
  const service =
      'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/service/PocketClawService.kt';

  String read(String path) => File(path).readAsStringSync();

  group('dashboard public mode', () {
    test('is always passed explicitly, including when off', () {
      final source = read(service);
      expect(
        source,
        contains('cmdList.add("-public=" + publicMode)'),
        reason: 'the native decision must reach the backend as an explicit '
            'value; omitting the flag makes the backend fall back to the '
            'persisted launcher config instead',
      );
    });

    test('never adds a bare -public that would carry no decision', () {
      final source = read(service);
      expect(
        source.contains('cmdList.add("-public")'),
        isFalse,
        reason: 'a bare -public states only the ON case. The OFF case would '
            'then be silence, which is exactly what PC-DEF-020 was.',
      );
    });

    test('does not branch on publicMode when building the argument', () {
      final source = read(service);
      // The bug lived in the else-branch that added nothing. A conditional
      // here is not automatically wrong, but it is how the defect looked, so
      // reintroducing one must be a deliberate act that fails this test first.
      // Anchored on the declaration, not on a full signature: runWebService
      // takes the owning runtime epoch since PC-DEF-072, and this guard is
      // about the argv it assembles, not about its parameter list.
      final webServiceStart = source.indexOf('private fun runWebService(');
      expect(webServiceStart, greaterThan(-1),
          reason: 'runWebService() is where the launcher argv is assembled');
      final webServiceEnd = source.indexOf('-port', webServiceStart);
      expect(webServiceEnd, greaterThan(webServiceStart));
      final argvRegion = source.substring(webServiceStart, webServiceEnd);
      expect(
        argvRegion.contains('if (publicMode)'),
        isFalse,
        reason: 'the argument is unconditional now; both states are stated',
      );
    });

    test('records why the explicit value matters', () {
      final source = read(service);
      expect(
        source,
        contains('PC-DEF-020'),
        reason: 'the next person to simplify this needs the reason in place',
      );
    });
  });
}
