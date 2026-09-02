package whatsapp

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/whatsapp/selfchat"
	"github.com/sipeed/picoclaw/pkg/whatsapp/session"
)

func init() {
	channels.RegisterFactory(
		config.ChannelWhatsAppNative,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.WhatsAppSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			// Where the session database lives is not the config's decision on
			// Android. The whatsmeow store holds the linked device's identity
			// keys, and the workspace it would otherwise default into is both
			// public external storage and the root the agent's file tools are
			// restricted to. When the host names a directory, it wins.
			store := session.Resolve(c.SessionStorePath, cfg.WorkspacePath())
			if store.HostOwned && store.IgnoredConfigured != "" {
				logger.InfoCF("whatsapp", "Ignoring configured session_store_path; the Android host owns this path", map[string]any{
					"channel": channelName,
				})
			}
			// The Self-Chat number is this channel's whole world: it is the
			// only conversation it reads, the only one it writes to, and the
			// number the companion pairing code is requested for. Without one
			// the channel starts but denies in both directions, which is the
			// safe reading of "not configured yet".
			selfNumber := selfchat.NumberFromConfig(cfg)
			if selfNumber == "" {
				logger.WarnCF("whatsapp", "WhatsApp Agent Channel has no Self-Chat number; it will accept and send nothing", map[string]any{
					"channel": channelName,
				})
			}
			ch, err := NewWhatsAppNativeChannel(bc, channelName, c, b, store.Path, selfNumber)
			if err != nil {
				return nil, err
			}
			return ch, nil
		},
	)
}
