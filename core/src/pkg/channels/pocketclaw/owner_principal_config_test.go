package pocketclaw

import (
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// The console no longer renders the Web channel's allow_from field: the value
// is never consulted for authorization, and both the console backend and the
// Android host rewrite it on every provisioning pass, so offering it as an
// editable control showed an internal principal the user could not meaningfully
// change. These pin what that hiding must not disturb.

func newOwnerOnlyChannel(t *testing.T, allowFrom config.FlexibleStringSlice) *PocketClawChannel {
	t.Helper()
	bc := &config.Channel{Type: config.ChannelPocketClaw, Enabled: true, AllowFrom: allowFrom}
	cfg := &config.PocketClawSettings{}
	cfg.SetToken("test-token")
	ch, err := NewPocketClawChannel(bc, cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("NewPocketClawChannel: %v", err)
	}
	return ch
}

func assertOwnerOnly(t *testing.T, ch *PocketClawChannel, context string) {
	t.Helper()
	if !ch.IsAllowedSender(bus.SenderInfo{Platform: "pocketclaw", PlatformID: OwnerPrincipal}) {
		t.Errorf("%s: the server-derived owner principal was rejected", context)
	}
	// The legacy principal is included deliberately. Stored configuration is
	// canonicalized by the migration; a sender presenting the old label is not
	// an upgrade path, it is someone else, and owner-only enforcement compares
	// against exactly one value.
	for _, impostor := range []string{
		"attacker", "", "*", "PICO-USER", "pico-user2",
		config.LegacyPocketClawOwnerPrincipal,
		"POCKETCLAW-USER", "pocketclaw-user2", "pocketclaw_user",
	} {
		if ch.IsAllowedSender(bus.SenderInfo{Platform: "pocketclaw", PlatformID: impostor}) {
			t.Errorf("%s: %q was authorized", context, impostor)
		}
	}
}

// TestOwnerOnlyRegardlessOfOnDiskAllowFrom is the property that makes hiding
// the field safe: the channel's authorization does not come from config at all.
func TestOwnerOnlyRegardlessOfOnDiskAllowFrom(t *testing.T) {
	cases := []struct {
		name      string
		allowFrom config.FlexibleStringSlice
	}{
		{"provisioned value", config.FlexibleStringSlice{config.PocketClawOwnerPrincipal}},
		{"absent", nil},
		{"empty", config.FlexibleStringSlice{}},
		{"permissive wildcard", config.FlexibleStringSlice{"*"}},
		{"someone else", config.FlexibleStringSlice{"attacker"}},
		{"owner plus someone else", config.FlexibleStringSlice{config.PocketClawOwnerPrincipal, "attacker"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertOwnerOnly(t, newOwnerOnlyChannel(t, tc.allowFrom), tc.name)
		})
	}
}

// TestProvisionedAllowFromStillParses covers both shapes an existing install can
// have on disk. The console collapses list fields to a newline-joined string on
// submit, so a config saved from the Web channel page carries a bare string
// rather than an array — and it has to keep loading.
func TestProvisionedAllowFromStillParses(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"array as written by the provisioners", `{"enabled":true,"type":"pocketclaw","allow_from":["pocketclaw-user"]}`},
		{"bare string as written by a console save", `{"enabled":true,"type":"pocketclaw","allow_from":"pocketclaw-user"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bc config.Channel
			if err := json.Unmarshal([]byte(tc.raw), &bc); err != nil {
				t.Fatalf("an existing Web channel config failed to parse: %v", err)
			}
			if len(bc.AllowFrom) != 1 || bc.AllowFrom[0] != config.PocketClawOwnerPrincipal {
				t.Fatalf("AllowFrom = %v, want exactly the owner principal", bc.AllowFrom)
			}
		})
	}
}

// TestPicoChannelStillRequiresItsToken keeps the hiding from being mistaken for
// "this channel validates nothing": a genuinely invalid configuration must
// still fail rather than start an unauthenticated realtime transport.
func TestPicoChannelStillRequiresItsToken(t *testing.T) {
	bc := &config.Channel{Type: config.ChannelPocketClaw, Enabled: true}
	if _, err := NewPocketClawChannel(bc, &config.PocketClawSettings{}, bus.NewMessageBus()); err == nil {
		t.Error("a Web channel with no token was accepted")
	}
}
