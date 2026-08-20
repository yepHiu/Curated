package tools

import (
	"context"
	"fmt"
	"strings"

	"curated-backend/internal/agent/core"
)

func RegisterPresentTools(reg *core.Registry, refs *core.MovieRefStore) error {
	if err := reg.Register(presentMovies(refs)); err != nil {
		return err
	}
	return nil
}

func presentMovies(refs *core.MovieRefStore) core.ToolDefinition {
	itemSchema := object(map[string]core.Schema{
		"movieId": strField("Movie id from search_movies, get_movie_detail, or get_watch_history in this turn"),
		"reason":  {Type: "string", Description: "One-line reason using retrieved fields only", MaxLength: 200},
	}, "movieId")
	schema := object(map[string]core.Schema{
		"items": {
			Type:        "array",
			Description: "Movies to show the user as cards, max 6. IDs must already have been retrieved.",
			MinItems:    1,
			MaxItems:    core.PresentMoviesMaxItems,
			Items:       &itemSchema,
		},
	}, "items")
	return core.ToolDefinition{
		Name: core.PresentMoviesName,
		Description: "Show up to 6 movie cards in the chat UI. Call this when recommending or pointing at specific titles. " +
			"movieId values must come from search_movies, get_movie_detail, or get_watch_history in this turn. " +
			"Never invent IDs. Optional reason is one sentence citing retrieved fields (actors, runtime, tags, rating).",
		ParamsSchema: schema,
		Permission:   core.PermissionRead,
		Domain:       core.DomainPresent,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			requested, ids := presentMovieItems(args["items"])
			if len(ids) == 0 {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: "items must include at least one movieId",
				}}, nil
			}
			found, missing := refs.Lookup(call.SessionID, ids)
			if len(missing) > 0 {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: fmt.Sprintf("unknown movie ids: %s; retrieve them with search_movies or get_movie_detail first", strings.Join(missing, ", ")),
				}}, nil
			}
			cards := make([]map[string]any, 0, len(found))
			for _, ref := range found {
				card := map[string]any{
					"id":       ref.ID,
					"movieId":  ref.ID,
					"title":    ref.Title,
					"code":     ref.Code,
					"actors":   ref.Actors,
					"coverUrl": ref.CoverURL,
					"thumbUrl": ref.ThumbURL,
				}
				if reason := requested[ref.ID]; reason != "" {
					card["reason"] = reason
				}
				cards = append(cards, card)
			}
			return core.Result{OK: true, Data: wrapSource(map[string]any{"items": cards})}, nil
		},
	}
}

func presentMovieItems(raw any) (map[string]string, []string) {
	arr, ok := raw.([]any)
	if !ok {
		return map[string]string{}, nil
	}
	reasons := map[string]string{}
	ids := make([]string, 0, len(arr))
	seen := map[string]bool{}
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := strArg(obj, "movieId")
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		if reason := strArg(obj, "reason"); reason != "" {
			reasons[id] = reason
		}
	}
	return reasons, ids
}
