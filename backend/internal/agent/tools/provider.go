package tools

import (
	"context"
	"strings"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"curated-backend/internal/library/moviecode"
)

// ProviderLookup searches configured scrape providers and reconciles codes with the local library.
type ProviderLookup interface {
	SearchProviderTitles(ctx context.Context, keyword string, limit int) ([]ProviderTitleHit, error)
	FindLibraryMoviesByCodes(ctx context.Context, codes []string) (map[string]contracts.MovieListItemDTO, error)
}

// ProviderTitleHit is one source-site search row before local reconciliation.
type ProviderTitleHit struct {
	Code     string
	Title    string
	Provider string
	Homepage string
	Score    float64
}

func RegisterProviderTools(reg *core.Registry, query LibraryQuery, lookup ProviderLookup, pages SourcePageFetcher, movies *core.MovieRefStore, actors *core.ActorRefStore, urls *core.SourceURLStore) error {
	if err := reg.Register(searchProviderTitles(query, lookup, movies, actors)); err != nil {
		return err
	}
	return reg.Register(getSourcePage(pages, urls))
}

func searchProviderTitles(q LibraryQuery, lookup ProviderLookup, movies *core.MovieRefStore, actors *core.ActorRefStore) core.ToolDefinition {
	maxLimit := float64(core.ProviderSearchMaxLimit)
	minLimit := 1.0
	schema := object(map[string]core.Schema{
		"actorName": strField("Actor name already retrieved this turn via list_actors, get_actor_profile, movie actors, page context, or @-mention"),
		"movieId":   strField("Movie id already retrieved this turn via search_movies, get_movie_detail, get_watch_history, page context, or @-mention"),
		"limit":     {Type: "integer", Description: "Max titles, default 15, max 25", Minimum: &minLimit, Maximum: &maxLimit},
	})
	return core.ToolDefinition{
		Name: core.SearchProviderTitlesName,
		Description: "Search configured metadata providers for related titles (reviews, filmography, off-library works). " +
			"Requires actorName and/or movieId already seen this turn. Do not pass a free-text query. " +
			"Off-library rows have inLibrary=false and no movieId; never pass those to present_movies. " +
			"Use homepage and score from this result plus get_movie_detail/get_actor_profile; do not invent local ids.",
		ParamsSchema: schema,
		Permission:   core.PermissionRead,
		Domain:       core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			actorName := strArg(args, "actorName")
			movieID := strArg(args, "movieId")
			if actorName == "" && movieID == "" {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: "actorName or movieId is required; retrieve the entity this turn first",
				}}, nil
			}
			if actorName != "" && (actors == nil || !actors.Known(call.SessionID, actorName)) {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: "unknown actorName; call list_actors or get_actor_profile first",
				}}, nil
			}
			if movieID != "" {
				_, missing := movies.Lookup(call.SessionID, []string{movieID})
				if len(missing) > 0 {
					return core.Result{OK: false, Error: &core.ToolError{
						Code:    "AI_TOOL_INVALID_ARGS",
						Message: "unknown movieId; call search_movies or get_movie_detail first",
					}}, nil
				}
			}

			keyword := actorName
			queryCode := ""
			if movieID != "" {
				detail, err := q.GetMovieDetail(ctx, movieID)
				if err != nil {
					return core.Result{OK: false, Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "movie not found"}}, nil
				}
				queryCode = strings.TrimSpace(detail.Code)
				if keyword == "" {
					if len(detail.Actors) > 0 {
						keyword = strings.TrimSpace(detail.Actors[0])
					} else {
						keyword = queryCode
					}
				}
			}
			if keyword == "" {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: "could not build a provider search keyword from the given movie or actor",
				}}, nil
			}

			limit := intArg(args, "limit", core.ProviderSearchDefaultLimit)
			if limit <= 0 {
				limit = core.ProviderSearchDefaultLimit
			}
			if limit > core.ProviderSearchMaxLimit {
				limit = core.ProviderSearchMaxLimit
			}
			hits, err := lookup.SearchProviderTitles(ctx, keyword, limit)
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
			}
			codes := make([]string, 0, len(hits))
			for _, hit := range hits {
				codes = append(codes, hit.Code)
			}
			local, err := lookup.FindLibraryMoviesByCodes(ctx, codes)
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
			}

			items := make([]map[string]any, 0, len(hits))
			for _, hit := range hits {
				item := map[string]any{
					"code":      hit.Code,
					"title":     hit.Title,
					"provider":  hit.Provider,
					"inLibrary": false,
				}
				if hit.Homepage != "" {
					item["homepage"] = hit.Homepage
				}
				if hit.Score != 0 {
					item["score"] = hit.Score
				}
				key := moviecode.NormalizeForStorageID(hit.Code)
				if movie, ok := local[key]; ok {
					item["inLibrary"] = true
					item["movieId"] = movie.ID
				}
				items = append(items, item)
			}
			truncated := len(hits) >= limit
			return core.Result{
				OK: true,
				Data: wrapSource(map[string]any{
					"query": map[string]any{
						"actorName": actorName,
						"code":      queryCode,
						"keyword":   keyword,
					},
					"items": items,
				}),
				Truncated: truncated,
			}, nil
		},
	}
}
