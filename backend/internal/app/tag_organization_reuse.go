package app

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/storage"
)

// topicVocabularyHints bounds proposal context, not the persisted vocabulary.
// Reconciliation subsequently checks every candidate page, including omitted hints.
func topicVocabularyHints(defs []storage.TopicDefinition, inputs []topicVocabularySample) []storage.TopicDefinition {
	text, _ := json.Marshal(inputs)
	source := strings.ToLower(string(text))
	ordered := append([]storage.TopicDefinition(nil), defs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return strings.Contains(source, strings.ToLower(ordered[i].Name)) && !strings.Contains(source, strings.ToLower(ordered[j].Name))
	})
	out := []storage.TopicDefinition{}
	bytes := 0
	for _, d := range ordered {
		// Definitions are retained for classification; the hint only needs names.
		d.Description = ""
		encoded, _ := json.Marshal(d)
		if bytes+len(encoded) > 6000 {
			continue
		}
		out = append(out, d)
		bytes += len(encoded)
	}
	return out
}

func topicCatalog(defs, candidates []storage.TopicDefinition) []storage.TopicDefinition {
	out := append([]storage.TopicDefinition(nil), defs...)
	seen := map[string]bool{}
	for _, d := range defs {
		seen[strings.ToLower(d.Name)] = true
	}
	for _, d := range candidates {
		if !seen[strings.ToLower(d.Name)] {
			out = append(out, d)
			seen[strings.ToLower(d.Name)] = true
		}
	}
	return out
}

// reuseTopicProposals maps names only to supplied candidates. It never asks the
// model to merge existing tags or grants a write outside the current job scope.
func (a *App) reuseTopicProposals(ctx context.Context, proposed, catalog []storage.TopicDefinition) ([]storage.TopicDefinition, error) {
	resolved := append([]storage.TopicDefinition(nil), proposed...)
	owners := map[string]storage.TopicDefinition{}
	for _, d := range catalog {
		for _, name := range append([]string{d.Name}, d.Aliases...) {
			key := strings.ToLower(strings.TrimSpace(name))
			if _, ok := owners[key]; !ok {
				owners[key] = d
			}
		}
	}
	pending := map[string]int{}
	for i, d := range proposed {
		if existing, ok := owners[strings.ToLower(d.Name)]; ok {
			if existing.Description == "" {
				existing.Description = d.Description
			}
			resolved[i] = existing
		} else {
			pending[d.Name] = i
		}
	}
	size := 100
	for offset := 0; offset < len(catalog) && len(pending) > 0; {
		end := min(offset+size, len(catalog))
		page := catalog[offset:end]
		proposals := []storage.TopicDefinition{}
		for _, d := range proposed {
			if _, ok := pending[d.Name]; ok {
				proposals = append(proposals, d)
			}
		}
		raw, err := a.topicComplete(ctx, prompts.TopicReusePrompt(), map[string]any{"proposed": proposals, "existing": page})
		if err != nil {
			if topicOrganizationErrorCode(err) == "AI_CONTEXT_TOO_LARGE" && len(page) > 1 {
				size = max(1, len(page)/2)
				continue
			}
			return nil, err
		}
		var result struct {
			Matches []struct {
				ProposedName  string `json:"proposedName"`
				CanonicalName string `json:"canonicalName"`
			} `json:"matches"`
		}
		if err := decodeTopicJSON(raw, &result); err != nil || len(result.Matches) > len(pending) {
			return nil, &core.ToolError{Code: "AI_ORGANIZATION_INVALID_JSON", Message: "Invalid reuse response"}
		}
		targets := map[string]storage.TopicDefinition{}
		for _, d := range page {
			targets[d.Name] = d
		}
		for _, match := range result.Matches {
			i, ok := pending[match.ProposedName]
			target, targetOK := targets[match.CanonicalName]
			if !ok || !targetOK {
				return nil, &core.ToolError{Code: "AI_ORGANIZATION_VOCABULARY_CONFLICT", Message: "Out of scope reuse mapping"}
			}
			if target.Description == "" {
				target.Description = proposed[i].Description
			}
			resolved[i] = target
			delete(pending, match.ProposedName)
		}
		offset = end
	}
	return resolved, nil
}
