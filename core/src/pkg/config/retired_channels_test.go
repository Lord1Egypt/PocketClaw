package config

import (
	"encoding/json"
	"testing"
)

// TestRetiredChannelDoesNotBreakAnUpgrade is a regression test for a real
// device failure.
//
// Removing WhatsApp Self-Chat deleted its channel type, but an install that had
// configured it still had the block in config.json. An unknown type fails the
// whole config, so the gateway refused to start:
//
//	Skip auto-starting gateway: failed to load config:
//	channel "whatsapp_self_chat" has unknown type "whatsapp_self_chat"
//
// Removing a feature must never brick the upgrade for the users who adopted it.
func TestRetiredChannelDoesNotBreakAnUpgrade(t *testing.T) {
	raw := []byte(`{
	  "telegram": {"enabled": true, "type": "telegram", "settings": {"token": "t"}},
	  "whatsapp_self_chat": {"enabled": false, "type": "whatsapp_self_chat", "settings": {"self_number": "+201012345678"}}
	}`)

	var channels ChannelsConfig
	if err := json.Unmarshal(raw, &channels); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if err := InitChannelList(channels); err != nil {
		t.Fatalf("a retired channel block failed the whole config: %v", err)
	}

	if _, present := channels["whatsapp_self_chat"]; present {
		t.Error("the retired channel survived InitChannelList")
	}
	if _, present := channels["telegram"]; !present {
		t.Error("Telegram was dropped alongside the retired channel")
	}
}

// TestRetiredChannelIsDroppedByTypeNotByName catches a block a user renamed:
// the map key is theirs to choose, so the type is what identifies it.
func TestRetiredChannelIsDroppedByTypeNotByName(t *testing.T) {
	raw := []byte(`{"my_whatsapp": {"enabled": true, "type": "whatsapp_self_chat"}}`)

	var channels ChannelsConfig
	if err := json.Unmarshal(raw, &channels); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := InitChannelList(channels); err != nil {
		t.Fatalf("InitChannelList: %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("a renamed retired channel survived: %v", channels)
	}
}

// TestGenuinelyUnknownChannelStillFails keeps the retirement list from becoming
// a blanket "ignore anything we do not recognise", which would turn a typo in a
// channel type into a channel that silently never starts.
func TestGenuinelyUnknownChannelStillFails(t *testing.T) {
	raw := []byte(`{"telegran": {"enabled": true, "type": "telegran"}}`)

	var channels ChannelsConfig
	if err := json.Unmarshal(raw, &channels); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := InitChannelList(channels); err == nil {
		t.Error("a misspelled channel type was accepted")
	}
}
