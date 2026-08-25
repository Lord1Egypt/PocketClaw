package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// customChannelSettings stands in for an out-of-tree channel's settings struct.
type customChannelSettings struct {
	Token string `json:"token"`
}

// TestRegisterChannelSettings verifies the public registration hook makes a
// previously-unknown channel type valid and decodable — the behavior out-of-tree
// channels rely on (they call RegisterChannelSettings from init()).
func TestRegisterChannelSettings(t *testing.T) {
	const typ = "custom_test_channel"

	assert.False(t, isValidChannelType(typ), "type should be unknown before registration")

	RegisterChannelSettings(typ, customChannelSettings{})

	assert.True(t, isValidChannelType(typ), "type should be valid after registration")

	got := newChannelSettings(typ)
	_, ok := got.(*customChannelSettings)
	assert.Truef(t, ok, "newChannelSettings(%q) = %T, want *customChannelSettings", typ, got)
}

// TestRegisterChannelSettings_InitChannelList verifies that a config carrying a
// registered out-of-tree channel type passes InitChannelList and decodes its
// settings — the full path that previously errored with "unknown type".
func TestRegisterChannelSettings_InitChannelList(t *testing.T) {
	const typ = "custom_initlist_channel"
	RegisterChannelSettings(typ, customChannelSettings{})

	channels := ChannelsConfig{
		"mychan": {
			Type:     typ,
			Enabled:  true,
			Settings: RawNode(`{"token":"secret-123"}`),
		},
	}

	require.NoError(t, InitChannelList(channels))

	decoded, err := channels["mychan"].GetDecoded()
	require.NoError(t, err)
	cfg, ok := decoded.(*customChannelSettings)
	require.True(t, ok)
	assert.Equal(t, "secret-123", cfg.Token)
}
