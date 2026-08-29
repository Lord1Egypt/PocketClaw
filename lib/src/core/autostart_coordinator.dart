import 'dart:async';

enum AutoStartRuntimeState { stopped, starting, running, failed }

class AutoStartPreferences {
  const AutoStartPreferences({
    required this.serviceEnabled,
    required this.gatewayEnabled,
  });

  final bool serviceEnabled;
  final bool gatewayEnabled;

  static const defaults = AutoStartPreferences(
    serviceEnabled: true,
    gatewayEnabled: true,
  );
}

class AutoStartTransitionResult {
  const AutoStartTransitionResult({
    required this.state,
    required this.actualStart,
    this.reason = '',
    this.error = '',
    this.timedOut = false,
    this.duration = Duration.zero,
    this.retryCount = 0,
    this.timeout = Duration.zero,
  });

  final AutoStartRuntimeState state;
  final bool actualStart;
  final String reason;
  final String error;
  final bool timedOut;
  final Duration duration;
  final int retryCount;
  final Duration timeout;

  bool get isReady => state == AutoStartRuntimeState.running;
}

abstract interface class AutoStartRuntime {
  Future<AutoStartRuntimeState> inspectServiceState();

  Future<AutoStartTransitionResult> ensureServiceReady({
    required String operationId,
    required String source,
  });

  Future<AutoStartRuntimeState> inspectGatewayState();

  Future<AutoStartTransitionResult> ensureGatewayReady({
    required String operationId,
    required String source,
  });
}

typedef AutoStartLifecycleLogger =
    void Function(String event, Map<String, Object?> metadata);

class AutoStartEvaluation {
  const AutoStartEvaluation({
    required this.operationId,
    required this.serviceState,
    required this.gatewayState,
    this.error = '',
  });

  final String operationId;
  final AutoStartRuntimeState serviceState;
  final AutoStartRuntimeState gatewayState;
  final String error;
}

/// Serializes app-open auto-start evaluation and delegates both transitions to
/// the same state-aware runtime methods used by manual controls.
class AutoStartCoordinator {
  AutoStartCoordinator({
    required AutoStartRuntime runtime,
    required AutoStartLifecycleLogger logger,
    DateTime Function()? clock,
  }) : _runtime = runtime,
       _logger = logger,
       _clock = clock ?? DateTime.now;

  final AutoStartRuntime _runtime;
  final AutoStartLifecycleLogger _logger;
  final DateTime Function() _clock;

  Future<AutoStartEvaluation>? _inFlight;
  String? _inFlightOperationId;
  int _operationSequence = 0;

  Future<AutoStartEvaluation> ensureForAppOpen({
    required AutoStartPreferences preferences,
    required String source,
    String preferenceSource = 'runtime_memory',
  }) {
    final current = _inFlight;
    if (current != null) {
      _logger('autostart.evaluate', {
        'operation_id': _inFlightOperationId ?? 'coalesced',
        'reason': 'evaluation_already_in_progress',
        'source': source,
        'service_autostart': preferences.serviceEnabled,
        'gateway_autostart': preferences.gatewayEnabled,
        'service_state': 'evaluation_in_progress',
        'gateway_state': 'evaluation_in_progress',
        'preference_source': preferenceSource,
        'result': 'skipped',
        'retry_count': 0,
      });
      return current;
    }

    final operationId = _newOperationId();
    final task = _evaluate(
      operationId: operationId,
      preferences: preferences,
      source: source,
      preferenceSource: preferenceSource,
    );
    _inFlight = task;
    _inFlightOperationId = operationId;
    task.whenComplete(() {
      if (identical(_inFlight, task)) {
        _inFlight = null;
        _inFlightOperationId = null;
      }
    });
    return task;
  }

  String _newOperationId() {
    _operationSequence += 1;
    return '${_clock().toUtc().microsecondsSinceEpoch}-$_operationSequence';
  }

  Future<AutoStartEvaluation> _evaluate({
    required String operationId,
    required AutoStartPreferences preferences,
    required String source,
    required String preferenceSource,
  }) async {
    final startedAt = _clock();
    var serviceState = AutoStartRuntimeState.stopped;
    var gatewayState = AutoStartRuntimeState.stopped;
    try {
      serviceState = await _runtime.inspectServiceState();
      gatewayState = serviceState == AutoStartRuntimeState.running
          ? await _runtime.inspectGatewayState()
          : AutoStartRuntimeState.stopped;
      _logger('autostart.evaluate', {
        'operation_id': operationId,
        'reason': 'app_open',
        'source': source,
        'service_autostart': preferences.serviceEnabled,
        'gateway_autostart': preferences.gatewayEnabled,
        'service_state': serviceState.name,
        'gateway_state': gatewayState.name,
        'preference_source': preferenceSource,
        'result': 'evaluated',
        'retry_count': 0,
      });

      if (!preferences.serviceEnabled) {
        _logSkipped(
          component: 'service',
          operationId: operationId,
          source: source,
          reason: 'preference_disabled',
          previousState: serviceState,
        );
      } else if (serviceState == AutoStartRuntimeState.running) {
        _logSkipped(
          component: 'service',
          operationId: operationId,
          source: source,
          reason: 'already_running',
          previousState: serviceState,
        );
      } else {
        final previousState = serviceState;
        if (previousState == AutoStartRuntimeState.starting) {
          _logSkipped(
            component: 'service',
            operationId: operationId,
            source: source,
            reason: 'already_starting',
            previousState: previousState,
          );
        } else {
          _logRequested(
            component: 'service',
            operationId: operationId,
            source: source,
            previousState: previousState,
          );
        }
        final result = await _runtime.ensureServiceReady(
          operationId: operationId,
          source: source,
        );
        serviceState = result.state;
        _logTransitionResult(
          component: 'service',
          operationId: operationId,
          source: source,
          previousState: previousState,
          result: result,
        );
      }

      if (!preferences.gatewayEnabled) {
        _logSkipped(
          component: 'gateway',
          operationId: operationId,
          source: source,
          reason: 'preference_disabled',
          previousState: gatewayState,
        );
      } else if (serviceState != AutoStartRuntimeState.running) {
        _logSkipped(
          component: 'gateway',
          operationId: operationId,
          source: source,
          reason: 'service_not_ready',
          previousState: gatewayState,
        );
      } else {
        gatewayState = await _runtime.inspectGatewayState();
        if (gatewayState == AutoStartRuntimeState.running) {
          _logSkipped(
            component: 'gateway',
            operationId: operationId,
            source: source,
            reason: 'already_running',
            previousState: gatewayState,
          );
        } else {
          final previousState = gatewayState;
          if (previousState == AutoStartRuntimeState.starting) {
            _logSkipped(
              component: 'gateway',
              operationId: operationId,
              source: source,
              reason: 'already_starting',
              previousState: previousState,
            );
          } else {
            _logRequested(
              component: 'gateway',
              operationId: operationId,
              source: source,
              previousState: previousState,
            );
          }
          final result = await _runtime.ensureGatewayReady(
            operationId: operationId,
            source: source,
          );
          gatewayState = result.state;
          _logTransitionResult(
            component: 'gateway',
            operationId: operationId,
            source: source,
            previousState: previousState,
            result: result,
          );
        }
      }

      return AutoStartEvaluation(
        operationId: operationId,
        serviceState: serviceState,
        gatewayState: gatewayState,
      );
    } catch (error) {
      _logger('autostart.evaluate', {
        'operation_id': operationId,
        'reason': 'unexpected_error',
        'source': source,
        'service_autostart': preferences.serviceEnabled,
        'gateway_autostart': preferences.gatewayEnabled,
        'service_state': serviceState.name,
        'gateway_state': gatewayState.name,
        'preference_source': preferenceSource,
        'duration_ms': _clock().difference(startedAt).inMilliseconds,
        'result': 'failed',
        'error': error.toString(),
        'retry_count': 0,
      });
      return AutoStartEvaluation(
        operationId: operationId,
        serviceState: serviceState,
        gatewayState: gatewayState,
        error: error.toString(),
      );
    }
  }

  void _logRequested({
    required String component,
    required String operationId,
    required String source,
    required AutoStartRuntimeState previousState,
  }) {
    _logger('$component.start.requested', {
      'operation_id': operationId,
      'reason': 'autostart_enabled',
      'source': source,
      'previous_state': previousState.name,
      'target_state': AutoStartRuntimeState.running.name,
      'result': 'requested',
      'retry_count': 0,
    });
  }

  void _logSkipped({
    required String component,
    required String operationId,
    required String source,
    required String reason,
    required AutoStartRuntimeState previousState,
  }) {
    _logger('$component.start.skipped', {
      'operation_id': operationId,
      'reason': reason,
      'source': source,
      'previous_state': previousState.name,
      'target_state': AutoStartRuntimeState.running.name,
      'result': 'skipped',
      'retry_count': 0,
    });
  }

  void _logTransitionResult({
    required String component,
    required String operationId,
    required String source,
    required AutoStartRuntimeState previousState,
    required AutoStartTransitionResult result,
  }) {
    final metadata = <String, Object?>{
      'operation_id': operationId,
      'reason': result.reason.isEmpty ? 'state_transition' : result.reason,
      'source': source,
      'duration_ms': result.duration.inMilliseconds,
      'previous_state': previousState.name,
      'target_state': AutoStartRuntimeState.running.name,
      'result': result.isReady ? 'ready' : 'failed',
      'retry_count': result.retryCount,
    };
    if (result.error.isNotEmpty) metadata['error'] = result.error;
    if (result.timeout > Duration.zero) {
      metadata['timeout_ms'] = result.timeout.inMilliseconds;
    }

    if (result.actualStart) {
      _logger('$component.start.started', metadata);
    }
    if (result.isReady) {
      _logger('$component.start.ready', metadata);
    } else if (result.timedOut) {
      _logger('$component.start.timeout', metadata);
    } else {
      _logger('$component.start.failed', metadata);
    }
  }
}
