package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsGeminiOpenAICompatibleBase(t *testing.T) {
	cases := map[string]bool{
		"https://generativelanguage.googleapis.com/v1beta":         false,
		"https://generativelanguage.googleapis.com/v1beta/openai":  true,
		"https://generativelanguage.googleapis.com/v1beta/openai/": true,
		"https://proxy.example.com/gemini/openai":                  true,
		"https://proxy.example.com/openai-compat":                  false,
		"": false,
	}
	for base, want := range cases {
		if got := isGeminiOpenAICompatibleBase(base); got != want {
			t.Errorf("isGeminiOpenAICompatibleBase(%q) = %v, want %v", base, got, want)
		}
	}
}

// The native Gemini listing authenticates with X-Goog-Api-Key, never Bearer,
// and returns fully qualified "models/<id>" resource names.
func TestFetchGeminiNativeModels(t *testing.T) {
	var gotAuthorization, gotGoogKey, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		gotGoogKey = r.Header.Get("X-Goog-Api-Key")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[
			{"name":"models/gemini-3-flash-preview"},
			{"name":"models/gemini-3.1-pro-preview"},
			{"name":""}
		]}`))
	}))
	defer srv.Close()

	models, err := fetchGeminiNativeModels(t.Context(), srv.URL+"/v1beta/models", "test-key")
	if err != nil {
		t.Fatalf("fetchGeminiNativeModels() error = %v", err)
	}
	if gotPath != "/v1beta/models" {
		t.Errorf("request path = %q, want %q", gotPath, "/v1beta/models")
	}
	if gotGoogKey != "test-key" {
		t.Errorf("X-Goog-Api-Key = %q, want %q", gotGoogKey, "test-key")
	}
	if gotAuthorization != "" {
		t.Errorf("Authorization = %q, want it unset for the native Gemini API", gotAuthorization)
	}
	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}
	if models[0].ID != "gemini-3-flash-preview" || models[1].ID != "gemini-3.1-pro-preview" {
		t.Errorf("models = %+v, want the models/ prefix stripped", models)
	}
}

func TestFetchUpstreamModelsGeminiRoutesByBase(t *testing.T) {
	var nativeHit, openAIHit bool

	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nativeHit = r.Header.Get("X-Goog-Api-Key") == "k" && r.URL.Path == "/v1beta/models"
		_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-3-flash-preview"}]}`))
	}))
	defer native.Close()

	openAI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openAIHit = r.Header.Get("Authorization") == "Bearer k" && r.URL.Path == "/v1beta/openai/models"
		_, _ = w.Write([]byte(`{"data":[{"id":"gemini-3-flash-preview"}]}`))
	}))
	defer openAI.Close()

	if _, err := fetchUpstreamModels(t.Context(), "gemini", native.URL+"/v1beta", "k"); err != nil {
		t.Fatalf("native fetch error = %v", err)
	}
	if !nativeHit {
		t.Error("native Gemini base did not produce an X-Goog-Api-Key request to /v1beta/models")
	}

	// Trailing slash included on purpose: base-relative path handling must not regress.
	if _, err := fetchUpstreamModels(t.Context(), "gemini", openAI.URL+"/v1beta/openai/", "k"); err != nil {
		t.Fatalf("openai-compatible fetch error = %v", err)
	}
	if !openAIHit {
		t.Error("Gemini OpenAI-compatible base did not produce a Bearer request to /v1beta/openai/models")
	}
}

func TestDescribeModelFetchHTTPErrorClassification(t *testing.T) {
	cases := []struct {
		status int
		want   string
	}{
		{http.StatusUnauthorized, "authentication failed"},
		{http.StatusForbidden, "authentication failed"},
		{http.StatusTooManyRequests, "rate limited"},
		{http.StatusNotFound, "no model listing endpoint"},
		{http.StatusBadGateway, "is unavailable"},
		{http.StatusServiceUnavailable, "is unavailable"},
		{http.StatusTeapot, "returned HTTP 418"},
	}
	for _, tc := range cases {
		err := describeModelFetchHTTPError("provider", tc.status)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("status %d error = %v, want it to mention %q", tc.status, err, tc.want)
		}
	}
}

// A failed discovery must never echo the API key back to the caller.
func TestModelFetchErrorsDoNotLeakAPIKey(t *testing.T) {
	const secret = "sk-do-not-leak-this-value"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	for _, provider := range []string{"gemini", "openai"} {
		_, err := fetchUpstreamModels(t.Context(), provider, srv.URL+"/v1", secret)
		if err == nil {
			t.Fatalf("%s: expected an error", provider)
		}
		if strings.Contains(err.Error(), secret) {
			t.Errorf("%s: error message contains the API key: %v", provider, err)
		}
	}
}
