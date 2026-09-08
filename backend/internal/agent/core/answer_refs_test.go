package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func answerResult(row map[string]any) Result {
	return Result{OK: true, Data: map[string]any{"source": map[string]any{"items": []any{row}}}}
}

func TestAnswerRefsIsolationAndImmutableSources(t *testing.T) {
	a, b := NewAnswerRefStore(), NewAnswerRefStore()
	row := map[string]any{"id": "m1", "code": "TEST-101", "title": "Local", "actors": []string{"Actor"}, "runtimeMinutes": 0}
	hints := a.Capture("search_movies", answerResult(row))
	if len(hints) != 1 {
		t.Fatal(hints)
	}
	if _, ok := b.Lookup(hints[0].RefID); ok {
		t.Fatal("cross-request reference accepted")
	}
	row["title"] = "mutated"
	ref, _ := a.Lookup(hints[0].RefID)
	if ref.Fields["title"] != "Local" || ref.Fields["runtimeMinutes"] != nil {
		t.Fatal(ref)
	}
	ref.Fields["title"] = "tampered"
	ref, _ = a.Lookup(hints[0].RefID)
	if ref.Fields["title"] != "Local" {
		t.Fatal("mutable snapshot")
	}
	provider := a.Capture(SearchProviderTitlesName, answerResult(map[string]any{"code": "TEST-101", "title": "Provider", "movieId": "m1", "inLibrary": true}))
	pr, _ := a.Lookup(provider[0].RefID)
	if pr.Fields["actors"] != nil || pr.Fields["title"] != "Provider" {
		t.Fatal("sources merged", pr)
	}
	local, _ := a.LocalMovie("m1")
	if local.Source != "local" || local.Fields["title"] != "Local" {
		t.Fatal(local)
	}
}

func TestAnswerRefsRejectUntrustedEnvelopes(t *testing.T) {
	s := NewAnswerRefStore()
	r := answerResult(map[string]any{"id": "m1", "code": "TEST-101"})
	for _, name := range []string{PresentMoviesName, SubmitAnswerName, SaveMovieCommentName, GetSourcePageName, "invented"} {
		if len(s.Capture(name, r)) != 0 {
			t.Fatal(name)
		}
	}
	for _, status := range []string{"ambiguous", "unmatched"} {
		r.Data = map[string]any{"source": map[string]any{"kind": "movie", "status": status, "candidates": []any{map[string]any{"movieId": "m1", "code": "TEST-101"}}}}
		if len(s.Capture("resolve_entities", r)) != 0 {
			t.Fatal(status)
		}
	}
	r = answerResult(map[string]any{"code": "TEST-101", "movieId": "forged", "inLibrary": false})
	h := s.Capture(SearchProviderTitlesName, r)
	ref, _ := s.Lookup(h[0].RefID)
	if ref.MovieID != "" {
		t.Fatal("off-library movie ID trusted")
	}
}

func TestGatewayCapturesBeforeMinimalProjection(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(ToolDefinition{Name: "search_movies", Permission: PermissionRead, ParamsSchema: Schema{Type: "object"}, Handler: func(context.Context, Call) (Result, error) {
		return answerResult(map[string]any{"id": "m1", "code": "TEST-101", "title": "PRIVATE_TITLE"}), nil
	}})
	s := NewAnswerRefStore()
	r := NewGateway(reg, nil, nil, nil).Invoke(WithAnswerRefs(context.Background(), s), Call{Name: "search_movies", Args: json.RawMessage(`{}`), Sanitize: SanitizeMinimal})
	raw, _ := json.Marshal(r)
	if strings.Contains(string(raw), "PRIVATE_TITLE") || len(r.AnswerRefs) != 1 {
		t.Fatalf("projection: %s", raw)
	}
	ref, _ := s.Lookup(r.AnswerRefs[0].RefID)
	if ref.Fields["title"] != "PRIVATE_TITLE" {
		t.Fatal("full snapshot lost")
	}
}
