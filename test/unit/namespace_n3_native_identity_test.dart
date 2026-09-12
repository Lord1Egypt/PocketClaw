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

    test('Core owns only the two Core executables', () {
      // Security-sensitive, and corrected after vc60 physical validation: the
      // Android app process reports comm gypt.pocketclaw, so a substring rule
      // classified PocketClaw's own UI as a live Core runtime and a recycled
      // PID would have blocked gateway startup. Ownership compares whole comm
      // values against the two Core executables.
      final unix = read('core/src/pkg/pid/pidfile_unix.go');
      expect(
        unix,
        contains('ownedProcessNames = []string{"libpocketclaw.so", "libpocketclaw-web.so"}'),
      );
      expect(unix, contains('func isOwnedComm('));
      expect(
        unix,
        isNot(contains('strings.Contains(strings.TrimSpace(string(data)), "pocketclaw")')),
        reason: 'a substring rule also accepts the Android app process and '
            'every libpocketclaw-* Managed Runtime payload',
      );
      expect(
        unix,
        isNot(contains('"picoclaw"')),
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

      // What N3 pinned is the *rename boundary*: upstream emits two artifacts
      // under their own names, and PocketClaw's identity is applied at the
      // install step rather than in the upstream recipe. So the contract is the
      // pairing, not the path.
      //
      // This used to assert the composed literal `build/picoclaw-android-arm64`.
      // H5B made the output root overridable — CORE_BUILD_DIR / CORE_OUTPUT_ROOT,
      // which is what lets reproducibility runs build into independent roots —
      // and that literal stopped existing while the guarantee it stood for did
      // not. Asserting a fixed root would break again the next time the build
      // legitimately moves its output. See PC-DEF-025.
      for (final pair in const [
        ('picoclaw-android-arm64', 'libpocketclaw.so'),
        ('picoclaw-launcher-android-arm64', 'libpocketclaw-web.so'),
      ]) {
        final (upstream, shipped) = pair;
        expect(
          script,
          matches(
            RegExp(
              r'install\s+-m\s+0755\s+"\$CORE_OUTPUT_ROOT/' +
                  RegExp.escape(upstream) +
                  r'"\s+"\$JNI_LIBS/' +
                  RegExp.escape(shipped) +
                  r'"',
            ),
          ),
          reason:
              'the install step must take upstream $upstream and ship it as '
              '$shipped; that rename is where PocketClaw identity is applied',
        );
      }

      // The source path is resolved through a variable, never a fixed root.
      // This is the assertion the old one should have been.
      expect(
        script,
        contains(r'CORE_OUTPUT_ROOT='),
        reason: 'the output root must stay relocatable for reproducibility runs',
      );
      expect(
        script,
        isNot(matches(RegExp(r'install\s+-m\s+0755\s+"?build/picoclaw-'))),
        reason: 'a hard-coded build/ root would defeat CORE_BUILD_DIR',
      );

      // The private-support step consumes the same two upstream names, so a
      // rename that missed it would archive symbols for the wrong binary.
      for (final upstream in const [
        'picoclaw-android-arm64',
        'picoclaw-launcher-android-arm64',
      ]) {
        expect(
          script,
          contains('source_binary=$upstream'),
          reason: 'the private symbol companion must come from $upstream',
        );
      }
      expect(script, contains(r'--source "$CORE_OUTPUT_ROOT/$source_binary"'));
    });

    test('runtime environment and on-disk compatibility names are unchanged', () {
      // N4E moved the emitted names to POCKETCLAW_* and gave the Core an
      // adapter that still accepts these. What N3 pinned — that renaming the
      // native binaries did not disturb the environment contract — is checked
      // against the adapter now; zero_pico_n4e_test.dart owns the emitted set.
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
        expect(adapter, contains(env));
      }
      expect(read('android/app/src/main/res/xml/backup_rules.xml'),
          contains('path="picoclaw/"'));
      // The notification channel id was pinned here too. N4C retired it;
      // zero_pico_n4c_test.dart owns that surface now.
    });

    test('the shared PID record still carries no credential', () {
      final pidfile = read('core/src/pkg/pid/pidfile.go');
      expect(pidfile, contains('record.Token = ""'),
          reason: 'A2 removed the token from the shared PID file');
    });
  });
}
