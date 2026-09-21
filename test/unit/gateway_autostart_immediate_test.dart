import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/pocketclaw_channel.dart';

/// PC-DEF-034. Turning "start Gateway automatically" on while the service is
/// already running must act now. The preference used to be persisted and
/// nothing else happened, so the switch looked broken and the only way forward
/// was a manual start the UI never mentioned.
///
/// These cover the channel contract the host implements. The OFF -> ON
/// transition itself is exercised on the device, and the authorization
/// boundary is proven in Go (android_gateway_start_test.go).
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/pocketclaw');

  tearDown(() async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('startGatewayNow reports a start', () async {
    var invoked = 0;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method != 'startGatewayNow') return null;
          invoked += 1;
          return 'ok';
        });

    expect(await PocketClawChannel.startGatewayNow(), 'ok');
    expect(invoked, 1);
  });

  test('startGatewayNow distinguishes an already-running Gateway', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method != 'startGatewayNow') return null;
          return 'already_running';
        });

    expect(await PocketClawChannel.startGatewayNow(), 'already_running');
  });

  test('a host that answers nothing is treated as a plain start', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async => null);

    expect(await PocketClawChannel.startGatewayNow(), 'ok');
  });

  test('a host failure surfaces rather than being swallowed', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          throw PlatformException(
            code: 'GATEWAY_START_FAILED',
            message: 'Gateway start failed (HTTP 500)',
          );
        });

    await expectLater(
      PocketClawChannel.startGatewayNow(),
      throwsA(isA<PlatformException>()),
    );
  });

  test('no bridge credential crosses the channel', () async {
    MethodCall? seen;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          seen = call;
          return 'ok';
        });

    await PocketClawChannel.startGatewayNow();

    // The host holds the Android bridge token. Flutter asks for the operation
    // and must never carry the credential.
    expect(seen, isNotNull);
    expect(seen!.arguments, isNull);
  });
}
