// Package deeplink builds the official Telegram managed-bot creation link.
//
// The format is fixed by Telegram, introduced in Bot API 9.6 (2026-04-03):
//
//	https://t.me/newbot/{manager_bot_username}/{suggested_bot_username}[?name={suggested_bot_name}]
//
// PocketClaw does not extend it. The link carries no secret: it names the
// manager bot, the username to suggest, and the display name to suggest, all
// of which are public by the time the user sees the creation screen.
package deeplink

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Lord1Egypt/PocketClaw/services/telegram-onboarding/internal/naming"
)

const telegramBase = "https://t.me/newbot"

// NewBot returns the managed-bot creation link for a suggested child bot.
func NewBot(managerUsername, suggestedUsername, suggestedName string) (string, error) {
	managerUsername = strings.TrimPrefix(strings.TrimSpace(managerUsername), "@")
	if managerUsername == "" {
		return "", fmt.Errorf("manager bot username is not configured")
	}
	if err := naming.ValidateUsername(managerUsername); err != nil {
		return "", fmt.Errorf("manager bot username is invalid: %w", err)
	}
	if err := naming.ValidateUsername(suggestedUsername); err != nil {
		return "", fmt.Errorf("suggested bot username is invalid: %w", err)
	}
	if err := naming.ValidateDisplayName(suggestedName); err != nil {
		return "", fmt.Errorf("suggested bot name is invalid: %w", err)
	}

	link := fmt.Sprintf("%s/%s/%s", telegramBase, managerUsername, suggestedUsername)
	// url.Values encodes a space as "+", which is correct for form bodies but
	// wrong inside a Telegram deep-link path query. PathEscape gives "%20".
	return link + "?name=" + url.PathEscape(suggestedName), nil
}

// BotChat returns the link that opens a conversation with a bot.
func BotChat(username string) (string, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if err := naming.ValidateUsername(username); err != nil {
		return "", fmt.Errorf("bot username is invalid: %w", err)
	}
	return "https://t.me/" + username, nil
}
