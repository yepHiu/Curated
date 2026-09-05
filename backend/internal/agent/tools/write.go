package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

// LibraryWrite is the user-data write surface tools may call. Implemented by app.App.
type LibraryWrite interface {
	MovieExists(ctx context.Context, movieID string) (bool, error)
	GetMovieComment(ctx context.Context, movieID string) (contracts.MovieCommentDTO, error)
	UpsertMovieComment(ctx context.Context, movieID, body string, expected ...string) (contracts.MovieCommentDTO, error)
	GetMovieDetail(ctx context.Context, movieID string) (contracts.MovieDetailDTO, error)
	PatchMovieDisplayOverrides(ctx context.Context, movieID string, patch contracts.PatchMovieInput) (contracts.MovieDetailDTO, error)
	CreateSavedView(ctx context.Context, name string, filters contracts.SavedViewFiltersV1) (contracts.SavedViewDTO, error)
}

func RegisterWriteTools(reg *core.Registry, write LibraryWrite) error {
	defs := []core.ToolDefinition{
		saveMovieComment(write),
		updateMovieDisplayOverrides(write),
		createSavedView(write),
	}
	for _, def := range defs {
		if err := reg.Register(def); err != nil {
			return err
		}
	}
	return nil
}

func saveMovieComment(w LibraryWrite) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"movieId": strField("Movie id whose user comment to replace"),
		"body":    strField("Replacement comment text; empty clears the note"),
	}, "movieId", "body")
	preview := func(ctx context.Context, call core.Call) (core.Result, error) {
		return previewSaveMovieComment(ctx, w, call)
	}
	apply := func(ctx context.Context, call core.Call) (core.Result, error) {
		return applySaveMovieComment(ctx, w, call)
	}
	return core.ToolDefinition{
		Name:         core.SaveMovieCommentName,
		Description:  "Propose replacing the user comment/note for one movie (same UPSERT semantics as PUT /comment). Does not write until the user confirms the preview. Never invent facts; pass the exact body the user or an action preset produced.",
		ParamsSchema: schema,
		Permission:   core.PermissionWritePreview,
		Domain:       core.DomainUserWrite,
		Handler:      preview,
		Apply:        apply,
	}
}

func previewSaveMovieComment(ctx context.Context, w LibraryWrite, call core.Call) (core.Result, error) {
	args, err := decodeWriteArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	id := strArg(args, "movieId")
	body := strArg(args, "body")
	if id == "" {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "movieId is required"}}, nil
	}
	if utf8.RuneCountInString(body) > contracts.MaxMovieCommentRunes {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "comment body too long"}}, nil
	}
	exists, err := w.MovieExists(ctx, id)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
	}
	if !exists {
		return core.Result{Error: &core.ToolError{Code: "COMMON_NOT_FOUND", Message: "movie not found"}}, nil
	}
	current, err := w.GetMovieComment(ctx, id)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
	}
	if current.Body == body {
		return core.Result{OK: true, Data: map[string]any{"movieId": id, "noop": true}}, nil
	}
	return core.Result{
		OK:            true,
		Preconditions: []core.Change{{Path: "comment.body", Before: current.Body}},
		Data: map[string]any{
			"movieId": id,
		},
		Changes: []core.Change{{
			Path:   "comment.body",
			Before: current.Body,
			After:  body,
		}},
	}, nil
}

func applySaveMovieComment(ctx context.Context, w LibraryWrite, call core.Call) (core.Result, error) {
	args, err := decodeWriteArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	id := strArg(args, "movieId")
	body := strArg(args, "body")
	before, valid := previewBefore(call, "comment.body")
	if !valid {
		return conflictResult(), nil
	}
	dto, err := w.UpsertMovieComment(ctx, id, body, before)
	if err != nil {
		if errors.Is(err, storage.ErrAIWriteConflict) {
			return conflictResult(), nil
		}
		if errors.Is(err, storage.ErrMovieNotFound) {
			return core.Result{Error: &core.ToolError{Code: "COMMON_NOT_FOUND", Message: "movie not found"}}, nil
		}
		if errors.Is(err, storage.ErrMovieCommentTooLong) {
			return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "comment body too long"}}, nil
		}
		return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
	}
	return core.Result{OK: true, Data: dto}, nil
}

func updateMovieDisplayOverrides(w LibraryWrite) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"movieId":     strField("Movie id whose display overrides to change"),
		"userTitle":   strField("Replacement display title written to user_title; omit to leave title unchanged"),
		"userSummary": strField("Replacement display summary written to user_summary; omit to leave summary unchanged"),
	}, "movieId")
	preview := func(ctx context.Context, call core.Call) (core.Result, error) {
		return previewUpdateMovieDisplay(ctx, w, call)
	}
	apply := func(ctx context.Context, call core.Call) (core.Result, error) {
		return applyUpdateMovieDisplay(ctx, w, call)
	}
	return core.ToolDefinition{
		Name:         core.UpdateMovieDisplayOverridesName,
		Description:  "Propose updating user_title and/or user_summary display overrides. Never mutates scraped title/summary columns. Does not write until the user confirms.",
		ParamsSchema: schema,
		Permission:   core.PermissionWritePreview,
		Domain:       core.DomainUserWrite,
		Handler:      preview,
		Apply:        apply,
	}
}

func previewUpdateMovieDisplay(ctx context.Context, w LibraryWrite, call core.Call) (core.Result, error) {
	args, err := decodeWriteArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	id := strArg(args, "movieId")
	if id == "" {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "movieId is required"}}, nil
	}
	titleSet, title := optionalStringArg(args, "userTitle")
	summarySet, summary := optionalStringArg(args, "userSummary")
	if !titleSet && !summarySet {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "userTitle or userSummary is required"}}, nil
	}
	if summarySet && len(summary) > core.MaxMovieSummaryBytes {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "userSummary too long"}}, nil
	}
	exists, err := w.MovieExists(ctx, id)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
	}
	if !exists {
		return core.Result{Error: &core.ToolError{Code: "COMMON_NOT_FOUND", Message: "movie not found"}}, nil
	}
	current, err := w.GetMovieDetail(ctx, id)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
	}
	var changes []core.Change
	if titleSet && title != current.Title {
		changes = append(changes, core.Change{Path: "display.userTitle", Before: current.Title, After: title})
	}
	if summarySet && summary != current.Summary {
		changes = append(changes, core.Change{Path: "display.userSummary", Before: current.Summary, After: summary})
	}
	if len(changes) == 0 {
		return core.Result{OK: true, Data: map[string]any{"movieId": id, "noop": true}}, nil
	}
	var conditions []core.Change
	if titleSet {
		conditions = append(conditions, core.Change{Path: "display.userTitle", Before: current.Title})
	}
	if summarySet {
		conditions = append(conditions, core.Change{Path: "display.userSummary", Before: current.Summary})
	}
	return core.Result{OK: true, Data: map[string]any{"movieId": id}, Changes: changes, Preconditions: conditions}, nil
}

func applyUpdateMovieDisplay(ctx context.Context, w LibraryWrite, call core.Call) (core.Result, error) {
	args, err := decodeWriteArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	id := strArg(args, "movieId")
	titleSet, title := optionalStringArg(args, "userTitle")
	summarySet, summary := optionalStringArg(args, "userSummary")
	patch := contracts.PatchMovieInput{}
	if titleSet {
		before, valid := previewBefore(call, "display.userTitle")
		if !valid {
			return conflictResult(), nil
		}
		patch.ExpectedTitle = &before
	}
	if summarySet {
		before, valid := previewBefore(call, "display.userSummary")
		if !valid {
			return conflictResult(), nil
		}
		patch.ExpectedSummary = &before
	}
	if titleSet {
		patch.UserTitleSet = true
		if title == "" {
			patch.UserTitleClear = true
		} else {
			patch.UserTitle = title
		}
	}
	if summarySet {
		patch.UserSummarySet = true
		if summary == "" {
			patch.UserSummaryClear = true
		} else {
			patch.UserSummary = summary
		}
	}
	dto, err := w.PatchMovieDisplayOverrides(ctx, id, patch)
	if err != nil {
		if errors.Is(err, storage.ErrAIWriteConflict) {
			return conflictResult(), nil
		}
		if errors.Is(err, storage.ErrMovieNotFound) || errors.Is(err, storage.ErrMovieNotFoundForPatch) {
			return core.Result{Error: &core.ToolError{Code: "COMMON_NOT_FOUND", Message: "movie not found"}}, nil
		}
		return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
	}
	return core.Result{OK: true, Data: dto}, nil
}

func previewBefore(call core.Call, path string) (string, bool) {
	for _, condition := range call.Preconditions {
		if condition.Path == path {
			before, ok := condition.Before.(string)
			return before, ok
		}
	}
	return "", false
}

func conflictResult() core.Result {
	return core.Result{Error: &core.ToolError{Code: "AI_WRITE_CONFLICT", Message: storage.ErrAIWriteConflict.Error()}}
}

func createSavedView(w LibraryWrite) core.ToolDefinition {
	filters := object(map[string]core.Schema{
		"schemaVersion":   intField("Must be 1; omitted defaults to 1", 1, 1),
		"mode":            core.Schema{Type: "string", Enum: []string{"library", "favorites", "recent", "tags", "trash"}},
		"q":               strField("Search query"),
		"tag":             strField("Comma-separated tags, AND"),
		"actor":           strField("Comma-separated actors, AND"),
		"studio":          strField("Comma-separated studios, OR"),
		"tab":             core.Schema{Type: "string", Enum: []string{"all", "new", "top-rated"}},
		"playState":       core.Schema{Type: "string", Enum: []string{"all", "unwatched", "in-progress", "completed"}},
		"userRating":      numField("Minimum local user rating 0-5", 0, 5),
		"unrated":         core.Schema{Type: "boolean", Description: "Only movies without a local rating"},
		"resolution":      strField("Normalized resolution such as 1080p or 4k"),
		"addedWithinDays": intField("Relative added window in days", 1, 3650),
		"year":            strField("Year YYYY or unknown"),
		"runtime":         core.Schema{Type: "string", Enum: []string{"short", "standard", "long"}, Description: "short (<90m), standard (90-150m), long (>150m); minute counts are accepted and mapped"},
		"catalog":         core.Schema{Type: "string", Enum: []string{"unscraped", "no-cover"}},
		"sort":            core.Schema{Type: "string", Enum: []string{"added", "release", "rating", "code", "actor", "studio", "year"}},
	})
	schema := object(map[string]core.Schema{
		"name":    strField("Saved view display name"),
		"filters": filters,
	}, "name", "filters")
	preview := func(ctx context.Context, call core.Call) (core.Result, error) {
		return previewCreateSavedView(ctx, w, call)
	}
	apply := func(ctx context.Context, call core.Call) (core.Result, error) {
		return applyCreateSavedView(ctx, w, call)
	}
	return core.ToolDefinition{
		Name:          core.CreateSavedViewName,
		Description:   "Propose creating a Saved View from canonical library filters v1. schemaVersion defaults to 1. runtime may be short/standard/long or a minute count. Never include navigation fields such as selected, from, browse, back, autoplay, or t. Does not write until the user confirms. If the request cannot be expressed with current filters, explain what is missing instead of calling this tool.",
		ParamsSchema:  schema,
		Permission:    core.PermissionWritePreview,
		Domain:        core.DomainUserWrite,
		Handler:       preview,
		Apply:         apply,
		NormalizeArgs: normalizeCreateSavedViewArgs,
	}
}

func previewCreateSavedView(ctx context.Context, w LibraryWrite, call core.Call) (core.Result, error) {
	name, filters, err := decodeSavedViewArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	_ = ctx
	_ = w
	return core.Result{
		OK: true,
		Data: map[string]any{
			"name":    name,
			"filters": filters,
		},
		Changes: []core.Change{
			{Path: "savedView.name", Before: "", After: name},
			{Path: "savedView.filters", Before: nil, After: filters},
		},
	}, nil
}

func applyCreateSavedView(ctx context.Context, w LibraryWrite, call core.Call) (core.Result, error) {
	name, filters, err := decodeSavedViewArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	dto, err := w.CreateSavedView(ctx, name, filters)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrSavedViewNameConflict):
			return core.Result{Error: &core.ToolError{Code: "SAVED_VIEW_NAME_CONFLICT", Message: err.Error()}}, nil
		case errors.Is(err, storage.ErrSavedViewLimit):
			return core.Result{Error: &core.ToolError{Code: "SAVED_VIEW_LIMIT_REACHED", Message: err.Error()}}, nil
		default:
			return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
		}
	}
	return core.Result{OK: true, Data: dto}, nil
}

func decodeSavedViewArgs(raw json.RawMessage) (string, contracts.SavedViewFiltersV1, error) {
	args, err := decodeWriteArgs(raw)
	if err != nil {
		return "", contracts.SavedViewFiltersV1{}, err
	}
	name, err := storage.NormalizeSavedViewName(strArg(args, "name"))
	if err != nil {
		return "", contracts.SavedViewFiltersV1{}, err
	}
	rawFilters, err := json.Marshal(args["filters"])
	if err != nil {
		return "", contracts.SavedViewFiltersV1{}, err
	}
	var filters contracts.SavedViewFiltersV1
	if err := json.Unmarshal(rawFilters, &filters); err != nil {
		return "", contracts.SavedViewFiltersV1{}, err
	}
	filters, err = storage.NormalizeSavedViewFilters(filters)
	if err != nil {
		return "", contracts.SavedViewFiltersV1{}, err
	}
	return name, filters, nil
}

func normalizeCreateSavedViewArgs(raw json.RawMessage) (json.RawMessage, error) {
	args, err := decodeWriteArgs(raw)
	if err != nil {
		return nil, err
	}
	filtersMap, err := savedViewFiltersMap(args["filters"])
	if err != nil {
		return nil, err
	}
	if _, hasQ := filtersMap["q"]; !hasQ {
		if query, ok := filtersMap["query"]; ok {
			filtersMap["q"] = query
		}
	}
	delete(filtersMap, "query")

	if sv, ok := coerceJSONInt(filtersMap["schemaVersion"]); ok && sv > 0 {
		filtersMap["schemaVersion"] = sv
	} else {
		filtersMap["schemaVersion"] = 1
	}

	if v, ok := filtersMap["runtime"]; ok {
		if bucket := coerceSavedViewRuntime(v); bucket != "" {
			filtersMap["runtime"] = bucket
		} else {
			delete(filtersMap, "runtime")
		}
	}
	if v, ok := filtersMap["addedWithinDays"]; ok {
		if n, ok := coerceJSONInt(v); ok && n >= 1 && n <= 3650 {
			filtersMap["addedWithinDays"] = n
		} else {
			delete(filtersMap, "addedWithinDays")
		}
	}
	if v, ok := filtersMap["userRating"]; ok {
		if n, ok := coerceJSONFloat(v); ok {
			filtersMap["userRating"] = n
		} else {
			delete(filtersMap, "userRating")
		}
	}

	allowed := map[string]struct{}{
		"schemaVersion": {}, "mode": {}, "q": {}, "tag": {}, "actor": {}, "studio": {},
		"tab": {}, "playState": {}, "userRating": {}, "unrated": {}, "resolution": {},
		"addedWithinDays": {}, "year": {}, "runtime": {}, "catalog": {}, "sort": {},
	}
	for key := range filtersMap {
		if _, ok := allowed[key]; !ok {
			delete(filtersMap, key)
		}
	}

	out := map[string]any{"filters": filtersMap}
	if name := strArg(args, "name"); name != "" {
		out["name"] = name
	} else if v, ok := args["name"]; ok {
		out["name"] = v
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func savedViewFiltersMap(raw any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}
	if s, ok := raw.(string); ok {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return map[string]any{}, nil
		}
		var parsed any
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return nil, fmt.Errorf("filters must be an object")
		}
		raw = parsed
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("filters must be an object")
	}
	return obj, nil
}

func coerceJSONInt(v any) (int, bool) {
	n, ok := coerceJSONFloat(v)
	if !ok || n != float64(int64(n)) {
		return 0, false
	}
	return int(n), true
}

func coerceJSONFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func coerceSavedViewRuntime(v any) string {
	switch t := v.(type) {
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		if s == "short" || s == "standard" || s == "long" {
			return s
		}
		if n, err := strconv.Atoi(s); err == nil {
			return runtimeMinutesBucket(n)
		}
	case float64:
		return runtimeMinutesBucket(int(t))
	case json.Number:
		parsed, err := t.Int64()
		if err == nil {
			return runtimeMinutesBucket(int(parsed))
		}
	case int:
		return runtimeMinutesBucket(t)
	}
	return ""
}

func runtimeMinutesBucket(minutes int) string {
	if minutes <= 0 {
		return ""
	}
	if minutes < 90 {
		return "short"
	}
	if minutes <= 150 {
		return "standard"
	}
	return "long"
}

func optionalStringArg(args map[string]any, key string) (bool, string) {
	v, ok := args[key]
	if !ok || v == nil {
		return false, ""
	}
	s, ok := v.(string)
	if !ok {
		return false, ""
	}
	return true, s
}

func decodeWriteArgs(raw json.RawMessage) (map[string]any, error) {
	trimmed := string(raw)
	if trimmed == "" {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	if args == nil {
		args = map[string]any{}
	}
	return args, nil
}
