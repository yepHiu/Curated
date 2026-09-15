package tools

import (
	"context"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

func getWatchHistory(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "get_watch_history",
		Description: "Recent playback progress rows, newest first. Bounded to 50. Use search_movies for unplayed inventory.",
		ParamsSchema: object(map[string]core.Schema{
			"limit": intField("Max rows, default 20, max 50", 1, 50),
		}),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			limit := clampLimit(intArg(args, "limit", 20))
			items, err := q.ListWatchHistory(ctx, limit)
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
			}
			truncated := len(items) == limit
			return core.Result{OK: true, Data: wrapSource(map[string]any{"items": items}), Truncated: truncated}, nil
		},
	}
}

func searchCuratedFrames(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "search_curated_frames",
		Description: "Search curated stills by q, actor, movieId, or tag.",
		ParamsSchema: object(map[string]core.Schema{
			"q":       strField("Free-text query"),
			"actor":   strField("Exact actor name"),
			"movieId": strField("Movie id"),
			"tag":     strField("Exact frame tag"),
			"limit":   intField("Page size, max 50", 1, 50),
			"offset":  intField("Page offset", 0, 100000),
		}),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			limit := clampLimit(intArg(args, "limit", 20))
			offset := intArg(args, "offset", 0)
			page, err := q.QueryCuratedFrames(ctx, strArg(args, "q"), strArg(args, "actor"), strArg(args, "movieId"), strArg(args, "tag"), limit, offset)
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
			}
			items := make([]map[string]any, 0, len(page.Items))
			for _, item := range page.Items {
				items = append(items, map[string]any{
					"id":          item.ID,
					"movieId":     item.MovieID,
					"title":       item.Title,
					"code":        item.Code,
					"actors":      item.Actors,
					"positionSec": item.PositionSec,
					"tags":        item.Tags,
				})
			}
			next, truncated := core.PageCursor(offset, limit, page.Total)
			return core.Result{
				OK:         true,
				Data:       wrapSource(map[string]any{"total": page.Total, "items": items}),
				Truncated:  truncated,
				NextCursor: next,
			}, nil
		},
	}
}

func getCuratedFramesStats(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name:         "get_curated_frames_stats",
		Description:  "Total curated frame count.",
		ParamsSchema: object(nil),
		Permission:   core.PermissionRead,
		Domain:       core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			total, err := q.CountCuratedFrames(ctx)
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
			}
			return core.Result{OK: true, Data: wrapSource(map[string]any{"total": total})}, nil
		},
	}
}

func getTaskStatus(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "get_task_status",
		Description: "Look up a background task by taskId. Movie scan/scrape/import tasks are always available. Comic or photo scan/import tasks are available only when that library Beta is enabled. Do not poll in a loop; report once and finish.",
		ParamsSchema: object(map[string]core.Schema{
			"taskId": strField("Task id"),
		}, "taskId"),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			task, ok := q.GetTask(ctx, strArg(args, "taskId"))
			if !ok || !isAgentVisibleTask(q, task.Type) {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "task not found"}}, nil
			}
			return core.Result{OK: true, Data: wrapSource(map[string]any{
				"taskId":   task.TaskID,
				"type":     task.Type,
				"status":   task.Status,
				"progress": task.Progress,
				"message":  task.Message,
			})}, nil
		},
	}
}

// isMovieTask reports movie-library background jobs that Agent may always read.
func isMovieTask(kind string) bool {
	switch kind {
	case "scan.library", "scrape.movie", "scrape.actor", "movie_clip_gif", "movie_clip_mp4", "movie_clip_webm",
		contracts.TaskTypeImportMovies, contracts.TaskTypeLibraryHealthRepair, contracts.TaskTypeLibraryHealthCleanup:
		return true
	default:
		return false
	}
}

// isAgentVisibleTask allows movie tasks always, and book-library tasks only while that Beta is enabled.
func isAgentVisibleTask(q LibraryQuery, kind string) bool {
	if isMovieTask(kind) {
		return true
	}
	books, ok := q.(BookLibraryQuery)
	if !ok {
		return false
	}
	switch kind {
	case contracts.TaskTypeScanComics, contracts.TaskTypeImportComics, contracts.TaskTypeComicCacheCleanup:
		return books.ComicLibraryEnabled()
	case contracts.TaskTypeScanPhotos, contracts.TaskTypeImportPhotos:
		return books.PhotoLibraryEnabled()
	default:
		return false
	}
}
