package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

const openCodeTestKey = "oc-test-key-not-a-real-secret"

func TestOpenCodeProvidersAreFetchable(t *testing.T) {
	for _, id := range []string{"opencode_zen", "opencode_go"} {
		if !providers.IsModelProviderFetchable(id) {
			t.Errorf("%s must support model discovery", id)
		}
	}
}

// Both official endpoints return an OpenAI-style list, so discovery goes
// through the shared OpenAI-compatible path: GET {base}/models with a bearer
// token, parsing data[].id.
func TestFetchOpenCodeModels(t *testing.T) {
	cases := []struct {
		provider string
		basePath string
		wantPath string
	}{
		{"opencode_zen", "/zen/v1", "/zen/v1/models"},
		{"opencode_go", "/zen/go/v1", "/zen/go/v1/models"},
		// A trailing slash must not produce a doubled separator.
		{"opencode_go", "/zen/go/v1/", "/zen/go/v1/models"},
	}

	for _, tc := range cases {
		var gotPath, gotAuth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[
				{"id":"kimi-k3","object":"model","owned_by":"opencode"},
				{"id":"deepseek-v4-flash","object":"model","owned_by":"opencode"},
				{"id":"glm-5.3","object":"model","owned_by":"opencode"}
			]}`))
		}))

		models, err := fetchUpstreamModels(t.Context(), tc.provider, srv.URL+tc.basePath, openCodeTestKey)
		srv.Close()

		if err != nil {
			t.Errorf("%s: fetchUpstreamModels() error = %v", tc.provider, err)
			continue
		}
		if gotPath != tc.wantPath {
			t.Errorf("%s: path = %q, want %q", tc.provider, gotPath, tc.wantPath)
		}
		if gotAuth != "Bearer "+openCodeTestKey {
			t.Errorf("%s: Authorization = %q, want a bearer token", tc.provider, gotAuth)
		}
		if len(models) != 3 {
			t.Fatalf("%s: len(models) = %d, want 3", tc.provider, len(models))
		}
		if models[0].ID != "kimi-k3" || models[2].ID != "glm-5.3" {
			t.Errorf("%s: parsed models = %+v, want bare ids from data[].id", tc.provider, models)
		}
		if models[0].OwnedBy != "opencode" {
			t.Errorf("%s: owned_by = %q, want %q", tc.provider, models[0].OwnedBy, "opencode")
		}
	}
}

// Discovery must reach the official host paths when the catalog default is used.
func TestOpenCodeDefaultFetchURLs(t *testing.T) {
	cases := map[string]string{
		"opencode_zen": "https://opencode.ai/zen/v1/models",
		"opencode_go":  "https://opencode.ai/zen/go/v1/models",
	}
	for id, want := range cases {
		base := providers.DefaultAPIBaseForProtocol(id)
		got := strings.TrimRight(base, "/") + "/models"
		if got != want {
			t.Errorf("%s discovery URL = %q, want %q", id, got, want)
		}
	}
}

func TestOpenCodeFetchErrorsDoNotLeakAPIKey(t *testing.T) {
	for _, status := range []int{
		http.StatusUnauthorized, http.StatusForbidden,
		http.StatusTooManyRequests, http.StatusNotFound, http.StatusBadGateway,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))

		_, err := fetchUpstreamModels(t.Context(), "opencode_zen", srv.URL+"/zen/v1", openCodeTestKey)
		srv.Close()

		if err == nil {
			t.Errorf("status %d: expected an error", status)
			continue
		}
		if strings.Contains(err.Error(), openCodeTestKey) {
			t.Errorf("status %d: error leaks the API key: %v", status, err)
		}
	}
}
