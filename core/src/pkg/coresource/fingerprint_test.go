package coresource

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// syntheticCore writes a miniature core/src: enough structure for the
// fingerprint to have something to say, small enough to modify one file at a
// time and see exactly what moves.
func syntheticCore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{
		"go.mod":                        "module github.com/sipeed/picoclaw\n",
		"go.sum":                        "example.com/dep v1.0.0 h1:abc=\n",
		"Makefile":                      "build:\n\tgo build -tags stdjson ./...\n",
		"onboard_workspace_embed.go":    "package picoclaw\n",
		"cmd/picoclaw/main.go":          "package main\n\nfunc main() {}\n",
		"pkg/tools/python_tool.go":      "package tools\n\nconst marker = \"before\"\n",
		"pkg/tools/python_tool_test.go": "package tools\n\n// a test, not shipped\n",
		"pkg/pcruntime/manifest.json":   "{\"catalog_version\": \"2026.08.31\"}\n",
		"pkg/pcruntime/README.md":       "documentation, not a build input\n",
		"workspace/SKILL.md":            "an embedded workspace file\n",
	}
	for relative, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", relative, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("cannot write %s: %v", relative, err)
		}
	}
	return root
}

func fingerprintOf(t *testing.T, root string) string {
	t.Helper()
	fingerprint, err := Fingerprint(root)
	if err != nil {
		t.Fatalf("cannot fingerprint %s: %v", root, err)
	}
	if len(fingerprint) != 64 {
		t.Fatalf("fingerprint %q is not a sha256 digest", fingerprint)
	}
	return fingerprint
}

func rewrite(t *testing.T, root, relative, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot rewrite %s: %v", relative, err)
	}
}

// The failure this guard exists for: a Go file changes, manifest.json does not,
// and the catalog guard therefore sees nothing wrong while the staged Core no
// longer contains the code that was edited.
func TestAGoOnlyChangeMovesTheFingerprint(t *testing.T) {
	root := syntheticCore(t)
	before := fingerprintOf(t, root)

	catalog, err := os.ReadFile(filepath.Join(root, "pkg", "pcruntime", "manifest.json"))
	if err != nil {
		t.Fatalf("cannot read the catalog: %v", err)
	}

	rewrite(t, root, "pkg/tools/python_tool.go", "package tools\n\nconst marker = \"after\"\n")
	after := fingerprintOf(t, root)

	if after == before {
		t.Error("editing python_tool.go did not move the fingerprint; " +
			"a Go-only change would ship stale")
	}

	unchanged, err := os.ReadFile(filepath.Join(root, "pkg", "pcruntime", "manifest.json"))
	if err != nil {
		t.Fatalf("cannot re-read the catalog: %v", err)
	}
	if string(unchanged) != string(catalog) {
		t.Fatal("the test moved the catalog; it must prove detection without it")
	}
}

// The catalog is still a build input in its own right: the fingerprint has to
// cover it as well, so the two guards overlap rather than leaving a seam.
func TestEmbeddedJSONAndWorkspaceMoveTheFingerprint(t *testing.T) {
	for _, change := range []struct {
		name     string
		relative string
		content  string
	}{
		{"catalog", "pkg/pcruntime/manifest.json", "{\"catalog_version\": \"2026.09.01\"}\n"},
		{"embedded workspace", "workspace/SKILL.md", "a changed embedded file\n"},
		{"module pins", "go.sum", "example.com/dep v1.0.1 h1:def=\n"},
		{"build configuration", "Makefile", "build:\n\tgo build -tags stdjson,goolm ./...\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			root := syntheticCore(t)
			before := fingerprintOf(t, root)
			rewrite(t, root, change.relative, change.content)
			if fingerprintOf(t, root) == before {
				t.Errorf("changing %s did not move the fingerprint", change.relative)
			}
		})
	}
}

// A fingerprint that moved on its own would be worse than none: every build
// would report stale, and the report would stop being read. Only things that
// change the compiled binary may move it.
func TestFingerprintIgnoresWhatCannotReachTheBinary(t *testing.T) {
	for _, change := range []struct {
		name     string
		relative string
		content  string
	}{
		{"a test file", "pkg/tools/python_tool_test.go", "package tools\n\n// rewritten test\n"},
		{"documentation", "pkg/pcruntime/README.md", "rewritten documentation\n"},
	} {
		t.Run(change.name, func(t *testing.T) {
			root := syntheticCore(t)
			before := fingerprintOf(t, root)
			rewrite(t, root, change.relative, change.content)
			if fingerprintOf(t, root) != before {
				t.Errorf("changing %s moved the fingerprint; it does not reach the binary",
					change.relative)
			}
		})
	}
}

// Two clean checkouts of identical source must agree, or the guard fires on
// every machine that is not the one that built the Core.
func TestFingerprintIsContentAddressedNotLocationOrTimeAddressed(t *testing.T) {
	first := syntheticCore(t)
	second := syntheticCore(t)
	if first == second {
		t.Fatal("the two trees share a directory; the test would prove nothing")
	}

	if fingerprintOf(t, first) != fingerprintOf(t, second) {
		t.Error("the same source in two directories produced different fingerprints")
	}

	// Rewriting a file with identical bytes changes its mtime and nothing else.
	rewrite(t, first, "pkg/tools/python_tool.go", "package tools\n\nconst marker = \"before\"\n")
	if fingerprintOf(t, first) != fingerprintOf(t, second) {
		t.Error("a touched file moved the fingerprint; mtime must play no part")
	}
}

// Both directions of the staged-Core check, so the guard is known to fail as
// well as known to pass.
func TestStagedCoreMatchesOnlyTheFingerprintItCarries(t *testing.T) {
	fingerprint := fingerprintOf(t, syntheticCore(t))

	current := []byte("ELF...\x00" + fingerprint + "\x00...more data")
	if !StagedCoreMatches(current, fingerprint) {
		t.Error("a Core carrying the fingerprint was reported as stale")
	}

	stale := []byte("ELF...\x00" + strings.Repeat("0", 64) + "\x00...more data")
	if StagedCoreMatches(stale, fingerprint) {
		t.Error("a Core built from other source was reported as current")
	}

	unstamped := []byte("ELF...\x00no fingerprint here\x00...")
	if StagedCoreMatches(unstamped, fingerprint) {
		t.Error("a Core built outside the canonical script was reported as current")
	}

	if StagedCoreMatches(current, "") {
		t.Error("an empty fingerprint matched; it would match every binary")
	}
}

func TestDescribeNamesAnUnstampedBuild(t *testing.T) {
	original := Stamped
	t.Cleanup(func() { Stamped = original })

	Stamped = ""
	if Describe() != "unstamped" {
		t.Errorf("Describe() = %q for an unstamped build", Describe())
	}
	Stamped = strings.Repeat("a", 64)
	if Describe() != Stamped {
		t.Errorf("Describe() = %q, want the stamped fingerprint", Describe())
	}
}

// Tests write files beside the packages they exercise. pkg/cron writes a JSON
// file into its own directory, which moved the fingerprint mid-run and made the
// gate fail against a Core that was in fact current. A build input is a file a
// build reads, and no build reads that one.
func TestTestArtifactsDoNotMoveTheFingerprint(t *testing.T) {
	root := syntheticCore(t)
	before := fingerprintOf(t, root)

	for _, litter := range []string{
		"pkg/cron/test_cron_1788189241759837435.json",
		"pkg/cron/.tmp-22370-1788189244991176026",
		"pkg/tools/coverage.out",
	} {
		full := filepath.Join(root, filepath.FromSlash(litter))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", litter, err)
		}
		if err := os.WriteFile(full, []byte("transient\n"), 0o644); err != nil {
			t.Fatalf("cannot write %s: %v", litter, err)
		}
	}

	if fingerprintOf(t, root) != before {
		t.Error("a file a test wrote moved the fingerprint; the gate would fail at random")
	}
}

// The named list of embedded assets is the one part of the input set that can
// go quietly stale: add a //go:embed and forget the list, and a change to the
// embedded file ships without moving the fingerprint. This reads the real
// directives out of the real Core source rather than trusting the list.
func TestEveryEmbeddedAssetIsAFingerprintInput(t *testing.T) {
	root := repoRoot()
	if root == "" {
		t.Skip("not running inside a PocketClaw checkout; there is no Core source to read")
	}
	coreSrc := filepath.Join(root, "core", "src")

	for _, directive := range embedDirectives(t, coreSrc) {
		if !covers(directive.target) {
			t.Errorf(
				"%s embeds %q, which is not a fingerprint input.\n"+
					"A change to it would ship without the guard noticing. Add it to "+
					"embeddedFiles in fingerprint.go.",
				directive.source, directive.target,
			)
		}
	}
}

type embedDirective struct {
	source string // the Go file carrying the directive, relative to core/src
	target string // the embedded path, relative to core/src
}

// embedDirectives finds every //go:embed inside the trees the fingerprint
// walks. Directives outside them belong to the separate launcher binary, which
// this fingerprint deliberately does not describe.
func embedDirectives(t *testing.T, coreSrc string) []embedDirective {
	t.Helper()

	var found []embedDirective
	for _, top := range append([]string{"."}, includedRoots...) {
		base := filepath.Join(coreSrc, top)
		err := filepath.WalkDir(base, func(full string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			relative, relErr := filepath.Rel(coreSrc, full)
			if relErr != nil {
				return relErr
			}
			relative = filepath.ToSlash(relative)

			if entry.IsDir() {
				// "." walks the whole tree; only its own files are wanted from
				// it, because the other roots are walked in their own right.
				if top == "." && relative != "." {
					return fs.SkipDir
				}
				if _, skipped := skippedDirs[entry.Name()]; skipped && relative != top {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(relative, ".go") || strings.HasSuffix(relative, "_test.go") {
				return nil
			}
			body, readErr := os.ReadFile(full)
			if readErr != nil {
				return readErr
			}
			for _, line := range strings.Split(string(body), "\n") {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "//go:embed ") {
					continue
				}
				for _, pattern := range strings.Fields(strings.TrimPrefix(line, "//go:embed ")) {
					pattern = strings.TrimPrefix(pattern, "all:")
					target := path.Join(path.Dir(relative), pattern)
					found = append(found, embedDirective{source: relative, target: target})
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("cannot scan %s for embed directives: %v", top, err)
		}
	}
	if len(found) == 0 {
		t.Fatal("no //go:embed directives found; the scan is not reading the Core source")
	}
	return found
}
