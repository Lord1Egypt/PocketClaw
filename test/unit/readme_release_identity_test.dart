import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// PC-DEF-081. The README described v0.2.0 — its download badge, checksum file,
/// APK name, hash and signing commit — for as long as v0.2.1 was the release.
///
/// The README describes the latest *published* release, which is not the
/// version under development: pubspec moves to the next version as soon as
/// work on it starts, long before anything is published. Holding the README to
/// pubspec either failed every development build or pushed the README into
/// announcing a release that does not exist. So the published identity is
/// pinned in docs/release/published.json, which changes only when a release is
/// published, and this test holds the README to it and pubspec at or ahead of
/// it. Nothing here asks GitHub.
void main() {
  final readme = File('README.md').readAsStringSync();
  final published =
      jsonDecode(File('docs/release/published.json').readAsStringSync())
          as Map<String, dynamic>;
  final version = published['version'] as String;
  final build = published['build'] as int;

  final declared = RegExp(
    r'^version:\s*([0-9]+)\.([0-9]+)\.([0-9]+)\+([0-9]+)',
    multiLine: true,
  ).firstMatch(File('pubspec.yaml').readAsStringSync())!;

  List<int> parts(String v) => v.split('.').map(int.parse).toList();

  int compareVersions(List<int> a, List<int> b) {
    for (var i = 0; i < 3; i++) {
      if (a[i] != b[i]) return a[i].compareTo(b[i]);
    }
    return 0;
  }

  test('the pinned identity is complete and self-consistent', () {
    expect(version, matches(RegExp(r'^[0-9]+\.[0-9]+\.[0-9]+$')));
    expect(published['tag'], 'v$version');
    expect(published['apk'], 'PocketClaw-v$version-arm64-v8a.apk');
    // v0.2.1 published PocketClaw-v0.2.1-SHA256SUMS.txt; from v0.2.2 the
    // checksum asset is SHA256SUMS.txt. Either is a checksum file for this
    // release; the README must name whichever was actually published.
    expect(
      published['checksums'],
      anyOf('SHA256SUMS.txt', 'PocketClaw-v$version-SHA256SUMS.txt'),
    );
    expect(published['commit'], matches(RegExp(r'^[0-9a-f]{40}$')));
    expect(published['apk_sha256'], matches(RegExp(r'^[0-9a-f]{64}$')));
    expect(published['apk_bytes'], isA<int>());
  });

  test('the published signer is the enrolled production signer', () {
    final enrolled = File('android/release-signing-cert.sha256')
        .readAsLinesSync()
        .firstWhere((line) => RegExp(r'^[0-9a-f]{64}$').hasMatch(line));
    expect(published['signer_sha256'], enrolled);
    expect(readme, contains('SHA-256 digest: $enrolled'));
  });

  test('the download badge and link name the published version', () {
    expect(readme, contains('Download-v${version}_APK'));
    expect(readme, contains('**[PocketClaw v$version — arm64-v8a APK]'));
  });

  test('the verification block names the published artifact', () {
    expect(readme, contains('sha256sum -c ${published['checksums']}'));
    expect(readme, contains('${published['apk_sha256']}  ${published['apk']}'));
    expect(
      readme,
      contains('apksigner verify --print-certs ${published['apk']}'),
    );
  });

  test('the release and status sections name the published tag', () {
    final commit = published['commit'] as String;
    expect(
      readme,
      contains('This release is tag `${published['tag']}` at commit'),
    );
    expect(readme, contains('commit/$commit'));
    expect(
      readme,
      contains('**Released.** `${published['tag']}` (build $build)'),
    );
  });

  test('no release-identity line names an unpublished version', () {
    for (final line in readme.split('\n')) {
      for (final match in RegExp(
        r'PocketClaw-v([0-9]+\.[0-9]+\.[0-9]+)-',
      ).allMatches(line)) {
        expect(match.group(1), version, reason: line);
      }
    }
  });

  test('the development version is the published one or ahead of it', () {
    final developing = [
      int.parse(declared.group(1)!),
      int.parse(declared.group(2)!),
      int.parse(declared.group(3)!),
    ];
    final developingBuild = int.parse(declared.group(4)!);
    final order = compareVersions(developing, parts(version));
    expect(
      order,
      greaterThanOrEqualTo(0),
      reason: 'pubspec is behind the published release',
    );
    if (order == 0) {
      expect(
        developingBuild,
        build,
        reason: 'a published version cannot be rebuilt under another code',
      );
    } else {
      expect(
        developingBuild,
        greaterThan(build),
        reason: 'the next version needs a higher versionCode',
      );
    }
  });
}
