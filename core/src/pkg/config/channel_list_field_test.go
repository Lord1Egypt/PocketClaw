package config

import (
	"encoding/json"
	"testing"
)

// Channel config list fields are canonically JSON arrays. Older console builds
// wrote them as a newline-joined string, so both shapes must still load — a
// config on disk today may carry either.

func TestChannelListFieldAcceptsBothShapes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "canonical array", raw: `["123","456"]`, want: []string{"123", "456"}},
		{name: "canonical single-entry array", raw: `["pocketclaw-user"]`, want: []string{"pocketclaw-user"}},
		{name: "legacy bare string", raw: `"pocketclaw-user"`, want: []string{"pocketclaw-user"}},
		{name: "numeric identifier", raw: `123`, want: []string{"123"}},
		{name: "mixed array", raw: `["abc",123]`, want: []string{"abc", "123"}},
		{name: "null is empty", raw: `null`, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got FlexibleStringSlice
			if err := json.Unmarshal([]byte(tt.raw), &got); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v", tt.raw, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Unmarshal(%s) = %#v, want %#v", tt.raw, []string(got), tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("entry %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestChannelListFieldRejectsInvalidTypes pins that a shape we cannot interpret
// still fails rather than silently loading as something else.
func TestChannelListFieldRejectsInvalidTypes(t *testing.T) {
	for _, raw := range []string{`{"a":1}`, `[{"a":1}`} {
		var got FlexibleStringSlice
		if err := json.Unmarshal([]byte(raw), &got); err == nil {
			t.Errorf("Unmarshal(%s) succeeded with %#v, want an error", raw, []string(got))
		}
	}
}

// TestGroupTriggerPrefixesAcceptsBothShapes covers a field that used to be a
// plain []string: the console's newline-joined write made it fail to decode
// outright, taking the surrounding channel settings with it.
func TestGroupTriggerPrefixesAcceptsBothShapes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "canonical array", raw: `{"prefixes":["!","/"]}`, want: 2},
		{name: "legacy bare string", raw: `{"prefixes":"!"}`, want: 1},
		{name: "legacy joined string stays one entry", raw: `{"prefixes":"!\n/"}`, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gt GroupTriggerConfig
			if err := json.Unmarshal([]byte(tt.raw), &gt); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v", tt.raw, err)
			}
			if len(gt.Prefixes) != tt.want {
				t.Errorf("Prefixes = %#v, want %d entries", []string(gt.Prefixes), tt.want)
			}
		})
	}
}

func TestPicoAllowOriginsAcceptsBothShapes(t *testing.T) {
	for _, raw := range []string{
		`{"allow_origins":["https://a.example"]}`,
		`{"allow_origins":"https://a.example"}`,
	} {
		var p PocketClawSettings
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			t.Fatalf("Unmarshal(%s) error = %v", raw, err)
		}
		if len(p.AllowOrigins) != 1 || p.AllowOrigins[0] != "https://a.example" {
			t.Errorf("AllowOrigins = %#v, want one origin", []string(p.AllowOrigins))
		}
	}
}

func TestOneBotGroupTriggerPrefixAcceptsBothShapes(t *testing.T) {
	for _, raw := range []string{
		`{"group_trigger_prefix":["!"]}`,
		`{"group_trigger_prefix":"!"}`,
	} {
		var s OneBotSettings
		if err := json.Unmarshal([]byte(raw), &s); err != nil {
			t.Fatalf("Unmarshal(%s) error = %v", raw, err)
		}
		if len(s.GroupTriggerPrefix) != 1 || s.GroupTriggerPrefix[0] != "!" {
			t.Errorf("GroupTriggerPrefix = %#v, want one prefix", []string(s.GroupTriggerPrefix))
		}
	}
}

// TestChannelAllowFromLegacyStringLoads is the end-to-end shape: a whole
// channel block carrying the legacy value must decode, keeping the channel
// usable rather than failing the surrounding config.
func TestChannelAllowFromLegacyStringLoads(t *testing.T) {
	raw := `{"enabled":true,"type":"pocketclaw","allow_from":"pocketclaw-user",` +
		`"group_trigger":{"prefixes":"!"},` +
		`"settings":{"allow_origins":"https://a.example"}}`

	var ch Channel
	if err := json.Unmarshal([]byte(raw), &ch); err != nil {
		t.Fatalf("Unmarshal(channel) error = %v", err)
	}
	if len(ch.AllowFrom) != 1 || ch.AllowFrom[0] != "pocketclaw-user" {
		t.Errorf("AllowFrom = %#v, want [pico-user]", []string(ch.AllowFrom))
	}
	if len(ch.GroupTrigger.Prefixes) != 1 {
		t.Errorf("Prefixes = %#v, want one entry", []string(ch.GroupTrigger.Prefixes))
	}
}
