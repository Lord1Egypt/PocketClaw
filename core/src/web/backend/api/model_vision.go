package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// handleSetVisionModel points the dedicated vision model at an existing entry,
// or clears it.
//
//	POST /api/models/vision   {"model_name": "..."}  set
//	POST /api/models/vision   {"model_name": ""}     clear
//
// Clearing is a real state, not an error: with no dedicated vision model an
// image turn goes to the default model exactly as it does today, and keeps the
// normal fallback chain behind it. That is the behaviour every existing install
// already has, and it stays the default.
//
// Setting one changes which chain an image turn walks, not just its first
// entry: the agent builds image candidates from image_model plus
// image_model_fallbacks and routeMediaTurn substitutes that list wholesale, so
// the Fallback Models configured for text turns do not stand behind a vision
// model. Nothing writes image_model_fallbacks yet, so a vision model set here
// runs without a fallback.
//
// Selecting a model that is already in model_list is this endpoint. Selecting a
// discovered model is `POST /api/models/materialize` with role "vision", which
// creates the entry and assigns the role in one write. Both share
// validateVisionModelSelection so neither can be the lenient one.
func (h *Handler) handleSetVisionModel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		ModelName string `json:"model_name"`
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

	modelName := strings.TrimSpace(req.ModelName)
	if modelName != "" {
		if reason := validateVisionModelSelection(cfg, modelName); reason != "" {
			status := http.StatusBadRequest
			if strings.Contains(reason, "not found in model_list") {
				status = http.StatusNotFound
			}
			http.Error(w, reason, status)
			return
		}
	}

	cfg.Agents.Defaults.ImageModel = modelName

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":      "ok",
		"image_model": modelName,
	})
}
