package app

import (
	"context"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/storage"
)

func topicNameKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// normalizeTopicProposals never merges distinct canonical subjects based on an
// ambiguous alias. Optional aliases that have multiple owners are simply omitted.
// Existing names/aliases are reserved and cannot be claimed by a new proposal.
func normalizeTopicProposals(proposed, existing []storage.TopicDefinition) ([]storage.TopicDefinition, error) {
	out := []storage.TopicDefinition{}
	positions := map[string]int{}
	for _, d := range proposed {
		d.Name = strings.TrimSpace(d.Name)
		d.Description = strings.TrimSpace(d.Description)
		key := topicNameKey(d.Name)
		if i, ok := positions[key]; ok {
			out[i].Aliases = append(out[i].Aliases, d.Aliases...)
			continue
		}
		positions[key] = len(out)
		d.Aliases = append([]string{}, d.Aliases...)
		out = append(out, d)
	}
	reserved := map[string]bool{}
	for _, d := range existing {
		for _, name := range append([]string{d.Name}, d.Aliases...) {
			reserved[topicNameKey(name)] = true
		}
	}
	owners := map[string]string{}
	for _, d := range out {
		for _, alias := range d.Aliases {
			key := topicNameKey(alias)
			if old, ok := owners[key]; ok && old != d.Name {
				owners[key] = ""
			} else if !ok {
				owners[key] = d.Name
			}
		}
	}
	for i, d := range out {
		aliases, seen := []string{}, map[string]bool{}
		for _, alias := range d.Aliases {
			alias = strings.TrimSpace(alias)
			key := topicNameKey(alias)
			_, canonical := positions[key]
			if key == "" || canonical || reserved[key] || seen[key] || owners[key] != d.Name || utf8.RuneCountInString(alias) > 64 {
				continue
			}
			seen[key] = true
			if len(aliases) < 20 {
				aliases = append(aliases, alias)
			}
		}
		out[i].Aliases = aliases
	}
	if err := validateTopicVocabulary(out); err != nil {
		return nil, &core.ToolError{Code: "AI_ORGANIZATION_VOCABULARY_INVALID", Message: "Invalid topic definition"}
	}
	return out, nil
}

// proposeTopicVocabulary retries an invalid model response once with bounded,
// structured feedback. No checkpoint advances until the response is valid.
func (a *App) proposeTopicVocabulary(ctx context.Context, inputs []topicVocabularySample, catalog []storage.TopicDefinition, locale string) ([]storage.TopicDefinition, error) {
	data := map[string]any{"existing": topicVocabularyHints(catalog, inputs), "movies": inputs, "labelLocale": locale}
	var validationErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := a.topicComplete(ctx, prompts.TopicVocabularyPrompt(), data)
		if err != nil {
			return nil, err
		}
		var result struct {
			Topics []storage.TopicDefinition `json:"topics"`
		}
		switch {
		case decodeTopicJSON(raw, &result) != nil:
			validationErr = &core.ToolError{Code: "AI_ORGANIZATION_INVALID_JSON", Message: "Invalid classification JSON"}
		case len(result.Topics) > 12:
			validationErr = &core.ToolError{Code: "AI_ORGANIZATION_VOCABULARY_LIMIT", Message: "Too many topic proposals"}
		default:
			var normalized []storage.TopicDefinition
			normalized, validationErr = normalizeTopicProposals(result.Topics, nil)
			if validationErr == nil {
				return normalized, nil
			}
		}
		data["validationError"] = topicOrganizationErrorCode(validationErr)
	}
	return nil, validationErr
}
