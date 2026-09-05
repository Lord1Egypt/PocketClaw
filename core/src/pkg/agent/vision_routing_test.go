package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Vision routing decides on the current turn and nothing else.
//
// The routing itself is exercised end to end by the agent-loop tests above;
// these pin the decision inputs, which is where a regression would be silent:
// a change to media detection or to candidate resolution would reroute every
// turn without any test failing on the loop.

func userMessage(content string, media ...string) providers.Message {
	return providers.Message{Role: "user", Content: content, Media: media}
}

func TestCurrentTurnMediaDetection(t *testing.T) {
	cases := []struct {
		name     string
		messages []providers.Message
		want     bool
	}{
		{
			name:     "text only",
			messages: []providers.Message{userMessage("summarise the repo")},
			want:     false,
		},
		{
			name:     "one attached image",
			messages: []providers.Message{userMessage("what is this", "media://abc")},
			want:     true,
		},
		{
			name: "several attached images",
			messages: []providers.Message{
				userMessage("compare these", "media://a", "media://b", "media://c"),
			},
			want: true,
		},
		{
			name: "a resolved image path tag",
			messages: []providers.Message{
				userMessage("look at [image:/tmp/sample.png] please"),
			},
			want: true,
		},
		{
			// The generic placeholder carries no path, so there is nothing for
			// a vision model to read and no reason to reroute.
			name:     "a bare image placeholder",
			messages: []providers.Message{userMessage("they sent [image]")},
			want:     false,
		},
		{
			name:     "a non-image attachment placeholder",
			messages: []providers.Message{userMessage("here is [file: notes.txt]")},
			want:     false,
		},
		{
			name:     "a video placeholder",
			messages: []providers.Message{userMessage("clip attached [video]")},
			want:     false,
		},
		{
			name: "an image later in the same turn",
			messages: []providers.Message{
				userMessage("first"),
				userMessage("and this", "media://abc"),
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := messagesContainCurrentTurnMediaTurn(tc.messages); got != tc.want {
				t.Fatalf("messagesContainCurrentTurnMediaTurn() = %v, want %v", got, tc.want)
			}
		})
	}
}

// The property the brief calls critical: a text turn that follows an image turn
// goes back to the default model. It holds because detection reads only from
// currentTurnStart, never the whole transcript.
func TestMediaDetectionIgnoresEarlierTurns(t *testing.T) {
	transcript := []providers.Message{
		userMessage("what is in this", "media://abc"),
		{Role: "assistant", Content: "a cat"},
		userMessage("now write a haiku about it"),
	}

	// The image turn: detection starts at the image.
	if !messagesContainCurrentTurnMediaTurn(currentTurnMessages(transcript, 0)) {
		t.Fatal("the image turn was not detected as a media turn")
	}

	// The follow-up text turn: the image is history, so this is not a media
	// turn and the default model keeps the request.
	if messagesContainCurrentTurnMediaTurn(currentTurnMessages(transcript, 2)) {
		t.Fatal("history media leaked into the current turn's routing decision")
	}

	// And a new image turn routes again.
	withNewImage := append(transcript, userMessage("and this one", "media://def"))
	if !messagesContainCurrentTurnMediaTurn(currentTurnMessages(withNewImage, 3)) {
		t.Fatal("a fresh image turn was not detected")
	}
}

func TestCurrentTurnMessages_ClampsAnOutOfRangeStart(t *testing.T) {
	transcript := []providers.Message{userMessage("a"), userMessage("b")}
	if got := len(currentTurnMessages(transcript, -1)); got != 2 {
		t.Fatalf("negative start gave %d messages, want the whole slice", got)
	}
	if got := len(currentTurnMessages(transcript, 99)); got != 0 {
		t.Fatalf("start past the end gave %d messages, want none", got)
	}
}

func visionConfig(imageModel string) *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName:  "text-model",
				ImageModel: imageModel,
			},
		},
		ModelList: []*config.ModelConfig{
			{ModelName: "text-model", Model: "openai/text-model"},
			{ModelName: "vision-model", Model: "openai/vision-model"},
		},
	}
}

// With no dedicated vision model there are no image candidates, so routeMediaTurn
// finds nothing to switch to and an image turn stays on the default. That is the
// behaviour every existing install already has, and configuring nothing must
// preserve it.
func TestNoVisionModelLeavesImageTurnsOnTheDefault(t *testing.T) {
	cfg := visionConfig("")
	imageCandidates := resolveModelCandidates(
		cfg,
		cfg.Agents.Defaults.Provider,
		cfg.Agents.Defaults.ImageModel,
		cfg.Agents.Defaults.ImageModelFallbacks,
	)
	if len(imageCandidates) != 0 {
		t.Fatalf("len(imageCandidates) = %d, want 0 when no vision model is set", len(imageCandidates))
	}

	textCandidates := resolveModelCandidates(
		cfg,
		cfg.Agents.Defaults.Provider,
		cfg.Agents.Defaults.GetModelName(),
		cfg.Agents.Defaults.ModelFallbacks,
	)
	if len(textCandidates) != 1 {
		t.Fatalf("len(textCandidates) = %d, want the default model", len(textCandidates))
	}
}

func TestVisionModelResolvesToItsOwnCandidate(t *testing.T) {
	cfg := visionConfig("vision-model")
	imageCandidates := resolveModelCandidates(
		cfg,
		cfg.Agents.Defaults.Provider,
		cfg.Agents.Defaults.ImageModel,
		cfg.Agents.Defaults.ImageModelFallbacks,
	)
	if len(imageCandidates) != 1 {
		t.Fatalf("len(imageCandidates) = %d, want 1", len(imageCandidates))
	}
	if imageCandidates[0].Model != "vision-model" {
		t.Fatalf("image candidate model = %q, want %q", imageCandidates[0].Model, "vision-model")
	}
}

// A dangling vision reference does NOT degrade quietly, which is why the write
// paths validate it and why deleting the referenced entry clears it.
//
// An unresolvable name is parsed as a bare model ref rather than dropped, so it
// becomes a candidate carrying no credential of its own and fails at call time.
// This test records that so nobody assumes a stale reference is harmless: the
// safety comes from validateVisionModelSelection on write and from
// handleDeleteModel clearing the field, not from resolution.
func TestDanglingVisionReferenceIsNotSilentlyDropped(t *testing.T) {
	cfg := visionConfig("deleted-model")
	imageCandidates := resolveModelCandidates(
		cfg,
		cfg.Agents.Defaults.Provider,
		cfg.Agents.Defaults.ImageModel,
		cfg.Agents.Defaults.ImageModelFallbacks,
	)
	if len(imageCandidates) != 1 {
		t.Fatalf("len(imageCandidates) = %d, want 1 unusable candidate", len(imageCandidates))
	}
	if imageCandidates[0].Model != "deleted-model" {
		t.Fatalf("candidate = %q, want the dangling name", imageCandidates[0].Model)
	}
	// It resolves to no model_list entry, which is what makes it unusable.
	if mc := lookupModelConfigByRef(cfg, "deleted-model", ""); mc != nil {
		t.Fatal("a deleted model should not resolve to a config entry")
	}
}

// The vision model may be the same entry as the default. Nothing needs to be
// duplicated and both roles resolve to the same candidate.
func TestVisionModelMayEqualTheDefault(t *testing.T) {
	cfg := visionConfig("text-model")
	imageCandidates := resolveModelCandidates(
		cfg,
		cfg.Agents.Defaults.Provider,
		cfg.Agents.Defaults.ImageModel,
		cfg.Agents.Defaults.ImageModelFallbacks,
	)
	if len(imageCandidates) != 1 || imageCandidates[0].Model != "text-model" {
		t.Fatalf("image candidates = %+v, want the default model", imageCandidates)
	}
	if len(cfg.ModelList) != 2 {
		t.Fatalf("model_list has %d entries, want no duplication", len(cfg.ModelList))
	}
}
