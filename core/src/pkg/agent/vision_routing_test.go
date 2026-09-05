package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
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

// --- what image_model_fallbacks actually does -------------------------------
//
// The field is ACTIVE, not dormant. `instance.go` builds the agent's
// ImageCandidates from [image_model, ...image_model_fallbacks], and
// routeMediaTurn *replaces* exec.activeCandidates with that list — it does not
// append to, or fall through to, the normal model_fallbacks chain.
//
// So an image turn and a text turn walk two different chains:
//
//	text turn   -> [model_name,  ...model_fallbacks]
//	image turn  -> [image_model, ...image_model_fallbacks]   (when image_model is set)
//	image turn  -> [model_name,  ...model_fallbacks]          (when it is not)
//
// The Dashboard configures only the first and third of those. With image_model
// set and image_model_fallbacks left empty — which is every user who configures
// vision through the UI — an image turn has exactly one candidate and no
// fallback behind it. That is existing behaviour and is not changed here; these
// tests exist so it is written down rather than assumed.

func chainConfig(imageModel string, imageFallbacks []string) *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				ModelName:           "text-model",
				ModelFallbacks:      []string{"text-fallback"},
				ImageModel:          imageModel,
				ImageModelFallbacks: imageFallbacks,
			},
		},
		ModelList: []*config.ModelConfig{
			{ModelName: "text-model", Model: "openai/text-model"},
			{ModelName: "text-fallback", Model: "openai/text-fallback"},
			{ModelName: "vision-model", Model: "openai/vision-model"},
			{ModelName: "vision-fallback", Model: "openai/vision-fallback"},
		},
	}
}

func candidateModels(candidates []providers.FallbackCandidate) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, candidate.Model)
	}
	return out
}

func imageChain(cfg *config.Config) []providers.FallbackCandidate {
	return resolveModelCandidates(
		cfg,
		cfg.Agents.Defaults.Provider,
		cfg.Agents.Defaults.ImageModel,
		cfg.Agents.Defaults.ImageModelFallbacks,
	)
}

func textChain(cfg *config.Config) []providers.FallbackCandidate {
	return resolveModelCandidates(
		cfg,
		cfg.Agents.Defaults.Provider,
		cfg.Agents.Defaults.GetModelName(),
		cfg.Agents.Defaults.ModelFallbacks,
	)
}

// image_model_fallbacks is consumed, so it is active rather than dormant.
func TestImageModelFallbacksAreActive(t *testing.T) {
	cfg := chainConfig("vision-model", []string{"vision-fallback"})

	got := candidateModels(imageChain(cfg))
	want := []string{"vision-model", "vision-fallback"}
	if len(got) != len(want) {
		t.Fatalf("image chain = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("image chain = %v, want %v", got, want)
		}
	}
}

// The two chains are separate. An image turn does not fall through to the
// Fallback Models the Dashboard configures.
func TestImageChainDoesNotIncludeTheTextFallbacks(t *testing.T) {
	cfg := chainConfig("vision-model", []string{"vision-fallback"})

	for _, model := range candidateModels(imageChain(cfg)) {
		if model == "text-fallback" {
			t.Fatal("the text fallback chain leaked into the image chain")
		}
	}
	for _, model := range candidateModels(textChain(cfg)) {
		if model == "vision-model" || model == "vision-fallback" {
			t.Fatal("the image chain leaked into the text chain")
		}
	}
}

// The state every user who configures vision through the Dashboard is in: a
// vision model and no image fallbacks. The image turn then has exactly one
// candidate, and the Fallback Models list does not stand behind it.
func TestVisionWithoutImageFallbacksHasASingleCandidate(t *testing.T) {
	cfg := chainConfig("vision-model", nil)

	got := candidateModels(imageChain(cfg))
	if len(got) != 1 || got[0] != "vision-model" {
		t.Fatalf("image chain = %v, want exactly [vision-model]", got)
	}

	// The text chain is unaffected and still carries its fallback.
	text := candidateModels(textChain(cfg))
	if len(text) != 2 || text[0] != "text-model" || text[1] != "text-fallback" {
		t.Fatalf("text chain = %v, want [text-model text-fallback]", text)
	}
}

// With neither field set the image chain is empty, routeMediaTurn finds nothing
// to switch to, and an image turn keeps the normal chain — fallbacks included.
// This is the state of every install that has not configured vision.
func TestWithoutAVisionModelImageTurnsKeepTheTextChain(t *testing.T) {
	cfg := chainConfig("", nil)

	if got := imageChain(cfg); len(got) != 0 {
		t.Fatalf("image chain = %v, want empty when no vision model is set", candidateModels(got))
	}
	text := candidateModels(textChain(cfg))
	if len(text) != 2 || text[0] != "text-model" || text[1] != "text-fallback" {
		t.Fatalf("text chain = %v, want the default plus its fallback", text)
	}
}

// An edge case worth recording rather than fixing: image_model_fallbacks set
// while image_model is empty still yields an image chain, because
// resolveModelCandidates skips the empty primary and keeps the rest. Image
// turns would then route to the first image fallback with no primary ever
// having been chosen.
//
// It is pre-existing and unreachable from the Dashboard, which writes
// image_model and never image_model_fallbacks, so the only way into this shape
// is hand-editing config.yaml. It is left exactly as it is: changing it would
// alter routing for anyone who has already hand-configured that way, which is
// out of scope here. Note that instance.go only pre-creates providers for the
// image chain when image_model is non-empty, so this shape is also the one
// place the two halves of the wiring disagree.
func TestImageFallbacksWithoutAPrimaryStillFormAChain(t *testing.T) {
	cfg := chainConfig("", []string{"vision-fallback"})

	got := candidateModels(imageChain(cfg))
	if len(got) != 1 || got[0] != "vision-fallback" {
		t.Fatalf("image chain = %v, want [vision-fallback]", got)
	}
}

// The wiring, end to end through the registry: the agent an inbound message is
// served by carries both chains, built from the two config fields.
func TestAgentCarriesBothChainsSeparately(t *testing.T) {
	cfg := chainConfig("vision-model", []string{"vision-fallback"})
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.MaxTokens = 4096

	al := NewAgentLoop(cfg, bus.NewMessageBus(), &unexpectedTextAttachmentProvider{})
	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected default agent")
	}

	if got := candidateModels(agent.Candidates); len(got) != 2 ||
		got[0] != "text-model" || got[1] != "text-fallback" {
		t.Fatalf("agent.Candidates = %v", got)
	}
	if got := candidateModels(agent.ImageCandidates); len(got) != 2 ||
		got[0] != "vision-model" || got[1] != "vision-fallback" {
		t.Fatalf("agent.ImageCandidates = %v", got)
	}
}
