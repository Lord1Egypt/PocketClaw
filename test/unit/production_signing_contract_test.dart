import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// H1 — the production signing contract, read out of the files that implement
/// it rather than from a description of them.
///
/// The property under test is not "signing works". It is that every way of
/// getting a release artifact *without* an authentic production signer ends in
/// a failure, and that none of them ends in a debug-signed file that looks like
/// a release.
void main() {
  final gradle = File('android/app/build.gradle.kts').readAsStringSync();
  final gate = File('tool/release_gate.py').readAsStringSync();
  final certFile = File('android/release-signing-cert.sha256');

  group('production signing inputs', () {
    test('the four fields are declared once, by name, in one place', () {
      // One map, so a field cannot be added to the check and forgotten in the
      // error message, or the reverse.
      expect(gradle, contains('val releaseSigningFields = linkedMapOf('));
      for (final field in [
        'KEYSTORE_PATH',
        'KEYSTORE_PASSWORD',
        'KEY_ALIAS',
        'KEY_PASSWORD',
      ]) {
        expect(gradle, contains('"$field" to'),
            reason: '$field is not part of the declared field set');
      }
    });

    test('signing material is read only from the environment', () {
      for (final field in [
        'KEYSTORE_PATH',
        'KEYSTORE_PASSWORD',
        'KEY_ALIAS',
        'KEY_PASSWORD',
      ]) {
        expect(gradle, contains('System.getenv("$field")'),
            reason: '$field must come from the environment');
      }
      // Never from a property or a properties file: both are trivially
      // committed by accident, and a Gradle property lands in the build log.
      expect(gradle, isNot(contains('findProperty("KEYSTORE_PASSWORD")')));
      expect(gradle, isNot(contains('findProperty("KEY_PASSWORD")')));
    });

    test('a complete configuration is required before it is usable', () {
      expect(gradle,
          contains('val releaseSigningMaterialDeclared = missingReleaseSigningFields.isEmpty()'));
      expect(gradle, contains('releaseSigningMaterialDeclared &&'));
      expect(gradle, contains('file(releaseKeystorePath).isFile'),
          reason: 'a declared path that names no file is not a usable signer');
    });
  });

  group('fail closed', () {
    test('partial configuration outranks the debug opt-in', () {
      // The dangerous shape: three of four fields set, one name mistyped. If
      // the debug branch were reachable from there, the build would hand back a
      // plausible artifact signed with a development key.
      final order = RegExp(
        r'releaseSigningMaterialUsable ->.*?'
        r'releaseSigningPartiallyDeclared -> null.*?'
        r'allowDebugSigning ->',
        dotAll: true,
      );
      expect(gradle, matches(order),
          reason: 'partial production config must be checked before the debug '
              'opt-in, and must resolve to no signing config');
    });

    test('the failure names the missing fields and never a value', () {
      expect(gradle,
          contains('missingReleaseSigningFields.joinToString(", ")'),
          reason: 'the operator needs to know which field is missing');
      // The map's values are the secrets. Only its keys may be printed.
      expect(gradle, isNot(contains('releaseSigningFields.values')));
      for (final leak in [
        r'$releaseKeystorePassword',
        r'$releaseKeyPassword',
        'println(releaseKeystorePassword',
        'println(releaseKeyPassword',
      ]) {
        expect(gradle, isNot(contains(leak)),
            reason: 'a signing secret must never reach the build log');
      }
    });

    test('validation runs before anything is packaged', () {
      expect(gradle, contains('tasks.register("validateReleaseSigning")'));
      for (final task in ['preReleaseBuild', 'packageRelease', 'bundleRelease']) {
        expect(gradle, contains('"$task"'),
            reason: '$task must depend on the signing validation');
      }
    });

    test('with nothing configured at all there is still no signer', () {
      // The final arm of the when: no material, no opt-in, no config.
      expect(gradle, contains('else -> null'));
    });
  });

  group('keystore location', () {
    test('a keystore inside the repository is refused by the build', () {
      expect(gradle, contains('keystoreIsInsideRepository'));
      expect(gradle, contains('val releaseKeystoreInsideRepository'));
      // Refusal has to reach usability, not just print a warning.
      expect(gradle, contains('!releaseKeystoreInsideRepository'));
    });

    test('containment is resolved canonically, not by string prefix', () {
      // A relative path, or a symlink that leaves the tree and comes back,
      // walks straight past a startsWith() comparison.
      expect(gradle, contains('canonicalFile'));
    });

    test('ignoring signing material is a second line, not the only one', () {
      final ignore = File('.gitignore').readAsStringSync();
      for (final pattern in ['*.jks', '*.keystore', '*.p12', '*.pfx', '*.pem']) {
        expect(ignore, contains(pattern),
            reason: '$pattern must be ignored as well as refused');
      }
    });
  });

  group('signer verification', () {
    test('the known local-test certificate is pinned by digest', () {
      expect(gate, contains('DEV_SIGNER_SHA256 = '
          '"15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c"'));
    });

    test('production rejects the development signer unconditionally', () {
      expect(gate, contains('development signer — never a production identity'));
    });

    test('production requires an enrolled certificate, not merely a non-debug one', () {
      expect(gate, contains('no production signer enrolled yet'));
      expect(gate, contains('signer does not match the enrolled production certificate'));
    });

    test('more than one signer is rejected', () {
      expect(gate, contains('distinct signers; exactly one is expected'));
    });

    test('a test artifact is labelled non-releasable rather than blurred', () {
      expect(gate, contains('LOCAL TEST / NON-RELEASABLE'));
    });
  });

  group('certificate enrollment', () {
    test('the enrollment file exists and is public metadata', () {
      expect(certFile.existsSync(), isTrue,
          reason: 'the gate reads this file to learn the production identity');
      final body = certFile.readAsStringSync();
      expect(body, contains('PUBLIC METADATA'));
      // The word appears in the file, telling the reader a password must never
      // go here. What must not appear is a password being *assigned* one.
      final assigned = RegExp(
          r'(password|passphrase|storepass|keypass)\s*[:=]\s*\S',
          caseSensitive: false);
      expect(assigned.hasMatch(body), isFalse,
          reason: 'a tracked signing file must carry no password value');
    });

    test('no production fingerprint is enrolled yet, and none is faked', () {
      // A bare 64-hex line is what the gate reads. Before the key ceremony
      // there must be none: a placeholder would either fail confusingly or,
      // worse, certify a key nobody chose.
      final digestLine = RegExp(r'^[0-9a-f]{64}$', multiLine: true);
      expect(digestLine.hasMatch(certFile.readAsStringSync()), isFalse,
          reason: 'H1 must not enroll a fingerprint; H2 does that with a real key');
    });

    test('no private key material is tracked anywhere', () {
      final tracked = Process.runSync('git', ['ls-files']).stdout as String;
      final forbidden = RegExp(
          r'\.(jks|keystore|p12|pfx)$|(^|/)key\.properties$',
          multiLine: true);
      final hits = tracked
          .split('\n')
          .where((p) => p.isNotEmpty && forbidden.hasMatch(p))
          .toList();
      expect(hits, isEmpty, reason: 'signing material is tracked: $hits');
    });
  });

  group('documentation and the ceremony helper', () {
    final doc = File('docs/RELEASE_SIGNING.md');
    final helper = File('tool/create_release_keystore.sh');

    test('both exist', () {
      expect(doc.existsSync(), isTrue);
      expect(helper.existsSync(), isTrue);
    });

    test('neither contains a literal password or key material', () {
      final assigned = RegExp(
          r'(storepass|keypass)\s+\S|(password|passphrase)\s*[:=]\s*[^\s.…]',
          caseSensitive: false);
      for (final f in [doc, helper]) {
        final body = f.readAsStringSync();
        expect(assigned.hasMatch(body), isFalse,
            reason: '${f.path} appears to contain a literal secret');
        expect(body, isNot(contains('BEGIN PRIVATE KEY')));
        expect(body, isNot(contains('BEGIN RSA PRIVATE KEY')));
      }
    });

    test('the helper refuses a destination inside the repository', () {
      final body = helper.readAsStringSync();
      expect(body, contains(r'"$REPO_ROOT"/*'),
          reason: 'containment check against the repository root');
      expect(body, contains('pwd -P'),
          reason: 'paths must be resolved, not compared as strings');
    });

    test('the helper never takes a password as an argument', () {
      final body = helper.readAsStringSync();
      for (final flag in ['-storepass', '-keypass']) {
        expect(body, isNot(contains(flag)),
            reason: '$flag would put a secret in ps output and shell history');
      }
      expect(body, contains('keytool -genkeypair'),
          reason: 'keytool prompts for the passwords itself');
    });

    test('the helper proposes modern parameters and no obsolete algorithm', () {
      final body = helper.readAsStringSync();
      expect(body, contains('KEY_ALG="RSA"'));
      expect(body, contains('KEY_SIZE="4096"'));
      expect(body, contains('SIG_ALG="SHA256withRSA"'));
      expect(body, contains('STORE_TYPE="PKCS12"'));
      expect(body, isNot(contains('SHA1')));
      expect(body, isNot(contains('-keyalg DSA')));
    });

    test('the helper does not enroll the fingerprint by itself', () {
      // Writing a signing identity into the repository should be an act
      // someone chose, not a side effect of running a script.
      final body = helper.readAsStringSync();
      expect(body, isNot(contains('> android/release-signing-cert.sha256')));
      expect(body, isNot(contains('>> android/release-signing-cert.sha256')));
    });

    test('the doc separates the three keys people conflate', () {
      final body = doc.readAsStringSync();
      for (final term in [
        'PocketClaw developer app-signing key',
        'Google Play upload key',
        'Google Play app-signing key',
      ]) {
        expect(body, contains(term),
            reason: 'the three identities must be named distinctly');
      }
      expect(body, contains('vc62'),
          reason: 'the debug-signed test install implication must be recorded');
    });

    test('the doc models all three distribution channels', () {
      final body = doc.readAsStringSync();
      for (final channel in ['direct APK', 'Google Play', 'F-Droid']) {
        expect(body.toLowerCase(), contains(channel.toLowerCase()));
      }
      // Signing continuity between direct and F-Droid is the design target, and
      // the Play lineage being separate is a consequence to state, not hide.
      expect(body, contains('AllowedAPKSigningKeys'));
    });
  });

  group('F-Droid readiness', () {
    final doc = File('docs/FDROID_RELEASE.md');

    test('the readiness audit exists', () {
      expect(doc.existsSync(), isTrue);
    });

    test('it records findings rather than vague concerns', () {
      final body = doc.readAsStringSync();
      expect(body, contains('com.lord1egypt.pocketclaw'));
      expect(body, contains('SOURCE_DATE_EPOCH'));
      // H1.5 removed Firebase; the document must say so rather than still
      // describing it as the outstanding blocker.
      expect(body, contains('REMOVED'));
      expect(body, contains('BUNDLED'),
          reason: 'the font resolution must be recorded as closed');
    });

    test('the Firebase removal is real, not just documented', () {
      final pubspec = File('pubspec.yaml').readAsStringSync();
      expect(pubspec, isNot(contains('firebase_analytics')));
      expect(pubspec, isNot(contains('firebase_core')));
      expect(File('lib/src/core/firebase_device_reporter.dart').existsSync(), isFalse);
      final manifest =
          File('android/app/src/main/AndroidManifest.xml').readAsStringSync();
      expect(manifest, isNot(contains('google_app_id')));
      final gradle = File('android/app/build.gradle.kts').readAsStringSync();
      expect(gradle, isNot(contains('FIREBASE')));
    });

    test('device feedback defaults to off rather than to a provider', () {
      // The old fall-through meant a build that omitted the dart-define picked
      // an analytics provider by accident.
      final models =
          File('lib/src/core/device_feedback_models.dart').readAsStringSync();
      expect(models, isNot(contains('firebase,')),
          reason: 'the Firebase arm must be gone from the enum');
      expect(models, contains('return DeviceFeedbackProvider.none;'));
    });

    test('it does not preemptively introduce an F-Droid flavor', () {
      // A flavor is two configurations to verify and two reproducibility
      // stories. The recommendation is to make the dependency optional instead.
      final gradle = File('android/app/build.gradle.kts').readAsStringSync();
      expect(gradle, isNot(contains('productFlavors')),
          reason: 'no flavor should exist until a blocker actually requires one');
      expect(gradle, isNot(contains('fdroid')));
    });

    test('the reproducibility objective is tracked by the gate', () {
      final gate = File('tool/release_gate.py').readAsStringSync();
      expect(gate, contains('reproducibility not yet proven'));
      // The Firebase entry was retired when the dependency was; the remaining
      // F-Droid item is the font fetching.
      expect(gate, contains('committed gh payload predates its source'));
      expect(gate, isNot(contains('google_fonts fetches fonts at runtime')),
          reason: 'the font blocker was closed; it must not still be listed');
      expect(gate, isNot(contains('Firebase/GMS packaged unconditionally')),
          reason: 'a solved blocker must not still be listed as outstanding');
    });

    test('the Umeng precedent it points at is real', () {
      // The recommended fix is "do what Umeng already does". That is only
      // advice worth following while it remains true.
      final gradle = File('android/app/build.gradle.kts').readAsStringSync();
      expect(gradle, contains('compileOnly("com.umeng.umsdk:common'));
      expect(gradle, contains('umengAnalyticsRequested'));
    });
  });

  group('what H1 must not have changed', () {
    test('the package identity is untouched', () {
      expect(gradle, contains('applicationId = "com.lord1egypt.pocketclaw"'));
    });

    test('the local-test opt-in still exists and is still explicit', () {
      expect(gradle, contains('findProperty("allowDebugSigning")'));
      expect(gradle, contains('-PallowDebugSigning=true'));
    });
  });
}
