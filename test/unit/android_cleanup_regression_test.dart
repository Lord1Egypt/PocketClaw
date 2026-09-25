import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

// PC-DEF-080 and PC-DEF-083 removed a desktop host, dead background plugins,
// a boot receiver and Settings fields that could not work on Android. The
// dashboard side is held by config-page.android.test.tsx and the permission
// side by the release gate; this holds the Flutter and Android sources.
void main() {
  String read(String path) => File(path).readAsStringSync();

  test('no removed desktop or background plugin is declared again', () {
    final pubspec = read('pubspec.yaml');
    for (final plugin in const [
      'bitsdojo_window',
      'desktop_webview_window',
      'flutter_background_service',
      'flutter_local_notifications',
      'local_session_timeout',
      'process_run',
      'tray_manager',
      'webview_windows',
      'window_manager',
      'windows_single_instance',
    ]) {
      expect(
        RegExp('^\\s+$plugin:', multiLine: true).hasMatch(pubspec),
        isFalse,
        reason: '$plugin was removed in PC-DEF-083',
      );
    }
  });

  test('the app has no desktop platform and no desktop host', () {
    for (final platform in const ['linux', 'macos', 'windows']) {
      expect(Directory(platform).existsSync(), isFalse, reason: platform);
    }
    expect(File('lib/src/core/background_service.dart').existsSync(), isFalse);
  });

  test('no boot receiver source exists', () {
    final kotlin = Directory('android/app/src/main/kotlin');
    final receivers = kotlin
        .listSync(recursive: true)
        .whereType<File>()
        .where((f) => read(f.path).contains('BroadcastReceiver()'))
        .map((f) => f.path)
        .toList();
    expect(receivers, isEmpty, reason: 'PocketClaw declares no receiver');
  });

  test('Settings offers no port, arguments or launch-at-login field', () {
    final settings = read('lib/src/ui/config_page.dart');
    for (final removed in const [
      'l10n.port',
      'l10n.arguments',
      'launchAtLogin',
      'binaryPath',
    ]) {
      expect(settings.contains(removed), isFalse, reason: removed);
    }
    final english = read('lib/l10n/app_en.arb');
    for (final key in const [
      'arguments',
      'argumentsHint',
      'binaryPath',
      'showWindow',
    ]) {
      expect(english.contains('"$key":'), isFalse, reason: key);
    }
  });
}
