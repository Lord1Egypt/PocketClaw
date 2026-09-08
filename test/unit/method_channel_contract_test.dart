import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// The Flutter and Android halves of the platform bridge name the same channel.
///
/// Both ends ship in one APK and upgrade together, so there is no compatibility
/// alias and nothing to fall back to — which is exactly why a typo on one side
/// would be silent: every call would simply go unanswered at runtime rather
/// than fail to build. This reads the literal out of both sources.
void main() {
  const expected = 'com.lord1egypt.pocketclaw/pocketclaw';

  String read(String path) {
    final file = File(path);
    expect(file.existsSync(), isTrue, reason: '$path is missing');
    return file.readAsStringSync();
  }

  test('the Dart channel name is the canonical PocketClaw one', () {
    expect(read('lib/src/core/pocketclaw_channel.dart'), contains("'$expected'"));
  });

  test('the Kotlin channel name matches the Dart one exactly', () {
    final kotlin = read(
      'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/'
      'PocketClawMethodChannel.kt',
    );
    expect(kotlin, contains('"$expected"'));
  });

  test('no source still names the pre-N1 channel', () {
    for (final path in [
      'lib/src/core/pocketclaw_channel.dart',
      'lib/src/core/log_export_writer.dart',
      'lib/src/native/android_core_service_adapter.dart',
      'lib/src/ui/log_page.dart',
      'android/app/src/main/kotlin/com/lord1egypt/pocketclaw/'
          'PocketClawMethodChannel.kt',
    ]) {
      expect(
        read(path),
        isNot(contains('com.lord1egypt.pocketclaw/picoclaw')),
        reason: '$path still names the pre-N1 channel',
      );
    }
  });

  test('the manifest names the renamed Kotlin components', () {
    final manifest = read('android/app/src/main/AndroidManifest.xml');
    expect(manifest, contains('android:name=".PocketClawApp"'));
    expect(manifest, contains('android:name=".service.PocketClawService"'));
    expect(manifest, isNot(contains('PicoClawApp')));
    expect(manifest, isNot(contains('PicoClawService')));
  });
}
