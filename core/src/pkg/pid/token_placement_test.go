package pid

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// The gateway bearer credential must not be written into the PID record when a
// private sink is configured.
//
// The record lives in PICOCLAW_HOME, which on Android is a user-visible
// directory on shared external storage: the 0600 it is written with is
// synthesised by the filesystem rather than enforced, so any app holding
// storage access can read whatever the file contains. Android does not isolate
// loopback sockets between apps either, so a readable credential is one file
// read away from authenticating to /reload and detailed /health.
func TestGatewayTokenIsNotWrittenIntoTheSharedPidRecord(t *testing.T) {
	home := t.TempDir()
	private := t.TempDir()
	tokenPath := filepath.Join(private, "gateway_auth")
	t.Setenv(config.EnvGatewayTokenFile, tokenPath)

	data, err := WritePidFile(home, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}
	defer RemovePidFile(home)

	if strings.TrimSpace(data.Token) == "" {
		t.Fatal("the caller still needs the token in memory to configure the health server")
	}

	raw, err := os.ReadFile(filepath.Join(home, pidFileName))
	if err != nil {
		t.Fatalf("read pid record: %v", err)
	}

	// Checked as text as well as as a field: a future field name change must
	// not let the value itself slip back into the shared file.
	if strings.Contains(string(raw), data.Token) {
		t.Fatalf("the pid record contains the bearer token:\n%s", raw)
	}
	var record map[string]any
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatalf("pid record is not valid JSON: %v", err)
	}
	if _, present := record["token"]; present {
		t.Fatalf("the pid record still declares a token field:\n%s", raw)
	}

	// Discovery metadata is the record's actual job and must survive.
	for _, field := range []string{"pid", "version", "port", "host"} {
		if _, present := record[field]; !present {
			t.Errorf("the pid record lost its %q field:\n%s", field, raw)
		}
	}
}

// The private sink is the authoritative source, and it is owner-only.
func TestGatewayTokenGoesToThePrivateSink(t *testing.T) {
	home := t.TempDir()
	tokenPath := filepath.Join(t.TempDir(), "nested", "gateway_auth")
	t.Setenv(config.EnvGatewayTokenFile, tokenPath)

	data, err := WritePidFile(home, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}
	defer RemovePidFile(home)

	stored, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("read token sink: %v", err)
	}
	if strings.TrimSpace(string(stored)) != data.Token {
		t.Fatal("the token sink does not hold the credential the gateway is using")
	}
	// The file holds the credential and nothing else, so there is no adjacent
	// field for a reader to pick up by accident.
	if strings.ContainsAny(string(stored), "{}\n") {
		t.Fatalf("the token sink should hold a bare token, got %d bytes of structure", len(stored))
	}

	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("stat token sink: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("token sink mode = %04o, want 0600", perm)
	}

	// Shutdown takes the credential with it rather than leaving it for the
	// next process to find.
	RemovePidFile(home)
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Error("the token sink outlived the gateway that owned it")
	}
}

// Without a sink the behaviour is exactly what it was. PICOCLAW_HOME is already
// private on a desktop or a server, and changing that would be a regression for
// every non-Android deployment.
func TestWithoutASinkTheTokenStaysInTheRecord(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvGatewayTokenFile, "")

	data, err := WritePidFile(home, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}
	defer RemovePidFile(home)

	raw, err := os.ReadFile(filepath.Join(home, pidFileName))
	if err != nil {
		t.Fatalf("read pid record: %v", err)
	}
	if !strings.Contains(string(raw), data.Token) {
		t.Fatal("the token must remain in the record when no private sink is configured")
	}
}

// A new gateway must stand up on its own credential, not inherit one an older
// build left in shared storage.
func TestANewStartDoesNotReuseALegacySharedToken(t *testing.T) {
	home := t.TempDir()
	tokenPath := filepath.Join(t.TempDir(), "gateway_auth")

	legacy := `{"pid":999999,"token":"legacy-token-from-an-older-build","version":"v0","port":18790,"host":"127.0.0.1"}`
	if err := os.WriteFile(filepath.Join(home, pidFileName), []byte(legacy), 0o600); err != nil {
		t.Fatalf("seed legacy record: %v", err)
	}

	t.Setenv(config.EnvGatewayTokenFile, tokenPath)
	data, err := WritePidFile(home, "127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}
	defer RemovePidFile(home)

	if data.Token == "legacy-token-from-an-older-build" {
		t.Fatal("the new gateway adopted a credential from the old shared record")
	}
	raw, err := os.ReadFile(filepath.Join(home, pidFileName))
	if err != nil {
		t.Fatalf("read pid record: %v", err)
	}
	if strings.Contains(string(raw), "legacy-token-from-an-older-build") {
		t.Fatalf("the legacy credential survived in the shared record:\n%s", raw)
	}
}
