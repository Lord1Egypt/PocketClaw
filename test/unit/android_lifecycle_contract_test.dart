import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// PC-DEF-085. "PocketClaw keeps stopping" was reported after a phone reboot,
/// before the app was opened. Nothing may start PocketClaw without the owner:
/// no boot or package-replaced receiver, no direct-boot component, no scheduled
/// work, and a service that Android never brings back by itself. These are
/// static assertions over the manifest and the Kotlin sources, so an automatic
/// start cannot creep back unnoticed.
void main() {
  const kotlin = 'android/app/src/main/kotlin/com/lord1egypt/pocketclaw';
  const service = '$kotlin/service/PocketClawService.kt';
  const mergedManifest =
      'build/app/intermediates/merged_manifest/release/processReleaseMainManifest/AndroidManifest.xml';
  String read(String path) => File(path).readAsStringSync();
  String declared(String xml) =>
      xml.replaceAll(RegExp(r'<!--.*?-->', dotAll: true), '');

  const bootActions = [
    'RECEIVE_BOOT_COMPLETED',
    'BOOT_COMPLETED',
    'LOCKED_BOOT_COMPLETED',
    'QUICKBOOT_POWERON',
    'MY_PACKAGE_REPLACED',
    'PACKAGE_REPLACED',
  ];

  List<String> kotlinSources() => Directory(kotlin)
      .listSync(recursive: true)
      .whereType<File>()
      .where((file) => file.path.endsWith('.kt'))
      .map((file) => file.path)
      .toList();

  group('manifest', () {
    test('the app declares no receiver and nothing that runs at boot', () {
      final manifest = declared(
        read('android/app/src/main/AndroidManifest.xml'),
      );
      for (final action in bootActions) {
        expect(manifest, isNot(contains(action)), reason: action);
      }
      expect(manifest, isNot(contains('<receiver')));
      expect(manifest, isNot(contains('directBootAware="true"')));
    });

    test('the service is private to the app', () {
      final manifest = declared(
        read('android/app/src/main/AndroidManifest.xml'),
      );
      final start = manifest.indexOf('.service.PocketClawService');
      final tag = manifest.substring(
        manifest.lastIndexOf('<service', start),
        manifest.indexOf('/>', start),
      );
      expect(tag, contains('android:exported="false"'));
    });

    test(
      'the merged manifest adds nothing that can start the process, when built',
      () {
        final file = File(mergedManifest);
        if (!file.existsSync()) {
          markTestSkipped(
            'no merged manifest; run :app:processReleaseMainManifest',
          );
          return;
        }
        final merged = declared(file.readAsStringSync());
        for (final action in bootActions) {
          expect(merged, isNot(contains(action)), reason: action);
        }
        expect(merged, isNot(contains('directBootAware="true"')));
        expect(merged, isNot(contains('androidx.work')));
        // Every receiver and provider, with the reason it may exist.
        final receivers = RegExp(
          r'<receiver\s[^>]*android:name="([^"]+)"',
        ).allMatches(merged).map((m) => m.group(1)).toSet();
        expect(receivers, {
          // share_plus: learns which app the user shared to; not exported.
          'dev.fluttercommunity.plus.share.SharePlusPendingIntent',
          // AndroidX: exported only to holders of android.permission.DUMP
          // (adb, the system); installs baseline profiles.
          'androidx.profileinstaller.ProfileInstallReceiver',
        });
        final providers = RegExp(
          r'<provider\s[^>]*android:name="([^"]+)"',
        ).allMatches(merged).map((m) => m.group(1)).toSet();
        expect(providers, {
          // share_plus: serves files the user shares; not exported.
          'dev.fluttercommunity.plus.share.ShareFileProvider',
          // AndroidX Startup: runs only inside a process something else started.
          'androidx.startup.InitializationProvider',
        });
        final initializers = RegExp(
          r'<meta-data\s[^>]*android:name="([^"]+Initializer)"',
        ).allMatches(merged).map((m) => m.group(1)).toSet();
        expect(initializers, {
          'androidx.lifecycle.ProcessLifecycleInitializer',
          'androidx.profileinstaller.ProfileInstallerInitializer',
        });
      },
    );
  });

  group('service lifecycle', () {
    test('Android never restarts the service by itself', () {
      final source = read(service);
      expect(source, isNot(contains('START_STICKY')));
      expect(source, isNot(contains('START_REDELIVER_INTENT')));
      expect(source, isNot(contains('override fun onTaskRemoved')));
      expect(source, contains('return START_NOT_STICKY'));
      expect(
        source,
        contains('ServiceCommand.of(intent != null, intent?.action)'),
      );
      expect(source, contains('ServiceCommand.IGNORE_OS_RESTART ->'));
    });

    test(
      'a refused foreground start ends the service instead of crashing it',
      () {
        final source = read(service);
        expect(source, contains('ForegroundServiceStartNotAllowedException'));
        expect(source, contains('if (!enterForeground('));
      },
    );

    test('only the app itself starts the service', () {
      final starters = kotlinSources()
          .where((path) => read(path).contains('startForegroundService('))
          .toList();
      expect(
        starters,
        [service],
        reason:
            'start and restart live in PocketClawService and are called '
            'from the Flutter channel, which needs the Activity',
      );
    });

    test('no scheduled or deferred work exists', () {
      for (final path in kotlinSources()) {
        // Code only: a doc comment may name what the code does not use.
        final source = read(path)
            .split('\n')
            .where((line) => !RegExp(r'^\s*(//|\*|/\*)').hasMatch(line))
            .join('\n');
        for (final api in const [
          'AlarmManager',
          'JobScheduler',
          'JobService',
          'WorkManager',
          'setExactAndAllowWhileIdle',
        ]) {
          expect(source, isNot(contains(api)), reason: '$api in $path');
        }
      }
    });

    test('lifecycle entry points leave local breadcrumbs', () {
      expect(
        read('$kotlin/PocketClawApp.kt'),
        contains('LifecycleDiagnostics.onProcessStart(this)'),
      );
      expect(
        read('$kotlin/MainActivity.kt'),
        contains(
          'LifecycleDiagnostics.record(\n            this, "activity", "create"',
        ),
      );
      final source = read(service).replaceAll(RegExp(r'\s+'), ' ');
      for (final event in const ['"create"', '"start-command"', '"destroy"']) {
        expect(source, contains('"service", $event'), reason: event);
      }
    });
  });
}
