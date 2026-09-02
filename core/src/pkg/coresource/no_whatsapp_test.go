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
