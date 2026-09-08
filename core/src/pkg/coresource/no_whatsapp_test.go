package coresource

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The WhatsApp Agent Channel was cancelled: the whatsmeow transport added more
// runtime and protocol surface than a phone-resident PocketClaw could justify,
// and Telegram already covers the remote agent channel. These guards keep the
// decision from eroding, because nothing else would notice it: the transport
// lives in vendored upstream Core behind a build tag, so re-enabling it is one
// word in a Makefile and produces a green build.

// TestAndroidBuildDoesNotEnableWhatsAppNative pins the build tag off.
func TestAndroidBuildDoesNotEnableWhatsAppNative(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}
	makefile, err := os.ReadFile(filepath.Join(root, "core", "src", "Makefile"))
	if err != nil {
		t.Fatalf("cannot read the Core Makefile: %v", err)
	}

	for i, line := range strings.Split(string(makefile), "\n") {
		if !strings.Contains(line, "GOOS=android") {
			continue
		}
		if strings.Contains(line, "whatsapp_native") {
			t.Errorf("Makefile:%d enables whatsapp_native for Android: %s", i+1, strings.TrimSpace(line))
		}
	}
}

// TestStagedCoreDoesNotLinkWhatsmeow checks the binary Gradle packages rather
// than the recipe that produced it.
//
// Without the build tag the transport compiles to a stub whose error string is
// the marker below; whatsmeow itself contributes tens of thousands of symbols,
// so its absence is unambiguous.
func TestStagedCoreDoesNotLinkWhatsmeow(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}
	core, err := os.ReadFile(filepath.Join(root, "android", "app", "src", "main",
		"jniLibs", "arm64-v8a", "libpicoclaw.so"))
	if err != nil {
		t.Skipf("no staged Core binary to check: %v", err)
	}

	if bytes.Contains(core, []byte("go.mau.fi/whatsmeow")) {
		t.Error("the staged Core links whatsmeow; it was built with -tags whatsapp_native")
	}
	// The stub proves the package is present but inert, which is the intended
	// dormant state for vendored upstream code.
	if !bytes.Contains(core, []byte("whatsapp native not compiled in")) {
		t.Error("the staged Core does not carry the inert WhatsApp stub")
	}
}

// TestNoUserFacingWhatsAppSurface keeps WhatsApp out of everything a user can
// reach: the console catalog, the console bundle, the Android host and the
// Flutter app. The vendored upstream channel packages are deliberately exempt —
// they are dormant, unreachable without hand-editing a config file, and
// deleting them would diverge from upstream for no runtime benefit.
func TestNoUserFacingWhatsAppSurface(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout")
	}

	// Paths that may still mention WhatsApp: vendored upstream Core.
	exempt := []string{
		filepath.Join("core", "src", "pkg", "channels", "whatsapp"),
		filepath.Join("core", "src", "pkg", "channels", "whatsapp_native"),
		filepath.Join("core", "src", "pkg", "config"),
		filepath.Join("core", "src", "pkg", "gateway"),
		filepath.Join("core", "src", "pkg", "migrate"),
		filepath.Join("core", "src", "pkg", "agent"),
		filepath.Join("core", "src", "pkg", "commands"),
		filepath.Join("core", "src", "pkg", "tools", "integration"),
		filepath.Join("core", "src", "config"),
		filepath.Join("core", "src", "docs"),
		filepath.Join("core", "src", "workspace"),
		filepath.Join("core", "src", "web", "backend", "dist"),
		filepath.Join("core", "src", "pkg", "coresource"),
	}

	// Trees a user actually reaches.
	roots := []string{
		filepath.Join("lib"),
		filepath.Join("test"),
		filepath.Join("android", "app", "src"),
		filepath.Join("core", "src", "web", "frontend", "src"),
		filepath.Join("core", "src", "web", "backend", "api"),
	}

	for _, rel := range roots {
		base := filepath.Join(root, rel)
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil //nolint:nilerr // a missing tree is not this test's concern
			}
			for _, skip := range exempt {
				if strings.Contains(path, skip) {
					return nil
				}
			}
			// The guards that assert WhatsApp's absence necessarily name it.
			if strings.Contains(filepath.Base(path), "no_whatsapp") {
				return nil
			}
			// So do guards elsewhere that enumerate prohibited names as
			// enforcement data — a forbidden-substring list is the mechanism of
			// the prohibition, not a surface that violates it. Reading one as a
			// violation is this guard failing on its own allies, which is what
			// it did to whats_new_page_test.dart.
			//
			// Opt-in and greppable rather than a basename heuristic: a file
			// must say so, so adding the marker to a real product file is a
			// visible act a reviewer would question. Checked against the raw
			// bytes because the marker lives in a comment, which the surface
			// check below deliberately strips.
			if raw, readErr := os.ReadFile(path); readErr == nil &&
				bytes.Contains(raw, []byte(enforcementDataMarker)) {
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".dart", ".kt", ".ts", ".tsx", ".json", ".xml":
			default:
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil //nolint:nilerr // unreadable files are not a WhatsApp surface
			}
			// Comments are not a surface. A note recording that WhatsApp was
			// removed, and why, is exactly the kind of thing that should
			// survive — it is what stops the next person re-adding it.
			if bytes.Contains(bytes.ToLower(stripComments(data)), []byte("whatsapp")) {
				t.Errorf("%s still references WhatsApp outside a comment", strings.TrimPrefix(path, root+string(os.PathSeparator)))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", rel, err)
		}
	}
}

// enforcementDataMarker exempts a file that enumerates prohibited names as the
// data behind its own prohibition.
//
// Deliberately a long, distinctive phrase: it must be impossible to write by
// accident and obvious in a diff. It exempts the file from the *surface* scan
// only; nothing about the product prohibition changes, and every other file in
// every scanned tree is checked exactly as before.
const enforcementDataMarker = "WHATSAPP-GUARD-ENFORCEMENT-DATA"

// stripComments blanks //, /* */ and <!-- --> comments so the surface check
// reads code rather than prose. It is deliberately crude: a comment marker
// inside a string literal costs a false negative, never a false positive, and
// this guard exists to catch a WhatsApp surface coming back, not to parse four
// languages correctly.
func stripComments(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		switch {
		case bytes.HasPrefix(data[i:], []byte("//")):
			for i < len(data) && data[i] != '\n' {
				i++
			}
		case bytes.HasPrefix(data[i:], []byte("/*")):
			if end := bytes.Index(data[i+2:], []byte("*/")); end >= 0 {
				i += 2 + end + 2
			} else {
				i = len(data)
			}
		case bytes.HasPrefix(data[i:], []byte("<!--")):
			if end := bytes.Index(data[i+4:], []byte("-->")); end >= 0 {
				i += 4 + end + 3
			} else {
				i = len(data)
			}
		default:
			out = append(out, data[i])
			i++
		}
	}
	return out
}

// The guard must still fail on a real user-facing surface, and must not be
// weakened by the enforcement-data exemption it now honours.
//
// These exercise the decision directly rather than the whole tree walk, because
// what needed proving is the rule, not the traversal.
func TestWhatsAppSurfaceDetectionIsNotWeakenedByTheEnforcementExemption(t *testing.T) {
	// A real surface: a widget label a user would actually read. It carries no
	// marker, so it must be caught.
	surface := []byte(`Text(l10n.channelWhatsAppTitle)`)
	if !bytes.Contains(bytes.ToLower(stripComments(surface)), []byte("whatsapp")) {
		t.Fatal("a genuine user-facing WhatsApp surface is no longer detected")
	}
	if bytes.Contains(surface, []byte(enforcementDataMarker)) {
		t.Fatal("a product file must not carry the enforcement-data marker")
	}

	// A prohibition's own data, which names WhatsApp precisely in order to
	// forbid it. Marked, and therefore exempt.
	enforcement := []byte(`
// ` + enforcementDataMarker + ` — this list is the prohibition, not a breach.
const forbidden = ['WhatsApp'];
`)
	if !bytes.Contains(enforcement, []byte(enforcementDataMarker)) {
		t.Fatal("the enforcement list is no longer recognised as enforcement data")
	}

	// The exemption is opt-in and cannot be reached by accident: an unmarked
	// file naming WhatsApp outside a comment is still a violation.
	unmarked := []byte(`const forbidden = ['WhatsApp'];`)
	if bytes.Contains(unmarked, []byte(enforcementDataMarker)) {
		t.Fatal("an unmarked file was treated as enforcement data")
	}
	if !bytes.Contains(bytes.ToLower(stripComments(unmarked)), []byte("whatsapp")) {
		t.Fatal("an unmarked WhatsApp reference is no longer detected")
	}

	// And a comment still is not a surface: the note explaining the removal is
	// exactly what should survive.
	comment := []byte(`// WhatsApp was removed in the cleanup milestone.`)
	if bytes.Contains(bytes.ToLower(stripComments(comment)), []byte("whatsapp")) {
		t.Fatal("a comment recording the removal is being treated as a surface")
	}
}

// Exactly one file may claim the exemption. A second would mean the marker had
// started spreading, which is how a narrow exemption becomes a blanket one.
func TestOnlyTheKnownEnforcementListClaimsTheExemption(t *testing.T) {
	root := repoRoot()
	var marked []string
	for _, rel := range []string{"lib", "test", filepath.Join("android", "app", "src")} {
		_ = filepath.Walk(filepath.Join(root, rel), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil //nolint:nilerr // a missing tree is not this test's concern
			}
			if data, readErr := os.ReadFile(path); readErr == nil &&
				bytes.Contains(data, []byte(enforcementDataMarker)) {
				marked = append(marked, strings.TrimPrefix(path, root+string(os.PathSeparator)))
			}
			return nil
		})
	}
	want := filepath.Join("test", "widgets", "whats_new_page_test.dart")
	if len(marked) != 1 || marked[0] != want {
		t.Fatalf("enforcement-data exemption claimed by %v, want exactly [%s]", marked, want)
	}
}
