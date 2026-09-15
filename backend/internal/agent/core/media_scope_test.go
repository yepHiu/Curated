package core

import (
	"curated-backend/internal/contracts"
	"testing"
)

func TestMoviePageContextKeepsBookPayloadOnBookRoutes(t *testing.T) {
	page := &contracts.AIChatContext{
		Route:            "comic-detail",
		MovieID:          "secret-book",
		ComicID:          "comic-1",
		ActorName:        "secret-person",
		Query:            "secret-query",
		SelectedMovieIDs: []string{"secret-selection"},
		SelectedComicIDs: []string{"comic-1"},
		ActiveFilters:    &contracts.AIChatActiveFilters{Tag: "secret-tag", Actor: "secret-person", ReadStatus: "unread"},
	}
	got := MoviePageContext(page)
	if got == nil || got.ComicID != "comic-1" || got.Query != "secret-query" {
		t.Fatalf("book context dropped: %+v", got)
	}
	if got.MovieID != "" || got.ActorName != "" || len(got.SelectedMovieIDs) != 0 {
		t.Fatalf("movie anchors leaked from book route: %+v", got)
	}
	if got.ActiveFilters == nil || got.ActiveFilters.Tag != "secret-tag" || got.ActiveFilters.Actor != "" || got.ActiveFilters.ReadStatus != "unread" {
		t.Fatalf("book filters not normalized: %+v", got.ActiveFilters)
	}
}

func TestMoviePageContextStripsBookPayloadOnMovieRoutes(t *testing.T) {
	page := &contracts.AIChatContext{Route: "detail", MovieID: "movie-1", ComicID: "comic-1", PhotoID: "photo-1"}
	got := MoviePageContext(page)
	if got == nil || got.MovieID != "movie-1" || got.ComicID != "" || got.PhotoID != "" {
		t.Fatalf("movie context lost or kept book ids: %+v", got)
	}
	plain := &contracts.AIChatContext{Route: "detail", MovieID: "movie-1"}
	if MoviePageContext(plain) != plain || MoviePageContext(nil) != nil {
		t.Fatal("movie context or nil changed")
	}
}
