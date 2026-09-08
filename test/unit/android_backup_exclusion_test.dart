import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// The two Android backup mechanisms, and the one thing they must both refuse
/// to copy off the device.
///
/// Core keeps `config.json` and `.security.yml` under the app-private
/// `files/picoclaw/` directory. Android onboarding declines credential
/// encryption, so `.security.yml` holds provider API keys and channel bot
/// tokens as plaintext. App-private storage stops another app reading them; it
/// says nothing about backup, and until these exclusions existed both files
/// were copied verbatim into Google cloud backup and device-to-device transfer.
void main() {
  const backupRules = 'android/app/src/main/res/xml/backup_rules.xml';
  const extractionRules =
      'android/app/src/main/res/xml/data_extraction_rules.xml';
  const manifest = 'android/app/src/main/AndroidManifest.xml';
  const service =
      'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/service/PocketClawService.kt';
  const credentialStore =
      'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/security/GitHubCredentialStore.kt';

  String read(String path) => File(path).readAsStringSync();

  /// The literal both the exclusion rules and the host code must agree on.
  ///
  /// This is the whole point of the test. The rules match a directory by name,
  /// so the PicoClaw to PocketClaw namespace migration renaming `internalHome`
  /// would silently unprotect every secret without breaking anything else.
  /// Asserting both sides against one string makes that rename fail here first.
  const corePrivateDirectory = 'picoclaw';
  const credentialDirectory = 'credentials';

  test('the host writes Core private state where the rules exclude it', () {
    expect(
      read(service),
      contains('File(context.filesDir, "$corePrivateDirectory")'),
      reason:
          'PocketClawService no longer puts Core state in files/$corePrivateDirectory/. '
          'Update backup_rules.xml and data_extraction_rules.xml to match, '
          'or every provider key and bot token becomes backup-eligible again.',
    );
    expect(
      read(credentialStore),
      contains('DIRECTORY = "$credentialDirectory"'),
      reason:
          'the GitHub credential directory moved away from the excluded path',
    );
  });

  test('Android 11 and earlier: both directories are excluded from backup', () {
    final rules = read(backupRules);
    expect(rules, contains('<full-backup-content>'));
    for (final path in const [credentialDirectory, corePrivateDirectory]) {
      expect(
        rules,
        contains('<exclude domain="file" path="$path/" />'),
        reason: '$path/ must not reach the backup transport',
      );
    }
  });

  test(
    'Android 12 and later: excluded from cloud backup and device transfer',
    () {
      final rules = read(extractionRules);

      String section(String tag) {
        final start = rules.indexOf('<$tag>');
        final end = rules.indexOf('</$tag>');
        expect(start, isNonNegative, reason: '$extractionRules has no <$tag>');
        expect(end, greaterThan(start));
        return rules.substring(start, end);
      }

      // Cloud backup and device transfer are separate decisions in this format,
      // and an exclusion in one does not imply the other.
      for (final tag in const ['cloud-backup', 'device-transfer']) {
        final body = section(tag);
        for (final path in const [credentialDirectory, corePrivateDirectory]) {
          expect(
            body,
            contains('<exclude domain="file" path="$path/" />'),
            reason: '$path/ must be excluded from <$tag>',
          );
        }
      }
    },
  );

  test('backup stays enabled deliberately, with both rule files wired up', () {
    final source = read(manifest);
    // Not allowBackup="false": preferences and the seen-release marker are
    // legitimately restorable. The exclusions are what carry the policy.
    expect(source, contains('android:allowBackup="true"'));
    expect(source, contains('android:fullBackupContent="@xml/backup_rules"'));
    expect(
      source,
      contains('android:dataExtractionRules="@xml/data_extraction_rules"'),
    );
  });

  test('both rule files carry the namespace-migration recheck note', () {
    for (final path in const [backupRules, extractionRules]) {
      expect(
        read(path).toUpperCase(),
        contains('RECHECK AFTER'),
        reason: '$path must say that a namespace rename silently unprotects it',
      );
    }
  });
}
