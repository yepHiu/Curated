package core

import "testing"

func TestActorRefStoreRejectsUnknownNames(t *testing.T) {
	t.Parallel()
	store := NewActorRefStore()
	store.Remember("ses", []string{"Alice"})
	if !store.Known("ses", " alice ") {
		t.Fatal("expected remembered actor")
	}
	if store.Known("ses", "Bob") {
		t.Fatal("unknown actor should be rejected")
	}
}

func TestExtractActorNamesFromListAndProfile(t *testing.T) {
	t.Parallel()
	fromList := ExtractActorNames(Result{OK: true, Data: map[string]any{
		"source": map[string]any{"items": []any{map[string]any{"name": "Alice", "movieCount": 3}}},
	}})
	if len(fromList) != 1 || fromList[0] != "Alice" {
		t.Fatalf("list names = %#v", fromList)
	}
	fromProfile := ExtractActorNames(Result{OK: true, Data: map[string]any{
		"source": map[string]any{"name": "Alice", "aliases": []any{"A. Li"}},
	}})
	if len(fromProfile) != 2 {
		t.Fatalf("profile names = %#v", fromProfile)
	}
}

func TestActorRefStoreResetClearsTurn(t *testing.T) {
	t.Parallel()
	store := NewActorRefStore()
	store.Remember("ses", []string{"Alice"})
	store.Reset("ses")
	if store.Known("ses", "Alice") {
		t.Fatal("reset should clear names")
	}
}
