package api

import (
	"strings"
	"testing"
)

// TestChannelCatalogOffersNoWhatsAppChannel pins the product decision.
//
// PocketClaw ships no WhatsApp channel: the native transport carried more
// runtime and protocol surface than a phone-resident product could justify, and
// Telegram covers the remote agent channel. The catalog is what the console
// renders, so this is the line that decides whether a user can reach one.
//
// The vendored upstream transports still exist in pkg/channels and still start
// for an install that hand-edited a config block. That is deliberate: they are
// unreachable from the console, and deleting them would diverge from upstream
// for no runtime benefit.
func TestChannelCatalogOffersNoWhatsAppChannel(t *testing.T) {
	for _, item := range channelCatalog {
		if strings.Contains(strings.ToLower(item.Name), "whatsapp") {
			t.Errorf("catalog offers a WhatsApp channel: name=%q", item.Name)
		}
		if strings.Contains(strings.ToLower(item.ConfigKey), "whatsapp") {
			t.Errorf("catalog offers a WhatsApp config key: name=%q config_key=%q", item.Name, item.ConfigKey)
		}
	}

	for _, retired := range []string{
		"whatsapp", "whatsapp_native", "whatsapp_self_chat", "whatsapp_agent",
	} {
		if _, ok := findChannelCatalogItem(retired); ok {
			t.Errorf("%q is reachable through the channel catalog", retired)
		}
	}
}

// TestTelegramRemainsTheSupportedRemoteChannel is the other half: removing
// WhatsApp must not have disturbed the channel that replaced it.
func TestTelegramRemainsTheSupportedRemoteChannel(t *testing.T) {
	item, ok := findChannelCatalogItem("telegram")
	if !ok {
		t.Fatal("telegram is missing from the channel catalog")
	}
	if item.ConfigKey != "telegram" {
		t.Errorf("telegram ConfigKey = %q, want telegram", item.ConfigKey)
	}
	if secrets, present := channelSecretFieldMap["telegram"]; !present || len(secrets) == 0 {
		t.Error("telegram no longer declares its secret fields")
	}
}
