import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// A credential change has to reach a running Core without the user restarting
/// the app by hand, and without ever interrupting a service that is mid-start.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/pocketclaw');

  late ServiceManager service;
  late List<String> nativeCalls;
  late bool nativeRunning;

  void installNativeStub() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          nativeCalls.add(call.method);
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
            default:
              return null;
          }
        });
  }

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
    nativeCalls = <String>[];
    nativeRunning = false;
    installNativeStub();
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('a stopped service needs no restart to pick the credential up', () async {
    final outcome = await service.applyCredentialChange();

    expect(outcome, CredentialApplyOutcome.notRunning);
    expect(nativeCalls, isNot(contains('stopService')));
    expect(service.hasPendingCredentialRestart, isFalse);
  });

  test('a running service is restarted so the change takes effect', () async {
    nativeRunning = true;
    await service.pollNativeServiceStatusForTest();
    expect(service.status, ServiceStatus.running);

    // The host has a desktop adapter rather than the Android service, so the
    // observable proof of a restart is the status transition, not a native
    // method name.
    final seen = <ServiceStatus>[];
    service.addListener(() => seen.add(service.status));

    final outcome = await service.applyCredentialChange();

    expect(outcome, CredentialApplyOutcome.applied);
    expect(seen, contains(ServiceStatus.stopped));
  });

  test('a starting service is never interrupted', () async {
    nativeRunning = false;
    unawaitedStart(service);
    expect(service.status, ServiceStatus.starting);

    final outcome = await service.applyCredentialChange();

    expect(outcome, CredentialApplyOutcome.deferred);
    expect(service.hasPendingCredentialRestart, isTrue);
    // Nothing was torn down while the service was still coming up.
    expect(nativeCalls, isNot(contains('stopService')));
  });

  test('a deferred change is applied once the service settles', () async {
    nativeRunning = false;
    unawaitedStart(service);
    await service.applyCredentialChange();
    expect(service.hasPendingCredentialRestart, isTrue);

    // The existing status poll is what notices the service has settled.
    final seen = <ServiceStatus>[];
    service.addListener(() => seen.add(service.status));
    nativeRunning = true;
    await service.pollNativeServiceStatusForTest();
    await Future<void>.delayed(Duration.zero);

    expect(service.hasPendingCredentialRestart, isFalse);
    expect(seen, contains(ServiceStatus.stopped));
  });
}

/// Starts the service without awaiting it, leaving the manager in its
/// transitional state for the duration of a test.
void unawaitedStart(ServiceManager service) {
  service.start();
}
