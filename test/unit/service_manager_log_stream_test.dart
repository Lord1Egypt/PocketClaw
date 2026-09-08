import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Reproduces the physical-device regression where one backend log line filled
/// the entire Logs screen.
///
/// The native side exposes `lastLog`, a sticky snapshot of the most recent
/// line that never clears. The Flutter poll ran every three seconds and
/// appended that snapshot each time, so a single warning was re-added forever
/// until it evicted all 500 retained entries. The fix is a drain: the native
/// side hands each line out exactly once via `takeNewLogs`.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/pocketclaw');
  const staleWarning =
      'WRN api gateway.go:298 > removed stale pid file for PID 12302';

  late ServiceManager service;
  late List<String> pendingNativeLogs;
  late String stickyLastLog;
  late int takeNewLogsCalls;
  // ServiceManager is a singleton, so its log list survives between tests.
  // Each test asserts on the lines it appended rather than on the whole list.
  late int logBaseline;

  /// Stands in for the native service: `lastLog` stays set forever, while
  /// `takeNewLogs` drains, which is precisely the contract difference.
  void installNativeStub() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          switch (call.method) {
            case 'getServiceStatus':
              return <String, Object?>{
                'isRunning': false,
                'pid': 12302,
                'lastLog': stickyLastLog,
              };
            case 'takeNewLogs':
              takeNewLogsCalls++;
              final drained = List<String>.from(pendingNativeLogs);
              pendingNativeLogs.clear();
              return drained;
            default:
              return null;
          }
        });
  }

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
    stickyLastLog = '';
    pendingNativeLogs = <String>[];
    takeNewLogsCalls = 0;
    logBaseline = service.logs.length;
    installNativeStub();
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  /// Log appends are batched behind a 100 ms timer before they reach `logs`.
  Future<void> settleLogBatch() =>
      Future<void>.delayed(const Duration(milliseconds: 150));

  /// Only the lines this test appended.
  List<String> appendedLogs() => service.logs.sublist(logBaseline);

  test(
    'one backend warning is shown once, however many times status is polled',
    () async {
      // The backend emits the line exactly once...
      pendingNativeLogs.add(staleWarning);
      // ...and the sticky snapshot keeps reporting it forever afterwards.
      stickyLastLog = staleWarning;

      for (var poll = 0; poll < 25; poll++) {
        await service.pollNativeServiceStatusForTest();
      }
      await settleLogBatch();

      final occurrences = appendedLogs()
          .where((line) => line == staleWarning)
          .length;
      expect(
        occurrences,
        1,
        reason:
            'the device showed this single event repeated until it '
            'evicted every other log entry',
      );
      expect(takeNewLogsCalls, 25);
    },
  );

  test('a genuinely repeated backend line is not collapsed', () async {
    // Distinct emissions of identical text must survive: the fix is a drain,
    // not text deduplication.
    pendingNativeLogs.addAll([staleWarning, staleWarning, staleWarning]);
    stickyLastLog = staleWarning;

    await service.pollNativeServiceStatusForTest();
    await settleLogBatch();

    expect(appendedLogs().where((line) => line == staleWarning).length, 3);
  });

  test('polling with nothing new adds nothing', () async {
    stickyLastLog = staleWarning;

    for (var poll = 0; poll < 10; poll++) {
      await service.pollNativeServiceStatusForTest();
    }
    await settleLogBatch();

    expect(appendedLogs(), isEmpty);
  });

  test(
    'lines emitted between polls are all delivered, not just the last',
    () async {
      pendingNativeLogs.addAll(['first line', 'second line', 'third line']);
      stickyLastLog = 'third line';

      await service.pollNativeServiceStatusForTest();
      await settleLogBatch();

      expect(
        appendedLogs(),
        containsAllInOrder(<String>['first line', 'second line', 'third line']),
      );
    },
  );

  test('stored and therefore exported logs are plain Unicode text', () async {
    pendingNativeLogs.add('\x1b[38;2;213;70;70mتسلم 😊\x1b[0m\x1b[2K\u0000');

    await service.pollNativeServiceStatusForTest();
    await settleLogBatch();

    expect(appendedLogs(), <String>['تسلم 😊']);
    final exportedRepresentation = appendedLogs().join('\n');
    expect(exportedRepresentation, isNot(contains('\x1b')));
    expect(exportedRepresentation, isNot(contains('[38;2;')));
    expect(exportedRepresentation, isNot(contains('\u0000')));
  });

  test(
    'stored and exported logs apply the shared visibility contract',
    () async {
      pendingNativeLogs.addAll([
        '12:00 DBG http middleware.go:67 > GET /pico/ws 200 53.616µs',
        '12:01 DBG http middleware.go:67 > GET /pico/ws 500 1.2ms',
        '12:02 INF gateway gateway.go:1033 > Starting gateway process '
            '(/data/app/lib/arm64/libpicoclaw.so)',
        '12:03 INF pico pico.go:1013 > WebSocket client connected',
        '\x1b[38;2;1;2;3mمدة 53.616µs ✅\x1b[0m',
      ]);

      await service.pollNativeServiceStatusForTest();
      await settleLogBatch();

      final exportedRepresentation = appendedLogs().join('\n');
      expect(exportedRepresentation, isNot(contains('/pico/ws')));
      expect(exportedRepresentation, isNot(contains('libpicoclaw.so')));
      expect(exportedRepresentation, isNot(contains('\x1b')));
      expect(
        exportedRepresentation,
        contains('GET /internal realtime connection 500'),
      );
      expect(exportedRepresentation, contains('Starting gateway process'));
      expect(
        exportedRepresentation,
        contains('INF realtime realtime.go:1013 > WebSocket client connected'),
      );
      expect(exportedRepresentation, isNot(contains('INF pico pico.go:1013')));
      expect(exportedRepresentation, contains('مدة 53.616µs ✅'));
    },
  );

  test('no user-visible log line carries a module or developer path', () async {
    pendingNativeLogs.add(staleWarning);
    await service.pollNativeServiceStatusForTest();
    await settleLogBatch();

    for (final line in appendedLogs()) {
      for (final banned in const <String>[
        'github.com/sipeed',
        'picoclaw',
        'PicoClaw',
        'sipeed',
        'Sipeed',
        '/home/',
        '.upstream',
      ]) {
        expect(
          line.contains(banned),
          isFalse,
          reason: '"$line" leaks "$banned"',
        );
      }
    }
  });
}
