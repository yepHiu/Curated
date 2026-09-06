package tools

import (
	"context"
	"strings"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

func listActors(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "list_actors",
		Description: "List actors that appear in the library. q matches actor name or actor user tags. actorTag is an exact actor user tag. sort is name or movieCount.",
		ParamsSchema: object(map[string]core.Schema{
			"q":        strField("Substring match on actor name or actor user tags"),
			"actorTag": strField("Exact actor user tag"),
			"sort":     core.Schema{Type: "string", Enum: []string{"name", "movieCount"}},
			"limit":    intField("Page size, max 50", 1, 50),
			"offset":   intField("Page offset", 0, 100000),
		}),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			limit := clampLimit(intArg(args, "limit", 20))
			offset := intArg(args, "offset", 0)
			page, err := q.ListActors(ctx, contracts.ListActorsRequest{
				Q:        strArg(args, "q"),
				ActorTag: strArg(args, "actorTag"),
				Sort:     strArg(args, "sort"),
				Limit:    limit,
				Offset:   offset,
			})
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
			}
			items := make([]map[string]any, 0, len(page.Actors))
			for _, actor := range page.Actors {
				items = append(items, map[string]any{
					"name":       actor.Name,
					"movieCount": actor.MovieCount,
					"userTags":   actor.UserTags,
				})
			}
			next, truncated := core.PageCursor(offset, limit, page.Total)
			return core.Result{
				OK:         true,
				Data:       wrapSource(map[string]any{"total": page.Total, "items": items, "query": map[string]any{"q": strArg(args, "q"), "actorTag": strArg(args, "actorTag"), "sort": strArg(args, "sort")}}),
				Truncated:  truncated,
				NextCursor: next,
			}, nil
		},
	}
}

func getActorProfile(q LibraryQuery) core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "get_actor_profile",
		Description: "Get one actor profile by display name or alias. Includes summary, scraped homepage, aliases, user tags, and user external links.",
		ParamsSchema: object(map[string]core.Schema{
			"name": strField("Actor display name or alias"),
		}, "name"),
		Permission: core.PermissionRead,
		Domain:     core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			args := decodeArgs(call.Args)
			profile, err := q.GetActorProfile(ctx, strArg(args, "name"))
			if err != nil {
				return core.Result{OK: false, Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "actor not found"}}, nil
			}
			payload := map[string]any{
				"name":           profile.Name,
				"summary":        profile.Summary,
				"aliases":        profile.Aliases,
				"userTags":       profile.UserTags,
				"externalLinks":  profile.ExternalLinks,
				"hasLocalAvatar": profile.HasLocalAvatar,
			}
			if homepage := strings.TrimSpace(profile.Homepage); homepage != "" {
				payload["homepage"] = homepage
			}
			return core.Result{OK: true, Data: wrapSource(payload)}, nil
		},
	}
}
