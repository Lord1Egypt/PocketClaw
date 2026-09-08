// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg"
)

// Runtime environment variable keys for the picoclaw process.
// These control the location of files and binaries at runtime and are read
// directly via os.Getenv / os.LookupEnv. All picoclaw-specific keys use the
// PICOCLAW_ prefix. Reference these constants instead of inline string
// literals to keep all supported knobs visible in one place and to prevent
// typos.
const (
	// EnvHome overrides the base directory for all picoclaw data
	// (config, workspace, skills, auth store, …).
	// Default: ~/.picoclaw
	EnvHome = "PICOCLAW_HOME"

	// EnvConfig overrides the full path to the JSON config file.
	// Default: $PICOCLAW_HOME/config.json
	EnvConfig = "PICOCLAW_CONFIG"

	// EnvBuiltinSkills overrides the directory from which built-in
	// skills are loaded.
	// Default: <cwd>/skills
	EnvBuiltinSkills = "PICOCLAW_BUILTIN_SKILLS"

	// EnvBinary overrides the path to the picoclaw executable.
	// Used by the web launcher when spawning the gateway subprocess.
	// Default: resolved from the same directory as the current executable.
	EnvBinary = "PICOCLAW_BINARY"

	// EnvGatewayHost overrides the host address for the gateway server.
	// Default: "localhost"
	EnvGatewayHost = "PICOCLAW_GATEWAY_HOST"

	// EnvGatewayTokenFile moves the gateway's bearer credential out of the
	// PID record and into a file of its own.
	//
	// The PID record is a discovery artifact and lives in PICOCLAW_HOME, which
	// on Android is a user-visible directory on shared external storage where
	// POSIX modes are not honoured. A credential must not be in it. When this
	// is set the token is written here instead — with the record left
	// token-free — and the host reads it from a location only the app can
	// reach. Unset, the token stays in the record exactly as before, which is
	// the correct behaviour on a desktop or server where PICOCLAW_HOME is
	// already private.
	// Default: unset.
	EnvGatewayTokenFile = "PICOCLAW_GATEWAY_TOKEN_FILE"

	// EnvChannelsPicoToken supplies the realtime channel credential from the
	// host rather than from config. Its presence also means the credential is
	// host-managed, which is what lets the channel refuse query-string
	// authentication for it.
	EnvChannelsPicoToken = "PICOCLAW_CHANNELS_PICO_TOKEN"

	// EnvDashboardAuthDir overrides the directory holding the Dashboard
	// credential database.
	//
	// launcher-auth.db holds a bcrypt verifier — no plaintext and no session
	// token — so reading it buys an attacker little. Writing it is the problem:
	// under PICOCLAW_HOME on Android it sits on shared external storage, where
	// an app with storage write access can replace the stored verifier with one
	// for a password it chose and then log in normally over loopback, which
	// Android does not isolate between apps. That is an authentication bypass
	// that never has to break bcrypt at all. Pointing this at app-private
	// storage removes the write vector along with the read one.
	// Default: PICOCLAW_HOME
	EnvDashboardAuthDir = "PICOCLAW_DASHBOARD_AUTH_DIR"

	// EnvLogDir overrides the directory holding gateway.log and the panic log.
	//
	// Both default to a "logs" directory under PICOCLAW_HOME. On Android that
	// is shared external storage, readable by any app holding storage access,
	// and the file has never rotated — so a long-lived install accumulates an
	// unbounded diagnostic record in a public place. Pointing this at
	// app-private storage keeps the workspace user-visible, as intended, while
	// the logs are not.
	// Default: $PICOCLAW_HOME/logs
	EnvLogDir = "PICOCLAW_LOG_DIR"
)

// ResolveDashboardAuthDir returns the directory holding launcher-auth.db.
//
// homePath is the caller's PICOCLAW_HOME, which stays the default so desktop
// and server installs are unchanged. A host that can offer private storage says
// so through EnvDashboardAuthDir.
func ResolveDashboardAuthDir(homePath string) string {
	if dir := strings.TrimSpace(os.Getenv(EnvDashboardAuthDir)); dir != "" {
		return dir
	}
	return homePath
}

// ResolveLogDir returns the directory for gateway.log and the panic log.
//
// homePath is the caller's PICOCLAW_HOME. EnvLogDir wins when set, so a host
// that has somewhere better to put logs than the user's workspace can say so
// without every caller re-deriving the rule.
func ResolveLogDir(homePath string) string {
	if dir := strings.TrimSpace(os.Getenv(EnvLogDir)); dir != "" {
		return dir
	}
	return filepath.Join(homePath, "logs")
}

// ResolveConfigPath returns the JSON config file this process reads, applying
// the same precedence the CLI uses: EnvConfig when set, otherwise
// config.json under GetHome. Long-lived processes that must observe config
// edits made by the web console — rather than a snapshot taken at start —
// reload from this path.
func ResolveConfigPath() string {
	if configPath := os.Getenv(EnvConfig); configPath != "" {
		return configPath
	}
	return filepath.Join(GetHome(), "config.json")
}

func GetHome() string {
	homePath, _ := os.UserHomeDir()
	if picoclawHome := os.Getenv(EnvHome); picoclawHome != "" {
		homePath = picoclawHome
	} else if homePath != "" {
		homePath = filepath.Join(homePath, pkg.DefaultPicoClawHome)
	}
	if homePath == "" {
		homePath = "."
	}
	return homePath
}
