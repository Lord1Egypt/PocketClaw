package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// TelegramCredentialValidator proves a candidate token before it can replace
// the committed Telegram configuration.
type TelegramCredentialValidator func(
	ctx context.Context,
	token string,
	baseURL string,
	proxy string,
) error

// The candidate outcomes the writer distinguishes. Invalid credentials and a
// bot owned by another service are all terminal for the candidate, but they are
// not the same user-facing problem, so they are separate sentinels rather than
// one "validation failed".
var (
	ErrTelegramCredentialsInvalid = errors.New("telegram credentials invalid")
	ErrTelegramWebhookConflict    = errors.New("telegram webhook conflict")
	ErrTelegramBotInUse           = errors.New("telegram bot already in use")
)

const telegramCredentialValidationTimeout = 10 * time.Second

func (h *Handler) validateTelegramCredentials(
	ctx context.Context,
	token string,
	baseURL string,
	proxy string,
) error {
	validator := h.telegramCredentialValidator
	if validator == nil {
		validator = validateTelegramCredentials
	}
	return validator(ctx, token, baseURL, proxy)
}

// telegramValidationResponse is the parsed envelope of a Bot API call made
// during candidate validation. It never carries the token or the request URL.
type telegramValidationResponse struct {
	StatusCode  int
	OK          bool
	ErrorCode   int
	Result      json.RawMessage
	Description string
}

// validateTelegramCredentials is the replacement transaction's pre-commit gate.
//
// getMe proves only that the token is valid. It does not prove that PocketClaw
// can own the update stream: a bot attached to another service answers getMe
// normally and then refuses getUpdates. So the candidate is proved on three
// facts, and only the last one is allowed to touch the returned configuration:
//
//  1. getMe succeeds -- the token is valid.
//  2. getWebhookInfo shows no active webhook. This is non-destructive: the
//     webhook is never deleted, replaced or otherwise mutated. Taking over
//     somebody else's bot is not PocketClaw's decision.
//  3. a getUpdates probe is accepted. A 409 here means another long poller owns
//     the bot right now.
//
// The probe deliberately carries NO offset. Telegram documents that a negative
// offset retrieves updates from the end of the queue and forgets all earlier
// ones, so it would silently discard exactly the pending first /start that
// PC-DEF-061 exists to preserve. With no offset, an update is returned but not
// confirmed: Telegram confirms an update only when a later getUpdates is called
// with an offset higher than its update_id. This call never advances the offset,
// so every pending update remains available to the real channel Start, which
// polls from an unset offset.
//
// A decisive conflict or 401 rejects the candidate; a transport or unexpected
// error is not treated as a conflict, so a valid token is never refused because
// of a transient network failure. The real getUpdates 409 path remains the
// runtime backstop.
func validateTelegramCredentials(
	ctx context.Context,
	token string,
	baseURL string,
	proxy string,
) error {
	client, apiRoot, err := telegramValidationHTTP(baseURL, proxy)
	if err != nil {
		return errors.New("telegram credential validation unavailable")
	}

	validationCtx, cancel := context.WithTimeout(ctx, telegramCredentialValidationTimeout)
	defer cancel()

	// 1. The token must be valid.
	me, err := telegramValidationCall(validationCtx, client, apiRoot, token, "getMe", nil)
	if err != nil {
		return errors.New("telegram credential validation unavailable")
	}
	if me.decisiveStatus() == http.StatusUnauthorized {
		return ErrTelegramCredentialsInvalid
	}
	if !me.OK {
		return errors.New("telegram credential validation rejected")
	}

	// 2. Non-destructive webhook check.
	webhook, err := telegramValidationCall(validationCtx, client, apiRoot, token, "getWebhookInfo", nil)
	if err == nil && webhook.OK {
		var info struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(webhook.Result, &info) == nil && strings.TrimSpace(info.URL) != "" {
			return ErrTelegramWebhookConflict
		}
	}
	// A webhook check that cannot be read is not a conflict. The getUpdates
	// probe below still catches an active webhook with a 409.

	// 3. Can PocketClaw own the update stream?
	//
	// No offset field: see the function comment. This one short call confirms
	// nothing and drops nothing.
	probeBody := bytes.NewReader([]byte(`{"limit":1,"timeout":0}`))
	probe, err := telegramValidationCall(validationCtx, client, apiRoot, token, "getUpdates", probeBody)
	if err != nil {
		// A transport error is not a conflict; do not reject a valid candidate.
		return nil
	}
	switch probe.decisiveStatus() {
	case http.StatusUnauthorized:
		return ErrTelegramCredentialsInvalid
	case http.StatusConflict:
		// The 409 is authoritative; the subtype is decided by a fresh,
		// non-destructive webhook re-check, not by matching English prose.
		if validationConflictReason(validationCtx, client, apiRoot, token, probe.Description) == "webhook_active" {
			return ErrTelegramWebhookConflict
		}
		return ErrTelegramBotInUse
	}
	return nil
}

// validationConflictReason resolves a getUpdates 409 to a safe subtype. It asks
// Telegram for the current webhook non-destructively: a configured URL means the
// conflict is a webhook; no URL means another long poller owns the bot. The
// description is used only as a fallback when the re-check cannot be read, and
// never as the primary signal.
func validationConflictReason(
	ctx context.Context,
	client *http.Client,
	apiRoot, token, fallbackDescription string,
) string {
	webhook, err := telegramValidationCall(ctx, client, apiRoot, token, "getWebhookInfo", nil)
	if err == nil && webhook.OK {
		var info struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(webhook.Result, &info) == nil {
			if strings.TrimSpace(info.URL) != "" {
				return "webhook_active"
			}
			return "bot_in_use"
		}
	}
	if strings.Contains(strings.ToLower(fallbackDescription), "webhook") {
		return "webhook_active"
	}
	return "bot_in_use"
}

// decisiveStatus is the HTTP status that classifies a validation call, or 0
// when the answer is not decisive. Telegram answers 401 for a bad credential and
// 409 for a bot owned elsewhere; everything else is informational here.
func (r telegramValidationResponse) decisiveStatus() int {
	if r.StatusCode == http.StatusUnauthorized || r.ErrorCode == http.StatusUnauthorized {
		return http.StatusUnauthorized
	}
	if r.StatusCode == http.StatusConflict || r.ErrorCode == http.StatusConflict {
		return http.StatusConflict
	}
	return 0
}

func telegramValidationHTTP(baseURL, proxy string) (*http.Client, string, error) {
	apiRoot := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if apiRoot == "" {
		apiRoot = "https://api.telegram.org"
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.TrimSpace(proxy) != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, "", err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	} else if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" {
		transport.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{Transport: transport}, apiRoot, nil
}

// telegramValidationCall issues one Bot API method and parses its envelope. It
// never logs or returns the request URL, which contains the candidate token.
func telegramValidationCall(
	ctx context.Context,
	client *http.Client,
	apiRoot, token, method string,
	body io.Reader,
) (telegramValidationResponse, error) {
	if body == nil {
		body = bytes.NewReader([]byte("{}"))
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, apiRoot+"/bot"+token+"/"+method, body,
	)
	if err != nil {
		return telegramValidationResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return telegramValidationResponse{}, err
	}
	defer resp.Body.Close()

	result := telegramValidationResponse{StatusCode: resp.StatusCode}
	var envelope struct {
		OK          bool            `json:"ok"`
		ErrorCode   int             `json:"error_code"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 256<<10)).Decode(&envelope); decodeErr != nil {
		return result, nil
	}
	result.OK = envelope.OK
	result.ErrorCode = envelope.ErrorCode
	result.Description = envelope.Description
	result.Result = envelope.Result
	return result, nil
}
