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
  ///
  /// The canonical name is `pocketclaw-core`, not `pocketclaw`.
  /// [workspaceFallbackDirectory] already exists and belongs to the user — it
  /// is the workspace used when external storage access is unavailable, holding
  /// AGENT.md, SOUL.md, USER.md and memory/. It is deliberately **not** in the
  /// Core-private exclusion contract: whether a private workspace fallback
  /// should be backed up is a privacy decision on its own terms, not a side
  /// effect of moving Core's secrets.
  const canonicalPrivateDirectory = 'pocketclaw-core';
  const legacyPrivateDirectory = 'picoclaw';
  const workspaceFallbackDirectory = 'pocketclaw';
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
    // Protection, not migration: the exclusion for the canonical path exists so
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
      reason: 'the config path must not have moved yet',
    );
    expect(
      source,
      isNot(contains('"$canonicalPrivateDirectory"')),
      reason: 'nothing may create files/$canonicalPrivateDirectory/ before the '
          'phase that migrates into it',
    );
  });

  test('the workspace fallback is the user\'s and is left alone', () {
    // files/pocketclaw/ pre-dates this work: it holds AGENT.md, SOUL.md,
    // USER.md and memory/ when external storage access is unavailable. Moving
    // Core secrets into it, or changing its backup semantics as a side effect
    // of a namespace migration, would both be wrong.
    expect(
      read(service),
      contains('?: File(context.filesDir, "$workspaceFallbackDirectory")'),
      reason: 'the workspace fallback path must not change here',
    );

    for (final path in const [backupRules, extractionRules]) {
      expect(
        read(path),
        isNot(contains('path="$workspaceFallbackDirectory/"')),
        reason:
            '$path excludes the user workspace fallback from backup. That is a '
            'privacy decision to make deliberately, not a side effect of moving '
            'Core private state.',
      );
    }
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
