package api

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// PC-DEF-028. One owner contract, enforced in Core, with historical corruption
// surfaced rather than silently repaired.

func telegramChannel(enabled bool, owners ...string) *config.Channel {
	bc := &config.Channel{Type: config.ChannelTelegram, Enabled: enabled}
	bc.SetName(config.ChannelTelegram)
	bc.AllowFrom = config.FlexibleStringSlice(owners)
	return bc
}

func TestTelegramOwnerContract(t *testing.T) {
	cases := []struct {
		name    string
		channel *config.Channel
		wantErr bool
	}{
		{"exactly one positive numeric owner", telegramChannel(true, "123456789"), false},
		{"whitespace around a valid owner", telegramChannel(true, "  123456789  "), false},
		{"zero owners", telegramChannel(true), true},
		{"only blank owners", telegramChannel(true, "", "   "), true},
		{"multiple owners", telegramChannel(true, "123456789", "987654321"), true},
		{"non-numeric owner", telegramChannel(true, "@someone"), true},
		{"zero owner id", telegramChannel(true, "0"), true},
		{"negative owner id", telegramChannel(true, "-42"), true},
		{"disabled channel is not held to the rule", telegramChannel(false), false},
		{"disabled with several owners is still fine", telegramChannel(false, "1", "2"), false},
		{"absent channel", nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := telegramOwnerErrors(tc.channel)
			if tc.wantErr && len(errs) == 0 {
				t.Fatal("expected the configuration to be rejected")
			}
			if !tc.wantErr && len(errs) > 0 {
				t.Fatalf("unexpected rejection: %v", errs)
			}
		})
	}
}

// Multiple owners must be reported, never resolved. Picking one would hand the
// agent to whichever entry happened to sort first.
func TestMultipleOwnersAreReportedNotResolved(t *testing.T) {
	bc := telegramChannel(true, "111", "222")
	errs := telegramOwnerErrors(bc)

	if len(errs) == 0 {
		t.Fatal("multiple owners were accepted")
	}
	if len(bc.AllowFrom) != 2 {
		t.Fatalf("validation mutated the owner list to %v; it must not repair silently", bc.AllowFrom)
	}
}

func telegramConfig(t *testing.T, enabled bool, owners ...string) *config.Config {
	t.Helper()
	cfg := config.DefaultConfig()
	if cfg.Channels == nil {
		cfg.Channels = make(config.ChannelsConfig)
	}
	cfg.Channels[config.ChannelTelegram] = telegramChannel(enabled, owners...)
	return cfg
}

// telegramConfigWith builds the same shape with a bot token, which the plain
// helper leaves unset.
func telegramConfigWith(
	t *testing.T, enabled bool, owners []string, token string,
) *config.Config {
	t.Helper()
	cfg := telegramConfig(t, enabled, owners...)
	channel := cfg.Channels.GetByType(config.ChannelTelegram)
	settings := &config.TelegramSettings{}
	settings.Token.Set(token)
	if err := channel.Decode(settings); err != nil {
		t.Fatalf("Decode(TelegramSettings) error = %v", err)
	}
	return cfg
}

// Serialization noise is not an edit. Item 31.
func TestReorderedOwnersAreNotATelegramEdit(t *testing.T) {
	before := telegramConfig(t, true, "111", "222")
	after := telegramConfig(t, true, "222", "111")

	if telegramSubtreeChanged(before, after) {
		t.Fatal("reordering owners was treated as a Telegram edit; comparison must be semantic")
	}
}

func TestIdenticalTelegramConfigIsNotAnEdit(t *testing.T) {
	before := telegramConfig(t, true, "123456789")
	after := telegramConfig(t, true, "  123456789  ")

	if telegramSubtreeChanged(before, after) {
		t.Fatal("whitespace-only difference was treated as a Telegram edit")
	}
}

func TestRealTelegramChangesAreDetected(t *testing.T) {
	base := telegramConfig(t, true, "123456789")

	for name, changed := range map[string]*config.Config{
		"owner changed":   telegramConfig(t, true, "987654321"),
		"owner added":     telegramConfig(t, true, "123456789", "987654321"),
		"owner removed":   telegramConfig(t, true),
		"channel enabled": telegramConfig(t, false, "123456789"),
	} {
		t.Run(name, func(t *testing.T) {
			if !telegramSubtreeChanged(base, changed) {
				t.Fatal("a real Telegram change was not detected")
			}
		})
	}
}

// Item 29: a legacy-invalid Telegram config must not block an unrelated save.
func TestLegacyInvalidTelegramIsUnchangedByAnUnrelatedSave(t *testing.T) {
	legacy := telegramConfig(t, true, "111", "222") // invalid, and already persisted
	unrelated := telegramConfig(t, true, "111", "222")
	unrelated.Agents.Defaults.ModelName = "some-other-model"

	if telegramSubtreeChanged(legacy, unrelated) {
		t.Fatal("an unrelated settings change was seen as a Telegram edit, which " +
			"would make historical corruption block the whole settings UI")
	}
	// And the corruption is still corruption -- it is surfaced, not repaired.
	if errs := telegramOwnerErrors(unrelated.Channels.GetByType(config.ChannelTelegram)); len(errs) == 0 {
		t.Fatal("the legacy configuration stopped being reported as invalid")
	}
}

// Replacing the bot token is an edit to Telegram, and an edit to Telegram is
// held to the owner contract.
//
// The semantic key used to record only whether a token was present, so a
// Replace Bot save -- new token, same everything else -- produced an identical
// key and skipped the owner check. That is how a channel ends up enabled with a
// working token and a stale or malformed owner, which Core then refuses at
// startup with "telegram requires exactly one paired numeric owner": the
// historical failure, reachable from a path the validation could not see.
func TestTelegramSubtreeChangedDetectsAReplacedToken(t *testing.T) {
	before := telegramConfigWith(t, true, []string{"12345"}, "111:AAHoldTokenValue0123456789")
	after := telegramConfigWith(t, true, []string{"12345"}, "222:AAHnewTokenValue0123456789")

	if !telegramSubtreeChanged(before, after) {
		t.Fatal("a replaced bot token did not read as a Telegram edit, so the " +
			"owner contract would not be enforced on that save")
	}
}

func TestTelegramSubtreeUnchangedForAnIdenticalToken(t *testing.T) {
	before := telegramConfigWith(t, true, []string{"12345"}, "111:AAHoldTokenValue0123456789")
	after := telegramConfigWith(t, true, []string{"12345"}, "111:AAHoldTokenValue0123456789")

	if telegramSubtreeChanged(before, after) {
		t.Fatal("an unrelated save that carries the same Telegram subtree through " +
			"must not be held to the contract")
	}
}

// Adding a token where there was none, and removing one, are both edits.
func TestTelegramSubtreeChangedDetectsTokenPresence(t *testing.T) {
	none := telegramConfigWith(t, true, []string{"12345"}, "")
	some := telegramConfigWith(t, true, []string{"12345"}, "111:AAHtokenValue0123456789")

	if !telegramSubtreeChanged(none, some) {
		t.Fatal("adding a token is an edit")
	}
	if !telegramSubtreeChanged(some, none) {
		t.Fatal("removing a token is an edit")
	}
}

// The digest exists to compare, not to carry. Whatever else it does, it must
// not be the token.
func TestTelegramTokenDigestDoesNotCarryTheToken(t *testing.T) {
	const token = "123456789:AAHfSomeTelegramBotTokenValue0123456789"

	digest := telegramTokenDigest(token)
	if strings.Contains(digest, token) || strings.Contains(digest, "AAHfSome") {
		t.Fatalf("the digest carries the token: %q", digest)
	}
	if digest == telegramTokenDigest("") {
		t.Fatal("a set token is indistinguishable from no token")
	}
	if digest != telegramTokenDigest(" "+token+" ") {
		t.Fatal("surrounding whitespace changed the identity of the same token")
	}
}
