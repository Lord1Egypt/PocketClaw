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
//
// Note what the snapshot does not claim. Started=false records that no worker
// exists, which is all the runtime knows; it is not evidence of a failure,
// because StartAll keeps its failures in local variables and nothing retains
// them. The Status screen therefore renders this as Stopped, never as failed.
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

// TestSnapshotChannelsDoesNotDistinguishFailureFromNotStarted documents why the
// Status screen has no "failed to start" state.
//
// A channel that failed to start and one that StartAll has not reached yet
// produce byte-identical snapshots, because the only difference between them —
// the error StartAll saw — is discarded. Any label that named one of these a
// failure would necessarily also name the other one, so neither is named.
//
// If a retained start-failure state is ever added to the Manager, this test is
// the place that should start failing.
func TestSnapshotChannelsDoesNotDistinguishFailureFromNotStarted(t *testing.T) {
	failed := newTestManager()
	failed.config = &config.Config{}
	failed.channels["telegram"] = &snapshotTestChannel{
		name:     "telegram",
		startErr: errSnapshotStartFailed,
	}
	// A single failing channel makes StartAll return an error, which is the
	// only place the failure exists; nothing of it survives the call.
	_ = failed.StartAll(context.Background())
	t.Cleanup(func() { _ = failed.StopAll(context.Background()) })

	neverStarted := newTestManager()
	neverStarted.config = &config.Config{}
	neverStarted.channels["telegram"] = &snapshotTestChannel{name: "telegram"}
	// StartAll is deliberately not called: this is the boot window, before
	// channels have been started at all.

	afterFailure := failed.SnapshotChannels()
	beforeStart := neverStarted.SnapshotChannels()

	if len(afterFailure) != 1 || len(beforeStart) != 1 {
		t.Fatalf("expected one snapshot each, got %+v and %+v", afterFailure, beforeStart)
	}
	if afterFailure[0] != beforeStart[0] {
		t.Fatalf(
			"a failed channel and a not-yet-started one are now distinguishable "+
				"(%+v vs %+v); if the Manager retains start failures, Status may "+
				"report them — update this test and the screen together",
			afterFailure[0], beforeStart[0],
		)
	}
	if afterFailure[0].Running {
		t.Errorf("a channel with no worker reported Running: %+v", afterFailure[0])
	}
}
