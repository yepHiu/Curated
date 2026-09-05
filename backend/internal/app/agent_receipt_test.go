package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func TestAIReceiptSurvivesRestartAndDoesNotRepeatWrite(t *testing.T) {
	for _, tool := range []string{core.SaveMovieCommentName, core.UpdateMovieDisplayOverridesName, core.CreateSavedViewName} {
		t.Run(tool, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "receipts.db")
			store, err := storage.NewSQLiteStore(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			if err := store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			movie, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "t", Path: "D:/fixture/T-001.mp4", FileName: "T-001.mp4", Number: "T-001"})
			if err != nil {
				t.Fatal(err)
			}
			args := map[string]any{"movieId": movie.MovieID, "body": "AI note"}
			if tool == core.UpdateMovieDisplayOverridesName {
				args = map[string]any{"movieId": movie.MovieID, "userTitle": "AI title"}
			}
			if tool == core.CreateSavedViewName {
				args = map[string]any{"name": "AI view", "filters": map[string]any{"schemaVersion": 1, "mode": "library"}}
			}
			encoded, _ := json.Marshal(args)
			a := &App{store: store, cfg: enabledAITestConfig()}
			session, err := store.CreateAIChatSession(ctx, "receipt test")
			if err != nil {
				t.Fatal(err)
			}
			preview := a.ensureAgentGateway().Invoke(ctx, core.Call{Name: tool, Args: encoded, SessionID: session.ID, Channel: core.ChannelAction})
			if !preview.OK || preview.ConfirmToken == "" {
				t.Fatalf("preview: %+v", preview)
			}
			key := storage.NewAIApplyReceiptKey(preview.ConfirmToken, session.ID, tool, core.HashArgs(preview.ConfirmArgs))
			_, err = store.AppendAIChatMessage(ctx, session.ID, "assistant", "preview", "", "", contracts.AIChatSSEEvent{Type: "confirm_required", Name: tool, ReceiptID: key.TokenHash})
			if err != nil {
				t.Fatal(err)
			}
			req := contracts.AIToolApplyRequest{Name: tool, SessionID: session.ID, Arguments: encoded, ConfirmToken: preview.ConfirmToken}
			var results [2]contracts.AIToolApplyDTO
			var failures [2]error
			var wg sync.WaitGroup
			for i := range results {
				wg.Add(1)
				go func() { defer wg.Done(); results[i], failures[i] = a.ApplyAITool(ctx, req) }()
			}
			wg.Wait()
			for _, err := range failures {
				if err != nil {
					t.Fatal(err)
				}
			}
			if results[0].Replayed == results[1].Replayed {
				t.Fatal("exactly one concurrent confirmation must replay")
			}
			first, _ := json.Marshal(results[0].Data)
			second, _ := json.Marshal(results[1].Data)
			if string(first) != string(second) {
				t.Fatal("duplicate confirmations disagree")
			}
			if tool == core.SaveMovieCommentName {
				if _, err := store.UpsertMovieComment(ctx, movie.MovieID, "later manual note"); err != nil {
					t.Fatal(err)
				}
			}
			if tool == core.UpdateMovieDisplayOverridesName {
				if err := store.PatchMovieUserPrefs(ctx, movie.MovieID, contracts.PatchMovieInput{UserTitleSet: true, UserTitle: "later manual title"}); err != nil {
					t.Fatal(err)
				}
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			store, err = storage.NewSQLiteStore(path)
			if err != nil {
				t.Fatal(err)
			}
			restarted := &App{store: store, cfg: enabledAITestConfig()}
			replay, err := restarted.ApplyAITool(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if !replay.Replayed {
				t.Fatal("restart retry must be marked replayed")
			}
			replayed, _ := json.Marshal(replay.Data)
			if string(first) != string(replayed) {
				t.Fatal("receipt changed after restart")
			}
			if tool == core.SaveMovieCommentName {
				note, err := store.GetMovieComment(ctx, movie.MovieID)
				if err != nil || note.Body != "later manual note" {
					t.Fatal("receipt replay rewrote note")
				}
			}
			if tool == core.UpdateMovieDisplayOverridesName {
				detail, err := store.GetMovieDetail(ctx, movie.MovieID)
				if err != nil || detail.Title != "later manual title" {
					t.Fatal("receipt replay rewrote title")
				}
			}
			detail, err := restarted.GetAIChatSession(ctx, session.ID)
			if err != nil || !detail.Messages[0].Events[0].Applied {
				t.Fatalf("history receipt missing: %+v %v", detail, err)
			}
			drift := req
			drift.SessionID = "another-session"
			if _, err := restarted.ApplyAITool(ctx, drift); err == nil {
				t.Fatal("cross-session receipt exposed")
			}
			drift = req
			drift.Arguments = json.RawMessage(`{"movieId":"changed","body":"changed"}`)
			if _, err := restarted.ApplyAITool(ctx, drift); err == nil {
				t.Fatal("changed arguments accepted")
			}
			if tool == core.CreateSavedViewName {
				views, err := store.ListSavedViews(ctx)
				if err != nil || len(views) != 1 {
					t.Fatal("duplicate saved view")
				}
			}
			if err := store.DeleteAIChatSession(ctx, session.ID); err != nil {
				t.Fatal(err)
			}
			if _, found, err := store.GetAIApplyReceipt(ctx, key); err != nil || found {
				t.Fatal("deleted session retained receipts")
			}
		})
	}
}
