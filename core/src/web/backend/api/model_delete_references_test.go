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

func TestRemoveModelReference(t *testing.T) {
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
			got := removeModelReference(tc.references, tc.remove)
			if len(got) != len(tc.want) {
				t.Fatalf("removeModelReference() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("removeModelReference() = %v, want %v", got, tc.want)
				}
			}
		})
	}
}
