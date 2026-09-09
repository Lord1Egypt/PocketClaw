// Package androiddns configures Go's resolver from Android's active network.
//
// Android applications do not expose a conventional /etc/resolv.conf to
// statically linked Go binaries. The Android host passes the DNS servers from
// ConnectivityManager.LinkProperties in PICOCLAW_DNS_SERVER before launching
// either PicoClaw binary.
package androiddns

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/canonicalenv"
)

// EnvServer carries the DNS servers the Android host read from
// ConnectivityManager. It is exported because the Managed Runtime passes the
// same value on to the one bundled tool that needs it, and two spellings of one
// variable name is exactly the kind of drift that makes DNS fail silently.
const EnvServer = "PICOCLAW_DNS_SERVER"

var configureOnce sync.Once

// ConfigureDefaultResolverFromEnvironment installs an Android-provided DNS
// resolver when PICOCLAW_DNS_SERVER contains one or more semicolon-separated
// addresses. It deliberately leaves the platform resolver unchanged when the
// host did not supply DNS information.
func ConfigureDefaultResolverFromEnvironment() {
	configureOnce.Do(func() {
		servers := parseServers(canonicalenv.Getenv(EnvServer))
		if len(servers) == 0 {
			return
		}

		var next uint64
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
				server := servers[atomic.AddUint64(&next, 1)%uint64(len(servers))]
				dialer := net.Dialer{Timeout: 5 * time.Second}
				return dialer.DialContext(ctx, "udp", server)
			},
		}

		net.DefaultResolver = resolver
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			Resolver:  resolver,
		}
		if transport, ok := http.DefaultTransport.(*http.Transport); ok {
			transport.DialContext = dialer.DialContext
		}
	})
}

func parseServers(raw string) []string {
	servers := make([]string, 0)
	seen := make(map[string]struct{})
	for _, rawServer := range strings.Split(raw, ";") {
		server, ok := normalizeServer(rawServer)
		if !ok {
			continue
		}
		if _, exists := seen[server]; exists {
			continue
		}
		seen[server] = struct{}{}
		servers = append(servers, server)
	}
	return servers
}

func normalizeServer(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if host, port, err := net.SplitHostPort(raw); err == nil {
		if strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
			return "", false
		}
		return net.JoinHostPort(host, port), true
	}

	host := strings.Trim(raw, "[]")
	if host == "" || strings.ContainsAny(host, " \t\r\n") {
		return "", false
	}
	return net.JoinHostPort(host, "53"), true
}
