package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Provider management is a view over model_list. These tests hold the contract
// the owner specified: one credential rotation reaches every model of the
// provider, and deleting a provider leaves no orphan model and no reference
// naming a model that is gone.

// providerTestEnv seeds two models on one provider plus one on another, so every
// test can tell "applied to the provider" apart from "applied to everything".
func providerTestEnv(t *testing.T) (string, *http.ServeMux) {
	t.Helper()
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "opencode-flash", Provider: "opencode_go", Model: "deepseek-v4.1-flash",
			APIBase: "https://opencode.ai/zen/go/v1", Enabled: true,
			APIKeys: config.SimpleSecureStrings("sk-opencode-old")},
		{ModelName: "opencode-coder", Provider: "opencode_go", Model: "qwen3-coder",
			APIBase: "https://opencode.ai/zen/go/v1", Enabled: true,
			APIKeys: config.SimpleSecureStrings("sk-opencode-old")},
		{ModelName: "Gemini", Provider: "gemini", Model: "gemini-2.5-flash", Enabled: true,
			APIKeys: config.SimpleSecureStrings("sk-gemini")},
	}
	cfg.Agents.Defaults.ModelName = "opencode-flash"
	cfg.Agents.Defaults.ModelFallbacks = []string{"opencode-coder", "Gemini"}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return configPath, mux
}

func providerRequest(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeProvider(t *testing.T, rec *httptest.ResponseRecorder) providerResponse {
	t.Helper()
	var got providerResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding provider response: %v (body = %s)", err, rec.Body.String())
	}
	return got
}

// VIEW ---------------------------------------------------------------------

func TestListProvidersReportsEachConfiguredProviderOnce(t *testing.T) {
	_, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodGet, "/api/providers", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Providers []providerResponse `json:"providers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Providers) != 2 {
		t.Fatalf("providers = %d, want 2 (opencode, gemini)", len(body.Providers))
	}
	byKey := map[string]providerResponse{}
	for _, provider := range body.Providers {
		byKey[provider.Provider] = provider
	}
	opencode, ok := byKey["opencode_go"]
	if !ok {
		t.Fatalf("opencode missing from %v", byKey)
	}
	if opencode.ModelCount != 2 {
		t.Errorf("opencode model_count = %d, want 2", opencode.ModelCount)
	}
	if !opencode.HoldsDefaultModel {
		t.Error("opencode holds the default model and must say so before a delete is confirmed")
	}
}

// An "already configured" provider must expose its credential state without
// revealing the credential.
func TestGetProviderReportsSharedCredentialWithoutRevealingIt(t *testing.T) {
	_, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodGet, "/api/providers/opencode_go", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := decodeProvider(t, rec)
	if got.CredentialState != providerCredentialShared {
		t.Errorf("credential_state = %q, want %q", got.CredentialState, providerCredentialShared)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("sk-opencode-old")) {
		t.Error("provider response must not contain the raw API key")
	}
	if got.APIKeyMasked == "" {
		t.Error("a configured provider must report that a credential exists")
	}
	if got.APIBase != "https://opencode.ai/zen/go/v1" || got.APIBaseMixed {
		t.Errorf("api_base = %q mixed=%t, want the shared base", got.APIBase, got.APIBaseMixed)
	}
}

// Reporting one of several keys as "the provider key" would let a rotation
// update one model while its siblings kept an old credential.
func TestGetProviderReportsMixedCredentialsRatherThanPickingOne(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	cfg := loadConfigForTest(t, configPath)
	cfg.ModelList[1].APIKeys = config.SimpleSecureStrings("sk-opencode-different")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	got := decodeProvider(t, providerRequest(t, mux, http.MethodGet, "/api/providers/opencode_go", ""))
	if got.CredentialState != providerCredentialMixed {
		t.Errorf("credential_state = %q, want %q", got.CredentialState, providerCredentialMixed)
	}
	if got.APIKeyMasked != "" {
		t.Error("a mixed provider must not present one of its keys as the provider key")
	}
}

func TestGetProviderIsNotFoundForAnUnconfiguredProvider(t *testing.T) {
	_, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodGet, "/api/providers/anthropic", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestProviderLookupIsCaseAndAliasInsensitive(t *testing.T) {
	_, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodGet, "/api/providers/OpenCode-Go", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := decodeProvider(t, rec); got.Provider != "opencode_go" {
		t.Errorf("provider = %q, want the canonical key", got.Provider)
	}
}

// UPDATE / CREDENTIAL ROTATION --------------------------------------------

// This is the owner's contract: one rotation, every model of the provider.
func TestUpdateProviderRotatesTheKeyForEverySiblingModel(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodPut, "/api/providers/opencode_go",
		`{"api_key":"sk-opencode-new"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg := loadConfigForTest(t, configPath)
	for _, index := range []int{0, 1} {
		if got := cfg.ModelList[index].APIKey(); got != "sk-opencode-new" {
			t.Errorf("model_list[%d] (%s) key = %q, want the rotated key: a sibling left on "+
				"the old credential is the defect this rotation exists to prevent",
				index, cfg.ModelList[index].ModelName, got)
		}
	}
	if got := cfg.ModelList[2].APIKey(); got != "sk-gemini" {
		t.Errorf("another provider's key = %q, want it untouched", got)
	}
}

// A multi-key entry that kept its other keys would go on failing over to the
// credential the owner just rotated away from.
func TestUpdateProviderReplacesTheWholeKeyListNotJustTheFirstKey(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	cfg := loadConfigForTest(t, configPath)
	cfg.ModelList[0].APIKeys = config.SimpleSecureStrings("sk-first", "sk-second")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if rec := providerRequest(t, mux, http.MethodPut, "/api/providers/opencode_go",
		`{"api_key":"sk-only"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg = loadConfigForTest(t, configPath)
	if got := cfg.ModelList[0].APIKeys.Values(); len(got) != 1 || got[0] != "sk-only" {
		t.Fatalf("api_keys = %v, want exactly [sk-only]", got)
	}
}

// Rotating a credential must not rewrite the endpoint as a side effect.
func TestUpdateProviderLeavesUnsentFieldsAlone(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	if rec := providerRequest(t, mux, http.MethodPut, "/api/providers/opencode_go",
		`{"api_key":"sk-opencode-new"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg := loadConfigForTest(t, configPath)
	if got := cfg.ModelList[0].APIBase; got != "https://opencode.ai/zen/go/v1" {
		t.Errorf("api_base = %q, want it unchanged by a credential-only update", got)
	}
}

func TestUpdateProviderAppliesASentAPIBaseToEveryModel(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	if rec := providerRequest(t, mux, http.MethodPut, "/api/providers/opencode_go",
		`{"api_base":"https://opencode.ai/zen/go/v2"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg := loadConfigForTest(t, configPath)
	for _, index := range []int{0, 1} {
		if got := cfg.ModelList[index].APIBase; got != "https://opencode.ai/zen/go/v2" {
			t.Errorf("model_list[%d] api_base = %q, want the new base", index, got)
		}
	}
	if got := cfg.ModelList[0].APIKey(); got != "sk-opencode-old" {
		t.Errorf("key = %q, want an endpoint-only update to leave the credential alone", got)
	}
}

// A blank field in a submitted form must not silently revoke the provider's
// ability to answer.
func TestUpdateProviderWithAnEmptyKeyDoesNotClearTheStoredCredential(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodPut, "/api/providers/opencode_go", `{"api_key":"   "}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a request that changes nothing", rec.Code)
	}

	cfg := loadConfigForTest(t, configPath)
	if got := cfg.ModelList[0].APIKey(); got != "sk-opencode-old" {
		t.Fatalf("key = %q, want the stored credential preserved", got)
	}
}

func TestUpdateProviderIsNotFoundForAnUnconfiguredProvider(t *testing.T) {
	_, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodPut, "/api/providers/anthropic", `{"api_key":"sk-x"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// DELETE ------------------------------------------------------------------

func TestDeleteProviderRemovesEveryModelThatBelongedToIt(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodDelete, "/api/providers/opencode_go", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg := loadConfigForTest(t, configPath)
	if len(cfg.ModelList) != 1 {
		t.Fatalf("model_list = %d entries, want 1: an orphan model whose provider is gone "+
			"has no endpoint and no credential", len(cfg.ModelList))
	}
	if cfg.ModelList[0].ModelName != "Gemini" {
		t.Fatalf("surviving model = %q, want Gemini", cfg.ModelList[0].ModelName)
	}
}

func TestDeleteProviderClearsTheDefaultModelReference(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	if rec := providerRequest(t, mux, http.MethodDelete, "/api/providers/opencode_go", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg := loadConfigForTest(t, configPath)
	if got := cfg.Agents.Defaults.ModelName; got != "" {
		t.Fatalf("default model = %q, want cleared: chat must not retain a dead selection", got)
	}
}

func TestDeleteProviderPurgesItsModelsFromTheFallbackChain(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	if rec := providerRequest(t, mux, http.MethodDelete, "/api/providers/opencode_go", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg := loadConfigForTest(t, configPath)
	got := cfg.Agents.Defaults.ModelFallbacks
	if len(got) != 1 || got[0] != "Gemini" {
		t.Fatalf("fallbacks = %v, want [Gemini]: a removed name left in the chain is a "+
			"candidate the router tries and cannot resolve", got)
	}
}

// Every reference site, not only the two the single-model delete path covered.
func TestDeleteProviderClearsTheLightModelAndAgentReferences(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	cfg := loadConfigForTest(t, configPath)
	cfg.Agents.Defaults.Routing = &config.RoutingConfig{
		Enabled: true, LightModel: "opencode-coder", Threshold: 0.3,
	}
	cfg.Agents.Defaults.ImageModel = "opencode-flash"
	cfg.Agents.Defaults.ImageModelFallbacks = []string{"opencode-coder", "Gemini"}
	cfg.Agents.List = []config.AgentConfig{{
		ID: "main",
		Model: &config.AgentModelConfig{
			Primary:   "opencode-flash",
			Fallbacks: []string{"opencode-coder", "Gemini"},
		},
		Subagents: &config.SubagentsConfig{
			Model: &config.AgentModelConfig{Primary: "opencode-coder"},
		},
	}}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if rec := providerRequest(t, mux, http.MethodDelete, "/api/providers/opencode_go", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg = loadConfigForTest(t, configPath)
	if got := cfg.Agents.Defaults.Routing.LightModel; got != "" {
		t.Errorf("routing.light_model = %q, want cleared", got)
	}
	if got := cfg.Agents.Defaults.ImageModel; got != "" {
		t.Errorf("image_model = %q, want cleared", got)
	}
	if got := cfg.Agents.Defaults.ImageModelFallbacks; len(got) != 1 || got[0] != "Gemini" {
		t.Errorf("image_model_fallbacks = %v, want [Gemini]", got)
	}
	if len(cfg.Agents.List) != 1 || cfg.Agents.List[0].Model == nil {
		t.Fatalf("agent list did not round-trip: %+v", cfg.Agents.List)
	}
	agent := cfg.Agents.List[0]
	if got := agent.Model.Primary; got != "" {
		t.Errorf("agent model primary = %q, want cleared", got)
	}
	if got := agent.Model.Fallbacks; len(got) != 1 || got[0] != "Gemini" {
		t.Errorf("agent model fallbacks = %v, want [Gemini]", got)
	}
	if agent.Subagents == nil || agent.Subagents.Model == nil {
		t.Fatalf("subagent model did not round-trip: %+v", agent.Subagents)
	}
	if got := agent.Subagents.Model.Primary; got != "" {
		t.Errorf("subagent model primary = %q, want cleared", got)
	}
}

// The console confirms the delete reconciled rather than assuming it.
func TestDeleteProviderReportsWhatItRemoved(t *testing.T) {
	_, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodDelete, "/api/providers/opencode_go", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		ModelsRemoved   []string `json:"models_removed"`
		ClearedSites    []string `json:"cleared_sites"`
		ModelsRemaining int      `json:"models_remaining"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.ModelsRemoved) != 2 {
		t.Errorf("models_removed = %v, want both opencode models", body.ModelsRemoved)
	}
	if body.ModelsRemaining != 1 {
		t.Errorf("models_remaining = %d, want 1", body.ModelsRemaining)
	}
	if len(body.ClearedSites) == 0 {
		t.Error("cleared_sites must name the references the delete reconciled")
	}
}

func TestDeleteProviderIsNotFoundForAnUnconfiguredProvider(t *testing.T) {
	_, mux := providerTestEnv(t)

	rec := providerRequest(t, mux, http.MethodDelete, "/api/providers/anthropic", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// RECREATE ----------------------------------------------------------------

// After a delete the provider must be addable again cleanly, with no stale
// credential surviving from the deleted configuration.
func TestProviderCanBeRecreatedCleanlyAfterDeletion(t *testing.T) {
	configPath, mux := providerTestEnv(t)

	if rec := providerRequest(t, mux, http.MethodDelete, "/api/providers/opencode_go", ""); rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := providerRequest(t, mux, http.MethodGet, "/api/providers/opencode_go", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("after delete status = %d, want 404", rec.Code)
	}

	addBody := `{"model_name":"opencode-flash","provider":"opencode_go",` +
		`"model":"deepseek-v4.1-flash","api_base":"https://opencode.ai/zen/go/v1",` +
		`"api_key":"sk-opencode-fresh","enabled":true}`
	if rec := providerRequest(t, mux, http.MethodPost, "/api/models", addBody); rec.Code != http.StatusOK {
		t.Fatalf("re-add status = %d, body = %s", rec.Code, rec.Body.String())
	}

	got := decodeProvider(t, providerRequest(t, mux, http.MethodGet, "/api/providers/opencode_go", ""))
	if got.ModelCount != 1 {
		t.Fatalf("model_count = %d, want 1 after a clean recreate", got.ModelCount)
	}
	if got.CredentialState != providerCredentialShared {
		t.Errorf("credential_state = %q, want %q", got.CredentialState, providerCredentialShared)
	}

	cfg := loadConfigForTest(t, configPath)
	for _, mc := range cfg.ModelList {
		if mc.APIKey() == "sk-opencode-old" {
			t.Fatal("a deleted provider's credential must not survive into the recreated one")
		}
	}
}
