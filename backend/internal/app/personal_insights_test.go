package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

func newPersonalInsightsTestApp(t *testing.T) *App {
	t.Helper()
	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "personal-insights-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return &App{store: store}
}

func TestPersonalInsightsWindowUsesRequestedTimezoneAndNullEmptyRates(t *testing.T) {
	app := newPersonalInsightsTestApp(t)
	originalNow := personalInsightsNow
	personalInsightsNow = func() time.Time {
		return time.Date(2026, 7, 21, 16, 30, 0, 0, time.UTC)
	}
	t.Cleanup(func() { personalInsightsNow = originalNow })

	overview, err := app.GetPersonalInsightsOverview(context.Background(), "30d", "Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	if overview.From != "2026-06-23" || overview.To != "2026-07-22" ||
		overview.Timezone != "Asia/Shanghai" || overview.DataSince != nil ||
		overview.CompletionRate != nil || overview.AverageUserRating != nil ||
		overview.CompletionThreshold != 0.90 {
		t.Fatalf("unexpected overview window: %+v", overview)
	}

	all, err := app.GetPersonalInsightsOverview(context.Background(), "all", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if all.From != "2026-07-21" || all.To != "2026-07-21" {
		t.Fatalf("unexpected empty all-time window: %+v", all)
	}
}

func TestPersonalInsightsRejectsInvalidRangeTimezoneDimensionAndLimit(t *testing.T) {
	app := newPersonalInsightsTestApp(t)
	ctx := context.Background()
	if _, err := app.GetPersonalInsightsOverview(ctx, "7d", "UTC"); !errors.Is(err, contracts.ErrPersonalInsightsInvalidRange) {
		t.Fatalf("invalid range error=%v", err)
	}
	if _, err := app.GetPersonalInsightsOverview(ctx, "30d", "Mars/Olympus"); !errors.Is(err, contracts.ErrPersonalInsightsInvalidTimezone) {
		t.Fatalf("invalid timezone error=%v", err)
	}
	if _, err := app.GetPersonalInsightsBreakdown(ctx, "30d", "UTC", "director", 10); !errors.Is(err, contracts.ErrPersonalInsightsInvalidDimension) {
		t.Fatalf("invalid dimension error=%v", err)
	}
	if _, err := app.GetPersonalInsightsBreakdown(ctx, "30d", "UTC", "actor", 26); !errors.Is(err, contracts.ErrPersonalInsightsInvalidLimit) {
		t.Fatalf("invalid limit error=%v", err)
	}
}
