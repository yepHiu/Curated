package storage

import (
	"context"
	"testing"
)

func TestFindNearbyCuratedFrame(t *testing.T) {
	t.Parallel()
	store := newMigratedTestStore(t)
	ctx := context.Background()

	insertCuratedFrameForP1Test(t, store, CuratedFrameMeta{
		ID: "near-frame", MovieID: "movie-dup", PositionSec: 12.0, CapturedAt: "2026-04-11T01:00:00Z",
	})
	insertCuratedFrameForP1Test(t, store, CuratedFrameMeta{
		ID: "other-movie-frame", MovieID: "movie-other", PositionSec: 12.05, CapturedAt: "2026-04-11T02:00:00Z",
	})

	got, err := store.FindNearbyCuratedFrame(ctx, "movie-dup", 14.9, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != "near-frame" {
		t.Fatalf("near duplicate = %+v, want near-frame", got)
	}

	got, err = store.FindNearbyCuratedFrame(ctx, "movie-dup", 15.1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected no duplicate beyond threshold, got %+v", got)
	}
}
