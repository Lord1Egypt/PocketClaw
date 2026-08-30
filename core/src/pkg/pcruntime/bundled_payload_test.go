package pcruntime

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot walks up from the test's working directory looking for the Flutter
// project marker. It returns "" when the package is being tested outside a
// checkout, where there is no APK payload to compare against.
func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "pubspec.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// A bundled tool's pinned checksum and the payload the APK actually ships are
// two records of the same thing, kept in different files. When they drift, the
// tool resolves as checksum_mismatch on every device and the failure looks like
// a corrupt install rather than a stale catalog. This is the same shape of trap
// as a stale versionCode: a green build that ships the wrong bytes.
func TestBundledPayloadsMatchTheirPinnedChecksums(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout; there is no APK payload to compare")
	}

	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("cannot load catalog: %v", err)
	}

	checked := 0
	for _, tool := range manifest.Tools {
		if tool.Delivery != DeliveryBundled {
			continue
		}
		payload := filepath.Join(
			root, "android", "app", "src", "main", "jniLibs", tool.ABI, tool.LibraryName,
		)
		if _, err := os.Stat(payload); err != nil {
			t.Fatalf("catalog declares bundled tool %q but %s is not in the APK payload: %v",
				tool.ToolID, payload, err)
		}

		sum, err := fileSHA256(payload)
		if err != nil {
			t.Fatalf("cannot hash %s: %v", payload, err)
		}
		if sum != tool.SHA256 {
			t.Fatalf(
				"catalog pins %s for %s but the packaged payload hashes to %s.\n"+
					"Rebuild with runtime/build-%s-android-arm64.sh and update "+
					"core/src/pkg/pcruntime/manifest.json.",
				tool.SHA256, tool.ToolID, sum, tool.ToolID,
			)
		}

		if err := verifyELFTarget(payload, tool.ABI); err != nil {
			t.Fatalf("packaged payload for %q is not a %s binary: %v", tool.ToolID, tool.ABI, err)
		}
		checked++
	}

	if checked == 0 {
		t.Log("no bundled tools in the catalog; nothing to verify")
	}
}
