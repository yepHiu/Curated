package storage

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

func addWatchTimeMovieForTest(t *testing.T, store *SQLiteStore, code string) string {
	t.Helper()
	outcome, err := store.PersistScanMovie(context.Background(), contracts.ScanFileResultDTO{
		TaskID:   "watch-time-test",
		Path:     filepath.Join(t.TempDir(), code+".mp4"),
		FileName: code + ".mp4",
		Number:   code,
	})
	if err != nil {
		t.Fatal(err)
	}
	return outcome.MovieID
}

func TestPlaybackWatchTimeDailyAggregation(t *testing.T) {
	t.Parallel()
	store := newMigratedTestStore(t)
	ctx := context.Background()
	movieA := addWatchTimeMovieForTest(t, store, "WT-001")
	movieB := addWatchTimeMovieForTest(t, store, "WT-002")
	today := time.Now().UTC()
	day0 := today.Format("2006-01-02")
	day1 := today.AddDate(0, 0, -1).Format("2006-01-02")

	if err := store.AddPlaybackWatchTime(ctx, movieA, day1, 120); err != nil {
		t.Fatal(err)
	}
	if err := store.AddPlaybackWatchTime(ctx, movieA, day1, 30); err != nil {
		t.Fatal(err)
	}
	if err := store.AddPlaybackWatchTime(ctx, movieB, day1, 60); err != nil {
		t.Fatal(err)
	}
	if err := store.AddPlaybackWatchTime(ctx, movieA, day0, 240); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListPlaybackWatchTimeDaily(ctx, 91)
	if err != nil {
		t.Fatal(err)
	}

	want := []PlaybackWatchTimeDailyRow{
		{DayKey: day0, WatchedSec: 240},
		{DayKey: day1, WatchedSec: 210},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("daily watch time = %+v, want %+v", got, want)
	}
}

func TestPlaybackWatchTimeRejectsInvalidRows(t *testing.T) {
	t.Parallel()
	store := newMigratedTestStore(t)
	ctx := context.Background()
	movieID := addWatchTimeMovieForTest(t, store, "WT-INVALID")

	if err := store.AddPlaybackWatchTime(ctx, movieID, "20260430", 10); err == nil {
		t.Fatal("expected invalid day key to fail")
	}
	if err := store.AddPlaybackWatchTime(ctx, movieID, "2026-04-30", 0); err == nil {
		t.Fatal("expected non-positive watch time to fail")
	}
	if err := store.AddPlaybackWatchTime(ctx, "missing-movie", "2026-04-30", 10); err == nil {
		t.Fatal("expected missing movie to fail")
	}
}
