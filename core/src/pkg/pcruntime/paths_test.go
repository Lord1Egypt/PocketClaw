package pcruntime

import (
	"path/filepath"
	"strings"
	"testing"
)

// Runtime state that a user Skill can rewrite is worse than no runtime state at
// all, so the layout refuses to place metadata inside the user workspace.
func TestRuntimeMetadataMayNotLiveInTheUserWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "Download", "pocketclaw")

	t.Setenv(EnvWorkspace, workspace)
	t.Setenv(EnvRuntimeDir, filepath.Join(workspace, "runtime"))
	t.Setenv(EnvLibDir, filepath.Join(root, "lib"))

	if _, err := ResolvePaths(); err == nil || !strings.Contains(err.Error(), "user workspace") {
		t.Fatalf("expected the layout to refuse workspace-internal storage, got %v", err)
	}
}

func TestRuntimeMetadataOutsideTheWorkspaceIsAccepted(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "Download", "pocketclaw")
	metadata := filepath.Join(root, "files", "picoclaw", "runtime")

	t.Setenv(EnvWorkspace, workspace)
	t.Setenv(EnvRuntimeDir, metadata)
	t.Setenv(EnvLibDir, filepath.Join(root, "lib"))

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("layout was refused: %v", err)
	}
	if paths.MetadataDir != metadata {
		t.Fatalf("unexpected metadata dir %q", paths.MetadataDir)
	}
}

// The Android Service already exports the Core binary path, and the Core binary
// is resolved from nativeLibraryDir. Deriving the payload directory from it means
// a bundled tool resolves on an installed device without new plumbing.
func TestBundlePayloadDirectoryIsDerivedFromTheCoreBinaryPath(t *testing.T) {
	root := t.TempDir()
	libDir := filepath.Join(root, "lib", "arm64")

	t.Setenv(EnvWorkspace, filepath.Join(root, "workspace"))
	t.Setenv(EnvRuntimeDir, filepath.Join(root, "runtime"))
	t.Setenv(EnvLibDir, "")
	t.Setenv(EnvCoreBinary, filepath.Join(libDir, "libpicoclaw.so"))

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("layout was refused: %v", err)
	}
	if paths.LibDir != libDir {
		t.Fatalf("expected the payload directory to be %q, got %q", libDir, paths.LibDir)
	}
}

func TestBundledPathUsesTheDeclaredLibraryName(t *testing.T) {
	paths := &Paths{LibDir: "/data/app/x/lib/arm64", MetadataDir: "/data/data/x/files/runtime"}
	tool := &Tool{ToolID: "jq", LibraryName: "libpocketclaw-jq.so"}

	if got := paths.BundledPath(tool); got != "/data/app/x/lib/arm64/libpocketclaw-jq.so" {
		t.Fatalf("unexpected bundled path %q", got)
	}
}
