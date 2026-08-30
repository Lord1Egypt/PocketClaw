import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/picoclaw_channel.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/picoclaw');

  tearDown(() async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('getLaunchAutoStartPreferences maps the native record', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method != 'getLaunchAutoStartPreferences') return null;
          return <String, Object?>{
            'serviceEnabled': false,
            'gatewayEnabled': true,
            'initialized': true,
            'source': 'android_native_canonical',
          };
        });

    final prefs = await PicoClawChannel.getLaunchAutoStartPreferences();

    expect(prefs.serviceEnabled, isFalse);
    expect(prefs.gatewayEnabled, isTrue);
    expect(prefs.initialized, isTrue);
  });

  test('getLaunchAutoStartPreferences defaults a fresh install to on', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async => null);

    final prefs = await PicoClawChannel.getLaunchAutoStartPreferences();

    expect(prefs.serviceEnabled, isTrue);
    expect(prefs.gatewayEnabled, isTrue);
    expect(prefs.initialized, isFalse);
  });

  test('setLaunchAutoStartPreferences sends only the changed field', () async {
    MethodCall? observed;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          observed = call;
          return <String, Object?>{
            'serviceEnabled': true,
            'gatewayEnabled': false,
            'initialized': true,
            'source': 'android_native_canonical',
          };
        });

    final prefs = await PicoClawChannel.setLaunchAutoStartPreferences(
      gatewayEnabled: false,
    );

    expect(observed?.method, 'setLaunchAutoStartPreferences');
    expect(observed?.arguments, <String, Object?>{'gatewayEnabled': false});
    expect(prefs.serviceEnabled, isTrue);
    expect(prefs.gatewayEnabled, isFalse);
  });

  test('setLaunchAutoStartPreferences returns the native readback, '
      'not the requested value', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          // The host rejected the write and reports the state on disk.
          return <String, Object?>{
            'serviceEnabled': false,
            'gatewayEnabled': false,
            'initialized': true,
            'source': 'android_native_canonical',
          };
        });

    final prefs = await PicoClawChannel.setLaunchAutoStartPreferences(
      serviceEnabled: true,
    );

    expect(prefs.serviceEnabled, isFalse);
  });

  test('setLaunchAutoStartPreferences surfaces a refused commit', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          throw PlatformException(
            code: 'SET_LAUNCH_AUTOSTART_FAILED',
            message: 'Could not persist launch auto-start preferences',
          );
        });

    // A refused write must not be reported as a saved preference.
    await expectLater(
      PicoClawChannel.setLaunchAutoStartPreferences(serviceEnabled: false),
      throwsA(isA<PlatformException>()),
    );
  });

  test('getCoreVersion returns the native version string', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method == 'getCoreVersion') return '0.24.1';
          return null;
        });

    expect(await PicoClawChannel.getCoreVersion(), '0.24.1');
  });

  test('getCoreVersion falls back to unknown on native failure', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          throw PlatformException(code: 'ERR', message: 'boom');
        });

    expect(await PicoClawChannel.getCoreVersion(), 'unknown');
  });

  test('getCoreVersion maps blank or null native values to unknown', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method == 'getCoreVersion') return '   ';
          return null;
        });

    expect(await PicoClawChannel.getCoreVersion(), 'unknown');

    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          return null;
        });

    expect(await PicoClawChannel.getCoreVersion(), 'unknown');
  });

  test('getLanIpv4Address returns active native LAN address', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method == 'getLanIpv4Address') return '10.0.0.24';
          return null;
        });

    expect(await PicoClawChannel.getLanIpv4Address(), '10.0.0.24');
  });

  test('getLanIpv4Address maps blank native result to unavailable', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (_) async => '  ');

    expect(await PicoClawChannel.getLanIpv4Address(), isNull);
  });

  test('applyPublicMode returns the launcher actual mode and result', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          expect(call.method, 'applyPublicMode');
          expect(call.arguments, {'public': true});
          return <String, Object?>{
            'success': true,
            'public': true,
            'message': '',
          };
        });

    final result = await PicoClawChannel.applyPublicMode(true);
    expect(result.success, isTrue);
    expect(result.publicMode, isTrue);
    expect(result.message, isEmpty);
  });

  test('applyPublicMode preserves reported rollback state', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (_) async {
          return <String, Object?>{
            'success': false,
            'public': false,
            'message': 'Could not enable LAN access.',
          };
        });

    final result = await PicoClawChannel.applyPublicMode(true);
    expect(result.success, isFalse);
    expect(result.publicMode, isFalse);
    expect(result.message, contains('Could not enable'));
  });
}
