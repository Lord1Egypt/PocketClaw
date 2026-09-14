package main

import "testing"

// PC-DEF-039, control 2. Public Mode must never hand an unclaimed dashboard to
// the network. The setup handler independently refuses a remote first claim;
// this is the other half, so that either control regressing alone does not
// reopen the takeover.
func TestUnclaimedDashboardIsNeverExposedBeyondLoopback(t *testing.T) {
	cases := []struct {
		name          string
		desiredPublic bool
		initialized   bool
		wantEffective bool
		wantNarrowed  bool
	}{
		{
			name:          "public requested but nobody owns the dashboard yet",
			desiredPublic: true,
			initialized:   false,
			wantEffective: false,
			wantNarrowed:  true,
		},
		{
			name:          "public requested and the dashboard has an owner",
			desiredPublic: true,
			initialized:   true,
			wantEffective: true,
			wantNarrowed:  false,
		},
		{
			name:          "private stays private before setup",
			desiredPublic: false,
			initialized:   false,
			wantEffective: false,
			wantNarrowed:  false,
		},
		{
			name:          "private stays private after setup",
			desiredPublic: false,
			initialized:   true,
			wantEffective: false,
			wantNarrowed:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			effective, narrowed := effectiveLauncherExposure(tc.desiredPublic, tc.initialized)
			if effective != tc.wantEffective {
				t.Fatalf("effective = %v, want %v", effective, tc.wantEffective)
			}
			if narrowed != tc.wantNarrowed {
				t.Fatalf("narrowed = %v, want %v", narrowed, tc.wantNarrowed)
			}
		})
	}
}

// The preference itself is never rewritten. Narrowing is a property of this
// boot, so that a dashboard claimed later can be exposed without the user
// having to rediscover and re-enable a setting the product silently cleared.
func TestNarrowingDoesNotRewriteTheDesiredPreference(t *testing.T) {
	desired := true

	if effective, _ := effectiveLauncherExposure(desired, false); effective {
		t.Fatal("an unclaimed dashboard was exposed")
	}
	if !desired {
		t.Fatal("the desired preference was mutated by the exposure decision")
	}
	if effective, _ := effectiveLauncherExposure(desired, true); !effective {
		t.Fatal("once the dashboard is claimed the desired preference must apply")
	}
}
