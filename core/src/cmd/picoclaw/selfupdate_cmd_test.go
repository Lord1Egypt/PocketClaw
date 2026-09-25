package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestAndroidCoreLinksNoSelfUpdater proves the Android Core cannot expose the
// upstream self-updater: not merely that the command is unregistered, but that
// neither pkg/updater nor minio/selfupdate is in the Android link at all. It
// resolves the package graph for the target rather than for this host, which a
// host test otherwise never sees.
func TestAndroidCoreLinksNoSelfUpdater(t *testing.T) {
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(goBin, "list", "-deps", "-tags", "stdjson", ".")
	cmd.Env = append(os.Environ(), "GOOS=android", "GOARCH=arm64", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list for android/arm64 failed: %v\n%s", err, out)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if pkg == "github.com/sipeed/picoclaw/pkg/updater" ||
			strings.HasPrefix(pkg, "github.com/minio/selfupdate") {
			t.Errorf("android Core links %s", pkg)
		}
	}
	if !strings.Contains(string(out), "github.com/spf13/cobra") {
		t.Fatalf("go list output does not look like the Core graph:\n%s", out)
	}
}
