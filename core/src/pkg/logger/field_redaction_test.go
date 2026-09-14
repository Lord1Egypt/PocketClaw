package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// The logging contract: DEBUG stays detailed enough to reconstruct a failure,
// and safe enough that a user can send the log to a developer.
//
// Every value here is fabricated. The canary is what a real credential would
// occupy: a value no pattern can recognise, so only the *field name* can save it.
const canary = "POCKETCLAW_TEST_SECRET_123456"

// sanitize is what the logger does to a field map before any writer sees it.
//
// HTML escaping is off so the markers read as `<redacted>` rather than
// `\u003credacted\u003e`; the assertions are about content, not encoding.
func sanitize(t *testing.T, fields map[string]any) string {
	t.Helper()
	safe := sanitizeFieldsForLog(fields)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(safe); err != nil {
		t.Fatalf("marshalling sanitized fields: %v", err)
	}
	return buf.String()
}

func assertNoCanary(t *testing.T, where, output string) {
	t.Helper()
	if strings.Contains(output, canary) {
		t.Errorf("%s leaked the canary:\n%s", where, output)
	}
}

// The owner's list, each through the field name that carries it in practice.
func TestCanaryNeverSurvivesAFieldNamedForACredential(t *testing.T) {
	cases := map[string]string{
		"model API key":         "api_key",
		"multi-key list":        "api_keys",
		"authorization header":  "authorization",
		"telegram bot token":    "bot_token",
		"slack app token":       "app_token",
		"oauth access token":    "access_token",
		"oauth refresh token":   "refresh_token",
		"password":              "password",
		"proxy password":        "proxy_password",
		"generic secret":        "secret",
		"client secret":         "client_secret",
		"cookie":                "cookie",
		"set-cookie":            "set-cookie",
		"private key":           "private_key",
		"matrix passphrase":     "crypto_passphrase",
		"irc nickserv password": "nickserv_password",
		"line channel token":    "channel_access_token",
		"feishu encrypt key":    "encrypt_key",
		"custom secret header":  "x_opencode_session",
		"web-search key":        "brave_api_key",
		"recovery code":         "recovery_code",
		"header-cased api key":  "X-Api-Key",
		"camel-cased api key":   "apiKey",
	}

	for description, field := range cases {
		t.Run(description, func(t *testing.T) {
			output := sanitize(t, map[string]any{field: canary})
			assertNoCanary(t, "field "+field, output)
			if !strings.Contains(output, "<redacted>") {
				t.Errorf("field %q was neither redacted nor removed: %s", field, output)
			}
		})
	}
}

// The hole the central layer closes: a secret one or more levels down.
func TestCanaryNeverSurvivesNesting(t *testing.T) {
	cases := map[string]map[string]any{
		"nested map": {
			"settings": map[string]any{"token": canary},
		},
		"twice-nested map": {
			"config": map[string]any{
				"channel": map[string]any{"bot_token": canary},
			},
		},
		"map of headers": {
			"custom_headers": map[string]string{"authorization": "Bearer " + canary},
		},
		"slice of maps": {
			"models": []any{
				map[string]any{"model": "gpt-4o", "api_key": canary},
			},
		},
		"struct": {
			"model": struct {
				Name   string `json:"name"`
				APIKey string `json:"api_key"`
			}{Name: "gpt-4o", APIKey: canary},
		},
		"pointer to struct": {
			"model": &struct {
				APIKey string `json:"api_key"`
			}{APIKey: canary},
		},
		"struct inside a map": {
			"providers": map[string]any{
				"opencode": struct {
					Token string `json:"token"`
				}{Token: canary},
			},
		},
		"slice of strings under a secret name": {
			"api_keys": []string{canary, canary},
		},
	}

	for description, fields := range cases {
		t.Run(description, func(t *testing.T) {
			assertNoCanary(t, description, sanitize(t, fields))
		})
	}
}

// A credential-shaped value must still be caught even under an innocent name,
// so the name rule and the pattern rule both hold.
func TestPatternRedactionStillAppliesUnderAnInnocentFieldName(t *testing.T) {
	const shaped = "sk-abcdefghijklmnopqrstuvwxyz0123"

	output := sanitize(t, map[string]any{
		"detail": "provider rejected key " + shaped,
		"nested": map[string]any{"note": "used " + shaped},
	})

	if strings.Contains(output, shaped) {
		t.Errorf("a credential-shaped value survived: %s", output)
	}
}

// The other half of the contract. DEBUG has to stay worth reading.
func TestUsefulDebugMetadataSurvivesRedaction(t *testing.T) {
	output := sanitize(t, map[string]any{
		"component":                  "provider.request",
		"provider":                   "opencode_go",
		"model":                      "deepseek-v4.1-flash",
		"protocol":                   "chat_completions",
		"method":                     "POST",
		"endpoint":                   "https://opencode.ai/zen/go/v1/chat/completions",
		"status":                     429,
		"duration_ms":                1234,
		"attempt":                    2,
		"stream":                     false,
		"messages":                   6,
		"tools":                      19,
		"authorization_present":      true,
		"x_opencode_session_present": true,
		"api_key_changed":            true,
		"auth_method":                "oauth",
		"changed_fields":             []string{"api_base", "api_key", "custom_headers"},
		"trace_id":                   "trc-0f1e2d",
		"error_class":                "rate_limited",
		"registered":                 14,
	})

	for _, expected := range []string{
		"provider.request", "opencode_go", "deepseek-v4.1-flash",
		"chat_completions", "POST", "opencode.ai/zen/go/v1/chat/completions",
		"429", "1234", "rate_limited", "trc-0f1e2d", "oauth",
		"api_base", "custom_headers", "14",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("DEBUG lost %q, which is what makes the log reconstructable:\n%s",
				expected, output)
		}
	}

	// The presence booleans are the whole point of logging them.
	for _, boolField := range []string{
		`"authorization_present":true`,
		`"x_opencode_session_present":true`,
		`"api_key_changed":true`,
	} {
		if !strings.Contains(output, boolField) {
			t.Errorf("a presence flag was redacted into uselessness: want %s in\n%s",
				boolField, output)
		}
	}
}

// changed_fields is a list of field names, which is exactly how a configuration
// change is logged without its values.
func TestChangedFieldNamesAreKeptWhileValuesAreNot(t *testing.T) {
	output := sanitize(t, map[string]any{
		"changed_fields":  []string{"api_base", "api_key"},
		"api_key_changed": true,
		"api_key":         canary,
	})

	assertNoCanary(t, "config change", output)
	if !strings.Contains(output, "api_base") || !strings.Contains(output, "api_key_changed") {
		t.Errorf("the names and the changed flag must survive: %s", output)
	}
}

// A bool cannot carry a secret, whatever it is called.
func TestBooleansAreNeverRedacted(t *testing.T) {
	output := sanitize(t, map[string]any{
		"token":         true,
		"password":      false,
		"api_key_valid": true,
	})

	for _, expected := range []string{`"token":true`, `"password":false`, `"api_key_valid":true`} {
		if !strings.Contains(output, expected) {
			t.Errorf("want %s in %s", expected, output)
		}
	}
}

// A numeric credential is still a credential: a Telegram owner id is an int.
func TestNumericSecretFieldsAreRedacted(t *testing.T) {
	output := sanitize(t, map[string]any{"session_token": 123456789})

	if strings.Contains(output, "123456789") {
		t.Errorf("a numeric credential survived: %s", output)
	}
}

// Errors are pattern-redacted, and a nil error must not panic.
func TestErrorValuesAreRedactedAndNilIsSafe(t *testing.T) {
	output := sanitize(t, map[string]any{
		"error":     errors.New("auth failed: Bearer sk-abcdefghijklmnopqrstuvwx"),
		"nil_error": error(nil),
		"nil_value": nil,
	})

	if strings.Contains(output, "sk-abcdefghijklmnopqrstuvwx") {
		t.Errorf("an error leaked a credential: %s", output)
	}
}

// The pre-existing rules must not have been weakened by adding the new ones.
func TestExistingOmitAndInternalRulesStillApply(t *testing.T) {
	output := sanitize(t, map[string]any{
		"prompt":      "the user's private question",
		"content":     "a private message body",
		"chat_id":     "-1001234567890",
		"sender_id":   "987654321",
		"session_key": "telegram:-100123",
	})

	for _, forbidden := range []string{
		"the user's private question", "a private message body",
		"-1001234567890", "987654321", "telegram:-100123",
	} {
		if strings.Contains(output, forbidden) {
			t.Errorf("an existing rule regressed, %q survived: %s", forbidden, output)
		}
	}
	if !strings.Contains(output, "<internal>") {
		t.Errorf("internal ids must be marked, not dropped silently: %s", output)
	}
}

// Omit and internal rules have to hold at every depth, not only the top level.
func TestOmitAndInternalRulesApplyWhenNested(t *testing.T) {
	output := sanitize(t, map[string]any{
		"turn": map[string]any{
			"prompt":  "the user's private question",
			"chat_id": "-1001234567890",
			"model":   "gpt-4o",
		},
	})

	if strings.Contains(output, "the user's private question") {
		t.Errorf("nested raw content survived: %s", output)
	}
	if strings.Contains(output, "-1001234567890") {
		t.Errorf("a nested chat id survived: %s", output)
	}
	if !strings.Contains(output, "gpt-4o") {
		t.Errorf("nested metadata must survive: %s", output)
	}
}

// Deep nesting must be bounded rather than walked forever.
func TestDeepNestingIsBounded(t *testing.T) {
	deepest := map[string]any{"token": canary}
	current := deepest
	for i := 0; i < maxLogFieldDepth+6; i++ {
		current = map[string]any{"level": current}
	}

	output := sanitize(t, map[string]any{"root": current})
	assertNoCanary(t, "deeply nested", output)
	if !strings.Contains(output, "<truncated>") {
		t.Errorf("the walk must stop and say so: %s", output)
	}
}

// The combined decision, which is what the sanitizer uses: safe metadata is
// checked before the credential-name rule.
func TestFieldClassification(t *testing.T) {
	secret := []string{
		"api_key", "API_KEY", "X-Api-Key", "apiKey", "bot_token",
		"authorization", "Proxy-Password", "crypto_passphrase", "cookie",
		// A digest or hash of a secret is a verifier and offline-crackable, so
		// the "facts about a credential" allowance deliberately stops short of
		// it. Nothing in this codebase logs one.
		"api_key_digest", "password_hash",
	}
	safe := []string{
		"auth_method", "authorization_present", "api_key_changed",
		"token_count", "secret_count", "provider", "model",
		"endpoint", "status", "duration_ms", "changed_fields", "token_type",
		"max_tokens", "prompt_tokens", "max_tokens_field",
	}

	for _, key := range secret {
		if !fieldIsSecret(key, "some-value") {
			t.Errorf("%q must be treated as a credential", key)
		}
	}
	for _, key := range safe {
		// Numeric where the name is a metric, so the value can be consulted.
		var value any = "some-value"
		switch key {
		case "token_count", "secret_count", "status", "duration_ms",
			"max_tokens", "prompt_tokens":
			value = 1
		}
		if fieldIsSecret(key, value) {
			t.Errorf("%q is metadata and must survive", key)
		}
	}
}

// An unserializable value must not be logged as a Go struct dump either.
func TestUnserializableValuesAreReplacedRatherThanDumped(t *testing.T) {
	output := sanitize(t, map[string]any{"fn": func() {}})

	if !strings.Contains(output, "<unserializable>") {
		t.Errorf("want <unserializable> in %s", output)
	}
}

// End to end through the real emit path, not just the sanitizer in isolation.
//
// sanitizeFieldsForLog being correct is only half the claim: it has to be what
// the writers actually receive. This drives logMessage and reads the bytes.
func TestCanaryNeverReachesTheWriterThroughAnyLevel(t *testing.T) {
	var captured bytes.Buffer
	restore := swapLoggerOutput(t, &captured)
	defer restore()

	fields := map[string]any{
		"api_key":        canary,
		"bot_token":      canary,
		"settings":       map[string]any{"authorization": "Bearer " + canary},
		"custom_headers": map[string]string{"x-opencode-session": canary},
		"provider":       "opencode_go",
		"status":         429,
	}

	for _, emit := range []struct {
		name string
		call func()
	}{
		{"DEBUG", func() { DebugCF("test", "probe", cloneFields(fields)) }},
		{"INFO", func() { InfoCF("test", "probe", cloneFields(fields)) }},
		{"WARN", func() { WarnCF("test", "probe", cloneFields(fields)) }},
		{"ERROR", func() { ErrorCF("test", "probe", cloneFields(fields)) }},
	} {
		captured.Reset()
		emit.call()
		output := captured.String()
		assertNoCanary(t, emit.name+" writer output", output)
		if !strings.Contains(output, "opencode_go") {
			t.Errorf("%s lost its useful metadata: %s", emit.name, output)
		}
	}
}

func cloneFields(fields map[string]any) map[string]any {
	copied := make(map[string]any, len(fields))
	for k, v := range fields {
		copied[k] = v
	}
	return copied
}

// swapLoggerOutput points the package logger at a buffer and lowers the level so
// DEBUG is emitted, restoring both afterwards.
func swapLoggerOutput(t *testing.T, sink *bytes.Buffer) func() {
	t.Helper()
	previousLogger := logger
	previousLevel := currentLevel

	previousGlobal := zerolog.GlobalLevel()

	logger = zerolog.New(sink).With().Timestamp().Logger()
	// SetLevel, not a direct assignment: zerolog gates on its own global level,
	// so setting only the package's currentLevel leaves DEBUG discarded and the
	// test silently passes on empty output.
	SetLevel(DEBUG)

	return func() {
		logger = previousLogger
		currentLevel = previousLevel
		zerolog.SetGlobalLevel(previousGlobal)
	}
}

// PC-DEF-057 follow-up. The physical DEBUG log showed `max_tokens=<redacted>`:
// `token` is a substring of every credential worth hiding and of every usage
// metric worth keeping, and resolving that by substring alone lost the metric.
//
// Both halves are asserted together, because fixing one by breaking the other is
// the failure mode here.
func TestTokenMetricsStayVisibleWhileTokenCredentialsDoNot(t *testing.T) {
	t.Run("metrics survive", func(t *testing.T) {
		output := sanitize(t, map[string]any{
			"max_tokens":              32768,
			"prompt_tokens":           123,
			"completion_tokens":       45,
			"total_tokens":            168,
			"reasoning_tokens":        12,
			"cached_tokens":           99,
			"input_tokens":            100,
			"output_tokens":           68,
			"tokens":                  168,
			"tokens_after":            140,
			"tokens_before":           168,
			"used_tokens":             168,
			"token_count":             168,
			"prompt_token_count":      123,
			"completion_token_count":  45,
			"summarize_token_percent": 75,
			"max_tokens_field":        "max_completion_tokens",
		})

		for _, expected := range []string{
			`"max_tokens":32768`, `"prompt_tokens":123`, `"completion_tokens":45`,
			`"total_tokens":168`, `"reasoning_tokens":12`, `"cached_tokens":99`,
			`"input_tokens":100`, `"output_tokens":68`, `"tokens":168`,
			`"tokens_after":140`, `"tokens_before":168`, `"used_tokens":168`,
			`"token_count":168`, `"prompt_token_count":123`,
			`"completion_token_count":45`, `"summarize_token_percent":75`,
			`"max_tokens_field":"max_completion_tokens"`,
		} {
			if !strings.Contains(output, expected) {
				t.Errorf("a token metric was redacted into uselessness: want %s in\n%s",
					expected, output)
			}
		}
		if strings.Contains(output, "<redacted>") {
			t.Errorf("no token metric should be redacted:\n%s", output)
		}
	})

	t.Run("credentials stay redacted", func(t *testing.T) {
		for _, field := range []string{
			"token", "api_token", "bot_token", "access_token", "refresh_token",
			"oauth_token", "id_token", "auth_token", "session_token",
			"launcher_token", "app_token", "channel_access_token",
			"context_token", "reply_token", "authorization", "api_key",
			"password", "secret",
		} {
			output := sanitize(t, map[string]any{field: canary})
			assertNoCanary(t, "credential field "+field, output)
			if !strings.Contains(output, "<redacted>") {
				t.Errorf("field %q must be redacted: %s", field, output)
			}
		}
	})

	// The name alone cannot decide `tokens`: a count in a log, a credential map
	// in weixin state and the codex CLI. The value's type decides.
	t.Run("a tokens map of credentials is still redacted", func(t *testing.T) {
		output := sanitize(t, map[string]any{
			"tokens": map[string]string{"access": canary, "refresh": canary},
		})
		assertNoCanary(t, "tokens credential map", output)
	})

	t.Run("a tokens breakdown of numbers survives", func(t *testing.T) {
		output := sanitize(t, map[string]any{
			"input_tokens_details": map[string]any{"cached_tokens": 64, "audio_tokens": 0},
		})
		if !strings.Contains(output, "64") {
			t.Errorf("a numeric token breakdown must survive: %s", output)
		}
	})

	// A string under a metric name is not a metric.
	t.Run("a string under a metric name is not trusted", func(t *testing.T) {
		output := sanitize(t, map[string]any{"tokens": canary})
		assertNoCanary(t, "string tokens", output)
	})
}

// A hash of a secret is a verifier and is offline-crackable, so the generic
// "facts about a credential" allowance must not extend to it.
func TestAPasswordHashIsStillRedacted(t *testing.T) {
	output := sanitize(t, map[string]any{
		"dashboard_password_hash": canary,
		"password_digest":         canary,
	})

	assertNoCanary(t, "password hash", output)
}

// The real field names this codebase logs, classified. A regression here is a
// fidelity loss the owner would see on the device.
func TestRealLogFieldNamesAreClassifiedCorrectly(t *testing.T) {
	safeWithValue := map[string]any{
		"auth_method":                "oauth",
		"auth_url":                   "https://accounts.example.com/authorize",
		"authenticated":              true,
		"authorization_present":      true,
		"session_header_present":     true,
		"x_opencode_session_present": true,
		"api_key_changed":            true,
		"api_key_valid":              true,
		"has_api_key":                true,
		"has_key":                    true,
		"credential_change":          true,
		"allow_token_query":          false,
		"secret_count":               2,
		"token_type":                 "bearer",
		"sessions_migrated":          7,
		"max_tokens":                 4096,
	}
	for key, value := range safeWithValue {
		if fieldIsSecret(key, value) {
			t.Errorf("%q is metadata and must stay visible", key)
		}
	}

	secret := map[string]any{
		"api_key": "x", "api_keys": []string{"x"}, "app_secret": "x",
		"channel_secret": "x", "client_secret": "x", "nickserv_password": "x",
		"password": "x", "dashboard_password_hash": "x", "secret": "x",
		"cookie": "x", "authorization": "x", "access_token": "x",
		"refresh_token": "x", "id_token": "x", "session_token": "x",
		"bot_token": "x", "app_token": "x", "channel_access_token": "x",
		"launcher_token": "x", "reply_token": "x", "context_token": "x",
		"session": "x",
	}
	for key, value := range secret {
		if !fieldIsSecret(key, value) {
			t.Errorf("%q is a credential and must be redacted", key)
		}
	}

	// These are covered by the logger's exact-name map, which runs before the
	// name rule, so they are asserted through the real path rather than the
	// predicate.
	for _, key := range []string{"session_key", "scope_key", "route_main_session"} {
		output := sanitize(t, map[string]any{key: canary})
		assertNoCanary(t, "exact-name field "+key, output)
	}
}

// PC-DEF-061. The polling lifecycle fields have to survive redaction or the
// instrumentation proves nothing -- the whole point of them is telling
// "Telegram never sent it" apart from "it was received and lost".
//
// They are safe to keep: a Telegram update id is a per-bot sequence number that
// identifies no person, and the identifiers the contract does treat as private
// -- chat_id, sender_id, user_id -- are not logged with them.
func TestPollingObservabilityFieldsSurviveRedaction(t *testing.T) {
	t.Parallel()

	safe := sanitizeFieldsForLog(map[string]any{
		"event":        "polling.update_delivered",
		"update_id":    9001,
		"next_offset":  9002,
		"first_update": true,
	})

	for key, want := range map[string]any{
		"event":        "polling.update_delivered",
		"update_id":    9001,
		"next_offset":  9002,
		"first_update": true,
	} {
		if got := safe[key]; got != want {
			t.Fatalf("field %q = %v, want %v", key, got, want)
		}
	}
}
