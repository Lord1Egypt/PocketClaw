// Package telegramonboarding talks to PocketClaw's managed-bot onboarding service.
//
// PC-DEF-060. The Dashboard opened in a desktop browser has no Android host, so the
// native managed-pairing flow cannot run there. Core runs it instead: the browser calls
// same-origin PocketClaw endpoints, and this package makes the server-to-server calls.
//
// That shape is deliberate. The browser never needs cross-origin access to the hosted
// service, never learns its URL, and never holds the poll token that authorises token
// collection — so nothing security-sensitive is reimplemented in JavaScript.
//
// The wire contract mirrors lib/src/telegram/telegram_onboarding_client.dart field for
// field, because both speak to the same deployment and a drift between them would be a
// pairing that works from one client and not the other.
package telegramonboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrKind classifies a failure so a caller can decide what to tell the user without
// repeating a transport detail that may quote the request.
type ErrKind string

const (
	// KindNotConfigured means this deployment has no onboarding endpoint.
	KindNotConfigured ErrKind = "not_configured"
	// KindNetwork covers anything that never produced an answer.
	KindNetwork ErrKind = "network"
	// KindRateLimited is the service refusing the request rate.
	KindRateLimited ErrKind = "rate_limited"
	// KindPairingGone means expired, unknown, or already delivered.
	KindPairingGone ErrKind = "pairing_gone"
	// KindServiceError is an answer that was not usable.
	KindServiceError ErrKind = "service_error"
)

// Error is a classified onboarding failure.
type Error struct {
	Kind ErrKind
	// Detail is for the developer log. It never carries a credential: the only
	// values interpolated into it are HTTP status codes.
	Detail string
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return string(e.Kind)
	}
	return string(e.Kind) + ": " + e.Detail
}

// AsError extracts a classified failure.
func AsError(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) && target != nil {
		return target, true
	}
	return nil, false
}

// State is where a pairing has got to.
type State string

const (
	StatePending State = "pending"
	StateCreated State = "created"
	StateReady   State = "ready"
	StateExpired State = "expired"
	StateFailed  State = "failed"
)

// ParseState maps a wire value, treating anything unrecognised as failed.
//
// A newer service reporting a state this build has never heard of must not be read as
// success: the pairing would appear to be progressing forever.
func ParseState(raw string) State {
	switch State(strings.TrimSpace(raw)) {
	case StatePending:
		return StatePending
	case StateCreated:
		return StateCreated
	case StateReady:
		return StateReady
	case StateExpired:
		return StateExpired
	default:
		return StateFailed
	}
}

// Pairing is a started session.
//
// PollToken authorises status reads and the single token collection, so it stays on
// this side of the boundary and is never handed to a browser.
type Pairing struct {
	PairingID         string
	PollToken         string
	SuggestedUsername string
	SuggestedName     string
	DeepLink          string
	QRPayload         string
	ExpiresAt         time.Time
	PollInterval      time.Duration
}

// Status is a pairing's state at one moment.
type Status struct {
	State       State
	Reason      string
	BotUsername string
	OwnerUserID int64
}

// Credentials are the child bot's, delivered exactly once by the service.
type Credentials struct {
	Token       string
	BotUserID   int64
	BotUsername string
	OwnerUserID int64
}

// String never renders the token. Overriding it is what stops a stray %v or an error
// interpolation from leaking the user's bot token into a log.
func (c Credentials) String() string {
	return fmt.Sprintf(
		"Credentials(@%s, botUserID: %d, ownerUserID: %d, token: <redacted>)",
		c.BotUsername, c.BotUserID, c.OwnerUserID)
}

// Client calls the onboarding service.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client for baseURL, or nil when there is none configured.
//
// Only an https base URL is accepted. Plain HTTP would put the poll token and, once,
// the bot token on the wire in the clear; refusing is better than downgrading.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: trimmed, http: httpClient}
}

// CreatePairing starts a session.
func (c *Client) CreatePairing(ctx context.Context) (*Pairing, error) {
	if c == nil {
		return nil, &Error{Kind: KindNotConfigured}
	}
	response, body, err := c.send(ctx, http.MethodPost, "/telegram/pairings", "")
	if err != nil {
		return nil, err
	}
	switch response.StatusCode {
	case http.StatusCreated:
	case http.StatusTooManyRequests:
		return nil, &Error{Kind: KindRateLimited}
	default:
		return nil, &Error{
			Kind:   KindServiceError,
			Detail: fmt.Sprintf("create pairing returned %d", response.StatusCode),
		}
	}

	var decoded struct {
		PairingID          string `json:"pairing_id"`
		PollToken          string `json:"poll_token"`
		SuggestedUsername  string `json:"suggested_username"`
		SuggestedName      string `json:"suggested_name"`
		DeepLink           string `json:"deep_link"`
		QRPayload          string `json:"qr_payload"`
		ExpiresAt          string `json:"expires_at"`
		PollIntervalSecond int    `json:"poll_interval_seconds"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, &Error{Kind: KindServiceError, Detail: "malformed response"}
	}
	if decoded.PairingID == "" || decoded.PollToken == "" {
		return nil, &Error{Kind: KindServiceError, Detail: "incomplete response"}
	}
	expires, err := time.Parse(time.RFC3339, decoded.ExpiresAt)
	if err != nil {
		return nil, &Error{Kind: KindServiceError, Detail: "malformed expiry"}
	}
	interval := time.Duration(decoded.PollIntervalSecond) * time.Second
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Pairing{
		PairingID:         decoded.PairingID,
		PollToken:         decoded.PollToken,
		SuggestedUsername: decoded.SuggestedUsername,
		SuggestedName:     decoded.SuggestedName,
		DeepLink:          decoded.DeepLink,
		QRPayload:         decoded.QRPayload,
		ExpiresAt:         expires.UTC(),
		PollInterval:      interval,
	}, nil
}

// FetchStatus reads a pairing's state. It never returns a bot token.
func (c *Client) FetchStatus(ctx context.Context, pairing *Pairing) (*Status, error) {
	if c == nil {
		return nil, &Error{Kind: KindNotConfigured}
	}
	response, body, err := c.send(ctx, http.MethodGet,
		"/telegram/pairings/"+url.PathEscape(pairing.PairingID), pairing.PollToken)
	if err != nil {
		return nil, err
	}
	// The service answers 404 for expired, unknown and wrong-token alike, so it
	// cannot be used to enumerate pairings. All three mean this pairing cannot
	// complete.
	if response.StatusCode == http.StatusNotFound {
		return &Status{State: StateExpired}, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, &Error{
			Kind:   KindServiceError,
			Detail: fmt.Sprintf("poll returned %d", response.StatusCode),
		}
	}

	var decoded struct {
		State       string `json:"state"`
		Reason      string `json:"reason"`
		BotUsername string `json:"bot_username"`
		OwnerUserID *int64 `json:"owner_user_id"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, &Error{Kind: KindServiceError, Detail: "malformed response"}
	}
	status := &Status{
		State:       ParseState(decoded.State),
		Reason:      decoded.Reason,
		BotUsername: decoded.BotUsername,
	}
	if decoded.OwnerUserID != nil {
		status.OwnerUserID = *decoded.OwnerUserID
	}
	return status, nil
}

// CollectCredentials takes the child bot token. The service allows this exactly once.
func (c *Client) CollectCredentials(ctx context.Context, pairing *Pairing) (*Credentials, error) {
	if c == nil {
		return nil, &Error{Kind: KindNotConfigured}
	}
	response, body, err := c.send(ctx, http.MethodPost,
		"/telegram/pairings/"+url.PathEscape(pairing.PairingID)+"/token", pairing.PollToken)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusConflict {
		return nil, &Error{Kind: KindPairingGone}
	}
	if response.StatusCode != http.StatusOK {
		return nil, &Error{
			Kind:   KindServiceError,
			Detail: fmt.Sprintf("token collection returned %d", response.StatusCode),
		}
	}

	var decoded struct {
		BotToken    string `json:"bot_token"`
		BotUserID   int64  `json:"bot_user_id"`
		BotUsername string `json:"bot_username"`
		OwnerUserID int64  `json:"owner_user_id"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, &Error{Kind: KindServiceError, Detail: "malformed response"}
	}
	if decoded.BotToken == "" || decoded.OwnerUserID <= 0 {
		return nil, &Error{Kind: KindServiceError, Detail: "incomplete credentials"}
	}
	return &Credentials{
		Token:       decoded.BotToken,
		BotUserID:   decoded.BotUserID,
		BotUsername: decoded.BotUsername,
		OwnerUserID: decoded.OwnerUserID,
	}, nil
}

// send performs one request, returning the response and its body.
//
// A transport failure is always reported as KindNetwork with no detail: the error text
// can quote the request URL, and no request detail is worth risking in something a
// caller may surface.
func (c *Client) send(
	ctx context.Context,
	method string,
	path string,
	pollToken string,
) (*http.Response, []byte, error) {
	var payload io.Reader
	if method == http.MethodPost {
		payload = bytes.NewReader([]byte{})
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return nil, nil, &Error{Kind: KindNetwork}
	}
	if pollToken != "" {
		request.Header.Set("Authorization", "Bearer "+pollToken)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, nil, &Error{Kind: KindNetwork}
	}
	defer response.Body.Close()
	// Bounded: a pairing response is small, and an unbounded read of an arbitrary
	// endpoint is a memory risk rather than a useful diagnostic.
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return nil, nil, &Error{Kind: KindNetwork}
	}
	return response, body, nil
}
