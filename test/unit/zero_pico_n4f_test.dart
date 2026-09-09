import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// N4F: Core's private state directory.
///
/// The state machine itself is tested where it can actually run, against a real
/// temporary directory, in
/// `android/app/src/test/kotlin/.../PocketClawCoreStateTest.kt`. What is guarded
/// here is everything around it that has no runtime assertion: that one owner
/// holds the path, that the config the host launches Core with points at the
/// canonical directory, and that three separate storage boundaries did not
/// quietly collapse into one during a migration whose whole diff is paths.
void main() {
  const kotlin = 'android/app/src/main/kotlin/com/lord1egypt/pocketclaw';
  const servicePath = '$kotlin/service/PocketClawService.kt';
  const ownerPath = '$kotlin/PocketClawCoreState.kt';
  const channelPath = '$kotlin/PocketClawMethodChannel.kt';

  String read(String path) => File(path).readAsStringSync();

  final service = read(servicePath);
  final owner = read(ownerPath);
  final channel = read(channelPath);

  group('one owner holds the path', () {
    test('the canonical name is declared once', () {
      expect(owner, contains('CANONICAL_DIR_NAME = "pocketclaw-core"'));

      for (final entity in Directory(kotlin).listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.kt')) continue;
        if (entity.path.endsWith('PocketClawCoreState.kt')) continue;
        expect(
          entity.readAsStringSync(),
          isNot(contains('"pocketclaw-core"')),
          reason: '${entity.path} spells the canonical directory itself; it '
              'must ask PocketClawCoreState, or a caller can reach a directory '
              'whose migration has not run',
        );
      }
    });

    test('the legacy name is move-only, and says so', () {
      expect(owner, contains('LEGACY MOVE-ONLY MIGRATION'));
      // No dual-write, no copy back, no alias. Each of these would leave the
      // user with two directories that disagree and no way to say which is live.
      expect(owner, isNot(contains('createSymbolicLink')));
      expect(owner, isNot(contains('copyTo')));
      expect(owner, isNot(contains('copyRecursively')));
    });

    test('every consumer goes through the owner', () {
      for (final call in const [
        'PocketClawCoreState.directory(context)',
        'PocketClawCoreState.configFile(context)',
      ]) {
        expect(service, contains(call));
      }
      // The Dart config editor reaches the same state through the method
      // channel, and can run before the service has ever started.
      expect(channel, contains('PocketClawCoreState.configFile(context)'));
    });
  });

  group('migration runs before anything reads or writes the state', () {
    test('resolution is serialized and happens once', () {
      expect(owner, contains('@Synchronized'));
      expect(owner, contains('fun directory(context: Context): File'));
    });

    test('the environment is built from the resolved directory', () {
      // buildEnvironment feeds every spawn path — the version probe, onboarding
      // and the web service — so resolving here puts the migration ahead of all
      // of them without each one having to remember.
      expect(service, contains('val coreState = PocketClawCoreState.directory(context)'));
      expect(service,
          contains('"POCKETCLAW_CONFIG" to configPath'));
      expect(service,
          contains('val configPath = PocketClawCoreState.configFile(context).absolutePath'));
    });

    test('a failed migration throws rather than returning a path', () {
      // Fail closed. A fresh empty directory handed back after a failure would
      // let Core onboard into it and present a factory reset as a successful
      // start, with the real state still on disk and nothing pointing at it.
      expect(owner, contains('class MigrationException'));
      expect(owner, contains('throw MigrationException'));
      expect(owner, isNot(contains('return canonical // fallback')));
    });

    test('the config editor surfaces a failure instead of an empty config', () {
      expect(channel, contains('CORE_STATE_UNAVAILABLE'));
      expect(channel, contains('READ_CONFIG_FAILED'));
      expect(channel, contains('SAVE_CONFIG_FAILED'));
    });
  });

  group('the storage boundaries stay separate', () {
    test('POCKETCLAW_HOME is the workspace, not Core state', () {
      // The regression this phase could most easily cause: Core-private state
      // and the user's workspace are different concepts, and collapsing them
      // would put provider keys in the user's document directory.
      expect(service, contains('"POCKETCLAW_HOME" to workspace.absolutePath'));
      expect(service, isNot(contains('"POCKETCLAW_HOME" to coreState')));
      expect(
        service,
        contains('val workspace = File(getWorkspacePath(context))'),
      );
    });

    test('the workspace fallback is still filesDir/pocketclaw', () {
      expect(service, contains('File(context.filesDir, "pocketclaw")'));
    });

    test('the A2 private paths did not move into Core state', () {
      expect(service,
          contains('"POCKETCLAW_GATEWAY_TOKEN_FILE" to gatewayTokenFile(context).absolutePath'));
      expect(service,
          contains('"POCKETCLAW_LOG_DIR" to privateLogDir(context).absolutePath'));
      expect(service,
          contains('"POCKETCLAW_DASHBOARD_AUTH_DIR" to privateAuthDir(context).absolutePath'));
      // Each resolves from noBackupFilesDir, not from the Core state directory.
      for (final resolver in const [
        'gatewayTokenFile',
        'privateLogDir',
        'privateAuthDir',
      ]) {
        final declaration = RegExp('fun $resolver\\(context: Context\\): File[^\\n]*\\n?[^\\n]*');
        final match = declaration.firstMatch(service);
        expect(match, isNotNull, reason: '$resolver is no longer declared');
        expect(match!.group(0), contains('noBackupFilesDir'),
            reason: '$resolver must not resolve from Core state');
      }
    });
  });

  group('N4E is not undone and later phases are not pulled forward', () {
    test('no legacy environment name returned', () {
      expect(RegExp(r'"PICOCLAW_[A-Z0-9_]*"\s+to\s').hasMatch(service), isFalse);
      expect(service, contains('"POCKETCLAW_CONFIG" to configPath'));
    });

    test('deferred surfaces are untouched', () {
      expect(service, contains('.picoclaw.pid'),
          reason: 'the PID record name migrates with the Core');
      expect(service, contains('picoTokenForHost'),
          reason: 'the serialized "pico" channel is renamed with the channel');
      expect(Directory('core/src/pkg/channels/pico').existsSync(), isTrue);
    });
  });
}
