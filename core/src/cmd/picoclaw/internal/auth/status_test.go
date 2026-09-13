package auth

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/config"
)

func captureAuthStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	fn()

	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return buf.String()
}

func setAuthStatusTestHome(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv(config.EnvHome, filepath.Join(tmpDir, ".picoclaw"))
	return tmpDir
}

func TestNewStatusSubcommand(t *testing.T) {
	cmd := newStatusCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "Show current auth status", cmd.Short)

	assert.False(t, cmd.HasFlags())
}
