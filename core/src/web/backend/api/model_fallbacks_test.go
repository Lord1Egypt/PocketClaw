package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// fallbackTestEnv sets up a config with a default model and two usable
// alternatives, which is the shape every fallback test needs.
func fallbackTestEnv(t *testing.T) (string, *http.ServeMux) {
	t.Helper()
	configPath, cleanup := setupOAuthTestEnv(t)
	t.Cleanup(cleanup)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "DeepSeek", Provider: "openai", Model: "deepseek-v4",
			APIKeys: config.SimpleSecureStrings("sk-primary")},
		{ModelName: "Gemini", Provider: "gemini", Model: "gemini-2.5-flash",
			APIKeys: config.SimpleSecureStrings("sk-gemini")},
		{ModelName: "Claude", Provider: "anthropic", Model: "claude-sonnet",
			APIKeys: config.SimpleSecureStrings("sk-claude")},
	}
	cfg.Agents.Defaults.ModelName = "DeepSeek"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return configPath, mux
}

func postFallbacks(t *testing.T, mux *http.ServeMux, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models/fallbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	return rec
}

func savedFallbacks(t *testing.T, configPath string) []string {
	t.Helper()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	return cfg.Agents.Defaults.ModelFallbacks
}

// Zero fallbacks is valid and is what every existing installation already has.
func TestSetModelFallbacks_EmptyListIsValid(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	rec := postFallbacks(t, mux, `{"fallbacks": []}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := savedFallbacks(t, configPath); len(got) != 0 {
		t.Fatalf("expected no fallbacks, got %v", got)
	}
}

func TestSetModelFallbacks_SingleFallbackPersists(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	rec := postFallbacks(t, mux, `{"fallbacks": ["Gemini"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := savedFallbacks(t, configPath)
	if len(got) != 1 || got[0] != "Gemini" {
		t.Fatalf("expected [Gemini], got %v", got)
	}
}

// Order is the whole feature: the chain tries candidates in the order given.
func TestSetModelFallbacks_PreservesDeclaredOrder(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	rec := postFallbacks(t, mux, `{"fallbacks": ["Claude", "Gemini"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := savedFallbacks(t, configPath)
	if len(got) != 2 || got[0] != "Claude" || got[1] != "Gemini" {
		t.Fatalf("order must be preserved exactly, got %v", got)
	}
}

// Reordering is a save of the same names in a different order.
func TestSetModelFallbacks_ReorderPersists(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	if rec := postFallbacks(t, mux, `{"fallbacks": ["Gemini", "Claude"]}`); rec.Code != http.StatusOK {
		t.Fatalf("initial save failed: %s", rec.Body.String())
	}
	if rec := postFallbacks(t, mux, `{"fallbacks": ["Claude", "Gemini"]}`); rec.Code != http.StatusOK {
		t.Fatalf("reorder failed: %s", rec.Body.String())
	}

	got := savedFallbacks(t, configPath)
	if len(got) != 2 || got[0] != "Claude" || got[1] != "Gemini" {
		t.Fatalf("reorder must persist, got %v", got)
	}
}

func TestSetModelFallbacks_RemovePersists(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	if rec := postFallbacks(t, mux, `{"fallbacks": ["Gemini", "Claude"]}`); rec.Code != http.StatusOK {
		t.Fatalf("initial save failed: %s", rec.Body.String())
	}
	if rec := postFallbacks(t, mux, `{"fallbacks": ["Gemini"]}`); rec.Code != http.StatusOK {
		t.Fatalf("removal failed: %s", rec.Body.String())
	}

	got := savedFallbacks(t, configPath)
	if len(got) != 1 || got[0] != "Gemini" {
		t.Fatalf("removal must persist, got %v", got)
	}
}

// A duplicate would make the chain try the same candidate twice, wasting the
// latency the feature exists to avoid.
func TestSetModelFallbacks_RejectsDuplicates(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	rec := postFallbacks(t, mux, `{"fallbacks": ["Gemini", "Gemini"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "more than once") {
		t.Fatalf("error should name the duplicate: %s", rec.Body.String())
	}
	if got := savedFallbacks(t, configPath); len(got) != 0 {
		t.Fatalf("a rejected save must not persist, got %v", got)
	}
}

// Self-reference would make the chain retry the candidate that just failed,
// which is the one thing a fallback must never do.
func TestSetModelFallbacks_RejectsSelfReference(t *testing.T) {
	_, mux := fallbackTestEnv(t)

	rec := postFallbacks(t, mux, `{"fallbacks": ["DeepSeek"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "already the default") {
		t.Fatalf("error should explain the self-reference: %s", rec.Body.String())
	}
}

func TestSetModelFallbacks_RejectsUnknownModel(t *testing.T) {
	_, mux := fallbackTestEnv(t)

	rec := postFallbacks(t, mux, `{"fallbacks": ["NotConfigured"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not in model_list") {
		t.Fatalf("error should say the entry is unknown: %s", rec.Body.String())
	}
}

// A fallback names a configured entry; it never carries credentials of its own,
// so the primary's key cannot reach it and its own key is what gets used.
func TestSetModelFallbacks_KeepsEachModelsOwnCredentials(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	if rec := postFallbacks(t, mux, `{"fallbacks": ["Gemini"]}`); rec.Code != http.StatusOK {
		t.Fatalf("save failed: %s", rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	// The stored fallback is a name, not an inline model definition.
	if len(cfg.Agents.Defaults.ModelFallbacks) != 1 ||
		cfg.Agents.Defaults.ModelFallbacks[0] != "Gemini" {
		t.Fatalf("fallback should be stored as a reference, got %v",
			cfg.Agents.Defaults.ModelFallbacks)
	}
	for _, m := range cfg.ModelList {
		switch m.ModelName {
		case "DeepSeek":
			if key := m.APIKey(); key != "sk-primary" {
				t.Fatalf("primary key changed: %q", key)
			}
		case "Gemini":
			// The decisive assertion: the fallback keeps its own credential.
			// Copying the primary's key into a fallback would send one
			// provider's secret to another provider's endpoint.
			if key := m.APIKey(); key != "sk-gemini" {
				t.Fatalf("fallback must keep its own key, got %q", key)
			}
			if m.Provider != "gemini" {
				t.Fatalf("fallback must keep its own provider, got %q", m.Provider)
			}
		}
	}
}

// A config written before this feature existed has no fallback list at all and
// must keep working untouched.
func TestModelsList_ExistingConfigWithoutFallbacksStaysValid(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Agents.Defaults.ModelFallbacks = nil
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/models", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Emitted even when empty, so the UI can tell "none configured" from
	// "this build does not support fallbacks".
	raw, present := body["model_fallbacks"]
	if !present {
		t.Fatal("model_fallbacks must always be present in the list response")
	}
	list, ok := raw.([]any)
	if !ok || len(list) != 0 {
		t.Fatalf("expected an empty list, got %#v", raw)
	}
}

func TestModelsList_ReportsConfiguredFallbacksInOrder(t *testing.T) {
	_, mux := fallbackTestEnv(t)

	if rec := postFallbacks(t, mux, `{"fallbacks": ["Claude", "Gemini"]}`); rec.Code != http.StatusOK {
		t.Fatalf("save failed: %s", rec.Body.String())
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/models", nil))

	var body struct {
		ModelFallbacks []string `json:"model_fallbacks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.ModelFallbacks) != 2 ||
		body.ModelFallbacks[0] != "Claude" || body.ModelFallbacks[1] != "Gemini" {
		t.Fatalf("list must report the saved order, got %v", body.ModelFallbacks)
	}
}
