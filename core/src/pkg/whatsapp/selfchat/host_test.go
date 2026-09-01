package selfchat

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// answerOnce plays the Android host: it waits for a request file, records it,
// and writes the reply the test asked for.
func answerOnce(t *testing.T, dir, status, reason string) chan hostRequest {
	t.Helper()
	seen := make(chan hostRequest, 1)
	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				return
			}
			for _, entry := range entries {
				name := entry.Name()
				if !strings.HasPrefix(name, "req-") || !strings.HasSuffix(name, ".json") {
					continue
				}
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					continue
				}
				var req hostRequest
				if json.Unmarshal(data, &req) != nil {
					continue
				}
				os.Remove(filepath.Join(dir, name))
				reply, _ := json.Marshal(hostResponse{ID: req.ID, Status: status, Error: reason})
				os.WriteFile(filepath.Join(dir, "res-"+req.ID+".json"), reply, 0o600)
				seen <- req
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return seen
}

func TestOpenSendsNumberAndMessageAndSucceeds(t *testing.T) {
	dir := t.TempDir()
	seen := answerOnce(t, dir, "opened", "")

	if err := openInDir(context.Background(), dir, "+201012345678", "PocketClaw Agent Test"); err != nil {
		t.Fatalf("openInDir returned %v", err)
	}

	select {
	case req := <-seen:
		if req.Action != ActionOpenSelfChat {
			t.Errorf("action = %q", req.Action)
		}
		if req.Number != "+201012345678" {
			t.Errorf("number = %q", req.Number)
		}
		if req.Message != "PocketClaw Agent Test" {
			t.Errorf("message = %q", req.Message)
		}
	default:
		t.Fatal("host never saw a request")
	}

	// Neither the request nor the reply may outlive the call: a leftover
	// request would open WhatsApp out of the blue the next time the host starts.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("outbox still holds %d file(s)", len(entries))
	}
}

func TestOpenCarriesUnicodeUnchanged(t *testing.T) {
	dir := t.TempDir()
	const message = "مرحبا من PocketClaw 🦞"
	seen := answerOnce(t, dir, "opened", "")

	if err := openInDir(context.Background(), dir, "+905321234567", message); err != nil {
		t.Fatalf("openInDir returned %v", err)
	}
	req := <-seen
	if req.Message != message {
		t.Errorf("message = %q, want %q", req.Message, message)
	}
}

func TestOpenReportsHostFailureReason(t *testing.T) {
	dir := t.TempDir()
	answerOnce(t, dir, "error", "not_installed")

	err := openInDir(context.Background(), dir, "+201012345678", "hello")
	if err == nil {
		t.Fatal("expected an error when the host reports not_installed")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error = %q, want it to name the missing app", err)
	}
}

func TestOpenWithoutHostIsUnavailable(t *testing.T) {
	if err := openInDir(context.Background(), "", "+201012345678", "hello"); err != ErrHostUnavailable {
		t.Errorf("error = %v, want ErrHostUnavailable", err)
	}
	if err := openInDir(context.Background(), filepath.Join(t.TempDir(), "missing"), "+2010", "hi"); err != ErrHostUnavailable {
		t.Errorf("error = %v, want ErrHostUnavailable", err)
	}
}

func TestOpenAbandonsRequestWhenCancelled(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := openInDir(ctx, dir, "+201012345678", "hello"); err != context.Canceled {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("abandoned request left %d file(s) behind", len(entries))
	}
}

func TestHostAvailableFollowsTheEnvironment(t *testing.T) {
	t.Setenv(EnvHostOutbox, "")
	if HostAvailable() {
		t.Error("HostAvailable() is true with no host directory")
	}
	dir := t.TempDir()
	t.Setenv(EnvHostOutbox, dir)
	if !HostAvailable() {
		t.Error("HostAvailable() is false with a host directory present")
	}
	if HostOutboxDir() != dir {
		t.Errorf("HostOutboxDir() = %q", HostOutboxDir())
	}
}
