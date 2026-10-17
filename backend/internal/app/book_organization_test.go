package app

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

func TestBookOrganizationDurableReview(t *testing.T) {
	for _, kind := range []string{"comic", "photo"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "tasks.db")
			store, err := storage.NewSQLiteStore(path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = store.Close() }()
			if err = store.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			a := &App{store: store, cfg: enabledAITestConfig(), comicLibraryEnabled: true, photoLibraryEnabled: true}
			ids := []string{}
			for _, name := range []string{"夏の記録", "冬の記録"} {
				var id string
				if kind == "comic" {
					book, e := store.UpsertComicBook(ctx, storage.ComicBookUpsert{Title: name, SourceFileName: name + ".zip", Location: filepath.Join(t.TempDir(), name+".zip"), PageCount: 1})
					id, err = book.ID, e
				} else {
					book, e := store.UpsertPhotoBook(ctx, storage.PhotoBookUpsert{Title: name, SourceFileName: name + ".zip", Location: filepath.Join(t.TempDir(), name+".zip"), PageCount: 1})
					id, err = book.ID, e
				}
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, id)
			}
			request := contracts.BookOrganizationRequest{Kind: kind, BookIDs: ids, Locale: "zh-CN", RequestID: "request"}
			if _, err = store.CreateBookOrganization(ctx, "job", request); err != nil {
				t.Fatal(err)
			}
			if id, e := store.CreateBookOrganization(ctx, "duplicate", request); e != nil || id != "job" {
				t.Fatalf("idempotency %s %v", id, e)
			}
			calls := 0
			recognize := func(ctx context.Context, req contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
				calls++
				return a.runBookMetadataAction(ctx, metadataCompleter(func(_ context.Context, _ []llm.ChatMessage, _ int) (string, error) {
					raw, _ := json.Marshal(contracts.BookMetadata{CleanTitle: "記録", TranslatedTitle: "记录"})
					return string(raw), nil
				}), req)
			}
			a.processBookOrganization(ctx, "job", recognize)
			job, err := store.GetBookOrganization(ctx, "job")
			if err != nil || job.Status != "needs_confirmation" || job.PendingReview != 2 {
				t.Fatalf("job %+v %v", job, err)
			}
			for _, id := range ids {
				state, _ := a.GetBookMetadataState(ctx, kind, id)
				if state.Title == "记录" {
					t.Fatal("generation wrote metadata")
				}
			}
			summary, err := store.TaskCenterSummary(ctx)
			if err != nil || summary.Attention != 1 {
				t.Fatalf("summary %+v %v", summary, err)
			}
			keys, _, err := store.TaskCenterKeys(ctx, storage.TaskCenterFilter{Kind: "organize", State: "attention"})
			if err != nil || len(keys) != 1 || keys[0] != "book-organize:job" {
				t.Fatalf("keys %v %v", keys, err)
			}
			// Restart with a fresh gateway: none of the original volatile tickets survive.
			if err = store.Close(); err != nil {
				t.Fatal(err)
			}
			store, err = storage.NewSQLiteStore(path)
			if err != nil {
				t.Fatal(err)
			}
			a = &App{store: store, cfg: enabledAITestConfig(), comicLibraryEnabled: true, photoLibraryEnabled: true}
			task, err := store.TaskCenterTask(ctx, "book-organize:job")
			if err != nil || task.State != "needs_confirmation" || task.Unit != "books" || task.BookOrganization == nil {
				t.Fatalf("task %+v %v", task, err)
			}
			review := contracts.BookOrganizationReview{BookID: ids[0], Revision: 99, Decision: "apply"}
			if err = a.ReviewBookOrganization(ctx, "job", review); !errors.Is(err, storage.ErrAIWriteConflict) {
				t.Fatalf("stale revision: %v", err)
			}
			review.Revision = 1
			if err = a.ReviewBookOrganization(ctx, "job", review); err != nil {
				t.Fatal(err)
			}
			applied, _ := store.BookOrganizationItem(ctx, "job", ids[0])
			if applied.Status != "applied" {
				t.Fatalf("not atomic: %+v", applied)
			}
			manual := "手动标题"
			patch := func(id string) {
				if kind == "comic" {
					_, err = store.PatchComicBook(ctx, id, contracts.PatchComicBookRequest{Title: &manual})
				} else {
					_, err = store.PatchPhotoBook(ctx, id, contracts.PatchPhotoBookRequest{Title: &manual})
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			patch(ids[0])
			patch(ids[1])
			if err = a.ReviewBookOrganization(ctx, "job", review); err != nil {
				t.Fatal(err)
			}
			state, _ := a.GetBookMetadataState(ctx, kind, ids[0])
			if state.Title != manual {
				t.Fatal("replay overwrote later manual edit")
			}
			review.BookID = ids[1]
			if err = a.ReviewBookOrganization(ctx, "job", review); err == nil {
				t.Fatal("source conflict accepted")
			}
			conflict, _ := store.BookOrganizationItem(ctx, "job", ids[1])
			if conflict.Status != "conflict" {
				t.Fatalf("conflict %+v", conflict)
			}
			if err = store.ControlBookOrganization(ctx, "job", "retry"); err != nil {
				t.Fatal(err)
			}
			a.processBookOrganization(ctx, "job", recognize)
			if calls != 3 {
				t.Fatalf("retry regenerated applied item: %d", calls)
			}
			if err = a.ReviewBookOrganization(ctx, "job", review); err == nil {
				t.Fatal("old revision applied regenerated result")
			}
			updated, _ := store.BookOrganizationItem(ctx, "job", ids[1])
			review.Revision = updated.Revision
			review.Decision = "discard"
			if err = a.ReviewBookOrganization(ctx, "job", review); err != nil {
				t.Fatal(err)
			}
			state, _ = a.GetBookMetadataState(ctx, kind, ids[1])
			if state.Title != manual {
				t.Fatal("discard wrote metadata")
			}
			job, _ = store.GetBookOrganization(ctx, "job")
			if job.Status != "completed" || job.Applied != 1 || job.PendingReview != 0 {
				t.Fatalf("final %+v", job)
			}
		})
	}
}

func TestBookOrganizationCancellationRejectsLateResultAndResumes(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "cancel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	book, err := store.UpsertComicBook(ctx, storage.ComicBookUpsert{Title: "夏", SourceFileName: "夏.zip", Location: "fixture.zip", PageCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateBookOrganization(ctx, "cancel", contracts.BookOrganizationRequest{Kind: "comic", BookIDs: []string{book.ID}, RequestID: "cancel", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	a := &App{store: store}
	a.processBookOrganization(ctx, "cancel", func(context.Context, contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
		if err := store.ControlBookOrganization(ctx, "cancel", "cancel"); err != nil {
			t.Fatal(err)
		}
		return contracts.AIActionPreviewDTO{}, errors.New("late provider response")
	})
	item, _ := store.BookOrganizationItem(ctx, "cancel", book.ID)
	if item.Status != "pending" || item.Revision != 0 {
		t.Fatalf("late result saved %+v", item)
	}
	if err = store.ControlBookOrganization(ctx, "cancel", "retry"); err != nil {
		t.Fatal(err)
	}
	a.processBookOrganization(ctx, "cancel", func(context.Context, contracts.AIActionRequest) (contracts.AIActionPreviewDTO, error) {
		return contracts.AIActionPreviewDTO{}, errors.New("provider failure")
	})
	job, _ := store.GetBookOrganization(ctx, "cancel")
	if job.Status != "partial_failed" || job.Failed != 1 {
		t.Fatalf("resume %+v", job)
	}
}
