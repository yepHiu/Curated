package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDailyHomepageRecommendationSnapshotLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "homepage-daily.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	snapshot := HomepageDailyRecommendationSnapshot{
		DateUTC:                "2026-04-15",
		HeroMovieIDs:           []string{"m1", "m2"},
		RecommendationMovieIDs: []string{"m3", "m4"},
		GeneratedAt:            "2026-04-15T00:00:00Z",
		GenerationVersion:      "v1",
	}

	if err := store.UpsertHomepageDailyRecommendationSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("UpsertHomepageDailyRecommendationSnapshot() error = %v", err)
	}

	got, ok, err := store.GetHomepageDailyRecommendationSnapshot(ctx, "2026-04-15")
	if err != nil {
		t.Fatalf("GetHomepageDailyRecommendationSnapshot() error = %v", err)
	}
	if !ok {
		t.Fatalf("snapshot not found")
	}
	if got.DateUTC != snapshot.DateUTC {
		t.Fatalf("DateUTC = %q, want %q", got.DateUTC, snapshot.DateUTC)
	}
	if len(got.HeroMovieIDs) != 2 || got.HeroMovieIDs[0] != "m1" || got.HeroMovieIDs[1] != "m2" {
		t.Fatalf("HeroMovieIDs = %#v", got.HeroMovieIDs)
	}
	if len(got.RecommendationMovieIDs) != 2 || got.RecommendationMovieIDs[0] != "m3" || got.RecommendationMovieIDs[1] != "m4" {
		t.Fatalf("RecommendationMovieIDs = %#v", got.RecommendationMovieIDs)
	}
}

func TestHomepageDailyRecommendationSnapshotReturnsMissingForUnknownDay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "homepage-daily.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	_, ok, err := store.GetHomepageDailyRecommendationSnapshot(ctx, "2026-04-16")
	if err != nil {
		t.Fatalf("GetHomepageDailyRecommendationSnapshot() error = %v", err)
	}
	if ok {
		t.Fatalf("expected missing snapshot")
	}
}

func TestListHomepageDailyRecommendationSnapshotsInRangeReturnsDescendingRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "homepage-daily.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	for _, snapshot := range []HomepageDailyRecommendationSnapshot{
		{
			DateUTC:                "2026-04-13",
			HeroMovieIDs:           []string{"m1"},
			RecommendationMovieIDs: []string{"m2"},
			GeneratedAt:            "2026-04-13T00:00:00Z",
			GenerationVersion:      "v1",
		},
		{
			DateUTC:                "2026-04-14",
			HeroMovieIDs:           []string{"m3"},
			RecommendationMovieIDs: []string{"m4"},
			GeneratedAt:            "2026-04-14T00:00:00Z",
			GenerationVersion:      "v1",
		},
		{
			DateUTC:                "2026-04-15",
			HeroMovieIDs:           []string{"m5"},
			RecommendationMovieIDs: []string{"m6"},
			GeneratedAt:            "2026-04-15T00:00:00Z",
			GenerationVersion:      "v1",
		},
	} {
		if err := store.UpsertHomepageDailyRecommendationSnapshot(ctx, snapshot); err != nil {
			t.Fatalf("UpsertHomepageDailyRecommendationSnapshot(%s) error = %v", snapshot.DateUTC, err)
		}
	}

	got, err := store.ListHomepageDailyRecommendationSnapshotsInRange(ctx, "2026-04-14", "2026-04-15")
	if err != nil {
		t.Fatalf("ListHomepageDailyRecommendationSnapshotsInRange() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].DateUTC != "2026-04-15" || got[1].DateUTC != "2026-04-14" {
		t.Fatalf("dates = %#v", []string{got[0].DateUTC, got[1].DateUTC})
	}
}
