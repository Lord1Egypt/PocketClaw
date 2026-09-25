package channels

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
)

// TypingCapable — channels that can show a typing/thinking indicator.
// StartTyping begins the indicator and returns a stop function.
// The stop function MUST be idempotent and safe to call multiple times.
type TypingCapable interface {
	StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

// MessageEditor — channels that can edit an existing message.
// messageID is always string; channels convert platform-specific types internally.
type MessageEditor interface {
	EditMessage(ctx context.Context, chatID string, messageID string, content string) error
}

// MessageEditorWithPayload extends MessageEditor for channels that can update
// structured message metadata in addition to plain text content.
type MessageEditorWithPayload interface {
	EditMessageWithPayload(
		ctx context.Context,
		chatID string,
		messageID string,
		payload map[string]any,
	) error
}

// MessageDeleter — channels that can delete a message by ID.
type MessageDeleter interface {
	DeleteMessage(ctx context.Context, chatID string, messageID string) error
}

// ReactionCapable — channels that can add a reaction (e.g. 👀) to an inbound message.
// ReactToMessage adds a reaction and returns an undo function to remove it.
// The undo function MUST be idempotent and safe to call multiple times.
type ReactionCapable interface {
	ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error)
}

// PlaceholderCapable — channels that can send a placeholder message
// (e.g. "Thinking... 💭") that will later be edited to the actual response.
// The channel MUST also implement MessageEditor for the placeholder to be useful.
// SendPlaceholder returns the platform message ID of the placeholder so that
// Manager.preSend can later edit it via MessageEditor.EditMessage.
type PlaceholderCapable interface {
	SendPlaceholder(ctx context.Context, chatID string) (messageID string, err error)
}

// QueueNoticeCapable — channels that can tell a user their message is waiting
// behind a turn that is still running. The notice is a short reply to the
// queued message; it is not a placeholder, is never edited into an answer, and
// the agent deletes it (MessageDeleter) once the message starts executing.
// It returns an empty ID, and no error, when the channel has status messages
// switched off.
type QueueNoticeCapable interface {
	SendQueueNotice(ctx context.Context, chatID, replyToMessageID, text string) (messageID string, err error)
}

// StreamingCapable — channels that can show partial LLM output in real-time.
// The channel SHOULD gracefully degrade if the platform rejects streaming
// (e.g. Telegram bot without forum mode). In that case, Update becomes a no-op
// and Finalize still delivers the final message.
type StreamingCapable interface {
	BeginStream(ctx context.Context, chatID string) (Streamer, error)
}

// Streamer is defined in pkg/bus to avoid circular imports.
// This alias keeps channel implementations using channels.Streamer unchanged.
type Streamer = bus.Streamer

// PlaceholderRecorder is injected into channels by Manager.
// Channels call these methods on inbound to register typing/placeholder state.
// Manager uses the registered state on outbound to stop typing and edit placeholders.
type PlaceholderRecorder interface {
	RecordPlaceholder(channel, chatID, placeholderID string)
	RecordTypingStop(channel, chatID string, stop func())
	RecordReactionUndo(channel, chatID string, undo func())
}

// CorrelatedPlaceholderRecorder keeps per-inbound lifecycle state separate
// when multiple messages reach the same chat close together. Implementations
// retain the legacy PlaceholderRecorder methods for channels without a
// correlation ID.
type CorrelatedPlaceholderRecorder interface {
	RecordPlaceholderForLifecycle(channel, chatID, lifecycleID, placeholderID string)
	RecordTypingStopForLifecycle(channel, chatID, lifecycleID string, stop func())
	RecordReactionUndoForLifecycle(channel, chatID, lifecycleID string, undo func())
}

// CommandRegistrarCapable is implemented by channels that can register
// command menus with their upstream platform (e.g. Telegram BotCommand).
// Channels that do not support platform-level command menus can ignore it.
type CommandRegistrarCapable interface {
	RegisterCommands(ctx context.Context, defs []commands.Definition) error
}

// CommandMenuReporter is implemented by a channel that can say whether its
// command menu has actually reached the platform.
//
// PC-DEF-061. "Connected" has to mean the bot is ready for the owner's first
// message, and registration is part of that. Registration is asynchronous and
// retried, so its outcome is a fact only the channel holds -- the menu itself
// cannot be read back from inside the process, since only Telegram knows what
// it is showing.
type CommandMenuReporter interface {
	// CommandsRegistered reports whether the menu was published successfully.
	CommandsRegistered() bool
}

// PollingGenerationReporter is implemented by a channel whose readiness depends
// on a polling owner, and which can name the owner that is currently active.
//
// PC-DEF-061. A readiness decision must be able to say which getUpdates owner
// it authorized, so a stale generation's success cannot be inherited by the
// generation that replaced it. The value is a process-local counter, never a
// token, bot id, owner id or chat id.
type PollingGenerationReporter interface {
	// PollingGeneration reports the active owner's id, or zero when none.
	PollingGeneration() uint64
}

// RuntimeFailureReporter is implemented by a channel that can expose a safe,
// terminal lifecycle code. Codes are machine-readable and must never contain an
// upstream response, identity or credential.
type RuntimeFailureReporter interface {
	RuntimeFailure() string
}

// OwnerMissingReporter is implemented by a channel that requires a configured
// owner identity and can say when the credential is valid but no owner is set.
//
// It is a boolean fact only, carrying no identity: it lets readiness report an
// incomplete setup instead of "connected", so a UI never presents a bot that
// will not answer its owner as ready.
type OwnerMissingReporter interface {
	OwnerMissing() bool
}
