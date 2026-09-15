package run

import (
	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"testing"
)

func TestBookContextCannotSeedMovieOrActorAnchors(t *testing.T) {
	gateway := core.NewGateway(core.NewRegistry(), nil, nil, nil)
	for _, route := range []string{"comic-detail", "photo-viewer"} {
		seedTurnEntities(gateway, route, &contracts.AIChatContext{
			Route: route, MovieID: "book-id", ActorName: "book-person",
			SelectedMovieIDs: []string{"selected-book"},
		})
		found, _ := gateway.MovieRefs().Lookup(route, []string{"book-id", "selected-book"})
		if len(found) != 0 || gateway.ActorRefs().Known(route, "book-person") {
			t.Fatal("book context became a movie/actor anchor")
		}
	}
	seedTurnEntities(gateway, "movie", &contracts.AIChatContext{Route: "detail", MovieID: "movie-id"})
	found, _ := gateway.MovieRefs().Lookup("movie", []string{"movie-id"})
	if len(found) != 1 {
		t.Fatal("movie anchor was lost")
	}
}

func TestBookContextSeedsBookAnchors(t *testing.T) {
	gateway := core.NewGateway(core.NewRegistry(), nil, nil, nil)
	seedTurnEntities(gateway, "comic", &contracts.AIChatContext{
		Route: "comic-detail", ComicID: "comic-1",
		Mentions: []contracts.AIChatMention{{Kind: "comic", ID: "comic-2", Label: "Title"}},
	})
	found, missing := gateway.BookRefs().Lookup("comic", "comic", []string{"comic-1", "comic-2"})
	if len(found) != 2 || len(missing) != 0 {
		t.Fatalf("book anchors missing: found=%d missing=%v", len(found), missing)
	}
}
