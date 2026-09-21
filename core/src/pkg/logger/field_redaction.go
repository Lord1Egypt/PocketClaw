package logger

import (
	"encoding/json"
	"strings"
)

// Central, recursive redaction for structured log fields.
//
// The rule this enforces is that no call site has to remember to redact. Before
// this existed, sanitizeFieldsForLog matched a short list of exact field names
// and pattern-redacted string values, and its default branch handed every other
// type straight to the encoder. So a secret survived in two shapes:
//
//   - a field *named* for a credential whose value no pattern recognises --
//     `token: "hunter2"` is not `sk-...`, has no vendor prefix, and is not
//     `KEY=value`, so every pattern declined it;
//   - a secret nested inside a map, slice or struct, which never reached the
//     string case at all.
//
// Both are closed here: names are classified, and any non-primitive is walked.
// This narrows what is logged and never widens it -- every existing redaction
// still runs.

// secretFieldNameParts are the substrings that mark a field name as holding a
// credential *value*. Substrings rather than exact names, because credentials
// arrive under compound names nobody can enumerate in advance: `bot_token`,
// `x_opencode_session`, `proxy_password`, `channel_access_token`,
// `crypto_passphrase`, `nickserv_password`.
var secretFieldNameParts = []string{
	"api_key", "apikey", "api_keys",
	"authorization", "auth_token", "authtoken",
	"token", "secret", "password", "passwd", "passphrase",
	"credential", "cookie", "private_key", "privatekey",
	"recovery_code", "client_secret", "encrypt_key",
	"access_key", "bearer",
	// Bare "session" too. A session identifier is credential-adjacent and this
	// codebase already treats every one it names as sensitive -- session_key,
	// scope_key and route_main_session were redacted before this file existed --
	// and the custom header `x-opencode-session` is a secret header value. The
	// facts *about* a session survive through the suffix rule
	// (`session_present`, `session_count`) and `session_id` is marked
	// `<internal>` before this check runs.
	"session",
}

// safeFieldNames are names that match a secret part but are metadata, not a
// credential. Kept deliberately short: a broad allowlist would quietly reopen
// the hole this file closes.
//
//   - auth_method names a method ("oauth", "token"), and is the field that makes
//     a provider failure diagnosable.
//   - changed_fields is a list of field *names*, which is the whole point of
//     logging a configuration change without its values.
var safeFieldNames = map[string]struct{}{
	"auth_method": {},
	// Lists and names of fields, never their values.
	"changed_fields":   {},
	"max_tokens_field": {},
	"token_type":       {},
	"secret_count":     {},
}

// safeFieldSuffixes mark a field as a fact *about* a credential rather than the
// credential. `authorization_present=true` and `api_key_changed=true` are
// exactly what DEBUG is supposed to carry.
var safeFieldSuffixes = []string{
	"_present", "_set", "_configured", "_changed", "_change", "_required",
	"_valid", "_count", "_length", "_len", "_type", "_kind",
	"_percent", "_migrated",
}

// safeFieldPrefixes mark a name as a question about a credential rather than the
// credential: `has_api_key` is a fact, not a key.
var safeFieldPrefixes = []string{"has_", "is_", "any_"}

// tokenMetricNames are the exact names that carry a token *measurement*.
//
// PC-DEF-057 follow-up. `token` is a substring of every credential name worth
// redacting and of every usage metric worth keeping, and the physical DEBUG log
// showed the cost of resolving that the wrong way: `max_tokens=<redacted>`, which
// is a model parameter, not a secret.
//
// Enumerated rather than pattern-matched because the two families genuinely
// overlap. `tokens` is the clearest case: as a log field it is a count
// (pkg/seahorse), and as a struct field it is a map of credentials
// (pkg/channels/weixin, pkg/providers/cli) -- so the name alone cannot decide it
// and the value's type has to.
var tokenMetricNames = map[string]struct{}{
	"tokens": {}, "max_tokens": {}, "min_tokens": {},
	"max_completion_tokens": {},
	"prompt_tokens":         {}, "completion_tokens": {}, "total_tokens": {},
	"reasoning_tokens": {}, "thinking_tokens": {}, "context_tokens": {},
	"input_tokens": {}, "output_tokens": {},
	"input_tokens_details": {}, "output_tokens_details": {},
	"cached_tokens": {}, "cached_input_tokens": {},
	"cache_creation_input_tokens": {}, "cache_read_input_tokens": {},
	"tokens_after": {}, "tokens_before": {}, "tokens_used": {}, "used_tokens": {},
	"history_tokens": {}, "fresh_tail_tokens": {}, "original_fresh_tokens": {},
	"compress_at_tokens": {}, "summarize_at_tokens": {},
	"tokens_per_second": {},
}

// The markers. Spelled once so every surface redacts identically, and not a
// fixed-width mask, so a reader cannot infer a secret's length from the log.
const (
	redactedFieldMarker = "<redacted>"
	internalFieldMarker = "<internal>"
)

// maxLogFieldDepth bounds the walk. A cycle cannot occur in JSON-decoded data,
// but deeply nested input should not turn a log call into a traversal.
const maxLogFieldDepth = 8

// normalizeFieldName makes `X-Api-Key`, `x_api_key` and `xApiKey` compare equal.
func normalizeFieldName(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range key {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r == '-' || r == ' ' || r == '.':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// fieldIsSafeMetadata reports whether a field is a measurement or a fact about a
// credential rather than the credential itself.
//
// Checked before the secret-name match, so explicit safe semantics win over the
// broad substring rule. The value is consulted only where the name genuinely
// cannot decide -- see tokenMetricNames.
func fieldIsSafeMetadata(key string, value any) bool {
	// A bool cannot carry a credential whatever it is called. Stated here as well
	// as in sanitizeLogValue so the predicate and the sanitizer cannot disagree.
	if _, isBool := value.(bool); isBool {
		return true
	}

	name := normalizeFieldName(key)

	if _, safe := safeFieldNames[name]; safe {
		return true
	}
	for _, suffix := range safeFieldSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	for _, prefix := range safeFieldPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	if _, metric := tokenMetricNames[name]; metric {
		// A counted token is a number or a breakdown of numbers. A credential
		// under one of these names -- weixin's `tokens` map, the codex CLI's
		// `tokens` struct -- is a string or a map of strings, and stays secret.
		return isNumericOrNumericContainer(value)
	}
	// The generic shape, for a metric name not yet enumerated. Numeric only, for
	// the same reason.
	if strings.HasSuffix(name, "_tokens") ||
		strings.HasSuffix(name, "_token_count") ||
		strings.HasSuffix(name, "_token_percent") {
		return isNumeric(value)
	}
	return false
}

// isNumeric reports whether value is a number, including the float64 a JSON
// round trip produces.
func isNumeric(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	}
	return false
}

// isNumericOrNumericContainer also accepts a breakdown such as
// `input_tokens_details`, whose members are themselves walked by key.
func isNumericOrNumericContainer(value any) bool {
	if isNumeric(value) {
		return true
	}
	switch typed := value.(type) {
	case map[string]any:
		for _, v := range typed {
			if !isNumeric(v) {
				return false
			}
		}
		return true
	case map[string]int:
		return true
	}
	return false
}

// fieldNameHoldsSecret reports whether a field name denotes a credential value.
//
// Name-only. Callers that have the value should use fieldIsSafeMetadata first.
func fieldNameHoldsSecret(key string) bool {
	name := normalizeFieldName(key)
	for _, part := range secretFieldNameParts {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

// fieldIsSecret is the decision the sanitizer uses: safe metadata wins, then the
// credential-name rule.
func fieldIsSecret(key string, value any) bool {
	if fieldIsSafeMetadata(key, value) {
		return false
	}
	return fieldNameHoldsSecret(key)
}

// sanitizeLogValue returns value with credential material removed, walking
// maps, slices and anything else JSON can describe.
//
// A bool is returned untouched whatever its name: a boolean cannot carry a
// secret, and `authorization_present=true` is a fact worth keeping.
func sanitizeLogValue(key string, value any, depth int) any {
	if depth > maxLogFieldDepth {
		return "<truncated>"
	}

	switch typed := value.(type) {
	case nil:
		return nil
	case bool:
		return typed
	case string:
		if fieldIsSecret(key, typed) {
			return redactedFieldMarker
		}
		return redactSecrets(typed)
	case error:
		if typed == nil {
			return nil
		}
		return redactSecrets(typed.Error())
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		// A numeric field named for a credential is still one: a Telegram owner
		// id arrives as an int. A token *metric* is not, which is what
		// fieldIsSafeMetadata separates.
		if fieldIsSecret(key, value) {
			return redactedFieldMarker
		}
		return value
	case map[string]any:
		if fieldIsSecret(key, typed) {
			return redactedFieldMarker
		}
		return sanitizeLogMap(typed, depth)
	case map[string]string:
		if fieldIsSecret(key, typed) {
			return redactedFieldMarker
		}
		safe := make(map[string]any, len(typed))
		for k, v := range typed {
			safe[k] = sanitizeLogValue(k, v, depth+1)
		}
		return safe
	case []string:
		if fieldIsSecret(key, typed) {
			return redactedFieldMarker
		}
		safe := make([]any, 0, len(typed))
		for _, v := range typed {
			safe = append(safe, redactSecrets(v))
		}
		return safe
	case []any:
		if fieldIsSecret(key, typed) {
			return redactedFieldMarker
		}
		safe := make([]any, 0, len(typed))
		for _, v := range typed {
			// Elements inherit the field's name, so a []map with a `token` key
			// inside is still walked by key.
			safe = append(safe, sanitizeLogValue(key, v, depth+1))
		}
		return safe
	}

	// Anything else -- a struct, a pointer to one, a typed slice, a custom
	// string type. A field named for a credential is dropped outright rather
	// than inspected.
	if fieldIsSecret(key, value) {
		return redactedFieldMarker
	}
	return sanitizeViaJSON(key, value, depth)
}

// sanitizeViaJSON walks a value by the shape JSON gives it.
//
// Round-tripping rather than reflecting keeps one code path for structs, nested
// structs and typed collections, and means the keys walked are exactly the keys
// the encoder would have written -- so what is checked is what would have been
// logged.
func sanitizeViaJSON(key string, value any, depth int) any {
	encoded, err := json.Marshal(value)
	if err != nil {
		// Unserializable, so the encoder could not have written it either.
		return "<unserializable>"
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return redactSecrets(string(encoded))
	}
	switch typed := decoded.(type) {
	case map[string]any:
		return sanitizeLogMap(typed, depth)
	case []any:
		return sanitizeLogValue(key, typed, depth)
	case string:
		return redactSecrets(typed)
	default:
		return decoded
	}
}

func sanitizeLogMap(fields map[string]any, depth int) map[string]any {
	safe := make(map[string]any, len(fields))
	for key, value := range fields {
		// The omit and internal-id rules apply at every level, not only the top.
		if _, omit := rawContentLogFields[normalizeFieldName(key)]; omit {
			continue
		}
		if _, internal := internalLogIDFields[normalizeFieldName(key)]; internal {
			safe[key] = internalFieldMarker
			continue
		}
		safe[key] = sanitizeLogValue(key, value, depth+1)
	}
	return safe
}
