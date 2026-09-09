import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// N4B: the Android-local identities PocketClaw actively writes.
///
/// These are source-contract guards. Kotlin has no test harness in this
/// repository, so the properties are asserted against the sources that produce
/// them — which is also where a regression would be introduced.
///
/// The scope is deliberately narrow. Notification channel IDs, the private
/// config directory, the PID file, the Core environment, the wire namespace and
/// the Dashboard cookie are all still Pico and belong to later phases; a
/// repository-wide guard here would fail on work that has not happened yet.
void main() {
  const kotlin = 'android/app/src/main/kotlin/com/lord1egypt/pocketclaw';

  String read(String path) {
    final file = File(path);
    expect(file.existsSync(), isTrue, reason: '$path is missing');
    return file.readAsStringSync();
  }

  group('preferences store', () {
    test('one place names the store, and it is the PocketClaw one', () {
      final prefs = read('$kotlin/PocketClawPreferences.kt');
      expect(prefs, contains('const val NAME = "pocketclaw_prefs"'));
    });

    test('no other Kotlin source names a preference store', () {
      // Three files used to declare PREF_NAME independently. That is how a
      // rename lands in two of them.
      for (final entity in Directory(kotlin).listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.kt')) continue;
        if (entity.path.endsWith('PocketClawPreferences.kt')) continue;
        final source = entity.readAsStringSync();
        expect(source, isNot(contains('getSharedPreferences(')),
            reason: '${entity.path} bypasses PocketClawPreferences.open, so the '
                'migration would not run before it reads');
        expect(source, isNot(contains('picoclaw_prefs')),
            reason: '${entity.path} still names the legacy store');
      }
    });

    test('the legacy store is read-only and never written', () {
      final prefs = read('$kotlin/PocketClawPreferences.kt');
      expect(prefs, contains('LEGACY READ-ONLY MIGRATION'));
      expect(prefs, contains('private const val LEGACY_NAME = "picoclaw_prefs"'));
      // The only operations permitted on the legacy name.
      expect(prefs, contains('context.getSharedPreferences(LEGACY_NAME'));
      expect(prefs, contains('context.deleteSharedPreferences(LEGACY_NAME)'));
      // And never a write into it.
      expect(prefs, isNot(contains('getSharedPreferences(LEGACY_NAME, Context.MODE_PRIVATE).edit()')));
    });

    test('legacy values are verified before the legacy store is deleted', () {
      final prefs = read('$kotlin/PocketClawPreferences.kt');
      final commitAt = prefs.indexOf('editor.commit()');
      final readBackAt = prefs.indexOf('val readBack =');
      final deleteAt = prefs.indexOf('deleteLegacy(context, "\${legacyValues.size}');
      expect(commitAt, greaterThan(-1));
      expect(readBackAt, greaterThan(commitAt),
          reason: 'the read-back must follow the synchronous commit');
      expect(deleteAt, greaterThan(readBackAt),
          reason: 'the legacy store is the only copy until the new one is '
              'proven durable, so it must be deleted last');
    });

    test('a both-present conflict preserves data rather than merging', () {
      final prefs = read('$kotlin/PocketClawPreferences.kt');
      expect(prefs, contains('if (canonicalExists)'));
      expect(prefs, contains('keeping both'),
          reason: 'canonical wins; a disagreeing legacy store is preserved, '
              'not overwritten or merged');
    });

    test('BootReceiver reads through the migrating accessor', () {
      expect(read('$kotlin/receiver/BootReceiver.kt'),
          contains('PocketClawPreferences.open(context)'),
          reason: 'the first boot after upgrade must not read an empty '
              'canonical store and silently reset auto-start');
    });
  });

  group('orphan Core process cleanup', () {
    final service = File('$kotlin/service/PocketClawService.kt').readAsStringSync();

    test('the N3 regression is gone', () {
      expect(service, isNot(contains('cmdline.contains("picoclaw")')),
          reason: 'after N3 the cmdline is libpocketclaw.so, so this matched '
              'nothing and orphans were never cleaned');
      expect(service, isNot(contains('killPicoClawOrphanProcesses')));
      expect(service, contains('killPocketClawOrphanProcesses'));
    });

    test('matching is exact and shares the spawn constants', () {
      expect(service, contains('basename == GATEWAY_BINARY_NAME || basename == WEB_BINARY_NAME'),
          reason: 'orphan cleanup and the names used to spawn Core must not '
              'drift apart');
      // Assert the call shape, not the prose: the comment above the helper
      // deliberately quotes both rejected rules to explain why they are wrong,
      // and a guard that forbade the explanation would delete the reasoning.
      expect(service, isNot(contains('cmdline.contains("pocketclaw")')));
      expect(service, isNot(contains('basename.contains("pocketclaw")')));
      expect(service, isNot(contains('basename.startsWith("libpocketclaw")')));
    });

    test('argv[0] basename is what is compared', () {
      expect(service, contains("substringBefore('\\u0000')"));
      expect(service, contains("substringAfterLast('/')"));
    });
  });

  group('active Android-local identities', () {
    final service = File('$kotlin/service/PocketClawService.kt').readAsStringSync();
    final channel = File('$kotlin/PocketClawMethodChannel.kt').readAsStringSync();

    test('wake lock tag', () {
      expect(service, contains('"PocketClaw::ServiceWakeLock"'));
      expect(service, isNot(contains('PicoClaw::ServiceWakeLock')));
    });

    test('log reader thread name', () {
      expect(service, contains('"pocketclaw-web-log-reader"'));
      expect(service, isNot(contains('picoclaw-web-log-reader')));
    });

    test('logcat tag', () {
      expect(channel, contains('"PocketClawChannel"'));
      expect(channel, isNot(contains('"PicoClawChannel"')));
    });

    test('exported log filename default', () {
      expect(channel, contains('"pocketclaw_logs.txt"'));
      expect(channel, isNot(contains('picoclaw_logs.txt')));
    });
  });

  group('method channel', () {
    test('both endpoints agree on getPocketClawToken', () {
      final dart = read('lib/src/core/pocketclaw_channel.dart');
      final kt = read('$kotlin/PocketClawMethodChannel.kt');
      expect(dart, contains("invokeMethod<String>('getPocketClawToken')"));
      expect(kt, contains('"getPocketClawToken" ->'));
      for (final source in [dart, kt]) {
        expect(source, isNot(contains('getPicoToken')));
      }
    });

    test('no caller still uses the old name', () {
      for (final entity in Directory('lib').listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        expect(entity.readAsStringSync(), isNot(contains('getPicoToken')),
            reason: entity.path);
      }
    });
  });

  test('deferred phases are deliberately untouched', () {
    // Proves N4B did not quietly widen: these must all still be Pico.
    //
    // The two notification channel ids used to be asserted here too. N4C moved
    // them, and this guard failing is how that scope change had to be declared
    // rather than absorbed silently; zero_pico_n4c_test.dart owns them now.
    //
    // The environment key names were asserted here too. N4E took them, and this
    // guard failing is again how that was declared rather than absorbed;
    // zero_pico_n4e_test.dart owns them now. The private directory they point
    // at is still deferred, so that half stays.
    final service = File('$kotlin/service/PocketClawService.kt').readAsStringSync();
    expect(service, contains('File(context.filesDir, "picoclaw")'));
    expect(File('android/app/src/main/res/xml/backup_rules.xml').readAsStringSync(),
        contains('path="picoclaw/"'));
  });
}
