package app

import (
	"context"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/storage"
)

// classifyTopicBatch preserves full source evidence. Once movie batching reaches
// one movie, a large existing vocabulary is paged too, rather than failing every
// remaining movie with the same context overflow.
func (a *App) classifyTopicBatch(ctx context.Context, defs []storage.TopicDefinition, inputs []storage.TopicMovieInput) ([]topicClassification, map[string][]string, error) {
	combined := make([]topicClassification, len(inputs))
	indices := make(map[string]int, len(inputs))
	for i, input := range inputs {
		combined[i] = topicClassification{MovieID: input.MovieID, Matches: []topicMatch{}}
		indices[input.MovieID] = i
	}
	size := max(1, len(defs))
	for offset := 0; ; {
		end := min(offset+size, len(defs))
		page := defs[offset:end]
		raw, err := a.topicComplete(ctx, prompts.TopicClassificationPrompt(), map[string]any{"vocabulary": page, "movies": inputs})
		if err != nil {
			if topicOrganizationErrorCode(err) == "AI_CONTEXT_TOO_LARGE" && len(inputs) == 1 && len(page) > 1 {
				size = max(1, len(page)/2)
				continue
			}
			return nil, nil, err
		}
		var result struct {
			Movies []topicClassification `json:"movies"`
		}
		if err := decodeTopicJSON(raw, &result); err != nil {
			return nil, nil, &core.ToolError{Code: "AI_ORGANIZATION_INVALID_JSON", Message: "Invalid classification JSON"}
		}
		if _, err := validateTopicClassification(result.Movies, inputs, page); err != nil {
			return nil, nil, &core.ToolError{Code: "AI_ORGANIZATION_INVALID_EVIDENCE", Message: "Invalid classification evidence"}
		}
		for _, movie := range result.Movies {
			target := &combined[indices[movie.MovieID]]
			target.Reason = movie.Reason
			seen := map[string]bool{}
			for _, match := range target.Matches {
				seen[match.Topic] = true
			}
			for _, match := range movie.Matches {
				if !seen[match.Topic] && len(target.Matches) < 12 {
					target.Matches = append(target.Matches, match)
					seen[match.Topic] = true
				}
			}
		}
		if end == len(defs) {
			matches, err := validateTopicClassification(combined, inputs, defs)
			return combined, matches, err
		}
		offset = end
	}
}
