package selfchat

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ConfigKey is the channel_list entry the Self-Chat surface persists into.
//
// It is deliberately not the legacy "whatsapp" block: that one still drives
// the bridge and native transports for existing installs, and reusing it would
// turn configuring Self-Chat into enabling a transport the user never asked
// for.
const ConfigKey = "whatsapp_self_chat"

// ConfiguredNumber returns the canonical self number from the config file on
// disk, or "" when Self-Chat is not configured.
//
// It reads the file rather than a snapshot so the agent tool observes a number
// the user just changed or disconnected in the console, which otherwise would
// only take effect after a gateway restart. Any read or decode failure means
// "not configured": the caller then reports that plainly instead of acting on
// a number it could not confirm.
func ConfiguredNumber() string {
	cfg, err := config.LoadConfig(config.ResolveConfigPath())
	if err != nil || cfg == nil {
		return ""
	}
	return NumberFromConfig(cfg)
}

// NumberFromConfig reads the canonical self number out of an already loaded
// config.
func NumberFromConfig(cfg *config.Config) string {
	channel := cfg.Channels.Get(ConfigKey)
	if channel == nil {
		return ""
	}
	decoded, err := channel.GetDecoded()
	if err != nil {
		return ""
	}
	settings, ok := decoded.(*config.WhatsAppSelfChatSettings)
	if !ok {
		return ""
	}
	// A number that was hand-edited into the config file has never been through
	// the console's validation, so it is normalized here too rather than being
	// handed to Android as typed.
	canonical, err := Normalize(strings.TrimSpace(settings.SelfNumber))
	if err != nil {
		return ""
	}
	return canonical
}
