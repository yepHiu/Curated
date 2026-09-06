package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

func TestPresentMoviesRequiresSeenIDs(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQuery{movies: contracts.MoviesPageDTO{
		Total: 1,
		Items: []contracts.MovieListItemDTO{{
			ID:       "m1",
			Title:    "Hello",
			Code:     "ABC-123",
			CoverURL: "/api/library/movies/m1/asset/cover",
			Actors:   []string{"A"},
		}},
	}}
	if err := RegisterQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := RegisterPresentTools(reg, gateway.MovieRefs()); err != nil {
		t.Fatal(err)
	}

	unknown := gateway.Invoke(context.Background(), core.Call{
		Name:      core.PresentMoviesName,
		Args:      json.RawMessage(`{"items":[{"movieId":"m1","reason":"轻松"}]}`),
		SessionID: "ses_1",
	})
	if unknown.OK {
		t.Fatalf("present before search should fail: %+v", unknown)
	}

	search := gateway.Invoke(context.Background(), core.Call{
		Name:      "search_movies",
		Args:      json.RawMessage(`{"q":"ABC","limit":20}`),
		SessionID: "ses_1",
	})
	if !search.OK {
		t.Fatalf("search = %+v", search)
	}
	gateway.RememberMovieRefs("ses_1", core.ExtractMovieRefs(search))

	shown := gateway.Invoke(context.Background(), core.Call{
		Name:      core.PresentMoviesName,
		Args:      json.RawMessage(`{"items":[{"movieId":"m1","reason":"轻松短片"}]}`),
		SessionID: "ses_1",
	})
	if !shown.OK {
		t.Fatalf("present after search = %+v", shown)
	}
	raw, _ := json.Marshal(shown.Data)
	if !strings.Contains(string(raw), "ABC-123") || !strings.Contains(string(raw), "轻松短片") {
		t.Fatalf("missing projection: %s", raw)
	}
}
