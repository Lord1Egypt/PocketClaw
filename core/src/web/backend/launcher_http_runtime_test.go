package main

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/netbind"
)

func newTestLauncherRuntime(t *testing.T, public bool, handler http.Handler) *launcherHTTPRuntime {
	t.Helper()
	initial, err := openLauncherListeners("", public, "0")
	if err != nil {
		t.Fatal(err)
	}
	runtime := newLauncherHTTPRuntime(handler, "", public, public, initial)
	runtime.Start()
	t.Cleanup(runtime.Shutdown)
	return runtime
}

func runtimeAddresses(runtime *launcherHTTPRuntime) []string {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	addresses := make([]string, 0, len(runtime.groups))
	for _, group := range runtime.groups {
		addresses = append(addresses, group.listener.Addr().String())
	}
	return addresses
}

func TestLauncherHTTPRuntimeAppliesPublicModeWithoutReplacingHandler(t *testing.T) {
	var requests atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, "same authenticated dashboard")
	})
	runtime := newTestLauncherRuntime(t, false, handler)
	port := runtime.port

	if err := runtime.ApplyPublicMode(true); err != nil {
		t.Fatalf("OFF -> ON: %v", err)
	}
	if !runtime.PublicMode() {
		t.Fatal("runtime did not enter Public Mode")
	}
	publicAddresses := strings.Join(runtimeAddresses(runtime), ",")
	if !strings.Contains(publicAddresses, "0.0.0.0:") && !strings.Contains(publicAddresses, "[::]:") {
		t.Fatalf("public listeners = %s, want wildcard", publicAddresses)
	}

	response, err := http.Get("http://" + net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatalf("Dashboard after OFF -> ON: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || requests.Load() != 1 {
		t.Fatalf("same handler was not served after rebind: status=%d requests=%d", response.StatusCode, requests.Load())
	}

	if err := runtime.ApplyPublicMode(false); err != nil {
		t.Fatalf("ON -> OFF: %v", err)
	}
	if runtime.PublicMode() {
		t.Fatal("runtime did not return to local mode")
	}
	localAddresses := strings.Join(runtimeAddresses(runtime), ",")
	if strings.Contains(localAddresses, "0.0.0.0:") || strings.Contains(localAddresses, "[::]:") {
		t.Fatalf("local listeners = %s, want loopback only", localAddresses)
	}
}

func TestLauncherHTTPRuntimeRestoresPreviousListenerWhenNewBindFails(t *testing.T) {
	runtime := newTestLauncherRuntime(t, false, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	port := runtime.port
	realOpen := runtime.open
	runtime.open = func(hostInput string, public bool, requestedPort string) (netbind.OpenResult, error) {
		if public {
			return netbind.OpenResult{}, errors.New("injected wildcard bind failure")
		}
		return realOpen(hostInput, public, requestedPort)
	}

	if err := runtime.ApplyPublicMode(true); err == nil {
		t.Fatal("ApplyPublicMode() succeeded despite injected bind failure")
	}
	if runtime.PublicMode() {
		t.Fatal("failed rebind changed the actual mode")
	}

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatalf("restored local Dashboard is unavailable: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("restored Dashboard status = %d", response.StatusCode)
	}
}
