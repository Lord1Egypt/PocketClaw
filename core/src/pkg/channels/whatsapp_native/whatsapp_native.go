//go:build whatsapp_native

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/whatsapp/pairing"
)

const (
	sqliteDriver   = "sqlite"
	whatsappDBName = "store.db"

	reconnectInitial    = 5 * time.Second
	reconnectMax        = 5 * time.Minute
	reconnectMultiplier = 2.0
)

// WhatsAppNativeChannel implements the WhatsApp channel using whatsmeow (in-process, no external bridge).
type WhatsAppNativeChannel struct {
	*channels.BaseChannel
	config       *config.WhatsAppSettings
	storePath    string
	client       *whatsmeow.Client
	container    *sqlstore.Container
	mu           sync.Mutex
	runCtx       context.Context
	runCancel    context.CancelFunc
	reconnectMu  sync.Mutex
	reconnecting bool
	stopping     atomic.Bool    // set once Stop begins; prevents new wg.Add calls
	wg           sync.WaitGroup // tracks background goroutines (QR handler, reconnect)

	// pairing carries QR and lifecycle state to the console backend process.
	// It is nil off Android, where no host directory is named.
	pairing *pairing.Store
	// allowList is the configured allow_from, kept here because this channel
	// treats an empty list as deny rather than as allow-all.
	allowList []string
	// loggedOut is set when WhatsApp revoked the link. It stops the reconnect
	// loop from retrying a session the server has already thrown away.
	loggedOut atomic.Bool
	// selfNumber is the configured WhatsApp Self-Chat number in bare digits, and
	// it is the only identity this channel will ever read from or write to.
	// Empty means not configured, which denies everything in both directions.
	selfNumber string
	// pairRequested keeps the companion pairing code to a single request: it is
	// valid for one linking window, and asking again mid-flow would swap the
	// code out from under a user part-way through typing it.
	pairRequested atomic.Bool
}

// NewWhatsAppNativeChannel creates a WhatsApp channel that uses whatsmeow for connection.
// storePath is the directory for the SQLite session store (e.g. workspace/whatsapp).
func NewWhatsAppNativeChannel(
	bc *config.Channel,
	name string,
	cfg *config.WhatsAppSettings,
	bus *bus.MessageBus,
	storePath string,
	selfNumber string,
) (channels.Channel, error) {
	// One matching rule, applied on both sides. BaseChannel compares the
	// allow-list against the sender it is given, and it does that from inside
	// HandleMessageWithContext where an override on this type cannot intervene,
	// so the normalization has to happen to the data rather than to the check.
	allowList := normalizeAllowList(bc.AllowFrom)
	base := channels.NewBaseChannel(name, cfg, bus, allowList, channels.WithMaxMessageLength(65536))
	if storePath == "" {
		storePath = "whatsapp"
	}
	c := &WhatsAppNativeChannel{
		BaseChannel: base,
		config:      cfg,
		storePath:   storePath,
		pairing:     pairing.NewStore(),
		allowList:   allowList,
		selfNumber:  whatsAppNumber(selfNumber),
	}
	return c, nil
}

// deviceProps is applied once per process, before the first Connect, because
// whatsmeow sends it during companion registration.
var deviceProps sync.Once

// configureDeviceProps names PocketClaw in the user's Linked Devices list and
// asks the server for as little history as the protocol allows.
//
// The default asks for a 10 GB history quota with no day limit. PocketClaw
// reads one chat — the user's own — so every other conversation the server
// would ship is data it has no use for: it costs memory on a phone, and it is
// account content this feature has no business receiving.
func configureDeviceProps() {
	deviceProps.Do(func() {
		store.DeviceProps.Os = proto.String("PocketClaw")
		if hs := store.DeviceProps.HistorySyncConfig; hs != nil {
			hs.RecentSyncDaysLimit = proto.Uint32(0)
			hs.StorageQuotaMb = proto.Uint32(1)
			hs.SupportGroupHistory = proto.Bool(false)
			hs.SupportCallLogHistory = proto.Bool(false)
		}
		store.DeviceProps.RequireFullSync = proto.Bool(false)
	})
}

func (c *WhatsAppNativeChannel) Start(ctx context.Context) error {
	logger.InfoCF("whatsapp", "Starting WhatsApp native channel (whatsmeow)", map[string]any{"store": c.storePath})
	configureDeviceProps()

	// Reset lifecycle state from any previous Stop() so a restarted channel
	// behaves correctly.  Use reconnectMu to be consistent with eventHandler
	// and Stop() which coordinate under the same lock.
	c.reconnectMu.Lock()
	c.stopping.Store(false)
	c.reconnecting = false
	c.reconnectMu.Unlock()
	c.pairRequested.Store(false)

	if err := os.MkdirAll(c.storePath, 0o700); err != nil {
		return fmt.Errorf("create session store dir: %w", err)
	}

	dbPath := filepath.Join(c.storePath, whatsappDBName)
	connStr := "file:" + dbPath + "?_foreign_keys=on"

	db, err := sql.Open(sqliteDriver, connStr)
	if err != nil {
		return fmt.Errorf("open whatsapp store: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	waLogger := waLog.Stdout("WhatsApp", "WARN", true)
	container := sqlstore.NewWithDB(db, sqliteDriver, waLogger)
	if err = container.Upgrade(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("open whatsapp store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return fmt.Errorf("get device store: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, waLogger)

	// Create runCtx/runCancel BEFORE registering event handler and starting
	// goroutines so that Stop() can cancel them at any time, including during
	// the QR-login flow.
	c.runCtx, c.runCancel = context.WithCancel(ctx)

	client.AddEventHandler(c.eventHandler)

	c.mu.Lock()
	c.container = container
	c.client = client
	c.mu.Unlock()

	// cleanupOnError clears struct references and releases resources when
	// Start() fails after fields are already assigned.  This prevents
	// Stop() from operating on stale references (double-close, disconnect
	// of a partially-initialized client, or stray event handler callbacks).
	startOK := false
	defer func() {
		if startOK {
			return
		}
		c.runCancel()
		client.Disconnect()
		c.mu.Lock()
		c.client = nil
		c.container = nil
		c.mu.Unlock()
		_ = container.Close()
	}()

	if client.Store.ID == nil {
		_ = c.pairing.PublishState(pairing.StateNotPaired, "")
		qrChan, err := client.GetQRChannel(c.runCtx)
		if err != nil {
			return fmt.Errorf("get QR channel: %w", err)
		}
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		// Handle QR events in a background goroutine so Start() returns
		// promptly.  The goroutine is tracked via c.wg and respects
		// c.runCtx for cancellation.
		// Guard wg.Add with reconnectMu + stopping check (same protocol
		// as eventHandler) so a concurrent Stop() cannot enter wg.Wait()
		// while we call wg.Add(1).
		c.reconnectMu.Lock()
		if c.stopping.Load() {
			c.reconnectMu.Unlock()
			return fmt.Errorf("channel stopped during QR setup")
		}
		c.wg.Add(1)
		c.reconnectMu.Unlock()
		go func() {
			defer c.wg.Done()
			for {
				select {
				case <-c.runCtx.Done():
					return
				case evt, ok := <-qrChan:
					if !ok {
						return
					}
					if evt.Event == "code" {
						// A QR arriving is also whatsmeow's signal that the
						// login socket is fully up, which is what PairPhone
						// requires. Requesting the companion code here rather
						// than earlier also spends the least of the ~160s
						// linking window.
						linkCode := c.requestPairingCode(client)

						// Both credentials go only to the pairing store, which
						// the console reads over an authenticated request.
						// Printing either here would put it in PocketClaw's
						// persisted Logs screen, where whoever used it first
						// would own the link.
						if err := c.pairing.PublishPairing(evt.Code, linkCode); err != nil {
							logger.WarnCF("whatsapp", "Could not publish pairing credentials", map[string]any{
								"error": err.Error(),
							})
						}
						logger.InfoC("whatsapp", "WhatsApp pairing ready; open the console to complete it")
					} else {
						// evt.Event is a whatsmeow lifecycle word ("success",
						// "timeout", "err-scanned"), never the code itself.
						logger.InfoCF("whatsapp", "WhatsApp login event", map[string]any{"event": evt.Event})
						if evt.Event == "success" {
							_ = c.pairing.PublishState(pairing.StateConnected, "")
						} else {
							_ = c.pairing.PublishState(pairing.StateNotPaired, evt.Event)
						}
					}
				}
			}
		}()
	} else {
		_ = c.pairing.PublishState(pairing.StateConnecting, "")
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
	}

	startOK = true
	c.SetRunning(true)
	logger.InfoC("whatsapp", "WhatsApp native channel connected")
	return nil
}

// requestPairingCode asks WhatsApp for a companion linking code, once.
//
// This is the primary flow on Android: PocketClaw and WhatsApp share one
// screen, so a QR displayed in PocketClaw is a QR the phone's own camera
// cannot reach. The code is requested for the configured Self-Chat number —
// the account this channel is allowed to touch at all — so there is no second
// number to ask the user for.
//
// It returns "" rather than failing the pairing: the QR is still published, and
// on a desktop or second screen that is a complete flow on its own.
func (c *WhatsAppNativeChannel) requestPairingCode(client *whatsmeow.Client) string {
	if c.selfNumber == "" || !c.pairRequested.CompareAndSwap(false, true) {
		return ""
	}
	// "Browser (OS)" is a server-validated format; anything else is rejected
	// with a 400, so this is not a place to put a product name.
	// Named linkCode, like the QR's evt.Code, so one guard can assert that no
	// logger line in this file ever mentions either.
	linkCode, err := client.PairPhone(
		c.runCtx, c.selfNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)",
	)
	if err != nil {
		// The error is reported, never the credential.
		logger.WarnCF("whatsapp", "Could not request a companion pairing code; QR remains available", map[string]any{
			"error": err.Error(),
		})
		c.pairRequested.Store(false)
		return ""
	}
	logger.InfoC("whatsapp", "WhatsApp companion pairing is ready; open the console to complete it")
	return linkCode
}

func (c *WhatsAppNativeChannel) Stop(ctx context.Context) error {
	logger.InfoC("whatsapp", "Stopping WhatsApp native channel")

	// Mark as stopping under reconnectMu so the flag is visible to
	// eventHandler atomically with respect to its wg.Add(1) call.
	// This closes the TOCTOU window where eventHandler could check
	// stopping (false), then Stop sets it true + enters wg.Wait,
	// then eventHandler calls wg.Add(1) — causing a panic.
	c.reconnectMu.Lock()
	c.stopping.Store(true)
	c.reconnectMu.Unlock()

	if c.runCancel != nil {
		c.runCancel()
	}

	// Disconnect the client first so any blocking Connect()/reconnect loops
	// can be interrupted before we wait on the goroutines.
	c.mu.Lock()
	client := c.client
	container := c.container
	c.mu.Unlock()

	if client != nil {
		client.Disconnect()
	}

	// Wait for background goroutines (QR handler, reconnect) to finish in a
	// context-aware way so Stop can be bounded by ctx.
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines have finished.
	case <-ctx.Done():
		// Context canceled or timed out; log and proceed with best-effort cleanup.
		logger.WarnC("whatsapp", fmt.Sprintf("Stop context canceled before all goroutines finished: %v", ctx.Err()))
	}

	// Now it is safe to clear and close resources.
	c.mu.Lock()
	c.client = nil
	c.container = nil
	c.mu.Unlock()

	if container != nil {
		_ = container.Close()
	}
	// Retire the snapshot. A QR that outlived the process would be a live
	// credential the console still offered, and a stale "connected" would
	// describe a channel that is no longer running.
	_ = c.pairing.Clear()
	c.SetRunning(false)
	return nil
}

func (c *WhatsAppNativeChannel) eventHandler(evt any) {
	switch v := evt.(type) {
	case *events.Message:
		c.handleIncoming(v)
	case *events.Connected:
		c.loggedOut.Store(false)
		_ = c.pairing.PublishState(pairing.StateConnected, "")
	case *events.LoggedOut:
		// WhatsApp revoked this link — the user unlinked the device, or the
		// server rejected the session. Reconnecting cannot fix it, and the
		// stored keys are now worthless, so they are deleted rather than left
		// behind for a reconnect loop to keep replaying.
		c.handleLoggedOut(v)
	case *events.Disconnected:
		// A transient drop. The session stays on disk; only a LoggedOut
		// deletes it. Conflating the two would wipe a good session every time
		// the phone changed networks.
		logger.InfoCF("whatsapp", "WhatsApp disconnected, will attempt reconnection", nil)
		_ = c.pairing.PublishState(pairing.StateDisconnected, "reconnecting")
		c.reconnectMu.Lock()
		if c.reconnecting {
			c.reconnectMu.Unlock()
			return
		}
		// Check stopping while holding the lock so the check and wg.Add
		// are atomic with respect to Stop() setting the flag + calling
		// wg.Wait(). This prevents the TOCTOU race.
		if c.stopping.Load() {
			c.reconnectMu.Unlock()
			return
		}
		c.reconnecting = true
		c.wg.Add(1)
		c.reconnectMu.Unlock()
		go func() {
			defer c.wg.Done()
			c.reconnectWithBackoff()
		}()
	}
}

func (c *WhatsAppNativeChannel) reconnectWithBackoff() {
	defer func() {
		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()
	}()

	backoff := reconnectInitial
	for {
		select {
		case <-c.runCtx.Done():
			return
		default:
		}

		// A revoked session cannot be reconnected. Without this the loop would
		// retry it every five minutes forever.
		if c.loggedOut.Load() {
			logger.InfoC("whatsapp", "WhatsApp session was logged out; not reconnecting")
			return
		}

		c.mu.Lock()
		client := c.client
		c.mu.Unlock()
		if client == nil {
			return
		}

		logger.InfoCF("whatsapp", "WhatsApp reconnecting", map[string]any{"backoff": backoff.String()})
		err := client.Connect()
		if err == nil {
			logger.InfoC("whatsapp", "WhatsApp reconnected")
			return
		}

		logger.WarnCF("whatsapp", "WhatsApp reconnect failed", map[string]any{"error": err.Error()})

		select {
		case <-c.runCtx.Done():
			return
		case <-time.After(backoff):
			if backoff < reconnectMax {
				next := time.Duration(float64(backoff) * reconnectMultiplier)
				if next > reconnectMax {
					next = reconnectMax
				}
				backoff = next
			}
		}
	}
}

// handleLoggedOut retires a session WhatsApp has revoked.
//
// It is deliberately not reachable from a plain disconnect: only the server
// saying the link is gone removes local state. The client is disconnected, the
// device row is deleted so the next Start shows a fresh QR, and the console is
// told the channel is unpaired.
func (c *WhatsAppNativeChannel) handleLoggedOut(evt *events.LoggedOut) {
	reason := "unlinked"
	if evt != nil && evt.OnConnect {
		reason = evt.Reason.String()
	}
	logger.WarnCF("whatsapp", "WhatsApp session logged out", map[string]any{"reason": reason})

	c.loggedOut.Store(true)

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client != nil {
		client.Disconnect()
		if client.Store != nil {
			// Delete rather than keep: these keys no longer authenticate
			// anything, and leaving them on disk only preserves credential
			// material for a session that cannot be resumed.
			if err := client.Store.Delete(context.Background()); err != nil {
				logger.WarnCF("whatsapp", "Could not delete revoked WhatsApp session", map[string]any{
					"error": err.Error(),
				})
			}
		}
	}

	c.SetRunning(false)
	_ = c.pairing.PublishState(pairing.StateLoggedOut, reason)
}

// IsAllowedSender overrides the BaseChannel default, which treats an empty
// allow-list as allow-all.
//
// That default is wrong here. Inbound WhatsApp text reaches an agent holding
// shell and Python tools, and this transport is linked to the user's personal
// account, so an unconfigured channel must accept nobody rather than everybody.
// An empty allow_from denies.
func (c *WhatsAppNativeChannel) IsAllowedSender(sender bus.SenderInfo) bool {
	if len(c.allowList) == 0 {
		logger.WarnCF("whatsapp", "Inbound denied: no allow_from configured", map[string]any{
			"channel": "whatsapp",
			"status":  "denied",
		})
		return false
	}

	return c.BaseChannel.IsAllowedSender(sender)
}

// normalizeAllowList reduces phone-number entries to bare digits so they match
// the identity handleIncoming builds. Anything that is not a phone number — a
// canonical "whatsapp:..." entry, or "*" — is passed through untouched.
func normalizeAllowList(entries []string) []string {
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if number := whatsAppNumber(entry); number != "" {
			out = append(out, number)
			continue
		}
		out = append(out, entry)
	}
	return out
}

// isSelfChat reports whether an inbound message is the user talking to
// themselves in the configured Self-Chat.
//
// Both halves must match the configured number: the chat it arrived in and the
// account that sent it. Requiring both is what separates "the user messaged
// their own notes" from "somebody messaged the user" and from "the user sent
// something to a contact", which a linked device also sees.
//
// A group, broadcast, newsletter or status JID fails on the chat half, because
// whatsAppNumber only yields digits for the phone-number server.
func (c *WhatsAppNativeChannel) isSelfChat(info types.MessageInfo) bool {
	if c.selfNumber == "" {
		return false
	}
	if info.Chat.Server != types.DefaultUserServer {
		return false
	}
	return whatsAppNumber(info.Chat.String()) == c.selfNumber &&
		whatsAppNumber(info.Sender.String()) == c.selfNumber
}

// whatsAppNumber extracts the subscriber number from a JID or a typed phone
// number. It returns "" for anything that is not all digits once the JID
// decoration is removed, so a @lid identity or a username never collides with
// a number.
func whatsAppNumber(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if idx := strings.IndexByte(v, '@'); idx >= 0 {
		// Only a phone-number server carries a subscriber number; @lid and
		// friends use an opaque id that must not be compared as one.
		if v[idx+1:] != types.DefaultUserServer {
			return ""
		}
		v = v[:idx]
	}
	// Strip the device/agent suffix: "20100...:5" -> "20100...".
	if idx := strings.IndexByte(v, ':'); idx >= 0 {
		v = v[:idx]
	}
	v = strings.TrimPrefix(v, "+")
	if v == "" {
		return ""
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return v
}

func (c *WhatsAppNativeChannel) handleIncoming(evt *events.Message) {
	if evt.Message == nil {
		return
	}
	// Self-Chat only, decided from routing metadata before the body is read.
	//
	// A linked device receives account-level traffic for every chat, group and
	// broadcast the user is in. PocketClaw is allowed to read exactly one
	// conversation — the user's own — so everything else is dropped here, at
	// the top, where the decision needs nothing but evt.Info. Nothing below
	// this line runs for another person's message: no body, no bus, no agent,
	// no provider call.
	if !c.isSelfChat(evt.Info) {
		// Metadata only, and deliberately coarse: naming the chat or the
		// sender would record who the user talks to, which is the very thing
		// this boundary exists to avoid.
		logger.InfoCF("whatsapp", "Inbound ignored", map[string]any{
			"channel": "whatsapp_agent",
			"scope":   "other",
			"status":  "ignored",
		})
		return
	}

	senderID := c.selfNumber
	chatID := evt.Info.Chat.String()
	content := evt.Message.GetConversation()
	if content == "" && evt.Message.ExtendedTextMessage != nil {
		content = evt.Message.ExtendedTextMessage.GetText()
	}
	content = utils.SanitizeMessageContent(content)

	if content == "" {
		return
	}

	metadata := map[string]string{
		"message_id": evt.Info.ID,
		"peer_kind":  "direct",
		"peer_id":    senderID,
	}
	if evt.Info.PushName != "" {
		metadata["user_name"] = evt.Info.PushName
	}

	sender := bus.SenderInfo{
		Platform:    "whatsapp",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("whatsapp", senderID),
		DisplayName: evt.Info.PushName,
	}

	if !c.IsAllowedSender(sender) {
		logger.InfoCF("whatsapp", "Inbound denied", map[string]any{
			"channel":   "whatsapp",
			"chat_type": "direct",
			"status":    "denied",
		})
		return
	}

	// Metadata only. The message body belongs to the user's personal WhatsApp
	// account and never reaches PocketClaw's Logs screen; the length is enough
	// to tell "arrived" from "arrived empty" when diagnosing the channel.
	logger.InfoCF("whatsapp", "Inbound received", map[string]any{
		"channel":       "whatsapp",
		"chat_type":     "direct",
		"message_chars": len([]rune(content)),
		"status":        "received",
	})

	inboundCtx := bus.InboundContext{
		Channel:   "whatsapp",
		ChatID:    chatID,
		SenderID:  senderID,
		MessageID: evt.Info.ID,
		ChatType:  "direct",
		Raw:       metadata,
	}

	c.HandleInboundContext(c.runCtx, chatID, content, nil, inboundCtx, sender)
}

func (c *WhatsAppNativeChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return nil, fmt.Errorf("whatsapp connection not established: %w", channels.ErrTemporary)
	}

	// Detect unpaired state: the client is connected (to WhatsApp servers)
	// but has not completed QR-login yet, so sending would fail.
	if client.Store.ID == nil {
		return nil, fmt.Errorf("whatsapp not yet paired (QR login pending): %w", channels.ErrTemporary)
	}

	to, err := c.allowedTarget(msg.ChatID)
	if err != nil {
		return nil, err
	}

	waMsg := &waE2E.Message{
		Conversation: proto.String(msg.Content),
	}

	if _, err = client.SendMessage(ctx, to, waMsg); err != nil {
		return nil, fmt.Errorf("whatsapp send: %w", channels.ErrTemporary)
	}

	// Metadata only, for the same reason as inbound: the reply text is the
	// user's WhatsApp conversation, not diagnostic material.
	logger.InfoCF("whatsapp", "Outbound sent", map[string]any{
		"channel":       "whatsapp",
		"message_chars": len([]rune(msg.Content)),
		"status":        "sent",
	})
	return nil, nil
}

// allowedTarget resolves an outbound chat id and refuses anything that is not
// the configured Self-Chat.
//
// This is the outbound boundary, enforced below the model. PocketClaw's
// WhatsApp channel writes to exactly one conversation: the user's own. A prompt
// cannot widen that, and neither can a crafted message that talks the agent
// into addressing a contact — the target is checked here, against the
// configured number, after everything the model influenced.
//
// Refusal is permanent rather than ErrTemporary: retrying the same wrong target
// would only send it again.
func (c *WhatsAppNativeChannel) allowedTarget(chatID string) (types.JID, error) {
	denied := func() (types.JID, error) {
		// The rejected target is not logged. It would name whoever the agent
		// was talked into addressing.
		logger.WarnCF("whatsapp", "Outbound denied: target is not the configured Self-Chat", map[string]any{
			"channel": "whatsapp_agent",
			"scope":   "other",
			"status":  "denied",
		})
		return types.JID{}, fmt.Errorf("whatsapp: refusing to send outside Self-Chat: %w", channels.ErrSendFailed)
	}

	if c.selfNumber == "" {
		return denied()
	}
	to, err := parseJID(chatID)
	if err != nil {
		return denied()
	}
	if to.Server != types.DefaultUserServer || whatsAppNumber(to.String()) != c.selfNumber {
		return denied()
	}
	return to, nil
}

// parseJID converts a chat ID (phone number or JID string) to types.JID.
func parseJID(s string) (types.JID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return types.JID{}, fmt.Errorf("empty chat id")
	}
	if strings.Contains(s, "@") {
		return types.ParseJID(s)
	}
	return types.NewJID(s, types.DefaultUserServer), nil
}
