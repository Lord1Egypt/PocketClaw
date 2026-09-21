package wecom

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReqIDStorePersistsRoutes(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "reqids.json")
	store := newReqIDStore(storePath)
	if err := store.Put("chat-1", "req-1", 2, time.Hour); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	reloaded := newReqIDStore(storePath)
	route, ok := reloaded.Get("chat-1")
	if !ok {
		t.Fatal("expected persisted route to be loaded")
	}
	if route.ChatID != "chat-1" || route.ReqID != "req-1" || route.ChatType != 2 {
		t.Fatalf("loaded route = %+v", route)
	}
}

func TestDefaultReqIDStoreMigratesWithoutCreatingPicoPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	canonical, legacy := defaultReqIDStorePaths()
	relativeCanonical, err := filepath.Rel(home, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(relativeCanonical), "pico") {
		t.Fatalf("canonical request-ID store contains Pico: %q", canonical)
	}
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	legacyStore := newReqIDStore(legacy)
	if err := legacyStore.Put("chat-legacy", "req-legacy", 2, time.Hour); err != nil {
		t.Fatal(err)
	}

	migrated := newReqIDStore("")
	if route, ok := migrated.Get("chat-legacy"); !ok || route.ReqID != "req-legacy" {
		t.Fatalf("legacy route was not migrated: route=%+v ok=%v", route, ok)
	}
	if _, err := os.Stat(canonical); err != nil {
		t.Fatalf("canonical request-ID store was not created: %v", err)
	}
	if _, err := os.Stat(legacy); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy request-ID store survived migration: %v", err)
	}
}
