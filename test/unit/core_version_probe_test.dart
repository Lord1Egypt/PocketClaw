import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// PC-DEF-063. A failed Core version probe must not become the Core version.
///
/// The About screen showed the Core version as unknown and only later as
/// 0.3.1. The cause was one value doing two jobs: the adapter answered the
/// string 'unknown' whether the probe failed or there was nothing to report,
/// that string passed the non-empty test in the cache, and it was then
/// displayed as the version until something happened to re-probe.
///
/// A failure is an absence now, and an absence is not cached — so the next read
/// tries again, which is the behaviour these pin.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late ServiceManager service;
  late Directory tempDir;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
    tempDir = await Directory.systemTemp.createTemp('core-version-probe-test');
  });

  tearDown(() async {
    if (!(Platform.isWindows || Platform.isAndroid)) {
      await service.updateConfig(
        '127.0.0.1',
        18800,
        binaryPath: '',
        arguments: '',
        publicMode: false,
      );
    }
    if (await tempDir.exists()) {
      await tempDir.delete(recursive: true);
    }
  });

  Future<String> writeExecutable(String name, String script) async {
    final file = File('${tempDir.path}/$name');
    await file.writeAsString(script);
    await Process.run('chmod', ['+x', file.path]);
    return file.path;
  }

  Future<void> useBinary(String path) => service.updateConfig(
    '127.0.0.1',
    18800,
    binaryPath: path,
    arguments: '',
    publicMode: false,
  );

  test('a failing probe reports absence rather than a version', () async {
    if (Platform.isWindows || Platform.isAndroid) return;

    final failing = await writeExecutable('picoclaw-failing', '#!/bin/sh\nexit 1\n');
    await useBinary(failing);

    expect(await service.getCoreVersion(), isNull);
    // The label stays empty, which reads as "not known yet" and keeps the
    // pending state honest. It must never be the word unknown.
    expect(service.coreVersionLabel, isEmpty);
  });

  test('a probe that prints nothing reports absence', () async {
    if (Platform.isWindows || Platform.isAndroid) return;

    final silent = await writeExecutable('picoclaw-silent', '#!/bin/sh\nexit 0\n');
    await useBinary(silent);

    expect(await service.getCoreVersion(), isNull);
    expect(service.coreVersionLabel, isEmpty);
  });

  // The regression itself: one transient failure used to be cached and shown as
  // the Core version, so a later successful probe was the only way out.
  test('a failed probe is not cached, so the next read succeeds', () async {
    if (Platform.isWindows || Platform.isAndroid) return;

    final failing = await writeExecutable('picoclaw-transient', '#!/bin/sh\nexit 1\n');
    await useBinary(failing);
    expect(await service.getCoreVersion(), isNull);

    final working = await writeExecutable(
      'picoclaw-working',
      "#!/bin/sh\nprintf 'version: 0.3.1\\n'\n",
    );
    await useBinary(working);

    expect(await service.getCoreVersion(), '0.3.1');
    expect(service.coreVersionLabel, '0.3.1');
  });

  test('a successful probe is cached and reported by the label', () async {
    if (Platform.isWindows || Platform.isAndroid) return;

    final working = await writeExecutable(
      'picoclaw-version',
      "#!/bin/sh\nprintf 'version: 0.3.1\\n'\n",
    );
    await useBinary(working);

    expect(await service.getCoreVersion(), '0.3.1');
    expect(service.coreVersionLabel, '0.3.1');
  });
}
