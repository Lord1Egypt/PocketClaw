package bus

import (
	"context"
	"strings"
)

// LifecycleIDMetadataKey carries a process-local, non-user correlation ID from
// inbound acceptance through final channel delivery. It must never contain a
// platform user, chat, message, token, or message-content value.
const LifecycleIDMetadataKey = "pocketclaw_lifecycle_id"

type lifecycleIDContextKey struct{}

// InboundLifecycleID returns the safe correlation ID attached to an inbound or
// outbound context.
func InboundLifecycleID(ctx *InboundContext) string {
	if ctx == nil || len(ctx.Raw) == 0 {
		return ""
	}
	return strings.TrimSpace(ctx.Raw[LifecycleIDMetadataKey])
}

// WithLifecycleID makes the correlation ID available to direct streaming
// delivery without changing public Streamer method signatures.
func WithLifecycleID(ctx context.Context, lifecycleID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if lifecycleID = strings.TrimSpace(lifecycleID); lifecycleID == "" {
		return ctx
	}
	return context.WithValue(ctx, lifecycleIDContextKey{}, lifecycleID)
}

// LifecycleIDFromContext reads a value installed by WithLifecycleID.
func LifecycleIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(lifecycleIDContextKey{}).(string)
	return strings.TrimSpace(value)
}

// SenderInfo provides structured sender identity information.
type SenderInfo struct {
	Platform    string `json:"platform,omitempty"`     // "telegram", "discord", "slack", ...
	PlatformID  string `json:"platform_id,omitempty"`  // raw platform ID, e.g. "123456"
	CanonicalID string `json:"canonical_id,omitempty"` // "platform:id" format
	Username    string `json:"username,omitempty"`     // username (e.g. @alice)
	DisplayName string `json:"display_name,omitempty"` // display name
}

// InboundContext captures the normalized, platform-agnostic facts about an
// inbound message. This is the source of truth for routing and session
// allocation.
type InboundContext struct {
	Channel string `json:"channel"`
	Account string `json:"account,omitempty"`

	ChatID   string `json:"chat_id"`
	ChatType string `json:"chat_type,omitempty"` // direct / group / channel
	TopicID  string `json:"topic_id,omitempty"`

	SpaceID   string `json:"space_id,omitempty"`
	SpaceType string `json:"space_type,omitempty"` // guild / team / workspace / tenant

	SenderID  string `json:"sender_id"`
	MessageID string `json:"message_id,omitempty"`

	Mentioned bool `json:"mentioned,omitempty"`

	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	ReplyToSenderID  string `json:"reply_to_sender_id,omitempty"`

	ReplyHandles map[string]string `json:"reply_handles,omitempty"`
	Raw          map[string]string `json:"raw,omitempty"`
}

type InboundMessage struct {
	Context    InboundContext `json:"context"`
	Sender     SenderInfo     `json:"sender"`
	Content    string         `json:"content"`
	Media      []string       `json:"media,omitempty"`
	MediaScope string         `json:"media_scope,omitempty"` // media lifecycle scope
	SessionKey string         `json:"session_key"`

	// Convenience mirrors derived from Context for runtime consumers.
	Channel   string `json:"channel"`
	SenderID  string `json:"sender_id"`
	ChatID    string `json:"chat_id"`
	MessageID string `json:"message_id,omitempty"` // platform message ID
}

// OutboundScope captures the structured session scope associated with an
// outbound turn result without depending on the session package.
type OutboundScope struct {
	Version    int               `json:"version,omitempty"`
	AgentID    string            `json:"agent_id,omitempty"`
	Channel    string            `json:"channel,omitempty"`
	Account    string            `json:"account,omitempty"`
	Dimensions []string          `json:"dimensions,omitempty"`
	Values     map[string]string `json:"values,omitempty"`
}

// ContextUsage describes how much of the model's context window the current
// session consumes, and how far it is from triggering compression.
type ContextUsage struct {
	UsedTokens        int `json:"used_tokens"`
	TotalTokens       int `json:"total_tokens"`        // model context window
	HistoryTokens     int `json:"history_tokens"`      // history-message tokens only (what maybeSummarize checks)
	CompressAtTokens  int `json:"compress_at_tokens"`  // hard budget compression threshold (contextWindow - maxTokens)
	SummarizeAtTokens int `json:"summarize_at_tokens"` // soft summarization trigger (vs history tokens)
	UsedPercent       int `json:"used_percent"`        // 0-100, relative to compressAt
}

type OutboundMessage struct {
	Channel          string         `json:"channel"`
	ChatID           string         `json:"chat_id"`
	Context          InboundContext `json:"context"`
	AgentID          string         `json:"agent_id,omitempty"`
	SessionKey       string         `json:"session_key,omitempty"`
	Scope            *OutboundScope `json:"scope,omitempty"`
	Content          string         `json:"content"`
	ReplyToMessageID string         `json:"reply_to_message_id,omitempty"`
	ContextUsage     *ContextUsage  `json:"context_usage,omitempty"`

	// Menu offers the reader a set of choices to tap instead of a command to
	// type. It is optional and additive: a channel that cannot render choices
	// ignores it and sends Content as it always has, so this changes nothing
	// for messages that do not set it.
	Menu *InteractiveMenu `json:"menu,omitempty"`
}

// InteractiveMenu is a set of tappable choices attached to a message.
//
// It is deliberately not a UI framework. It describes one flat list of labelled
// buttons and nothing else — no nesting, no styling, no layout engine — because
// one inline choice menu is all PocketClaw needs, and a generic abstraction
// would have to guess at what every channel's UI can express.
type InteractiveMenu struct {
	Rows []MenuRow `json:"rows"`
}

// MenuRow is one row of buttons, rendered side by side where the channel allows.
type MenuRow struct {
	Buttons []MenuButton `json:"buttons"`
}

// MenuButton is one choice.
//
// Label is what the reader sees. Action and Value say what tapping it means, in
// PocketClaw's own terms — never a credential, an API base or a registry key.
// The channel converts these into whatever opaque handle its platform needs;
// they are not themselves transmitted to the platform.
type MenuButton struct {
	Label   string `json:"label"`
	Action  string `json:"action"`
	Value   string `json:"value,omitempty"`
	Current bool   `json:"current,omitempty"`
}

// MediaPart describes a single media attachment to send.
type MediaPart struct {
	Type        string `json:"type"`                   // "image" | "audio" | "video" | "file"
	Ref         string `json:"ref"`                    // media store ref, e.g. "media://abc123"
	Caption     string `json:"caption,omitempty"`      // optional caption text
	Filename    string `json:"filename,omitempty"`     // original filename hint
	ContentType string `json:"content_type,omitempty"` // MIME type hint
}

// OutboundMediaMessage carries media attachments from Agent to channels via the bus.
type OutboundMediaMessage struct {
	Channel    string         `json:"channel"`
	ChatID     string         `json:"chat_id"`
	Context    InboundContext `json:"context"`
	AgentID    string         `json:"agent_id,omitempty"`
	SessionKey string         `json:"session_key,omitempty"`
	Scope      *OutboundScope `json:"scope,omitempty"`
	Parts      []MediaPart    `json:"parts"`
}

// AudioChunk represents a chunk of streaming voice data.
type AudioChunk struct {
	SessionID  string `json:"session_id"`
	SpeakerID  string `json:"speaker_id"` // User ID or SSRC
	ChatID     string `json:"chat_id"`    // Where to respond
	Channel    string `json:"channel"`    // Source channel type (e.g. "discord")
	Sequence   uint64 `json:"sequence"`
	Timestamp  uint32 `json:"timestamp"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Format     string `json:"format"` // "opus", "pcm", etc
	Data       []byte `json:"data"`
}

// VoiceControl represents state or commands for voice sessions.
type VoiceControl struct {
	SessionID string `json:"session_id"`
	ChatID    string `json:"chat_id"`
	Type      string `json:"type"`   // "state", "command"
	Action    string `json:"action"` // "idle", "listening", "start", "stop", "leave"
}
