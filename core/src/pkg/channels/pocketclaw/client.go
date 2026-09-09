package pocketclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// PocketClawClientChannel connects to a remote Pico Protocol WebSocket server.
type PocketClawClientChannel struct {
	*channels.BaseChannel
	config *config.PocketClawClientSettings
	conn   *pocketClawConn
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPocketClawClientChannel creates a new Pico Protocol client channel.
// Identity for conversations this channel opens against a remote Gateway.
const (
	clientChatIDPrefix = config.ChannelPocketClawClient + ":"
	clientSenderID     = config.ChannelPocketClaw + "-remote"

	// legacyClientChatIDPrefix is LEGACY READ-ONLY MIGRATION: a conversation id
	// minted before the realtime namespace migration. Parsed so a message
	// already in flight still resolves; never minted.
	legacyClientChatIDPrefix = config.LegacyChannelPocketClawClient + ":"
)

// clientSessionIDFromChatID recovers the remote session from a conversation id
// in either form.
func clientSessionIDFromChatID(chatID string) string {
	if rest, found := strings.CutPrefix(chatID, clientChatIDPrefix); found {
		return rest
	}
	return strings.TrimPrefix(chatID, legacyClientChatIDPrefix)
}

func NewPocketClawClientChannel(
	bc *config.Channel,
	cfg *config.PocketClawClientSettings,
	messageBus *bus.MessageBus,
) (*PocketClawClientChannel, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("%s url is required", config.ChannelPocketClawClient)
	}

	base := channels.NewBaseChannel(config.ChannelPocketClawClient, cfg, messageBus, bc.AllowFrom)

	return &PocketClawClientChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// Start dials the remote server and begins reading.
func (c *PocketClawClientChannel) Start(ctx context.Context) error {
	logger.InfoC(config.ChannelPocketClawClient, "Starting PocketClaw client channel")
	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.dial(); err != nil {
		c.cancel()
		return fmt.Errorf("%s initial connect: %w", config.ChannelPocketClawClient, err)
	}

	c.SetRunning(true)
	go c.reconnectLoop()

	logger.InfoCF(config.ChannelPocketClawClient, "Connected", map[string]any{"url": c.config.URL})
	return nil
}

// Stop closes the connection.
func (c *PocketClawClientChannel) Stop(ctx context.Context) error {
	logger.InfoC(config.ChannelPocketClawClient, "Stopping PocketClaw client channel")
	c.SetRunning(false)
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Lock()
	if c.conn != nil {
		c.conn.close()
	}
	c.mu.Unlock()
	logger.InfoC(config.ChannelPocketClawClient, "PocketClaw client channel stopped")
	return nil
}

func (c *PocketClawClientChannel) dial() error {
	header := http.Header{}
	if c.config.Token.String() != "" {
		header.Set("Authorization", "Bearer "+c.config.Token.String())
	}

	ws, resp, err := websocket.DefaultDialer.DialContext(c.ctx, c.config.URL, header)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return err
	}

	connCtx, connCancel := context.WithCancel(c.ctx)

	pc := &pocketClawConn{
		id:        uuid.New().String(),
		conn:      ws,
		sessionID: c.config.SessionID,
		cancel:    connCancel,
	}
	if pc.sessionID == "" {
		pc.sessionID = uuid.New().String()
	}

	c.mu.Lock()
	c.conn = pc
	c.mu.Unlock()

	go c.readLoop(connCtx, pc)
	return nil
}

// reconnectLoop re-dials when the connection drops.
func (c *PocketClawClientChannel) reconnectLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.Lock()
		pc := c.conn
		c.mu.Unlock()

		if pc == nil || pc.closed.Load() {
			backoff := 5 * time.Second
			logger.InfoC(config.ChannelPocketClawClient, "Reconnecting...")
			if err := c.dial(); err != nil {
				logger.WarnCF(config.ChannelPocketClawClient, "Reconnect failed", map[string]any{
					"error": err.Error(),
				})
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(backoff):
				}
				continue
			}
			logger.InfoC(config.ChannelPocketClawClient, "Reconnected")
		}

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}
}

func (c *PocketClawClientChannel) readLoop(connCtx context.Context, pc *pocketClawConn) {
	defer pc.close()

	readTimeout := time.Duration(c.config.ReadTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}

	_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	pc.conn.SetPongHandler(func(string) error {
		return pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	})

	pingInterval := time.Duration(c.config.PingInterval) * time.Second
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	go c.pingLoop(connCtx, pc, pingInterval)

	for {
		select {
		case <-connCtx.Done():
			return
		default:
		}

		_, raw, err := pc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
			) {
				logger.DebugCF(config.ChannelPocketClawClient, "Read error", map[string]any{
					"error": err.Error(),
				})
			}
			return
		}

		_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var msg PocketClawMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		c.handleInbound(pc, msg)
	}
}

func (c *PocketClawClientChannel) pingLoop(connCtx context.Context, pc *pocketClawConn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-connCtx.Done():
			return
		case <-ticker.C:
			if pc.closed.Load() {
				return
			}
			pc.writeMu.Lock()
			err := pc.conn.WriteMessage(websocket.PingMessage, nil)
			pc.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// handleInbound processes messages from the remote server.
// In client mode the server sends message.create (responses) and the client
// sends message.send (user input). We treat message.create from the server
// as inbound user messages to feed into the agent loop.
func (c *PocketClawClientChannel) handleInbound(pc *pocketClawConn, msg PocketClawMessage) {
	switch msg.Type {
	case TypePong:
		// response to our ping, ignore
	case TypeMessageCreate:
		// Server sent us a message — treat as inbound
		c.handleServerMessage(pc, msg)
	case TypeMediaCreate:
		c.handleServerMessage(pc, msg)
	default:
		logger.DebugCF(config.ChannelPocketClawClient, "Ignoring message type", map[string]any{
			"type": msg.Type,
		})
	}
}

func (c *PocketClawClientChannel) handleServerMessage(pc *pocketClawConn, msg PocketClawMessage) {
	if isThoughtPayload(msg.Payload) {
		return
	}

	content, _ := msg.Payload[PayloadKeyContent].(string)
	media, err := parseInlineImageMedia(msg.Payload)
	if err != nil {
		logger.WarnCF(config.ChannelPocketClawClient, "Ignoring invalid media payload", map[string]any{
			"error": err.Error(),
		})
		if strings.TrimSpace(content) == "" {
			return
		}
		media = nil
	}
	if strings.TrimSpace(content) == "" && len(media) == 0 {
		return
	}

	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = pc.sessionID
	}

	chatID := clientChatIDPrefix + sessionID
	senderID := clientSenderID
	sender := bus.SenderInfo{
		Platform:    config.ChannelPocketClawClient,
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID(config.ChannelPocketClawClient, senderID),
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	inboundCtx := bus.InboundContext{
		Channel:   config.ChannelPocketClawClient,
		ChatID:    chatID,
		ChatType:  "direct",
		SenderID:  senderID,
		MessageID: msg.ID,
		Raw: map[string]string{
			"platform":   config.ChannelPocketClawClient,
			"session_id": sessionID,
		},
	}

	c.HandleInboundContext(c.ctx, chatID, content, media, inboundCtx, sender)
}

// Send sends a message to the remote server.
func (c *PocketClawClientChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	c.mu.Lock()
	pc := c.conn
	c.mu.Unlock()
	if pc == nil || pc.closed.Load() {
		return nil, channels.ErrSendFailed
	}

	outMsg := newMessage(TypeMessageSend, map[string]any{
		PayloadKeyContent: msg.Content,
	})
	outMsg.SessionID = clientSessionIDFromChatID(msg.ChatID)
	return nil, pc.writeJSON(outMsg)
}

// StartTyping implements channels.TypingCapable.
func (c *PocketClawClientChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	c.mu.Lock()
	pc := c.conn
	c.mu.Unlock()
	if pc == nil || pc.closed.Load() {
		return func() {}, nil
	}

	startMsg := newMessage(TypeTypingStart, nil)
	startMsg.SessionID = clientSessionIDFromChatID(chatID)
	if err := pc.writeJSON(startMsg); err != nil {
		return func() {}, err
	}
	return func() {
		c.mu.Lock()
		currentPC := c.conn
		c.mu.Unlock()
		if currentPC == nil {
			return
		}
		stopMsg := newMessage(TypeTypingStop, nil)
		stopMsg.SessionID = clientSessionIDFromChatID(chatID)
		currentPC.writeJSON(stopMsg)
	}, nil
}
