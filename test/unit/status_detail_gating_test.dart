import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// The detailed Status payload rides on the health poll that already runs
/// every three seconds for Start/Stop. That timer must keep running whatever
/// screen is visible, so what these tests pin is the gate on the *extra* work:
/// detail is asked for only while the Status tab is showing, and the snapshot
/// is dropped the moment it is not.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/picoclaw');

  late ServiceManager service;
  late List<bool> detailFlagsSeen;
  late bool serviceRunning;

  final detailPayload = jsonEncode({
    'activity': {'active_turns': 1, 'completed': 12},
    'model': {'active_model': 'mimo-v2.5', 'provider': 'openai'},
    'channels': [
      {'name': 'telegram', 'configured': true, 'started': true, 'running': true},
    ],
    'resources': {'memory_rss_bytes': 88080384, 'cpu_seconds': 12.0},
  });

  void installNativeStub() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          switch (call.method) {
            case 'getServiceStatus':
              return <String, Object?>{
                'isRunning': serviceRunning,
                'pid': 4242,
                'lastLog': '',
              };
            case 'takeNewLogs':
              return <String>[];
            case 'checkHealth':
              final wantDetail =
                  (call.arguments as Map?)?['detail'] as bool? ?? false;
              detailFlagsSeen.add(wantDetail);
              return <String, Object?>{
                'isHealthy': true,
                'status': 'ok',
                'uptime': '1h24m0s',
                'pid': 4242,
                'error': '',
                // The host only returns detail when it was asked for, which is
                // what makes an unwatched Status tab free.
                'detail': wantDetail ? detailPayload : null,
              };
            default:
              return null;
          }
        });
  }

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
    detailFlagsSeen = <bool>[];
    serviceRunning = true;
    service.setStatusDetailWanted(false);
    installNativeStub();
  });

  tearDown(() {
    service.setStatusDetailWanted(false);
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('does not request detail while Status is not showing', () async {
    await service.pollNativeServiceStatusForTest();
    await service.pollNativeServiceStatusForTest();

    expect(detailFlagsSeen, everyElement(isFalse));
    expect(service.statusSnapshot, isNull);
    // Basic health still works, so Start/Stop is unaffected.
    expect(service.healthUptime, '1h24m0s');
  });

  test('requests detail and exposes a snapshot while Status is showing', () async {
    service.setStatusDetailWanted(true);
    await service.pollNativeServiceStatusForTest();

    expect(detailFlagsSeen.last, isTrue);
    final snapshot = service.statusSnapshot;
    expect(snapshot, isNotNull);
    expect(snapshot!.activity.completed, 12);
    expect(snapshot.model.activeModel, 'mimo-v2.5');
    expect(snapshot.channels.single.name, 'telegram');
  });

  test('leaving Status stops detail requests and drops the snapshot', () async {
    service.setStatusDetailWanted(true);
    await service.pollNativeServiceStatusForTest();
    expect(service.statusSnapshot, isNotNull);

    service.setStatusDetailWanted(false);
    // The retained reading is dropped immediately, so a stale snapshot can
    // never be shown as current when Status is opened again.
    expect(service.statusSnapshot, isNull);

    detailFlagsSeen.clear();
    await service.pollNativeServiceStatusForTest();
    await service.pollNativeServiceStatusForTest();

    expect(detailFlagsSeen, everyElement(isFalse));
    expect(service.statusSnapshot, isNull);
  });

  test('a stopped gateway clears the snapshot even while Status shows', () async {
    service.setStatusDetailWanted(true);
    await service.pollNativeServiceStatusForTest();
    expect(service.statusSnapshot, isNotNull);

    serviceRunning = false;
    await service.pollNativeServiceStatusForTest();

    expect(service.statusSnapshot, isNull);
    expect(service.healthUptime, '');
  });

  test('notifies listeners as fresh snapshots arrive', () async {
    // The service status does not change between polls, so without an explicit
    // notification for a new snapshot the screen would render the first
    // reading and then never update.
    service.setStatusDetailWanted(true);
    await service.pollNativeServiceStatusForTest();

    var notifications = 0;
    void listener() => notifications++;
    service.addListener(listener);
    addTearDown(() => service.removeListener(listener));

    await service.pollNativeServiceStatusForTest();
    expect(notifications, greaterThan(0));
  });
}
