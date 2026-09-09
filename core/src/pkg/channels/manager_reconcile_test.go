package channels

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// channelWithSecret names every channel that carries a credential in its
// settings, with the raw settings JSON that configures it. Reconcile behaviour
// is a property of the shared supervisor, so it is asserted once for all of
// them rather than per channel.
type channelWithSecret struct {
	name        string
	channelType string
	settings    func(secret string) string
}

func secretBearingChannels() []channelWithSecret {
	simpleToken := func(extra string) func(string) string {
		return func(secret string) string {
			return `{"enabled":true,"token":"` + secret + `"` + extra + `}`
		}
	}
	return []channelWithSecret{
		{"telegram", config.ChannelTelegram, simpleToken("")},
		{"discord", config.ChannelDiscord, simpleToken("")},
		{"vk", config.ChannelVK, simpleToken(`,"group_id":7`)},
		{"weixin", config.ChannelWeixin, simpleToken("")},
		{"pocketclaw", config.ChannelPocketClaw, simpleToken("")},
		{"slack", config.ChannelSlack, func(secret string) string {
			return `{"enabled":true,"bot_token":"` + secret + `","app_token":"xapp-1"}`
		}},
		{"matrix", config.ChannelMatrix, func(secret string) string {
			return `{"enabled":true,"homeserver":"https://h","user_id":"@a:h","access_token":"` + secret + `"}`
		}},
		{"line", config.ChannelLINE, func(secret string) string {
			return `{"enabled":true,"channel_secret":"s","channel_access_token":"` + secret + `"}`
		}},
		{"onebot", config.ChannelOneBot, func(secret string) string {
			return `{"enabled":true,"ws_url":"ws://x","access_token":"` + secret + `"}`
		}},
		{"dingtalk", config.ChannelDingTalk, func(secret string) string {
			return `{"enabled":true,"client_id":"c","client_secret":"` + secret + `"}`
		}},
		{"qq", config.ChannelQQ, func(secret string) string {
			return `{"enabled":true,"app_id":"a","app_secret":"` + secret + `"}`
		}},
		{"feishu", config.ChannelFeishu, func(secret string) string {
			return `{"enabled":true,"app_id":"a","app_secret":"` + secret + `"}`
		}},
		{"irc", config.ChannelIRC, func(secret string) string {
			return `{"enabled":true,"server":"irc://x","nick":"n","nickserv_password":"` + secret + `"}`
		}},
		{"mqtt", config.ChannelMQTT, func(secret string) string {
			return `{"enabled":true,"broker":"tcp://x","agent_id":"a","password":"` + secret + `"}`
		}},
		{"wecom", config.ChannelWeCom, func(secret string) string {
			return `{"enabled":true,"bot_id":"b","secret":"` + secret + `"}`
		}},
		{"slack_webhook", config.ChannelSlackWebHook, func(secret string) string {
			return `{"enabled":true,"webhooks":{"main":{"webhook_url":"` + secret + `"}}}`
		}},
		{"teams_webhook", config.ChannelTeamsWebHook, func(secret string) string {
			return `{"enabled":true,"webhooks":{"main":{"webhook_url":"` + secret + `"}}}`
		}},
	}
}

func configWithChannel(spec channelWithSecret, secret string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.Channels[spec.name] = &config.Channel{
		Enabled:  true,
		Type:     spec.channelType,
		Settings: config.RawNode(spec.settings(secret)),
	}
	return cfg
}

//  1. A credential change must be visible to the reconcile for every channel
//     that has one, including when the credential is not present in the raw
//     settings JSON. Stored credentials are resolved onto the decoded settings
//     struct after load, and config.SecureString marshals to a fixed
//     placeholder — so without a digest of the resolved value the reconcile
//     cannot see the change and the channel is never restarted.
func TestResolvedCredentialChangeTriggersReconcile(t *testing.T) {
	for _, spec := range secretBearingChannels() {
		t.Run(spec.name, func(t *testing.T) {
			hashFor := func(secret string) string {
				cfg := configWithChannel(spec, placeholderSecret)
				decoded, err := cfg.Channels[spec.name].GetDecoded()
				if err != nil || decoded == nil {
					t.Fatalf("decode %s settings: %v", spec.name, err)
				}
				if !setEverySecret(decoded, secret) {
					t.Skipf("%s exposes no settable credential field", spec.name)
				}
				hashes := toChannelHashes(cfg)
				if len(hashes) != 1 {
					t.Fatalf("channel %s produced no hash: %v", spec.name, hashes)
				}
				return hashes[spec.name]
			}

			if hashFor("secret-one") == hashFor("secret-two") {
				t.Fatalf("changing the resolved %s credential did not change the reconcile "+
					"hash; the channel would never be restarted", spec.name)
			}
		})
	}
}

// The reconcile hash must be a pure function of the configuration. It used to
// depend on whether the config object had been decoded yet: the startup hash is
// taken after initChannels has decoded every channel, while every reload hash
// is taken on a config freshly loaded from disk. Those two serializations never
// matched, so the first save after startup stopped and restarted every enabled
// channel — including the ones the user had not touched.
func TestReconcileHashDoesNotDependOnDecodeState(t *testing.T) {
	for _, spec := range secretBearingChannels() {
		t.Run(spec.name, func(t *testing.T) {
			undecoded := configWithChannel(spec, "same-secret")
			beforeDecode := toChannelHashes(undecoded)

			decodedCfg := configWithChannel(spec, "same-secret")
			if _, err := decodedCfg.Channels[spec.name].GetDecoded(); err != nil {
				t.Fatalf("decode %s settings: %v", spec.name, err)
			}
			afterDecode := toChannelHashes(decodedCfg)

			if beforeDecode[spec.name] != afterDecode[spec.name] {
				t.Fatalf("%s hashes differently once decoded; the first save after startup "+
					"would restart it for no reason", spec.name)
			}
			if again := toChannelHashes(undecoded); again[spec.name] != beforeDecode[spec.name] {
				t.Fatalf("%s hash is not stable across repeated calls", spec.name)
			}
		})
	}
}

//  2. An unchanged configuration must not restart anything. A reconcile that
//     fires on every save would drop live connections for a cosmetic edit.
func TestIdenticalConfigDoesNotReconcile(t *testing.T) {
	for _, spec := range secretBearingChannels() {
		t.Run(spec.name, func(t *testing.T) {
			before := toChannelHashes(configWithChannel(spec, "same-secret"))
			after := toChannelHashes(configWithChannel(spec, "same-secret"))
			added, removed := compareChannels(before, after)
			if len(added) != 0 || len(removed) != 0 {
				t.Fatalf("unchanged %s config reconciled anyway: added=%v removed=%v",
					spec.name, added, removed)
			}
		})
	}
}

// 3 + 4. Enabling starts a channel; disabling stops it.
func TestEnableStartsAndDisableStopsChannel(t *testing.T) {
	spec := channelWithSecret{"discord", config.ChannelDiscord, func(s string) string {
		return `{"enabled":true,"token":"` + s + `"}`
	}}

	disabled := config.DefaultConfig()
	enabled := configWithChannel(spec, "tok")

	added, removed := compareChannels(toChannelHashes(disabled), toChannelHashes(enabled))
	if len(added) != 1 || added[0] != "discord" || len(removed) != 0 {
		t.Fatalf("enabling discord: added=%v removed=%v", added, removed)
	}

	added, removed = compareChannels(toChannelHashes(enabled), toChannelHashes(disabled))
	if len(removed) != 1 || removed[0] != "discord" || len(added) != 0 {
		t.Fatalf("disabling discord: added=%v removed=%v", added, removed)
	}
}

//  6. A change to one channel must not disturb another. Channel isolation is
//     what keeps a broken Discord token from taking Telegram down with it.
func TestChangingOneChannelLeavesOthersUntouched(t *testing.T) {
	build := func(discordToken string) *config.Config {
		cfg := config.DefaultConfig()
		cfg.Channels["discord"] = &config.Channel{
			Enabled:  true,
			Type:     config.ChannelDiscord,
			Settings: config.RawNode(`{"enabled":true,"token":"` + discordToken + `"}`),
		}
		cfg.Channels["telegram"] = &config.Channel{
			Enabled:  true,
			Type:     config.ChannelTelegram,
			Settings: config.RawNode(`{"enabled":true,"token":"telegram-token"}`),
		}
		return cfg
	}

	before := toChannelHashes(build("old"))
	after := toChannelHashes(build("new"))
	if before["telegram"] != after["telegram"] {
		t.Fatal("a Discord token change altered Telegram's reconcile hash")
	}

	added, removed := compareChannels(before, after)
	for _, name := range append(append([]string(nil), added...), removed...) {
		if name != "discord" {
			t.Fatalf("reconcile touched an unrelated channel: %s", name)
		}
	}
}

//  7. The reconcile state must never carry a credential in the clear. The hash
//     input is derived from a digest, so neither the map nor the stored hash can
//     surrender the token.
func TestReconcileStateNeverHoldsPlaintextCredentials(t *testing.T) {
	const secret = "8123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw"
	for _, spec := range secretBearingChannels() {
		t.Run(spec.name, func(t *testing.T) {
			cfg := configWithChannel(spec, secret)
			value := map[string]any{}
			if err := json.Unmarshal([]byte(cfg.Channels[spec.name].Settings), &value); err != nil {
				t.Fatalf("unmarshal settings: %v", err)
			}
			hiddenValues(spec.name, value, cfg.Channels[spec.name])

			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("marshal hash input: %v", err)
			}
			if strings.Contains(string(encoded), secret) {
				t.Fatalf("%s reconcile input exposed the credential: %s", spec.name, encoded)
			}
			for _, hash := range toChannelHashes(cfg) {
				if strings.Contains(hash, secret) {
					t.Fatalf("%s reconcile hash exposed the credential", spec.name)
				}
			}
		})
	}
}

// 5 + 9. A channel whose configuration is enabled but which never became a
//
//	running instance must be reconciled away without panicking. The hash map
//	tracks configured channels, not live ones, and dereferencing that absence
//	would panic while the manager lock is held — freezing every later
//	reconcile rather than failing one channel.
func TestReconcileRemovesChannelThatNeverStarted(t *testing.T) {
	msgBus := bus.NewMessageBus()
	manager, err := NewManager(&config.Config{}, msgBus, nil)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	live := &reconcileTestChannel{name: "telegram"}
	manager.RegisterChannel("telegram", live)

	next := config.DefaultConfig()
	next.Channels["telegram"] = &config.Channel{
		Enabled:  true,
		Type:     config.ChannelTelegram,
		Settings: config.RawNode(`{"enabled":true,"token":"telegram-token"}`),
	}

	// Telegram is unchanged; "discord" is present in the tracked hashes but was
	// never registered as a running instance.
	tracked := toChannelHashes(next)
	tracked["discord"] = "discord-hash"
	manager.channelHashes = tracked

	if err := manager.Reload(context.Background(), next); err != nil {
		t.Fatalf("Reload with an unregistered removed channel: %v", err)
	}
	if live.stopCount() != 0 {
		t.Fatal("an unrelated live channel was stopped by another channel's removal")
	}
}

// 8 + 9. Concurrent saves must serialize into a single instance per channel.
func TestConcurrentReloadsLeaveOneInstancePerChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()
	manager, err := NewManager(&config.Config{}, msgBus, nil)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cfg := config.DefaultConfig()
			cfg.Channels["telegram"] = &config.Channel{
				Enabled: true,
				Type:    config.ChannelTelegram,
				Settings: config.RawNode(
					`{"enabled":true,"token":"token-` + string(rune('a'+i)) + `"}`),
			}
			_ = manager.Reload(context.Background(), cfg)
		}(i)
	}
	wg.Wait()

	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if len(manager.channels) > 1 {
		t.Fatalf("concurrent reloads produced duplicate channel instances: %d", len(manager.channels))
	}
}

type reconcileTestChannel struct {
	name string

	mu    sync.Mutex
	stops int
}

func (c *reconcileTestChannel) Name() string                        { return c.name }
func (c *reconcileTestChannel) Start(context.Context) error         { return nil }
func (c *reconcileTestChannel) IsRunning() bool                     { return true }
func (c *reconcileTestChannel) IsAllowed(string) bool               { return true }
func (c *reconcileTestChannel) IsAllowedSender(bus.SenderInfo) bool { return true }
func (c *reconcileTestChannel) ReasoningChannelID() string          { return "" }

func (c *reconcileTestChannel) Stop(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stops++
	return nil
}

func (c *reconcileTestChannel) Send(context.Context, bus.OutboundMessage) ([]string, error) {
	return []string{"sent"}, nil
}

func (c *reconcileTestChannel) stopCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stops
}

// placeholderSecret stands in for the marshalled form of a stored credential:
// config.SecureString never serializes its value.
const placeholderSecret = "[NOT_HERE]"

// setEverySecret assigns secret to every config.SecureString reachable from a
// decoded settings struct, mirroring how stored credentials are resolved onto
// the struct after load. Reports whether it found one.
func setEverySecret(decoded any, secret string) bool {
	return setSecretsInValue(reflect.ValueOf(decoded), secret)
}

func setSecretsInValue(value reflect.Value, secret string) bool {
	for value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		if value.Type() == reflect.TypeOf(config.SecureString{}) {
			if !value.CanSet() {
				return false
			}
			value.Set(reflect.ValueOf(*config.NewSecureString(secret)))
			return true
		}
		found := false
		for i := range value.NumField() {
			if !value.Type().Field(i).IsExported() {
				continue
			}
			if setSecretsInValue(value.Field(i), secret) {
				found = true
			}
		}
		return found
	case reflect.Slice:
		if value.Type() == reflect.TypeOf(config.SecureStrings{}) {
			if !value.CanSet() {
				return false
			}
			value.Set(reflect.ValueOf(config.SecureStrings{config.NewSecureString(secret)}))
			return true
		}
		return false
	case reflect.Map:
		found := false
		for _, key := range value.MapKeys() {
			entry := reflect.New(value.Type().Elem()).Elem()
			entry.Set(value.MapIndex(key))
			if setSecretsInValue(entry, secret) {
				value.SetMapIndex(key, entry)
				found = true
			}
		}
		return found
	default:
		return false
	}
}
