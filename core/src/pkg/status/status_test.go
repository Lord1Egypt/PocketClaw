package status

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

// TestSnapshotSerializesOnlySafeScalars pins the whole Status payload's shape.
//
// The rule this enforces is that Status is built from explicit scalar DTOs
// rather than by handing an internal struct to the encoder. Adding a field
// that carries a prompt, a session key, a chat id, a tool argument or a
// credential to any of these structs fails here.
func TestSnapshotSerializesOnlySafeScalars(t *testing.T) {
	snapshot := Snapshot{
		Activity: Activity{
			ActiveTurns: 1, ActiveSubagents: 2, Waiting: 3,
			Completed: 4, Failed: 5, Cancelled: 6,
			ToolCalls: 7, ToolCallsFailed: 8,
			LastActivityUnix: 1757203200,
		},
		Model: Model{
			ActiveModel:     "mimo-v2.5",
			ConfiguredModel: "gpt-5.6-sol",
			Provider:        "openai",
			FallbackCount:   3,
		},
		Channels:  []Channel{{Name: "telegram", Configured: true, Started: true, Running: true}},
		Resources: Resources{MemoryRSSBytes: 88080384, CPUSeconds: 252.5},
	}

	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	wantTop := map[string]bool{"activity": true, "model": true, "channels": true, "resources": true}
	for key := range generic {
		if !wantTop[key] {
			t.Errorf("unexpected top-level Status field %q", key)
		}
	}

	activity, _ := generic["activity"].(map[string]any)
	wantActivity := map[string]bool{
		"active_turns": true, "active_subagents": true, "waiting": true,
		"completed": true, "failed": true, "cancelled": true,
		"tool_calls": true, "tool_calls_failed": true, "last_activity_unix": true,
	}
	for key := range activity {
		if !wantActivity[key] {
			t.Errorf("unexpected activity field %q", key)
		}
	}

	model, _ := generic["model"].(map[string]any)
	wantModel := map[string]bool{
		"active_model": true, "configured_model": true,
		"provider": true, "fallback_count": true,
	}
	for key := range model {
		if !wantModel[key] {
			t.Errorf("unexpected model field %q", key)
		}
	}

	// No field name anywhere in the payload may hint at content or identity.
	for _, banned := range []string{
		"session", "chat", "turn_id", "message", "prompt", "reply",
		"reasoning", "args", "result", "token", "key", "secret",
		"endpoint", "url", "path", "sender", "user",
	} {
		if strings.Contains(string(raw), `"`+banned) {
			t.Errorf("Status payload carries a field name containing %q: %s", banned, raw)
		}
	}
}

// A zero snapshot must encode cleanly, since that is what the screen shows
// before the gateway has done any work.
func TestZeroSnapshotEncodes(t *testing.T) {
	raw, err := json.Marshal(Snapshot{})
	if err != nil {
		t.Fatalf("marshal zero snapshot: %v", err)
	}
	if !strings.Contains(string(raw), `"active_turns":0`) {
		t.Errorf("zero snapshot lost its counters: %s", raw)
	}
}

func TestReadResourcesIsSaneOrZero(t *testing.T) {
	res := ReadResources()

	if runtime.GOOS != "linux" {
		if res != (Resources{}) {
			t.Fatalf("non-Linux platforms must report zeroes, got %+v", res)
		}
		return
	}

	// On Linux the reading process must have some resident memory. CPU time
	// may legitimately round to zero for a very short-lived test binary, so
	// only its sign is asserted.
	if res.MemoryRSSBytes == 0 {
		t.Errorf("RSS read as 0 on Linux")
	}
	if res.CPUSeconds < 0 {
		t.Errorf("CPU seconds negative: %v", res.CPUSeconds)
	}
}

// The executable name in /proc/<pid>/stat may itself contain spaces and
// parentheses, so the CPU fields must be located from after its closing
// parenthesis rather than by splitting the whole line.
func TestParseSelfStatHandlesAwkwardProcessNames(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("parser is only built on Linux")
	}

	// Fields 1 and 2 are the pid and the parenthesised executable name. The
	// parser starts counting after that closing parenthesis, where index 0 is
	// the state field, so utime lands at index 11 and stime at index 12 of
	// what it sees — which is offset by the two leading fields here.
	const leading = 2
	fields := make([]string, leading+22)
	fields[0] = "4242"
	fields[1] = "(weird (name) with spaces)"
	fields[2] = "R"
	for i := 3; i < len(fields); i++ {
		fields[i] = "0"
	}
	fields[leading+11] = "150"
	fields[leading+12] = "50"

	utime, stime, ok := parseSelfStatCPUTicks(strings.Join(fields, " "))
	if !ok {
		t.Fatal("parser rejected a valid stat line")
	}
	if utime != 150 || stime != 50 {
		t.Fatalf("utime=%d stime=%d, want 150 and 50", utime, stime)
	}
}

func TestParseSelfStatRejectsGarbage(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("parser is only built on Linux")
	}
	for _, line := range []string{"", "no parenthesis here", "1 (short) R 1 2 3"} {
		if _, _, ok := parseSelfStatCPUTicks(line); ok {
			t.Errorf("parser accepted garbage %q", line)
		}
	}
}
