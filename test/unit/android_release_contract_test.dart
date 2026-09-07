import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Static guards over the three release decisions that used to have silent
/// defaults: how a release is signed, where its version comes from, and whether
/// an unused analytics SDK costs the user a permission.
///
/// These are deliberately source assertions rather than Gradle executions. Each
/// one names the exact regression that produced a wrong artifact once already,
/// and none of them needs a keystore, a network or a device to run.
void main() {
  const gradle = 'android/app/build.gradle.kts';
  const manifest = 'android/app/src/main/AndroidManifest.xml';
  const pubspec = 'pubspec.yaml';
  const localProperties = 'android/local.properties';

  /// The lowest versionCode installed on a device. Android refuses to install
  /// anything lower over it.
  const acceptedVersionCodeFloor = 55;

  String read(String path) => File(path).readAsStringSync();

  group('signing', () {
    test('a release with no signer and no opt-in cannot be produced', () {
      final source = read(gradle);

      // The defect: `else { signingConfigs.getByName("debug") }` as the tail of
      // the release signingConfig selection. Every artifact built without
      // KEYSTORE_* in the environment was signed with the Android debug key,
      // whose private half ships with every SDK install, and nothing said so.
      expect(
        source,
        isNot(contains('// Fallback to debug signing for local development')),
        reason: 'the silent debug-signing fallback is back',
      );
      expect(
        source,
        contains(
          'releaseSigningMaterialUsable -> signingConfigs.getByName("release")',
        ),
      );
      expect(
        source,
        contains('allowDebugSigning -> signingConfigs.getByName("debug")'),
      );
      expect(
        source,
        contains('else -> null'),
        reason:
            'with neither a signer nor the opt-in the release must have no '
            'signing config at all, so nothing can be quietly produced',
      );
    });

    test('the failure is wired to run before anything is built', () {
      final source = read(gradle);
      expect(source, contains('tasks.register("validateReleaseSigning")'));
      for (final task in const [
        'preReleaseBuild',
        'packageRelease',
        'bundleRelease',
      ]) {
        expect(
          source,
          contains('"$task"'),
          reason: '$task must depend on validateReleaseSigning',
        );
      }
      expect(source, contains('dependsOn("validateReleaseSigning")'));
    });

    test('debug signing is reachable only by asking for it', () {
      final source = read(gradle);
      expect(source, contains('findProperty("allowDebugSigning")'));
      expect(
        source,
        contains('-PallowDebugSigning=true'),
        reason: 'the error must name the one supported way to proceed',
      );
    });

    test('no signing secret is committed or echoed', () {
      final source = read(gradle);

      // Read from the environment, never from a tracked or gitignored file.
      for (final variable in const [
        'KEYSTORE_PATH',
        'KEYSTORE_PASSWORD',
        'KEY_ALIAS',
        'KEY_PASSWORD',
      ]) {
        expect(source, contains('System.getenv("$variable")'));
      }
      expect(
        source,
        isNot(contains('key.properties')),
        reason: 'signing material must not be read from a properties file',
      );

      // Nothing may print a credential, and the success line must not name the
      // alias or the path either.
      for (final leak in const [
        'println(releaseKeystorePassword',
        'println(releaseKeyPassword',
        'println(releaseKeyAlias',
        'println(releaseKeystorePath',
        r'$releaseKeystorePassword',
        r'$releaseKeyPassword',
        r'$releaseKeyAlias',
        r'$releaseKeystorePath',
      ]) {
        expect(
          source,
          isNot(contains(leak)),
          reason: 'signing material must never reach the build log',
        );
      }

      // And none of it may be in the repository at all.
      final tracked = Process.runSync('git', [
        'ls-files',
      ], workingDirectory: '.').stdout.toString().split('\n');
      final offenders = tracked.where(
        (path) =>
            path.endsWith('.jks') ||
            path.endsWith('.keystore') ||
            path.endsWith('key.properties'),
      );
      expect(offenders, isEmpty, reason: 'a keystore is committed: $offenders');
    });

    test('ordinary debug builds are untouched', () {
      final source = read(gradle);
      // The debug build type is not configured here at all, so it keeps AGP's
      // defaults. Asserting that keeps a future signing change from leaking
      // into the variant developers use every day.
      expect(source, isNot(contains('debug {')));
      expect(source, contains('buildTypes {\n        release {'));
    });
  });

  group('version', () {
    test('pubspec is the tracked source and is at or above the floor', () {
      final version = RegExp(
        r'^version:\s*(\d+\.\d+\.\d+)\+(\d+)\s*$',
        multiLine: true,
      ).firstMatch(read(pubspec));

      expect(
        version,
        isNotNull,
        reason: 'pubspec.yaml has no <name>+<code> version',
      );
      expect(
        version!.group(1),
        '0.2.0',
        reason: 'this milestone does not change the product version',
      );
      expect(
        int.parse(version.group(2)!),
        greaterThanOrEqualTo(acceptedVersionCodeFloor),
        reason:
            'versionCode must not regress below $acceptedVersionCodeFloor, '
            'which is already installed on a device',
      );
    });

    test('Gradle reads the version from pubspec, not from local.properties', () {
      final source = read(gradle);

      // The Flutter Gradle plugin reads flutter.versionCode from the gitignored
      // local.properties and silently defaults to 1 when it is absent, so a
      // clean checkout built 1 while the release was 55.
      expect(source, isNot(contains('versionCode = flutter.versionCode')));
      expect(source, isNot(contains('versionName = flutter.versionName')));
      expect(source, contains('versionCode = resolvedVersionCode'));
      expect(source, contains('versionName = resolvedVersionName'));
      expect(
        source,
        contains('readTrackedAppVersion(rootProject.file("../pubspec.yaml"))'),
      );
    });

    test('a version in local.properties is rejected, not silently obeyed', () {
      final source = read(gradle);
      expect(source, contains('assertLocalPropertiesCarriesNoVersion'));
      expect(source, contains('flutter.versionCode'));
      expect(source, contains('flutter.versionName'));

      // And this machine's own file must already be clean, so the build here
      // matches what a clean checkout produces.
      final file = File(localProperties);
      if (file.existsSync()) {
        final lines = file.readAsLinesSync();
        expect(
          lines.where(
            (line) =>
                line.startsWith('flutter.versionCode') ||
                line.startsWith('flutter.versionName'),
          ),
          isEmpty,
          reason: '$localProperties still declares a version',
        );
      }
    });

    test('an explicit override is validated against the same floor', () {
      final source = read(gradle);
      expect(source, contains('findProperty("versionCode")'));
      expect(source, contains('acceptedVersionCodeFloor'));
      expect(
        source,
        contains('val acceptedVersionCodeFloor = $acceptedVersionCodeFloor'),
      );
    });
  });

  group('analytics surface', () {
    test('the default build does not package the analytics SDK', () {
      final source = read(gradle);
      expect(source, contains('val umengAnalyticsRequested ='));
      expect(source, contains('if (umengAnalyticsRequested) {'));
      expect(
        source,
        contains('compileOnly("com.umeng.umsdk:common'),
        reason:
            'the SDK must stay off the runtime classpath of a build that never '
            'calls it, while AnalyticsReporter still compiles',
      );
      expect(
        RegExp(
          r'^\s*implementation\("com\.umeng',
          multiLine: true,
        ).allMatches(source).length,
        2,
        reason: 'the real dependency belongs only inside the analytics branch',
      );
    });

    test('no dangerous telephony permission is requested for it', () {
      // Measured, not assumed: READ_PHONE_STATE was contributed by this file
      // alone, and removing it removes it from the merged manifest.
      expect(
        read(manifest),
        isNot(
          contains(
            '<uses-permission android:name="android.permission.READ_PHONE_STATE"',
          ),
        ),
      );
    });

    test('the permission baseline holds in the merged manifest when built', () {
      // The source manifest is only half the answer; a dependency can add a
      // permission the app never wrote. This asserts the merged result whenever
      // a Gradle manifest task has produced one, and skips otherwise so the
      // Dart suite stays runnable without Android tooling.
      const merged =
          'build/app/intermediates/merged_manifest/release/processReleaseMainManifest/AndroidManifest.xml';
      final file = File(merged);
      if (!file.existsSync()) {
        markTestSkipped(
          'no merged manifest; run :app:processReleaseMainManifest',
        );
        return;
      }

      final source = file.readAsStringSync();
      final permissions = RegExp(
        r'android:name="([^"]+)"',
      ).allMatches(source).map((match) => match.group(1)!).toSet();

      // Attributed to the analytics SDK by building the merged manifest with
      // and without it: these two are the entire difference.
      for (final permission in const [
        'android.permission.READ_PHONE_STATE',
        'freemme.permission.msa',
      ]) {
        expect(
          permissions,
          isNot(contains(permission)),
          reason: '$permission is back in the default build',
        );
      }

      // Still expected, and still load-bearing.
      for (final permission in const [
        'android.permission.INTERNET',
        'android.permission.FOREGROUND_SERVICE_SPECIAL_USE',
        'android.permission.RECEIVE_BOOT_COMPLETED',
        'android.permission.MANAGE_EXTERNAL_STORAGE',
      ]) {
        expect(permissions, contains(permission));
      }
    });
  });
}
