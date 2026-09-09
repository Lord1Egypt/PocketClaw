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

  /// The literals the exclusion rules and the host code must agree on.
  ///
  /// This is the whole point of the test. The rules match a directory by name,
  /// so renaming `internalHome` would silently unprotect every secret without
  /// breaking anything else. Asserting both sides against one string makes that
  /// rename fail here first.
  ///
  /// Two Core paths are protected, not one. [canonicalPrivateDirectory] is where
  /// Core state is going; its exclusion exists *before* any code creates it,
  /// because a migration that ran first would leave secrets in an unprotected
  /// path. [legacyPrivateDirectory] is where Core state still is, and its
  /// exclusion is retained afterwards as a LEGACY SECURITY EXCLUSION: an
  /// interrupted migration or a downgrade can leave files behind, and changing
  /// where PocketClaw writes must not make what is already there
  /// backup-eligible.
  const canonicalPrivateDirectory = 'pocketclaw';
  const legacyPrivateDirectory = 'picoclaw';
  const credentialDirectory = 'credentials';

  test('the host writes Core private state where the rules exclude it', () {
    expect(
      read(service),
      contains('File(context.filesDir, "$legacyPrivateDirectory")'),
      reason:
          'PocketClawService no longer puts Core state in files/$legacyPrivateDirectory/. '
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
    for (final path in const [
      credentialDirectory,
      canonicalPrivateDirectory,
      legacyPrivateDirectory,
    ]) {
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
        for (final path in const [
          credentialDirectory,
          canonicalPrivateDirectory,
          legacyPrivateDirectory,
        ]) {
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

  test('Core private state has not moved to the canonical path yet', () {
    // N4D is protection, not migration: the exclusion for pocketclaw/ exists so
    // the later move cannot create an unprotected directory even for an
    // instant. Core state must still be written to the legacy path here.
    final source = read(service);
    expect(
      source,
      contains('val internalHome = File(context.filesDir, "$legacyPrivateDirectory")'),
      reason: 'the private-directory migration belongs to a later phase',
    );
    expect(
      source,
      contains('File(filesDir, "$legacyPrivateDirectory/config.json")'),
      reason: 'the config path must not have moved in N4D',
    );

    // files/pocketclaw/ is already referenced — as the *workspace* fallback
    // used when MANAGE_EXTERNAL_STORAGE is denied, not as Core private state.
    // That collision is recorded for the private-directory phase; here it only
    // means the new exclusion already covers a real path.
    expect(source, contains('?: File(context.filesDir, "$canonicalPrivateDirectory")'));
  });

  test('the legacy exclusion is classified, not left as an oversight', () {
    for (final path in const [backupRules, extractionRules]) {
      expect(
        read(path),
        contains('LEGACY SECURITY EXCLUSION'),
        reason:
            '$path must say why the $legacyPrivateDirectory/ line is kept, so a '
            'future Zero-Pico sweep does not delete it as leftover namespace',
      );
    }
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
