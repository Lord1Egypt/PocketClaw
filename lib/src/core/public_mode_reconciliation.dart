/// Whether the host should re-apply Public Mode after first dashboard setup.
///
/// PC-DEF-040. PC-DEF-039 refuses to expose an unclaimed dashboard beyond
/// loopback, which leaves a real gap: a user whose Public Mode preference is
/// ON completes setup locally and is still on loopback, with nothing in the UI
/// saying why. The preference was never wrong and was never cleared, so the
/// product must apply it rather than make the user find a toggle to cycle.
///
/// Reconciliation is the Android host's job, not the auth handler's: rebinding
/// from inside POST /api/auth/setup would tear down the listener serving that
/// very response. The host re-applies through the existing authenticated
/// loopback network-mode bridge, and the bridge token never leaves native code.
enum PublicModeReconciliation {
  /// Setup has not produced an owner yet. Nothing to reconcile.
  dashboardNotInitialized,

  /// The user does not want Public Mode. Loopback is the correct final state.
  notRequested,

  /// Already exposed; re-applying would rebind the listener for no reason.
  alreadyApplied,

  /// Desired ON, dashboard now owned, not yet exposed: re-apply.
  reapply,
}

/// Pure decision, so the contract is provable without a device or a listener.
///
/// [dashboardInitialized] must come from a real `/api/auth/status` read at a
/// host lifecycle transition -- never from a poll, and never from the page.
PublicModeReconciliation resolvePublicModeReconciliation({
  required bool dashboardInitialized,
  required bool desiredPublic,
  required bool alreadyPublic,
}) {
  // Order matters. A failed or abandoned setup leaves the dashboard unowned,
  // and that case must be answered before the preference is even consulted --
  // otherwise a failed setup on a Public-Mode device would re-expose an
  // unclaimed dashboard, which is precisely what PC-DEF-039 forbids.
  if (!dashboardInitialized) return PublicModeReconciliation.dashboardNotInitialized;
  if (!desiredPublic) return PublicModeReconciliation.notRequested;
  if (alreadyPublic) return PublicModeReconciliation.alreadyApplied;
  return PublicModeReconciliation.reapply;
}
