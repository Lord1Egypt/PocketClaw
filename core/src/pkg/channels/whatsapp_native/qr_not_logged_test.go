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

	// Both pairing credentials — the QR payload and the companion linking code
	// — are published to the pairing store and go nowhere else. They are named
	// consistently so this guard can cover both: anything that put either into
	// a log line would defeat the whole design.
	credentials := []string{"evt.Code", "linkCode"}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if !strings.Contains(line, "logger.") {
			continue
		}
		for _, credential := range credentials {
			if strings.Contains(line, credential) {
				t.Errorf("a log line references the pairing credential %s: %s", credential, trimmed)
			}
		}
	}

	// evt.Code may only ever be handed to the pairing store.
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(line, "evt.Code") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if !strings.Contains(line, "PublishPairing") {
			t.Errorf("evt.Code is used outside PublishPairing: %s", trimmed)
		}
	}

	if !strings.Contains(text, "c.pairing.PublishPairing(evt.Code, linkCode)") {
		t.Error("the pairing credentials are no longer published to the pairing store")
	}
	if !strings.Contains(text, "return linkCode") {
		t.Error("the companion pairing code is no longer returned to the pairing store")
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
