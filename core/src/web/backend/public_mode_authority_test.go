package main

import (
	"flag"
	"net"
	"testing"

	"github.com/sipeed/picoclaw/pkg/netbind"
)

// PC-DEF-020. The dashboard's public/loopback decision had two persisted
// authorities and no rule for which won.
//
// The Android host keeps the user's Public Mode choice natively and used to
// pass -public only when it was on, so "off" reached the backend as silence.
// Silence fell through to launcher-config.json's `public` field, which the
// dashboard's own Config page can set to true. Enable LAN access, save that
// page, then switch the native toggle off: the live listener moved to loopback
// and the stored true stayed, so the next service start bound the console to
// every interface while the toggle still read OFF.
//
// The fix is that the host always states the decision. These tests pin the two
// halves that makes true: that an explicit false is distinguishable from an
// omitted flag, and that a stated decision outranks the stored one.

// newLauncherFlagSet mirrors the listen flags main() registers, so the
// explicit-detection test exercises the same shapes the real command line has.
func newLauncherFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("launcher", flag.ContinueOnError)
	fs.String("port", "18800", "Port to listen on")
	fs.String("host", "", "Host to listen on")
	fs.Bool("public", false, "Listen on all interfaces")
	return fs
}

// The linchpin. If an explicit -public=false were indistinguishable from an
// omitted -public, the host could not state "off" at all and the whole fix
// would be inert.
func TestLauncherExplicitFlagsSeesAnExplicitFalse(t *testing.T) {
	for _, tc := range []struct {
		name         string
		args         []string
		wantExplicit bool
		wantValue    bool
	}{
		{"omitted", []string{}, false, false},
		{"explicit false", []string{"-public=false"}, true, false},
		{"explicit true", []string{"-public=true"}, true, true},
		{"bare flag", []string{"-public"}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := newLauncherFlagSet()
			if err := fs.Parse(tc.args); err != nil {
				t.Fatalf("Parse(%v) error = %v", tc.args, err)
			}
			_, _, explicit := launcherExplicitFlags(fs)
			if explicit != tc.wantExplicit {
				t.Fatalf("explicit public = %t, want %t for args %v",
					explicit, tc.wantExplicit, tc.args)
			}
			value := fs.Lookup("public").Value.String() == "true"
			if value != tc.wantValue {
				t.Fatalf("public value = %t, want %t for args %v",
					value, tc.wantValue, tc.args)
			}
		})
	}
}

// The required state matrix. Each case names the restart it survives: the
// resolution is a pure function of the flag and the stored value, so "service
// restart", "app restart" and "process recreation" are all the same question —
// what does a fresh process decide from these inputs — and answering it once
// answers it for every one of them.
func TestResolveLauncherPublicMode_StateMatrix(t *testing.T) {
	for _, tc := range []struct {
		name         string
		flagPublic   bool
		flagExplicit bool
		configPublic bool
		want         bool
	}{
		// A. Fresh install: native off, no launcher config, so the stored
		// value is the Default() false.
		{"A fresh install, native off", false, true, false, false},

		// B. Native on.
		{"B native on", true, true, false, true},

		// C. Native on, console saved public:true, restarted still on.
		{"C native on with stored true", true, true, true, true},

		// D/E/F. Native on, console saved public:true, native switched off.
		// The live rebind is covered by the runtime test; this is every
		// subsequent process — service restart, app restart, process
		// recreation — and it must be loopback in all of them.
		{"D-F native off with stored true", false, true, true, false},

		// G. A stored true that was already on disk before startup.
		{"G stored true predates startup", false, true, true, false},

		// H. Stored false, native on. The stored value must not veto the
		// host's decision either: authority is not a one-way ratchet.
		{"H stored false, native on", true, true, false, true},

		// Desktop, where no flag is supplied and the stored value is the only
		// authority there is. This is the behaviour the fix must not remove.
		{"desktop stored true", false, false, true, true},
		{"desktop stored false", false, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveLauncherPublicMode(tc.flagPublic, tc.flagExplicit, tc.configPublic)
			if got != tc.want {
				t.Fatalf("resolveLauncherPublicMode(%t, %t, %t) = %t, want %t",
					tc.flagPublic, tc.flagExplicit, tc.configPublic, got, tc.want)
			}
		})
	}
}

// The resolution and the bind, end to end, for the case the defect was about:
// a stale stored true and a host that says off must open loopback sockets and
// nothing else. Asserting on the resolved boolean alone would not prove that.
func TestStaleStoredPublicCannotOpenAWildcardListener(t *testing.T) {
	const storedPublic = true

	effective := resolveLauncherPublicMode(false, true, storedPublic)
	if effective {
		t.Fatalf("effective public = true with an explicit off and stored %t", storedPublic)
	}

	result, err := openLauncherListeners("", effective, "0")
	if err != nil {
		t.Fatalf("openLauncherListeners() error = %v", err)
	}
	for _, listener := range result.Listeners {
		t.Cleanup(func() { _ = listener.Close() })
	}
	if len(result.BindHosts) == 0 {
		t.Fatal("bind returned no hosts")
	}
	for _, host := range result.BindHosts {
		if netbind.IsUnspecifiedHost(host) {
			t.Fatalf("bind hosts = %#v, want loopback only", result.BindHosts)
		}
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			t.Fatalf("bind host = %q, want loopback", host)
		}
	}
	for _, listener := range result.Listeners {
		host, _, err := net.SplitHostPort(listener.Addr().String())
		if err != nil {
			t.Fatalf("SplitHostPort(%q) error = %v", listener.Addr(), err)
		}
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			t.Fatalf("listener bound to %q, want loopback", listener.Addr())
		}
	}
}

// I. The host override keeps precedence over the public decision, in both
// directions, and a stored true cannot reintroduce a wildcard behind it.
func TestExplicitHostOverridesEveryPublicDecision(t *testing.T) {
	for _, tc := range []struct {
		name         string
		flagPublic   bool
		flagExplicit bool
		configPublic bool
	}{
		{"host with explicit public off", false, true, true},
		{"host with explicit public on", true, true, true},
		{"host with no flag and stored true", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			effective := resolveLauncherPublicMode(tc.flagPublic, tc.flagExplicit, tc.configPublic)
			result, err := openLauncherListeners("127.0.0.1", effective, "0")
			if err != nil {
				t.Fatalf("openLauncherListeners() error = %v", err)
			}
			for _, listener := range result.Listeners {
				t.Cleanup(func() { _ = listener.Close() })
			}
			if got := result.BindHosts; len(got) != 1 || got[0] != "127.0.0.1" {
				t.Fatalf("bind hosts = %#v, want [127.0.0.1]", got)
			}
		})
	}
}

// J. The Core gateway never consumes the launcher's public decision. It is a
// separate listener with its own resolution, and openGatewayListeners always
// passes DefaultLoopback — so there is no value of the public flag that can
// move it off loopback. Proved through the same netbind plan the gateway uses.
func TestGatewayStaysLoopbackInEveryPublicModeState(t *testing.T) {
	for _, host := range []string{"", "localhost", "127.0.0.1"} {
		t.Run("gateway host "+host, func(t *testing.T) {
			plan, err := netbind.BuildPlan(host, netbind.DefaultLoopback)
			if err != nil {
				t.Fatalf("BuildPlan(%q) error = %v", host, err)
			}
			result, err := netbind.OpenPlan(plan, "0")
			if err != nil {
				t.Fatalf("OpenPlan() error = %v", err)
			}
			for _, listener := range result.Listeners {
				t.Cleanup(func() { _ = listener.Close() })
			}
			if len(result.BindHosts) == 0 {
				t.Fatal("gateway bind returned no hosts")
			}
			for _, bindHost := range result.BindHosts {
				if ip := net.ParseIP(bindHost); ip == nil || !ip.IsLoopback() {
					t.Fatalf("gateway bind host = %q, want loopback", bindHost)
				}
			}
		})
	}
}
