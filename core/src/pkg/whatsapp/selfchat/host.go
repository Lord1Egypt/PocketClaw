package selfchat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EnvHostOutbox names the directory the Android host watches for requests.
//
// Core runs as a child process of the Android app and cannot start an Activity
// itself, so opening WhatsApp is a request the host performs. The app sets this
// variable when it spawns Core; anywhere else — desktop, CI — it is unset and
// the feature reports itself unavailable rather than failing obscurely.
const EnvHostOutbox = "POCKETCLAW_ANDROID_HOST_OUTBOX"

// ActionOpenSelfChat is the only action this bridge carries.
const ActionOpenSelfChat = "whatsapp_self_chat"

// How long to wait for the host to answer, and how often to look. The host
// fires an Intent as soon as its watcher sees the file, so a normal answer
// lands well inside the first second; the ceiling only bounds a host that
// stopped watching.
const (
	hostReplyTimeout  = 12 * time.Second
	hostReplyInterval = 60 * time.Millisecond
)

// ErrHostUnavailable is returned when this process is not running under the
// PocketClaw Android host.
var ErrHostUnavailable = errors.New("whatsapp: WhatsApp Self-Chat is only available inside the PocketClaw Android app")

// ErrHostTimeout is returned when the host never answered a request.
var ErrHostTimeout = errors.New("whatsapp: the PocketClaw Android app did not answer; open the app and try again")

type hostRequest struct {
	ID      string `json:"id"`
	Action  string `json:"action"`
	Number  string `json:"number"`
	Message string `json:"message"`
}

type hostResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// HostOutboxDir returns the watched directory, or "" when there is no host.
func HostOutboxDir() string {
	return os.Getenv(EnvHostOutbox)
}

// HostAvailable reports whether an Android host is listening for requests.
func HostAvailable() bool {
	dir := HostOutboxDir()
	if dir == "" {
		return false
	}
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// Open asks the Android host to open WhatsApp on the user's own chat with
// message prepared in the compose box.
//
// It returns nil once the host reports the Intent started. That is "opened",
// never "sent": the message sits unsent in WhatsApp's compose box and only the
// user can send it.
func Open(ctx context.Context, number, message string) error {
	return openInDir(ctx, HostOutboxDir(), number, message)
}

func openInDir(ctx context.Context, dir, number, message string) error {
	if dir == "" {
		return ErrHostUnavailable
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return ErrHostUnavailable
	}

	id, err := newRequestID()
	if err != nil {
		return err
	}

	payload, err := json.Marshal(hostRequest{
		ID:      id,
		Action:  ActionOpenSelfChat,
		Number:  number,
		Message: message,
	})
	if err != nil {
		return err
	}

	// Write beside the target and rename, so the host's watcher never reads a
	// half-written request.
	target := filepath.Join(dir, "req-"+id+".json")
	staging := target + ".tmp"
	if err := os.WriteFile(staging, payload, 0o600); err != nil {
		return fmt.Errorf("whatsapp: could not reach the Android app: %w", err)
	}
	if err := os.Rename(staging, target); err != nil {
		os.Remove(staging)
		return fmt.Errorf("whatsapp: could not reach the Android app: %w", err)
	}

	resp, err := awaitResponse(ctx, dir, id)
	if err != nil {
		// A request nobody answered would otherwise sit in the directory and be
		// acted on whenever the host next starts, opening WhatsApp out of the blue.
		os.Remove(target)
		return err
	}
	if resp.Status != "opened" {
		return hostError(resp.Error)
	}
	return nil
}

func awaitResponse(ctx context.Context, dir, id string) (*hostResponse, error) {
	path := filepath.Join(dir, "res-"+id+".json")
	deadline := time.NewTimer(hostReplyTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(hostReplyInterval)
	defer ticker.Stop()

	for {
		data, err := os.ReadFile(path)
		if err == nil {
			os.Remove(path)
			var resp hostResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				return nil, fmt.Errorf("whatsapp: the Android app sent an unreadable reply: %w", err)
			}
			return &resp, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, ErrHostTimeout
		case <-ticker.C:
		}
	}
}

// hostError maps the host's machine-readable reason onto a message the model
// and the user can both act on. An unrecognised reason is reported as-is
// rather than swallowed.
func hostError(reason string) error {
	switch reason {
	case "not_installed":
		return errors.New("whatsapp: WhatsApp is not installed on this device")
	case "invalid_number":
		return errors.New("whatsapp: the configured self number is not a valid WhatsApp number")
	case "start_failed":
		return errors.New("whatsapp: Android refused to open WhatsApp; bring PocketClaw to the foreground and try again")
	case "":
		return errors.New("whatsapp: the Android app could not open WhatsApp")
	default:
		return fmt.Errorf("whatsapp: the Android app could not open WhatsApp (%s)", reason)
	}
}

func newRequestID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("whatsapp: could not create a request id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
