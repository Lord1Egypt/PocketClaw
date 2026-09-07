package channels

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/status"
)

var errSnapshotStartFailed = errors.New("channel refused to start")

// snapshotTestChannel lets each test decide whether Start succeeds and what
// the running flag says afterwards.
type snapshotTestChannel struct {
	name     string
	startErr error
	running  atomic.Bool
}

func (c *snapshotTestChannel) Name() string { return c.name }

func (c *snapshotTestChannel) Start(context.Context) error {
	if c.startErr != nil {
		return c.startErr
	}
	c.running.Store(true)
	return nil
}

func (c *snapshotTestChannel) Stop(context.Context) error {
	c.running.Store(false)
	return nil
}

func (c *snapshotTestChannel) Send(context.Context, bus.OutboundMessage) ([]string, error) {
	return nil, nil
}

func (c *snapshotTestChannel) IsRunning() bool                     { return c.running.Load() }
func (c *snapshotTestChannel) IsAllowed(string) bool               { return true }
func (c *snapshotTestChannel) IsAllowedSender(bus.SenderInfo) bool { return true }
func (c *snapshotTestChannel) ReasoningChannelID() string          { return "" }

func snapshotByName(t *testing.T, snapshots []status.Channel, name string) status.Channel {
	t.Helper()
	for _, s := range snapshots {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("no snapshot for channel %q in %+v", name, snapshots)
	return status.Channel{}
}

// TestSnapshotChannelsReportsFailedStartTruthfully guards against the worst
// thing this screen could do: tell someone a channel is running when it never
// started. A channel that fails to start stays in the manager's configured
// set, so only the absence of a worker distinguishes it.
func TestSnapshotChannelsReportsFailedStartTruthfully(t *testing.T) {
	m := newTestManager()
	m.config = &config.Config{}
	m.channels["telegram"] = &snapshotTestChannel{name: "telegram"}
	m.channels["discord"] = &snapshotTestChannel{name: "discord", startErr: errSnapshotStartFailed}

	if err := m.StartAll(context.Background()); err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	t.Cleanup(func() { _ = m.StopAll(context.Background()) })

	snapshots := m.SnapshotChannels()
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d: %+v", len(snapshots), snapshots)
	}

	started := snapshotByName(t, snapshots, "telegram")
	if !started.Configured || !started.Started || !started.Running {
		t.Errorf("a started channel reported %+v", started)
	}

	failed := snapshotByName(t, snapshots, "discord")
	if !failed.Configured {
		t.Errorf("a failed channel must still be reported as configured: %+v", failed)
	}
	if failed.Started {
		t.Errorf("a channel whose Start returned an error must not be Started: %+v", failed)
	}
	if failed.Running {
		t.Errorf("a channel that failed to start must never be Running: %+v", failed)
	}
}

// A latched running flag on a channel with no worker must not surface as
// Running either, which is why the snapshot requires both facts.
func TestSnapshotChannelsIgnoresStaleRunningFlagWithoutWorker(t *testing.T) {
	m := newTestManager()
	stale := &snapshotTestChannel{name: "telegram"}
	stale.running.Store(true)
	m.channels["telegram"] = stale

	snapshot := snapshotByName(t, m.SnapshotChannels(), "telegram")
	if snapshot.Running {
		t.Errorf("a channel with no worker reported Running: %+v", snapshot)
	}
	if !snapshot.Configured {
		t.Errorf("expected Configured: %+v", snapshot)
	}
	if snapshot.Started {
		t.Errorf("expected not Started: %+v", snapshot)
	}
}

// The internal `pico` id is PocketClaw's own Web Console transport and must
// reach the screen under its product name.
func TestSnapshotChannelsUsesPocketClawDisplayName(t *testing.T) {
	m := newTestManager()
	m.channels[config.ChannelPico] = &snapshotTestChannel{name: config.ChannelPico}

	snapshots := m.SnapshotChannels()
	for _, s := range snapshots {
		if s.Name == config.ChannelPico {
			t.Fatalf("internal channel id leaked to the Status payload: %+v", snapshots)
		}
	}
	snapshotByName(t, snapshots, "pocketclaw")
}

// The snapshot must carry channel identity only — never an account, bot
// username or chat id — so its serialized shape is pinned here.
func TestSnapshotChannelsExposesOnlySafeFields(t *testing.T) {
	m := newTestManager()
	m.channels["telegram"] = &snapshotTestChannel{name: "telegram"}

	snapshots := m.SnapshotChannels()
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %+v", snapshots)
	}
	got := snapshots[0]
	want := status.Channel{Name: "telegram", Configured: true, Started: false, Running: false}
	if got != want {
		t.Fatalf("snapshot = %+v, want %+v", got, want)
	}
}
