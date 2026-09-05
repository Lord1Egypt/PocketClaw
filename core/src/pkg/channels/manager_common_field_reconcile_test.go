package channels

import (
	"context"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// telegramChannelConfig builds a Telegram entry whose common fields can be
// varied one at a time.
func telegramChannelConfig(mutate func(*config.Channel)) *config.Config {
	cfg := config.DefaultConfig()
	ch := &config.Channel{
		Enabled:  true,
		Type:     config.ChannelTelegram,
		Settings: config.RawNode(`{"enabled":true,"token":"telegram-token"}`),
	}
	if mutate != nil {
		mutate(ch)
	}
	cfg.Channels["telegram"] = ch
	return cfg
}

// commonFieldMutations covers every field config.Channel keeps beside Settings.
// A hash built only from the settings payload cannot see any of them, which is
// what made the Typing Indicator toggle apply only after a manual restart.
func commonFieldMutations() map[string]func(*config.Channel) {
	return map[string]func(*config.Channel){
		"typing":               func(c *config.Channel) { c.Typing = config.TypingConfig{Enabled: true} },
		"placeholder":          func(c *config.Channel) { c.Placeholder = config.PlaceholderConfig{Enabled: true} },
		"allow_from":           func(c *config.Channel) { c.AllowFrom = config.FlexibleStringSlice{"12345"} },
		"reasoning_channel_id": func(c *config.Channel) { c.ReasoningChannelID = "reasoning-chat" },
		"group_trigger": func(c *config.Channel) {
			c.GroupTrigger = config.GroupTriggerConfig{MentionOnly: true}
		},
	}
}

// A. Changing any common field must reconcile that channel — and only it.
func TestCommonChannelFieldChangeReconcilesOnlyThatChannel(t *testing.T) {
	withNeighbour := func(mutate func(*config.Channel)) *config.Config {
		cfg := telegramChannelConfig(mutate)
		cfg.Channels["discord"] = &config.Channel{
			Enabled:  true,
			Type:     config.ChannelDiscord,
			Settings: config.RawNode(`{"enabled":true,"token":"discord-token"}`),
		}
		return cfg
	}

	base := toChannelHashes(withNeighbour(nil))

	for name, mutate := range commonFieldMutations() {
		t.Run(name, func(t *testing.T) {
			changed := toChannelHashes(withNeighbour(mutate))

			if base["telegram"] == changed["telegram"] {
				t.Fatalf("changing %s left the reconcile hash unchanged; the channel "+
					"would never be restarted and the setting would need a manual "+
					"Gateway restart", name)
			}
			if base["discord"] != changed["discord"] {
				t.Fatalf("changing telegram's %s disturbed discord's hash; a save on one "+
					"channel must not restart another", name)
			}

			added, removed := compareChannels(base, changed)
			if len(added) != 1 || added[0] != "telegram" {
				t.Fatalf("expected only telegram to be restarted, added=%v", added)
			}
			if len(removed) != 1 || removed[0] != "telegram" {
				t.Fatalf("expected only telegram to be stopped, removed=%v", removed)
			}
		})
	}
}

// E. The same configuration must never reconcile, whichever shape it is in.
func TestCommonFieldsHashEquivalentlyRawAndDecoded(t *testing.T) {
	for name, mutate := range commonFieldMutations() {
		t.Run(name, func(t *testing.T) {
			undecoded := telegramChannelConfig(mutate)
			before := toChannelHashes(undecoded)

			decoded := telegramChannelConfig(mutate)
			if _, err := decoded.Channels["telegram"].GetDecoded(); err != nil {
				t.Fatalf("decode telegram settings: %v", err)
			}
			after := toChannelHashes(decoded)

			if before["telegram"] != after["telegram"] {
				t.Fatalf("with %s set, the hash depends on decode state; every save "+
					"would restart the channel", name)
			}
			if again := toChannelHashes(undecoded); again["telegram"] != before["telegram"] {
				t.Fatalf("with %s set, the hash is not stable across repeated calls", name)
			}

			added, removed := compareChannels(before, after)
			if len(added) != 0 || len(removed) != 0 {
				t.Fatalf("identical config reconciled anyway: added=%v removed=%v", added, removed)
			}
		})
	}
}

// The digest rule still holds once the common fields are in the input.
func TestCommonFieldHashStillCarriesNoPlaintextCredential(t *testing.T) {
	const secret = "8123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw"
	cfg := telegramChannelConfig(func(c *config.Channel) {
		c.Typing = config.TypingConfig{Enabled: true}
		c.Settings = config.RawNode(`{"enabled":true,"token":"` + secret + `"}`)
	})
	for _, hash := range toChannelHashes(cfg) {
		if hash == "" {
			t.Fatal("no hash produced")
		}
		if len(hash) != 32 {
			t.Fatalf("hash is not an md5 digest: %q", hash)
		}
	}
}

// typingSpyChannel records whether the channel's own StartTyping was reached.
type typingSpyChannel struct {
	reconcileTestChannel

	mu     sync.Mutex
	starts int
}

func (c *typingSpyChannel) StartTyping(context.Context, string) (func(), error) {
	c.mu.Lock()
	c.starts++
	c.mu.Unlock()
	return func() {}, nil
}

func (c *typingSpyChannel) startCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.starts
}

// B. The typing setting is honoured at the one entry point that every
//
//	deferred-activity channel goes through.
func TestStartTypingHonoursTheTypingSetting(t *testing.T) {
	for _, test := range []struct {
		name       string
		enabled    bool
		wantStarts int
		wantResult bool
	}{
		{name: "disabled", enabled: false, wantStarts: 0, wantResult: false},
		{name: "enabled", enabled: true, wantStarts: 1, wantResult: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := telegramChannelConfig(func(c *config.Channel) {
				c.Typing = config.TypingConfig{Enabled: test.enabled}
			})
			manager, err := NewManager(cfg, bus.NewMessageBus(), nil)
			if err != nil {
				t.Fatalf("NewManager: %v", err)
			}
			spy := &typingSpyChannel{reconcileTestChannel: reconcileTestChannel{name: "telegram"}}
			manager.RegisterChannel("telegram", spy)

			got := manager.StartTyping(context.Background(), "telegram", "chat")
			if got != test.wantResult {
				t.Fatalf("StartTyping() = %v, want %v", got, test.wantResult)
			}
			if spy.startCount() != test.wantStarts {
				t.Fatalf("channel StartTyping called %d times, want %d",
					spy.startCount(), test.wantStarts)
			}
		})
	}
}

// A channel with no configuration at all keeps the indicator: absence of
// configuration is not a user asking for it to be off.
func TestStartTypingDefaultsToEnabledForAnUnconfiguredChannel(t *testing.T) {
	manager, err := NewManager(config.DefaultConfig(), bus.NewMessageBus(), nil)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	spy := &typingSpyChannel{reconcileTestChannel: reconcileTestChannel{name: "unlisted"}}
	manager.RegisterChannel("unlisted", spy)

	if !manager.StartTyping(context.Background(), "unlisted", "chat") {
		t.Fatal("an unconfigured channel must keep its typing indicator")
	}
	if spy.startCount() != 1 {
		t.Fatalf("channel StartTyping called %d times, want 1", spy.startCount())
	}
}

// C. End to end: a reload carrying only a typing change must stop and recreate
//
//	that channel, and leave every other channel alone.
func TestReloadRecreatesTheChannelWhoseTypingChanged(t *testing.T) {
	start := telegramChannelConfig(func(c *config.Channel) {
		c.Typing = config.TypingConfig{Enabled: true}
	})
	start.Channels["discord"] = &config.Channel{
		Enabled:  true,
		Type:     config.ChannelDiscord,
		Settings: config.RawNode(`{"enabled":true,"token":"discord-token"}`),
	}

	manager, err := NewManager(start, bus.NewMessageBus(), nil)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	telegram := &typingSpyChannel{reconcileTestChannel: reconcileTestChannel{name: "telegram"}}
	discord := &typingSpyChannel{reconcileTestChannel: reconcileTestChannel{name: "discord"}}
	manager.RegisterChannel("telegram", telegram)
	manager.RegisterChannel("discord", discord)
	manager.channelHashes = toChannelHashes(start)

	next := telegramChannelConfig(func(c *config.Channel) {
		c.Typing = config.TypingConfig{Enabled: false}
	})
	next.Channels["discord"] = start.Channels["discord"]

	if err := manager.Reload(context.Background(), next); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if telegram.stopCount() == 0 {
		t.Fatal("the telegram instance was not stopped, so it still carries the old " +
			"typing setting and the change needs a manual Gateway restart")
	}
	if discord.stopCount() != 0 {
		t.Fatal("discord was restarted by a telegram-only change; channel isolation broken")
	}

	// The manager must now answer with the new setting.
	if manager.StartTyping(context.Background(), "telegram", "chat") {
		t.Fatal("typing still starts after the reload disabled it")
	}
}

// E. A reload carrying no change must not restart anything.
func TestReloadWithIdenticalConfigDoesNotRestartTelegram(t *testing.T) {
	cfg := telegramChannelConfig(func(c *config.Channel) {
		c.Typing = config.TypingConfig{Enabled: true}
	})

	manager, err := NewManager(cfg, bus.NewMessageBus(), nil)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	telegram := &typingSpyChannel{reconcileTestChannel: reconcileTestChannel{name: "telegram"}}
	manager.RegisterChannel("telegram", telegram)
	manager.channelHashes = toChannelHashes(cfg)

	same := telegramChannelConfig(func(c *config.Channel) {
		c.Typing = config.TypingConfig{Enabled: true}
	})
	if err := manager.Reload(context.Background(), same); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if telegram.stopCount() != 0 {
		t.Fatalf("an unchanged config restarted telegram %d times", telegram.stopCount())
	}
}
