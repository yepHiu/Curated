package storage

import (
	"context"
	"encoding/json"
	"testing"

	"curated-backend/internal/contracts"
)

func TestBookCreditTagsFollowConfirmedMetadata(t *testing.T) {
	for _, kind := range []string{"comic", "photo"} {
		t.Run(kind, func(t *testing.T) {
			s := newTopicTestStore(t)
			ctx := context.Background()
			const title = "青井 Studio 夏の記録"
			var id string
			var err error
			if kind == "comic" {
				book, e := s.UpsertComicBook(ctx, ComicBookUpsert{Title: title, SourceFileName: title + ".zip", Location: "comic.zip", PageCount: 1})
				id, err = book.ID, e
			} else {
				book, e := s.UpsertPhotoBook(ctx, PhotoBookUpsert{Title: title, SourceFileName: title + ".zip", Location: "photo.zip", PageCount: 1})
				id, err = book.ID, e
			}
			if err != nil {
				t.Fatal(err)
			}
			check := func(tag, query string, want int) {
				t.Helper()
				if kind == "comic" {
					page, err := s.ListComicBooks(ctx, contracts.ListComicBooksRequest{Tag: tag, Query: query, Limit: 10})
					if err != nil || page.Total != want || len(page.Items) != want {
						t.Fatalf("tag %q: %+v, %v", tag, page, err)
					}
				} else {
					page, err := s.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{Tag: tag, Query: query, Limit: 10})
					if err != nil || page.Total != want || len(page.Items) != want {
						t.Fatalf("tag %q: %+v, %v", tag, page, err)
					}
				}
			}
			metadata := contracts.BookMetadata{CleanTitle: "夏の記録", TranslatedTitle: "夏日记录"}
			person := []contracts.BookCredit{{Name: "青井", Evidence: "青井"}}
			studio := []contracts.BookCredit{{Name: "Studio", Evidence: "Studio"}}
			if kind == "comic" {
				metadata.Authors, metadata.Circles = person, studio
			} else {
				metadata.Models, metadata.Photographers = person, studio
			}
			state := contracts.BookMetadataState{Title: title, OriginalTitle: title, SourceFileName: title + ".zip"}
			snapshot, _ := json.Marshal(state)
			if _, err := s.CreateBookOrganization(ctx, "job", contracts.BookOrganizationRequest{Kind: kind, BookIDs: []string{id}, Locale: "zh-CN", RequestID: "r"}); err != nil {
				t.Fatal(err)
			}
			if err := s.SaveBookOrganizationProposal(ctx, "job", id, "ready", string(snapshot), metadata, ""); err != nil {
				t.Fatal(err)
			}
			check("青井", "", 0) // A proposed name and a title hit are not confirmed tags.
			state, err = s.ApplyBookMetadata(WithBookOrganizationReview(ctx, "job", id, 1), kind, id, metadata, string(snapshot))
			if err != nil {
				t.Fatal(err)
			}
			check(" 青井 ", "", 1)
			check("Studio", "", 1)
			check("青", "", 0)
			check("青井", "missing", 0)
			check("青井", "夏日", 1)
			// A manual tag with the same name survives removal of the credit.
			tags := []string{"Studio"}
			if kind == "comic" {
				_, err = s.PatchComicBook(ctx, id, contracts.PatchComicBookRequest{Tags: &tags})
			} else {
				err = s.ReplacePhotoTags(ctx, id, tags)
			}
			if err != nil {
				t.Fatal(err)
			}
			snapshot, _ = json.Marshal(state)
			metadata.Authors, metadata.Circles, metadata.Models, metadata.Photographers = nil, nil, nil, nil
			if _, err := s.ApplyBookMetadata(ctx, kind, id, metadata, string(snapshot)); err != nil {
				t.Fatal(err)
			}
			check("青井", "", 0)
			check("Studio", "", 1)
		})
	}
}
