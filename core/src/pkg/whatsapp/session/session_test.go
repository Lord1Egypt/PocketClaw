package session

import (
	"path/filepath"
	"testing"
)

func TestResolve_HostDirectoryWinsOverConfiguredPath(t *testing.T) {
	// This is the security property, not a preference. On Android the config
	// file is writable by the agent, and the workspace it could point at is the
	// agent's own sandbox root, so a configured path must never be able to
	// relocate the account keys.
	t.Setenv(EnvStoreDir, "/data/data/app/no_backup/whatsapp")

	got := Resolve("/storage/emulated/0/Download/pocketclaw/whatsapp", "/workspace")

	if got.Path != "/data/data/app/no_backup/whatsapp" {
		t.Errorf("Path = %q, want the host directory", got.Path)
	}
	if !got.HostOwned {
		t.Error("HostOwned = false, want true when the host named the path")
	}
	if got.IgnoredConfigured != "/storage/emulated/0/Download/pocketclaw/whatsapp" {
		t.Errorf("IgnoredConfigured = %q, want the discarded configured path", got.IgnoredConfigured)
	}
}

func TestResolve_HostDirectoryWinsWithNoConfiguredPath(t *testing.T) {
	t.Setenv(EnvStoreDir, "/data/data/app/no_backup/whatsapp")

	got := Resolve("", "/workspace")

	if got.Path != "/data/data/app/no_backup/whatsapp" {
		t.Errorf("Path = %q, want the host directory", got.Path)
	}
	if got.IgnoredConfigured != "" {
		t.Errorf("IgnoredConfigured = %q, want empty when nothing was configured", got.IgnoredConfigured)
	}
}

func TestResolve_HostDirectoryIgnoredWhenBlank(t *testing.T) {
	// An env var set to whitespace is not a claim of ownership.
	t.Setenv(EnvStoreDir, "   ")

	got := Resolve("/configured", "/workspace")

	if got.Path != "/configured" {
		t.Errorf("Path = %q, want the configured path", got.Path)
	}
	if got.HostOwned {
		t.Error("HostOwned = true, want false for a blank host directory")
	}
}

func TestResolve_UsesConfiguredPathOffAndroid(t *testing.T) {
	t.Setenv(EnvStoreDir, "")

	got := Resolve("/configured/whatsapp", "/workspace")

	if got.Path != "/configured/whatsapp" {
		t.Errorf("Path = %q, want the configured path", got.Path)
	}
	if got.HostOwned {
		t.Error("HostOwned = true, want false with no host")
	}
}

func TestResolve_FallsBackToWorkspaceOffAndroid(t *testing.T) {
	t.Setenv(EnvStoreDir, "")

	got := Resolve("", "/workspace")

	want := filepath.Join("/workspace", "whatsapp")
	if got.Path != want {
		t.Errorf("Path = %q, want %q", got.Path, want)
	}
}

func TestHostOwnsPath(t *testing.T) {
	t.Setenv(EnvStoreDir, "")
	if HostOwnsPath() {
		t.Error("HostOwnsPath() = true with no host directory")
	}
	t.Setenv(EnvStoreDir, "/data/data/app/no_backup/whatsapp")
	if !HostOwnsPath() {
		t.Error("HostOwnsPath() = false with a host directory")
	}
}
