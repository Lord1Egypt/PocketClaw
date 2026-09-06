package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// handleCallbackQuery runs a tapped button.
//
// A callback is an action request, not trusted UI state: the message carrying
// it can be forwarded into a group, and the person tapping is not necessarily
// the person the picker was built for. So the order here is deliberate —
// authorize the human first, resolve the opaque handle second, check the handle
// belongs to that human third, and only then let the agent act.
//
// Every path answers the callback. An unanswered one leaves Telegram's spinner
// turning on the user's device until it times out.
func (c *TelegramChannel) handleCallbackQuery(ctx context.Context, query telego.CallbackQuery) error {
	if query.Message == nil {
		c.answerCallback(ctx, query.ID, "That menu is no longer available.")
		return nil
	}

	chatID := fmt.Sprintf("%d", query.Message.GetChat().ID)
	messageID := strconv.Itoa(query.Message.GetMessageID())
	senderID := fmt.Sprintf("%d", query.From.ID)

	// Same allow-list the protected commands use. A user who could not send
	// /model must not be able to act by tapping one someone else opened.
	sender := bus.SenderInfo{
		Platform:    "telegram",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("telegram", senderID),
		Username:    query.From.Username,
		DisplayName: query.From.FirstName,
	}
	if !c.IsAllowedSender(sender) {
		logger.WarnCF("telegram", "Rejected menu action from an unauthorized sender",
			map[string]any{"chat_id": chatID})
		c.answerCallback(ctx, query.ID, "You are not allowed to do that.")
		return nil
	}

	entry, ok := c.callbacks.resolve(query.Data)
	if !ok {
		// Unknown or expired. Both are the same to the user: the picker they
		// are looking at is stale.
		c.answerCallback(ctx, query.ID, "That menu has expired. Send /model again.")
		return nil
	}
	if !entry.authorizes(chatID, senderID) {
		logger.WarnCF("telegram", "Rejected menu action bound to another chat or sender",
			map[string]any{"chat_id": chatID})
		c.answerCallback(ctx, query.ID, "That menu is not yours.")
		return nil
	}

	result, handled := c.Bus().RunMenuAction(ctx, bus.MenuActionRequest{
		Channel:  c.Name(),
		ChatID:   chatID,
		SenderID: senderID,
		Action:   entry.action,
		Value:    entry.value,
	})
	if !handled {
		c.answerCallback(ctx, query.ID, "That action is unavailable right now.")
		return nil
	}

	c.answerCallback(ctx, query.ID, callbackToast(result.Message))

	// The picker updates in place. A model switch is configuration, not
	// conversation, so it leaves no new message behind — the message the user
	// tapped becomes the record of what changed.
	c.callbacks.invalidateMessage(chatID, messageID)
	if entry.action == menuActionCancel {
		c.callbacks.clearActivePicker(chatID, messageID)
	}
	if result.Menu == nil && result.Text == "" {
		c.clearKeyboard(ctx, query.Message.GetChat().ID, query.Message.GetMessageID())
		return nil
	}

	rows, handles := c.menuToKeyboard(result.Menu, chatID, senderID)
	if c.updatePicker(
		ctx,
		query.Message.GetChat().ID,
		query.Message.GetMessageID(),
		result.Text,
		rows,
	) {
		for _, handle := range handles {
			c.callbacks.bindMessage(handle, chatID, messageID)
		}
	}
	return nil
}

// menuActionCancel mirrors the agent's cancel action name. It is duplicated
// here rather than imported so the channel does not depend on the agent.
const menuActionCancel = "menu.cancel"

// callbackToast keeps the spinner's replacement short. Telegram truncates these
// anyway, and a long one reads as an error even when it is not.
func callbackToast(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "Done."
	}
	if len([]rune(message)) > 180 {
		return string([]rune(message)[:177]) + "..."
	}
	return message
}

// answerCallback clears the client spinner. Failures are logged and swallowed:
// the action already happened, and a failed acknowledgement is not worth
// reporting to the user as an error.
func (c *TelegramChannel) answerCallback(ctx context.Context, queryID, text string) {
	if err := c.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: queryID,
		Text:            text,
	}); err != nil {
		logger.DebugCF("telegram", "Failed to answer callback query",
			map[string]any{"error": err.Error()})
	}
}

// updatePicker rewrites the tapped message in place — its body, its buttons, or
// both. Reports whether it worked, so the caller only binds new handles to a
// message that actually carries them.
//
// Editing text and markup together is one Telegram call; markup alone is
// another. Both are edits of the same message, which is what keeps a switch
// from adding to the chat.
func (c *TelegramChannel) updatePicker(
	ctx context.Context,
	chatID int64,
	messageID int,
	text string,
	rows [][]inlineButton,
) bool {
	markup := inlineKeyboardMarkup(rows)

	if strings.TrimSpace(text) == "" {
		if markup == nil {
			c.clearKeyboard(ctx, chatID, messageID)
			return false
		}
		_, err := c.bot.EditMessageReplyMarkup(ctx, &telego.EditMessageReplyMarkupParams{
			ChatID:      tu.ID(chatID),
			MessageID:   messageID,
			ReplyMarkup: markup,
		})
		if err != nil {
			logger.DebugCF("telegram", "Failed to update menu keyboard",
				map[string]any{"error": err.Error()})
			return false
		}
		return true
	}

	params := &telego.EditMessageTextParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
		Text:      parseContent(text, c.tgCfg.UseMarkdownV2),
	}
	if c.tgCfg.UseMarkdownV2 {
		params.ParseMode = telego.ModeMarkdownV2
	} else {
		params.ParseMode = telego.ModeHTML
	}
	params.ReplyMarkup = markup

	if _, err := c.bot.EditMessageText(ctx, params); err != nil {
		logger.DebugCF("telegram", "Failed to update picker text",
			map[string]any{"error": err.Error()})
		return false
	}
	return true
}

// clearKeyboard removes a picker's buttons so a closed menu cannot be tapped
// again and does not sit in the chat looking live.
func (c *TelegramChannel) clearKeyboard(ctx context.Context, chatID int64, messageID int) {
	if _, err := c.bot.EditMessageReplyMarkup(ctx, &telego.EditMessageReplyMarkupParams{
		ChatID:    tu.ID(chatID),
		MessageID: messageID,
	}); err != nil {
		logger.DebugCF("telegram", "Failed to clear menu keyboard",
			map[string]any{"error": err.Error()})
	}
}

// retirePicker turns a superseded picker into a closed card: its body says so
// and its buttons are gone.
//
// It edits the existing message rather than sending anything, because closing
// an old card must not add to the conversation, and rather than deleting it,
// because a message vanishing from a chat is more startling than one that says
// it is finished. A failure is logged and dropped — the handles are already
// invalidated, so the stale card is inert either way.
func (c *TelegramChannel) retirePicker(ctx context.Context, chatID int64, messageID string) {
	id, err := strconv.Atoi(messageID)
	if err != nil {
		return
	}
	c.updatePicker(ctx, chatID, id, agentMenuTextPickerClosed, nil)
}

// agentMenuTextPickerClosed mirrors agent.MenuTextPickerClosed. It is repeated
// rather than imported so the channel keeps no dependency on the agent, the
// same way the cancel action name is.
const agentMenuTextPickerClosed = "Model selection closed."
