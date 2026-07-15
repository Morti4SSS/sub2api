package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

func normalizeGroupModelsListConfig(cfg GroupModelsListConfig) GroupModelsListConfig {
	out := GroupModelsListConfig{Enabled: cfg.Enabled}
	if len(cfg.Models) == 0 {
		return out
	}

	seen := make(map[string]struct{}, len(cfg.Models))
	out.Models = make([]string, 0, len(cfg.Models))
	for _, model := range cfg.Models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		out.Models = append(out.Models, model)
	}
	if len(out.Models) == 0 {
		out.Models = nil
	}
	return out
}

func (g *Group) CustomModelsListEnabled() bool {
	return g != nil && g.ModelsListConfig.Enabled && len(g.ModelsListConfig.Models) > 0
}

// ClaudeCodeModelsForGroup returns the stable Claude Code shells exposed by a group.
func ClaudeCodeModelsForGroup(group *Group) []claude.Model {
	if group == nil || !group.CustomModelsListEnabled() {
		models := make([]claude.Model, len(claude.DefaultModels))
		copy(models, claude.DefaultModels)
		return models
	}

	models := make([]claude.Model, 0, len(group.ModelsListConfig.Models))
	for _, id := range group.ModelsListConfig.Models {
		model, ok := claude.DefaultModelByID(id)
		if ok {
			models = append(models, model)
		}
	}
	return models
}

// ExposesClaudeCodeModel reports whether a shell belongs to the group's stable catalog.
func (g *Group) ExposesClaudeCodeModel(modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if _, ok := claude.DefaultModelByID(modelID); !ok {
		return false
	}
	if g == nil || !g.CustomModelsListEnabled() {
		return true
	}
	for _, configured := range g.ModelsListConfig.Models {
		if strings.TrimSpace(configured) == modelID {
			return true
		}
	}
	return false
}
