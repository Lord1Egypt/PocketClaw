import 'package:flutter/services.dart';
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
/// tries again, which is the behaviour these pin. They drive the Android host
/// channel, which is the only path the app has.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/pocketclaw');
  late ServiceManager service;

  /// Answers getCoreVersion with [reply]; an Exception is thrown instead.
  void hostAnswers(Object? reply) {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method != 'getCoreVersion') return null;
          if (reply is Exception) throw reply;
          return reply;
        });
  }

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('a failing probe reports absence rather than a version', () async {
    hostAnswers(PlatformException(code: 'CORE_VERSION_FAILED'));

    expect(await service.getCoreVersion(), isNull);
    // The label stays empty, which reads as "not known yet" and keeps the
    // pending state honest. It must never be the word unknown.
    expect(service.coreVersionLabel, isNot('unknown'));
  });

  test('a probe that answers nothing, or "unknown", reports absence', () async {
    for (final reply in <Object?>[null, '', '  ', 'unknown']) {
      hostAnswers(reply);
      expect(await service.getCoreVersion(), isNull, reason: '$reply');
    }
  });

  // The regression itself: one transient failure used to be cached and shown as
  // the Core version, so a later successful probe was the only way out.
  test('a failed probe is not cached, so the next read succeeds', () async {
    hostAnswers(PlatformException(code: 'CORE_VERSION_FAILED'));
    expect(await service.getCoreVersion(), isNull);

    hostAnswers('0.3.1');
    expect(await service.getCoreVersion(), '0.3.1');
    expect(service.coreVersionLabel, '0.3.1');
  });

  test('a successful probe is cached and reported by the label', () async {
    hostAnswers('0.3.1');

    expect(await service.getCoreVersion(), '0.3.1');
    expect(service.coreVersionLabel, '0.3.1');
  });
}
