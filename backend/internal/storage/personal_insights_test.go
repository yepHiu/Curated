package storage

import (
	"context"
	"math"
	"path/filepath"
	"testing"
)

func newPersonalInsightsTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "personal-insights.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedPersonalInsightsMovie(t *testing.T, store *SQLiteStore, id, studio string, rating *float64) {
	t.Helper()
	if _, err := store.db.Exec(`
		INSERT INTO movies (
			id, title, code, studio, summary, added_at, location, resolution, year, user_rating
		) VALUES (?, ?, ?, ?, '', '2026-01-01', ?, '1080p', 2026, ?)`,
		id, id, id, studio, filepath.Join(t.TempDir(), id+".mp4"), rating,
	); err != nil {
		t.Fatal(err)
	}
}

func seedPersonalInsightsWatch(t *testing.T, store *SQLiteStore, day, movieID string, seconds float64) {
	t.Helper()
	if _, err := store.db.Exec(`
		INSERT INTO playback_daily_watch_time (day_key, movie_id, watched_sec, updated_at)
		VALUES (?, ?, ?, '2026-07-21T00:00:00Z')`, day, movieID, seconds); err != nil {
		t.Fatal(err)
	}
}

func TestPersonalInsightsAggregatesOverviewAndBoundedBreakdowns(t *testing.T) {
	store := newPersonalInsightsTestStore(t)
	ctx := context.Background()
	ratingA := 4.5
	ratingC := 3.5
	seedPersonalInsightsMovie(t, store, "movie-a", "Studio Alpha", &ratingA)
	seedPersonalInsightsMovie(t, store, "movie-b", "Studio Beta", nil)
	seedPersonalInsightsMovie(t, store, "movie-c", "Studio Alpha", &ratingC)
	seedPersonalInsightsWatch(t, store, "2026-07-01", "movie-a", 120)
	seedPersonalInsightsWatch(t, store, "2026-07-02", "movie-a", 30)
	seedPersonalInsightsWatch(t, store, "2026-07-02", "movie-b", 50)
	seedPersonalInsightsWatch(t, store, "2026-06-01", "movie-c", 80)
	if _, err := store.db.Exec(`
		INSERT INTO playback_progress (movie_id, position_sec, duration_sec, updated_at)
		VALUES ('movie-a', 90, 100, '2026-07-02T00:00:00Z'),
		       ('movie-b', 20, 100, '2026-07-02T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}

	actorAlice := seedActorMergeActor(t, store, "Alice", actorMergeFields(nil))
	actorBob := seedActorMergeActor(t, store, "Bob", actorMergeFields(nil))
	actorShared := seedActorMergeActor(t, store, "Shared", actorMergeFields(nil))
	for _, pair := range []struct {
		movieID string
		actorID int64
	}{{"movie-a", actorAlice}, {"movie-a", actorShared}, {"movie-b", actorBob}, {"movie-b", actorShared}} {
		if _, err := store.db.Exec(`INSERT INTO movie_actors (movie_id, actor_id) VALUES (?, ?)`, pair.movieID, pair.actorID); err != nil {
			t.Fatal(err)
		}
	}
	for _, tag := range []struct {
		name    string
		typeKey string
		movies  []string
	}{
		{"Action", "nfo", []string{"movie-a", "movie-b"}},
		{"Action", "user", []string{"movie-a"}},
		{"Drama", "nfo", []string{"movie-a"}},
	} {
		result, err := store.db.Exec(`INSERT INTO tags (name, type) VALUES (?, ?)`, tag.name, tag.typeKey)
		if err != nil {
			t.Fatal(err)
		}
		tagID, _ := result.LastInsertId()
		for _, movieID := range tag.movies {
			if _, err := store.db.Exec(`INSERT INTO movie_tags (movie_id, tag_id) VALUES (?, ?)`, movieID, tagID); err != nil {
				t.Fatal(err)
			}
		}
	}

	overview, err := store.PersonalInsightsOverview(ctx, "2026-07-01", "2026-07-02", 0.90)
	if err != nil {
		t.Fatal(err)
	}
	if overview.WatchedSeconds != 200 || overview.StartedMovies != 2 ||
		overview.CompletedMovies != 1 || overview.RatedMovies != 1 ||
		overview.AverageRating == nil || *overview.AverageRating != 4.5 {
		t.Fatalf("unexpected overview: %+v", overview)
	}
	dataSince, err := store.PersonalInsightsDataSince(ctx, "2026-07-02")
	if err != nil || dataSince == nil || *dataSince != "2026-06-01" {
		t.Fatalf("unexpected dataSince=%v err=%v", dataSince, err)
	}

	assertBreakdown := func(dimension string, expected []PersonalInsightsBreakdownAggregate) {
		t.Helper()
		items, err := store.PersonalInsightsBreakdown(ctx, "2026-07-01", "2026-07-02", dimension, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != len(expected) {
			t.Fatalf("%s item count=%d want=%d: %+v", dimension, len(items), len(expected), items)
		}
		for index, want := range expected {
			got := items[index]
			if got.Name != want.Name || got.MovieCount != want.MovieCount || math.Abs(got.WatchedSeconds-want.WatchedSeconds) > 0.001 {
				t.Fatalf("%s item[%d]=%+v want=%+v", dimension, index, got, want)
			}
		}
	}
	assertBreakdown("actor", []PersonalInsightsBreakdownAggregate{
		{Name: "Shared", WatchedSeconds: 200, MovieCount: 2},
		{Name: "Alice", WatchedSeconds: 150, MovieCount: 1},
		{Name: "Bob", WatchedSeconds: 50, MovieCount: 1},
	})
	assertBreakdown("studio", []PersonalInsightsBreakdownAggregate{
		{Name: "Studio Alpha", WatchedSeconds: 150, MovieCount: 1},
		{Name: "Studio Beta", WatchedSeconds: 50, MovieCount: 1},
	})
	assertBreakdown("tag", []PersonalInsightsBreakdownAggregate{
		{Name: "Action", WatchedSeconds: 200, MovieCount: 2},
		{Name: "Drama", WatchedSeconds: 150, MovieCount: 1},
	})
}

func TestPersonalInsightsOverviewReturnsNullDenominatorsForEmptyRange(t *testing.T) {
	store := newPersonalInsightsTestStore(t)
	overview, err := store.PersonalInsightsOverview(context.Background(), "2026-07-01", "2026-07-02", 0.90)
	if err != nil {
		t.Fatal(err)
	}
	if overview.WatchedSeconds != 0 || overview.StartedMovies != 0 || overview.AverageRating != nil {
		t.Fatalf("unexpected empty overview: %+v", overview)
	}
}

func TestPersonalInsightsBreakdownUsesMovieCountNameTieBreakAndLimit(t *testing.T) {
	store := newPersonalInsightsTestStore(t)
	for _, fixture := range []struct {
		id      string
		studio  string
		seconds float64
	}{
		{id: "two-a", studio: "Two", seconds: 30},
		{id: "two-b", studio: "Two", seconds: 30},
		{id: "alpha", studio: "alpha", seconds: 60},
		{id: "zeta", studio: "Zeta", seconds: 60},
	} {
		seedPersonalInsightsMovie(t, store, fixture.id, fixture.studio, nil)
		seedPersonalInsightsWatch(t, store, "2026-07-01", fixture.id, fixture.seconds)
	}

	items, err := store.PersonalInsightsBreakdown(
		context.Background(),
		"2026-07-01",
		"2026-07-01",
		"studio",
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 ||
		items[0].Name != "Two" || items[0].MovieCount != 2 ||
		items[1].Name != "alpha" || items[1].MovieCount != 1 {
		t.Fatalf("unexpected bounded tie order: %+v", items)
	}
}
