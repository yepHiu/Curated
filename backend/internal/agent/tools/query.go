package tools

import (
	"context"
	"strconv"
	"strings"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

// LibraryQuery is the read surface tools may call. Implemented by app.App.
type LibraryQuery interface {
	AgentLibraryOverview(ctx context.Context) (map[string]any, error)
	ListMovies(ctx context.Context, req contracts.ListMoviesRequest) (contracts.MoviesPageDTO, error)
	GetMovieDetail(ctx context.Context, movieID string) (contracts.MovieDetailDTO, error)
	GetMovieComment(ctx context.Context, movieID string) (contracts.MovieCommentDTO, error)
	GetPlaybackProgress(ctx context.Context, movieID string) (positionSec, durationSec float64, updatedAt string, ok bool, err error)
	ListActors(ctx context.Context, req contracts.ListActorsRequest) (contracts.ListActorsResponse, error)
	GetActorProfile(ctx context.Context, name string) (contracts.ActorProfileDTO, error)
	GetPersonalInsightsOverview(ctx context.Context, rangeValue, timezone string) (contracts.PersonalInsightsOverviewDTO, error)
	GetPersonalInsightsBreakdown(ctx context.Context, rangeValue, timezone, dimension string, limit int) (contracts.PersonalInsightsBreakdownDTO, error)
	ListWatchHistory(ctx context.Context, limit int) ([]WatchHistoryItem, error)
	QueryCuratedFrames(ctx context.Context, q, actor, movieID, tag string, limit, offset int) (contracts.CuratedFramesListDTO, error)
	CountCuratedFrames(ctx context.Context) (int, error)
	GetTask(ctx context.Context, taskID string) (contracts.TaskDTO, bool)
}

type WatchHistoryItem struct {
	MovieID     string  `json:"movieId"`
	Title       string  `json:"title,omitempty"`
	Code        string  `json:"code,omitempty"`
	PositionSec float64 `json:"positionSec"`
	DurationSec float64 `json:"durationSec"`
	UpdatedAt   string  `json:"updatedAt"`
}

func RegisterQueryTools(reg *core.Registry, query LibraryQuery) error {
	defs := []core.ToolDefinition{
		resolveEntities(query),
		getLibraryOverview(query),
		searchMovies(query),
		getMovieDetail(query),
		listActors(query),
		getActorProfile(query),
		getInsightsOverview(query),
		getInsightsBreakdown(query),
		getWatchHistory(query),
		searchCuratedFrames(query),
		getCuratedFramesStats(query),
		getTaskStatus(query),
	}
	for _, def := range defs {
		if err := reg.Register(def); err != nil {
			return err
		}
	}
	return nil
}

func object(props map[string]core.Schema, required ...string) core.Schema {
	falseVal := false
	return core.Schema{Type: "object", Properties: props, Required: required, AdditionalProperties: &falseVal}
}

func strField(desc string) core.Schema {
	return core.Schema{Type: "string", Description: desc}
}

func intField(desc string, min, max float64) core.Schema {
	return core.Schema{Type: "integer", Description: desc, Minimum: &min, Maximum: &max}
}

func numField(desc string, min, max float64) core.Schema {
	return core.Schema{Type: "number", Description: desc, Minimum: &min, Maximum: &max}
}

func strArg(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func intArg(args map[string]any, key string, fallback int) int {
	v, ok := args[key]
	if !ok || v == nil {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(n))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func boolPtrArg(args map[string]any, key string) *bool {
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
}

func floatPtrArg(args map[string]any, key string) *float64 {
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	n, ok := v.(float64)
	if !ok {
		return nil
	}
	return &n
}

func clampLimit(n int) int {
	if n <= 0 {
		return 20
	}
	if n > core.DefaultListLimit {
		return core.DefaultListLimit
	}
	return n
}

func wrapSource(data any) map[string]any {
	return map[string]any{"source": data}
}
