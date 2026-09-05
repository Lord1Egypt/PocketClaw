package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// capturedRequest is the normalized shape of one provider call: everything the
// invariant is stated over.
type capturedRequest struct {
	Model    string            `json:"model"`
	Messages []capturedMessage `json:"messages"`
	Tools    []string          `json:"tools"`
	Options  map[string]any    `json:"options"`
}

type capturedMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Media   []string `json:"media"`
}

func (c capturedRequest) normalized() string {
	encoded, err := json.Marshal(c)
	if err != nil {
		return "unmarshalable:" + err.Error()
	}
	return string(encoded)
}

func (c capturedRequest) imageCount() int {
	total := 0
	for _, msg := range c.Messages {
		total += len(msg.Media)
	}
	return total
}

// capturingProvider records exactly what it was handed, deep-copying so a later
// mutation of the caller's slices cannot rewrite history.
type capturingProvider struct {
	name string
	fail error

	mu       sync.Mutex
	requests []capturedRequest
}

func (p *capturingProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	captured := capturedRequest{
		Model:    model,
		Messages: make([]capturedMessage, 0, len(messages)),
		Tools:    make([]string, 0, len(tools)),
		Options:  make(map[string]any, len(options)),
	}
	for _, msg := range messages {
		captured.Messages = append(captured.Messages, capturedMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Media:   append([]string(nil), msg.Media...),
		})
	}
	for _, tool := range tools {
		captured.Tools = append(captured.Tools, tool.Function.Name)
	}
	for key, value := range options {
		captured.Options[key] = value
	}

	p.mu.Lock()
	p.requests = append(p.requests, captured)
	p.mu.Unlock()

	if p.fail != nil {
		return nil, p.fail
	}
	return &providers.LLMResponse{Content: "ok from " + p.name, FinishReason: "stop"}, nil
}

func (p *capturingProvider) GetDefaultModel() string { return "captured-model" }

func (p *capturingProvider) captured() []capturedRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]capturedRequest(nil), p.requests...)
}

// consumingProvider models an adapter that rewrites the slice it was handed,
// stripping media as it "consumes" it. A candidate after such an adapter must
// still receive the complete turn.
type consumingProvider struct {
	capturingProvider
}

func (p *consumingProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	resp, err := p.capturingProvider.Chat(ctx, messages, tools, model, options)
	for i := range messages {
		messages[i].Media = nil
		messages[i].Content = ""
	}
	return resp, err
}

func imageMessages(caption string, count int) []providers.Message {
	media := make([]string, 0, count)
	for i := range count {
		media = append(media, fmt.Sprintf("data:image/png;base64,IMAGE%02d", i))
	}
	return []providers.Message{{Role: "user", Content: caption, Media: media}}
}

func modelListEntry(name, provider, model, apiBase string) *config.ModelConfig {
	return &config.ModelConfig{
		ModelName: name,
		Provider:  provider,
		Model:     model,
		APIBase:   apiBase,
		Enabled:   true,
		APIKeys:   config.SecureStrings{config.NewSecureString("key-for-" + name)},
	}
}

type mediaTurnResult struct {
	primary   *capturingProvider
	err       error
	candidate []providers.FallbackCandidate
}

// runMediaTurn drives a real turn through Pipeline.CallLLM with the given
// candidate list, so the assertions are about the shipping request path rather
// than a reimplementation of it.
func runMediaTurn(
	t *testing.T,
	candidates []providers.FallbackCandidate,
	primary providers.LLMProvider,
	candidateProviders map[string]providers.LLMProvider,
	images int,
	caption string,
) (capturedRequest, mediaTurnResult) {
	t.Helper()
	return runMediaTurnInWorkspace(
		t, t.TempDir(), candidates, primary, candidateProviders, images, caption)
}

// runMediaTurnInWorkspace pins the workspace so two runs can be compared: the
// workspace path appears in the system prompt.
func runMediaTurnInWorkspace(
	t *testing.T,
	workspace string,
	candidates []providers.FallbackCandidate,
	primary providers.LLMProvider,
	candidateProviders map[string]providers.LLMProvider,
	images int,
	caption string,
) (capturedRequest, mediaTurnResult) {
	t.Helper()

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "primary-entry",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				MaxLLMRetries:     1,
			},
		},
	}

	recorder, _ := primary.(*capturingProvider)
	al := NewAgentLoop(cfg, bus.NewMessageBus(), primary)
	t.Cleanup(al.Close)

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatal("expected a default agent")
	}
	if candidateProviders != nil {
		agent.CandidateProviders = candidateProviders
	}

	pipeline := NewPipeline(al)
	pipeline.Fallback = providers.NewFallbackChain(
		providers.NewCooldownTracker(), providers.NewRateLimiterRegistry())

	ts := newTurnState(agent, makeTestProcessOpts("media-session"), turnEventScope{
		turnID:  "turn-media",
		context: newTurnContext(nil, nil, nil),
	})
	exec, err := pipeline.SetupTurn(context.Background(), ts)
	if err != nil {
		t.Fatalf("SetupTurn: %v", err)
	}
	exec.messages = append(exec.messages, imageMessages(caption, images)...)
	exec.activeCandidates = candidates
	exec.activeModel = candidates[0].Model
	exec.activeProvider = primary

	_, callErr := pipeline.CallLLM(context.Background(), context.Background(), ts, exec, 1)

	result := mediaTurnResult{primary: recorder, err: callErr, candidate: candidates}
	if recorder == nil {
		return capturedRequest{}, result
	}
	captured := recorder.captured()
	if len(captured) == 0 {
		return capturedRequest{}, result
	}
	return captured[0], result
}

func primaryCandidate() providers.FallbackCandidate {
	return providers.FallbackCandidate{
		Provider: "openai", Model: "primary-model", IdentityKey: "model_name:primary-entry",
	}
}

func fallbackCandidates() []providers.FallbackCandidate {
	return []providers.FallbackCandidate{
		{Provider: "openai", Model: "fallback-b", IdentityKey: "model_name:b"},
		{Provider: "openai", Model: "fallback-c", IdentityKey: "model_name:c"},
	}
}

// 1. One image, no fallbacks: the primary receives it.
func TestSingleImageReachesPrimary(t *testing.T) {
	primary := &capturingProvider{name: "primary"}
	got, _ := runMediaTurn(t,
		[]providers.FallbackCandidate{primaryCandidate()}, primary, nil, 1, "one image")
	if got.imageCount() != 1 {
		t.Fatalf("primary received %d images, want 1", got.imageCount())
	}
}

// 2. Seven images, no fallbacks: all seven arrive, in order.
func TestSevenImagesReachPrimaryInOrder(t *testing.T) {
	primary := &capturingProvider{name: "primary"}
	got, _ := runMediaTurn(t,
		[]providers.FallbackCandidate{primaryCandidate()}, primary, nil, 7, "seven images")
	assertSevenImagesInOrder(t, got)
}

func assertSevenImagesInOrder(t *testing.T, got capturedRequest) {
	t.Helper()
	if got.imageCount() != 7 {
		t.Fatalf("received %d images, want 7", got.imageCount())
	}
	var media []string
	for _, msg := range got.Messages {
		media = append(media, msg.Media...)
	}
	for i, ref := range media {
		want := fmt.Sprintf("data:image/png;base64,IMAGE%02d", i)
		if ref != want {
			t.Fatalf("image %d = %q, want %q (order not preserved)", i, ref, want)
		}
	}
}

// 3 + REGRESSION. Configuring fallback candidates must not change the request
// sent to the first candidate. This is the reported failure: seven images
// succeeded with the primary alone and were rejected as soon as fallbacks were
// configured.
func TestConfiguringFallbacksDoesNotChangeThePrimaryRequest(t *testing.T) {
	workspace := t.TempDir()

	withoutFallbacks := &capturingProvider{name: "primary"}
	alone, _ := runMediaTurnInWorkspace(t, workspace,
		[]providers.FallbackCandidate{primaryCandidate()},
		withoutFallbacks, nil, 7, "seven images with a caption")

	withFallbacks := &capturingProvider{name: "primary"}
	candidates := append([]providers.FallbackCandidate{primaryCandidate()}, fallbackCandidates()...)
	chained, _ := runMediaTurnInWorkspace(t, workspace, candidates,
		withFallbacks, nil, 7, "seven images with a caption")

	if alone.normalized() != chained.normalized() {
		t.Fatalf("the primary request changed when fallbacks were configured:\n"+
			"  without fallbacks: %s\n  with fallbacks:    %s",
			alone.normalized(), chained.normalized())
	}
	assertSevenImagesInOrder(t, chained)

	// And the primary still succeeded, so no fallback was consulted.
	if len(withFallbacks.captured()) != 1 {
		t.Fatalf("primary was called %d times, want exactly one",
			len(withFallbacks.captured()))
	}
}

// A fallback entry naming the same protocol and model id as the primary must
// not take ownership of the primary's provider. That collision is what sent the
// primary's request to the fallback's endpoint with the fallback's credentials.
//
// Both halves of the identity are exercised here: the map is built by the same
// code the agent uses, and the lookup is the one the request path performs.
func TestFallbackSharingTheModelIDDoesNotHijackThePrimaryProvider(t *testing.T) {
	cfg := &config.Config{
		ModelList: []*config.ModelConfig{
			modelListEntry("PrimaryKey", "deepseek", "deepseek-vision", "https://primary.example/v1"),
			modelListEntry("BackupKey", "deepseek", "deepseek-vision", "https://backup.example/v1"),
		},
	}
	workspace := t.TempDir()

	registered := make(map[string]providers.LLMProvider)
	populateCandidateProvidersFromNames(
		cfg, workspace, []string{"PrimaryKey", "BackupKey"}, registered)

	primary, ok := candidateFromModelConfig("openai", cfg.ModelList[0])
	if !ok {
		t.Fatal("candidateFromModelConfig rejected the primary entry")
	}
	backup, ok := candidateFromModelConfig("openai", cfg.ModelList[1])
	if !ok {
		t.Fatal("candidateFromModelConfig rejected the backup entry")
	}

	agent := &AgentInstance{CandidateProviders: registered}
	sentinel := &capturingProvider{name: "inherited"}

	primaryProvider, err := providerForFallbackCandidate(
		agent, sentinel, []providers.FallbackCandidate{primary, backup}, primary)
	if err != nil {
		t.Fatalf("resolving the primary provider: %v", err)
	}
	backupProvider, err := providerForFallbackCandidate(
		agent, sentinel, []providers.FallbackCandidate{primary, backup}, backup)
	if err != nil {
		t.Fatalf("resolving the backup provider: %v", err)
	}

	if primaryProvider == backupProvider {
		t.Fatal("the primary and the fallback resolved to the same provider; the " +
			"primary's request would go to the fallback's endpoint with its credentials")
	}
	if primaryProvider != registered[primary.StableKey()] {
		t.Fatal("the primary did not resolve to the provider built from its own model_list entry")
	}
	if backupProvider != registered[backup.StableKey()] {
		t.Fatal("the fallback did not resolve to the provider built from its own model_list entry")
	}
}

// The registration key itself must be the stable identity. Two model_list
// entries for one model are distinct candidates — usually a second API key —
// and provider/model cannot tell them apart.
func TestCandidateProviderKeyDistinguishesEntriesForTheSameModel(t *testing.T) {
	first := modelListEntry("PrimaryKey", "deepseek", "deepseek-vision", "https://primary.example/v1")
	second := modelListEntry("BackupKey", "deepseek", "deepseek-vision", "https://backup.example/v1")

	if candidateProviderKey(first) == candidateProviderKey(second) {
		t.Fatal("two model_list entries for the same model collapsed onto one provider key")
	}

	primaryCand, ok := candidateFromModelConfig("openai", first)
	if !ok {
		t.Fatal("candidateFromModelConfig rejected a valid entry")
	}
	if primaryCand.StableKey() != candidateProviderKey(first) {
		t.Fatalf("candidate identity %q does not match its provider registration key %q; "+
			"the candidate would resolve to another entry's provider",
			primaryCand.StableKey(), candidateProviderKey(first))
	}
}

// 4 + 5. Each candidate in turn receives the complete turn.
func TestEveryCandidateReceivesAllImages(t *testing.T) {
	failure := errors.New("service unavailable (503)")
	first := &capturingProvider{name: "first", fail: failure}
	second := &capturingProvider{name: "second", fail: failure}
	third := &capturingProvider{name: "third"}

	candidates := []providers.FallbackCandidate{
		{Provider: "openai", Model: "m-a", IdentityKey: "model_name:a"},
		{Provider: "openai", Model: "m-b", IdentityKey: "model_name:b"},
		{Provider: "openai", Model: "m-c", IdentityKey: "model_name:c"},
	}
	candidateProviders := map[string]providers.LLMProvider{
		candidates[0].StableKey(): first,
		candidates[1].StableKey(): second,
		candidates[2].StableKey(): third,
	}

	_, _ = runMediaTurn(t, candidates, first, candidateProviders, 7, "seven images")

	for name, provider := range map[string]*capturingProvider{
		"first": first, "second": second, "third": third,
	} {
		captured := provider.captured()
		if len(captured) == 0 {
			t.Fatalf("candidate %s was never attempted", name)
		}
		if got := captured[0].imageCount(); got != 7 {
			t.Fatalf("candidate %s received %d images, want 7", name, got)
		}
		assertSevenImagesInOrder(t, captured[0])
	}
}

// 6 + 7 + 8. An adapter that consumes the slice it was handed must not empty
// the turn for the candidate that follows it, and the original turn must be
// unchanged after the chain has run.
func TestConsumingAdapterCannotEmptyTheNextCandidate(t *testing.T) {
	consumer := &consumingProvider{capturingProvider: capturingProvider{
		name: "consumer", fail: errors.New("service unavailable (503)"),
	}}
	next := &capturingProvider{name: "next"}

	candidates := []providers.FallbackCandidate{
		{Provider: "openai", Model: "m-consumer", IdentityKey: "model_name:consumer"},
		{Provider: "openai", Model: "m-next", IdentityKey: "model_name:next"},
	}
	candidateProviders := map[string]providers.LLMProvider{
		candidates[0].StableKey(): consumer,
		candidates[1].StableKey(): next,
	}

	_, _ = runMediaTurn(t, candidates, &consumer.capturingProvider, candidateProviders,
		7, "seven images")

	captured := next.captured()
	if len(captured) == 0 {
		t.Fatal("the second candidate was never attempted")
	}
	if got := captured[0].imageCount(); got != 7 {
		t.Fatalf("the second candidate received %d images after the first consumed "+
			"the turn, want 7", got)
	}
	if captured[0].Messages[len(captured[0].Messages)-1].Content == "" {
		t.Fatal("the second candidate received an emptied caption")
	}
}

// 9. A candidate that genuinely cannot take images fails cleanly and the next
// candidate still gets the whole turn.
func TestVisionUnsupportedCandidateDoesNotCorruptTheNextAttempt(t *testing.T) {
	unsupported := &capturingProvider{
		name: "text-only",
		fail: errors.New("no endpoints found that support image input"),
	}
	capable := &capturingProvider{name: "vision"}

	candidates := []providers.FallbackCandidate{
		{Provider: "openai", Model: "m-text", IdentityKey: "model_name:text"},
		{Provider: "openai", Model: "m-vision", IdentityKey: "model_name:vision"},
	}
	candidateProviders := map[string]providers.LLMProvider{
		candidates[0].StableKey(): unsupported,
		candidates[1].StableKey(): capable,
	}

	_, _ = runMediaTurn(t, candidates, unsupported, candidateProviders, 7, "seven images")

	captured := capable.captured()
	if len(captured) == 0 {
		t.Fatal("the vision-capable candidate was never attempted")
	}
	assertSevenImagesInOrder(t, captured[0])
}

// 10. A caption travels with its images.
func TestCaptionIsPreservedAlongsideSevenImages(t *testing.T) {
	primary := &capturingProvider{name: "primary"}
	const caption = "please describe all of these"
	got, _ := runMediaTurn(t,
		[]providers.FallbackCandidate{primaryCandidate()}, primary, nil, 7, caption)

	assertSevenImagesInOrder(t, got)
	found := false
	for _, msg := range got.Messages {
		if strings.Contains(msg.Content, caption) {
			found = true
		}
	}
	if !found {
		t.Fatalf("the caption did not reach the model: %s", got.normalized())
	}
}
