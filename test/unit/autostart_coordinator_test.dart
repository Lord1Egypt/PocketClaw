import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/autostart_coordinator.dart';

class _FakeRuntime implements AutoStartRuntime {
  AutoStartRuntimeState serviceState = AutoStartRuntimeState.stopped;
  AutoStartRuntimeState gatewayState = AutoStartRuntimeState.stopped;
  AutoStartTransitionResult? serviceResult;
  AutoStartTransitionResult? gatewayResult;
  Completer<void>? serviceGate;
  Completer<void>? gatewayGate;
  int serviceStarts = 0;
  int gatewayStarts = 0;
  Future<AutoStartTransitionResult>? _serviceTask;
  Future<AutoStartTransitionResult>? _gatewayTask;

  @override
  Future<AutoStartRuntimeState> inspectServiceState() async => serviceState;

  @override
  Future<AutoStartRuntimeState> inspectGatewayState() async => gatewayState;

  @override
  Future<AutoStartTransitionResult> ensureServiceReady({
    required String operationId,
    required String source,
  }) {
    final current = _serviceTask;
    if (current != null) return current;
    final task = _startService();
    _serviceTask = task;
    task.whenComplete(() => _serviceTask = null);
    return task;
  }

  Future<AutoStartTransitionResult> _startService() async {
    final actualStart = serviceState != AutoStartRuntimeState.starting;
    if (actualStart) serviceStarts += 1;
    serviceState = AutoStartRuntimeState.starting;
    await serviceGate?.future;
    final result =
        serviceResult ??
        AutoStartTransitionResult(
          state: AutoStartRuntimeState.running,
          actualStart: actualStart,
          reason: 'ready',
        );
    serviceState = result.state;
    return result;
  }

  @override
  Future<AutoStartTransitionResult> ensureGatewayReady({
    required String operationId,
    required String source,
  }) {
    final current = _gatewayTask;
    if (current != null) return current;
    final task = _startGateway();
    _gatewayTask = task;
    task.whenComplete(() => _gatewayTask = null);
    return task;
  }

  Future<AutoStartTransitionResult> _startGateway() async {
    final actualStart = gatewayState != AutoStartRuntimeState.starting;
    if (actualStart) gatewayStarts += 1;
    gatewayState = AutoStartRuntimeState.starting;
    await gatewayGate?.future;
    final result =
        gatewayResult ??
        AutoStartTransitionResult(
          state: AutoStartRuntimeState.running,
          actualStart: actualStart,
          reason: 'ready',
        );
    gatewayState = result.state;
    return result;
  }
}

AutoStartCoordinator _coordinator(
  _FakeRuntime runtime, [
  List<String>? events,
]) => AutoStartCoordinator(
  runtime: runtime,
  logger: (event, _) => events?.add(event),
  clock: () => DateTime.utc(2026, 8, 29),
);

const _bothOn = AutoStartPreferences(
  serviceEnabled: true,
  gatewayEnabled: true,
);

void main() {
  test('Service and Gateway auto-start defaults are both on', () {
    expect(AutoStartPreferences.defaults.serviceEnabled, isTrue);
    expect(AutoStartPreferences.defaults.gatewayEnabled, isTrue);
  });

  test('fresh launch starts Service then Gateway exactly once', () async {
    final runtime = _FakeRuntime();
    final result = await _coordinator(
      runtime,
    ).ensureForAppOpen(preferences: _bothOn, source: 'app_launch');

    expect(runtime.serviceStarts, 1);
    expect(runtime.gatewayStarts, 1);
    expect(result.serviceState, AutoStartRuntimeState.running);
    expect(result.gatewayState, AutoStartRuntimeState.running);
  });

  test('running Service with stopped Gateway starts only Gateway', () async {
    final runtime = _FakeRuntime()
      ..serviceState = AutoStartRuntimeState.running;
    await _coordinator(
      runtime,
    ).ensureForAppOpen(preferences: _bothOn, source: 'app_launch');

    expect(runtime.serviceStarts, 0);
    expect(runtime.gatewayStarts, 1);
  });

  test('running Service and Gateway are not restarted', () async {
    final runtime = _FakeRuntime()
      ..serviceState = AutoStartRuntimeState.running
      ..gatewayState = AutoStartRuntimeState.running;
    await _coordinator(
      runtime,
    ).ensureForAppOpen(preferences: _bothOn, source: 'app_launch');

    expect(runtime.serviceStarts, 0);
    expect(runtime.gatewayStarts, 0);
  });

  test('Gateway off starts only Service', () async {
    final runtime = _FakeRuntime();
    await _coordinator(runtime).ensureForAppOpen(
      preferences: const AutoStartPreferences(
        serviceEnabled: true,
        gatewayEnabled: false,
      ),
      source: 'app_launch',
    );

    expect(runtime.serviceStarts, 1);
    expect(runtime.gatewayStarts, 0);
  });

  test('both preferences off perform no startup action', () async {
    final runtime = _FakeRuntime();
    await _coordinator(runtime).ensureForAppOpen(
      preferences: const AutoStartPreferences(
        serviceEnabled: false,
        gatewayEnabled: false,
      ),
      source: 'app_launch',
    );

    expect(runtime.serviceStarts, 0);
    expect(runtime.gatewayStarts, 0);
  });

  test('Gateway on never overrides disabled Service dependency', () async {
    final runtime = _FakeRuntime();
    final events = <String>[];
    await _coordinator(runtime, events).ensureForAppOpen(
      preferences: const AutoStartPreferences(
        serviceEnabled: false,
        gatewayEnabled: true,
      ),
      source: 'app_launch',
    );

    expect(runtime.serviceStarts, 0);
    expect(runtime.gatewayStarts, 0);
    expect(events, contains('gateway.start.skipped'));
  });

  test(
    'repeated resume evaluations do not duplicate Service or Gateway',
    () async {
      final runtime = _FakeRuntime()..serviceGate = Completer<void>();
      final coordinator = _coordinator(runtime);
      final first = coordinator.ensureForAppOpen(
        preferences: _bothOn,
        source: 'app_launch',
      );
      final second = coordinator.ensureForAppOpen(
        preferences: _bothOn,
        source: 'app_resume',
      );
      await Future<void>.delayed(Duration.zero);
      expect(runtime.serviceStarts, 1);
      runtime.serviceGate!.complete();
      await Future.wait([first, second]);

      await coordinator.ensureForAppOpen(
        preferences: _bothOn,
        source: 'app_resume',
      );
      expect(runtime.serviceStarts, 1);
      expect(runtime.gatewayStarts, 1);
    },
  );

  test(
    'concurrent manual and automatic Service start share one transition',
    () async {
      final runtime = _FakeRuntime()..serviceGate = Completer<void>();
      final manual = runtime.ensureServiceReady(
        operationId: 'manual',
        source: 'manual',
      );
      final automatic = _coordinator(runtime).ensureForAppOpen(
        preferences: const AutoStartPreferences(
          serviceEnabled: true,
          gatewayEnabled: false,
        ),
        source: 'app_resume',
      );
      await Future<void>.delayed(Duration.zero);
      expect(runtime.serviceStarts, 1);
      runtime.serviceGate!.complete();
      await Future.wait([manual, automatic]);
      expect(runtime.serviceStarts, 1);
    },
  );

  test(
    'repeated resume while Gateway is starting joins one Gateway start',
    () async {
      final runtime = _FakeRuntime()
        ..serviceState = AutoStartRuntimeState.running
        ..gatewayGate = Completer<void>();
      final coordinator = _coordinator(runtime);
      final first = coordinator.ensureForAppOpen(
        preferences: _bothOn,
        source: 'app_launch',
      );
      await Future<void>.delayed(Duration.zero);
      expect(runtime.gatewayStarts, 1);

      final second = coordinator.ensureForAppOpen(
        preferences: _bothOn,
        source: 'app_resume',
      );
      runtime.gatewayGate!.complete();
      await Future.wait([first, second]);
      expect(runtime.gatewayStarts, 1);
    },
  );

  test('Gateway timeout produces deterministic failed state', () async {
    final runtime = _FakeRuntime()
      ..serviceState = AutoStartRuntimeState.running
      ..gatewayResult = const AutoStartTransitionResult(
        state: AutoStartRuntimeState.failed,
        actualStart: true,
        reason: 'health_check',
        error: 'gateway readiness timed out',
        timedOut: true,
      );
    final events = <String>[];
    final result = await _coordinator(
      runtime,
      events,
    ).ensureForAppOpen(preferences: _bothOn, source: 'app_launch');

    expect(result.gatewayState, AutoStartRuntimeState.failed);
    expect(events, contains('gateway.start.timeout'));
  });

  test('Service failure prevents blind Gateway start', () async {
    final runtime = _FakeRuntime()
      ..serviceResult = const AutoStartTransitionResult(
        state: AutoStartRuntimeState.failed,
        actualStart: true,
        reason: 'start_rejected',
        error: 'service start failed',
      );
    final result = await _coordinator(
      runtime,
    ).ensureForAppOpen(preferences: _bothOn, source: 'app_launch');

    expect(result.serviceState, AutoStartRuntimeState.failed);
    expect(runtime.gatewayStarts, 0);
  });

  test(
    'evaluation log identifies canonical preference source and states',
    () async {
      final runtime = _FakeRuntime()
        ..serviceState = AutoStartRuntimeState.running
        ..gatewayState = AutoStartRuntimeState.running;
      final records = <Map<String, Object?>>[];
      final coordinator = AutoStartCoordinator(
        runtime: runtime,
        logger: (event, metadata) {
          if (event == 'autostart.evaluate') records.add(metadata);
        },
        clock: () => DateTime.utc(2026, 8, 29),
      );

      await coordinator.ensureForAppOpen(
        preferences: _bothOn,
        source: 'app_resume',
        preferenceSource: 'android_native_canonical',
      );

      expect(records, hasLength(1));
      expect(records.single, containsPair('service_autostart', true));
      expect(records.single, containsPair('gateway_autostart', true));
      expect(records.single, containsPair('service_state', 'running'));
      expect(records.single, containsPair('gateway_state', 'running'));
      expect(
        records.single,
        containsPair('preference_source', 'android_native_canonical'),
      );
    },
  );
}
