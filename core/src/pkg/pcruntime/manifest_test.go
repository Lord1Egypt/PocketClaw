package pcruntime

import (
	"strings"
	"testing"
)

func TestEmbeddedManifestIsValid(t *testing.T) {
	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("the shipped runtime catalog must be valid: %v", err)
	}
	if manifest.RuntimeVersion != ManifestVersion {
		t.Fatalf("catalog declares runtime_version %d, build expects %d",
			manifest.RuntimeVersion, ManifestVersion)
	}
	if len(manifest.Tools) == 0 {
		t.Fatal("the shipped catalog declares no tools")
	}
}

// Every shipped entry must be system-delivered until a bundled payload with a
// real checksum exists. A bundled entry naming a payload the APK does not carry
// would resolve unavailable on every device and mislead anyone reading it.
func TestEmbeddedManifestBundledEntriesNameARealPayload(t *testing.T) {
	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("cannot load catalog: %v", err)
	}
	for _, tool := range manifest.Tools {
		if tool.Delivery != DeliveryBundled {
			continue
		}
		if !strings.HasPrefix(tool.LibraryName, "lib") || !strings.HasSuffix(tool.LibraryName, ".so") {
			t.Fatalf("bundled tool %q names payload %q, which Android will not extract",
				tool.ToolID, tool.LibraryName)
		}
		if !isSHA256Hex(tool.SHA256) {
			t.Fatalf("bundled tool %q does not pin a checksum", tool.ToolID)
		}
	}
}

func TestManifestRejectsUnsupportedRuntimeVersion(t *testing.T) {
	_, err := ParseManifest([]byte(`{"runtime_version":99,"catalog_version":"1","tools":[]}`))
	if err == nil || !strings.Contains(err.Error(), "not supported by this build") {
		t.Fatalf("expected a version rejection, got %v", err)
	}
}

func TestManifestRejectsDuplicateCommandNames(t *testing.T) {
	manifest := &Manifest{
		RuntimeVersion: ManifestVersion,
		CatalogVersion: "test",
		Tools: []Tool{
			systemTool("first", TimeoutQuick, 1024),
			systemTool("second", TimeoutQuick, 1024),
		},
	}
	// Make both answer to the same command.
	manifest.Tools[1].CommandName = "first"

	if err := manifest.validate(); err == nil || !strings.Contains(err.Error(), "both") {
		t.Fatalf("expected duplicate command rejection, got %v", err)
	}
}

// A system binary is replaced by every OS update, so a pinned hash would be a
// guarantee PocketClaw cannot keep. The catalog must refuse to record one.
func TestManifestRejectsChecksumOnSystemTool(t *testing.T) {
	tool := systemTool("cat", TimeoutQuick, 1024)
	tool.SHA256 = strings.Repeat("a", 64)
	manifest := &Manifest{RuntimeVersion: ManifestVersion, CatalogVersion: "t", Tools: []Tool{tool}}

	if err := manifest.validate(); err == nil || !strings.Contains(err.Error(), "must not pin a sha256") {
		t.Fatalf("expected system checksum rejection, got %v", err)
	}
}

// Android's package manager only unpacks lib/<abi>/*.so. Any other name never
// reaches nativeLibraryDir, so the catalog must reject it at build time rather
// than shipping a tool that can never resolve.
func TestManifestRejectsBundledPayloadWithNonLibraryName(t *testing.T) {
	tool := Tool{
		ToolID: "jq", DisplayName: "jq", CommandName: "jq", Version: "1.7.1",
		ABI: "arm64-v8a", Delivery: DeliveryBundled, TrustedSource: "test",
		SHA256: strings.Repeat("a", 64), Capabilities: []string{"json.transform"},
		TimeoutProfile: TimeoutStandard, MaxOutputBytes: 1024,
		SecurityClass: SecurityClassBundledVerified,
		LibraryName:   "jq",
	}
	manifest := &Manifest{RuntimeVersion: ManifestVersion, CatalogVersion: "t", Tools: []Tool{tool}}

	if err := manifest.validate(); err == nil || !strings.Contains(err.Error(), "lib*.so") {
		t.Fatalf("expected library-name rejection, got %v", err)
	}
}

func TestManifestRejectsUnknownTimeoutProfile(t *testing.T) {
	tool := systemTool("cat", "forever", 1024)
	manifest := &Manifest{RuntimeVersion: ManifestVersion, CatalogVersion: "t", Tools: []Tool{tool}}

	if err := manifest.validate(); err == nil || !strings.Contains(err.Error(), "timeout_profile") {
		t.Fatalf("expected timeout profile rejection, got %v", err)
	}
}
