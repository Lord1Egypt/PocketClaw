// Package session resolves where the WhatsApp native transport keeps its
// whatsmeow session database.
//
// That database holds the Signal-protocol identity and session keys for a
// linked WhatsApp device: whoever can read it can impersonate the account.
// It therefore must never live in the agent workspace, which on Android is
// public external storage (Download/pocketclaw) and is also the root the
// agent's own file tools are restricted to.
//
// So the path is not a setting. When PocketClaw's Android host launches Core
// it names an app-private, no-backup directory in the environment, and that
// value wins over anything in the config file — including a value the model
// wrote there.
package session

import (
	"os"
	"path/filepath"
	"strings"
)

// EnvStoreDir names the directory the Android host owns for the session
// database. The host sets it to app-private no-backup storage; anywhere else —
// desktop, CI — it is unset and the configured path is used instead.
const EnvStoreDir = "POCKETCLAW_WHATSAPP_SESSION_DIR"

// Resolution reports the chosen store directory and how it was chosen.
type Resolution struct {
	// Path is the directory the session database lives in.
	Path string
	// HostOwned is true when the Android host named the path, which means the
	// configured session_store_path was ignored rather than merged.
	HostOwned bool
	// IgnoredConfigured is the configured value that was discarded because the
	// host owns the path. It exists so the caller can say so once, at INFO,
	// instead of silently dropping a setting the user can see in their config.
	IgnoredConfigured string
}

// Resolve picks the session store directory.
//
// The host's directory wins unconditionally. This is the whole point: on
// Android a configured path is untrusted input, because the agent can write
// the config file, and a path pointing back into the workspace would put the
// account keys inside the agent's own sandbox.
func Resolve(configured, workspaceFallback string) Resolution {
	if hostDir := strings.TrimSpace(os.Getenv(EnvStoreDir)); hostDir != "" {
		return Resolution{
			Path:              hostDir,
			HostOwned:         true,
			IgnoredConfigured: strings.TrimSpace(configured),
		}
	}
	if c := strings.TrimSpace(configured); c != "" {
		return Resolution{Path: c}
	}
	return Resolution{Path: filepath.Join(workspaceFallback, "whatsapp")}
}

// HostOwnsPath reports whether an Android host has claimed the session path.
func HostOwnsPath() bool {
	return strings.TrimSpace(os.Getenv(EnvStoreDir)) != ""
}
