package app

import (
	"context"
	"encoding/json"
	"fmt"
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
	// Canonical labels take precedence over aliases in legacy catalogs.
	for _, d := range catalog {
		owners[topicNameKey(d.Name)] = d
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
		mappings, err := a.topicReusePage(ctx, proposals, page)
		if err != nil {
			if topicOrganizationErrorCode(err) == "AI_CONTEXT_TOO_LARGE" && len(page) > 1 {
				size = max(1, len(page)/2)
				continue
			}
			return nil, err
		}
		for proposedName, target := range mappings {
			i := pending[proposedName]
			if target.Description == "" {
				target.Description = proposed[i].Description
			}
			resolved[i] = target
			delete(pending, proposedName)
		}
		offset = end
	}
	return resolved, nil
}

// topicReusePage validates the whole page before mutating any pending mapping.
// Identical duplicate mappings are harmless; conflicting or invented targets retry.
func (a *App) topicReusePage(ctx context.Context, proposals, page []storage.TopicDefinition) (map[string]storage.TopicDefinition, error) {
	data := struct {
		Existing        []storage.TopicDefinition `json:"existing"`
		Proposed        []storage.TopicDefinition `json:"proposed"`
		ValidationError string                    `json:"validationError,omitempty"`
	}{Existing: page, Proposed: proposals}
	targets := map[string]storage.TopicDefinition{}
	names := map[string]string{}
	for _, d := range page {
		targets[topicNameKey(d.Name)] = d
	}
	for _, d := range proposals {
		names[topicNameKey(d.Name)] = d.Name
	}
	var validationErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := a.topicComplete(ctx, prompts.TopicReusePrompt(), data)
		if err != nil {
			return nil, err
		}
		var result struct {
			Matches []struct {
				ProposedName  string `json:"proposedName"`
				CanonicalName string `json:"canonicalName"`
			} `json:"matches"`
		}
		validationErr = decodeTopicJSON(raw, &result)
		mappings := map[string]storage.TopicDefinition{}
		if validationErr == nil {
			for _, match := range result.Matches {
				name, ok := names[topicNameKey(match.ProposedName)]
				target, targetOK := targets[topicNameKey(match.CanonicalName)]
				old, duplicate := mappings[name]
				if !ok || !targetOK || (duplicate && old.Name != target.Name) {
					validationErr = fmt.Errorf("invalid reuse mapping")
					break
				}
				mappings[name] = target
			}
		}
		if validationErr == nil {
			return mappings, nil
		}
		data.ValidationError = "AI_ORGANIZATION_REUSE_INVALID"
	}
	return nil, &core.ToolError{Code: "AI_ORGANIZATION_REUSE_INVALID", Message: "Invalid topic reuse mapping after correction"}
}
