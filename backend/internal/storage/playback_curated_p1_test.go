package storage

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func newMigratedTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "curated-p1.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return store
}

func insertCuratedFrameForP1Test(t *testing.T, store *SQLiteStore, meta CuratedFrameMeta) {
	t.Helper()
	if meta.ID == "" {
		t.Fatal("test frame id is required")
	}
	if meta.MovieID == "" {
		meta.MovieID = "movie-" + meta.ID
	}
	if meta.CapturedAt == "" {
		meta.CapturedAt = "2026-04-11T00:00:00Z"
	}
	if err := store.InsertCuratedFrame(context.Background(), meta, []byte("png-"+meta.ID)); err != nil {
		t.Fatal(err)
	}
}

func TestCuratedFrameQueryFiltersCountsAndPages(t *testing.T) {
	t.Parallel()
	store := newMigratedTestStore(t)
	ctx := context.Background()

	insertCuratedFrameForP1Test(t, store, CuratedFrameMeta{
		ID: "frame-1", MovieID: "movie-1", Title: "First Scene", Code: "ABC-001",
		Actors: []string{"Airi", "Mina"}, PositionSec: 12, CapturedAt: "2026-04-11T03:00:00Z", Tags: []string{"closeup", "favorite"},
	})
	insertCuratedFrameForP1Test(t, store, CuratedFrameMeta{
		ID: "frame-2", MovieID: "movie-2", Title: "Second Scene", Code: "XYZ-002",
		Actors: []string{"Mina"}, PositionSec: 24, CapturedAt: "2026-04-11T02:00:00Z", Tags: []string{"wide"},
	})
	insertCuratedFrameForP1Test(t, store, CuratedFrameMeta{
		ID: "frame-3", MovieID: "movie-1", Title: "Third Scene", Code: "ABC-003",
		Actors: []string{"Rin"}, PositionSec: 36, CapturedAt: "2026-04-11T01:00:00Z", Tags: []string{"closeup"},
	})

	page, err := store.QueryCuratedFrames(ctx, CuratedFrameQuery{
		Query:  "abc",
		Tag:    "closeup",
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("total = %d, want 2", page.Total)
	}
	if page.Limit != 1 || page.Offset != 1 {
		t.Fatalf("page bounds = %d/%d, want 1/1", page.Limit, page.Offset)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "frame-3" {
		t.Fatalf("items = %+v, want frame-3", page.Items)
	}

	byActor, err := store.QueryCuratedFrames(ctx, CuratedFrameQuery{Actor: "Mina", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if byActor.Total != 2 {
		t.Fatalf("actor total = %d, want 2", byActor.Total)
	}
}

func TestCuratedFrameFacetAggregates(t *testing.T) {
	t.Parallel()
	store := newMigratedTestStore(t)
	ctx := context.Background()

	insertCuratedFrameForP1Test(t, store, CuratedFrameMeta{
		ID: "frame-a", Actors: []string{"Mina", "Airi"}, Tags: []string{"closeup", "favorite"},
	})
	insertCuratedFrameForP1Test(t, store, CuratedFrameMeta{
		ID: "frame-b", Actors: []string{"Mina"}, Tags: []string{"closeup"},
	})

	actors, err := store.ListCuratedFrameActors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actors, []CuratedFrameFacet{{Name: "Mina", Count: 2}, {Name: "Airi", Count: 1}}) {
		t.Fatalf("actors = %+v", actors)
	}

	tags, err := store.ListCuratedFrameTags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tags, []CuratedFrameFacet{{Name: "closeup", Count: 2}, {Name: "favorite", Count: 1}}) {
		t.Fatalf("tags = %+v", tags)
	}
}

func TestCuratedFrameThumbnailFallsBackToImage(t *testing.T) {
	t.Parallel()
	store := newMigratedTestStore(t)
	ctx := context.Background()

	insertCuratedFrameForP1Test(t, store, CuratedFrameMeta{ID: "legacy-frame"})

	got, err := store.GetCuratedFrameThumbnail(ctx, "legacy-frame")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "png-legacy-frame" {
		t.Fatalf("thumbnail fallback = %q", string(got))
	}
}
