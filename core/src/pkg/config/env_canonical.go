package config

import (
	"os"

	"github.com/caarlos0/env/v11"

	"github.com/sipeed/picoclaw/pkg/canonicalenv"
)

// parserEnvironment is the environment the struct-tag config parser sees.
//
// It is the real environment plus the canonical overlay, built fresh for each
// parse and handed to env.ParseWithOptions. Nothing here calls os.Setenv: the
// process environment stays canonical, so os.Getenv("PICOCLAW_LOG_DIR")
// returns whatever it returned before — including nothing, if it was never
// set. Writing legacy names into the live environment would be observable to
// every other reader in the process and inherited by every child it spawns,
// which is exactly the namespace this migration removes.
//
// Unrelated entries are carried through untouched, so env-tagged configuration
// that has nothing to do with this mapping keeps working, for upstream users
// on the legacy names as much as for PocketClaw on the canonical ones.
//
// Presence decides, matching how the parser itself decides it — env.getOr does
// `value, exists := envs[key]`. So a canonical variable set to the empty string
// is present and beats a non-empty legacy one. What an empty value then means
// (a field's envDefault may still apply) is upstream's own behaviour, and is
// the same whichever namespace supplied it.
func parserEnvironment() map[string]string {
	environment := env.ToMap(os.Environ())
	for canonical, legacy := range canonicalenv.Aliases() {
		if value, ok := environment[canonical]; ok {
			environment[legacy] = value
		}
	}
	return environment
}

// parseEnv decodes environment overrides into v.
//
// Every struct-tag env decode in this package goes through here. Two call sites
// read the environment — the main config and per-channel settings — and a
// canonical variable that worked in one but not the other would be worse than
// one that worked in neither, because it would look supported.
func parseEnv(v any) error {
	return env.ParseWithOptions(v, env.Options{Environment: parserEnvironment()})
}
