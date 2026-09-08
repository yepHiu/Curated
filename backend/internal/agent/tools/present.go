package tools

import (
	"context"
	"fmt"
	"strings"

	"curated-backend/internal/agent/core"
)

// RegisterPresentTools 注册结构化终结回答与兼容卡片工具。
func RegisterPresentTools(reg *core.Registry, refs *core.MovieRefStore) error {
	if err := reg.Register(core.SubmitAnswerTool()); err != nil {
		return err
	}
	if err := reg.Register(presentMovies(refs)); err != nil {
		return err
	}
	return nil
}

// presentMovies 从本轮本地读取快照生成卡片，忽略模型自由推荐理由。
func presentMovies(refs *core.MovieRefStore) core.ToolDefinition {
	itemSchema := object(map[string]core.Schema{
		"movieId": strField("Movie id from search_movies, get_movie_detail, or get_watch_history in this turn"),
		"reason":  {Type: "string", Description: "Deprecated; not displayed. Use submit_answer with reasonFacts for recorded facts.", MaxLength: 200},
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
			"movieId needs a local read from search_movies, get_movie_detail or get_watch_history in this request; page IDs and provider rows alone are insufficient. " +
			"Never invent IDs. Off-library provider titles have no movieId. Prefer submit_answer to finish recommendations with server-rendered facts. reason is deprecated and is not displayed.",
		ParamsSchema: schema,
		Permission:   core.PermissionRead,
		Domain:       core.DomainPresent,
		// 先验证实体锚点，再从请求内本地快照填入字段，不复用来源站字段。
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			_, ids := presentMovieItems(args["items"])
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
				if store := core.AnswerRefsFromContext(ctx); store != nil {
					snapshot, ok := store.LocalMovie(ref.ID)
					if !ok {
						return core.Result{Error: &core.ToolError{Code: "AI_ANSWER_REJECTED", Message: "Read local movie details before presenting a card."}}, nil
					}
					card := snapshot.Fields
					card["id"], card["movieId"] = snapshot.MovieID, snapshot.MovieID
					cards = append(cards, card)
					continue
				}
				card := map[string]any{
					"id":       ref.ID,
					"movieId":  ref.ID,
					"title":    ref.Title,
					"code":     ref.Code,
					"actors":   ref.Actors,
					"coverUrl": ref.CoverURL,
					"thumbUrl": ref.ThumbURL,
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
