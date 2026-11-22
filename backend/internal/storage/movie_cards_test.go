package storage

import (
	"context"
	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
	"path/filepath"
	"testing"
)

func TestMovieCardsOrderedUniqueActiveAndOverrides(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "cards.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, code := range []string{"AAA-001", "BBB-002", "CCC-003"} {
		outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{Path: filepath.Join(t.TempDir(), code+".mp4"), FileName: code + ".mp4", Number: code})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.SaveMovieMetadata(ctx, scraper.Metadata{MovieID: outcome.MovieID, Number: code, Title: "original", Actors: []string{"Actor"}}); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, outcome.MovieID)
	}
	if _, err := store.db.ExecContext(ctx, "UPDATE movies SET user_title='custom', user_studio='studio', user_rating=4.5, user_release_date='2024-01-02' WHERE id=?", ids[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, "UPDATE movies SET trashed_at='2026-01-01' WHERE id=?", ids[2]); err != nil {
		t.Fatal(err)
	}
	cards, err := store.MovieCards(ctx, []string{ids[1], ids[0], ids[1], ids[2], "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cards.Items) != 2 || cards.Items[0].ID != ids[1] || cards.Items[1].ID != ids[0] {
		t.Fatalf("order/dedup: %+v", cards)
	}
	c := cards.Items[0]
	if c.Title != "custom" || c.Studio != "studio" || c.Rating != 4.5 || c.Year != 2024 || len(c.Actors) != 1 {
		t.Fatalf("overrides: %+v", c)
	}
	if len(cards.MissingIDs) != 2 || cards.MissingIDs[0] != ids[2] {
		t.Fatalf("missing: %+v", cards)
	}
	empty, err := store.MovieCards(ctx, nil)
	if err != nil || empty.Items == nil || empty.MissingIDs == nil {
		t.Fatalf("empty: %+v %v", empty, err)
	}
	if _, err := store.MovieCards(ctx, make([]string, 51)); err == nil {
		t.Fatal("expected batch limit")
	}
}
