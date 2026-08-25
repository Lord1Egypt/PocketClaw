import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('launch backgrounds use drawable-backed layers', () {
    for (final path in const [
      'android/app/src/main/res/drawable/launch_background.xml',
      'android/app/src/main/res/drawable-v21/launch_background.xml',
    ]) {
      final source = File(path).readAsStringSync();

      expect(source, contains('@color/pocketclaw_splash_background'));
      expect(source, isNot(contains('<item android:color=')));
    }
  });
}
