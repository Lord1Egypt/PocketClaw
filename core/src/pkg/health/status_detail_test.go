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

// TestBasicHealthKeepsLegacyUptimeString pins the field existing consumers
// already read.
//
// The launcher and the Android host both parse /health, and the anonymous
// response must keep Go's duration formatting exactly as it always was. The
// numeric field added for Status is additive and lives only in the
// authenticated detail payload.
func TestBasicHealthKeepsLegacyUptimeString(t *testing.T) {
	s := newDetailServer(t, testGatewayToken)
	s.startTime = time.Now().Add(-27707765309 * time.Nanosecond)

	rec := httptest.NewRecorder()
	s.healthHandler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	body := rec.Body.String()
	if strings.Contains(body, "uptime_seconds") {
		t.Fatalf("the numeric field leaked into the anonymous response: %s", body)
	}

	resp := decodeHealth(t, body)
	// Still a Go duration string, still parseable as one.
	if !strings.HasSuffix(resp.Uptime, "s") {
		t.Errorf("legacy Uptime lost its duration formatting: %q", resp.Uptime)
	}
	parsed, err := time.ParseDuration(resp.Uptime)
	if err != nil {
		t.Fatalf("legacy Uptime is no longer a Go duration string: %q (%v)", resp.Uptime, err)
	}
	if parsed < 27*time.Second || parsed > 29*time.Second {
		t.Errorf("legacy Uptime = %q, want about 27.7s", resp.Uptime)
	}
}

// TestDetailUptimeIsWholeSeconds is the regression test for the value the
// Status screen used to render.
//
// 27.707765309s must reach the screen as 27. Not 27707765309, which is that
// duration's nanosecond count, and not "27.707765309s", which is what the
// screen used to print verbatim because nothing on the path ever parsed it.
func TestDetailUptimeIsWholeSeconds(t *testing.T) {
	tests := []struct {
		name string
		age  time.Duration
		want int64
	}{
		{"the observed 27.7 seconds", 27707765309 * time.Nanosecond, 27},
		{"sub-second floors to zero", 940 * time.Millisecond, 0},
		{"four minutes twelve", 4*time.Minute + 12*time.Second, 252},
		{"one hour twenty-four", time.Hour + 24*time.Minute, 5040},
		{"two days three hours", 2*24*time.Hour + 3*time.Hour, 183600},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newDetailServer(t, testGatewayToken)
			s.startTime = time.Now().Add(-tc.age)

			req := httptest.NewRequest(http.MethodGet, "/health?detail=1", nil)
			req.Header.Set("Authorization", "Bearer "+testGatewayToken)
			rec := httptest.NewRecorder()
			s.healthHandler(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			body := rec.Body.String()

			// The nanosecond count must never appear anywhere in the payload.
			if strings.Contains(body, "27707765309") {
				t.Fatalf("a nanosecond count reached the payload: %s", body)
			}

			resp := decodeHealth(t, body)
			if resp.Detail == nil {
				t.Fatal("no detail in an authorized response")
			}
			got := resp.Detail.System.UptimeSeconds
			// Allow one second for the time that passes during the request.
			if got != tc.want && got != tc.want+1 {
				t.Fatalf("UptimeSeconds = %d, want %d", got, tc.want)
			}
		})
	}
}

// The field must be a JSON number, not a string: the whole point is that the
// unit is carried by the contract rather than by text a reader has to parse.
func TestDetailUptimeSerializesAsANumber(t *testing.T) {
	s := newDetailServer(t, testGatewayToken)
	s.startTime = time.Now().Add(-252 * time.Second)

	req := httptest.NewRequest(http.MethodGet, "/health?detail=1", nil)
	req.Header.Set("Authorization", "Bearer "+testGatewayToken)
	rec := httptest.NewRecorder()
	s.healthHandler(rec, req)

	var generic map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &generic); err != nil {
		t.Fatalf("decode: %v", err)
	}
	detail, _ := generic["detail"].(map[string]any)
	system, _ := detail["system"].(map[string]any)
	raw, present := system["uptime_seconds"]
	if !present {
		t.Fatalf("uptime_seconds missing from detail.system: %s", rec.Body.String())
	}
	if _, ok := raw.(float64); !ok {
		t.Fatalf("uptime_seconds is %T (%v), want a JSON number", raw, raw)
	}
}
