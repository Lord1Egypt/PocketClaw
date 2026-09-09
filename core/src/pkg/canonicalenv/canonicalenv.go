// Package canonicalenv resolves PocketClaw's canonical POCKETCLAW_*
// environment variables over the legacy PICOCLAW_* names the vendored Core
// still reads.
//
// PocketClaw's Android host emits only POCKETCLAW_* names. The Core it launches
// is upstream source that reads PICOCLAW_* — in roughly 175 struct tags and in
// a dozen direct os.Getenv calls. Renaming those would be a permanent
// divergence from upstream for no behavioural gain, so the translation lives
// here instead, in one table, and every reader of a migrated variable goes
// through it.
//
// The PICOCLAW_* strings in this package are LEGACY / UPSTREAM ENV INPUT
// COMPATIBILITY: names this build still accepts, never names it produces.
// This is the whole compatibility surface, so the Zero-Pico guard can allowlist
// exactly this file rather than a scatter of literals.
package canonicalenv

import "os"

// aliases maps each canonical PocketClaw variable to the legacy name the
// vendored Core reads it under.
//
// Deliberately an explicit table rather than a blanket POCKETCLAW_ → PICOCLAW_
// prefix rewrite. A prefix rule would invent a legacy twin for every
// PocketClaw-owned variable that has nothing to do with upstream config —
// POCKETCLAW_RUNTIME_DIR, POCKETCLAW_ANDROID_BRIDGE_TOKEN and the rest — and
// any one of those could collide with a real upstream name added later.
//
// The channel token is the one entry whose suffix also changes. The channel is
// called pocketclaw everywhere it is configured and routed; what still says
// PICO is the upstream struct tag that reads this variable, and retagging it
// would put a single canonical name among ~175 legacy ones for no behavioural
// gain. The two halves differ because the tag has not moved, not because the
// channel has not.
var aliases = map[string]string{
	"POCKETCLAW_HOME":                      "PICOCLAW_HOME",
	"POCKETCLAW_CONFIG":                    "PICOCLAW_CONFIG",
	"POCKETCLAW_BINARY":                    "PICOCLAW_BINARY",
	"POCKETCLAW_GATEWAY_TOKEN_FILE":        "PICOCLAW_GATEWAY_TOKEN_FILE",
	"POCKETCLAW_LOG_DIR":                   "PICOCLAW_LOG_DIR",
	"POCKETCLAW_DASHBOARD_AUTH_DIR":        "PICOCLAW_DASHBOARD_AUTH_DIR",
	"POCKETCLAW_DNS_SERVER":                "PICOCLAW_DNS_SERVER",
	"POCKETCLAW_GATEWAY_HOT_RELOAD":        "PICOCLAW_GATEWAY_HOT_RELOAD",
	"POCKETCLAW_TOOLS_I2C_ENABLED":         "PICOCLAW_TOOLS_I2C_ENABLED",
	"POCKETCLAW_TOOLS_SPI_ENABLED":         "PICOCLAW_TOOLS_SPI_ENABLED",
	"POCKETCLAW_TOOLS_SERIAL_ENABLED":      "PICOCLAW_TOOLS_SERIAL_ENABLED",
	"POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN": "PICOCLAW_CHANNELS_PICO_TOKEN",
}

// legacyToCanonical is the reverse index, built once so a lookup by the legacy
// name a caller already holds does not have to scan the table.
var legacyToCanonical = func() map[string]string {
	reverse := make(map[string]string, len(aliases))
	for canonical, legacy := range aliases {
		reverse[legacy] = canonical
	}
	return reverse
}()

// Aliases returns a copy of the canonical → legacy mapping.
//
// A copy, because the parser overlay hands its result to a third-party decoder
// and a shared map would let any caller edit the compatibility contract for the
// whole process.
func Aliases() map[string]string {
	out := make(map[string]string, len(aliases))
	for canonical, legacy := range aliases {
		out[canonical] = legacy
	}
	return out
}

// LookupEnv reads the variable named by its legacy key, preferring the
// canonical POCKETCLAW_* name when that is set.
//
// Callers pass the legacy name because that is what upstream code already has
// in hand — config.EnvLogDir, androiddns.EnvServer and so on. An unmapped key
// falls straight through to os.LookupEnv, so this is safe to use anywhere.
//
// Presence, not emptiness, decides. POCKETCLAW_LOG_DIR="" is set, and it wins
// over a non-empty PICOCLAW_LOG_DIR rather than falling through to it: a host
// that deliberately blanks a variable is saying something, and reading past it
// to a stale legacy value would silently restore whatever the host was trying
// to turn off.
func LookupEnv(key string) (string, bool) {
	if canonical, mapped := legacyToCanonical[key]; mapped {
		if value, ok := os.LookupEnv(canonical); ok {
			return value, true
		}
	}
	return os.LookupEnv(key)
}

// Getenv is LookupEnv for callers that do not distinguish unset from empty.
func Getenv(key string) string {
	value, _ := LookupEnv(key)
	return value
}
