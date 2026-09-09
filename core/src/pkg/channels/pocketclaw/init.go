package pocketclaw

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory(
		config.ChannelPocketClaw,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.PocketClawSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			ch, err := NewPocketClawChannel(bc, c, b)
			if err != nil {
				return nil, err
			}
			if channelName != config.ChannelPocketClaw {
				ch.SetName(channelName)
			}
			return ch, nil
		},
	)
	channels.RegisterFactory(
		config.ChannelPocketClawClient,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.PocketClawClientSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			ch, err := NewPocketClawClientChannel(bc, c, b)
			if err != nil {
				return nil, err
			}
			if channelName != config.ChannelPocketClawClient {
				ch.SetName(channelName)
			}
			return ch, nil
		},
	)
}
