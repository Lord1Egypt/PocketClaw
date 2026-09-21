package agent

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Reporting the one transition the launcher needs: this gateway just finished
// the last answer it had in flight.
//
// PC-DEF-030. A configuration change that arrived while the gateway was busy
// is persisted but not live, and the launcher applies it when this fires. It
// is a notification, not a request for anything: the launcher decides whether
// there is work, and this side neither knows nor cares.

const gatewayIdleNotifyTimeout = 2 * time.Second

var gatewayIdleNotifier = struct {
	once   sync.Once
	url    string
	token  string
	client *http.Client
}{}

func gatewayIdleNotifyConfig() (url, token string, client *http.Client) {
	gatewayIdleNotifier.once.Do(func() {
		gatewayIdleNotifier.url = strings.TrimSpace(os.Getenv(config.EnvGatewayIdleURL))
		gatewayIdleNotifier.token = strings.TrimSpace(os.Getenv(config.EnvGatewayIdleToken))
		gatewayIdleNotifier.client = &http.Client{Timeout: gatewayIdleNotifyTimeout}
	})
	return gatewayIdleNotifier.url, gatewayIdleNotifier.token, gatewayIdleNotifier.client
}

// notifyGatewayIdle posts the notification, best effort.
//
// Bounded by a short timeout and never retried: a missed notification costs a
// configuration apply that the next explicit change or gateway start will make
// anyway, while a retry loop here would turn an unreachable launcher into a
// permanent background spin inside the gateway. Nothing it can do is allowed
// to fail an answer that has already been delivered, so every error is
// discarded deliberately.
func notifyGatewayIdle() {
	url, token, client := gatewayIdleNotifyConfig()
	if url == "" || token == "" {
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		return
	}
	req.Header.Set("X-PocketClaw-Gateway-Idle", token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}
