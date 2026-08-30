package pcruntime

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestResolveSupportedSystemTool(t *testing.T) {
	manager, binDir := newTestManager(t, newTestManifest(t, systemTool("echoer", TimeoutQuick, 4096)))
	writeScript(t, binDir, "echoer", "exit 0\n")

	resolved, err := manager.Registry().Resolve("echoer")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if !resolved.Available() {
		t.Fatalf("expected available, got %s (%s)", resolved.Availability, resolved.Diagnostics)
	}
	if resolved.Verification != VerificationPlatformOwned {
		t.Fatalf("a system binary must report platform_owned, got %s", resolved.Verification)
	}
	if resolved.ExecutablePath != filepath.Join(binDir, "echoer") {
		t.Fatalf("unexpected path %q", resolved.ExecutablePath)
	}
}

func TestResolveUnsupportedToolFailsSafely(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("echoer", TimeoutQuick, 4096)))

	_, err := manager.Registry().Resolve("nmap")
	if !errors.Is(err, ErrNotInCatalog) {
		t.Fatalf("expected ErrNotInCatalog, got %v", err)
	}
}

func TestResolveMissingSystemToolReportsNotInstalled(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("dig", TimeoutQuick, 4096)))

	resolved, err := manager.Registry().Resolve("dig")
	if err != nil {
		t.Fatalf("resolve returned an error for a catalog tool: %v", err)
	}
	if resolved.Available() {
		t.Fatal("a tool the platform does not ship must not resolve as available")
	}
	if resolved.UnavailableReason != ReasonNotInstalled {
		t.Fatalf("expected not_installed, got %s", resolved.UnavailableReason)
	}
	if !strings.Contains(resolved.Diagnostics, "not provided by this platform image") {
		t.Fatalf("diagnostics should say why: %q", resolved.Diagnostics)
	}
}

// bundledFixture stages a host-architecture ELF as a bundled payload.
func bundledFixture(t *testing.T, abi string, corrupt bool) (*Manager, Tool, string) {
	t.Helper()
	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	workspace := filepath.Join(root, "workspace")
	for _, dir := range []string{libDir, workspace} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", dir, err)
		}
	}

	// A real ELF is required: verification opens it as one. The test binary
	// itself is the most convenient ELF that is guaranteed to exist.
	self, err := os.Executable()
	if err != nil {
		t.Skipf("cannot locate an ELF fixture on this host: %v", err)
	}
	payload := filepath.Join(libDir, "libpocketclaw-fixture.so")
	if err := copyFileMode(self, payload, 0o755); err != nil {
		t.Fatalf("cannot stage payload: %v", err)
	}

	sum, err := fileSHA256(payload)
	if err != nil {
		t.Fatalf("cannot hash payload: %v", err)
	}
	if corrupt {
		sum = strings.Repeat("0", 64)
	}

	tool := Tool{
		ToolID: "fixture", DisplayName: "fixture", CommandName: "fixture",
		Version: "1.0.0", ABI: abi, Delivery: DeliveryBundled,
		TrustedSource: "test", SHA256: sum,
		Capabilities: []string{"test.execute"}, TimeoutProfile: TimeoutQuick,
		MaxOutputBytes: 4096, SecurityClass: SecurityClassBundledVerified,
		LibraryName: "libpocketclaw-fixture.so",
	}
	manifest := newTestManifest(t, tool)
	registry, err := NewRegistryWith(manifest, &Paths{
		LibDir: libDir, MetadataDir: filepath.Join(root, "runtime"), Workspace: workspace,
	})
	if err != nil {
		t.Fatalf("cannot build registry: %v", err)
	}
	manager, err := NewManagerWith(registry, workspace)
	if err != nil {
		t.Fatalf("cannot build manager: %v", err)
	}
	return manager, tool, payload
}

func TestResolveBundledPayloadVerifiesChecksum(t *testing.T) {
	manager, _, _ := bundledFixture(t, hostABI(t), false)

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if !resolved.Available() {
		t.Fatalf("expected available, got %s (%s)", resolved.Availability, resolved.Diagnostics)
	}
	if resolved.Verification != VerificationSHA256Match {
		t.Fatalf("expected sha256_match, got %s", resolved.Verification)
	}
}

func TestResolveBundledPayloadRejectsChecksumMismatch(t *testing.T) {
	manager, _, _ := bundledFixture(t, hostABI(t), true)

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved.Available() {
		t.Fatal("a payload that fails its checksum must never be activated")
	}
	if resolved.UnavailableReason != ReasonChecksumMismatch {
		t.Fatalf("expected checksum_mismatch, got %s", resolved.UnavailableReason)
	}
	if resolved.ExecutablePath != "" {
		t.Fatal("an unverified payload must not expose an executable path")
	}
}

// A payload built for the wrong machine fails at exec with a bare ENOEXEC.
// Rejecting it at verification turns that into a diagnosis.
func TestResolveBundledPayloadRejectsWrongABI(t *testing.T) {
	wrong := "arm64-v8a"
	if hostABI(t) == "arm64-v8a" {
		wrong = "x86_64"
	}
	manager, _, _ := bundledFixture(t, wrong, false)

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved.Available() {
		t.Fatal("a payload for another architecture must not resolve as available")
	}
	if resolved.UnavailableReason != ReasonABIMismatch {
		t.Fatalf("expected abi_mismatch, got %s", resolved.UnavailableReason)
	}
	if resolved.Verification != VerificationABIRejected {
		t.Fatalf("expected abi_rejected, got %s", resolved.Verification)
	}
}

func TestResolveMissingBundledPayloadNamesThePackagingContract(t *testing.T) {
	manager, _, payload := bundledFixture(t, hostABI(t), false)
	if err := os.Remove(payload); err != nil {
		t.Fatalf("cannot remove payload: %v", err)
	}

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved.Available() {
		t.Fatal("a payload the installer did not unpack must not resolve as available")
	}
	if !strings.Contains(resolved.Diagnostics, "extractNativeLibs") {
		t.Fatalf("diagnostics must point at the packaging contract: %q", resolved.Diagnostics)
	}
}

// Concurrent callers must share one verification pass rather than each hashing
// the payload, and must all observe the same result.
func TestConcurrentResolutionIsDeduplicated(t *testing.T) {
	manager, _, _ := bundledFixture(t, hostABI(t), false)

	const callers = 16
	results := make([]*ResolvedTool, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func(index int) {
			defer wg.Done()
			resolved, err := manager.Registry().Resolve("fixture")
			if err != nil {
				t.Errorf("resolve failed: %v", err)
				return
			}
			results[index] = resolved
		}(i)
	}
	wg.Wait()

	for i, resolved := range results {
		if resolved == nil {
			t.Fatalf("caller %d got no result", i)
		}
		if resolved != results[0] {
			t.Fatal("concurrent resolution produced more than one verification result")
		}
	}
}

func hostABI(t *testing.T) string {
	t.Helper()
	switch runtimeGOARCH() {
	case "arm64":
		return "arm64-v8a"
	case "amd64":
		return "x86_64"
	case "arm":
		return "armeabi-v7a"
	default:
		t.Skipf("no catalog ABI name for host architecture %q", runtimeGOARCH())
		return ""
	}
}
