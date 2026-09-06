package telegram

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// callbackHandleTTL bounds how long a tapped button stays valid.
//
// Ten minutes is long enough that a picker left open while the user reads
// something else still works, and short enough that a menu screenshotted or
// forwarded into a group chat stops being an action anyone can replay. The
// registry is in memory, so a Core restart invalidates every handle as well —
// which is the behaviour we want, because the agent's model binding is itself
// reset by that restart.
const callbackHandleTTL = 10 * time.Minute

// callbackDataPrefix keeps PocketClaw's own callbacks distinguishable from
// anything else that might arrive on the same bot.
const callbackDataPrefix = "pc:"

// maxTrackedCallbacks bounds the registry so a user repeatedly opening pickers
// cannot grow it without limit.
const maxTrackedCallbacks = 512

// callbackEntry is what a handle resolves to. It holds PocketClaw's own action
// vocabulary and the identity allowed to use it — never a credential, an
// endpoint or a model configuration.
type callbackEntry struct {
	action    string
	value     string
	chatID    string
	senderID  string
	messageID string
	expiresAt time.Time
}

// callbackRegistry maps opaque handles to the action they stand for.
//
// The handle is the only thing that travels to Telegram. Putting a model name
// in callback_data would mean the platform, anyone who forwards the message,
// and anyone reading the chat export all learn PocketClaw's configuration, and
// it would let a crafted callback name a model that was never offered.
type callbackRegistry struct {
	mu      sync.Mutex
	entries map[string]callbackEntry
	// active records the one live picker per chat. A second picker makes the
	// first one a dead card: its buttons stop working, and without this it
	// would sit in the conversation still saying "Choose a model".
	active map[string]string
	now    func() time.Time
}

func newCallbackRegistry() *callbackRegistry {
	return &callbackRegistry{
		entries: make(map[string]callbackEntry),
		active:  make(map[string]string),
		now:     time.Now,
	}
}

// mint stores an action and returns the opaque handle that stands for it.
func (r *callbackRegistry) mint(entry callbackEntry) (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	handle := base64.RawURLEncoding.EncodeToString(raw)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.pruneLocked()
	entry.expiresAt = r.now().Add(callbackHandleTTL)
	r.entries[handle] = entry
	return callbackDataPrefix + handle, nil
}

// resolve returns the entry a handle stands for, if it is still valid.
func (r *callbackRegistry) resolve(data string) (callbackEntry, bool) {
	handle, ok := strings.CutPrefix(data, callbackDataPrefix)
	if !ok {
		return callbackEntry{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	entry, found := r.entries[handle]
	if !found {
		return callbackEntry{}, false
	}
	if r.now().After(entry.expiresAt) {
		delete(r.entries, handle)
		return callbackEntry{}, false
	}
	return entry, true
}

// invalidateMessage drops every handle belonging to one picker, so a closed or
// superseded menu cannot be tapped again.
func (r *callbackRegistry) invalidateMessage(chatID, messageID string) {
	if messageID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for handle, entry := range r.entries {
		if entry.chatID == chatID && entry.messageID == messageID {
			delete(r.entries, handle)
		}
	}
}

// bindMessage records which message a freshly minted handle belongs to. Handles
// are minted before the message exists, because the keyboard has to be built
// first, so the id is filled in once Telegram has assigned one.
func (r *callbackRegistry) bindMessage(data, chatID, messageID string) {
	handle, ok := strings.CutPrefix(data, callbackDataPrefix)
	if !ok {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, found := r.entries[handle]; found {
		entry.chatID = chatID
		entry.messageID = messageID
		r.entries[handle] = entry
	}
}

// pruneLocked removes expired handles, and if the registry is still at its
// bound, the oldest ones. Caller holds the lock.
func (r *callbackRegistry) pruneLocked() {
	now := r.now()
	for handle, entry := range r.entries {
		if now.After(entry.expiresAt) {
			delete(r.entries, handle)
		}
	}
	if len(r.entries) < maxTrackedCallbacks {
		return
	}
	oldest := ""
	var oldestAt time.Time
	for handle, entry := range r.entries {
		if oldest == "" || entry.expiresAt.Before(oldestAt) {
			oldest, oldestAt = handle, entry.expiresAt
		}
	}
	delete(r.entries, oldest)
}

// authorizes reports whether this entry may be acted on by the given sender in
// the given chat.
//
// A callback is an action request, not trusted UI state: the message it came
// from can be forwarded, and the person tapping is not necessarily the person
// who opened the picker.
func (e callbackEntry) authorizes(chatID, senderID string) bool {
	return e.chatID == chatID && e.senderID == senderID
}

// menuToKeyboard mints a handle per button and returns the rows Telegram needs.
func (c *TelegramChannel) menuToKeyboard(
	menu *bus.InteractiveMenu,
	chatID, senderID string,
) ([][]inlineButton, []string) {
	if menu == nil || len(menu.Rows) == 0 {
		return nil, nil
	}

	rows := make([][]inlineButton, 0, len(menu.Rows))
	handles := make([]string, 0, len(menu.Rows))
	for _, row := range menu.Rows {
		buttons := make([]inlineButton, 0, len(row.Buttons))
		for _, button := range row.Buttons {
			data, err := c.callbacks.mint(callbackEntry{
				action:   button.Action,
				value:    button.Value,
				chatID:   chatID,
				senderID: senderID,
			})
			if err != nil {
				continue
			}
			handles = append(handles, data)
			buttons = append(buttons, inlineButton{Label: button.Label, Data: data})
		}
		if len(buttons) > 0 {
			rows = append(rows, buttons)
		}
	}
	return rows, handles
}

// inlineButton is the channel-local shape of one rendered button.
type inlineButton struct {
	Label string
	Data  string
}

// inlineKeyboardMarkup converts rendered rows into Telegram's markup. Returns
// nil when there is nothing to render, so an ordinary message is unchanged.
func inlineKeyboardMarkup(rows [][]inlineButton) *telego.InlineKeyboardMarkup {
	if len(rows) == 0 {
		return nil
	}
	keyboard := make([][]telego.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		buttons := make([]telego.InlineKeyboardButton, 0, len(row))
		for _, button := range row {
			buttons = append(buttons, telego.InlineKeyboardButton{
				Text:         button.Label,
				CallbackData: button.Data,
			})
		}
		keyboard = append(keyboard, buttons)
	}
	return &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

// replaceActivePicker records messageID as the chat's live picker and returns
// the one it displaces, whose handles are invalidated in the same step.
//
// Returning the displaced id is what lets the caller retire that card visually.
// An empty return means there was nothing live to retire.
func (r *callbackRegistry) replaceActivePicker(chatID, messageID string) string {
	if chatID == "" || messageID == "" {
		return ""
	}

	r.mu.Lock()
	previous := r.active[chatID]
	r.active[chatID] = messageID
	if previous == messageID {
		previous = ""
	}
	if previous != "" {
		for handle, entry := range r.entries {
			if entry.chatID == chatID && entry.messageID == previous {
				delete(r.entries, handle)
			}
		}
	}
	r.mu.Unlock()
	return previous
}

// clearActivePicker forgets the chat's live picker, for when it is closed
// rather than replaced.
func (r *callbackRegistry) clearActivePicker(chatID, messageID string) {
	r.mu.Lock()
	if r.active[chatID] == messageID {
		delete(r.active, chatID)
	}
	r.mu.Unlock()
}

// activePicker reports the chat's live picker, if any.
func (r *callbackRegistry) activePicker(chatID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active[chatID]
}
