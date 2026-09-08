import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// The canonical Android native identity is PocketClaw's own.
///
/// The two packaged executables are the name that reaches `nativeLibraryDir`,
/// `/proc/<pid>/comm` and the APK payload, so renaming them touched the build
/// script, the launcher, Gradle packaging, the release gate and the Core
/// process-ownership check at once. This guard holds all of those to the same
/// answer, and — as in N1 — it is two-sided: it also pins the upstream and
/// on-disk names that N3 deliberately did not touch.
void main() {
  String read(String path) {
    final file = File(path);
    expect(file.existsSync(), isTrue, reason: '$path is missing');
    return file.readAsStringSync();
  }

  const gateway = 'libpocketclaw.so';
  const web = 'libpocketclaw-web.so';

  group('canonical packaged native identity', () {
    test('the build script installs the PocketClaw names', () {
      final script = read('core/build-android-arm64.sh');
      expect(script, contains('"\$JNI_LIBS/$gateway"'));
      expect(script, contains('"\$JNI_LIBS/$web"'));
      expect(script, isNot(contains('JNI_LIBS/libpicoclaw')));
    });

    test('the launcher spawns the PocketClaw names', () {
      final service = read(
        'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/service/'
        'PocketClawService.kt',
      );
      expect(service, contains('GATEWAY_BINARY_NAME = "$gateway"'));
      expect(service, contains('WEB_BINARY_NAME = "$web"'));
      expect(service, isNot(contains('libpicoclaw')));
    });

    test('Gradle packages and protects the PocketClaw names', () {
      final gradle = read('android/app/build.gradle.kts');
      expect(gradle, contains('keepDebugSymbols += "**/$gateway"'));
      expect(gradle, contains('keepDebugSymbols += "**/$web"'));
      expect(gradle, contains('"lib/arm64-v8a/$gateway"'));
      expect(gradle, contains('"lib/arm64-v8a/$web"'));
      expect(gradle, isNot(contains('libpicoclaw')),
          reason: 'the APK must not carry or expect the pre-N3 payload names');
    });

    test('the release gate expects the PocketClaw names', () {
      final gate = read('tool/release_gate.py');
      expect(gate, contains('CORE_LIBS = ("$gateway", "$web")'));
      expect(gate, contains('$gateway"'));
      expect(gate, isNot(contains('libpicoclaw')));
    });

    test('only the PocketClaw-named binaries are staged', () {
      final staged = Directory('android/app/src/main/jniLibs/arm64-v8a')
          .listSync()
          .whereType<File>()
          .map((f) => f.uri.pathSegments.last)
          .toList();
      expect(staged, contains(gateway));
      expect(staged, contains(web));
      expect(
        staged.where((n) => n.startsWith('libpicoclaw')),
        isEmpty,
        reason: 'a stale pre-N3 binary would be packaged alongside the new one',
      );
    });

    test('Core owns only the current process name', () {
      // Security-sensitive: a stale .picoclaw.pid can name a PID the kernel
      // has since reused, and every accepted name is another way for that
      // process to be honoured as our own gateway.
      final unix = read('core/src/pkg/pid/pidfile_unix.go');
      expect(unix, contains('ownedProcessName = "pocketclaw"'));
      expect(
        unix,
        isNot(contains('Contains(strings.TrimSpace(string(data)), "picoclaw")')),
        reason: 'the pre-N3 name must not be an accepted alias',
      );
    });
  });

  group('deliberately untouched by N3', () {
    test('upstream Core build identity is unchanged', () {
      final makefile = read('core/src/Makefile');
      expect(makefile, contains('BINARY_NAME=picoclaw'));
      expect(Directory('core/src/cmd/picoclaw').existsSync(), isTrue);
      expect(read('core/src/go.mod'), contains('github.com/sipeed/picoclaw'));
    });

    test('the build script still consumes the upstream artifact names', () {
      final script = read('core/build-android-arm64.sh');
      expect(script, contains('build/picoclaw-android-arm64'));
      expect(script, contains('build/picoclaw-launcher-android-arm64'));
    });

    test('runtime environment and on-disk compatibility names are unchanged', () {
      final service = read(
        'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/service/'
        'PocketClawService.kt',
      );
      for (final env in [
        'PICOCLAW_HOME',
        'PICOCLAW_CONFIG',
        'PICOCLAW_GATEWAY_TOKEN_FILE',
        'PICOCLAW_LOG_DIR',
        'PICOCLAW_DASHBOARD_AUTH_DIR',
        'PICOCLAW_CHANNELS_PICO_TOKEN',
        'PICOCLAW_DNS_SERVER',
      ]) {
        expect(service, contains(env));
      }
      expect(read('android/app/src/main/res/xml/backup_rules.xml'),
          contains('path="picoclaw/"'));
      expect(read('lib/src/core/background_service.dart'),
          contains('picoclaw_foreground'));
    });

    test('the shared PID record still carries no credential', () {
      final pidfile = read('core/src/pkg/pid/pidfile.go');
      expect(pidfile, contains('record.Token = ""'),
          reason: 'A2 removed the token from the shared PID file');
    });
  });
}
