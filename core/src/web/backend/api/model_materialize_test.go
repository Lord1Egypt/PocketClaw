package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// materializeTestEnv gives two configured providers plus one unconfigured
// template entry — the shape the shipped default config actually has, where
// most of model_list is keyless placeholders the user never touched.
func materializeTestEnv(t *testing.T) (string, *http.ServeMux) {
	t.Helper()
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "opencode-primary", Provider: "openai", Model: "deepseek-v4-flash-free",
			APIBase: "https://opencode.example/v1",
			APIKeys: config.SimpleSecureStrings("sk-opencode"),
		},
		{
			ModelName: "gemini-primary", Provider: "gemini", Model: "gemini-3.7-flash",
			APIBase: "https://generativelanguage.example/v1beta",
			APIKeys: config.SimpleSecureStrings("sk-gemini"),
		},
		// A shipped placeholder: no key, never configured by the user.
		{ModelName: "cerebras-llama-3.3-70b", Provider: "cerebras", Model: "llama-3.3-70b",
			APIBase: "https://api.cerebras.ai/v1"},
	}
	cfg.Agents.Defaults.ModelName = "opencode-primary"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return configPath, mux
}

func postMaterialize(t *testing.T, mux *http.ServeMux, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models/materialize", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeMaterialize(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v (body %q)", err, rec.Body.String())
	}
	return out
}

func loadConfigForTest(t *testing.T, configPath string) *config.Config {
	t.Helper()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	return cfg
}

func TestMaterializeModel_CreatesEntryAndAppliesDefault(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	rec := postMaterialize(t, mux, `{
		"source_index": 0,
		"model": "deepseek-v4-flash-vision-exp",
		"role": "default"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}

	body := decodeMaterialize(t, rec)
	if created, _ := body["created"].(bool); !created {
		t.Fatalf("expected the model to be created, got %v", body["created"])
	}
	name, _ := body["model_name"].(string)
	if name == "" {
		t.Fatal("response carried no model_name")
	}

	cfg := loadConfigForTest(t, configPath)
	if got := cfg.Agents.Defaults.GetModelName(); got != name {
		t.Fatalf("default model = %q, want %q", got, name)
	}

	var created *config.ModelConfig
	for _, m := range cfg.ModelList {
		if m.ModelName == name {
			created = m
		}
	}
	if created == nil {
		t.Fatalf("model %q was not written to model_list", name)
	}
	if created.Model != "deepseek-v4-flash-vision-exp" {
		t.Fatalf("model id = %q", created.Model)
	}
	// The point of materializing from a source index: the new entry inherits
	// the provider instance rather than being handed one by the browser.
	if created.APIBase != "https://opencode.example/v1" {
		t.Fatalf("api_base = %q, want the source provider's base", created.APIBase)
	}
	if created.APIKey() != "sk-opencode" {
		t.Fatalf("credential was not inherited from the source entry")
	}
	if !created.Enabled {
		t.Fatal("a materialized model should be enabled")
	}
}

func TestMaterializeModel_AppendsToFallbackChain(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	rec := postMaterialize(t, mux, `{
		"source_index": 1,
		"model": "gemini-3.7-pro",
		"role": "fallback"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	name, _ := decodeMaterialize(t, rec)["model_name"].(string)

	cfg := loadConfigForTest(t, configPath)
	if len(cfg.Agents.Defaults.ModelFallbacks) != 1 ||
		cfg.Agents.Defaults.ModelFallbacks[0] != name {
		t.Fatalf("fallbacks = %v, want [%q]", cfg.Agents.Defaults.ModelFallbacks, name)
	}
}

func TestMaterializeModel_ReusesAnExistingEntryRatherThanDuplicating(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	before := len(loadConfigForTest(t, configPath).ModelList)

	// The model the user already configured, offered again by discovery. It is
	// the current default, so it is asked for with no role — reuse is what is
	// under test here, not the role rules.
	rec := postMaterialize(t, mux, `{
		"source_index": 0,
		"model": "deepseek-v4-flash-free",
		"role": ""
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}

	body := decodeMaterialize(t, rec)
	if created, _ := body["created"].(bool); created {
		t.Fatal("an already-configured model must be reused, not duplicated")
	}
	if name, _ := body["model_name"].(string); name != "opencode-primary" {
		t.Fatalf("model_name = %q, want the existing entry", name)
	}
	if after := len(loadConfigForTest(t, configPath).ModelList); after != before {
		t.Fatalf("model_list grew from %d to %d", before, after)
	}
}

// Two configured providers can expose the same model id. Deduplicating on the
// id alone would route one provider's model through the other's credential.
func TestMaterializeModel_IdentityIsProviderScopedNotModelIDAlone(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)

	cfg := loadConfigForTest(t, configPath)
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "a-chat", Provider: "openai", Model: "deepseek-chat",
			APIBase: "https://provider-a.example/v1",
			APIKeys: config.SimpleSecureStrings("sk-a")},
		{ModelName: "b-other", Provider: "openai", Model: "something-else",
			APIBase: "https://provider-b.example/v1",
			APIKeys: config.SimpleSecureStrings("sk-b")},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// The same model id, but offered by provider B.
	rec := postMaterialize(t, mux, `{
		"source_index": 1,
		"model": "deepseek-chat",
		"role": ""
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	body := decodeMaterialize(t, rec)
	if created, _ := body["created"].(bool); !created {
		t.Fatal("the same id from a different provider instance must be a distinct entry")
	}
	name, _ := body["model_name"].(string)
	if name == "a-chat" {
		t.Fatal("provider B's model collapsed onto provider A's entry")
	}

	saved := loadConfigForTest(t, configPath)
	if len(saved.ModelList) != 3 {
		t.Fatalf("model_list has %d entries, want 3", len(saved.ModelList))
	}
	for _, m := range saved.ModelList {
		if m.ModelName == name && m.APIKey() != "sk-b" {
			t.Fatalf("new entry inherited the wrong credential: %q", m.APIKey())
		}
	}
}

func TestMaterializeModel_RefusesAnUnconfiguredSourceProvider(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	// Index 2 is the shipped keyless placeholder.
	rec := postMaterialize(t, mux, `{
		"source_index": 2,
		"model": "llama-3.3-70b",
		"role": "default"
	}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %q", rec.Code, rec.Body.String())
	}
	if got := len(loadConfigForTest(t, configPath).ModelList); got != 3 {
		t.Fatalf("model_list changed on a rejected request: %d entries", got)
	}
}

// The half-configured state this endpoint exists to prevent: if the role is
// rejected, the entry it would have referenced must not survive either.
func TestMaterializeModel_WritesNothingWhenTheRoleIsRejected(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	before := loadConfigForTest(t, configPath)
	beforeCount := len(before.ModelList)

	rec := postMaterialize(t, mux, `{
		"source_index": 0,
		"model": "deepseek-v4-flash-free",
		"role": "default"
	}`)
	// The source model is already the default, and a model cannot be its own
	// fallback — but as a default this is a no-op, so use the fallback path to
	// get a genuine rejection.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}

	rec = postMaterialize(t, mux, `{
		"source_index": 0,
		"model": "deepseek-v4-flash-free",
		"role": "fallback"
	}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a self-referencing fallback; body %q",
			rec.Code, rec.Body.String())
	}

	after := loadConfigForTest(t, configPath)
	if len(after.ModelList) != beforeCount {
		t.Fatalf("model_list grew to %d despite the rejected role", len(after.ModelList))
	}
	if len(after.Agents.Defaults.ModelFallbacks) != 0 {
		t.Fatalf("fallbacks = %v, want none", after.Agents.Defaults.ModelFallbacks)
	}
}

func TestMaterializeModel_CreatesNoDuplicateWhenTheRoleIsRejected(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	beforeCount := len(loadConfigForTest(t, configPath).ModelList)

	// A brand new model that would be created, then rejected as a fallback
	// because it is about to become the default at the same time. Making it
	// the default first is what sets up the self-reference.
	if rec := postMaterialize(t, mux, `{
		"source_index": 1, "model": "gemini-3.7-pro", "role": "default"
	}`); rec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d %q", rec.Code, rec.Body.String())
	}
	afterDefault := len(loadConfigForTest(t, configPath).ModelList)
	if afterDefault != beforeCount+1 {
		t.Fatalf("expected one new entry, got %d", afterDefault-beforeCount)
	}

	// Now the same model as a fallback: rejected, and no second entry.
	if rec := postMaterialize(t, mux, `{
		"source_index": 1, "model": "gemini-3.7-pro", "role": "fallback"
	}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := len(loadConfigForTest(t, configPath).ModelList); got != afterDefault {
		t.Fatalf("model_list grew to %d on a rejected role", got)
	}
}

// Roles fail closed: anything the switch does not name writes nothing at all,
// rather than creating the entry and quietly assigning no role.
func TestMaterializeModel_RejectsUnknownRole(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	for _, role := range []string{"embedding", "audio", "Default", "vision-model"} {
		rec := postMaterialize(t, mux,
			`{"source_index": 0, "model": "x", "role": "`+role+`"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("role %q: status = %d, want 400", role, rec.Code)
		}
		cfg := loadConfigForTest(t, configPath)
		if got := len(cfg.ModelList); got != 3 {
			t.Fatalf("role %q: model_list changed to %d entries", role, got)
		}
		if cfg.Agents.Defaults.ImageModel != "" {
			t.Fatalf("role %q: image_model was set", role)
		}
	}
}

func TestMaterializeModel_RejectsAnOutOfRangeSource(t *testing.T) {
	_, mux := materializeTestEnv(t)
	if rec := postMaterialize(t, mux, `{"source_index": 99, "model": "x"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if rec := postMaterialize(t, mux, `{"source_index": -1, "model": "x"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestMaterializeModel_RequiresAModel(t *testing.T) {
	_, mux := materializeTestEnv(t)
	if rec := postMaterialize(t, mux, `{"source_index": 0, "model": "  "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// A provider-prefixed id is stored as a bare id with the provider carried in
// its own field, which is what the rest of the config assumes.
func TestMaterializeModel_StripsAProviderPrefixFromTheModelID(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	rec := postMaterialize(t, mux, `{
		"source_index": 1, "model": "gemini/gemini-3.7-pro", "role": ""
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	name, _ := decodeMaterialize(t, rec)["model_name"].(string)
	for _, m := range loadConfigForTest(t, configPath).ModelList {
		if m.ModelName == name {
			if m.Model != "gemini-3.7-pro" {
				t.Fatalf("model = %q, want the bare id", m.Model)
			}
			return
		}
	}
	t.Fatalf("entry %q not found", name)
}

func TestMaterializeModel_DoesNotCollideWithAnExistingAlias(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)

	cfg := loadConfigForTest(t, configPath)
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "shared-id", Provider: "openai", Model: "other",
			APIBase: "https://a.example/v1", APIKeys: config.SimpleSecureStrings("sk-a")},
		{ModelName: "b", Provider: "gemini", Model: "seed",
			APIBase: "https://b.example/v1", APIKeys: config.SimpleSecureStrings("sk-b")},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// The natural alias for this id is already taken by an unrelated entry.
	rec := postMaterialize(t, mux, `{
		"source_index": 1, "model": "shared-id", "role": ""
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	name, _ := decodeMaterialize(t, rec)["model_name"].(string)
	if name == "shared-id" {
		t.Fatal("the new entry shadowed an existing alias")
	}

	saved := loadConfigForTest(t, configPath)
	seen := map[string]int{}
	for _, m := range saved.ModelList {
		seen[m.ModelName]++
	}
	for alias, count := range seen {
		if count > 1 {
			t.Fatalf("alias %q appears %d times", alias, count)
		}
	}
}
