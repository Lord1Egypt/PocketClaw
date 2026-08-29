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

  test(
    'managed Gateway status and start use separate native operations',
    () async {
      final methods = <String>[];
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
            methods.add(call.method);
            if (call.method == 'getGatewayStatus') {
              return <String, Object?>{'gateway_status': 'stopped'};
            }
            if (call.method == 'startGateway') {
              return <String, Object?>{
                'status': 'already_starting',
                'gateway_status': 'starting',
              };
            }
            return null;
          });

      expect(
        await PicoClawChannel.getGatewayStatus(),
        containsPair('gateway_status', 'stopped'),
      );
      expect(
        await PicoClawChannel.startGateway(),
        containsPair('status', 'already_starting'),
      );
      expect(methods, ['getGatewayStatus', 'startGateway']);
    },
  );

  test(
    'launch auto-start preferences round-trip through native bridge',
    () async {
      final calls = <MethodCall>[];
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
            calls.add(call);
            return <String, Object?>{
              'serviceEnabled': true,
              'gatewayEnabled': false,
              'initialized': true,
              'source': 'android_native_canonical',
            };
          });

      final read = await PicoClawChannel.getLaunchAutoStartPreferences();
      final written = await PicoClawChannel.setLaunchAutoStartPreferences(
        gatewayEnabled: false,
      );

      expect(read.serviceEnabled, isTrue);
      expect(written.gatewayEnabled, isFalse);
      expect(calls.map((call) => call.method), [
        'getLaunchAutoStartPreferences',
        'setLaunchAutoStartPreferences',
      ]);
      expect(calls.last.arguments, {'gatewayEnabled': false});
    },
  );

  test('Service start and stop carry exact source and operation IDs', () async {
    final calls = <MethodCall>[];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          calls.add(call);
          return true;
        });

    await PicoClawChannel.startService(
      source: 'app_launch_autostart',
      operationId: 'start-1',
    );
    await PicoClawChannel.stopService(source: 'manual', operationId: 'stop-1');

    expect(calls.first.method, 'startService');
    expect(
      calls.first.arguments,
      containsPair('source', 'app_launch_autostart'),
    );
    expect(calls.first.arguments, containsPair('operationId', 'start-1'));
    expect(calls.last.method, 'stopService');
    expect(calls.last.arguments, containsPair('source', 'manual'));
    expect(calls.last.arguments, containsPair('operationId', 'stop-1'));
  });
}
