package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

var (
	reHeading    = regexp.MustCompile(`(?m)^#{1,6}\s+([^\n]+)`)
	reBlockquote = regexp.MustCompile(`^>\s*(.*)$`)
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reBoldStar   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBoldUnder  = regexp.MustCompile(`__(.+?)__`)
	reItalic     = regexp.MustCompile(`_([^_]+)_`)
	reStrike     = regexp.MustCompile(`~~(.+?)~~`)
	reListItem   = regexp.MustCompile(`^[-*]\s+`)
	reCodeBlock  = regexp.MustCompile("```[\\w]*\\n?([\\s\\S]*?)```")
	reInlineCode = regexp.MustCompile("`([^`]+)`")
)

const (
	defaultMediaGroupDelay = 500 * time.Millisecond
	telegramCaptionLimit   = 1024
	telegramHTTPTimeout    = 45 * time.Second

	// startCleanupTTL bounds how long a received private /start may wait for
	// its built-in reply before the cosmetic cleanup is abandoned. It is a
	// timestamp check, never a sleep or a delay.
	startCleanupTTL = 2 * time.Minute
	// startCleanupTimeout bounds the best-effort deleteMessage call. A slow
	// delete must not hold the outbound path.
	startCleanupTimeout = 5 * time.Second
)

type TelegramChannel struct {
	*channels.BaseChannel
	bot       *telego.Bot
	bh        *th.BotHandler
	bc        *config.Channel
	chatIDsMu sync.Mutex

	// ownerMissing records that this channel was configured with a valid bot
	// credential but no owner identity. It is an incomplete setup, not a
	// working bot: private senders get deterministic setup guidance and no
	// agent turn is ever created, and other chat types are left unanswered.
	// It is fixed at construction; configuring an owner requires a new channel.
	ownerMissing bool
	chatIDs      map[string]int64
	ctx          context.Context
	cancel       context.CancelFunc
	tgCfg        *config.TelegramSettings
	progress     *channels.ToolFeedbackAnimator

	// pollingDone is closed when the long-polling goroutine has unwound. Stop
	// waits on it so a stopped channel can be started again.
	pollingDone chan struct{}

	// intakeRef is read by this channel's Bot API caller to learn which
	// generation's getUpdates request it is recording. It is installed before
	// UpdatesViaLongPolling, so the first poll is attributable.
	intakeRef *telegramIntakeRef
	// generation is the local id of the active getUpdates owner. Zero means no
	// owner has been established.
	generation atomic.Uint64
	// intakeProbe is replaceable only by package tests. Production waits on the
	// real caller-observed intake and fails closed when it cannot be confirmed.
	intakeProbe func(*telegramIntake, time.Duration) bool

	registerFunc      func(context.Context, []commands.Definition) error
	commandRegDelayFn func(int) time.Duration
	commandRegCancel  context.CancelFunc
	// commandsRegistered latches once the menu has reached Telegram. Read from
	// the status snapshot, so it is atomic rather than mutex-guarded: the
	// registration goroutine writes it and a health request reads it.
	commandsRegistered atomic.Bool
	// runtimeFailure is a sanitized terminal lifecycle code. It intentionally
	// carries no Telegram response, bot identity or credential material.
	runtimeFailure atomic.Uint32
	botIdentityMu  sync.RWMutex
	botIdentity    string
	// firstStartReplied makes the first successful /start response an explicit,
	// safe lifecycle fact without logging message, chat or owner data.
	firstStartReplied atomic.Bool

	// startCleanup holds, per private chat, the exact inbound /start message
	// whose built-in reply is still owed. It is consumed only after that reply
	// is sent successfully, so a /start is never deleted before PocketClaw has
	// proved it received and handled it. Cosmetic and best-effort only.
	startCleanupMu sync.Mutex
	startCleanup   map[int64]telegramStartCleanup

	// handlerReadyProbe is replaceable only by package tests. Production uses
	// the real telego handler state and fails closed when it cannot be confirmed.
	handlerReadyProbe func(updateConsumer, time.Duration) bool

	mediaGroupMu    sync.Mutex
	mediaGroups     map[string]*telegramMediaGroup
	mediaGroupDelay time.Duration
}

type telegramMediaGroup struct {
	messages   []*telego.Message
	timer      *time.Timer
	generation uint64
}

// telegramStartCleanup records the exact inbound /start message that is
// awaiting its built-in reply, bound to the generation that received it.
type telegramStartCleanup struct {
	messageID  int
	generation uint64
	recordedAt time.Time
}

type telegramMessageParts struct {
	content    []string
	mediaPaths []string
}

func NewTelegramChannel(
	bc *config.Channel,
	telegramCfg *config.TelegramSettings,
	bus *bus.MessageBus,
) (*TelegramChannel, error) {
	// The owner contract, minus the empty case. Exactly one positive numeric
	// owner authorizes the agent. Zero owners is no longer a construction
	// failure: it is the explicit owner-missing setup state, handled below
	// without ever granting agent access. Several owners, or one that is not a
	// positive numeric id, remain errors -- there is nothing sensible to pick.
	ownerMissing := false
	switch len(bc.AllowFrom) {
	case 0:
		ownerMissing = true
	case 1:
		ownerID, err := strconv.ParseInt(strings.TrimSpace(bc.AllowFrom[0]), 10, 64)
		if err != nil || ownerID <= 0 {
			return nil, fmt.Errorf("telegram owner must be exactly one positive numeric Telegram user ID")
		}
	default:
		return nil, fmt.Errorf("telegram requires exactly one paired numeric owner")
	}
	channelName := bc.Name()
	var opts []telego.BotOption

	httpClient := &http.Client{Timeout: telegramHTTPTimeout}
	if telegramCfg.Proxy != "" {
		proxyURL, parseErr := url.Parse(telegramCfg.Proxy)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", telegramCfg.Proxy, parseErr)
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	} else if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" {
		httpClient.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	// Telego otherwise defaults to fasthttp without a deadline. A lost mobile
	// connection can then block one outbound worker indefinitely while polling
	// continues accepting updates and emitting Thinking placeholders.
	//
	// PC-DEF-061. The caller is wrapped so the channel can tell when a
	// generation's first getUpdates request is actually issued. Telego starts
	// its poller goroutine and returns before that, so this is the only proof
	// that the Bot API is holding a poll for the generation being reported
	// ready.
	intakeRef := &telegramIntakeRef{}
	opts = append(opts, telego.WithAPICaller(&telegramIntakeCaller{
		base: ta.HTTPCaller{Client: httpClient},
		ref:  intakeRef,
	}))

	if baseURL := strings.TrimRight(strings.TrimSpace(telegramCfg.BaseURL), "/"); baseURL != "" {
		opts = append(opts, telego.WithAPIServer(baseURL))
	}
	opts = append(opts, telego.WithLogger(logger.NewLogger("telego")))

	bot, err := telego.NewBot(telegramCfg.Token.String(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	// Defense in depth for the owner-missing state. The configured AllowFrom is
	// empty, and BaseChannel reads an empty allowlist as "allow everyone" -- a
	// false warning here and, worse, a permissive base layer behind the
	// owner-missing branch. A non-numeric sentinel matches no real Telegram
	// user, so the base layer denies every sender even if a future path
	// bypassed the branch above.
	baseAllowFrom := bc.AllowFrom
	if ownerMissing {
		baseAllowFrom = config.FlexibleStringSlice{telegramOwnerMissingSentinel}
	}
	base := channels.NewBaseChannel(
		channelName,
		telegramCfg,
		bus,
		baseAllowFrom,
		channels.WithMaxMessageLength(4000),
		channels.WithGroupTrigger(bc.GroupTrigger),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	ch := &TelegramChannel{
		BaseChannel:  base,
		bot:          bot,
		bc:           bc,
		ownerMissing: ownerMissing,
		chatIDs:      make(map[string]int64),
		tgCfg:        telegramCfg,
		intakeRef:    intakeRef,
		startCleanup: make(map[int64]telegramStartCleanup),

		mediaGroups:     make(map[string]*telegramMediaGroup),
		mediaGroupDelay: telegramMediaGroupDelay(telegramCfg),
	}
	ch.progress = channels.NewToolFeedbackAnimator(ch.EditMessage)
	return ch, nil
}

func telegramMediaGroupDelay(telegramCfg *config.TelegramSettings) time.Duration {
	if telegramCfg != nil && telegramCfg.MediaGroupDelayMS > 0 {
		return time.Duration(telegramCfg.MediaGroupDelayMS) * time.Millisecond
	}
	return defaultMediaGroupDelay
}

func (c *TelegramChannel) Start(ctx context.Context) error {
	logger.InfoC("telegram", "Starting Telegram bot (polling mode)...")

	c.SetRunning(false)
	c.commandsRegistered.Store(false)
	c.runtimeFailure.Store(telegramRuntimeFailureNone)
	c.ctx, c.cancel = context.WithCancel(ctx)

	// PC-DEF-061. Every activation is a distinct getUpdates owner with its own
	// intake proof. Readiness is only allowed to describe the generation whose
	// request is actually in flight.
	if c.intakeRef == nil {
		c.intakeRef = &telegramIntakeRef{}
	}
	generation := nextTelegramGeneration()
	intake := newTelegramIntake(generation)
	c.intakeRef.store(intake)
	c.generation.Store(uint64(generation))
	logTelegramLifecycleGeneration("generation_created", uint64(generation))

	// Authentication is established before polling is allowed to own intake.
	// This call cannot lose an update because no getUpdates request exists yet.
	// It also gives invalid replacement credentials a synchronous terminal path
	// rather than letting Telego's generic polling retry loop own the outcome.
	me, err := c.bot.GetMe(c.ctx)
	if err != nil {
		if errors.Is(err, errTelegramAuthentication) {
			c.runtimeFailure.Store(telegramRuntimeFailureAuthentication)
			logger.ErrorCF("telegram", "Telegram rejected the configured bot credentials", map[string]any{
				"event":               "telegram.authentication_failed",
				"telegram_generation": uint64(generation),
			})
		}
		c.refuseStart(uint64(generation), nil)
		if errors.Is(err, errTelegramAuthentication) {
			return errTelegramAuthentication
		}
		return fmt.Errorf("telegram authentication check failed: %w", err)
	}
	c.botIdentityMu.Lock()
	c.botIdentity = me.Username
	c.botIdentityMu.Unlock()
	// From here on a 401 is asynchronous (polling or menu registration), so
	// this generation owns the callback that revokes readiness and retires it.
	intake.onUnauthorized = func() { c.failAuthentication(uint64(generation)) }

	logger.DebugCF("telegram", "Telegram polling starting", map[string]any{
		"event": "polling.prepare",
	})
	// Offset is deliberately unset, which asks Telegram for everything it still
	// holds. PocketClaw persists no offset of its own, so this is the only
	// starting point there is and a replaced bot cannot inherit one.
	updates, err := c.bot.UpdatesViaLongPolling(c.ctx, &telego.GetUpdatesParams{
		Timeout: 30,
	})
	if err != nil {
		c.refuseStart(uint64(generation), nil)
		return fmt.Errorf("failed to start long polling: %w", err)
	}
	logger.DebugCF("telegram", "Telegram polling started", map[string]any{
		"event":                "polling.started",
		"poll_timeout_seconds": 30,
	})
	logTelegramLifecycleGeneration("poller_created", uint64(generation))
	logTelegramLifecycle("polling_live")

	bh, err := th.NewBotHandler(c.bot, c.observeUpdates(updates))
	if err != nil {
		c.refuseStart(uint64(generation), nil)
		return fmt.Errorf("failed to create bot handler: %w", err)
	}
	c.bh = bh

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.handleMessage(ctx, &message)
	}, th.AnyMessage())
	logTelegramLifecycleGeneration("consumer_attached", uint64(generation))

	// PC-DEF-061. The consumer goes live here, before anything else and before
	// Running is reported. Only allocation separates it from the poller now:
	// polling confirms updates to Telegram as it fetches them, so a blocking
	// call in this gap -- getMe used to sit here, at four seconds on the device
	// -- is a window in which a delivered update is already unrecoverable.
	go func() {
		if err := bh.Start(); err != nil {
			logger.ErrorCF("telegram", "Bot handler failed", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	readyProbe := c.handlerReadyProbe
	if readyProbe == nil {
		readyProbe = waitForHandlerConsuming
	}
	if !readyProbe(bh, handlerConsumingWait) {
		// Fail closed. Reporting Running here used to authorize the Android
		// handoff even though the receiver had not confirmed it was consuming.
		// Stop both halves before returning so a later retry cannot overlap this
		// unconfirmed poller.
		logger.WarnCF("telegram",
			"Telegram intake did not report consuming; channel start refused",
			map[string]any{"event": "polling.ready_unconfirmed"})
		c.refuseStart(uint64(generation), bh)
		return fmt.Errorf("telegram update handler did not become ready")
	}

	logTelegramLifecycleGeneration("handler_ready", uint64(generation))

	// PC-DEF-061. Handler consumption and a request handed to the transport are
	// still insufficient: Telegram must successfully answer this generation's
	// first getUpdates call. The API caller makes only that first call a short
	// poll, so success is immediate without changing the steady 30-second poll.
	var intakeErr error
	if c.intakeProbe != nil {
		if !c.intakeProbe(intake, telegramIntakeWait) {
			intakeErr = fmt.Errorf("telegram getUpdates intake did not become usable")
		}
	} else {
		intakeErr = waitForIntakeUsable(intake, telegramIntakeWait)
	}
	if intakeErr != nil {
		logger.WarnCF("telegram",
			"Telegram getUpdates intake did not become usable; channel start refused",
			map[string]any{
				"event":               "polling.intake_unconfirmed",
				"telegram_generation": uint64(generation),
			})
		c.refuseStart(uint64(generation), bh)
		if errors.Is(intakeErr, errTelegramAuthentication) {
			return errTelegramAuthentication
		}
		return intakeErr
	}

	c.SetRunning(true)
	logger.InfoC("telegram", "Telegram bot connected")
	logger.DebugCF("telegram", "Telegram polling ready", map[string]any{
		"event":               "polling.ready",
		"telegram_generation": uint64(generation),
	})

	c.startCommandRegistration(c.ctx, commands.BuiltinDefinitions())

	logger.InfoCF("telegram", "Telegram bot identified", map[string]any{
		"event": "polling.identity",
		"bot":   c.botUsername(),
	})

	return nil
}

const (
	telegramRuntimeFailureNone uint32 = iota
	telegramRuntimeFailureAuthentication
)

// RuntimeFailure reports a sanitized terminal code for readiness. It remains
// latched after the failed generation is retired and is cleared only by a new
// Start attempt.
func (c *TelegramChannel) RuntimeFailure() string {
	if c.runtimeFailure.Load() == telegramRuntimeFailureAuthentication {
		return "authentication_failed"
	}
	return ""
}

// failAuthentication owns the asynchronous 401 path (getUpdates and command
// registration). It immediately revokes Running, cancels every loop, then
// retires the exact generation after its poller confirms exit.
func (c *TelegramChannel) failAuthentication(generation uint64) {
	if generation == 0 || c.generation.Load() != generation {
		return
	}
	c.runtimeFailure.Store(telegramRuntimeFailureAuthentication)
	c.SetRunning(false)
	c.commandsRegistered.Store(false)
	if c.commandRegCancel != nil {
		c.commandRegCancel()
	}
	if c.cancel != nil {
		c.cancel()
	}
	logger.ErrorCF("telegram", "Telegram rejected the configured bot credentials", map[string]any{
		"event":               "telegram.authentication_failed",
		"telegram_generation": generation,
	})
	bh := c.bh
	go c.retireFailedGeneration(generation, bh)
}

func (c *TelegramChannel) refuseStart(generation uint64, bh *th.BotHandler) {
	if c.cancel != nil {
		c.cancel()
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), pollingStopWait)
	defer stopCancel()
	if bh != nil {
		_ = bh.StopWithContext(stopCtx)
	}
	c.awaitPollingStopped(stopCtx)
	if c.generation.CompareAndSwap(generation, 0) {
		logTelegramLifecycleGeneration("generation_retired", generation)
	}
}

func (c *TelegramChannel) retireFailedGeneration(generation uint64, bh *th.BotHandler) {
	logTelegramLifecycleGeneration("generation_retiring", generation)
	stopCtx, stopCancel := context.WithTimeout(context.Background(), pollingStopWait)
	defer stopCancel()
	if bh != nil {
		_ = bh.StopWithContext(stopCtx)
	}
	c.awaitPollingStopped(stopCtx)
	if c.generation.CompareAndSwap(generation, 0) {
		logTelegramLifecycleGeneration("poller_exit_confirmed", generation)
		logTelegramLifecycleGeneration("generation_retired", generation)
	}
}

// PollingGeneration reports the local id of the active getUpdates owner, or
// zero when no owner has been established. It is a process-local counter and
// carries no bot, owner, chat or credential identity.
func (c *TelegramChannel) PollingGeneration() uint64 {
	return c.generation.Load()
}

// OwnerMissing reports that this channel has a valid credential but no
// configured owner, so it is in the incomplete-setup state rather than ready.
// It carries no identity: only the boolean fact.
func (c *TelegramChannel) OwnerMissing() bool {
	return c.ownerMissing
}

// telegramOwnerMissingSentinel is a non-numeric marker installed as the base
// channel's allowlist while no owner is configured. It matches no real Telegram
// user id, so the base layer denies every sender rather than reading an empty
// allowlist as open access.
const telegramOwnerMissingSentinel = "__owner_missing__"

// telegramOwnerMissingReply is the deterministic local guidance a private
// sender receives while Telegram is configured with a valid token but no
// owner. It names the sender's own numeric id -- the one fact the inbound
// update carries that the user would otherwise need a third-party bot to learn
// -- and nothing about anyone else.
//
// Core has no locale, so this is English like every other Core reply; the stable
// machine-readable state is `setup_required` / `owner_missing` on readiness, and
// the Dashboard localizes its own copy.
const telegramOwnerMissingReply = "PocketClaw is connected, but Telegram setup is incomplete.\n\n" +
	"Add your Telegram numeric ID in PocketClaw -> Telegram -> Manual setup / Allowed From.\n\n" +
	"Your Telegram numeric ID is: %d\n\n" +
	"That ID identifies which account is allowed to control this bot."

// sendOwnerMissingSetupReply delivers the setup guidance to the sender's own
// private chat. It logs nothing about the sender: not the id, not the chat.
func (c *TelegramChannel) sendOwnerMissingSetupReply(ctx context.Context, chatID, senderID int64) {
	tgMsg := tu.Message(tu.ID(chatID), fmt.Sprintf(telegramOwnerMissingReply, senderID))
	if _, err := c.bot.SendMessage(ctx, tgMsg); err != nil {
		logger.WarnC("telegram",
			"Could not deliver the Telegram owner-missing setup guidance")
	}
}

func (c *TelegramChannel) Stop(ctx context.Context) error {
	logger.InfoC("telegram", "Stopping Telegram bot...")
	c.SetRunning(false)

	generation := c.generation.Load()
	logTelegramLifecycleGeneration("generation_retiring", generation)

	// Stop the bot handler
	if c.bh != nil {
		_ = c.bh.StopWithContext(ctx)
	}
	c.flushPendingMediaGroups(ctx)

	// Cancel our context (stops long polling)
	if c.cancel != nil {
		c.cancel()
	}
	logTelegramLifecycleGeneration("poller_cancel_requested", generation)
	// And wait for it to actually stop. Returning earlier reported a channel as
	// stopped while the library still held its long-polling lock, so the next
	// Start on this channel failed and Telegram stayed down.
	c.awaitPollingStopped(ctx)
	logTelegramLifecycleGeneration("poller_exit_confirmed", generation)
	if c.progress != nil {
		c.progress.StopAll()
	}
	if c.commandRegCancel != nil {
		c.commandRegCancel()
	}
	// The generation is only retired once its poller has confirmed exit. A
	// successor must never inherit a reported generation that could still be
	// acknowledging updates.
	c.generation.Store(0)
	logTelegramLifecycleGeneration("generation_retired", generation)

	return nil
}

func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	useMarkdownV2 := c.tgCfg.UseMarkdownV2

	chatID, threadID, err := resolveTelegramOutboundTarget(msg.ChatID, &msg.Context)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID %s: %w", msg.ChatID, channels.ErrSendFailed)
	}

	if msg.Content == "" {
		return nil, nil
	}

	isToolFeedback := outboundMessageIsToolFeedback(msg)
	toolFeedbackContent := msg.Content
	if isToolFeedback {
		toolFeedbackContent = fitToolFeedbackForTelegram(msg.Content, useMarkdownV2, 4096)
	}
	trackedChatID := telegramToolFeedbackChatKey(msg.ChatID, &msg.Context)
	if isToolFeedback {
		if msgID, handled, err := c.progress.Update(ctx, trackedChatID, toolFeedbackContent); handled {
			if err != nil {
				return nil, err
			}
			return []string{msgID}, nil
		}
	}
	trackedMsgID, hasTrackedMsg := c.currentToolFeedbackMessage(trackedChatID)
	if !isToolFeedback {
		if msgIDs, handled := c.finalizeToolFeedbackMessageForChat(ctx, trackedChatID, msg); handled {
			return msgIDs, nil
		}
	}

	// The Manager already splits messages to ≤4000 chars (WithMaxMessageLength),
	// so msg.Content is guaranteed to be within that limit. We still need to
	// check if HTML expansion pushes it beyond Telegram's 4096-char API limit.
	replyToID := msg.ReplyToMessageID
	var messageIDs []string
	queue := []string{msg.Content}
	if isToolFeedback {
		queue = []string{channels.InitialAnimatedToolFeedbackContent(toolFeedbackContent)}
	}
	for len(queue) > 0 {
		chunk := queue[0]
		queue = queue[1:]

		content := parseContent(chunk, useMarkdownV2)

		if len([]rune(content)) > 4096 {
			if isToolFeedback {
				fittedChunk := fitToolFeedbackForTelegram(chunk, useMarkdownV2, 4096)
				if fittedChunk != "" && fittedChunk != chunk {
					queue = append([]string{fittedChunk}, queue...)
					continue
				}
			}
			runeChunk := []rune(chunk)
			ratio := float64(len(runeChunk)) / float64(len([]rune(content)))
			smallerLen := int(float64(4096) * ratio * 0.95) // 5% safety margin

			// Guarantee progress: if estimated length is >= chunk length, force it smaller
			if smallerLen >= len(runeChunk) {
				smallerLen = len(runeChunk) - 1
			}

			if smallerLen <= 0 {
				msgID, err := c.sendChunk(ctx, sendChunkParams{
					chatID:        chatID,
					threadID:      threadID,
					content:       content,
					replyToID:     replyToID,
					mdFallback:    chunk,
					useMarkdownV2: useMarkdownV2,
				})
				if err != nil {
					return nil, err
				}
				messageIDs = append(messageIDs, msgID)
				replyToID = ""
				continue
			}

			// Use the estimated smaller length as a guide for SplitMessage.
			// SplitMessage will find natural break points (newlines/spaces) and respect code blocks.
			subChunks := channels.SplitMessage(chunk, smallerLen)

			// Safety fallback: If SplitMessage failed to shorten the chunk, force a manual hard split.
			if len(subChunks) == 1 && subChunks[0] == chunk {
				part1 := string(runeChunk[:smallerLen])
				part2 := string(runeChunk[smallerLen:])
				subChunks = []string{part1, part2}
			}

			// Filter out empty chunks to avoid sending empty messages to Telegram.
			nonEmpty := make([]string, 0, len(subChunks))
			for _, s := range subChunks {
				if s != "" {
					nonEmpty = append(nonEmpty, s)
				}
			}

			// Push sub-chunks back to the front of the queue
			queue = append(nonEmpty, queue...)
			continue
		}

		msgID, err := c.sendChunk(ctx, sendChunkParams{
			chatID:        chatID,
			threadID:      threadID,
			content:       content,
			replyToID:     replyToID,
			mdFallback:    chunk,
			useMarkdownV2: useMarkdownV2,
		})
		if err != nil {
			return nil, err
		}
		messageIDs = append(messageIDs, msgID)
		// Only the first chunk should be a reply; subsequent chunks are normal messages.
		replyToID = ""
	}

	if isToolFeedback && len(messageIDs) > 0 {
		c.RecordToolFeedbackMessage(trackedChatID, messageIDs[0], toolFeedbackContent)
	} else if !isToolFeedback && hasTrackedMsg {
		c.dismissTrackedToolFeedbackMessage(ctx, trackedChatID, trackedMsgID)
	}
	if msg.Content == commands.StartReplyText &&
		c.firstStartReplied.CompareAndSwap(false, true) {
		logTelegramLifecycle("first_start_replied")
	}
	// The reply is delivered by this point. Only now may the cosmetic /start
	// cleanup run; it is best-effort and cannot affect the reply above.
	c.cleanupStartMessage(ctx, chatID, msg.Content)

	return messageIDs, nil
}

type sendChunkParams struct {
	chatID        int64
	threadID      int
	content       string
	replyToID     string
	mdFallback    string
	useMarkdownV2 bool
}

// sendChunk sends a single HTML/MarkdownV2 message, falling back to the original
// markdown as plain text on parse failure so users never see raw HTML/MarkdownV2 tags.
func (c *TelegramChannel) sendChunk(
	ctx context.Context,
	params sendChunkParams,
) (string, error) {
	tgMsg := tu.Message(tu.ID(params.chatID), params.content)
	tgMsg.MessageThreadID = params.threadID
	if params.useMarkdownV2 {
		tgMsg.WithParseMode(telego.ModeMarkdownV2)
	} else {
		tgMsg.WithParseMode(telego.ModeHTML)
	}

	if params.replyToID != "" {
		if mid, parseErr := strconv.Atoi(params.replyToID); parseErr == nil {
			tgMsg.ReplyParameters = &telego.ReplyParameters{
				MessageID: mid,
			}
		}
	}

	pMsg, err := c.bot.SendMessage(ctx, tgMsg)
	if err != nil {
		logParseFailed(err, params.useMarkdownV2)

		tgMsg.Text = params.mdFallback
		tgMsg.ParseMode = ""
		pMsg, err = c.bot.SendMessage(ctx, tgMsg)
		if err != nil {
			return "", fmt.Errorf("telegram send: %w", channels.ErrTemporary)
		}
	}

	return strconv.Itoa(pMsg.MessageID), nil
}

// maxTypingDuration limits how long the typing indicator can run.
// Prevents endless typing when the LLM fails/hangs and preSend never invokes cancel.
// Matches channels.Manager's typingStopTTL (5 min) so behavior is consistent.
const maxTypingDuration = 5 * time.Minute

// StartTyping implements channels.TypingCapable.
// It sends ChatAction(typing) immediately and then repeats every 4 seconds
// (Telegram's typing indicator expires after ~5s) in a background goroutine.
// The returned stop function is idempotent and cancels the goroutine.
// The goroutine also exits automatically after maxTypingDuration if cancel is
// never called (e.g. when the LLM fails or times out without publishing).
func (c *TelegramChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	cid, threadID, err := parseTelegramChatID(chatID)
	if err != nil {
		return func() {}, err
	}

	action := tu.ChatAction(tu.ID(cid), telego.ChatActionTyping)
	action.MessageThreadID = threadID

	// Send the first typing action immediately
	_ = c.bot.SendChatAction(ctx, action)

	typingCtx, cancel := context.WithCancel(ctx)
	// Cap lifetime so the goroutine cannot run indefinitely if cancel is never called
	maxCtx, maxCancel := context.WithTimeout(typingCtx, maxTypingDuration)
	go func() {
		defer maxCancel()
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-maxCtx.Done():
				return
			case <-ticker.C:
				a := tu.ChatAction(tu.ID(cid), telego.ChatActionTyping)
				a.MessageThreadID = threadID
				_ = c.bot.SendChatAction(typingCtx, a)
			}
		}
	}()

	return cancel, nil
}

// EditMessage implements channels.MessageEditor.
func (c *TelegramChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	useMarkdownV2 := c.tgCfg.UseMarkdownV2
	cid, _, err := parseTelegramChatID(chatID)
	if err != nil {
		return err
	}
	mid, err := strconv.Atoi(messageID)
	if err != nil {
		return err
	}
	parsedContent := parseContent(content, useMarkdownV2)
	editMsg := tu.EditMessageText(tu.ID(cid), mid, parsedContent)
	if useMarkdownV2 {
		editMsg.WithParseMode(telego.ModeMarkdownV2)
	} else {
		editMsg.WithParseMode(telego.ModeHTML)
	}
	_, err = c.bot.EditMessageText(ctx, editMsg)
	if err != nil {
		// If it failed because it was already modified (likely from a previous
		// attempt that timed out on our end but landed on Telegram), we treat
		// it as success to prevent the Manager from sending a duplicate message.
		if strings.Contains(err.Error(), "message is not modified") {
			return nil
		}

		// Only fallback to plain text if the error looks like a parsing failure (Bad Request).
		// Network errors or timeouts should NOT trigger a retry with different content.
		if strings.Contains(err.Error(), "Bad Request") {
			logParseFailed(err, useMarkdownV2)
			_, err = c.bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(cid), mid, content))
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return nil
		}

	}

	return err
}

// DeleteMessage implements channels.MessageDeleter.
func (c *TelegramChannel) DeleteMessage(ctx context.Context, chatID string, messageID string) error {
	cid, _, err := parseTelegramChatID(chatID)
	if err != nil {
		return err
	}
	mid, err := strconv.Atoi(messageID)
	if err != nil {
		return err
	}
	return c.bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    tu.ID(cid),
		MessageID: mid,
	})
}

func outboundMessageIsToolFeedback(msg bus.OutboundMessage) bool {
	if len(msg.Context.Raw) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(msg.Context.Raw["message_kind"]), "tool_feedback")
}

func (c *TelegramChannel) currentToolFeedbackMessage(chatID string) (string, bool) {
	if c.progress == nil {
		return "", false
	}
	return c.progress.Current(chatID)
}

func (c *TelegramChannel) takeToolFeedbackMessage(chatID string) (string, string, bool) {
	if c.progress == nil {
		return "", "", false
	}
	return c.progress.Take(chatID)
}

func (c *TelegramChannel) RecordToolFeedbackMessage(chatID, messageID, content string) {
	if c.progress == nil {
		return
	}
	c.progress.Record(chatID, messageID, content)
}

func (c *TelegramChannel) ClearToolFeedbackMessage(chatID string) {
	if c.progress == nil {
		return
	}
	c.progress.Clear(chatID)
}

func (c *TelegramChannel) DismissToolFeedbackMessage(ctx context.Context, chatID string) {
	msgID, ok := c.currentToolFeedbackMessage(chatID)
	if !ok {
		return
	}
	c.dismissTrackedToolFeedbackMessage(ctx, chatID, msgID)
}

func (c *TelegramChannel) dismissTrackedToolFeedbackMessage(ctx context.Context, chatID, messageID string) {
	if strings.TrimSpace(chatID) == "" || strings.TrimSpace(messageID) == "" {
		return
	}
	c.ClearToolFeedbackMessage(chatID)
	_ = c.DeleteMessage(ctx, chatID, messageID)
}

func (c *TelegramChannel) finalizeTrackedToolFeedbackMessage(
	ctx context.Context,
	chatID string,
	content string,
	editFn func(context.Context, string, string, string) error,
) ([]string, bool) {
	msgID, baseContent, ok := c.takeToolFeedbackMessage(chatID)
	if !ok || editFn == nil {
		return nil, false
	}
	if err := editFn(ctx, chatID, msgID, content); err != nil {
		c.RecordToolFeedbackMessage(chatID, msgID, baseContent)
		return nil, false
	}
	return []string{msgID}, true
}

func (c *TelegramChannel) FinalizeToolFeedbackMessage(ctx context.Context, msg bus.OutboundMessage) ([]string, bool) {
	if outboundMessageIsToolFeedback(msg) {
		return nil, false
	}
	return c.finalizeToolFeedbackMessageForChat(ctx, telegramToolFeedbackChatKey(msg.ChatID, &msg.Context), msg)
}

func (c *TelegramChannel) finalizeToolFeedbackMessageForChat(
	ctx context.Context,
	chatID string,
	msg bus.OutboundMessage,
) ([]string, bool) {
	return c.finalizeTrackedToolFeedbackMessage(ctx, chatID, msg.Content, c.EditMessage)
}

// SendPlaceholder implements channels.PlaceholderCapable.
// It sends a placeholder message (e.g. "Thinking... 💭") that will later be
// edited to the actual response via EditMessage (channels.MessageEditor).
func (c *TelegramChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	phCfg := c.bc.Placeholder
	if !phCfg.Enabled {
		return "", nil
	}

	text := phCfg.GetRandomText()

	cid, threadID, err := parseTelegramChatID(chatID)
	if err != nil {
		return "", err
	}

	phMsg := tu.Message(tu.ID(cid), text)
	phMsg.MessageThreadID = threadID
	pMsg, err := c.bot.SendMessage(ctx, phMsg)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", pMsg.MessageID), nil
}

// SendMedia implements the channels.MediaSender interface.
func (c *TelegramChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	trackedChatID := telegramToolFeedbackChatKey(msg.ChatID, &msg.Context)
	trackedMsgID, hasTrackedMsg := c.currentToolFeedbackMessage(trackedChatID)

	chatID, threadID, err := resolveTelegramOutboundTarget(msg.ChatID, &msg.Context)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID %s: %w", msg.ChatID, channels.ErrSendFailed)
	}

	store := c.GetMediaStore()
	if store == nil {
		return nil, fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	var messageIDs []string
	leadingCaption := telegramLeadingCaption(msg.Parts)
	if len([]rune(leadingCaption)) > telegramCaptionLimit {
		leadingIDs, leadingErr := c.sendCaptionText(ctx, chatID, threadID, leadingCaption)
		if leadingErr != nil {
			return nil, leadingErr
		}
		messageIDs = append(messageIDs, leadingIDs...)
		msg = telegramClearMediaCaptions(msg)
	}

	if len(msg.Parts) > 1 && telegramCanSendMediaGroup(msg.Parts) {
		groupIDs, err := c.sendImageMediaGroups(ctx, chatID, threadID, store, msg.Parts)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to send media group", map[string]any{
				"count": len(msg.Parts),
				"error": err.Error(),
			})
			return nil, fmt.Errorf("telegram send media group: %w", channels.ErrTemporary)
		}
		if len(groupIDs) > 0 {
			messageIDs = append(messageIDs, groupIDs...)
			if hasTrackedMsg {
				c.dismissTrackedToolFeedbackMessage(ctx, trackedChatID, trackedMsgID)
			}
			return messageIDs, nil
		}
	}

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		file, err := os.Open(localPath)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to open media file", map[string]any{
				"path":  localPath,
				"error": err.Error(),
			})
			continue
		}

		var tgResult *telego.Message
		switch part.Type {
		case "image":
			params := &telego.SendPhotoParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Photo:           telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			tgResult, err = c.bot.SendPhoto(ctx, params)
			if err != nil && strings.Contains(err.Error(), "PHOTO_INVALID_DIMENSIONS") {
				if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
					file.Close()
					return nil, fmt.Errorf("telegram rewind media after photo failure: %w", channels.ErrTemporary)
				}

				docParams := &telego.SendDocumentParams{
					ChatID:          tu.ID(chatID),
					MessageThreadID: threadID,
					Document:        telego.InputFile{File: file},
					Caption:         part.Caption,
				}
				tgResult, err = c.bot.SendDocument(ctx, docParams)
			}
		case "audio":
			// Send OGG files with "voice" in the filename as Telegram voice
			// bubbles (SendVoice) instead of audio attachments (SendAudio).
			fn := strings.ToLower(part.Filename)
			if strings.Contains(fn, "voice") && (strings.HasSuffix(fn, ".ogg") || strings.HasSuffix(fn, ".oga")) {
				vparams := &telego.SendVoiceParams{
					ChatID:          tu.ID(chatID),
					MessageThreadID: threadID,
					Voice:           telego.InputFile{File: file},
					Caption:         part.Caption,
				}
				tgResult, err = c.bot.SendVoice(ctx, vparams)
			} else {
				params := &telego.SendAudioParams{
					ChatID:          tu.ID(chatID),
					MessageThreadID: threadID,
					Audio:           telego.InputFile{File: file},
					Caption:         part.Caption,
				}
				tgResult, err = c.bot.SendAudio(ctx, params)
			}
		case "video":
			params := &telego.SendVideoParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Video:           telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			tgResult, err = c.bot.SendVideo(ctx, params)
		default: // "file" or unknown types
			params := &telego.SendDocumentParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Document:        telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			tgResult, err = c.bot.SendDocument(ctx, params)
		}

		if tgResult != nil {
			messageIDs = append(messageIDs, strconv.Itoa(tgResult.MessageID))
		}
		file.Close()

		if err != nil {
			logger.ErrorCF("telegram", "Failed to send media", map[string]any{
				"type":  part.Type,
				"error": err.Error(),
			})
			return nil, fmt.Errorf("telegram send media: %w", channels.ErrTemporary)
		}
	}

	if hasTrackedMsg {
		c.dismissTrackedToolFeedbackMessage(ctx, trackedChatID, trackedMsgID)
	}

	return messageIDs, nil
}

func telegramCanSendMediaGroup(parts []bus.MediaPart) bool {
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part.Type != "image" {
			return false
		}
	}
	return true
}

func (c *TelegramChannel) sendImageMediaGroups(
	ctx context.Context,
	chatID int64,
	threadID int,
	store media.MediaStore,
	parts []bus.MediaPart,
) ([]string, error) {
	const maxGroupSize = 10

	messageIDs := make([]string, 0, len(parts))
	for start := 0; start < len(parts); start += maxGroupSize {
		end := start + maxGroupSize
		if end > len(parts) {
			end = len(parts)
		}
		groupIDs, err := c.sendSingleImageMediaGroup(ctx, chatID, threadID, store, parts[start:end])
		if err != nil {
			return nil, err
		}
		messageIDs = append(messageIDs, groupIDs...)
	}
	return messageIDs, nil
}

func (c *TelegramChannel) sendSingleImageMediaGroup(
	ctx context.Context,
	chatID int64,
	threadID int,
	store media.MediaStore,
	parts []bus.MediaPart,
) ([]string, error) {
	opened := make([]*os.File, 0, len(parts))
	defer func() {
		for _, file := range opened {
			file.Close()
		}
	}()

	inputMedia := make([]telego.InputMedia, 0, len(parts))
	for i, part := range parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to resolve media ref for media group", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			return nil, err
		}

		file, err := os.Open(localPath)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to open media file for media group", map[string]any{
				"path":  localPath,
				"error": err.Error(),
			})
			return nil, err
		}
		opened = append(opened, file)

		mediaItem := &telego.InputMediaPhoto{
			Type:  telego.MediaTypePhoto,
			Media: telego.InputFile{File: file},
		}
		if i == 0 {
			mediaItem.Caption = part.Caption
		}
		inputMedia = append(inputMedia, mediaItem)
	}

	results, err := c.bot.SendMediaGroup(ctx, &telego.SendMediaGroupParams{
		ChatID:          tu.ID(chatID),
		MessageThreadID: threadID,
		Media:           inputMedia,
	})
	if err != nil {
		return nil, err
	}

	messageIDs := make([]string, 0, len(results))
	for _, result := range results {
		messageIDs = append(messageIDs, strconv.Itoa(result.MessageID))
	}
	return messageIDs, nil
}

func (c *TelegramChannel) sendCaptionText(
	ctx context.Context,
	chatID int64,
	threadID int,
	text string,
) ([]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	chunks := channels.SplitMessage(text, c.MaxMessageLength())
	messageIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		msgID, err := c.sendChunk(ctx, sendChunkParams{
			chatID:        chatID,
			threadID:      threadID,
			content:       chunk,
			mdFallback:    chunk,
			useMarkdownV2: false,
		})
		if err != nil {
			return nil, err
		}
		messageIDs = append(messageIDs, msgID)
	}
	return messageIDs, nil
}

func telegramLeadingCaption(parts []bus.MediaPart) string {
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0].Caption)
}

func telegramClearMediaCaptions(msg bus.OutboundMediaMessage) bus.OutboundMediaMessage {
	if len(msg.Parts) == 0 {
		return msg
	}
	cloned := msg
	cloned.Parts = append([]bus.MediaPart(nil), msg.Parts...)
	for i := range cloned.Parts {
		cloned.Parts[i].Caption = ""
	}
	return cloned
}

func (c *TelegramChannel) handleMessage(ctx context.Context, message *telego.Message) error {
	if message != nil && strings.TrimSpace(message.MediaGroupID) != "" {
		return c.bufferMediaGroupMessage(ctx, message)
	}
	return c.handleMessages(ctx, []*telego.Message{message})
}

func (c *TelegramChannel) bufferMediaGroupMessage(ctx context.Context, message *telego.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}
	groupID := strings.TrimSpace(message.MediaGroupID)
	if groupID == "" {
		return c.handleMessages(ctx, []*telego.Message{message})
	}

	msgCopy := *message
	msgCopy.Photo = append([]telego.PhotoSize(nil), message.Photo...)
	key := fmt.Sprintf("%d:%s", message.Chat.ID, groupID)

	c.mediaGroupMu.Lock()
	if c.mediaGroups == nil {
		c.mediaGroups = make(map[string]*telegramMediaGroup)
	}
	group := c.mediaGroups[key]
	if group == nil {
		group = &telegramMediaGroup{}
		c.mediaGroups[key] = group
	}
	group.messages = append(group.messages, &msgCopy)
	group.generation++
	generation := group.generation
	if group.timer != nil {
		group.timer.Stop()
	}
	delay := c.mediaGroupDelay
	if delay <= 0 {
		delay = defaultMediaGroupDelay
	}
	group.timer = time.AfterFunc(delay, func() {
		c.flushMediaGroup(c.ctx, key, generation)
	})
	c.mediaGroupMu.Unlock()

	logger.DebugCF("telegram", "Buffered media group message", map[string]any{
		"chat_id":        message.Chat.ID,
		"media_group_id": groupID,
		"message_id":     message.MessageID,
	})
	return nil
}

func (c *TelegramChannel) flushPendingMediaGroups(ctx context.Context) {
	c.mediaGroupMu.Lock()
	keys := make([]string, 0, len(c.mediaGroups))
	for key, group := range c.mediaGroups {
		if group.timer != nil {
			group.timer.Stop()
		}
		keys = append(keys, key)
	}
	c.mediaGroupMu.Unlock()

	for _, key := range keys {
		c.flushMediaGroup(ctx, key, 0)
	}
}

func (c *TelegramChannel) flushMediaGroup(ctx context.Context, key string, generation uint64) {
	c.mediaGroupMu.Lock()
	group := c.mediaGroups[key]
	if group == nil {
		c.mediaGroupMu.Unlock()
		return
	}
	if generation != 0 && group.generation != generation {
		c.mediaGroupMu.Unlock()
		return
	}
	delete(c.mediaGroups, key)
	if group.timer != nil {
		group.timer.Stop()
	}
	messages := append([]*telego.Message(nil), group.messages...)
	c.mediaGroupMu.Unlock()

	if len(messages) == 0 {
		return
	}
	slices.SortFunc(messages, func(a, b *telego.Message) int {
		switch {
		case a == nil && b == nil:
			return 0
		case a == nil:
			return -1
		case b == nil:
			return 1
		default:
			return a.MessageID - b.MessageID
		}
	})
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.handleMessages(ctx, messages); err != nil {
		logger.ErrorCF("telegram", "Failed to handle media group", map[string]any{
			"key":   key,
			"error": err.Error(),
		})
	}
}

func (c *TelegramChannel) handleMessages(ctx context.Context, messages []*telego.Message) error {
	if len(messages) == 0 {
		return nil
	}
	message := messages[0]
	for _, candidate := range messages {
		if candidate == nil {
			continue
		}
		if strings.TrimSpace(candidate.Text) != "" || strings.TrimSpace(candidate.Caption) != "" {
			message = candidate
			break
		}
	}
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	user := message.From
	if user == nil {
		return fmt.Errorf("message sender (user) is nil")
	}

	// The owner-missing setup state is handled before anything else reaches the
	// update. A valid credential with no owner is an incomplete setup, not a
	// working bot: it must never create an agent turn, call a provider, run a
	// tool, write session history, execute a built-in command, download media,
	// or mutate configuration. Only a private sender is answered, with the
	// deterministic setup guidance, and only their own numeric id.
	//
	// The check is before the allowlist deliberately: an empty AllowFrom makes
	// BaseChannel.IsAllowedSender permissive, so this branch is the boundary
	// that keeps an ownerless channel from ever being treated as open.
	if c.ownerMissing {
		isPrivate := strings.EqualFold(strings.TrimSpace(message.Chat.Type), "private")
		if isPrivate {
			c.sendOwnerMissingSetupReply(ctx, message.Chat.ID, user.ID)
		}
		// A group or channel gets no public setup guidance: it is left
		// unanswered rather than naming infrastructure or identity in a chat
		// the user did not control.
		logger.DebugCF("telegram", "Telegram message handled by the owner-missing setup path", map[string]any{
			"is_private": isPrivate,
		})
		return nil
	}

	platformID := fmt.Sprintf("%d", user.ID)
	sender := bus.SenderInfo{
		Platform:    "telegram",
		PlatformID:  platformID,
		CanonicalID: identity.BuildCanonicalID("telegram", platformID),
		Username:    user.Username,
		DisplayName: user.FirstName,
	}

	// check allowlist to avoid downloading attachments for rejected users
	if !c.IsAllowedSender(sender) {
		logger.DebugCF("telegram", "Message rejected by allowlist", map[string]any{
			"user_id": platformID,
		})
		return nil
	}

	chatID := message.Chat.ID
	c.chatIDsMu.Lock()
	c.chatIDs[platformID] = chatID
	c.chatIDsMu.Unlock()

	content := ""
	mediaPaths := []string{}

	chatIDStr := fmt.Sprintf("%d", chatID)
	messageIDStr := fmt.Sprintf("%d", message.MessageID)
	scope := channels.BuildMediaScope("telegram", chatIDStr, messageIDStr)

	// Helper to register a local file with the media store
	storeMedia := func(localPath, filename string) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename:      filename,
				Source:        "telegram",
				CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath // fallback: use raw path
	}

	for i, msg := range messages {
		if msg == nil {
			continue
		}
		parts := c.collectTelegramMessageParts(ctx, msg, i, len(messages), storeMedia)
		for _, part := range parts.content {
			if content != "" {
				content += "\n"
			}
			content += part
		}
		mediaPaths = append(mediaPaths, parts.mediaPaths...)
	}

	if content == "" && len(mediaPaths) == 0 {
		return nil
	}

	if content == "" {
		content = "[media only]"
	}

	// In group chats, apply unified group trigger filtering
	isMentioned := false
	if message.Chat.Type != "private" {
		isMentioned = c.isBotMentioned(message)
		if isMentioned {
			content = c.stripBotMention(content)
		}
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return nil
		}
		content = cleaned
	}

	if message.ReplyToMessage != nil {
		quotedMedia := quotedTelegramMediaRefs(
			message.ReplyToMessage,
			func(fileID, ext, filename string) string {
				localPath := c.downloadFile(ctx, fileID, ext)
				if localPath == "" {
					return ""
				}
				return storeMedia(localPath, filename)
			},
		)
		if len(quotedMedia) > 0 {
			mediaPaths = append(quotedMedia, mediaPaths...)
		}
		content = c.prependTelegramQuotedReply(content, message.ReplyToMessage)
	}

	// For forum topics, embed the thread ID as "chatID/threadID" so replies
	// route to the correct topic and each topic gets its own session.
	// Only forum groups (IsForum) are handled; regular group reply threads
	// must share one session per group.
	compositeChatID := fmt.Sprintf("%d", chatID)
	threadID := message.MessageThreadID
	if message.Chat.IsForum && threadID != 0 {
		compositeChatID = fmt.Sprintf("%d/%d", chatID, threadID)
	}

	logger.DebugCF("telegram", "Received message", map[string]any{
		"message_chars": utf8.RuneCountInString(content),
		"media_count":   len(mediaPaths),
		"has_thread":    threadID != 0,
	})

	peerKind := "direct"
	if message.Chat.Type != "private" {
		peerKind = "group"
	}
	messageID := fmt.Sprintf("%d", message.MessageID)

	metadata := map[string]string{
		"user_id":    fmt.Sprintf("%d", user.ID),
		"username":   user.Username,
		"first_name": user.FirstName,
		"is_group":   fmt.Sprintf("%t", message.Chat.Type != "private"),
	}

	inboundCtx := bus.InboundContext{
		Channel:   c.Name(),
		ChatID:    compositeChatID,
		ChatType:  peerKind,
		SenderID:  platformID,
		MessageID: messageID,
		Mentioned: isMentioned,
		Raw:       metadata,
	}
	if message.Chat.IsForum && threadID != 0 {
		inboundCtx.TopicID = fmt.Sprintf("%d", threadID)
	}
	if message.ReplyToMessage != nil {
		inboundCtx.ReplyToMessageID = fmt.Sprintf("%d", message.ReplyToMessage.MessageID)
	}

	// PC-DEF-061 UX cleanup. Note the exact inbound /start before it is
	// published, so the reply that follows can prove receipt and then remove
	// the command. This changes nothing about intake, generation ownership or
	// readiness; it only remembers the message id the reply belongs to.
	c.recordStartCleanupCandidate(message, chatID)

	c.HandleMessageWithContext(
		c.ctx,
		compositeChatID,
		content,
		mediaPaths,
		inboundCtx,
		sender,
	)
	return nil
}

// recordStartCleanupCandidate notes the exact inbound /start message in a
// private chat, bound to the active generation. The entry is acted on only
// after the built-in reply for that message is sent successfully.
//
// Groups and supergroups are deliberately excluded: deleting a command a user
// sent in a shared chat would be surprising, and the /start handshake is a
// private one.
func (c *TelegramChannel) recordStartCleanupCandidate(message *telego.Message, chatID int64) {
	if c == nil || message == nil || message.Chat.Type != "private" {
		return
	}
	name, ok := commands.CommandName(strings.TrimSpace(message.Text))
	if !ok || name != "start" {
		return
	}
	generation := c.generation.Load()
	if generation == 0 {
		// No active owner has been established, so nothing about this receipt
		// can be attributed. Record nothing.
		return
	}
	c.startCleanupMu.Lock()
	if c.startCleanup == nil {
		c.startCleanup = make(map[int64]telegramStartCleanup)
	}
	c.startCleanup[chatID] = telegramStartCleanup{
		messageID:  message.MessageID,
		generation: generation,
		recordedAt: time.Now(),
	}
	c.startCleanupMu.Unlock()
	logTelegramLifecycleGeneration("telegram_start_received", generation)
}

// cleanupStartMessage removes the user's /start message after PocketClaw has
// successfully sent the built-in reply for it.
//
// Ordering is the whole contract: this is reached only after the reply send
// succeeded, it targets the exact inbound message id recorded when the command
// was received, it acts only for the same active generation, and every failure
// is swallowed. The reply already stands, so a failed cosmetic delete must
// never fail the user turn, retry, or duplicate anything.
func (c *TelegramChannel) cleanupStartMessage(ctx context.Context, chatID int64, content string) {
	if c == nil || strings.TrimSpace(content) != commands.StartReplyText {
		return
	}
	c.startCleanupMu.Lock()
	entry, ok := c.startCleanup[chatID]
	if ok {
		delete(c.startCleanup, chatID)
	}
	c.startCleanupMu.Unlock()
	if !ok {
		return
	}
	generation := c.generation.Load()
	logTelegramLifecycleGeneration("telegram_start_reply_sent", generation)
	if generation == 0 || entry.generation != generation {
		// The reply was delivered by a generation that no longer owns intake;
		// deleting now would be cleanup for someone else's receipt.
		logTelegramLifecycleGeneration("telegram_start_cleanup_skipped", generation)
		return
	}
	if time.Since(entry.recordedAt) > startCleanupTTL {
		logTelegramLifecycleGeneration("telegram_start_cleanup_skipped", generation)
		return
	}
	logTelegramLifecycleGeneration("telegram_start_cleanup_requested", generation)
	delCtx, cancel := context.WithTimeout(ctx, startCleanupTimeout)
	defer cancel()
	if err := c.DeleteMessage(delCtx, strconv.FormatInt(chatID, 10), strconv.Itoa(entry.messageID)); err != nil {
		logger.DebugCF("telegram", "Telegram /start cleanup failed", map[string]any{
			"event":               "telegram_start_cleanup_failed",
			"telegram_generation": generation,
			"error":               err.Error(),
		})
		return
	}
	logTelegramLifecycleGeneration("telegram_start_cleanup_completed", generation)
}

func (c *TelegramChannel) collectTelegramMessageParts(
	ctx context.Context,
	msg *telego.Message,
	index int,
	total int,
	storeMedia func(localPath, filename string) string,
) telegramMessageParts {
	parts := telegramMessageParts{}
	if msg == nil {
		return parts
	}
	if text := strings.TrimSpace(msg.Text); text != "" {
		parts.content = append(parts.content, text)
	}
	if caption := strings.TrimSpace(msg.Caption); caption != "" {
		parts.content = append(parts.content, caption)
	}
	if msg.Location != nil {
		parts.content = append(parts.content, fmt.Sprintf(
			"[User location: lat=%.6f, lng=%.6f]",
			msg.Location.Latitude,
			msg.Location.Longitude,
		))
	}
	if len(msg.Photo) > 0 {
		photo := msg.Photo[len(msg.Photo)-1]
		photoPath := c.downloadPhoto(ctx, photo.FileID)
		if photoPath != "" {
			photoNumber := index + 1
			parts.mediaPaths = append(parts.mediaPaths, storeMedia(photoPath, fmt.Sprintf("photo-%d.jpg", photoNumber)))
			parts.content = append(parts.content, fmt.Sprintf("[image: photo %d]", photoNumber))
		}
	}
	if msg.Voice != nil {
		voicePath := c.downloadFile(ctx, msg.Voice.FileID, ".ogg")
		if voicePath != "" {
			parts.mediaPaths = append(
				parts.mediaPaths,
				storeMedia(voicePath, indexedMediaFilename("voice", ".ogg", index, total)),
			)
			parts.content = append(parts.content, "[voice]")
		}
	}
	if msg.Audio != nil {
		audioPath := c.downloadFile(ctx, msg.Audio.FileID, ".mp3")
		if audioPath != "" {
			filename := msg.Audio.FileName
			if strings.TrimSpace(filename) == "" {
				filename = indexedMediaFilename("audio", ".mp3", index, total)
			}
			parts.mediaPaths = append(parts.mediaPaths, storeMedia(audioPath, filename))
			parts.content = append(parts.content, "[audio]")
		}
	}
	if msg.Document != nil {
		docPath := c.downloadFile(ctx, msg.Document.FileID, "")
		if docPath != "" {
			filename := msg.Document.FileName
			if strings.TrimSpace(filename) == "" {
				filename = indexedMediaFilename("document", "", index, total)
			}
			parts.mediaPaths = append(parts.mediaPaths, storeMedia(docPath, filename))
			parts.content = append(parts.content, "[file]")
		}
	}
	return parts
}

func indexedMediaFilename(prefix, ext string, index int, total int) string {
	if total <= 1 {
		return prefix + ext
	}
	return fmt.Sprintf("%s-%d%s", prefix, index+1, ext)
}

func (c *TelegramChannel) prependTelegramQuotedReply(content string, reply *telego.Message) string {
	quoted := strings.TrimSpace(telegramQuotedContent(reply))
	if quoted == "" {
		return content
	}

	author := telegramQuotedAuthor(reply)
	role := c.telegramQuotedRole(reply)
	if strings.TrimSpace(content) == "" {
		return fmt.Sprintf("[quoted %s message from %s]: %s", role, author, quoted)
	}
	return fmt.Sprintf("[quoted %s message from %s]: %s\n\n%s", role, author, quoted, content)
}

func (c *TelegramChannel) telegramQuotedRole(message *telego.Message) string {
	if message == nil {
		return "unknown"
	}

	if message.From != nil {
		if !message.From.IsBot {
			return "user"
		}
		if c.isOwnBotUser(message.From) {
			return "assistant"
		}
		return "bot"
	}

	if message.SenderChat != nil {
		return "chat"
	}

	return "unknown"
}

func (c *TelegramChannel) isOwnBotUser(user *telego.User) bool {
	if c == nil || c.bot == nil || user == nil || !user.IsBot {
		return false
	}

	if botID := c.bot.ID(); botID != 0 && user.ID == botID {
		return true
	}

	botUsername := strings.TrimPrefix(strings.TrimSpace(c.bot.Username()), "@")
	if botUsername == "" {
		return false
	}
	return strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(user.Username), "@"), botUsername)
}

func telegramQuotedAuthor(message *telego.Message) string {
	if message == nil || message.From == nil {
		return "unknown"
	}
	if username := strings.TrimSpace(message.From.Username); username != "" {
		return username
	}
	if firstName := strings.TrimSpace(message.From.FirstName); firstName != "" {
		return firstName
	}
	return "unknown"
}

func telegramQuotedContent(message *telego.Message) string {
	if message == nil {
		return ""
	}

	var parts []string
	if text := strings.TrimSpace(message.Text); text != "" {
		parts = append(parts, text)
	}
	if caption := strings.TrimSpace(message.Caption); caption != "" {
		parts = append(parts, caption)
	}
	switch {
	case len(message.Photo) > 0:
		parts = append(parts, "[image: photo]")
	}
	switch {
	case message.Voice != nil:
		parts = append(parts, "[voice]")
	case message.Audio != nil:
		parts = append(parts, "[audio]")
	}
	if message.Document != nil {
		parts = append(parts, "[file]")
	}

	return strings.Join(parts, "\n")
}

func quotedTelegramMediaRefs(
	message *telego.Message,
	resolve func(fileID, ext, filename string) string,
) []string {
	if message == nil || resolve == nil {
		return nil
	}

	var refs []string
	if message.Voice != nil {
		if ref := resolve(message.Voice.FileID, ".ogg", "voice.ogg"); ref != "" {
			refs = append(refs, ref)
		}
	}
	if message.Audio != nil {
		if ref := resolve(message.Audio.FileID, ".mp3", "audio.mp3"); ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func (c *TelegramChannel) downloadPhoto(ctx context.Context, fileID string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get photo file", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	return c.downloadFileWithInfo(file, ".jpg")
}

func (c *TelegramChannel) downloadFileWithInfo(file *telego.File, ext string) string {
	if file.FilePath == "" {
		return ""
	}

	url := c.bot.FileDownloadURL(file.FilePath)
	logger.DebugCF("telegram", "File URL", map[string]any{"url": url})

	// Use FilePath as filename for better identification
	filename := file.FilePath + ext
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "telegram",
	})
}

func (c *TelegramChannel) downloadFile(ctx context.Context, fileID, ext string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get file", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	return c.downloadFileWithInfo(file, ext)
}

func parseContent(text string, useMarkdownV2 bool) string {
	if useMarkdownV2 {
		return markdownToTelegramMarkdownV2(text)
	}

	return markdownToTelegramHTML(text)
}

func fitToolFeedbackForTelegram(content string, useMarkdownV2 bool, maxParsedLen int) string {
	content = strings.TrimSpace(content)
	if content == "" || maxParsedLen <= 0 {
		return ""
	}
	animationSafeLen := maxParsedLen - channels.MaxToolFeedbackAnimationFrameLength()
	if animationSafeLen <= 0 {
		animationSafeLen = maxParsedLen
	}
	if len([]rune(parseContent(content, useMarkdownV2))) <= animationSafeLen {
		return content
	}

	low := 1
	high := len([]rune(content))
	best := utils.Truncate(content, 1)

	for low <= high {
		mid := (low + high) / 2
		candidate := utils.FitToolFeedbackMessage(content, mid)
		if candidate == "" {
			high = mid - 1
			continue
		}
		if len([]rune(parseContent(candidate, useMarkdownV2))) <= animationSafeLen {
			best = candidate
			low = mid + 1
			continue
		}
		high = mid - 1
	}

	return best
}

func (c *TelegramChannel) PrepareToolFeedbackMessageContent(content string) string {
	if c == nil || c.tgCfg == nil {
		return strings.TrimSpace(content)
	}
	return fitToolFeedbackForTelegram(content, c.tgCfg.UseMarkdownV2, 4096)
}

func telegramToolFeedbackChatKey(chatID string, outboundCtx *bus.InboundContext) string {
	resolvedChatID, threadID, err := resolveTelegramOutboundTarget(chatID, outboundCtx)
	if err != nil || threadID == 0 {
		return strings.TrimSpace(chatID)
	}
	return fmt.Sprintf("%d/%d", resolvedChatID, threadID)
}

func (c *TelegramChannel) ToolFeedbackMessageChatID(chatID string, outboundCtx *bus.InboundContext) string {
	return telegramToolFeedbackChatKey(chatID, outboundCtx)
}

// parseTelegramChatID splits "chatID/threadID" into its components.
// Returns threadID=0 when no "/" is present (non-forum messages).
func parseTelegramChatID(chatID string) (int64, int, error) {
	idx := strings.Index(chatID, "/")
	if idx == -1 {
		cid, err := strconv.ParseInt(chatID, 10, 64)
		return cid, 0, err
	}
	cid, err := strconv.ParseInt(chatID[:idx], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	tid, err := strconv.Atoi(chatID[idx+1:])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid thread ID in chat ID %q: %w", chatID, err)
	}
	return cid, tid, nil
}

func resolveTelegramOutboundTarget(chatID string, outboundCtx *bus.InboundContext) (int64, int, error) {
	targetChatID := strings.TrimSpace(chatID)
	if targetChatID == "" && outboundCtx != nil {
		targetChatID = strings.TrimSpace(outboundCtx.ChatID)
	}
	resolvedChatID, resolvedThreadID, err := parseTelegramChatID(targetChatID)
	if err != nil {
		return 0, 0, err
	}
	if resolvedThreadID != 0 || outboundCtx == nil {
		return resolvedChatID, resolvedThreadID, nil
	}
	topicID := strings.TrimSpace(outboundCtx.TopicID)
	if topicID == "" {
		return resolvedChatID, resolvedThreadID, nil
	}
	if threadID, convErr := strconv.Atoi(topicID); convErr == nil {
		return resolvedChatID, threadID, nil
	}
	return resolvedChatID, resolvedThreadID, nil
}

func logParseFailed(err error, useMarkdownV2 bool) {
	parsingName := "HTML"
	if useMarkdownV2 {
		parsingName = "MarkdownV2"
	}

	logger.ErrorCF("telegram",
		fmt.Sprintf("%s parse failed, falling back to plain text", parsingName),
		map[string]any{
			"error": err.Error(),
		},
	)
}

// isBotMentioned checks if the bot is mentioned in the message via entities.
func (c *TelegramChannel) isBotMentioned(message *telego.Message) bool {
	text, entities := telegramEntityTextAndList(message)
	if text == "" || len(entities) == 0 {
		return false
	}

	botUsername := ""
	if c.bot != nil {
		botUsername = c.bot.Username()
	}
	runes := []rune(text)

	for _, entity := range entities {
		entityText, ok := telegramEntityText(runes, entity)
		if !ok {
			continue
		}

		switch entity.Type {
		case telego.EntityTypeMention:
			if botUsername != "" && strings.EqualFold(entityText, "@"+botUsername) {
				return true
			}
		case telego.EntityTypeTextMention:
			if botUsername != "" && entity.User != nil && strings.EqualFold(entity.User.Username, botUsername) {
				return true
			}
		case telego.EntityTypeBotCommand:
			if isBotCommandEntityForThisBot(entityText, botUsername) {
				return true
			}
		}
	}
	return false
}

func telegramEntityTextAndList(message *telego.Message) (string, []telego.MessageEntity) {
	if message.Text != "" {
		return message.Text, message.Entities
	}
	return message.Caption, message.CaptionEntities
}

func telegramEntityText(runes []rune, entity telego.MessageEntity) (string, bool) {
	if entity.Offset < 0 || entity.Length <= 0 {
		return "", false
	}
	end := entity.Offset + entity.Length
	if entity.Offset >= len(runes) || end > len(runes) {
		return "", false
	}
	return string(runes[entity.Offset:end]), true
}

func isBotCommandEntityForThisBot(entityText, botUsername string) bool {
	if !strings.HasPrefix(entityText, "/") {
		return false
	}
	command := strings.TrimPrefix(entityText, "/")
	if command == "" {
		return false
	}

	at := strings.IndexRune(command, '@')
	if at == -1 {
		// A bare /command delivered to this bot is intended for this bot.
		return true
	}

	mentionUsername := command[at+1:]
	if mentionUsername == "" || botUsername == "" {
		return false
	}
	return strings.EqualFold(mentionUsername, botUsername)
}

// stripBotMention removes the @bot mention from the content.
func (c *TelegramChannel) stripBotMention(content string) string {
	botUsername := c.bot.Username()
	if botUsername == "" {
		return content
	}
	// Case-insensitive replacement
	re := regexp.MustCompile(`(?i)@` + regexp.QuoteMeta(botUsername))
	content = re.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

// BeginStream implements channels.StreamingCapable.
func (c *TelegramChannel) BeginStream(ctx context.Context, chatID string) (channels.Streamer, error) {
	if !c.tgCfg.Streaming.Enabled {
		return nil, fmt.Errorf("streaming disabled in config")
	}

	cid, threadID, err := parseTelegramChatID(chatID)
	if err != nil {
		return nil, err
	}

	streamCfg := c.tgCfg.Streaming.WithDefaults(3, 200)
	return &telegramStreamer{
		bot:              c.bot,
		chatID:           cid,
		threadID:         threadID,
		draftID:          cryptoRandInt(),
		throttleInterval: time.Duration(streamCfg.ThrottleSeconds) * time.Second,
		minGrowth:        streamCfg.MinGrowthChars,
	}, nil
}

// telegramStreamer streams partial LLM output via Telegram's sendMessageDraft API.
// Draft update failures are returned to the agent, which decides whether the
// stream was already visible enough to keep or should fall back to Chat().
type telegramStreamer struct {
	bot              *telego.Bot
	chatID           int64
	threadID         int
	draftID          int
	throttleInterval time.Duration
	minGrowth        int
	lastLen          int
	lastAt           time.Time
	failed           bool
	draftTouched     bool
	mu               sync.Mutex
}

func (s *telegramStreamer) Update(ctx context.Context, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failed {
		return fmt.Errorf("telegram streaming disabled after previous draft failure")
	}

	// Throttle: skip if not enough time or content has passed
	now := time.Now()
	growth := len(content) - s.lastLen
	if s.lastLen > 0 && now.Sub(s.lastAt) < s.throttleInterval && growth < s.minGrowth {
		return nil
	}

	htmlContent := markdownToTelegramHTML(content)
	s.draftTouched = true

	err := s.bot.SendMessageDraft(ctx, &telego.SendMessageDraftParams{
		ChatID:          s.chatID,
		MessageThreadID: s.threadID,
		DraftID:         s.draftID,
		Text:            htmlContent,
		ParseMode:       telego.ModeHTML,
	})
	if err != nil {
		logger.WarnCF("telegram", "sendMessageDraft failed, disabling streaming", map[string]any{
			"error": err.Error(),
		})
		s.failed = true
		return fmt.Errorf("telegram draft update: %w", err)
	}

	s.lastLen = len(content)
	s.lastAt = now
	return nil
}

func (s *telegramStreamer) Finalize(ctx context.Context, content string) error {
	htmlContent := markdownToTelegramHTML(content)
	tgMsg := tu.Message(tu.ID(s.chatID), htmlContent)
	tgMsg.MessageThreadID = s.threadID
	tgMsg.ParseMode = telego.ModeHTML

	if _, err := s.bot.SendMessage(ctx, tgMsg); err != nil {
		// Fallback to plain text
		tgMsg.ParseMode = ""
		if _, err = s.bot.SendMessage(ctx, tgMsg); err != nil {
			logger.ErrorCF("telegram", "Finalize failed after HTML and plain-text attempts", map[string]any{
				"chat_id": s.chatID,
				"error":   err.Error(),
				"len":     len(content),
			})
			return fmt.Errorf("telegram finalize: %w", err)
		}
	}
	s.Cancel(ctx)
	return nil
}

func (s *telegramStreamer) Cancel(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearDraft(ctx)
}

func (s *telegramStreamer) clearDraft(ctx context.Context) {
	if !s.draftTouched {
		return
	}
	if err := s.bot.SendMessageDraft(ctx, &telego.SendMessageDraftParams{
		ChatID:          s.chatID,
		MessageThreadID: s.threadID,
		DraftID:         s.draftID,
		Text:            " ",
	}); err != nil {
		logger.DebugCF("telegram", "failed to clear streaming draft", map[string]any{
			"chat_id": s.chatID,
			"error":   err.Error(),
		})
	}
	s.lastLen = 0
	s.draftTouched = false
}

// cryptoRandInt returns a non-zero random int using crypto/rand.
func cryptoRandInt() int {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return int(binary.BigEndian.Uint32(b[:])) | 1 // ensure non-zero
}

// VoiceCapabilities returns the voice capabilities of the channel.
func (c *TelegramChannel) VoiceCapabilities() channels.VoiceCapabilities {
	return channels.VoiceCapabilities{ASR: true, TTS: true}
}
