import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// The N1 boundary, asserted rather than described.
///
/// A repository-wide "no picoclaw" guard would be wrong by architecture: most
/// remaining occurrences are upstream identity, on-disk compatibility or legal
/// provenance, and forbidding them wholesale would either fail immediately or
/// pressure someone into renaming something that must not change. So this
/// guard is two-sided — it pins what N1 renamed *and* what N1 must not have
/// touched.
void main() {
  String read(String path) {
    final file = File(path);
    expect(file.existsSync(), isTrue, reason: '$path is missing');
    return file.readAsStringSync();
  }

  group('renamed by N1', () {
    test('no PocketClaw-owned Dart source still declares PicoClawChannel', () {
      final offenders = <String>[];
      for (final entity in Directory('lib').listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        if (entity.readAsStringSync().contains('PicoClawChannel')) {
          offenders.add(entity.path);
        }
      }
      expect(offenders, isEmpty);
    });

    test('the old Dart channel file is gone', () {
      expect(File('lib/src/core/picoclaw_channel.dart').existsSync(), isFalse);
      expect(File('lib/src/core/pocketclaw_channel.dart').existsSync(), isTrue);
    });

    test('PocketClaw-owned build-time defines are POCKETCLAW_*', () {
      final gradle = read('android/app/build.gradle.kts');
      expect(gradle, contains('POCKETCLAW_ONBOARDING_BASE_URL'));
      // The Firebase defines were renamed by N1 and then removed entirely by
      // H1.5 along with the SDK they configured. Their absence is the point:
      // a define that no longer exists cannot be misnamed.
      expect(gradle, isNot(contains('FIREBASE')),
          reason: 'the Firebase build configuration was removed, not renamed');
      for (final stale in [
        'PICOCLAW_ANALYTICS_PROVIDER',
        'PICOCLAW_UMENG_',
        'PICOCLAW_FIREBASE_',
      ]) {
        expect(gradle, isNot(contains(stale)),
            reason: '$stale is PocketClaw-owned and was renamed in N1');
      }
    });

    test('no component-name compatibility shim was introduced', () {
      // N0 established that Android resolves these from the manifest and from
      // compile-time class literals, so a shim would be dead code pretending
      // to protect something.
      for (final entity in Directory('android/app/src/main/kotlin')
          .listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.kt')) continue;
        expect(entity.path, isNot(contains('PicoClaw')),
            reason: 'a legacy-named Kotlin file remains');
      }
    });
  });

  group('deliberately preserved by N1', () {
    test('upstream Core runtime environment variables are unchanged', () {
      // N1 pinned these against the Kotlin host, because at the time the host
      // emitting them was the only thing that made them reachable. N4E added a
      // canonical adapter in the Core, so the host emits POCKETCLAW_* and these
      // names became what the Core accepts rather than what anything produces.
      // The invariant N1 cared about is unchanged and still checked — upstream's
      // interface is not renamed — only its location moved, and this guard
      // failing is how that had to be declared; zero_pico_n4e_test.dart owns the
      // emitted set now.
      final adapter = read('core/src/pkg/canonicalenv/canonicalenv.go');
      for (final env in [
        'PICOCLAW_HOME',
        'PICOCLAW_CONFIG',
        'PICOCLAW_GATEWAY_TOKEN_FILE',
        'PICOCLAW_LOG_DIR',
        'PICOCLAW_DASHBOARD_AUTH_DIR',
        'PICOCLAW_CHANNELS_PICO_TOKEN',
        'PICOCLAW_DNS_SERVER',
      ]) {
        expect(adapter, contains(env),
            reason: '$env is Core\'s own interface, not PocketClaw source '
                'identity; dropping it fails the Gateway closed');
      }
    });

    test('the native Core library names are the N3 canonical ones', () {
      final service = read(
        'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/service/'
        'PocketClawService.kt',
      );
      expect(service, contains('libpocketclaw.so'));
      expect(service, contains('libpocketclaw-web.so'));
    });

    test('the backup exclusion still matches Core private state', () {
      // A rename here fails silently by sending secrets to cloud backup.
      expect(read('android/app/src/main/res/xml/backup_rules.xml'),
          contains('path="picoclaw/"'));
      expect(read('android/app/src/main/res/xml/data_extraction_rules.xml'),
          contains('path="picoclaw/"'));
    });

    // N1 preserved the desktop adapter's upstream artifact names. The adapter
    // itself is gone now: PocketClaw is Android-only, so nothing in the app
    // looks up a picoclaw-launcher binary on a host PATH any more.
    test('no desktop adapter remains to look up upstream artifact names', () {
      expect(
        File('lib/src/native/desktop_core_service_adapter.dart').existsSync(),
        isFalse,
      );
    });
  });
}
