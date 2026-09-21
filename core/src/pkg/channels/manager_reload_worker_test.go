package channels

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// PC-DEF-061. A channel whose configuration changed is reported by
// compareChannels as both removed and added, and Reload rebuilt it under the
// same name. The outgoing instance's deferred UnregisterChannel then ran after
// the replacement was registered and closed the replacement's worker, leaving a
// channel that was configured, started and polling but could not send a reply.
// That is a generation that can acknowledge an update and never answer it.
func TestReloadChangedChannelKeepsExactlyOneWorker(t *testing.T) {
	const typeName = "reload-worker-probe"
	RegisterFactory(typeName, func(channelName, _ string, _ *config.Config, _ *bus.MessageBus) (Channel, error) {
		return &reconcileTestChannel{name: channelName}, nil
	})

	msgBus := bus.NewMessageBus()
	t.Cleanup(msgBus.Close)
	manager, err := NewManager(config.DefaultConfig(), msgBus, nil)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	configFor := func(token string) *config.Config {
		cfg := config.DefaultConfig()
		cfg.Channels["probe"] = &config.Channel{
			Enabled:  true,
			Type:     typeName,
			Settings: config.RawNode(`{"token":"` + token + `"}`),
		}
		return cfg
	}

	if err := manager.Reload(context.Background(), configFor("one")); err != nil {
		t.Fatalf("first Reload: %v", err)
	}

	// The second Reload changes the hash, which is the replaced-in-place path.
	if err := manager.Reload(context.Background(), configFor("two")); err != nil {
		t.Fatalf("changed Reload: %v", err)
	}

	// The deferred registration goroutine must not be able to remove the
	// replacement's worker after this point.
	deadline := time.Now().Add(time.Second)
	for {
		manager.mu.RLock()
		_, channelOK := manager.channels["probe"]
		_, workerOK := manager.workers["probe"]
		channelCount := len(manager.channels)
		workerCount := len(manager.workers)
		manager.mu.RUnlock()

		if !channelOK || !workerOK {
			t.Fatalf("changed channel lost its runtime: channel=%v worker=%v", channelOK, workerOK)
		}
		if channelCount != 1 || workerCount != 1 {
			t.Fatalf("reload produced duplicates: channels=%d workers=%d", channelCount, workerCount)
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}
