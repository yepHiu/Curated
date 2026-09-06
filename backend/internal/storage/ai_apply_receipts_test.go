package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

func TestAIReceiptFailureRollsBackBusinessWrite(t *testing.T) {
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "atomic.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	movie, err := s.PersistScanMovie(ctx, contracts.ScanFileResultDTO{TaskID: "t", Path: "D:/test/A-001.mp4", FileName: "A-001.mp4", Number: "A-001"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpsertMovieComment(ctx, movie.MovieID, "original"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_receipt BEFORE INSERT ON ai_apply_receipts BEGIN SELECT RAISE(ABORT,'receipt unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	key := NewAIApplyReceiptKey("token", "session", "save_movie_comment", "hash")
	if _, err := s.UpsertMovieComment(WithAIApplyReceipt(ctx, key), movie.MovieID, "new", "original"); err == nil {
		t.Fatal("receipt failure ignored")
	}
	note, err := s.GetMovieComment(ctx, movie.MovieID)
	if err != nil || note.Body != "original" {
		t.Fatalf("write did not roll back: %+v %v", note, err)
	}
	if _, found, err := s.GetAIApplyReceipt(ctx, key); err != nil || found {
		t.Fatalf("receipt for failed write: %v %v", found, err)
	}
	before, err := s.GetMovieDetail(ctx, movie.MovieID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PatchMovieUserPrefs(WithAIApplyReceipt(ctx, key), movie.MovieID, contracts.PatchMovieInput{UserTitleSet: true, UserTitle: "new title"}); err == nil {
		t.Fatal("display receipt failure ignored")
	}
	after, err := s.GetMovieDetail(ctx, movie.MovieID)
	if err != nil || after.Title != before.Title {
		t.Fatal("display write did not roll back")
	}
	if _, err := s.CreateSavedView(WithAIApplyReceipt(ctx, key), "view-id", "new view", contracts.SavedViewFiltersV1{SchemaVersion: 1, Mode: "library"}, time.Now()); err == nil {
		t.Fatal("saved view receipt failure ignored")
	}
	views, err := s.ListSavedViews(ctx)
	if err != nil || len(views) != 0 {
		t.Fatal("saved view write did not roll back")
	}
}
