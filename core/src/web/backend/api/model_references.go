package api

import (
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// modelReferenceSet is the set of model_list names being removed, keyed by the
// trimmed name exactly as a reference would spell it.
type modelReferenceSet map[string]bool

func newModelReferenceSet(names ...string) modelReferenceSet {
	set := make(modelReferenceSet, len(names))
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}

func (s modelReferenceSet) has(name string) bool {
	return s[strings.TrimSpace(name)]
}

// purgeModelReferences clears every reference to a removed model_list name and
// returns the names of the sites it changed, sorted, for reporting.
//
// A model_list name is referenced from more places than the default model and
// the two default fallback chains that the single-model delete path covered:
// the image model, the complexity router's light model, and each agent's own
// primary/fallback pair — including a subagent override — all name entries the
// same way. A name left in any of them is a candidate the router resolves
// against a model that no longer exists.
//
// Scalar references are cleared to the empty string rather than repointed at
// some surviving model. Which model should take over is the user's decision, and
// every consumer of these fields already treats empty as "not configured": an
// empty light model, for instance, means no router is constructed at all.
func purgeModelReferences(cfg *config.Config, removed modelReferenceSet) []string {
	if cfg == nil || len(removed) == 0 {
		return nil
	}
	changed := make(map[string]bool)

	defaults := &cfg.Agents.Defaults
	clearScalar := func(site string, field *string) {
		if removed.has(*field) {
			*field = ""
			changed[site] = true
		}
	}
	clearList := func(site string, field *[]string) {
		kept := make([]string, 0, len(*field))
		for _, reference := range *field {
			if removed.has(reference) {
				changed[site] = true
				continue
			}
			kept = append(kept, reference)
		}
		if len(kept) == 0 {
			// nil rather than an empty slice, so the field is omitted from the
			// saved config instead of written as [].
			*field = nil
			return
		}
		*field = kept
	}

	clearScalar("agents.defaults.model_name", &defaults.ModelName)
	clearScalar("agents.defaults.image_model", &defaults.ImageModel)
	clearList("agents.defaults.model_fallbacks", &defaults.ModelFallbacks)
	clearList("agents.defaults.image_model_fallbacks", &defaults.ImageModelFallbacks)
	if defaults.Routing != nil {
		clearScalar("agents.defaults.routing.light_model", &defaults.Routing.LightModel)
	}

	for _, agent := range cfg.Agents.List {
		if agent.Model != nil {
			clearScalar("agents.list.model.primary", &agent.Model.Primary)
			clearList("agents.list.model.fallbacks", &agent.Model.Fallbacks)
		}
		if agent.Subagents != nil && agent.Subagents.Model != nil {
			clearScalar("agents.list.subagents.model.primary", &agent.Subagents.Model.Primary)
			clearList("agents.list.subagents.model.fallbacks", &agent.Subagents.Model.Fallbacks)
		}
	}

	sites := make([]string, 0, len(changed))
	for site := range changed {
		sites = append(sites, site)
	}
	sort.Strings(sites)
	return sites
}

// modelReferenceSites reports which sites currently name any of the given
// model_list entries, without changing anything. It answers "what does removing
// these depend on" for a confirmation prompt, and is a read-only scan of the
// same site list purgeModelReferences writes to.
func modelReferenceSites(cfg *config.Config, names modelReferenceSet) []string {
	if cfg == nil || len(names) == 0 {
		return nil
	}
	changed := make(map[string]bool)
	noteScalar := func(site, value string) {
		if names.has(value) {
			changed[site] = true
		}
	}
	noteList := func(site string, values []string) {
		for _, value := range values {
			if names.has(value) {
				changed[site] = true
				return
			}
		}
	}

	defaults := cfg.Agents.Defaults
	noteScalar("agents.defaults.model_name", defaults.ModelName)
	noteScalar("agents.defaults.image_model", defaults.ImageModel)
	noteList("agents.defaults.model_fallbacks", defaults.ModelFallbacks)
	noteList("agents.defaults.image_model_fallbacks", defaults.ImageModelFallbacks)
	if defaults.Routing != nil {
		noteScalar("agents.defaults.routing.light_model", defaults.Routing.LightModel)
	}
	for _, agent := range cfg.Agents.List {
		if agent.Model != nil {
			noteScalar("agents.list.model.primary", agent.Model.Primary)
			noteList("agents.list.model.fallbacks", agent.Model.Fallbacks)
		}
		if agent.Subagents != nil && agent.Subagents.Model != nil {
			noteScalar("agents.list.subagents.model.primary", agent.Subagents.Model.Primary)
			noteList("agents.list.subagents.model.fallbacks", agent.Subagents.Model.Fallbacks)
		}
	}

	sites := make([]string, 0, len(changed))
	for site := range changed {
		sites = append(sites, site)
	}
	sort.Strings(sites)
	return sites
}
