package app

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestAIWritePreviewRejectsInterveningEdit(t *testing.T) {
	for _, field := range []string{"comment", "title", "summary", "unchanged-title"} {
		t.Run(field, func(t *testing.T) {
			ctx := context.Background()
			store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "conflict.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			if err := store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			movie, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "test", Path: "D:/fixtures/ABC-001.mp4", FileName: "ABC-001.mp4", Number: "ABC-001"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.UpsertMovieComment(ctx, movie.MovieID, "old note"); err != nil {
				t.Fatal(err)
			}
			if err := store.PatchMovieUserPrefs(ctx, movie.MovieID, contracts.PatchMovieInput{UserTitleSet: true, UserTitle: "old title", UserSummarySet: true, UserSummary: "old summary"}); err != nil {
				t.Fatal(err)
			}
			a := &App{store: store}
			tool := core.UpdateMovieDisplayOverridesName
			args := map[string]string{"movieId": movie.MovieID}
			switch field {
			case "comment":
				tool = core.SaveMovieCommentName
				args["body"] = "AI note"
			case "title":
				args["userTitle"] = "AI title"
			case "summary":
				args["userSummary"] = "AI summary"
			case "unchanged-title":
				args["userTitle"] = "old title"
				args["userSummary"] = "AI summary"
			}
			encoded, err := json.Marshal(args)
			if err != nil {
				t.Fatal(err)
			}
			preview := a.ensureAgentGateway().Invoke(ctx, core.Call{Name: tool, Args: encoded, SessionID: "action", Channel: core.ChannelAction})
			if !preview.OK || preview.ConfirmToken == "" {
				t.Fatalf("preview: %+v", preview)
			}
			if field == "comment" {
				if _, err := store.UpsertMovieComment(ctx, movie.MovieID, "manual note"); err != nil {
					t.Fatal(err)
				}
			} else {
				patch := contracts.PatchMovieInput{UserTitleSet: true, UserTitle: "manual title"}
				if field == "summary" {
					patch = contracts.PatchMovieInput{UserSummarySet: true, UserSummary: "manual summary"}
				}
				if err := store.PatchMovieUserPrefs(ctx, movie.MovieID, patch); err != nil {
					t.Fatal(err)
				}
			}
			request := contracts.AIToolApplyRequest{Name: tool, SessionID: "action", Arguments: encoded, ConfirmToken: preview.ConfirmToken}
			_, err = a.ApplyAITool(ctx, request)
			var toolErr *core.ToolError
			if !errors.As(err, &toolErr) || toolErr.Code != "AI_WRITE_CONFLICT" {
				t.Fatalf("expected conflict, got %v", err)
			}
			if field == "comment" {
				note, err := store.GetMovieComment(ctx, movie.MovieID)
				if err != nil || note.Body != "manual note" {
					t.Fatalf("overwritten note: %+v %v", note, err)
				}
			} else {
				detail, err := store.GetMovieDetail(ctx, movie.MovieID)
				if err != nil {
					t.Fatal(err)
				}
				if field == "summary" && detail.Summary != "manual summary" {
					t.Fatalf("overwritten summary: %q", detail.Summary)
				}
				if field != "summary" && (detail.Title != "manual title" || detail.Summary != "old summary") {
					t.Fatalf("non-atomic conflict: %s / %s", detail.Title, detail.Summary)
				}
			}
			_, err = a.ApplyAITool(ctx, request)
			if !errors.As(err, &toolErr) || toolErr.Code != "AI_CONFIRM_EXPIRED" {
				t.Fatalf("ticket reused: %v", err)
			}
			fresh := a.ensureAgentGateway().Invoke(ctx, core.Call{Name: tool, Args: encoded, SessionID: "action", Channel: core.ChannelAction})
			request.ConfirmToken = fresh.ConfirmToken
			if result, err := a.ApplyAITool(ctx, request); err != nil || !result.OK {
				t.Fatalf("fresh preview cannot apply: %+v %v", result, err)
			}
			if _, err := a.ApplyAITool(ctx, request); err == nil {
				t.Fatal("successful ticket reused")
			}
		})
	}
}
