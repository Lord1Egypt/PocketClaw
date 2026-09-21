package common

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Conversation identity for gateways that route on it.
//
// PC-DEF-032. OpenCode Go refuses a request that does not say which
// conversation it belongs to:
//
//	HTTP 400 {"type":"error","error":{"type":"MissingSessionID",
//	 "message":"Error from provider (Console Go): Request is missing
//	 x-opencode-session and cannot be routed efficiently..."}}
//
// The same request with an `x-opencode-session` header succeeds. Owner-verified
// against the live service on 2026-09-13.
//
// Two properties matter and pull in opposite directions:
//
//   - Stability. Every request in one conversation must carry the same value —
//     each turn, each streamed turn, each tool-call continuation, each retry
//     and each follow-up — or the gateway cannot route them together. A value
//     generated per request would satisfy the header and defeat its purpose.
//   - Disclosure. A PocketClaw session key is not anonymous. It carries the
//     channel and the chat or user id, and on Telegram that is the owner's own
//     account. Sending it to a third party would leak an identity the product
//     otherwise keeps to itself.
//
// So the key is never sent. It is mapped through a process-lifetime random salt
// into an opaque, UUID-shaped value: stable for as long as the gateway runs,
// unlinkable to the conversation by anyone who does not hold the salt, and
// carrying nothing of the user's. A restarted gateway re-salts, so a
// conversation that spans a restart continues under a new id — a routing
// inefficiency on the provider's side, never an error.
//
// No credential is an input. The salt is random, the scope is a session key,
// and an API key is neither.

// SessionOptionKey is where the agent puts the conversation scope on the
// per-request options map. It is read here and never placed in a request body.
const SessionOptionKey = "session_key"

// sessionSalt is generated once per process and never leaves it.
var sessionSalt = sync.OnceValue(func() []byte {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err == nil {
		return salt
	}
	// Randomness is unavailable. A derived salt is weaker but still not the
	// scope itself, which is the property that actually matters here: the
	// alternative of hashing the session key unsalted would put a guessable
	// preimage of the user's chat id on the wire.
	return []byte(fmt.Sprintf("pocketclaw-session-salt-%d-%d",
		os.Getpid(), time.Now().UnixNano()))
})

// StableSessionID maps a conversation scope to its opaque identifier.
//
// An empty scope is legitimate: summarisation and context compaction call a
// provider outside any one conversation. They get the process's own identifier
// rather than no header at all, because "no header" is the 400 this exists to
// prevent.
func StableSessionID(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "pocketclaw:process"
	}

	mac := hmac.New(sha256.New, sessionSalt())
	mac.Write([]byte(scope))
	sum := mac.Sum(nil)

	// UUID-shaped, because that is the form the gateway is documented and
	// observed to accept. Version 4 and the RFC 4122 variant are stamped so the
	// value parses as a UUID; it is derived, not random, which no consumer of
	// an opaque routing token can tell or needs to.
	sum[6] = (sum[6] & 0x0f) | 0x40
	sum[8] = (sum[8] & 0x3f) | 0x80

	hexed := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hexed[0:8], hexed[8:12], hexed[12:16], hexed[16:20], hexed[20:32])
}

// SessionIDFromOptions reads the conversation scope off a request's options.
func SessionIDFromOptions(options map[string]any) string {
	scope, _ := options[SessionOptionKey].(string)
	return StableSessionID(scope)
}

// ApplySessionHeader sets the conversation header, when the provider has one.
//
// A provider with no session header configured is untouched: this is opt-in per
// provider, so a header one gateway requires is never sent to a service that
// did not ask for it.
func ApplySessionHeader(req *http.Request, header string, options map[string]any) {
	header = strings.TrimSpace(header)
	if header == "" || req == nil {
		return
	}
	req.Header.Set(header, SessionIDFromOptions(options))
}
