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

func TestFieldNameClassification(t *testing.T) {
	secret := []string{
		"api_key", "API_KEY", "X-Api-Key", "apiKey", "bot_token",
		"authorization", "Proxy-Password", "crypto_passphrase", "cookie",
	}
	safe := []string{
		"auth_method", "authorization_present", "api_key_changed",
		"token_count", "secret_count", "api_key_digest", "provider", "model",
		"endpoint", "status", "duration_ms", "changed_fields", "token_type",
	}

	for _, key := range secret {
		if !fieldNameHoldsSecret(key) {
			t.Errorf("%q must be treated as a credential name", key)
		}
	}
	for _, key := range safe {
		if fieldNameHoldsSecret(key) {
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
