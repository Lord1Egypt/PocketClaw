import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// N4E: the environment PocketClaw hands its Core child process.
///
/// The host now emits only POCKETCLAW_* names. Core still reads PICOCLAW_* —
/// in ~175 upstream struct tags and a dozen direct lookups — and a canonical
/// adapter in the Core translates between them. That split is the whole
/// contract: if the host quietly re-emits a legacy name, or the values move to
/// a different directory while the keys are being renamed, nothing fails
/// loudly, so it is guarded here. The Kotlin has no JVM test harness in this
/// repository.
void main() {
  const kotlin = 'android/app/src/main/kotlin/com/lord1egypt/pocketclaw';
  const servicePath = '$kotlin/service/PocketClawService.kt';
  final service = File(servicePath).readAsStringSync();

  /// Every string literal the host uses as an environment key.
  ///
  /// Matches the two shapes the emitter actually uses — `"KEY" to value` in the
  /// map literal and `environment["KEY"] =` for the conditional entries — so a
  /// key mentioned only in a comment is not counted as an emission.
  Set<String> emittedKeys(String source) {
    final keys = <String>{};
    for (final match
        in RegExp(r'"([A-Z][A-Z0-9_]*)"\s+to\s').allMatches(source)) {
      keys.add(match.group(1)!);
    }
    for (final match
        in RegExp(r'environment\["([A-Z][A-Z0-9_]*)"\]\s*=').allMatches(source)) {
      keys.add(match.group(1)!);
    }
    return keys;
  }

  final emitted = emittedKeys(service);

  group('the host emits no legacy environment name', () {
    test('no emitted key is in the PICOCLAW_ namespace', () {
      final legacy = emitted.where((key) => key.startsWith('PICOCLAW_')).toList();
      expect(legacy, isEmpty,
          reason: 'the Core adapter accepts legacy names as input; the host '
              'must never produce them');
    });

    test('no emitted key contains the legacy namespace at all', () {
      final tainted =
          emitted.where((key) => key.toLowerCase().contains('pico')).toList();
      expect(tainted, isEmpty);
    });

    test('the launcher spawn path writes no legacy name anywhere', () {
      for (final entity in Directory(kotlin).listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.kt')) continue;
        for (final key in emittedKeys(entity.readAsStringSync())) {
          expect(key.startsWith('PICOCLAW_'), isFalse,
              reason: '${entity.path} emits $key');
        }
      }
    });
  });

  group('the canonical set is complete', () {
    // The twelve variables migrated in N4E. Listed in full rather than checked
    // by prefix: the failure that matters is a key silently dropped during the
    // rename, and a prefix check cannot see an absence.
    const migrated = <String>[
      'POCKETCLAW_HOME',
      'POCKETCLAW_CONFIG',
      'POCKETCLAW_BINARY',
      'POCKETCLAW_GATEWAY_TOKEN_FILE',
      'POCKETCLAW_LOG_DIR',
      'POCKETCLAW_DASHBOARD_AUTH_DIR',
      'POCKETCLAW_DNS_SERVER',
      'POCKETCLAW_GATEWAY_HOT_RELOAD',
      'POCKETCLAW_TOOLS_I2C_ENABLED',
      'POCKETCLAW_TOOLS_SPI_ENABLED',
      'POCKETCLAW_TOOLS_SERIAL_ENABLED',
      'POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN',
    ];

    for (final key in migrated) {
      test('$key is emitted', () => expect(emitted, contains(key)));
    }

    test('the channel token uses the canonical name on both halves', () {
      expect(emitted, isNot(contains('PICOCLAW_CHANNELS_PICO_TOKEN')));
      expect(emitted, isNot(contains('POCKETCLAW_CHANNELS_PICO_TOKEN')));
    });
  });

  group('N4E renames keys and moves no values', () {
    // N4E renamed the keys and left the values alone, and pinned the private
    // directory here to prove it. N4F is the phase that moved it, so those two
    // assertions went with it to android_backup_exclusion_test.dart. What N4E
    // still owns is the separation the rename could have quietly collapsed:
    // POCKETCLAW_HOME is the user's workspace and is not Core's state
    // directory, whatever that directory is called this phase.
    test('POCKETCLAW_HOME still resolves to the user workspace', () {
      expect(service, contains('"POCKETCLAW_HOME" to workspace.absolutePath'));
    });

    test('the A2 private paths are unchanged', () {
      expect(service,
          contains('"POCKETCLAW_GATEWAY_TOKEN_FILE" to gatewayTokenFile(context).absolutePath'));
      expect(service,
          contains('"POCKETCLAW_LOG_DIR" to privateLogDir(context).absolutePath'));
      expect(service,
          contains('"POCKETCLAW_DASHBOARD_AUTH_DIR" to privateAuthDir(context).absolutePath'));
      expect(service, contains('private const val GATEWAY_AUTH_FILE = "gateway_auth"'));
      expect(service, contains('private const val PRIVATE_LOG_DIR = "logs"'));
      expect(service, contains('private const val PRIVATE_AUTH_DIR = "auth"'));
    });

    test('no deferred migration was pulled forward', () {
      // Each of these belongs to a later phase and each would be invisible in
      // a diff full of environment-key renames.
      // The PID record name was pinned here as deferred. N4G migrated it, and
      // pkg/pid's own tests own it now — including a guard that fails if any
      // production file outside the migration owner names the old record.
      // The channel identity and its token provider were pinned here as
      // deferred. N4H migrated both; zero_pico_n4h_test.dart owns them now,
      // together with the Core-side migration tests in pkg/config.
    });
  });

  group('the Core adapter owns the compatibility surface', () {
    final adapter =
        File('core/src/pkg/canonicalenv/canonicalenv.go').readAsStringSync();

    test('the mapping table is the single authoritative structure', () {
      for (final key in [
        'POCKETCLAW_HOME',
        'POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN',
        'PICOCLAW_CHANNELS_PICO_TOKEN',
      ]) {
        expect(adapter, contains(key));
      }
    });

    test('the adapter never writes to the process environment', () {
      // Setting a legacy name in the live environment would leak it to every
      // child the Core spawns, which is the namespace being removed.
      expect(adapter, isNot(contains('os.Setenv')));
      expect(
        File('core/src/pkg/config/env_canonical.go').readAsStringSync(),
        isNot(contains('os.Setenv(')),
      );
    });
  });
}
