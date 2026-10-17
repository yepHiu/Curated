package storage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"curated-backend/internal/contracts"
)

func TestBookReviewTransactionRollsBackMetadataOnStaleRevision(t *testing.T) {
	s := newTopicTestStore(t)
	ctx := context.Background()
	book, err := s.UpsertComicBook(ctx, ComicBookUpsert{Title: "夏の記録", SourceFileName: "夏の記録.zip", Location: "review.zip", PageCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	state := contracts.BookMetadataState{Title: book.Title, OriginalTitle: book.Title, SourceFileName: "夏の記録.zip"}
	snapshot, _ := json.Marshal(state)
	metadata := contracts.BookMetadata{CleanTitle: "夏の記録", TranslatedTitle: "夏日记录"}
	_, err = s.CreateBookOrganization(ctx, "job", contracts.BookOrganizationRequest{Kind: "comic", BookIDs: []string{book.ID}, Locale: "zh-CN", RequestID: "r"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SaveBookOrganizationProposal(ctx, "job", book.ID, "ready", string(snapshot), metadata, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ApplyBookMetadata(WithBookOrganizationReview(ctx, "job", book.ID, 0), "comic", book.ID, metadata, string(snapshot)); !errors.Is(err, ErrAIWriteConflict) {
		t.Fatalf("stale review %v", err)
	}
	var title, raw string
	if err = s.db.QueryRow(`SELECT user_title,metadata_json FROM comic_books WHERE id=?`, book.ID).Scan(&title, &raw); err != nil {
		t.Fatal(err)
	}
	if title != "" || raw != "{}" {
		t.Fatalf("partial transaction escaped: %q %q", title, raw)
	}
	item, _ := s.BookOrganizationItem(ctx, "job", book.ID)
	if item.Status != "ready" {
		t.Fatal("stale review consumed proposal")
	}
	if _, err = s.ApplyBookMetadata(WithBookOrganizationReview(ctx, "job", book.ID, 1), "comic", book.ID, metadata, string(snapshot)); err != nil {
		t.Fatal(err)
	}
	item, _ = s.BookOrganizationItem(ctx, "job", book.ID)
	job, _ := s.GetBookOrganization(ctx, "job")
	if item.Status != "applied" || job.Status != "completed" {
		t.Fatalf("state not committed with metadata: %+v %+v", item, job)
	}
}
