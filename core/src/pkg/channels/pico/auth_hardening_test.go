package pico

import (
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func channelWithToken(token string, allowQuery bool) *PicoChannel {
	settings := &config.PicoSettings{AllowTokenQuery: allowQuery}
	settings.SetToken(token)
	return &PicoChannel{config: settings}
}

// The health server has always compared the gateway bearer with
// subtle.ConstantTimeCompare; this path used == and was the odd one out.
func TestRealtimeAuthUsesConstantTimeComparison(t *testing.T) {
	const token = "realtime-credential-0123456789abcdef"

	t.Run("the correct credential is accepted", func(t *testing.T) {
		c := channelWithToken(token, false)
		r := httptest.NewRequest("GET", "/pico/ws", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		if !c.authenticate(r) {
			t.Fatal("a valid credential was rejected")
		}
	})

	for _, wrong := range []string{
		"",
		"realtime-credential-0123456789abcdee",  // last byte differs
		"Realtime-credential-0123456789abcdef",  // case differs
		"realtime-credential-0123456789abcde",   // shorter
		"realtime-credential-0123456789abcdef0", // longer
	} {
		t.Run("rejects "+wrong, func(t *testing.T) {
			c := channelWithToken(token, false)
			r := httptest.NewRequest("GET", "/pico/ws", nil)
			r.Header.Set("Authorization", "Bearer "+wrong)
			if c.authenticate(r) {
				t.Fatalf("an invalid credential was accepted: %q", wrong)
			}
		})
	}

	// An empty configured token must never authenticate anything, including
	// another empty string.
	t.Run("an unconfigured channel authenticates nobody", func(t *testing.T) {
		c := channelWithToken("", true)
		r := httptest.NewRequest("GET", "/pico/ws?token=", nil)
		if c.authenticate(r) {
			t.Fatal("an unconfigured channel accepted a request")
		}
	})
}

// A host-managed credential must never be presentable in a URL, where it would
// spread into request logs, referrers and history for no benefit.
func TestHostManagedCredentialRefusesQueryStringAuth(t *testing.T) {
	const token = "host-managed-credential-0123456789"

	t.Run("refused when the host supplied the credential", func(t *testing.T) {
		t.Setenv(config.EnvChannelsPicoToken, token)
		c := channelWithToken(token, true)

		r := httptest.NewRequest("GET", "/pico/ws?token="+token, nil)
		if c.authenticate(r) {
			t.Fatal("a host-managed credential was accepted from a query string")
		}

		// The header remains the supported way to present it, so the managed
		// channel keeps working.
		r = httptest.NewRequest("GET", "/pico/ws", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		if !c.authenticate(r) {
			t.Fatal("the header path must keep working for a managed credential")
		}
	})

	// Compatibility is not removed globally: a deployment whose client cannot
	// set a header still has the option, and it is off by default.
	t.Run("still available to a self-managed deployment", func(t *testing.T) {
		t.Setenv(config.EnvChannelsPicoToken, "")
		c := channelWithToken(token, true)
		r := httptest.NewRequest("GET", "/pico/ws?token="+token, nil)
		if !c.authenticate(r) {
			t.Fatal("query-string auth was removed for deployments that opt into it")
		}
	})

	t.Run("off by default", func(t *testing.T) {
		t.Setenv(config.EnvChannelsPicoToken, "")
		c := channelWithToken(token, false)
		r := httptest.NewRequest("GET", "/pico/ws?token="+token, nil)
		if c.authenticate(r) {
			t.Fatal("query-string auth worked without being enabled")
		}
	})
}
