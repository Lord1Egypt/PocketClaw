package pcruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helperFixture stages a bundled tool with one helper payload backing two
// logical names, the way git's transport helper actually ships.
func helperFixture(t *testing.T, corruptHelper bool) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	libDir := filepath.Join(root, "lib")
	workspace := filepath.Join(root, "workspace")
	metadata := filepath.Join(root, "runtime")
	for _, dir := range []string{libDir, workspace, metadata} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", dir, err)
		}
	}

	self, err := os.Executable()
	if err != nil {
		t.Skipf("cannot locate an ELF fixture on this host: %v", err)
	}
	main := filepath.Join(libDir, "libpocketclaw-fixture.so")
	helper := filepath.Join(libDir, "libpocketclaw-fixture-helper.so")
	for _, target := range []string{main, helper} {
		if err := copyFileMode(self, target, 0o755); err != nil {
			t.Fatalf("cannot stage payload: %v", err)
		}
	}

	mainSum, err := fileSHA256(main)
	if err != nil {
		t.Fatalf("cannot hash payload: %v", err)
	}
	helperSum := mainSum
	if corruptHelper {
		helperSum = strings.Repeat("0", 64)
	}

	tool := Tool{
		ToolID: "fixture", DisplayName: "fixture", CommandName: "fixture",
		Version: "1.0.0", ABI: hostABI(t), Delivery: DeliveryBundled,
		TrustedSource: "test", SHA256: mainSum,
		Capabilities: []string{"test.execute"}, TimeoutProfile: TimeoutQuick,
		MaxOutputBytes: 4096, SecurityClass: SecurityClassBundledVerified,
		LibraryName: "libpocketclaw-fixture.so",
		Helpers: []Helper{
			{LogicalName: "fixture-remote-http", LibraryName: "libpocketclaw-fixture-helper.so", SHA256: helperSum},
			{LogicalName: "fixture-remote-https", LibraryName: "libpocketclaw-fixture-helper.so", SHA256: helperSum},
		},
	}
	manifest := newTestManifest(t, tool)
	registry, err := NewRegistryWith(manifest, &Paths{
		LibDir: libDir, MetadataDir: metadata, Workspace: workspace,
	})
	if err != nil {
		t.Fatalf("cannot build registry: %v", err)
	}
	manager, err := NewManagerWith(registry, workspace)
	if err != nil {
		t.Fatalf("cannot build manager: %v", err)
	}
	return manager, helper
}

func TestHelperPayloadsAreVerifiedAndMapped(t *testing.T) {
	manager, helperPath := helperFixture(t, false)

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if !resolved.Available() {
		t.Fatalf("expected available, got %s (%s)", resolved.Availability, resolved.Diagnostics)
	}
	// Two logical names, one payload, exactly as git ships remote-http.
	for _, name := range []string{"fixture-remote-http", "fixture-remote-https"} {
		if resolved.HelperPaths[name] != helperPath {
			t.Fatalf("helper %q resolved to %q, want %q", name, resolved.HelperPaths[name], helperPath)
		}
	}
}

// A tool whose helper fails verification must not be reported available: git
// without a verified transport helper looks installed and then fails at the
// first https:// URL, which is a much harder failure to read.
func TestHelperChecksumMismatchMakesTheToolUnavailable(t *testing.T) {
	manager, _ := helperFixture(t, true)

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved.Available() {
		t.Fatal("a tool with an unverified helper must not resolve as available")
	}
	if resolved.UnavailableReason != ReasonChecksumMismatch {
		t.Fatalf("expected checksum_mismatch, got %s", resolved.UnavailableReason)
	}
	if resolved.ExecutablePath != "" {
		t.Fatal("an unverified tool must not expose an executable path")
	}
}

func TestMissingHelperPayloadNamesThePackagingContract(t *testing.T) {
	manager, helperPath := helperFixture(t, false)
	if err := os.Remove(helperPath); err != nil {
		t.Fatalf("cannot remove helper: %v", err)
	}

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if resolved.Available() {
		t.Fatal("a tool whose helper was not unpacked must not resolve as available")
	}
	if !strings.Contains(resolved.Diagnostics, "lib/") {
		t.Fatalf("diagnostics must point at the packaging contract: %q", resolved.Diagnostics)
	}
}

// The helper directory holds symlinks, never copies. A copied executable in
// app storage is exactly what Android refuses to run.
func TestHelperDirectoryContainsOnlySymlinksToPackagedPayloads(t *testing.T) {
	manager, helperPath := helperFixture(t, false)

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	dir, err := manager.materialiseHelpers(resolved)
	if err != nil {
		t.Fatalf("cannot materialise helpers: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read helper dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 helper links, got %d", len(entries))
	}
	for _, entry := range entries {
		info, err := os.Lstat(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("cannot stat %s: %v", entry.Name(), err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s is a real file; helper entries must be symlinks to the packaged payload",
				entry.Name())
		}
		target, err := os.Readlink(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("cannot read link %s: %v", entry.Name(), err)
		}
		if target != helperPath {
			t.Fatalf("%s points at %q, want the packaged payload %q", entry.Name(), target, helperPath)
		}
	}
}

// Re-materialising must repair a link that points somewhere stale, which is
// what happens after an app update moves nativeLibraryDir.
func TestHelperDirectoryRepairsAStaleLink(t *testing.T) {
	manager, helperPath := helperFixture(t, false)

	resolved, err := manager.Registry().Resolve("fixture")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	dir, err := manager.materialiseHelpers(resolved)
	if err != nil {
		t.Fatalf("cannot materialise helpers: %v", err)
	}

	stale := filepath.Join(dir, "fixture-remote-https")
	if err := os.Remove(stale); err != nil {
		t.Fatalf("cannot remove link: %v", err)
	}
	if err := os.Symlink("/nonexistent/old/path", stale); err != nil {
		t.Fatalf("cannot create stale link: %v", err)
	}

	if _, err := manager.materialiseHelpers(resolved); err != nil {
		t.Fatalf("re-materialising failed: %v", err)
	}
	target, err := os.Readlink(stale)
	if err != nil {
		t.Fatalf("cannot read repaired link: %v", err)
	}
	if target != helperPath {
		t.Fatalf("stale link was not repaired: points at %q", target)
	}
}

// A logical name becomes a filename in the helper directory, so a path-like
// one would place a link outside it.
func TestCatalogRejectsPathLikeHelperNames(t *testing.T) {
	for _, name := range []string{"../escape", "sub/dir", ".."} {
		tool := Tool{
			ToolID: "fixture", DisplayName: "f", CommandName: "f", Version: "1",
			ABI: "arm64-v8a", Delivery: DeliveryBundled, TrustedSource: "test",
			SHA256: strings.Repeat("a", 64), Capabilities: []string{"c"},
			TimeoutProfile: TimeoutQuick, MaxOutputBytes: 1,
			SecurityClass: SecurityClassBundledVerified,
			LibraryName:   "libfixture.so",
			Helpers: []Helper{
				{LogicalName: name, LibraryName: "libhelper.so", SHA256: strings.Repeat("b", 64)},
			},
		}
		manifest := &Manifest{RuntimeVersion: ManifestVersion, CatalogVersion: "t", Tools: []Tool{tool}}
		if err := manifest.validate(); err == nil {
			t.Fatalf("helper logical_name %q must be rejected", name)
		}
	}
}

func TestCatalogRejectsConflictingHelperChecksums(t *testing.T) {
	tool := Tool{
		ToolID: "fixture", DisplayName: "f", CommandName: "f", Version: "1",
		ABI: "arm64-v8a", Delivery: DeliveryBundled, TrustedSource: "test",
		SHA256: strings.Repeat("a", 64), Capabilities: []string{"c"},
		TimeoutProfile: TimeoutQuick, MaxOutputBytes: 1,
		SecurityClass: SecurityClassBundledVerified,
		LibraryName:   "libfixture.so",
		Helpers: []Helper{
			{LogicalName: "one", LibraryName: "libhelper.so", SHA256: strings.Repeat("b", 64)},
			{LogicalName: "two", LibraryName: "libhelper.so", SHA256: strings.Repeat("c", 64)},
		},
	}
	manifest := &Manifest{RuntimeVersion: ManifestVersion, CatalogVersion: "t", Tools: []Tool{tool}}
	if err := manifest.validate(); err == nil ||
		!strings.Contains(err.Error(), "two different checksums") {
		t.Fatalf("expected a conflicting-checksum rejection, got %v", err)
	}
}

func TestSymlinkExecutionProbeReportsAVerdict(t *testing.T) {
	manager, _ := newTestManager(t, newTestManifest(t, systemTool("present", TimeoutQuick, 4096)))

	probe := ProbeExecution(context.Background(), manager.Registry().Paths())
	switch probe.SymlinkExec {
	case WritableExecSupported, WritableExecBlocked, WritableExecInconclusive:
	default:
		t.Fatalf("symlink probe returned an unrecognised verdict %q", probe.SymlinkExec)
	}
	if probe.SymlinkExecDetail == "" {
		t.Fatal("the symlink probe must always explain its verdict")
	}
}
