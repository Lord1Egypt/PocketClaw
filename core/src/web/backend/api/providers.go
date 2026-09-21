package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Provider management is a view over model_list, not a second store.
//
// The configuration schema has no provider record: `model_list` is a flat list
// of model entries, each carrying its own provider name, endpoint and
// credential. A "provider" is therefore the set of entries sharing a canonical
// provider key, and provider-scoped state is whatever that set agrees on.
//
// These handlers exist because the owner has to be able to manage the provider
// itself — rotate one credential, retire one integration — without editing
// every model that belongs to it one at a time. They deliberately do not
// introduce a provider object: a derived view cannot disagree with the models it
// is derived from, and a stored one could.
func (h *Handler) registerProviderRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/providers", h.handleListProviders)
	mux.HandleFunc("GET /api/providers/{provider}", h.handleGetProvider)
	mux.HandleFunc("PUT /api/providers/{provider}", h.handleUpdateProvider)
	mux.HandleFunc("DELETE /api/providers/{provider}", h.handleDeleteProvider)
}

// providerCredentialState describes what the models of one provider agree on.
//
// "mixed" is reported rather than resolved. Per-model overrides are legitimate,
// and silently presenting one of several keys as "the provider key" would let a
// rotation update one model while its siblings kept an old credential — the
// exact failure the owner asked to be made impossible.
const (
	providerCredentialUnset  = "unset"
	providerCredentialShared = "shared"
	providerCredentialMixed  = "mixed"
)

type providerModelSummary struct {
	Index     int    `json:"index"`
	ModelName string `json:"model_name"`
	Model     string `json:"model"`
	Enabled   bool   `json:"enabled"`
	IsDefault bool   `json:"is_default"`
	HasAPIKey bool   `json:"has_api_key"`
}

type providerResponse struct {
	// Provider is the canonical provider key. The console owns the display
	// name and icon; the backend does not keep a second copy of that catalog.
	Provider        string `json:"provider"`
	ModelCount      int    `json:"model_count"`
	CredentialState string `json:"credential_state"`
	// APIKeyMasked is the shared credential, masked. Empty when the state is
	// not "shared".
	APIKeyMasked string `json:"api_key_masked,omitempty"`
	// APIBase is the endpoint every model agrees on; empty when they differ,
	// with APIBaseMixed saying which of the two it is.
	APIBase      string `json:"api_base,omitempty"`
	APIBaseMixed bool   `json:"api_base_mixed"`
	AuthMethod   string `json:"auth_method,omitempty"`
	// HoldsDefaultModel says whether the configured default chat model belongs
	// to this provider, so a delete confirmation can say so before it happens.
	HoldsDefaultModel bool                   `json:"holds_default_model"`
	ReferenceSites    []string               `json:"reference_sites,omitempty"`
	Models            []providerModelSummary `json:"models"`
}

// canonicalProviderKey normalizes a provider key arriving from a client or
// stored in a model entry, so "OpenCode", "opencode" and a registered alias all
// address the same provider.
func canonicalProviderKey(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized := providers.NormalizeProvider(trimmed)
	if normalized == "" {
		return strings.ToLower(trimmed)
	}
	return normalized
}

// providerModelIndexes returns the model_list positions belonging to one
// provider, in ascending order.
func providerModelIndexes(cfg *config.Config, providerKey string) []int {
	if cfg == nil || providerKey == "" {
		return nil
	}
	indexes := make([]int, 0, len(cfg.ModelList))
	for i, mc := range cfg.ModelList {
		if mc == nil || mc.IsVirtual() {
			continue
		}
		if canonicalProviderKey(mc.Provider) == providerKey {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func describeProvider(cfg *config.Config, providerKey string) providerResponse {
	indexes := providerModelIndexes(cfg, providerKey)
	response := providerResponse{
		Provider:        providerKey,
		ModelCount:      len(indexes),
		CredentialState: providerCredentialUnset,
		Models:          make([]providerModelSummary, 0, len(indexes)),
	}
	if len(indexes) == 0 {
		return response
	}

	defaultModelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	removedNames := make([]string, 0, len(indexes))

	keys := make(map[string]bool)
	bases := make(map[string]bool)
	var firstKey string
	for _, index := range indexes {
		mc := cfg.ModelList[index]
		key := strings.TrimSpace(mc.APIKey())
		if key != "" {
			if firstKey == "" {
				firstKey = key
			}
			keys[key] = true
		} else {
			keys[""] = true
		}
		bases[strings.TrimSpace(mc.APIBase)] = true
		isDefault := defaultModelName != "" && strings.TrimSpace(mc.ModelName) == defaultModelName
		if isDefault {
			response.HoldsDefaultModel = true
		}
		if response.AuthMethod == "" {
			response.AuthMethod = strings.TrimSpace(mc.AuthMethod)
		}
		removedNames = append(removedNames, mc.ModelName)
		response.Models = append(response.Models, providerModelSummary{
			Index:     index,
			ModelName: mc.ModelName,
			Model:     mc.Model,
			Enabled:   mc.Enabled,
			IsDefault: isDefault,
			HasAPIKey: key != "",
		})
	}

	switch {
	case len(keys) == 1 && firstKey != "":
		response.CredentialState = providerCredentialShared
		response.APIKeyMasked = maskAPIKey(firstKey)
	case len(keys) == 1:
		response.CredentialState = providerCredentialUnset
	default:
		response.CredentialState = providerCredentialMixed
	}

	if len(bases) == 1 {
		for base := range bases {
			response.APIBase = base
		}
	} else {
		response.APIBaseMixed = true
	}

	response.ReferenceSites = modelReferenceSites(cfg, newModelReferenceSet(removedNames...))
	return response
}

// handleListProviders returns every configured provider as a manageable object.
//
//	GET /api/providers
func (h *Handler) handleListProviders(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	seen := make(map[string]bool)
	keys := make([]string, 0)
	for _, mc := range cfg.ModelList {
		if mc == nil || mc.IsVirtual() {
			continue
		}
		key := canonicalProviderKey(mc.Provider)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	sort.Strings(keys)

	response := make([]providerResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, describeProvider(cfg, key))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"providers": response})
}

// handleGetProvider returns one provider's manageable state.
//
//	GET /api/providers/{provider}
func (h *Handler) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	providerKey := canonicalProviderKey(r.PathValue("provider"))
	if providerKey == "" {
		http.Error(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	if len(providerModelIndexes(cfg, providerKey)) == 0 {
		http.Error(w, fmt.Sprintf("Provider %q is not configured", providerKey), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(describeProvider(cfg, providerKey))
}

// handleUpdateProvider applies provider-scoped changes to every model that
// belongs to the provider.
//
//	PUT /api/providers/{provider}
//
// A field is applied only when the request carries it, so rotating a credential
// does not also rewrite the endpoint. An empty api_key means "leave the stored
// credential alone": clearing a key is deleting the provider's ability to
// answer, and is done by deleting the provider or editing a model, never as the
// silent consequence of submitting a form with a blank field.
func (h *Handler) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	providerKey := canonicalProviderKey(r.PathValue("provider"))
	if providerKey == "" {
		http.Error(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var rawFields map[string]json.RawMessage
	if err = json.Unmarshal(body, &rawFields); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	var req struct {
		APIKey  string `json:"api_key"`
		APIBase string `json:"api_base"`
		Proxy   string `json:"proxy"`
	}
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	indexes := providerModelIndexes(cfg, providerKey)
	if len(indexes) == 0 {
		http.Error(w, fmt.Sprintf("Provider %q is not configured", providerKey), http.StatusNotFound)
		return
	}

	newKey := strings.TrimSpace(req.APIKey)
	_, apiBaseSent := rawFields["api_base"]
	_, proxySent := rawFields["proxy"]

	if newKey == "" && !apiBaseSent && !proxySent {
		http.Error(w, "Nothing to update: send api_key, api_base or proxy", http.StatusBadRequest)
		return
	}

	updated := 0
	for _, index := range indexes {
		mc := cfg.ModelList[index]
		if newKey != "" {
			// Replace the whole key list rather than the first entry. A
			// multi-key entry that kept its other keys would go on failing over
			// to the credential the owner just rotated away from.
			mc.APIKeys = nil
			mc.SetAPIKey(newKey)
		}
		if apiBaseSent {
			mc.APIBase = strings.TrimSpace(req.APIBase)
		}
		if proxySent {
			mc.Proxy = strings.TrimSpace(req.Proxy)
		}
		updated++
	}

	normalizeStoredModelProviders(cfg)
	if err = cfg.ValidateModelList(); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}
	if err = config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	// The credential itself is never logged, only the fact of the rotation and
	// how many entries it reached.
	logger.Infof("provider %s updated: %d model(s), credential_rotated=%t", providerKey, updated, newKey != "")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":            "ok",
		"provider":          providerKey,
		"models_updated":    updated,
		"credential_change": newKey != "",
	})
}

// handleDeleteProvider removes a provider and everything that depended on it.
//
//	DELETE /api/providers/{provider}
//
// Deleting a provider deletes its models: a model whose provider is gone has no
// endpoint and no credential, and leaving it listed would offer the user a
// selection that cannot answer. Every reference to those models is cleared in
// the same write, so no chain, default or agent is left naming an entry that no
// longer exists.
//
// The response reports what it removed, so the console can confirm the delete
// actually reconciled rather than assume it.
func (h *Handler) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	providerKey := canonicalProviderKey(r.PathValue("provider"))
	if providerKey == "" {
		http.Error(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	indexes := providerModelIndexes(cfg, providerKey)
	if len(indexes) == 0 {
		http.Error(w, fmt.Sprintf("Provider %q is not configured", providerKey), http.StatusNotFound)
		return
	}

	removedNames := make([]string, 0, len(indexes))
	remove := make(map[int]bool, len(indexes))
	for _, index := range indexes {
		remove[index] = true
		removedNames = append(removedNames, cfg.ModelList[index].ModelName)
	}

	kept := make(config.SecureModelList, 0, len(cfg.ModelList)-len(indexes))
	for i, mc := range cfg.ModelList {
		if remove[i] {
			continue
		}
		kept = append(kept, mc)
	}
	cfg.ModelList = kept

	clearedSites := purgeModelReferences(cfg, newModelReferenceSet(removedNames...))

	if err = config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Infof("provider %s deleted: %d model(s) removed, %d reference site(s) cleared",
		providerKey, len(removedNames), len(clearedSites))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":           "ok",
		"provider":         providerKey,
		"models_removed":   removedNames,
		"cleared_sites":    clearedSites,
		"models_remaining": len(cfg.ModelList),
	})
}
