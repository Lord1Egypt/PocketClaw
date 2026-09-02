package whatsapp

import (
	"os"
	"strings"
	"testing"
)

// TestPairingCodeNeverReachesStdoutOrLogs guards the property that made the
// upstream transport unsafe to ship on Android.
//
// Upstream printed the pairing QR with qrterminal to os.Stdout. PocketClaw
// captures Core's stdout into a persisted Logs screen, and a WhatsApp pairing
// QR is credential material: whoever scans it first links their device to the
// user's account. This runs untagged, so the guard holds for every build rather
// than only the one that compiles the transport in.
func TestPairingCodeNeverReachesStdoutOrLogs(t *testing.T) {
	source, err := os.ReadFile("whatsapp_native.go")
	if err != nil {
		t.Fatalf("cannot read the transport source: %v", err)
	}
	text := string(source)

	for _, banned := range []string{
		"qrterminal",
		"os.Stdout",
		"os.Stderr",
		"fmt.Print",
		"println(",
	} {
		if strings.Contains(text, banned) {
			t.Errorf("the transport references %q; the pairing code must not reach stdout or the logs", banned)
		}
	}

	// The code itself is published to the pairing store and nowhere else.
	// Anything that put evt.Code into a log line would defeat the whole design.
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "evt.Code") {
			continue
		}
		if !strings.Contains(line, "PublishQR") {
			t.Errorf("evt.Code is used outside PublishQR: %s", strings.TrimSpace(line))
		}
	}

	if !strings.Contains(text, "c.pairing.PublishQR(evt.Code)") {
		t.Error("the pairing code is no longer published to the pairing store")
	}
}

// TestSessionStorePathIsNotTakenFromConfigDirectly keeps the store resolution
// going through pkg/whatsapp/session, which is what lets the Android host
// override a configured path rather than merge with it.
func TestSessionStorePathIsNotTakenFromConfigDirectly(t *testing.T) {
	source, err := os.ReadFile("init.go")
	if err != nil {
		t.Fatalf("cannot read init.go: %v", err)
	}
	text := string(source)

	if !strings.Contains(text, "session.Resolve(") {
		t.Error("the session store path no longer goes through session.Resolve")
	}
	// A direct read of c.SessionStorePath would reintroduce the path the agent
	// can write into the config file.
	if strings.Contains(text, "storePath := c.SessionStorePath") {
		t.Error("init.go reads SessionStorePath directly, bypassing the host override")
	}
}
