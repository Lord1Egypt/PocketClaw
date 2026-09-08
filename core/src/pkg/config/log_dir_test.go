package config

import (
	"path/filepath"
	"testing"
)

// Gateway logs default to the user's workspace, which on Android is shared
// external storage. A host with somewhere private to put them says so through
// the environment rather than every caller re-deriving the rule.
func TestResolveLogDir(t *testing.T) {
	home := t.TempDir()

	t.Run("defaults under the home directory", func(t *testing.T) {
		t.Setenv(EnvLogDir, "")
		if got, want := ResolveLogDir(home), filepath.Join(home, "logs"); got != want {
			t.Errorf("ResolveLogDir() = %q, want %q", got, want)
		}
	})

	t.Run("the environment wins when a host sets it", func(t *testing.T) {
		private := t.TempDir()
		t.Setenv(EnvLogDir, private)
		if got := ResolveLogDir(home); got != private {
			t.Errorf("ResolveLogDir() = %q, want %q", got, private)
		}
		if got := ResolveLogDir(home); got == filepath.Join(home, "logs") {
			t.Error("logs must not fall back into the shared workspace once redirected")
		}
	})

	t.Run("blank is treated as unset", func(t *testing.T) {
		t.Setenv(EnvLogDir, "   ")
		if got, want := ResolveLogDir(home), filepath.Join(home, "logs"); got != want {
			t.Errorf("ResolveLogDir() = %q, want %q", got, want)
		}
	})
}
