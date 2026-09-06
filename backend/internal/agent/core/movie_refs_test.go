package core

import (
	"testing"
)

func TestMovieRefStoreRejectsUnknownIDs(t *testing.T) {
	t.Parallel()
	store := NewMovieRefStore()
	store.Remember("ses", []MovieRef{{ID: "m1", Title: "Hello", Code: "ABC-123"}})
	found, missing := store.Lookup("ses", []string{"m1", "m2"})
	if len(found) != 1 || found[0].ID != "m1" {
		t.Fatalf("found = %+v", found)
	}
	if len(missing) != 1 || missing[0] != "m2" {
		t.Fatalf("missing = %v", missing)
	}
}

func TestExtractMovieRefsFromSearchEnvelope(t *testing.T) {
	t.Parallel()
	refs := ExtractMovieRefs(Result{OK: true, Data: map[string]any{
		"source": map[string]any{
			"items": []any{
				map[string]any{"id": "m1", "title": "Hello", "code": "ABC-123", "coverUrl": "/api/cover"},
			},
		},
	}})
	if len(refs) != 1 || refs[0].ID != "m1" || refs[0].CoverURL != "/api/cover" {
		t.Fatalf("refs = %+v", refs)
	}
}

func TestExtractMovieRefsFromDetailEnvelope(t *testing.T) {
	t.Parallel()
	refs := ExtractMovieRefs(Result{OK: true, Data: map[string]any{
		"source": map[string]any{"id": "m9", "title": "Nine"},
	}})
	if len(refs) != 1 || refs[0].ID != "m9" {
		t.Fatalf("refs = %+v", refs)
	}
}

func TestMovieRefStoreResetClearsTurn(t *testing.T) {
	t.Parallel()
	store := NewMovieRefStore()
	store.Remember("ses", []MovieRef{{ID: "m1"}})
	store.Reset("ses")
	found, missing := store.Lookup("ses", []string{"m1"})
	if len(found) != 0 || len(missing) != 1 {
		t.Fatalf("found=%v missing=%v", found, missing)
	}
}
