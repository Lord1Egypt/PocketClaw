// Package telegram is a minimal Bot API client covering exactly what the
// onboarding service needs: identifying the manager bot, long-polling for
// managed-bot updates, and retrieving a managed bot's token.
//
// Managed bots are Bot API 9.6 (2026-04-03). The relevant surface is:
//
//	User.can_manage_bots      Boolean
//	Update.managed_bot        ManagedBotUpdated{user User, bot User}
//	Message.managed_bot_created ManagedBotCreated{bot User}
//	getManagedBotToken(user_id) -> String
//	replaceManagedBotToken(user_id) -> String
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// User is the subset of Telegram's User object the service uses.
type User struct {
	ID            int64  `json:"id"`
	IsBot         bool   `json:"is_bot"`
	FirstName     string `json:"first_name"`
	Username      string `json:"username"`
	CanManageBots bool   `json:"can_manage_bots"`
}

// ManagedBotUpdated reports the creation of a managed bot, or a change to its
// token or owner.
type ManagedBotUpdated struct {
	User User `json:"user"`
	Bot  User `json:"bot"`
}

// Update is the subset of Telegram's Update object the service consumes.
type Update struct {
	UpdateID   int64              `json:"update_id"`
	ManagedBot *ManagedBotUpdated `json:"managed_bot"`
}

// Client talks to the Telegram Bot API as the manager bot.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

// New returns a client for the manager bot token. The token is held here and
// must never leave this package.
func New(token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 70 * time.Second}
	}
	return &Client{
		token:   token,
		baseURL: "https://api.telegram.org",
		http:    httpClient,
	}
}

// SetBaseURL points the client at a different host. For tests.
func (c *Client) SetBaseURL(base string) { c.baseURL = strings.TrimSuffix(base, "/") }

// GetMe identifies the manager bot and reports whether Telegram has granted it
// bot-management rights.
func (c *Client) GetMe(ctx context.Context) (User, error) {
	var user User
	if err := c.call(ctx, "getMe", nil, &user); err != nil {
		return User{}, err
	}
	return user, nil
}

// GetUpdates long-polls for new updates starting at offset.
func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	params := url.Values{}
	params.Set("offset", strconv.FormatInt(offset, 10))
	params.Set("timeout", strconv.Itoa(timeoutSeconds))
	// managed_bot is delivered by default, but naming it explicitly means a
	// future default change cannot silently stop onboarding from working.
	params.Set("allowed_updates", `["managed_bot"]`)

	var updates []Update
	if err := c.call(ctx, "getUpdates", params, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// GetManagedBotToken returns the token of a bot this manager bot manages.
// userID is the managed bot's own user identifier.
func (c *Client) GetManagedBotToken(ctx context.Context, userID int64) (string, error) {
	params := url.Values{}
	params.Set("user_id", strconv.FormatInt(userID, 10))

	var token string
	if err := c.call(ctx, "getManagedBotToken", params, &token); err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("getManagedBotToken returned an empty token for bot %d", userID)
	}
	return token, nil
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
}

func (c *Client) call(ctx context.Context, method string, params url.Values, out any) error {
	endpoint := fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.token, method)
	body := strings.NewReader(params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return fmt.Errorf("build %s request: %s", method, Redact(err.Error(), c.token))
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		// A transport error can quote the full URL, which contains the manager
		// token. Redact before it reaches a log.
		return fmt.Errorf("%s: %s", method, Redact(err.Error(), c.token))
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("%s: read response: %s", method, Redact(err.Error(), c.token))
	}

	var parsed apiResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("%s: decode response (HTTP %d)", method, resp.StatusCode)
	}
	if !parsed.OK {
		return &APIError{
			Method:      method,
			Code:        parsed.ErrorCode,
			Description: Redact(parsed.Description, c.token),
		}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(parsed.Result, out); err != nil {
		return fmt.Errorf("%s: decode result: %w", method, err)
	}
	return nil
}

// APIError is a Telegram-reported failure.
type APIError struct {
	Method      string
	Code        int
	Description string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram %s failed with %d: %s", e.Method, e.Code, e.Description)
}

// Redact removes a secret from text that is about to be logged or returned.
// Bot tokens reach error strings through request URLs, so every path that can
// produce a message from a request must pass through here.
func Redact(text, secret string) string {
	if secret == "" {
		return text
	}
	text = strings.ReplaceAll(text, secret, "[REDACTED]")
	// A token is "<bot id>:<secret>". Redact the secret half even when only
	// part of the token appears, which happens when Telegram echoes a URL.
	if idx := strings.Index(secret, ":"); idx > 0 && idx < len(secret)-1 {
		text = strings.ReplaceAll(text, secret[idx+1:], "[REDACTED]")
	}
	return text
}
