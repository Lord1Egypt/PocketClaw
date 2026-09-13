package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Removing a model has to remove every reference to it, not just the one the
// default-model check happens to cover.
//
// The fallback chain is a list of model_list names. A deleted name left in it
// is a candidate the router will try and cannot resolve, which is the "never
// silently continue with a deleted model" rule broken by the removal path
// itself.

func deleteModelAt(t *testing.T, mux *http.ServeMux, index string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/models/"+index, nil)
	mux.ServeHTTP(rec, req)
	return rec
}

func TestDeleteModelRemovesItFromTheFallbackChain(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	if rec := postFallbacks(t, mux, `{"fallbacks":["Gemini","Claude"]}`); rec.Code != http.StatusOK {
		t.Fatalf("seeding fallbacks: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Index 1 is Gemini, the first fallback.
	if rec := deleteModelAt(t, mux, "1"); rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	got := savedFallbacks(t, configPath)
	if len(got) != 1 || got[0] != "Claude" {
		t.Fatalf("fallbacks = %v, want [Claude]: a deleted model must not stay "+
			"in the chain as an unresolvable candidate", got)
	}
}

// Order is meaning in a fallback chain, and removing one entry must not
// reshuffle the rest.
func TestDeleteModelPreservesRemainingFallbackOrder(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = append(cfg.ModelList, &config.ModelConfig{
		ModelName: "Kimi", Provider: "openai", Model: "kimi-k3",
		APIKeys: config.SimpleSecureStrings("sk-kimi"),
	})
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if rec := postFallbacks(t, mux, `{"fallbacks":["Gemini","Claude","Kimi"]}`); rec.Code != http.StatusOK {
		t.Fatalf("seeding fallbacks: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Index 2 is Claude, the middle of the chain.
	if rec := deleteModelAt(t, mux, "2"); rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	got := savedFallbacks(t, configPath)
	if len(got) != 2 || got[0] != "Gemini" || got[1] != "Kimi" {
		t.Fatalf("fallbacks = %v, want [Gemini Kimi]", got)
	}
}

// A chain that becomes empty is no chain, not an empty one that later reads as
// "configured with nothing".
func TestDeleteModelEmptiesTheChainCompletely(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	if rec := postFallbacks(t, mux, `{"fallbacks":["Gemini"]}`); rec.Code != http.StatusOK {
		t.Fatalf("seeding fallbacks: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := deleteModelAt(t, mux, "1"); rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if got := savedFallbacks(t, configPath); len(got) != 0 {
		t.Fatalf("fallbacks = %v, want none", got)
	}
}

// The pre-existing behaviour has to keep working: the default is cleared, and
// an unrelated model's references are untouched.
func TestDeleteModelClearsTheDefaultAndLeavesOthersAlone(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	if rec := postFallbacks(t, mux, `{"fallbacks":["Gemini","Claude"]}`); rec.Code != http.StatusOK {
		t.Fatalf("seeding fallbacks: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	// Index 0 is DeepSeek, the default model.
	if rec := deleteModelAt(t, mux, "0"); rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := cfg.Agents.Defaults.GetModelName(); got != "" {
		t.Fatalf("default model = %q, want empty", got)
	}
	got := cfg.Agents.Defaults.ModelFallbacks
	if len(got) != 2 || got[0] != "Gemini" || got[1] != "Claude" {
		t.Fatalf("fallbacks = %v, want [Gemini Claude] untouched", got)
	}
}

// The list-clearing semantics the delete path depends on, held against the
// helper that now performs it for every reference site.
func TestPurgeModelReferencesListSemantics(t *testing.T) {
	cases := []struct {
		name       string
		references []string
		remove     string
		want       []string
	}{
		{"absent name changes nothing", []string{"a", "b"}, "c", []string{"a", "b"}},
		{"removes every occurrence", []string{"a", "b", "a"}, "a", []string{"b"}},
		{"whitespace around a stored name still matches", []string{" a ", "b"}, "a", []string{"b"}},
		{"empty list stays empty", nil, "a", nil},
		{"emptied list becomes nil", []string{"a"}, "a", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Agents.Defaults.ModelFallbacks = tc.references
			purgeModelReferences(cfg, newModelReferenceSet(tc.remove))

			got := cfg.Agents.Defaults.ModelFallbacks
			if len(got) != len(tc.want) {
				t.Fatalf("model_fallbacks = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("model_fallbacks = %v, want %v", got, tc.want)
				}
			}
			if len(tc.want) == 0 && got != nil {
				t.Fatal("an emptied chain must be nil so the field is omitted from the saved config")
			}
		})
	}
}

// A single-model delete has to reach every reference site, not only the default
// model and the two default fallback chains.
func TestDeleteModelClearsTheLightModelAndAgentReferences(t *testing.T) {
	configPath, mux := fallbackTestEnv(t)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Agents.Defaults.Routing = &config.RoutingConfig{
		Enabled: true, LightModel: "Gemini", Threshold: 0.3,
	}
	cfg.Agents.Defaults.ImageModel = "Gemini"
	cfg.Agents.List = []config.AgentConfig{{
		ID:    "main",
		Model: &config.AgentModelConfig{Primary: "Gemini", Fallbacks: []string{"Claude"}},
	}}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Index 1 is Gemini.
	if rec := deleteModelAt(t, mux, "1"); rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := cfg.Agents.Defaults.Routing.LightModel; got != "" {
		t.Errorf("routing.light_model = %q, want cleared", got)
	}
	if got := cfg.Agents.Defaults.ImageModel; got != "" {
		t.Errorf("image_model = %q, want cleared", got)
	}
	if len(cfg.Agents.List) != 1 || cfg.Agents.List[0].Model == nil {
		t.Fatalf("agent list did not round-trip: %+v", cfg.Agents.List)
	}
	if got := cfg.Agents.List[0].Model.Primary; got != "" {
		t.Errorf("agent model primary = %q, want cleared", got)
	}
	if got := cfg.Agents.List[0].Model.Fallbacks; len(got) != 1 || got[0] != "Claude" {
		t.Errorf("agent model fallbacks = %v, want [Claude] untouched", got)
	}
}
