package main

import (
	"net/http"
	"testing"
)

// PC-DEF-040, reopened. The owner reproduced Public Mode ON with an unreachable LAN
// Dashboard, fixed only by toggling Public Mode OFF then ON by hand.
//
// The cause was that nothing on this side reacted to the dashboard being claimed.
// PC-DEF-039 narrows an unclaimed dashboard to loopback whatever the user asked for,
// and the only thing that ever re-applied the preference was the Android app noticing
// its embedded WebView navigate away from /launcher-setup. Any other route to a first
// claim left desired and effective diverged with no recovery.
//
// These exercise the four cases the owner listed, against the runtime that owns the
// listener.

func newTestRuntime(t *testing.T, effectivePublic, desiredPublic bool) *launcherHTTPRuntime {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	initial, err := openLauncherListeners("", effectivePublic, "0")
	if err != nil {
		t.Fatal(err)
	}
	runtime := newLauncherHTTPRuntime(handler, "", effectivePublic, desiredPublic, initial)
	runtime.Start()
	t.Cleanup(runtime.Shutdown)
	return runtime
}

// Case B, the owner's headline: desired ON, narrowed to loopback while unclaimed,
// and LAN the moment a local owner claims it — with no toggle and no restart.
func TestClaimingTheDashboardAppliesTheRequestedPublicMode(t *testing.T) {
	runtime := newTestRuntime(t, false, true)

	if runtime.PublicMode() {
		t.Fatal("an unclaimed dashboard must start on loopback, whatever was desired")
	}

	if err := runtime.ReconcileAfterDashboardClaimed(); err != nil {
		t.Fatalf("ReconcileAfterDashboardClaimed() error = %v", err)
	}

	if !runtime.PublicMode() {
		t.Fatal("the requested Public Mode must be applied once the dashboard has an " +
			"owner, without a manual OFF/ON toggle")
	}
}

// Case A: the user never asked for LAN, so claiming the dashboard must not expose it.
// This is the direction that matters most — a reconciliation that widens on its own
// would be worse than the defect.
func TestClaimingTheDashboardDoesNotExposeAPrivateOne(t *testing.T) {
	runtime := newTestRuntime(t, false, false)

	if err := runtime.ReconcileAfterDashboardClaimed(); err != nil {
		t.Fatalf("ReconcileAfterDashboardClaimed() error = %v", err)
	}

	if runtime.PublicMode() {
		t.Fatal("Public Mode was never requested; claiming the dashboard must not " +
			"expose it to the LAN")
	}
}

// Idempotent, so it is safe on every claim and on a password change.
func TestReconcilingAnAlreadyPublicDashboardIsANoOp(t *testing.T) {
	runtime := newTestRuntime(t, true, true)

	for i := 0; i < 3; i++ {
		if err := runtime.ReconcileAfterDashboardClaimed(); err != nil {
			t.Fatalf("ReconcileAfterDashboardClaimed() error = %v", err)
		}
		if !runtime.PublicMode() {
			t.Fatal("a public dashboard must stay public")
		}
	}
}

// An explicit change is a change of intent. Otherwise turning Public Mode off and
// then claiming the dashboard would re-widen it from a desire the user has retracted.
func TestTurningPublicModeOffRetractsTheDesire(t *testing.T) {
	runtime := newTestRuntime(t, true, true)

	if err := runtime.ApplyPublicMode(false); err != nil {
		t.Fatalf("ApplyPublicMode(false) error = %v", err)
	}
	if runtime.PublicMode() {
		t.Fatal("Public Mode should be off")
	}

	if err := runtime.ReconcileAfterDashboardClaimed(); err != nil {
		t.Fatalf("ReconcileAfterDashboardClaimed() error = %v", err)
	}

	if runtime.PublicMode() {
		t.Fatal("a retracted Public Mode must not be resurrected by a later claim")
	}
}

// Case C: the state stays coherent after an explicit round trip, which is what a
// service restart reduces to from this side.
func TestDesiredAndEffectiveStayCoherentAcrossExplicitChanges(t *testing.T) {
	runtime := newTestRuntime(t, false, true)

	// Claim: loopback -> LAN.
	if err := runtime.ReconcileAfterDashboardClaimed(); err != nil {
		t.Fatal(err)
	}
	if !runtime.PublicMode() {
		t.Fatal("want public after the claim")
	}

	// User turns it off, then on again by hand — the old workaround must still work.
	if err := runtime.ApplyPublicMode(false); err != nil {
		t.Fatal(err)
	}
	if runtime.PublicMode() {
		t.Fatal("want loopback after an explicit off")
	}
	if err := runtime.ApplyPublicMode(true); err != nil {
		t.Fatal(err)
	}
	if !runtime.PublicMode() {
		t.Fatal("want public after an explicit on")
	}

	// And a further claim changes nothing.
	if err := runtime.ReconcileAfterDashboardClaimed(); err != nil {
		t.Fatal(err)
	}
	if !runtime.PublicMode() {
		t.Fatal("want public still")
	}
}

// PC-DEF-039 is the rule this must not undo: an unclaimed dashboard is loopback even
// when Public Mode is desired. Asserted here beside the reconciliation so the two are
// read together.
func TestAnUnclaimedDashboardIsNarrowedEvenWhenPublicIsDesired(t *testing.T) {
	effective, narrowed := effectiveLauncherExposure(true, false)
	if effective {
		t.Fatal("an unclaimed dashboard must never be exposed beyond loopback")
	}
	if !narrowed {
		t.Fatal("the narrowing must be reported, so the user can be told why")
	}

	effective, narrowed = effectiveLauncherExposure(true, true)
	if !effective || narrowed {
		t.Fatal("a claimed dashboard with Public Mode desired must bind to the LAN")
	}

	effective, narrowed = effectiveLauncherExposure(false, true)
	if effective || narrowed {
		t.Fatal("Public Mode off means loopback, and that is not a narrowing")
	}
}
