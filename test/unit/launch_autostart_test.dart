import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';

/// The auto-start contract, asserted without a device.
///
/// Auto-start is deliberately a one-shot app-launch decision. Everything the
/// physical regressions were about — resurrection after a manual stop, repeated
/// starts from lifecycle callbacks, starting something that is already up —
/// reduces to this function returning something other than `start`.
void main() {
  LaunchAutoStartDecision decide({
    bool alreadyEvaluated = false,
    bool preferenceEnabled = true,
    ServiceStatus status = ServiceStatus.stopped,
  }) => ServiceManager.decideLaunchAutoStart(
    alreadyEvaluated: alreadyEvaluated,
    preferenceEnabled: preferenceEnabled,
    status: status,
  );

  test('fresh launch with the preference on starts the service', () {
    expect(decide(), LaunchAutoStartDecision.start);
  });

  test('preference off starts nothing', () {
    expect(
      decide(preferenceEnabled: false),
      LaunchAutoStartDecision.preferenceOff,
    );
  });

  test('a running or starting service is never restarted', () {
    expect(
      decide(status: ServiceStatus.running),
      LaunchAutoStartDecision.alreadyActive,
    );
    expect(
      decide(status: ServiceStatus.starting),
      LaunchAutoStartDecision.alreadyActive,
    );
  });

  test('repeated evaluation in one process starts only once', () {
    expect(decide(), LaunchAutoStartDecision.start);
    expect(
      decide(alreadyEvaluated: true),
      LaunchAutoStartDecision.alreadyEvaluated,
    );
  });

  test('a manual stop is not undone by a later evaluation', () {
    // The service is stopped and the preference is still on, which is exactly
    // the state after a manual Stop. The one-shot latch is what keeps it
    // stopped, so resume or any repeated callback must not resurrect it.
    expect(
      decide(alreadyEvaluated: true, status: ServiceStatus.stopped),
      LaunchAutoStartDecision.alreadyEvaluated,
    );
  });

  test('the latch outranks every other reason', () {
    expect(
      decide(
        alreadyEvaluated: true,
        preferenceEnabled: false,
        status: ServiceStatus.running,
      ),
      LaunchAutoStartDecision.alreadyEvaluated,
    );
  });

  test('a new app process with the preference on may start again', () {
    // A fresh process resets the latch, which is what "start automatically on
    // the next legitimate app launch" means.
    expect(decide(alreadyEvaluated: false), LaunchAutoStartDecision.start);
  });
}
