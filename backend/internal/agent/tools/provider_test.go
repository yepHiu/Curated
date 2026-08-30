package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

type stubProviderLookup struct {
	hits  []ProviderTitleHit
	local map[string]contracts.MovieListItemDTO
	err   error
}

func (s stubProviderLookup) SearchProviderTitles(context.Context, string, int) ([]ProviderTitleHit, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.hits, nil
}

func (s stubProviderLookup) FindLibraryMoviesByCodes(context.Context, []string) (map[string]contracts.MovieListItemDTO, error) {
	if s.local == nil {
		return map[string]contracts.MovieListItemDTO{}, nil
	}
	return s.local, nil
}

func TestSearchProviderTitlesRequiresThisTurnAnchor(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQuery{movies: contracts.MoviesPageDTO{
		Total: 1,
		Items: []contracts.MovieListItemDTO{{ID: "m1", Title: "Hello", Code: "ABC-123", Actors: []string{"Alice"}}},
	}}
	lookup := stubProviderLookup{
		hits: []ProviderTitleHit{
			{Code: "ABC-123", Title: "Hello", Provider: "javbus", Homepage: "https://example.test/abc-123", Score: 4.2},
			{Code: "ABC-124", Title: "Other", Provider: "javbus", Homepage: "https://example.test/abc-124"},
		},
		local: map[string]contracts.MovieListItemDTO{
			"abc-123": {ID: "m1", Code: "ABC-123", Title: "Hello"},
		},
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := RegisterQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	if err := RegisterProviderTools(reg, query, lookup, nil, gateway.MovieRefs(), gateway.ActorRefs(), gateway.SourceURLs()); err != nil {
		t.Fatal(err)
	}

	rejected := gateway.Invoke(context.Background(), core.Call{
		Name:      core.SearchProviderTitlesName,
		Args:      json.RawMessage(`{"actorName":"Alice"}`),
		SessionID: "ses_1",
	})
	if rejected.OK {
		t.Fatalf("unanchored actor search should fail: %+v", rejected)
	}

	search := gateway.Invoke(context.Background(), core.Call{
		Name:      "search_movies",
		Args:      json.RawMessage(`{"q":"ABC","limit":5}`),
		SessionID: "ses_1",
	})
	if !search.OK {
		t.Fatalf("search_movies = %+v", search)
	}
	gateway.RememberMovieRefs("ses_1", core.ExtractMovieRefs(search))
	gateway.RememberActorNames("ses_1", core.ExtractActorNames(search))

	found := gateway.Invoke(context.Background(), core.Call{
		Name:      core.SearchProviderTitlesName,
		Args:      json.RawMessage(`{"actorName":"Alice","limit":15}`),
		SessionID: "ses_1",
	})
	if !found.OK {
		t.Fatalf("anchored search = %+v", found)
	}
	raw, _ := json.Marshal(found.Data)
	if !strings.Contains(string(raw), `"inLibrary":true`) || !strings.Contains(string(raw), `"movieId":"m1"`) {
		t.Fatalf("missing in-library mapping: %s", raw)
	}
	if !strings.Contains(string(raw), `"inLibrary":false`) || strings.Count(string(raw), `"movieId":"m1"`) != 1 {
		t.Fatalf("off-library row should omit extra movieId: %s", raw)
	}
}

func TestSearchProviderTitlesRejectsUnknownQueryField(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := RegisterProviderTools(reg, stubQuery{}, stubProviderLookup{}, nil, gateway.MovieRefs(), gateway.ActorRefs(), gateway.SourceURLs()); err != nil {
		t.Fatal(err)
	}
	result := gateway.Invoke(context.Background(), core.Call{
		Name: core.SearchProviderTitlesName,
		Args: json.RawMessage(`{"query":"weather in tokyo"}`),
	})
	if result.OK {
		t.Fatalf("free-text query should be rejected: %+v", result)
	}
}

func TestPresentMoviesRejectsOffLibraryProviderID(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQuery{movies: contracts.MoviesPageDTO{
		Total: 1,
		Items: []contracts.MovieListItemDTO{{ID: "m1", Title: "Hello", Code: "ABC-123", Actors: []string{"Alice"}}},
	}}
	lookup := stubProviderLookup{
		hits: []ProviderTitleHit{{Code: "ABC-999", Title: "Outside", Provider: "javbus"}},
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	if err := RegisterQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	if err := RegisterPresentTools(reg, gateway.MovieRefs()); err != nil {
		t.Fatal(err)
	}
	if err := RegisterProviderTools(reg, query, lookup, nil, gateway.MovieRefs(), gateway.ActorRefs(), gateway.SourceURLs()); err != nil {
		t.Fatal(err)
	}
	search := gateway.Invoke(context.Background(), core.Call{
		Name: "search_movies", Args: json.RawMessage(`{"q":"ABC"}`), SessionID: "ses_1",
	})
	gateway.RememberMovieRefs("ses_1", core.ExtractMovieRefs(search))
	gateway.RememberActorNames("ses_1", core.ExtractActorNames(search))
	provider := gateway.Invoke(context.Background(), core.Call{
		Name: core.SearchProviderTitlesName, Args: json.RawMessage(`{"actorName":"Alice"}`), SessionID: "ses_1",
	})
	if !provider.OK {
		t.Fatalf("provider search = %+v", provider)
	}
	gateway.RememberMovieRefs("ses_1", core.ExtractMovieRefs(provider))

	shown := gateway.Invoke(context.Background(), core.Call{
		Name:      core.PresentMoviesName,
		Args:      json.RawMessage(`{"items":[{"movieId":"ABC-999"}]}`),
		SessionID: "ses_1",
	})
	if shown.OK {
		t.Fatalf("present of off-library code should fail: %+v", shown)
	}
}

type stubDetailQuery struct {
	stubQuery
	detail contracts.MovieDetailDTO
}

func (s stubDetailQuery) GetMovieDetail(context.Context, string) (contracts.MovieDetailDTO, error) {
	return s.detail, nil
}

func (s stubDetailQuery) GetActorProfile(context.Context, string) (contracts.ActorProfileDTO, error) {
	return contracts.ActorProfileDTO{Name: "Alice", Homepage: "https://example.test/actor", Summary: "bio"}, nil
}

func TestGetToolsExposeScrapedHomepages(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubDetailQuery{
		detail: contracts.MovieDetailDTO{
			MovieListItemDTO: contracts.MovieListItemDTO{ID: "m1", Title: "Hello", Code: "ABC-123"},
			Homepage:         "https://www.javbus.com/ABC-123",
			MetadataRating:   4.5,
			MetadataProvider: "javbus",
		},
	}
	if err := RegisterQueryTools(reg, query); err != nil {
		t.Fatal(err)
	}
	gateway := core.NewGateway(reg, nil, nil, nil)
	detail := gateway.Invoke(context.Background(), core.Call{
		Name: "get_movie_detail", Args: json.RawMessage(`{"movieId":"m1"}`),
	})
	raw, _ := json.Marshal(detail.Data)
	if !strings.Contains(string(raw), "https://www.javbus.com/ABC-123") || !strings.Contains(string(raw), `"metadataRating":4.5`) {
		t.Fatalf("movie detail missing source anchors: %s", raw)
	}
	profile := gateway.Invoke(context.Background(), core.Call{
		Name: "get_actor_profile", Args: json.RawMessage(`{"name":"Alice"}`),
	})
	profileRaw, _ := json.Marshal(profile.Data)
	if !strings.Contains(string(profileRaw), "https://example.test/actor") {
		t.Fatalf("actor profile missing homepage: %s", profileRaw)
	}
}
