package tools

import (
	"context"
	"encoding/json"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

func TestResolveEntitiesMatchesMovieCode(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQuery{movies: contracts.MoviesPageDTO{Items: []contracts.MovieListItemDTO{{ID: "m1", Code: "ABC-123", Title: "One"}}}}
	if err := reg.Register(resolveEntities(query)); err != nil {
		t.Fatal(err)
	}
	result := core.NewGateway(reg, nil, nil, nil).Invoke(context.Background(), core.Call{Name: resolveEntitiesName, Args: []byte(`{"query":"abc 123","kind":"movie"}`), Channel: core.ChannelChat})
	resolution := decodeResolution(t, result)
	if resolution.Status != "matched" || len(resolution.Candidates) != 1 || resolution.Candidates[0].MovieID != "m1" {
		t.Fatalf("resolution = %+v", resolution)
	}
}

func TestResolveEntitiesReturnsAmbiguousCandidates(t *testing.T) {
	t.Parallel()
	reg := core.NewRegistry()
	query := stubQuery{movies: contracts.MoviesPageDTO{Items: []contracts.MovieListItemDTO{{ID: "m1", Title: "Same"}, {ID: "m2", Title: "Same"}}}}
	if err := reg.Register(resolveEntities(query)); err != nil {
		t.Fatal(err)
	}
	result := core.NewGateway(reg, nil, nil, nil).Invoke(context.Background(), core.Call{Name: resolveEntitiesName, Args: []byte(`{"query":"Same","kind":"movie"}`), Channel: core.ChannelChat})
	resolution := decodeResolution(t, result)
	if resolution.Status != "ambiguous" || len(resolution.Candidates) != 2 {
		t.Fatalf("resolution = %+v", resolution)
	}
}

func decodeResolution(t *testing.T, result core.Result) contracts.AIEntityResolutionDTO {
	t.Helper()
	if !result.OK {
		t.Fatalf("result = %+v", result)
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Source contracts.AIEntityResolutionDTO `json:"source"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Source
}
