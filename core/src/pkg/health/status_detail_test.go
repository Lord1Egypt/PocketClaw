package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/status"
)

// A synthetic value, named so it can never be mistaken for a real credential
// while grepping the tree. The gateway's actual token is generated at runtime
// and written to the PID file; nothing real is checked in.
const testGatewayToken = "test-gateway-token-not-a-real-credential"

func newDetailServer(t *testing.T, token string) *Server {
	t.Helper()
	s := NewServer("127.0.0.1", 0, token)
	s.SetStatusProbe(func() status.Snapshot {
		return status.Snapshot{
			Activity: status.Activity{ActiveTurns: 2, Completed: 7},
			Model:    status.Model{ActiveModel: "test-model", Provider: "test-provider"},
			Channels: []status.Channel{{
				Name: "telegram", Configured: true, Started: true, Running: true,
			}},
			Resources: status.Resources{MemoryRSSBytes: 1024, CPUSeconds: 1.5},
		}
	})
	return s
}

func decodeHealth(t *testing.T, body string) StatusResponse {
	t.Helper()
	var resp StatusResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode health response: %v (body %q)", err, body)
	}
	return resp
}

// TestBasicHealthRemainsBackwardCompatible proves that adding an
// authenticated detail mode did not change what an ordinary anonymous
// /health request returns. The launcher and the Android host both poll this
// for liveness, so a change here would be a silent regression in restart
// safety, not just a cosmetic one.
func TestBasicHealthRemainsBackwardCompatible(t *testing.T) {
	s := newDetailServer(t, testGatewayToken)

	rec := httptest.NewRecorder()
	s.healthHandler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "detail") {
		t.Fatalf("anonymous /health exposed a detail field: %s", body)
	}

	resp := decodeHealth(t, body)
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want \"ok\"", resp.Status)
	}
	if resp.Uptime == "" {
		t.Error("Uptime was empty")
	}
	if resp.PID == 0 {
		t.Error("PID was not reported")
	}
	if resp.Detail != nil {
		t.Error("Detail was populated on an unauthenticated basic request")
	}
}

// A detail request without the gateway credential must be refused outright.
// Downgrading it to a basic response would hide a broken credential behind a
// screen that merely looked empty.
func TestDetailedStatusRequiresAuthentication(t *testing.T) {
	s := newDetailServer(t, testGatewayToken)

	rec := httptest.NewRecorder()
	s.healthHandler(rec, httptest.NewRequest(http.MethodGet, "/health?detail=1", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	body := rec.Body.String()
	for _, leaked := range []string{"test-model", "test-provider", "telegram", "active_turns"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("rejected detail response leaked %q: %s", leaked, body)
		}
	}
}

func TestDetailedStatusRejectsWrongToken(t *testing.T) {
	s := newDetailServer(t, testGatewayToken)

	req := httptest.NewRequest(http.MethodGet, "/health?detail=1", nil)
	req.Header.Set("Authorization", "Bearer "+strings.Repeat("0", len(testGatewayToken)))
	rec := httptest.NewRecorder()
	s.healthHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// A gateway with no configured token has no credential to check, so detail
// mode must stay closed rather than fall open to every caller.
func TestDetailedStatusClosedWhenNoTokenConfigured(t *testing.T) {
	s := newDetailServer(t, "")

	req := httptest.NewRequest(http.MethodGet, "/health?detail=1", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	s.healthHandler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 when the gateway has no token", rec.Code)
	}
}

func TestAuthorizedDetailedStatusSucceeds(t *testing.T) {
	s := newDetailServer(t, testGatewayToken)

	req := httptest.NewRequest(http.MethodGet, "/health?detail=1", nil)
	req.Header.Set("Authorization", "Bearer "+testGatewayToken)
	rec := httptest.NewRecorder()
	s.healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decodeHealth(t, rec.Body.String())
	if resp.Detail == nil {
		t.Fatal("Detail was absent from an authorized detail response")
	}
	if resp.Detail.Activity.ActiveTurns != 2 || resp.Detail.Activity.Completed != 7 {
		t.Errorf("activity not carried through: %+v", resp.Detail.Activity)
	}
	if resp.Detail.Model.ActiveModel != "test-model" {
		t.Errorf("ActiveModel = %q", resp.Detail.Model.ActiveModel)
	}
	if len(resp.Detail.Channels) != 1 || resp.Detail.Channels[0].Name != "telegram" {
		t.Errorf("channels not carried through: %+v", resp.Detail.Channels)
	}
	// The basic fields must still be present alongside the detail.
	if resp.Status != "ok" || resp.Uptime == "" {
		t.Errorf("basic fields missing from detailed response: %+v", resp)
	}
}

// Uptime must only ever move forward, since the Status screen presents it as
// time since the gateway started.
func TestUptimeIsMonotonic(t *testing.T) {
	s := newDetailServer(t, testGatewayToken)

	read := func() time.Duration {
		rec := httptest.NewRecorder()
		s.healthHandler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
		parsed, err := time.ParseDuration(decodeHealth(t, rec.Body.String()).Uptime)
		if err != nil {
			t.Fatalf("parse uptime: %v", err)
		}
		return parsed
	}

	first := read()
	time.Sleep(5 * time.Millisecond)
	second := read()

	if second < first {
		t.Fatalf("uptime went backwards: %s then %s", first, second)
	}
	if second == 0 {
		t.Fatal("uptime never advanced")
	}
}
