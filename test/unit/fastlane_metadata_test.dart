import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Upstream store metadata F-Droid reads from this repository. The limits are
/// F-Droid's and Fastlane's; the icon is generated from the canonical mark by
/// tool/generate_android_launcher_icons.py, which the branding test checks.
void main() {
  const root = 'fastlane/metadata/android/en-US';
  String text(String name) => File('$root/$name').readAsStringSync().trim();

  test('the listing texts exist and fit their limits', () {
    expect(text('title.txt'), 'PocketClaw');
    expect(
      text('short_description.txt').runes.length,
      inInclusiveRange(10, 80),
    );
    expect(
      text('full_description.txt').runes.length,
      inInclusiveRange(200, 4000),
    );
  });

  test('every changelog fits the 500-character limit', () {
    final changelogs = Directory('$root/changelogs')
        .listSync()
        .whereType<File>()
        .where((file) => RegExp(r'/\d+\.txt$').hasMatch(file.path))
        .toList();
    expect(changelogs, isNotEmpty);
    for (final file in changelogs) {
      expect(
        file.readAsStringSync().trim().runes.length,
        lessThanOrEqualTo(500),
        reason: file.path,
      );
    }
  });

  test('the description states the scope a reviewer checks', () {
    final description = text('full_description.txt');
    for (final fact in const [
      'arm64-v8a',
      'Android 7.0',
      'API key',
      'non-free network services',
      'no analytics',
    ]) {
      expect(description, contains(fact), reason: fact);
    }
    // Settings removed in PC-DEF-080 must not be advertised.
    for (final removed in const [
      'Devices',
      'Launch at Login',
      'Service Port',
    ]) {
      expect(description, isNot(contains(removed)), reason: removed);
    }
  });

  test('the listing icon is a 512 px PNG', () {
    final bytes = File('$root/images/icon.png').readAsBytesSync();
    expect(bytes.sublist(1, 4), 'PNG'.codeUnits);
    int be32(int at) =>
        (bytes[at] << 24) |
        (bytes[at + 1] << 16) |
        (bytes[at + 2] << 8) |
        bytes[at + 3];
    expect(be32(16), 512);
    expect(be32(20), 512);
  });
}
