import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Where the Android host puts security-sensitive runtime state.
///
/// The rule this milestone establishes: the workspace under
/// `Download/pocketclaw` stays where the user can reach it, and the credentials
/// and diagnostic logs that used to sit beside it do not. These are static
/// source assertions — they need no device, and each names the regression it
/// exists to catch.
void main() {
  const service =
      'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/service/PocketClawService.kt';
  const healthChecker =
      'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/util/HealthChecker.kt';

  String read(String path) => File(path).readAsStringSync();

  group('gateway credential', () {
    test('is written to app-private no-backup storage', () {
      final source = read(service);
      expect(source, contains('POCKETCLAW_GATEWAY_TOKEN_FILE'));
      expect(
        source,
        contains('noBackupFilesDir'),
        reason: 'the credential must not be backup-eligible',
      );
      // Named relative to noBackupFilesDir, never assembled from the workspace.
      expect(
        source,
        contains('private fun gatewayTokenFile(context: Context): File'),
      );
      expect(
        source,
        isNot(contains(r'File(getWorkspacePath(context), GATEWAY_AUTH_FILE)')),
        reason: 'the credential must never be placed in the shared workspace',
      );
    });

    test(
      'the host reads it from that private path, not from the pid record',
      () {
        final source = read(healthChecker);
        expect(source, contains('gatewayTokenFileProvider'));
        expect(
          source,
          contains('PocketClawService.gatewayTokenFilePath(context)'),
        );
        // The old reader opened the pid record in the workspace and pulled the
        // "token" field out of it. The prose above still explains that
        // history, so this asserts on the code that did it, not on the word.
        expect(
          source,
          isNot(contains('PID_FILE_NAME')),
          reason: 'the credential no longer lives in the shared pid record',
        );
        expect(source, isNot(contains('optString("token"')));
        expect(source, isNot(contains('workspacePathProvider')));
      },
    );

    test('never crosses into Dart', () {
      // The credential is confined to the Kotlin host. Nothing in the Flutter
      // layer may name it, and no method channel may return it.
      final dartSources = Directory('lib')
          .listSync(recursive: true)
          .whereType<File>()
          .where((file) => file.path.endsWith('.dart'));
      for (final file in dartSources) {
        final body = file.readAsStringSync();
        for (final forbidden in const [
          'POCKETCLAW_GATEWAY_TOKEN_FILE',
          'PICOCLAW_GATEWAY_TOKEN_FILE',
          'gatewayToken',
          'gateway_auth',
        ]) {
          expect(
            body,
            isNot(contains(forbidden)),
            reason: '${file.path} references the gateway credential',
          );
        }
      }
    });

    test('is never put in a URL or a log line', () {
      final source = read(healthChecker);
      // Presented as a header, and the error path reports a status code
      // rather than the request.
      expect(
        source,
        contains('conn.setRequestProperty("Authorization", "Bearer \$token")'),
      );
      expect(source, isNot(contains('?token=')));
      expect(source, isNot(contains('Log.d("HealthChecker"')));
      expect(
        read(service),
        isNot(contains(r'Log.i(TAG, "token')),
        reason: 'the credential must not reach logcat',
      );
    });
  });

  group('dashboard credential store', () {
    test('is placed in app-private no-backup storage', () {
      final source = read(service);
      expect(source, contains('POCKETCLAW_DASHBOARD_AUTH_DIR'));
      expect(
        source,
        contains(
          'File(context.applicationContext.noBackupFilesDir, PRIVATE_AUTH_DIR)',
        ),
        reason: 'the verifier must not be writable through shared storage',
      );
      // Never assembled from the workspace path.
      expect(
        source,
        isNot(contains(r'File(getWorkspacePath(context), PRIVATE_AUTH_DIR)')),
      );
    });

    test('no auth secret or verifier crosses into Dart', () {
      final dartSources = Directory('lib')
          .listSync(recursive: true)
          .whereType<File>()
          .where((file) => file.path.endsWith('.dart'));
      for (final file in dartSources) {
        final body = file.readAsStringSync();
        for (final forbidden in const [
          'POCKETCLAW_DASHBOARD_AUTH_DIR',
          'PICOCLAW_DASHBOARD_AUTH_DIR',
          'launcher-auth.db',
          'bcrypt_hash',
        ]) {
          expect(
            body,
            isNot(contains(forbidden)),
            reason: '${file.path} references the Dashboard credential store',
          );
        }
      }
    });
  });

  group('gateway logs', () {
    test('are written to app-private storage', () {
      final source = read(service);
      expect(source, contains('POCKETCLAW_LOG_DIR'));
      expect(
        source,
        contains('private fun privateLogDir(context: Context): File'),
      );
      expect(
        source,
        contains(
          'File(context.applicationContext.noBackupFilesDir, PRIVATE_LOG_DIR)',
        ),
      );
    });

    test('legacy shared cleanup is narrow, named and non-fatal', () {
      final source = read(service);
      expect(
        source,
        contains('private fun removeLegacySharedLogs(context: Context)'),
      );

      // Only files this app is known to have written, matched by exact name.
      for (final name in const [
        '"gateway.log"',
        '"gateway_panic.log"',
        '"launcher_panic.log"',
      ]) {
        expect(source, contains(name));
      }

      // Nothing may be matched by pattern, and nothing outside logs/ touched.
      // Scoped to the cleanup function: elsewhere this class legitimately
      // enumerates directories for the managed runtime payload.
      final start = source.indexOf('private fun removeLegacySharedLogs');
      final body = source.substring(
        start,
        source.indexOf('\n        }', start),
      );
      for (final forbidden in const [
        'deleteRecursively()',
        '.walk(',
        'endsWith(".log")',
        'listFiles()',
      ]) {
        expect(
          body,
          isNot(contains(forbidden)),
          reason: 'cleanup must not match files by pattern or recurse',
        );
      }
      // Best effort: a failure here must never stop the service starting.
      expect(source, contains('legacy shared log cleanup skipped'));
    });

    test('the user workspace itself is untouched', () {
      final source = read(service);
      // getWorkspacePath still returns the shared Downloads directory — the
      // workspace staying user-visible is the product decision this milestone
      // is careful not to reverse.
      expect(source, contains('DIRECTORY_DOWNLOADS'));
      expect(source, contains('"POCKETCLAW_HOME" to workspace.absolutePath'));
    });
  });
}
