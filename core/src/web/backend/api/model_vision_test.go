package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func postVision(t *testing.T, mux *http.ServeMux, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models/vision", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	return rec
}

func savedVisionModel(t *testing.T, configPath string) string {
	t.Helper()
	return loadConfigForTest(t, configPath).Agents.Defaults.ImageModel
}

func TestSetVisionModel_PointsAtAnExistingEntry(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	rec := postVision(t, mux, `{"model_name": "gemini-primary"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	if got := savedVisionModel(t, configPath); got != "gemini-primary" {
		t.Fatalf("image_model = %q, want %q", got, "gemini-primary")
	}
	// The default is untouched: choosing a vision model must not silently
	// change which model answers ordinary turns.
	if got := loadConfigForTest(t, configPath).Agents.Defaults.GetModelName(); got != "opencode-primary" {
		t.Fatalf("default_model = %q, want it unchanged", got)
	}
}

// Unset is a real state, not a failure: image turns fall back to the default,
// which is what every install does today.
func TestSetVisionModel_ClearsWithAnEmptyName(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	if rec := postVision(t, mux, `{"model_name": "gemini-primary"}`); rec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", rec.Code)
	}
	rec := postVision(t, mux, `{"model_name": ""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	if got := savedVisionModel(t, configPath); got != "" {
		t.Fatalf("image_model = %q, want it cleared", got)
	}
}

func TestSetVisionModel_RejectsAnUnknownModel(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	rec := postVision(t, mux, `{"model_name": "never-configured"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body %q", rec.Code, rec.Body.String())
	}
	if got := savedVisionModel(t, configPath); got != "" {
		t.Fatalf("a rejected reference was written: %q", got)
	}
}

func TestSetVisionModel_RejectsAVirtualModel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)

	cfg := loadConfigForTest(t, configPath)
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "real", Provider: "openai", Model: "m",
			APIKeys: config.SimpleSecureStrings("sk-a", "sk-b")},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Multi-key entries expand into virtual models, which have no upstream of
	// their own and so cannot hold a routing role.
	reloaded := loadConfigForTest(t, configPath)
	var virtualName string
	for _, m := range reloaded.ModelList {
		if m.IsVirtual() {
			virtualName = m.ModelName
			break
		}
	}
	if virtualName == "" {
		t.Skip("this build does not expand multi-key entries into virtual models")
	}
	if rec := postVision(t, mux, `{"model_name": "`+virtualName+`"}`); rec.Code == http.StatusOK {
		t.Fatalf("a virtual model was accepted as the vision model")
	}
}

func TestListModels_ReportsTheConfiguredVisionModel(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	if rec := postVision(t, mux, `{"model_name": "gemini-primary"}`); rec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", rec.Code)
	}
	_ = configPath

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/models", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var out struct {
		ImageModel   string `json:"image_model"`
		DefaultModel string `json:"default_model"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ImageModel != "gemini-primary" {
		t.Fatalf("image_model = %q", out.ImageModel)
	}
	if out.DefaultModel != "opencode-primary" {
		t.Fatalf("default_model = %q", out.DefaultModel)
	}
}

func TestListModels_ReportsNoVisionModelWhenUnset(t *testing.T) {
	_, mux := materializeTestEnv(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/models", nil))
	var out struct {
		ImageModel string `json:"image_model"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ImageModel != "" {
		t.Fatalf("image_model = %q, want empty for a fresh install", out.ImageModel)
	}
}

func TestMaterializeModel_CreatesEntryAndAppliesVisionRole(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	rec := postMaterialize(t, mux, `{
		"source_index": 1,
		"model": "gemini-3.7-pro-vision",
		"role": "vision"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}

	body := decodeMaterialize(t, rec)
	if created, _ := body["created"].(bool); !created {
		t.Fatal("expected the model to be created")
	}
	name, _ := body["model_name"].(string)
	if got, _ := body["image_model"].(string); got != name {
		t.Fatalf("response image_model = %q, want %q", got, name)
	}

	cfg := loadConfigForTest(t, configPath)
	if cfg.Agents.Defaults.ImageModel != name {
		t.Fatalf("image_model = %q, want %q", cfg.Agents.Defaults.ImageModel, name)
	}
	// The default is untouched.
	if cfg.Agents.Defaults.GetModelName() != "opencode-primary" {
		t.Fatalf("default changed to %q", cfg.Agents.Defaults.GetModelName())
	}

	for _, m := range cfg.ModelList {
		if m.ModelName == name {
			if m.APIKey() != "sk-gemini" {
				t.Fatalf("credential not inherited: %q", m.APIKey())
			}
			if m.APIBase != "https://generativelanguage.example/v1beta" {
				t.Fatalf("api_base = %q", m.APIBase)
			}
			return
		}
	}
	t.Fatalf("entry %q not written", name)
}

func TestMaterializeModel_VisionReusesAnExistingEntry(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	before := len(loadConfigForTest(t, configPath).ModelList)

	rec := postMaterialize(t, mux, `{
		"source_index": 1,
		"model": "gemini-3.7-flash",
		"role": "vision"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	body := decodeMaterialize(t, rec)
	if created, _ := body["created"].(bool); created {
		t.Fatal("an already-configured model must be reused, not duplicated")
	}
	if name, _ := body["model_name"].(string); name != "gemini-primary" {
		t.Fatalf("model_name = %q", name)
	}
	if after := len(loadConfigForTest(t, configPath).ModelList); after != before {
		t.Fatalf("model_list grew from %d to %d", before, after)
	}
	if got := savedVisionModel(t, configPath); got != "gemini-primary" {
		t.Fatalf("image_model = %q", got)
	}
}

// The vision model may be the same entry as the default if the user says so.
// It is harmless — the same model simply answers both kinds of turn — and it
// must not require a duplicate entry.
func TestSetVisionModel_MayEqualTheDefaultWithoutDuplicating(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	before := len(loadConfigForTest(t, configPath).ModelList)

	if rec := postVision(t, mux, `{"model_name": "opencode-primary"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	cfg := loadConfigForTest(t, configPath)
	if cfg.Agents.Defaults.ImageModel != "opencode-primary" {
		t.Fatalf("image_model = %q", cfg.Agents.Defaults.ImageModel)
	}
	if cfg.Agents.Defaults.GetModelName() != "opencode-primary" {
		t.Fatalf("default_model = %q", cfg.Agents.Defaults.GetModelName())
	}
	if len(cfg.ModelList) != before {
		t.Fatalf("model_list grew to %d", len(cfg.ModelList))
	}
}

func TestMaterializeModel_VisionWritesNothingWhenRejected(t *testing.T) {
	configPath, mux := materializeTestEnv(t)
	before := len(loadConfigForTest(t, configPath).ModelList)

	// Source index 2 is the shipped keyless placeholder, so there is no
	// credential to inherit and nothing may be written.
	rec := postMaterialize(t, mux, `{
		"source_index": 2, "model": "some-vision-model", "role": "vision"
	}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	cfg := loadConfigForTest(t, configPath)
	if len(cfg.ModelList) != before {
		t.Fatalf("model_list grew to %d on a rejected request", len(cfg.ModelList))
	}
	if cfg.Agents.Defaults.ImageModel != "" {
		t.Fatalf("image_model was set to %q despite rejection", cfg.Agents.Defaults.ImageModel)
	}
}

// Two providers exposing the same id stay distinct under the vision role too.
func TestMaterializeModel_VisionIdentityIsProviderScoped(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)

	cfg := loadConfigForTest(t, configPath)
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "a-vision", Provider: "openai", Model: "qwen-vl",
			APIBase: "https://provider-a.example/v1",
			APIKeys: config.SimpleSecureStrings("sk-a")},
		{ModelName: "b-seed", Provider: "openai", Model: "seed",
			APIBase: "https://provider-b.example/v1",
			APIKeys: config.SimpleSecureStrings("sk-b")},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := postMaterialize(t, mux, `{
		"source_index": 1, "model": "qwen-vl", "role": "vision"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rec.Code, rec.Body.String())
	}
	name, _ := decodeMaterialize(t, rec)["model_name"].(string)
	if name == "a-vision" {
		t.Fatal("provider B's model collapsed onto provider A's entry")
	}
	saved := loadConfigForTest(t, configPath)
	if saved.Agents.Defaults.ImageModel != name {
		t.Fatalf("image_model = %q, want %q", saved.Agents.Defaults.ImageModel, name)
	}
	for _, m := range saved.ModelList {
		if m.ModelName == name && m.APIKey() != "sk-b" {
			t.Fatalf("wrong credential inherited: %q", m.APIKey())
		}
	}
}

// Setting the vision model must not disturb the fallback chain.
func TestSetVisionModel_LeavesTheFallbackChainAlone(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	if rec := postFallbacks(t, mux, `{"fallbacks": ["gemini-primary"]}`); rec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d %q", rec.Code, rec.Body.String())
	}
	if rec := postVision(t, mux, `{"model_name": "gemini-primary"}`); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	cfg := loadConfigForTest(t, configPath)
	if len(cfg.Agents.Defaults.ModelFallbacks) != 1 ||
		cfg.Agents.Defaults.ModelFallbacks[0] != "gemini-primary" {
		t.Fatalf("fallbacks = %v, want them unchanged", cfg.Agents.Defaults.ModelFallbacks)
	}
}

// Deleting the entry the vision model points at clears the reference, so image
// turns return to the default instead of routing at a name that resolves to no
// credential.
func TestDeleteModel_ClearsADanglingVisionReference(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	if rec := postVision(t, mux, `{"model_name": "gemini-primary"}`); rec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", rec.Code)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/models/1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body %q", rec.Code, rec.Body.String())
	}

	cfg := loadConfigForTest(t, configPath)
	if cfg.Agents.Defaults.ImageModel != "" {
		t.Fatalf("image_model = %q, want it cleared with the deleted entry",
			cfg.Agents.Defaults.ImageModel)
	}
	// The default was a different entry and is untouched.
	if cfg.Agents.Defaults.GetModelName() != "opencode-primary" {
		t.Fatalf("default_model = %q", cfg.Agents.Defaults.GetModelName())
	}
}

// Deleting an unrelated model leaves the vision reference alone.
func TestDeleteModel_KeepsAnUnrelatedVisionReference(t *testing.T) {
	configPath, mux := materializeTestEnv(t)

	if rec := postVision(t, mux, `{"model_name": "gemini-primary"}`); rec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", rec.Code)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/models/2", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d", rec.Code)
	}
	if got := savedVisionModel(t, configPath); got != "gemini-primary" {
		t.Fatalf("image_model = %q, want it preserved", got)
	}
}
