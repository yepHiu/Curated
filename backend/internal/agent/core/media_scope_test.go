package core

import (
	"curated-backend/internal/contracts"
	"testing"
)

func TestMoviePageContextDropsBookPayload(t *testing.T) {
	for _, route := range []string{"comics", "comic-detail", "comic-reader", "photos", "photo-detail", "photo-viewer", "/comics/secret/read/0", "#/photos?q=secret"} {
		page := &contracts.AIChatContext{Route: route, MovieID: "secret-book", ActorName: "secret-person", Query: "secret-query", SelectedMovieIDs: []string{"secret-selection"}, ActiveFilters: &contracts.AIChatActiveFilters{Tag: "secret-tag"}}
		if got := MoviePageContext(page); got != nil {
			t.Errorf("book context escaped for %s: %+v", route, got)
		}
	}
	page := &contracts.AIChatContext{Route: "detail", MovieID: "movie-1"}
	if MoviePageContext(page) != page || MoviePageContext(nil) != nil {
		t.Fatal("movie context or nil changed")
	}
}
