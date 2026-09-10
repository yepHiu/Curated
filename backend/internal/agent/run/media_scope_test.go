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
			Mentions:         []contracts.AIChatMention{{Kind: "movie", ID: "mentioned-book"}, {Kind: "actor", ID: "mentioned-person"}},
		})
		found, _ := gateway.MovieRefs().Lookup(route, []string{"book-id", "selected-book", "mentioned-book"})
		if len(found) != 0 || gateway.ActorRefs().Known(route, "book-person") || gateway.ActorRefs().Known(route, "mentioned-person") {
			t.Fatal("book context became a movie/actor anchor")
		}
	}
	seedTurnEntities(gateway, "movie", &contracts.AIChatContext{Route: "detail", MovieID: "movie-id"})
	found, _ := gateway.MovieRefs().Lookup("movie", []string{"movie-id"})
	if len(found) != 1 {
		t.Fatal("movie anchor was lost")
	}
}
