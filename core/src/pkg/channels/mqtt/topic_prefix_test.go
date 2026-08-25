package mqtt

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// A fresh configuration gets the PocketClaw default, while any explicitly
// configured prefix — including the legacy upstream one — survives untouched so
// that broker-side topics are never rewritten under a running deployment.
func TestTopicPrefixDefaultsAndPreservesConfiguredValues(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		want       string
	}{
		{
			name:       "fresh config uses the PocketClaw default",
			configured: "",
			want:       "/pocketclaw",
		},
		{
			name:       "explicit legacy prefix is preserved",
			configured: "/picoclaw",
			want:       "/picoclaw",
		},
		{
			name:       "custom prefix is preserved",
			configured: "/fleet/agents",
			want:       "/fleet/agents",
		},
		{
			name:       "prefix without a leading slash is preserved as configured",
			configured: "pocketclaw",
			want:       "pocketclaw",
		},
		{
			name:       "trailing slashes are stripped",
			configured: "/custom///",
			want:       "/custom",
		},
		{
			name:       "a prefix of only slashes falls back to the default",
			configured: "///",
			want:       "/pocketclaw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MQTTChannel{cfg: &config.MQTTSettings{TopicPrefix: tt.configured}}
			if got := c.topicPrefix(); got != tt.want {
				t.Errorf("topicPrefix() with %q = %q, want %q", tt.configured, got, tt.want)
			}
		})
	}
}

func TestDefaultTopicPrefixIsPocketClaw(t *testing.T) {
	if DefaultTopicPrefix != "/pocketclaw" {
		t.Errorf("DefaultTopicPrefix = %q, want %q", DefaultTopicPrefix, "/pocketclaw")
	}
}
