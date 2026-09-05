package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Roles a materialized model can be given in one operation.
const (
	materializeRoleNone     = ""
	materializeRoleDefault  = "default"
	materializeRoleFallback = "fallback"
)

// materializeModelRequest asks for a discovered model to be made routable.
//
// SourceIndex names the configured model_list entry whose provider identity and
// credential the new entry inherits. The API key is never part of this request:
// the caller only ever names an index, and the key is read from stored config
// on this side of the boundary.
type materializeModelRequest struct {
	SourceIndex int    `json:"source_index"`
	Model       string `json:"model"`
	Role        string `json:"role"`
	ModelName   string `json:"model_name,omitempty"`
}

// handleMaterializeModel turns a discovered model into a routable model_list
// entry and applies a role to it, in one config write.
//
//	POST /api/models/materialize
//
// Default, fallback and any future role reference model_list entries by name,
// and that invariant is not relaxed here — a discovered model simply cannot be
// referenced until it exists. Doing both halves in one handler is what keeps a
// failed role from leaving a stray entry behind: everything below mutates an
// in-memory config and there is exactly one SaveConfig at the end, so a
// rejected role writes nothing at all.
func (h *Handler) handleMaterializeModel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req materializeModelRequest
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	requestedModel := strings.TrimSpace(req.Model)
	if requestedModel == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}

	switch req.Role {
	case materializeRoleNone, materializeRoleDefault, materializeRoleFallback:
	default:
		http.Error(w, fmt.Sprintf("Unknown role %q", req.Role), http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	normalizeStoredModelProviders(cfg)

	if req.SourceIndex < 0 || req.SourceIndex >= len(cfg.ModelList) {
		http.Error(w, fmt.Sprintf("Index %d out of range (0-%d)", req.SourceIndex, len(cfg.ModelList)-1), http.StatusNotFound)
		return
	}
	source := cfg.ModelList[req.SourceIndex]

	// Discovery is only ever offered for providers the user has configured, and
	// the same rule has to hold on this side: an unconfigured template entry
	// carries no credential to inherit.
	if !hasModelConfiguration(source) {
		http.Error(
			w,
			fmt.Sprintf("Model %q is not configured, so it cannot supply provider credentials", source.ModelName),
			http.StatusBadRequest,
		)
		return
	}

	entry, index, created, err := upsertModelForProviderInstance(cfg, source, requestedModel, req.ModelName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Role {
	case materializeRoleDefault:
		if reason := validateDefaultModelSelection(cfg, entry.ModelName); reason != "" {
			http.Error(w, reason, http.StatusBadRequest)
			return
		}
		cfg.Agents.Defaults.ModelName = entry.ModelName
	case materializeRoleFallback:
		appended := append(append([]string{}, cfg.Agents.Defaults.ModelFallbacks...), entry.ModelName)
		normalized, reason := normalizeModelFallbacks(cfg, appended)
		if reason != "" {
			http.Error(w, reason, http.StatusBadRequest)
			return
		}
		cfg.Agents.Defaults.ModelFallbacks = normalized
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	fallbacks := cfg.Agents.Defaults.ModelFallbacks
	if fallbacks == nil {
		fallbacks = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":          "ok",
		"model_name":      entry.ModelName,
		"index":           index,
		"created":         created,
		"role":            req.Role,
		"default_model":   cfg.Agents.Defaults.GetModelName(),
		"model_fallbacks": fallbacks,
	})
}

// upsertModelForProviderInstance finds, or creates, the model_list entry for a
// model offered by the same provider instance as source.
//
// Identity is the provider instance, not the model id: the normalized provider,
// the normalized API base and the model id together. Two configured providers
// can expose the same model id — deduplicating on the id alone would silently
// route one provider's model through the other's credential.
func upsertModelForProviderInstance(
	cfg *config.Config,
	source *config.ModelConfig,
	requestedModel string,
	preferredName string,
) (*config.ModelConfig, int, bool, error) {
	sourceProvider, _ := providers.ExtractProtocol(source)
	modelID := strings.TrimSpace(requestedModel)
	// A discovered id may arrive provider-prefixed. Store the bare id and let
	// the Provider field carry the routing, which is what the rest of the
	// config already assumes.
	if prefix := providers.NormalizeProvider(sourceProvider) + "/"; strings.HasPrefix(strings.ToLower(modelID), prefix) {
		modelID = modelID[len(prefix):]
	}
	if modelID == "" {
		return nil, 0, false, fmt.Errorf("model is required")
	}

	sourceBase := providerInstanceAPIBase(source, sourceProvider)

	for i, candidate := range cfg.ModelList {
		if candidate == nil || candidate.IsVirtual() {
			continue
		}
		candidateProvider, candidateModel := providers.ExtractProtocol(candidate)
		if providers.NormalizeProvider(candidateProvider) != providers.NormalizeProvider(sourceProvider) {
			continue
		}
		if normalizeAPIBaseForCompare(providerInstanceAPIBase(candidate, candidateProvider)) !=
			normalizeAPIBaseForCompare(sourceBase) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(candidateModel), modelID) {
			continue
		}
		// Already routable through this exact provider instance. Reusing it is
		// what stops a second selection creating a duplicate entry.
		return candidate, i, false, nil
	}

	entry := &config.ModelConfig{
		ModelName: uniqueModelName(cfg, preferredName, modelID, sourceProvider),
		Provider:  sourceProvider,
		Model:     modelID,
		APIBase:   source.APIBase,
		Proxy:     source.Proxy,

		AuthMethod:  source.AuthMethod,
		ConnectMode: source.ConnectMode,
		Workspace:   source.Workspace,

		RPM:                 source.RPM,
		MaxTokensField:      source.MaxTokensField,
		RequestTimeout:      source.RequestTimeout,
		ThinkingLevel:       source.ThinkingLevel,
		ToolSchemaTransform: source.ToolSchemaTransform,
		Streaming:           source.Streaming,
		ExtraBody:           source.ExtraBody,
		CustomHeaders:       source.CustomHeaders,

		// The credential is carried across as the stored SecureStrings value.
		// It is never decrypted here and never crosses the API boundary: the
		// caller named an index, not a key.
		APIKeys: source.APIKeys,
		Enabled: true,
	}

	cfg.ModelList = append(cfg.ModelList, entry)
	return entry, len(cfg.ModelList) - 1, true, nil
}

// providerInstanceAPIBase is the base a model actually talks to, with the
// provider default filled in so a stored empty string and an explicit default
// compare equal.
func providerInstanceAPIBase(m *config.ModelConfig, protocol string) string {
	base := strings.TrimSpace(m.APIBase)
	if base != "" {
		return base
	}
	return providers.DefaultAPIBaseForProtocol(protocol)
}

// uniqueModelName picks an alias that does not collide with an existing entry.
//
// model_name is the key every role reference uses, so a collision would make a
// new entry shadow an existing one in the chain.
func uniqueModelName(cfg *config.Config, preferred, modelID, provider string) string {
	taken := make(map[string]bool, len(cfg.ModelList))
	for _, m := range cfg.ModelList {
		if m != nil {
			taken[m.ModelName] = true
		}
	}

	candidates := make([]string, 0, 3)
	if trimmed := strings.TrimSpace(preferred); trimmed != "" {
		candidates = append(candidates, trimmed)
	}
	candidates = append(candidates, modelID)
	if normalized := providers.NormalizeProvider(provider); normalized != "" {
		candidates = append(candidates, normalized+"-"+modelID)
	}

	for _, candidate := range candidates {
		if candidate != "" && !taken[candidate] {
			return candidate
		}
	}

	base := candidates[len(candidates)-1]
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", base, suffix)
		if !taken[candidate] {
			return candidate
		}
	}
}
