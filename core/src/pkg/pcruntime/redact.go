package pcruntime

import (
	"regexp"
	"strings"
)

// redactedMarker is what replaces a secret. It is deliberately not a fixed-width
// mask, so a reader cannot infer the secret's length from the log.
const redactedMarker = "<redacted>"

var (
	// Header-style credentials, whether passed as one argv element
	// (-H "Authorization: Bearer X") or embedded in a longer string. The value
	// runs to the closing quote, a comma, or the end of the string: a header
	// value contains spaces ("Bearer <token>"), so stopping at whitespace would
	// redact the scheme and leave the credential.
	authorizationHeaderPattern = regexp.MustCompile(
		`(?i)\b(authorization|proxy-authorization|cookie|set-cookie|x-api-key|x-auth-token)(\s*:\s*)[^"',]+`,
	)
	// Bearer/Basic credentials that appear without a header name.
	bearerPattern = regexp.MustCompile(`(?i)\b(bearer|basic|token)(\s+)[A-Za-z0-9._~+/=-]{8,}`)
	// Telegram bot tokens, bare and in api.telegram.org URLs.
	telegramTokenPattern    = regexp.MustCompile(`\b\d{6,12}:[A-Za-z0-9_-]{30,}`)
	telegramBotURLPattern   = regexp.MustCompile(`(?i)/bot\d{6,12}:[A-Za-z0-9_-]{30,}`)
	githubTokenPattern      = regexp.MustCompile(`\b(gh[pousr]_[A-Za-z0-9]{16,}|github_pat_[A-Za-z0-9_]{20,})`)
	providerAPIKeyPattern   = regexp.MustCompile(`\b(sk-[A-Za-z0-9_-]{16,}|AIza[A-Za-z0-9_-]{30,}|xox[baprs]-[A-Za-z0-9-]{10,})`)
	urlUserinfoPattern      = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^/\s:@]+:[^/\s@]+@`)
	secretEnvKeyPattern     = regexp.MustCompile(`(?i)(token|secret|password|passwd|credential|api[_-]?key|auth|cookie|session|private[_-]?key)`)
	secretAssignmentPattern = regexp.MustCompile(
		`(?i)\b([A-Za-z_][A-Za-z0-9_]*(?:TOKEN|SECRET|PASSWORD|PASSWD|CREDENTIAL|API_?KEY|AUTH|COOKIE|SESSION)[A-Za-z0-9_]*)=\S+`,
	)
)

// secretBearingFlags maps a tool to the flags whose *following* argv element is
// a credential. It is deliberately per-tool rather than global: a password is
// often an ordinary-looking word that no pattern can recognise, but the same
// short flags mean something harmless elsewhere. Redacting `-u` globally would
// blank the filename in `sort -u notes.txt`, and `-E` would blank the pattern in
// `grep -E '<expr>' file`, destroying the diagnostics these logs exist for.
//
// A tool absent from this map gets pattern redaction only.
var secretBearingFlags = map[string]map[string]struct{}{
	"curl": {
		"-H": {}, "--header": {},
		"-u": {}, "--user": {},
		"-E": {}, "--cert": {},
		"-b": {}, "--cookie": {},
		"--oauth2-bearer": {},
		"--proxy-user":    {},
	},
	"wget": {
		"--header": {}, "--password": {}, "--user": {},
		"--http-password": {}, "--http-user": {},
		"--proxy-password": {}, "--proxy-user": {},
	},
	"gh": {
		"--with-token": {}, "--token": {},
	},
	"git": {
		"-c": {}, "--config": {},
	},
}

// RedactText removes credential material from a free-text string. It is applied
// before persistence, never to the value handed back to the caller in memory.
func RedactText(s string) string {
	if s == "" {
		return s
	}
	s = urlUserinfoPattern.ReplaceAllString(s, "${1}"+redactedMarker+"@")
	s = telegramBotURLPattern.ReplaceAllString(s, "/bot"+redactedMarker)
	s = telegramTokenPattern.ReplaceAllString(s, redactedMarker)
	s = githubTokenPattern.ReplaceAllString(s, redactedMarker)
	s = providerAPIKeyPattern.ReplaceAllString(s, redactedMarker)
	s = secretAssignmentPattern.ReplaceAllString(s, "${1}="+redactedMarker)
	s = authorizationHeaderPattern.ReplaceAllString(s, "${1}${2}"+redactedMarker)
	s = bearerPattern.ReplaceAllString(s, "${1}${2}"+redactedMarker)
	return s
}

// RedactArgv redacts an argument vector for logging. Flag-directed redaction for
// the named tool runs first, so a credential that no pattern would recognise is
// still removed; every remaining element then goes through the text patterns.
func RedactArgv(toolID string, argv []string) []string {
	if len(argv) == 0 {
		return nil
	}
	flags := secretBearingFlags[toolID]
	out := make([]string, len(argv))
	redactNext := false
	for i, arg := range argv {
		if redactNext {
			out[i] = redactedMarker
			redactNext = false
			continue
		}
		out[i] = RedactText(arg)

		if len(flags) == 0 {
			continue
		}
		// --flag=value carries the credential in the same element.
		if equals := strings.IndexByte(arg, '='); equals > 0 {
			if _, secret := flags[arg[:equals]]; secret {
				out[i] = arg[:equals] + "=" + redactedMarker
				continue
			}
		}
		if _, secret := flags[arg]; secret {
			redactNext = true
		}
	}
	return out
}

// RedactEnv redacts an environment map for logging. A value is dropped whenever
// its *key* looks secret, because an API key is indistinguishable from any other
// opaque string once it is separated from its name.
func RedactEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	out := make(map[string]string, len(env))
	for key, value := range env {
		if secretEnvKeyPattern.MatchString(key) {
			out[key] = redactedMarker
			continue
		}
		out[key] = RedactText(value)
	}
	return out
}

// redactFieldValue is the single redaction gate for structured log fields.
func redactFieldValue(key string, value any) any {
	if secretEnvKeyPattern.MatchString(key) {
		return redactedMarker
	}
	switch typed := value.(type) {
	case string:
		return RedactText(typed)
	case []string:
		out := make([]string, len(typed))
		for i, item := range typed {
			out[i] = RedactText(item)
		}
		return out
	case error:
		return RedactText(typed.Error())
	case map[string]string:
		return RedactEnv(typed)
	default:
		return value
	}
}
