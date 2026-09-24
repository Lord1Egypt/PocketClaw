import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// PC-DEF-081. The README described v0.2.0 — its download badge, checksum file,
/// APK name, hash and signing commit — for as long as v0.2.1 was the release.
/// Nothing compared the two. This holds the README's release identity to the
/// version the app declares, so the next version bump fails here until the
/// README moves with it.
void main() {
  final pubspec = File('pubspec.yaml').readAsStringSync();
  final readme = File('README.md').readAsStringSync();
  final versionName =
      RegExp(r'^version:\s*([0-9]+\.[0-9]+\.[0-9]+)\+', multiLine: true)
          .firstMatch(pubspec)!
          .group(1)!;

  test('the download badge and link name the declared version', () {
    expect(readme, contains('Download-v${versionName}_APK'));
    expect(readme, contains('**[PocketClaw v$versionName — arm64-v8a APK]'));
  });

  test('the verification block names the declared version', () {
    expect(readme, contains('PocketClaw-v$versionName-SHA256SUMS.txt'));
    expect(readme, contains('PocketClaw-v$versionName-arm64-v8a.apk'));
    expect(readme, contains('**Released.** `v$versionName`'));
  });

  test('no release-identity line names another version', () {
    for (final line in readme.split('\n')) {
      final named = RegExp(r'PocketClaw-v([0-9]+\.[0-9]+\.[0-9]+)-')
          .allMatches(line)
          .map((m) => m.group(1));
      for (final version in named) {
        expect(version, versionName, reason: line);
      }
    }
  });
}
