package coresource

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The stub compiled when the build tag is absent carries this sentence. Its
// presence in the shipped binary means the Android Core has no WhatsApp
// transport at all — the channel would fail to start with an error nobody
// could act on from a phone.
const whatsAppNativeStubMarker = "whatsapp native not compiled in"

// A log line that exists only in the real transport, so its presence is
// positive evidence rather than merely the absence of the stub.
const whatsAppNativeRealMarker = "WhatsApp pairing code ready"

// TestAndroidBuildCompilesTheRealWhatsAppTransport pins the build tag.
//
// The transport is behind //go:build whatsapp_native, and for its whole life
// the Android target built with -tags stdjson alone — so what shipped was the
// stub. Dropping the tag again would not break any build; it would silently
// ship a channel that cannot start.
func TestAndroidBuildCompilesTheRealWhatsAppTransport(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}

	makefile, err := os.ReadFile(filepath.Join(root, "core", "src", "Makefile"))
	if err != nil {
		t.Fatalf("cannot read the Core Makefile: %v", err)
	}
	if !strings.Contains(string(makefile), "ANDROID_BUILD_TAGS?=stdjson,whatsapp_native") {
		t.Error("the Android Core build no longer sets the whatsapp_native tag")
	}
	for _, line := range strings.Split(string(makefile), "\n") {
		if strings.Contains(line, "GOOS=android") && strings.Contains(line, "-tags stdjson ") {
			t.Errorf("an Android target still hardcodes -tags stdjson: %s", strings.TrimSpace(line))
		}
	}
}

// TestStagedCoreCarriesTheRealWhatsAppTransport checks the binary Gradle
// actually packages, not just the recipe that should have produced it.
func TestStagedCoreCarriesTheRealWhatsAppTransport(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}
	core, err := os.ReadFile(filepath.Join(root, "android", "app", "src", "main",
		"jniLibs", "arm64-v8a", "libpicoclaw.so"))
	if err != nil {
		t.Skipf("no staged Core binary to check: %v", err)
	}

	if bytes.Contains(core, []byte(whatsAppNativeStubMarker)) {
		t.Error("the staged Core carries the WhatsApp stub; it was built without -tags whatsapp_native")
	}
	if !bytes.Contains(core, []byte(whatsAppNativeRealMarker)) {
		t.Error("the staged Core does not carry the real WhatsApp transport")
	}
}

// TestAndroidHostOwnsTheWhatsAppSessionPath pins the one link the Kotlin unit
// tests cannot reach without Robolectric: that the service derives the WhatsApp
// directories from noBackupFilesDir rather than from the workspace.
//
// The workspace on Android is public external storage and is also the root the
// agent's file tools are restricted to, so deriving the session database from
// it would put the account's identity keys inside the agent's own sandbox.
func TestAndroidHostOwnsTheWhatsAppSessionPath(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}
	service, err := os.ReadFile(filepath.Join(root, "android", "app", "src", "main",
		"kotlin", "com", "lord1egypt", "pocketclaw", "service", "PicoClawService.kt"))
	if err != nil {
		t.Fatalf("cannot read PicoClawService.kt: %v", err)
	}
	text := string(service)

	if !strings.Contains(text, "WhatsAppAgentStorage.environment(context.noBackupFilesDir)") {
		t.Error("the WhatsApp directories are no longer derived from noBackupFilesDir")
	}
	if strings.Contains(text, "WhatsAppAgentStorage.environment(workspace") {
		t.Error("the WhatsApp directories are derived from the workspace, which the agent can read")
	}
}
