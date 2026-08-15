package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
)

func newSavedViewTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "saved-views.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func testSavedViewFilters() contracts.SavedViewFiltersV1 {
	return contracts.SavedViewFiltersV1{
		SchemaVersion: contracts.SavedViewSchemaVersion,
		Mode:          "library",
		PlayState:     "all",
		Tab:           "all",
	}
}

func TestSavedViewsOrderedCRUDAndConstraints(t *testing.T) {
	t.Parallel()
	store := newSavedViewTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 20, 1, 2, 3, 0, time.UTC)

	first, err := store.CreateSavedView(ctx, "view_first", "Unwatched", testSavedViewFilters(), now)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	secondFilters := testSavedViewFilters()
	secondFilters.Resolution = "4k"
	second, err := store.CreateSavedView(ctx, "view_second", "4K", secondFilters, now.Add(time.Second))
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if first.SortOrder != 0 || second.SortOrder != 1 {
		t.Fatalf("unexpected initial order: first=%d second=%d", first.SortOrder, second.SortOrder)
	}

	_, err = store.CreateSavedView(ctx, "view_duplicate", "  uNwAtChEd ", testSavedViewFilters(), now)
	if !errors.Is(err, ErrSavedViewNameConflict) {
		t.Fatalf("duplicate name error = %v, want ErrSavedViewNameConflict", err)
	}

	if err := store.ReorderSavedViews(ctx, []string{second.ID, first.ID}, now.Add(2*time.Second)); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	items, err := store.ListSavedViews(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 || items[0].ID != second.ID || items[1].ID != first.ID {
		t.Fatalf("unexpected reordered items: %#v", items)
	}

	if err := store.ReorderSavedViews(ctx, []string{second.ID, second.ID}, now); !errors.Is(err, ErrSavedViewOrderInvalid) {
		t.Fatalf("duplicate reorder error = %v", err)
	}

	updatedFilters := testSavedViewFilters()
	rating := 5.0
	updatedFilters.UserRating = &rating
	updated, err := store.UpdateSavedView(ctx, first.ID, "Five stars", updatedFilters, now.Add(3*time.Second))
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Five stars" || updated.Filters.UserRating == nil || *updated.Filters.UserRating != 5 {
		t.Fatalf("unexpected updated view: %#v", updated)
	}

	if err := store.DeleteSavedView(ctx, second.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	items, err = store.ListSavedViews(ctx)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(items) != 1 || items[0].ID != first.ID || items[0].SortOrder != 0 {
		t.Fatalf("delete did not compact order: %#v", items)
	}
}

func TestSavedViewsEnforceMaximum(t *testing.T) {
	t.Parallel()
	store := newSavedViewTestStore(t)
	ctx := context.Background()
	for index := 0; index < MaxSavedViews; index++ {
		_, err := store.CreateSavedView(
			ctx,
			fmt.Sprintf("view_%02d", index),
			fmt.Sprintf("View %02d", index),
			testSavedViewFilters(),
			time.Unix(int64(index), 0),
		)
		if err != nil {
			t.Fatalf("create view %d: %v", index, err)
		}
	}
	_, err := store.CreateSavedView(ctx, "view_over", "Over limit", testSavedViewFilters(), time.Now())
	if !errors.Is(err, ErrSavedViewLimit) {
		t.Fatalf("over-limit error = %v, want ErrSavedViewLimit", err)
	}
}

func TestListMoviesSavedViewFilters(t *testing.T) {
	t.Parallel()
	store := newSavedViewTestStore(t)
	ctx := context.Background()

	type seed struct {
		code       string
		tag        string
		actor      string
		studio     string
		resolution string
		addedAt    string
		userRating float64
		position   float64
		duration   float64
		played     bool
	}
	seeds := []seed{
		{code: "VIEW-001", tag: "Featured", actor: "Actor A", studio: "Studio A", resolution: "2160p", addedAt: "2026-07-19T00:00:00Z", userRating: 5},
		{code: "VIEW-002", tag: "Drama", actor: "Actor B", studio: "Studio B", resolution: "1080p", addedAt: "2026-06-01T00:00:00Z", userRating: 4, position: 100, duration: 1000, played: true},
		{code: "VIEW-003", tag: "Drama", actor: "Actor C", studio: "Studio C", resolution: "720p", addedAt: "2025-01-01T00:00:00Z", userRating: 3, position: 960, duration: 1000, played: true},
	}
	ids := make(map[string]string)
	for _, item := range seeds {
		outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
			TaskID:   "saved-view-filter-test",
			Path:     filepath.Join(t.TempDir(), item.code+".mp4"),
			FileName: item.code + ".mp4",
			Number:   item.code,
		})
		if err != nil {
			t.Fatalf("persist %s: %v", item.code, err)
		}
		ids[item.code] = outcome.MovieID
		actors := []string{item.actor}
		if item.code == "VIEW-001" {
			actors = []string{item.actor, "Actor Dual"}
		}
		if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
			MovieID: outcome.MovieID,
			Number:  item.code,
			Title:   item.code,
			Studio:  item.studio,
			Actors:  actors,
			Tags:    []string{item.tag},
		}); err != nil {
			t.Fatalf("metadata %s: %v", item.code, err)
		}
		if _, err := store.db.ExecContext(ctx, `
			UPDATE movies SET resolution = ?, added_at = ?, user_rating = ?, year = 2026, runtime_minutes = 120 WHERE id = ?`,
			item.resolution,
			item.addedAt,
			item.userRating,
			outcome.MovieID,
		); err != nil {
			t.Fatalf("update filter fields %s: %v", item.code, err)
		}
		if item.played {
			if err := store.RecordPlayedMovie(ctx, outcome.MovieID); err != nil {
				t.Fatalf("record played %s: %v", item.code, err)
			}
			if err := store.UpsertPlaybackProgress(ctx, outcome.MovieID, item.position, item.duration); err != nil {
				t.Fatalf("progress %s: %v", item.code, err)
			}
		}
	}

	unratedOutcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID:   "saved-view-filter-test",
		Path:     filepath.Join(t.TempDir(), "VIEW-004.mp4"),
		FileName: "VIEW-004.mp4",
		Number:   "VIEW-004",
	})
	if err != nil {
		t.Fatalf("persist VIEW-004: %v", err)
	}
	ids["VIEW-004"] = unratedOutcome.MovieID
	if _, err := store.db.ExecContext(ctx, `
		UPDATE movies
		SET user_rating = NULL,
			year = 0,
			runtime_minutes = 60,
			cover_url = '',
			thumb_url = '',
			added_at = '2025-02-01T00:00:00Z'
		WHERE id = ?`, unratedOutcome.MovieID); err != nil {
		t.Fatalf("update VIEW-004: %v", err)
	}

	assertIDs := func(name string, request contracts.ListMoviesRequest, want ...string) {
		t.Helper()
		request.Limit = 20
		page, err := store.ListMovies(ctx, request)
		if err != nil {
			t.Fatalf("%s list: %v", name, err)
		}
		got := make(map[string]bool)
		for _, item := range page.Items {
			got[item.ID] = true
		}
		if len(got) != len(want) {
			t.Fatalf("%s ids=%v, want %v", name, got, want)
		}
		for _, code := range want {
			if !got[ids[code]] {
				t.Fatalf("%s missing %s in %v", name, code, got)
			}
		}
	}

	five := 5.0
	four := 4.0
	assertIDs("tag", contracts.ListMoviesRequest{Tag: "Featured"}, "VIEW-001")
	if err := store.PatchMovieUserPrefs(ctx, ids["VIEW-001"], contracts.PatchMovieInput{
		UserTagsSet: true,
		UserTags:    []string{"mine"},
	}); err != nil {
		t.Fatalf("add user tag: %v", err)
	}
	assertIDs("user and metadata tags AND", contracts.ListMoviesRequest{Tag: "Featured,mine"}, "VIEW-001")
	assertIDs("repeated tags", contracts.ListMoviesRequest{Tags: []string{"Featured", "mine"}}, "VIEW-001")
	assertIDs("unmatched AND", contracts.ListMoviesRequest{Tag: "Featured,Drama"})
	assertIDs("actor", contracts.ListMoviesRequest{Actor: "Actor B"}, "VIEW-002")
	assertIDs("actors AND", contracts.ListMoviesRequest{Actor: "Actor A,Actor Dual"}, "VIEW-001")
	assertIDs("actors AND unmatched", contracts.ListMoviesRequest{Actor: "Actor A,Actor B"})
	assertIDs("studio", contracts.ListMoviesRequest{Studio: "Studio C"}, "VIEW-003")
	assertIDs("studios OR", contracts.ListMoviesRequest{Studio: "Studio A,Studio C"}, "VIEW-001", "VIEW-003")
	assertIDs("4k", contracts.ListMoviesRequest{Resolution: "4k"}, "VIEW-001")
	assertIDs("rating", contracts.ListMoviesRequest{UserRating: &five}, "VIEW-001")
	assertIDs("min rating", contracts.ListMoviesRequest{UserRating: &four}, "VIEW-001", "VIEW-002")
	assertIDs("recent", contracts.ListMoviesRequest{AddedAfter: "2026-07-01T00:00:00Z"}, "VIEW-001")
	assertIDs("unwatched", contracts.ListMoviesRequest{PlayState: "unwatched"}, "VIEW-001", "VIEW-004")
	assertIDs("in progress", contracts.ListMoviesRequest{PlayState: "in-progress"}, "VIEW-002")
	assertIDs("completed", contracts.ListMoviesRequest{PlayState: "completed"}, "VIEW-003")
	assertIDs("unrated", contracts.ListMoviesRequest{Unrated: true}, "VIEW-004")
	assertIDs("unknown year", contracts.ListMoviesRequest{Year: "unknown"}, "VIEW-004")
	assertIDs("short runtime", contracts.ListMoviesRequest{Runtime: "short"}, "VIEW-004")
	assertIDs("unscraped", contracts.ListMoviesRequest{Catalog: "unscraped"}, "VIEW-004")
	assertIDs("no cover", contracts.ListMoviesRequest{Catalog: "no-cover"}, "VIEW-001", "VIEW-002", "VIEW-003", "VIEW-004")

	page, err := store.ListMovies(ctx, contracts.ListMoviesRequest{UserRating: &five, Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("rating list: items=%#v err=%v", page.Items, err)
	}
	if page.Items[0].UserRating == nil || *page.Items[0].UserRating != 5 {
		t.Fatalf("list item lost explicit user rating: %#v", page.Items[0])
	}
}
