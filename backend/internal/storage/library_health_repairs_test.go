package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/contracts"
)

func TestMovieMetadataScrapeAttemptKeepsNewestTaskOutcome(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "attempts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
		TaskID: "scan-attempt", Path: filepath.Join(t.TempDir(), "ATTEMPT-001.mp4"), FileName: "ATTEMPT-001.mp4", Number: "ATTEMPT-001",
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	if err := store.StartMovieMetadataScrapeAttempt(ctx, "attempt-001", "scrape-old", now); err != nil {
		t.Fatal(err)
	}
	if err := store.StartMovieMetadataScrapeAttempt(ctx, "attempt-001", "scrape-new", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishMovieMetadataScrapeAttempt(ctx, MovieMetadataScrapeAttempt{
		MovieID: "attempt-001", TaskID: "scrape-old", Status: "failed", ErrorMessage: "stale failure",
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishMovieMetadataScrapeAttempt(ctx, MovieMetadataScrapeAttempt{
		MovieID: "attempt-001", TaskID: "scrape-new", Status: "failed", ErrorCode: "SCRAPER_RUN_FAILED",
		ErrorCategory: "connect_timeout", ErrorMessage: "latest failure", Provider: "provider-a",
	}, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.InspectLibraryHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.MetadataAttempts) != 1 {
		t.Fatalf("attempts = %#v", snapshot.MetadataAttempts)
	}
	got := snapshot.MetadataAttempts[0]
	if got.TaskID != "scrape-new" || got.ErrorMessage != "latest failure" || got.ErrorCategory != "connect_timeout" {
		t.Fatalf("attempt = %#v", got)
	}
	if err := store.StartMovieMetadataScrapeAttempt(ctx, "attempt-001", "scrape-success", now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishMovieMetadataScrapeAttempt(ctx, MovieMetadataScrapeAttempt{
		MovieID: "attempt-001", TaskID: "scrape-success", Status: "completed", Provider: "provider-b",
	}, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	snapshot, err = store.InspectLibraryHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.MetadataAttempts) != 0 {
		t.Fatalf("successful latest attempt still reported failed: %#v", snapshot.MetadataAttempts)
	}
}

func TestLibraryHealthRepairRunPersistsItemsCountersAndInterruption(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "repairs.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"REPAIR-001", "REPAIR-002"} {
		if _, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
			TaskID: "scan-" + code, Path: filepath.Join(t.TempDir(), code+".mp4"), FileName: code + ".mp4", Number: code,
		}); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 7, 20, 2, 0, 0, 0, time.UTC)
	run := LibraryHealthRepairRun{
		RepairID: "repair-test", TaskID: "task-repair", Action: "rescrape_metadata",
		CategoriesJSON: `["metadata_missing"]`, CreatedAt: now.Format(time.RFC3339),
		Items: []LibraryHealthRepairItem{
			{Ordinal: 0, FindingID: "finding-1", Category: "metadata_missing", MovieID: "repair-001", Label: "REPAIR-001"},
			{Ordinal: 1, FindingID: "finding-2", Category: "metadata_missing", MovieID: "repair-002", Label: "REPAIR-002"},
		},
	}
	if err := store.CreateLibraryHealthRepairRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := store.StartLibraryHealthRepairRun(ctx, run.RepairID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.QueueLibraryHealthRepairItem(ctx, run.RepairID, 0, "child-1", now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteLibraryHealthRepairItem(ctx, run.RepairID, 0, "succeeded", "", "", now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.InterruptLibraryHealthRepairRuns(ctx, now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetLibraryHealthRepairRun(ctx, run.RepairID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contracts.TaskCancelled || got.TotalItems != 2 || got.CompletedItems != 2 || got.SucceededItems != 1 || got.FailedItems != 1 {
		t.Fatalf("run = %#v", got)
	}
	if got.Items[0].Status != "succeeded" || got.Items[1].Status != "cancelled" || got.Items[1].ErrorCode != "HEALTH_REPAIR_INTERRUPTED" {
		t.Fatalf("items = %#v", got.Items)
	}
	if _, err := store.GetLibraryHealthRepairRun(ctx, "missing"); !errors.Is(err, ErrLibraryHealthRepairNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}
