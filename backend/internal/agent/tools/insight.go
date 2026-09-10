package tools

import (
	"context"

	"curated-backend/internal/agent/core"
)

func getInsightsOverview(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "get_insights_overview",
		Description: "Personal movie-watch overview for range 30d|90d|365d|all. These statistics describe movies only. timezone is IANA (default UTC). Completed means current saved progress >= 90%. Null rates mean the denominator is empty.",
		ParamsSchema: object(map[string]core.Schema{
			"range":    core.Schema{Type: "string", Enum: []string{"30d", "90d", "365d", "all"}, Description: "Time range"},
			"timezone": strField("IANA timezone, default UTC"),
		}),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			rangeValue := strArg(args, "range")
			if rangeValue == "" {
				rangeValue = "30d"
			}
			tz := strArg(args, "timezone")
			if tz == "" {
				tz = "UTC"
			}
			dto, err := q.GetPersonalInsightsOverview(ctx, rangeValue, tz)
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
			}
			return core.Result{OK: true, Data: wrapSource(dto)}, nil
		},
	}
}

func getInsightsBreakdown(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "get_insights_breakdown",
		Description: "Personal watch breakdown by actor, studio, or tag. full-per-entity attribution; shares may sum over 100% for multi-actor/tag titles. limit max 25.",
		ParamsSchema: object(map[string]core.Schema{
			"range":     core.Schema{Type: "string", Enum: []string{"30d", "90d", "365d", "all"}},
			"timezone":  strField("IANA timezone, default UTC"),
			"dimension": core.Schema{Type: "string", Enum: []string{"actor", "studio", "tag"}},
			"limit":     intField("Number of rows, max 25", 1, 25),
		}, "dimension"),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			rangeValue := strArg(args, "range")
			if rangeValue == "" {
				rangeValue = "30d"
			}
			tz := strArg(args, "timezone")
			if tz == "" {
				tz = "UTC"
			}
			limit := intArg(args, "limit", 10)
			if limit > 25 {
				limit = 25
			}
			dto, err := q.GetPersonalInsightsBreakdown(ctx, rangeValue, tz, strArg(args, "dimension"), limit)
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
			}
			return core.Result{OK: true, Data: wrapSource(dto)}, nil
		},
	}
}
