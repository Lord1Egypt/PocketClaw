package channels

import (
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// The exact channel_list block the console's Pair button writes. It is
// reproduced verbatim rather than described, because the failure this guards
// against is a shape mismatch: a block the console happily saves and the
// gateway silently never starts looks, from the phone, exactly like a broken
// transport.
const pairedWhatsAppAgentBlock = `{
  "enabled": true,
  "type": "whatsapp_native",
  "allow_from": ["+201012345678"],
  "settings": { "use_native": true }
}`

func TestPairedWhatsAppAgentBlockMakesTheChannelReady(t *testing.T) {
	var bc config.Channel
	if err := json.Unmarshal([]byte(pairedWhatsAppAgentBlock), &bc); err != nil {
		t.Fatalf("the console's Pair payload does not decode: %v", err)
	}

	channels := config.ChannelsConfig{config.ChannelWhatsAppNative: &bc}
	if err := config.InitChannelList(channels); err != nil {
		t.Fatalf("InitChannelList: %v", err)
	}

	m := &Manager{config: &config.Config{Channels: channels}}
	got, ready := m.getChannelConfigAndEnabled(config.ChannelWhatsAppNative)
	if got == nil {
		t.Fatal("the channel block was not found under its config key")
	}
	if !ready {
		t.Error("the console's Pair payload does not make the channel ready; the gateway would never start it")
	}
}

// TestPairedWhatsAppAgentBlockCarriesTheAllowList catches the allow-list being
// routed into settings instead of the channel block, which would leave
// allow_from empty and — because this channel denies by default — drop every
// inbound message while looking correctly configured in the console.
func TestPairedWhatsAppAgentBlockCarriesTheAllowList(t *testing.T) {
	var bc config.Channel
	if err := json.Unmarshal([]byte(pairedWhatsAppAgentBlock), &bc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(bc.AllowFrom) != 1 || bc.AllowFrom[0] != "+201012345678" {
		t.Errorf("AllowFrom = %v, want the seeded number", bc.AllowFrom)
	}
}

// TestDisabledWhatsAppAgentBlockIsNotStarted is the Disconnect half.
func TestDisabledWhatsAppAgentBlockIsNotStarted(t *testing.T) {
	var bc config.Channel
	if err := json.Unmarshal([]byte(pairedWhatsAppAgentBlock), &bc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	bc.Enabled = false

	channels := config.ChannelsConfig{config.ChannelWhatsAppNative: &bc}
	if err := config.InitChannelList(channels); err != nil {
		t.Fatalf("InitChannelList: %v", err)
	}

	m := &Manager{config: &config.Config{Channels: channels}}
	if _, ready := m.getChannelConfigAndEnabled(config.ChannelWhatsAppNative); ready {
		t.Error("a disabled channel reported ready")
	}
}
