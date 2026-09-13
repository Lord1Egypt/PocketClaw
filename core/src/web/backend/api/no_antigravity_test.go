package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// PC-DEF-023. Google Antigravity is deliberately not shipped in v0.2.0.
//
// The concern was never secret disclosure — an installed-app OAuth client
// cannot keep a secret, which is why RFC 8252 does not treat one as
// confidential. It was ownership: the client ID and secret belonged to another
// project, so a third party could revoke them and break the provider for every
// PocketClaw user, for a reason PocketClaw could neither predict nor fix.
//
// These pin the removal at the surfaces a user or caller can actually reach.

func TestAntigravityIsNotAnOAuthProvider(t *testing.T) {
	for _, raw := range []string{
		"antigravity", "google-antigravity", "Antigravity",
		"  GOOGLE-ANTIGRAVITY  ", "google_antigravity",
	} {
		t.Run(raw, func(t *testing.T) {
			provider, err := normalizeOAuthProvider(raw)
			if err == nil {
				t.Fatalf("normalizeOAuthProvider(%q) = %q, want the unsupported-provider error",
					raw, provider)
			}
			if !strings.Contains(err.Error(), "unsupported provider") {
				t.Fatalf("error = %v, want the normal unsupported-provider behaviour", err)
			}
		})
	}
}

func TestOAuthProviderSurfaceIsOpenAIAndAnthropicOnly(t *testing.T) {
	want := []string{oauthProviderOpenAI, oauthProviderAnthropic}
	if len(oauthProviderOrder) != len(want) {
		t.Fatalf("oauthProviderOrder = %v, want %v", oauthProviderOrder, want)
	}
	for i, provider := range want {
		if oauthProviderOrder[i] != provider {
			t.Fatalf("oauthProviderOrder[%d] = %q, want %q", i, oauthProviderOrder[i], provider)
		}
	}
	for _, table := range []string{"methods", "labels"} {
		var keys []string
		if table == "methods" {
			for k := range oauthProviderMethods {
				keys = append(keys, k)
			}
		} else {
			for k := range oauthProviderLabels {
				keys = append(keys, k)
			}
		}
		if len(keys) != len(want) {
			t.Fatalf("oauthProvider%s keys = %v, want %v", table, keys, want)
		}
		for _, k := range keys {
			if strings.Contains(strings.ToLower(k), "antigravity") {
				t.Fatalf("oauthProvider%s still carries %q", table, k)
			}
		}
	}
}

// The product-facing provider catalogue is what the dashboard renders and what
// a caller can create a model against. A removed provider must be absent from
// it, not merely unreachable from the OAuth flow.
//
// NormalizeProvider is deliberately not used here: it is a string normaliser
// with an alias table and echoes an unknown id straight back, so it says
// nothing about whether a provider is registered.
func TestAntigravityIsNotInTheProviderCatalogue(t *testing.T) {
	for _, option := range providers.ModelProviderOptions() {
		if strings.Contains(strings.ToLower(option.ID), "antigravity") {
			t.Fatalf("provider catalogue still lists %q", option.ID)
		}
		for _, alias := range option.Aliases {
			if strings.Contains(strings.ToLower(alias), "antigravity") {
				t.Fatalf("provider %q still aliases %q", option.ID, alias)
			}
		}
	}
}

// Gemini is a different provider with its own API-key auth and its own
// endpoint. Removing Antigravity must not have taken it out.
func TestGeminiSurvivesTheAntigravityRemoval(t *testing.T) {
	if normalized := providers.NormalizeProvider("gemini"); normalized != "gemini" {
		t.Fatalf("NormalizeProvider(\"gemini\") = %q, want \"gemini\"", normalized)
	}
	if normalized := providers.NormalizeProvider("google"); normalized != "gemini" {
		t.Fatalf("the google alias must still resolve to gemini, got %q", normalized)
	}
}

// The third-party credential must be gone from source, not merely unreferenced.
// Encoded is what shipped, so both forms are checked.
func TestThirdPartyOAuthCredentialIsAbsentFromSource(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	markers := []string{
		"R09DU1BYLU",                   // encoded client-secret prefix
		"GOCSPX-",                      // decoded client-secret prefix
		"MTA3MTAwNjA2MDU5MS10bWhzc2lu", // encoded client-id prefix
		".apps.googleusercontent.com",
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", "build", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".ts", ".tsx", ".json", ".dart", ".kt":
		default:
			return nil
		}
		if strings.HasSuffix(path, "no_antigravity_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, marker := range markers {
			if strings.Contains(string(data), marker) {
				t.Errorf("%s still carries third-party OAuth client material", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
