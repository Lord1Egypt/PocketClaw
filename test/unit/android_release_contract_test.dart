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

  const baselineFile = 'android/release-baseline.properties';

  /// The last physically accepted versionCode, read from the same tracked file
  /// the build reads. Hard-coding it here would let the two drift, and would
  /// reintroduce exactly the permanent floor this contract replaces.
  int readAcceptedVersionCodeFloor() {
    final line = File(baselineFile).readAsLinesSync().firstWhere(
      (line) => line.startsWith('lastAcceptedVersionCode='),
    );
    return int.parse(line.split('=')[1].trim());
  }

  String read(String path) => File(path).readAsStringSync();

  group('signing', () {
    test('a release with no signer and no opt-in cannot be produced', () {
      final source = read(gradle);

      // The defect: `else { signingConfigs.getByName("debug") }` as the tail of
      // the release signingConfig selection. Every artifact built without
      // KEYSTORE_* in the environment carried a local development signing
      // identity rather than a release one, and nothing said so.
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

    test('what the build says about debug signing is accurate', () {
      final source = read(gradle);

      // Debug signing material is local development material. It is not one
      // universal key distributed with the SDK, and the build must not say so.
      expect(
        source.toLowerCase(),
        isNot(contains('private half')),
        reason: 'inaccurate claim about a shared debug key',
      );
      expect(
        source,
        contains('differs between development'),
        reason: 'the accurate reason must be stated: keys vary per environment',
      );
      expect(
        source,
        contains('in-place update of an existing installation'),
        reason: 'the practical consequence is what a reader needs',
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
      final floor = readAcceptedVersionCodeFloor();
      expect(
        int.parse(version.group(2)!),
        greaterThanOrEqualTo(floor),
        reason:
            'versionCode must not regress below $floor, the last physically '
            'accepted build recorded in $baselineFile',
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
      expect(source, contains('override < acceptedVersionCodeFloor'));
    });

    test('the floor advances with acceptance instead of being pinned', () {
      // A constant in the build file would keep waving through 56 long after
      // 120 had shipped. The floor is one tracked number that moves when a
      // build is physically accepted.
      final source = read(gradle);
      expect(
        source,
        isNot(contains('val acceptedVersionCodeFloor = 55')),
        reason: 'the floor must not be pinned in the build file',
      );
      expect(
        source,
        contains(
          'readAcceptedVersionCodeFloor(rootProject.file("release-baseline.properties"))',
        ),
      );

      final baseline = File(baselineFile);
      expect(baseline.existsSync(), isTrue, reason: '$baselineFile is missing');
      final contents = baseline.readAsStringSync();
      expect(contents, contains('lastAcceptedVersionCode='));
      expect(
        contents.toLowerCase(),
        contains('physically accepted'),
        reason: '$baselineFile must say when the number advances',
      );
      expect(readAcceptedVersionCodeFloor(), greaterThanOrEqualTo(55));
    });
  });

  group('analytics capability', () {
    const reporter =
        'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/AnalyticsReporter.kt';

    test('the runtime guard is tied to what was packaged', () {
      // Before this, the guard only checked the provider name and app key. It
      // happened to imply the packaging condition, so no crash was reachable —
      // but the safety was a coincidence between two independently editable
      // conditions rather than a stated invariant.
      expect(
        read(gradle),
        contains(
          'buildConfigField("boolean", "POCKETCLAW_UMENG_PACKAGED", umengAnalyticsRequested.toString())',
        ),
        reason: 'the flag must come from the value that decides the dependency',
      );
      expect(read(reporter), contains('BuildConfig.POCKETCLAW_UMENG_PACKAGED'));
      expect(read(reporter), contains('if (!umengPackaged) {'));
    });

    test('an unpackaged SDK is a disabled capability, never a crash', () {
      final source = read(reporter);
      // Every entry point already returns early on isUmengProviderEnabled(),
      // which now returns false when the SDK is absent. LinkageError is the
      // backstop for the one failure that flag exists to prevent.
      expect(source, contains('catch (e: LinkageError)'));
      expect(source, contains('Analytics SDK is not available in this build.'));
      for (final entryPoint in const [
        'fun preInit(',
        'fun submitConsent(',
        'fun uploadDeviceReport(',
      ]) {
        expect(source, contains(entryPoint));
      }
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
      // and without it: the first two are the entire difference the SDK makes.
      // The rest come from play-services-measurement via firebase_analytics and
      // are removed by the documented manifest opt-out — Firebase itself stays,
      // because device feedback is a real feature that logs a custom event and
      // needs none of the advertising surface.
      for (final permission in const [
        'android.permission.READ_PHONE_STATE',
        'freemme.permission.msa',
        'com.google.android.gms.permission.AD_ID',
        'android.permission.ACCESS_ADSERVICES_AD_ID',
        'android.permission.ACCESS_ADSERVICES_ATTRIBUTION',
        'com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE',
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

  group('Dart release hardening', () {
    const helper = 'tool/build_hardened_android.py';

    test('release compilation is fail-closed around one complete contract', () {
      final source = read(gradle);
      expect(source, contains('tasks.register("validateDartHardening")'));
      expect(source, contains('dartHardeningMode != "true"'));
      expect(source, contains('dartObfuscationProperty != "true"'));
      expect(source, contains('splitDebugInfoProperty'));
      expect(source, contains('dartTargetPlatformProperty != "android-arm64"'));
      expect(source, contains('dependsOn("validateDartHardening")'));
      expect(source, contains('"compileFlutterBuildRelease"'));
    });

    test('the canonical helper passes the exact pinned Flutter properties', () {
      final source = read(helper);
      for (final property in const [
        '-PpocketclawDartHardening=true',
        '-Pdart-obfuscation=true',
        '-Psplit-debug-info=',
        '-Ptarget-platform=android-arm64',
      ]) {
        expect(source, contains(property));
      }
      expect(source, contains('DEFAULT_SYMBOLS_DIR'));
      expect(source, contains('build/private-symbols/dart/android-arm64'));
    });

    test('the generated registrant uses a stable package URI', () {
      final gradleSource = read(gradle);
      final helperSource = read(helper);
      expect(gradleSource, contains('pocketclaw_generated'));
      expect(gradleSource, contains('rootUri'));
      expect(helperSource, contains('"rootUri": "flutter_build/"'));
      expect(helperSource, contains('"packageUri": "./"'));
      expect(
        helperSource,
        contains('package:pocketclaw_generated/dart_plugin_registrant.dart'),
      );
      expect(
        gradleSource,
        contains('competing generated-source options'),
        reason: 'the pinned plugin must not silently accept a second URI path',
      );
    });

    test('local validation cannot accidentally select production material', () {
      final source = read(helper);
      expect(source, contains('LOCAL TEST mode refuses declared production'));
      expect(source, contains('environment.pop(name, None)'));
      expect(source, contains('-PallowDebugSigning=true'));
      expect(source, contains('LOCAL TEST / NON-RELEASABLE'));
    });

    test('private symbols are ignored and verified outside the APK', () {
      final ignore = read('.gitignore');
      final source = read(helper);
      expect(ignore, contains('build/'));
      expect(ignore, contains('split-debug-info/'));
      expect(ignore, contains('symbols/'));
      expect(source, contains('private Dart symbols were packaged'));
      expect(source, contains('app.android-arm64.symbols'));
      expect(source, contains('.debug_info'));
      expect(source, contains('.debug_line'));
    });
  });
}
