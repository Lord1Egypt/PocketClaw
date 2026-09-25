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
      'Android 8.0',
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

  test('the version being built has its changelog', () {
    final code = RegExp(
      r'^version:\s*[0-9.]+\+([0-9]+)',
      multiLine: true,
    ).firstMatch(File('pubspec.yaml').readAsStringSync())!.group(1)!;
    expect(
      File('$root/changelogs/$code.txt').existsSync(),
      isTrue,
      reason: 'F-Droid shows changelogs/<versionCode>.txt',
    );
  });

  // Screenshots must be captured from the app on a phone by the owner; none
  // are committed yet. When they are, each must be a real portrait PNG or
  // JPEG of phone size, never a placeholder.
  test('phone screenshots, when present, are real portrait captures', () {
    final dir = Directory('$root/images/phoneScreenshots');
    if (!dir.existsSync()) return;
    final files = dir.listSync().whereType<File>().toList();
    expect(files, isNotEmpty, reason: 'an empty directory is a placeholder');
    for (final file in files) {
      final bytes = file.readAsBytesSync();
      final (width, height) = _imageSize(bytes, file.path);
      expect(
        height,
        greaterThan(width),
        reason: '${file.path} is not portrait',
      );
      expect(width, greaterThanOrEqualTo(320), reason: file.path);
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

(int, int) _imageSize(List<int> b, String path) {
  int be16(int at) => (b[at] << 8) | b[at + 1];
  int be32(int at) => (be16(at) << 16) | be16(at + 2);
  if (b.length > 24 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47) {
    return (be32(16), be32(20));
  }
  if (b.length > 4 && b[0] == 0xFF && b[1] == 0xD8) {
    var at = 2;
    while (at + 9 < b.length && b[at] == 0xFF) {
      final marker = b[at + 1];
      if (marker >= 0xC0 && marker <= 0xC3) {
        return (be16(at + 7), be16(at + 5));
      }
      at += 2 + be16(at + 2);
    }
  }
  fail('$path is not a PNG or JPEG screenshot');
}
