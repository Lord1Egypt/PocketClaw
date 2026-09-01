package integrationtools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/whatsapp/selfchat"
)

func writeSelfChatConfig(t *testing.T, path, number string) {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"channel_list": map[string]any{
			selfchat.ConfigKey: map[string]any{
				"type":     selfchat.ConfigKey,
				"enabled":  false,
				"settings": map[string]any{"self_number": number},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Connect, Change and Disconnect must reach the tool without a gateway
// restart. The console applies the configuration through the gateway's safe
// restart path so the rest of Core catches up, but that path deliberately
// declines to interrupt a running turn — so if the tool only ever saw the
// number it booted with, Disconnect would leave a stale number usable for as
// long as the gateway stayed busy.
//
// This drives the real provider against a config file that changes underneath
// it, which is the only thing that proves the read is not a snapshot.
func TestWhatsAppSelfChatFollowsTheConfigFileWithoutARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv(config.EnvConfig, path)

	tool := NewWhatsAppSelfChatTool(selfchat.ConfiguredNumber)
	var opened []string
	tool.open = func(_ context.Context, number, _ string) error {
		opened = append(opened, number)
		return nil
	}

	// Connect.
	writeSelfChatConfig(t, path, "+201012345678")
	if result := tool.Execute(context.Background(), map[string]any{"message": "hi"}); result.IsError {
		t.Fatalf("after Connect the tool reported: %s", result.ForLLM)
	}

	// Change, with the number written the way the console normalises it.
	writeSelfChatConfig(t, path, "+905321234567")
	if result := tool.Execute(context.Background(), map[string]any{"message": "hi"}); result.IsError {
		t.Fatalf("after Change the tool reported: %s", result.ForLLM)
	}

	if want := []string{"+201012345678", "+905321234567"}; len(opened) != 2 ||
		opened[0] != want[0] || opened[1] != want[1] {
		t.Fatalf("numbers used = %v, want %v", opened, want)
	}

	// Disconnect.
	writeSelfChatConfig(t, path, "")
	result := tool.Execute(context.Background(), map[string]any{"message": "hi"})
	if !result.IsError {
		t.Fatal("after Disconnect the tool still opened WhatsApp")
	}
	if !strings.Contains(result.ForLLM, "not configured") {
		t.Errorf("ForLLM = %q, want it to report Self-Chat as not configured", result.ForLLM)
	}
	if len(opened) != 2 {
		t.Errorf("the host was asked to open WhatsApp %d times, want 2", len(opened))
	}
}
