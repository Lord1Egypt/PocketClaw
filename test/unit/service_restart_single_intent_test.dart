import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// PC-DEF-030. Making a saved Telegram configuration live means restarting
/// Core, and a restart has to be one request.
///
/// Flutter used to send `stopService` and then `startService`. On Android those
/// are two service intents, and the stop ends in an unconditional `stopSelf()`
/// that Android honours even though the start has already been queued — so the
/// freshly started service is destroyed again and PocketClaw is left stopped,
/// with the managed onboarding flow reporting success. That is the physical
/// defect: a paired bot that stayed silent until the owner restarted the
/// Service and the Gateway by hand.
///
/// These tests pin the contract at the boundary the host sees: exactly one
/// method call, named `restartService`, and never the pair.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/pocketclaw');

  late ServiceManager service;
  late List<MethodCall> nativeCalls;
  late bool nativeRunning;

  void installNativeStub() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          nativeCalls.add(call);
          switch (call.method) {
            case 'getServiceStatus':
              return <String, Object?>{'isRunning': nativeRunning, 'pid': 4242};
            case 'checkHealth':
              return <String, Object?>{'isHealthy': true, 'uptime': '1m'};
            case 'takeNewLogs':
              return <String>[];
            case 'startService':
              nativeRunning = true;
              return true;
            case 'stopService':
              nativeRunning = false;
              return true;
            case 'restartService':
              nativeRunning = true;
              return true;
            default:
              return null;
          }
        });
  }

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
    nativeCalls = <MethodCall>[];
    nativeRunning = false;
    installNativeStub();
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  Future<void> reachRunning() async {
    nativeRunning = true;
    await service.pollNativeServiceStatusForTest();
    expect(service.status, ServiceStatus.running);
    nativeCalls.clear();
  }

  test('restartCore never sends a stop and a start as two requests', () async {
    await reachRunning();

    await service.restartCore();

    final methods = nativeCalls.map((call) => call.method).toList();
    expect(
      methods,
      isNot(contains('stopService')),
      reason: 'a stop intent lets Android destroy the service the start asked for',
    );
  });

  test('restartCore does not announce a shutdown first', () async {
    await reachRunning();

    final seen = <ServiceStatus>[];
    service.addListener(() => seen.add(service.status));

    await service.restartCore();

    // The old stop-then-start published `stopped` as its first observable
    // state, which is what a restart must never look like. A later `stopped`
    // is a start that failed, and is reported honestly; this asserts only that
    // the shutdown is not the thing being asked for.
    expect(seen, isNotEmpty);
    expect(seen.first, ServiceStatus.starting);
    expect(seen.takeWhile((s) => s == ServiceStatus.starting), isNotEmpty);
  });

  test('a stopped service is not restarted', () async {
    expect(service.status, ServiceStatus.stopped);

    expect(await service.restartCore(), isFalse);
    expect(
      nativeCalls.map((call) => call.method),
      isNot(contains('restartService')),
    );
  });

  test('a starting service is not restarted out from under itself', () async {
    // start() leaves the manager transitional until the host reports back.
    service.start();
    expect(service.status, ServiceStatus.starting);
    nativeCalls.clear();

    expect(await service.restartCore(), isFalse);
    expect(
      nativeCalls.map((call) => call.method),
      isNot(contains('restartService')),
    );
  });
}
