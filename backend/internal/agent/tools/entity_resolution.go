package tools

import (
	"context"
	"strings"
	"unicode"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"curated-backend/internal/library/moviecode"
)

const resolveEntitiesName = "resolve_entities"

// resolveEntities is intentionally read-only and does not seed the turn's
// ref stores. A matched entity is still only remembered by the loop after the
// tool result; ambiguous candidates can only become a trusted anchor through a
// later explicit browser selection.
func resolveEntities(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name: resolveEntitiesName,
		Description: "Resolve one local movie code/title or actor canonical name/alias before acting on an entity named by the user. " +
			"Returns matched, ambiguous, or unmatched. For ambiguous results, ask the user to select a supplied candidate; do not guess.",
		ParamsSchema: object(map[string]core.Schema{
			"query": strField("Movie code, movie title, actor canonical name, or actor alias"),
			"kind":  {Type: "string", Description: "Optional expected entity kind", Enum: []string{"auto", "movie", "actor"}},
		}, "query"),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			query := strArg(args, "query")
			kind := strArg(args, "kind")
			if kind == "" {
				kind = "auto"
			}
			if query == "" {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "query is required"}}, nil
			}
			resolution, err := resolveLocalEntity(ctx, q, query, kind)
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
			}
			return core.Result{OK: true, Data: wrapSource(resolution)}, nil
		},
	}
}

func resolveLocalEntity(ctx context.Context, q LibraryQuery, query, kind string) (contracts.AIEntityResolutionDTO, error) {
	if kind == "actor" || kind == "auto" {
		if profile, err := q.GetActorProfile(ctx, query); err == nil && strings.TrimSpace(profile.Name) != "" {
			return contracts.AIEntityResolutionDTO{
				Query: query, Kind: "actor", Status: "matched",
				Candidates: []contracts.AIEntityCandidateDTO{{
					Kind: "actor", ActorName: profile.Name, Aliases: profile.Aliases, Reason: "canonical actor or saved alias",
				}},
			}, nil
		}
	}

	if kind == "movie" || kind == "auto" {
		page, err := q.ListMovies(ctx, contracts.ListMoviesRequest{Query: query, Limit: 6})
		if err != nil {
			return contracts.AIEntityResolutionDTO{}, err
		}
		matches := exactMovieCandidates(query, page.Items)
		if len(matches) == 1 {
			return contracts.AIEntityResolutionDTO{Query: query, Kind: "movie", Status: "matched", Candidates: matches}, nil
		}
		if len(matches) > 1 {
			return contracts.AIEntityResolutionDTO{Query: query, Kind: "movie", Status: "ambiguous", Candidates: matches, Reason: "multiple local movies share this code or title"}, nil
		}
		if kind == "movie" && len(page.Items) > 1 {
			return contracts.AIEntityResolutionDTO{Query: query, Kind: "movie", Status: "ambiguous", Candidates: movieCandidates(page.Items), Reason: "more than one local title is a possible match"}, nil
		}
	}

	if kind == "actor" || kind == "auto" {
		page, err := q.ListActors(ctx, contracts.ListActorsRequest{Q: query, Limit: 6})
		if err != nil {
			return contracts.AIEntityResolutionDTO{}, err
		}
		candidates := actorCandidates(page.Actors)
		if len(candidates) > 1 {
			return contracts.AIEntityResolutionDTO{Query: query, Kind: "actor", Status: "ambiguous", Candidates: candidates, Reason: "more than one local actor is a possible match"}, nil
		}
		if len(candidates) == 1 && normalizeEntityText(candidates[0].ActorName) == normalizeEntityText(query) {
			return contracts.AIEntityResolutionDTO{Query: query, Kind: "actor", Status: "matched", Candidates: candidates}, nil
		}
	}

	return contracts.AIEntityResolutionDTO{Query: query, Kind: kind, Status: "unmatched", Candidates: []contracts.AIEntityCandidateDTO{}, Reason: "no unique local entity matched"}, nil
}

func exactMovieCandidates(query string, items []contracts.MovieListItemDTO) []contracts.AIEntityCandidateDTO {
	code := moviecode.NormalizeForStorageID(query)
	title := normalizeEntityText(query)
	out := make([]contracts.AIEntityCandidateDTO, 0, len(items))
	for _, item := range items {
		if (code != "" && moviecode.NormalizeForStorageID(item.Code) == code) || (title != "" && normalizeEntityText(item.Title) == title) {
			out = append(out, movieCandidate(item))
		}
	}
	return out
}

func movieCandidates(items []contracts.MovieListItemDTO) []contracts.AIEntityCandidateDTO {
	out := make([]contracts.AIEntityCandidateDTO, 0, len(items))
	for _, item := range items {
		out = append(out, movieCandidate(item))
	}
	return out
}

func movieCandidate(item contracts.MovieListItemDTO) contracts.AIEntityCandidateDTO {
	return contracts.AIEntityCandidateDTO{Kind: "movie", MovieID: item.ID, Title: item.Title, Code: item.Code, Reason: "local library movie"}
}

func actorCandidates(items []contracts.ActorListItemDTO) []contracts.AIEntityCandidateDTO {
	out := make([]contracts.AIEntityCandidateDTO, 0, len(items))
	for _, item := range items {
		out = append(out, contracts.AIEntityCandidateDTO{Kind: "actor", ActorName: item.Name, Reason: "local library actor"})
	}
	return out
}

func normalizeEntityText(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
