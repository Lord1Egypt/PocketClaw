import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/public_mode_reconciliation.dart';

/// PC-DEF-040 matrix A-H. The decision is what has to be right; the rebind
/// itself is the existing, already-tested network-mode bridge.
void main() {
  PublicModeReconciliation decide({
    bool initialized = true,
    bool desired = true,
    bool already = false,
  }) => resolvePublicModeReconciliation(
    dashboardInitialized: initialized,
    desiredPublic: desired,
    alreadyPublic: already,
  );

  // A. Desired ON, not yet owned -> stay on loopback.
  test('an unclaimed dashboard is never re-exposed', () {
    expect(
      decide(initialized: false),
      PublicModeReconciliation.dashboardNotInitialized,
    );
  });

  // B. Setup succeeded, desired ON -> re-apply.
  test('a claimed dashboard with Public Mode on is re-applied', () {
    expect(decide(), PublicModeReconciliation.reapply);
  });

  // C. Setup succeeded, desired OFF -> stay loopback.
  test('Public Mode off stays loopback after setup', () {
    expect(decide(desired: false), PublicModeReconciliation.notRequested);
  });

  // D and E. A failed setup -- local, anonymous or remote -- leaves the
  // dashboard unowned, so there is nothing to reconcile and no rebind.
  test('a failed or remote-denied setup never triggers a rebind', () {
    for (final desired in [true, false]) {
      expect(
        decide(initialized: false, desired: desired),
        PublicModeReconciliation.dashboardNotInitialized,
        reason: 'an unowned dashboard must not be exposed, desired=$desired',
      );
    }
  });

  // G. Repeating it changes nothing.
  test('reconciliation is idempotent once applied', () {
    expect(decide(already: true), PublicModeReconciliation.alreadyApplied);
    expect(decide(already: true), PublicModeReconciliation.alreadyApplied);
  });

  // H. The same inputs decide the same way on a later run, so a restart
  // resolves from desired + initialized rather than from anything remembered.
  test('the decision depends only on desired and initialized state', () {
    expect(decide(initialized: true, desired: true), PublicModeReconciliation.reapply);
    expect(
      decide(initialized: false, desired: true),
      PublicModeReconciliation.dashboardNotInitialized,
    );
    expect(decide(initialized: true, desired: false), PublicModeReconciliation.notRequested);
  });

  test('PC-DEF-039 is not weakened: initialization gates everything', () {
    for (final desired in [true, false]) {
      for (final already in [true, false]) {
        expect(
          decide(initialized: false, desired: desired, already: already),
          PublicModeReconciliation.dashboardNotInitialized,
        );
      }
    }
  });
}
