package pcruntime

import (
	"bytes"
	"fmt"
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
		// Helper payloads are checked alongside the main one. A stale helper
		// checksum is the more dangerous of the two: the tool still resolves
		// far enough to look installed and then fails at its first network
		// operation.
		payloads := map[string]string{tool.LibraryName: tool.SHA256}
		for libraryName, sum := range tool.helperPayloads() {
			payloads[libraryName] = sum
		}

		for libraryName, expected := range payloads {
			payload := filepath.Join(
				root, "android", "app", "src", "main", "jniLibs", tool.ABI, libraryName,
			)
			if _, err := os.Stat(payload); err != nil {
				t.Fatalf("catalog declares bundled tool %q but %s is not in the APK payload: %v",
					tool.ToolID, payload, err)
			}

			sum, err := fileSHA256(payload)
			if err != nil {
				t.Fatalf("cannot hash %s: %v", payload, err)
			}
			if sum != expected {
				t.Fatalf(
					"catalog pins %s for %s payload %s but the packaged file hashes to %s.\n"+
						"Rebuild with the matching script in runtime/ and update "+
						"core/src/pkg/pcruntime/manifest.json.",
					expected, tool.ToolID, libraryName, sum,
				)
			}
			if err := verifyELFTarget(payload, tool.ABI); err != nil {
				t.Fatalf("packaged payload %s for %q is not a %s binary: %v",
					libraryName, tool.ToolID, tool.ABI, err)
			}
			checked++
		}
	}

	if checked == 0 {
		t.Log("no bundled tools in the catalog; nothing to verify")
	}
}

// TestStagedCoreEmbedsTheCurrentCatalog catches a stale Core binary.
//
// The catalog is compiled into the Core executable with //go:embed, but the
// executable itself is a committed artifact under jniLibs. Editing
// manifest.json therefore changes nothing on a device until core/
// build-android-arm64.sh is re-run and the rebuilt binary is staged. Nothing in
// the Gradle build does that, and the payload guard only checks that files are
// present, so a stale Core packages and installs cleanly while the app reports
// the previous catalog and cannot see the new tool.
//
// That is exactly what shipped once: an APK carrying the python payload beside
// a Core that had never heard of it.
func TestStagedCoreEmbedsTheCurrentCatalog(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout; there is no staged Core to compare")
	}
	corePath := filepath.Join(root, "android", "app", "src", "main", "jniLibs",
		"arm64-v8a", "libpocketclaw.so")
	core, err := os.ReadFile(corePath)
	if err != nil {
		t.Skipf("no staged Core binary to check: %v", err)
	}

	manifest, err := LoadEmbeddedManifest()
	if err != nil {
		t.Fatalf("cannot load catalog: %v", err)
	}

	version := []byte(fmt.Sprintf("%q: %q", "catalog_version", manifest.CatalogVersion))
	if !bytes.Contains(core, version) {
		t.Errorf(
			"the staged Core does not contain catalog version %s.\n"+
				"It was built before the current manifest.json and would ship a stale\n"+
				"catalog. Rebuild and re-stage it:\n"+
				"  ./core/build-android-arm64.sh",
			manifest.CatalogVersion,
		)
	}

	// Every bundled tool must be reachable from the Core that ships beside it.
	// The checksum is the load-bearing part: it is what the runtime verifies a
	// payload against, so a Core carrying an older one rejects the new payload
	// as a corrupt install.
	for _, tool := range manifest.Tools {
		if tool.Delivery != DeliveryBundled {
			continue
		}
		if !bytes.Contains(core, []byte(tool.LibraryName)) {
			t.Errorf("the staged Core does not know the payload %s for tool %q",
				tool.LibraryName, tool.ToolID)
		}
		if !bytes.Contains(core, []byte(tool.SHA256)) {
			t.Errorf("the staged Core does not carry the pinned checksum for %q; "+
				"it would reject the shipped payload as corrupt", tool.ToolID)
		}
	}
}
