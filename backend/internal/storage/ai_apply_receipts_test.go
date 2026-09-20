package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

func TestPhotoAIReceiptFailureRollsBackBusinessWrite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newComicRepositoryTestStore(t, t.TempDir())
	book, err := s.UpsertPhotoBook(ctx, PhotoBookUpsert{
		Location: filepath.Join(t.TempDir(), "book.cbz"), Title: "Original title", PageCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.UpsertPhotoComment(ctx, book.ID, "Original note")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_receipt BEFORE INSERT ON ai_apply_receipts BEGIN SELECT RAISE(ABORT,'receipt unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"save_photo_comment", "update_photo_title"} {
		t.Run(tool, func(t *testing.T) {
			key := NewAIApplyReceiptKey("token", "session", tool, "hash")
			applyCtx := WithAIApplyReceipt(ctx, key)
			var err error
			if tool == "save_photo_comment" {
				_, err = s.UpsertPhotoComment(applyCtx, book.ID, "AI note", before.Body)
			} else {
				title := "AI title"
				_, err = s.PatchPhotoBook(applyCtx, book.ID, contracts.PatchPhotoBookRequest{Title: &title, ExpectedTitle: &book.Title})
			}
			if err == nil {
				t.Fatal("receipt failure ignored")
			}
			note, err := s.GetPhotoComment(ctx, book.ID)
			if err != nil || note != before {
				t.Fatalf("note did not roll back: %+v %v", note, err)
			}
			detail, err := s.GetPhotoBookDetail(ctx, book.ID)
			if err != nil || detail.Title != book.Title {
				t.Fatalf("title did not roll back: %+v %v", detail, err)
			}
			if _, found, err := s.GetAIApplyReceipt(ctx, key); err != nil || found {
				t.Fatalf("receipt for failed write: %v %v", found, err)
			}
		})
	}
}

func TestComicAIReceiptFailureRollsBackBusinessWrite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newComicRepositoryTestStore(t, t.TempDir())
	book, err := s.UpsertComicBook(ctx, ComicBookUpsert{
		Location: filepath.Join(t.TempDir(), "book.cbz"), Title: "Original title", PageCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.UpsertComicComment(ctx, book.ID, "Original note")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_receipt BEFORE INSERT ON ai_apply_receipts BEGIN SELECT RAISE(ABORT,'receipt unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"save_comic_comment", "update_comic_title"} {
		t.Run(tool, func(t *testing.T) {
			key := NewAIApplyReceiptKey("token", "session", tool, "hash")
			applyCtx := WithAIApplyReceipt(ctx, key)
			var err error
			if tool == "save_comic_comment" {
				_, err = s.UpsertComicComment(applyCtx, book.ID, "AI note", before.Body)
			} else {
				title := "AI title"
				_, err = s.PatchComicBook(applyCtx, book.ID, contracts.PatchComicBookRequest{Title: &title, ExpectedTitle: &book.Title})
			}
			if err == nil {
				t.Fatal("receipt failure ignored")
			}
			note, err := s.GetComicComment(ctx, book.ID)
			if err != nil || note != before {
				t.Fatalf("note did not roll back: %+v %v", note, err)
			}
			detail, err := s.GetComicBookDetail(ctx, book.ID)
			if err != nil || detail.Title != book.Title {
				t.Fatalf("title did not roll back: %+v %v", detail, err)
			}
			if _, found, err := s.GetAIApplyReceipt(ctx, key); err != nil || found {
				t.Fatalf("receipt for failed write: %v %v", found, err)
			}
		})
	}
}

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
