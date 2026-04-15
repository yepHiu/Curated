package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/scraper"
	"curated-backend/internal/storage"
)

type homepageRecommendationFixture struct {
	app   *App
	store *storage.SQLiteStore
}

func newHomepageRecommendationFixture(t *testing.T, movieCount int) *homepageRecommendationFixture {
	t.Helper()

	store, err := storage.NewSQLiteStore(filepath.Join(t.TempDir(), "homepage-recommendations.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	for i := 1; i <= movieCount; i++ {
		code := testMovieCode(i)
		path := filepath.Join("D:/Media/JAV/Main", code+".mp4")

		outcome, err := store.PersistScanMovie(ctx, contracts.ScanFileResultDTO{
			TaskID:   "task-" + code,
			Path:     path,
			FileName: code + ".mp4",
			Number:   code,
		})
		if err != nil {
			t.Fatalf("PersistScanMovie(%s) error = %v", code, err)
		}
		if err := store.SaveMovieMetadata(ctx, scraper.Metadata{
			MovieID:        outcome.MovieID,
			Number:         code,
			Title:          "Movie " + code,
			Summary:        "Summary " + code,
			Studio:         "Studio A",
			Actors:         []string{"Actor A"},
			Tags:           []string{"Tag A"},
			RuntimeMinutes: 120,
			Rating:         4.5,
		}); err != nil {
			t.Fatalf("SaveMovieMetadata(%s) error = %v", code, err)
		}
	}

	return &homepageRecommendationFixture{
		app: &App{
			logger: zap.NewNop(),
			store:  store,
		},
		store: store,
	}
}

func (f *homepageRecommendationFixture) mustSaveSnapshot(t *testing.T, dateUTC string, heroMovieIDs []string, recommendationMovieIDs []string) {
	t.Helper()

	err := f.store.UpsertHomepageDailyRecommendationSnapshot(context.Background(), storage.HomepageDailyRecommendationSnapshot{
		DateUTC:                dateUTC,
		HeroMovieIDs:           heroMovieIDs,
		RecommendationMovieIDs: recommendationMovieIDs,
		GeneratedAt:            dateUTC + "T00:00:00Z",
		GenerationVersion:      "v1",
	})
	if err != nil {
		t.Fatalf("UpsertHomepageDailyRecommendationSnapshot() error = %v", err)
	}
}

func (f *homepageRecommendationFixture) mustResaveMovieMetadata(t *testing.T, movieID string, rating float64) {
	t.Helper()

	normalizedMovieID := strings.ToLower(movieID)
	if err := f.store.SaveMovieMetadata(context.Background(), scraper.Metadata{
		MovieID:        normalizedMovieID,
		Number:         strings.ToUpper(movieID),
		Title:          "Movie " + normalizedMovieID,
		Summary:        "Summary " + normalizedMovieID,
		Studio:         "Studio A",
		Actors:         []string{"Actor A"},
		Tags:           []string{"Tag A"},
		RuntimeMinutes: 120,
		Rating:         rating,
	}); err != nil {
		t.Fatalf("SaveMovieMetadata(%s) error = %v", movieID, err)
	}
}

func (f *homepageRecommendationFixture) mustResaveMovieMetadataWithProfile(
	t *testing.T,
	movieID string,
	rating float64,
	studio string,
	actors ...string,
) {
	t.Helper()

	normalizedMovieID := strings.ToLower(movieID)
	if len(actors) == 0 {
		actors = []string{"Actor A"}
	}
	if err := f.store.SaveMovieMetadata(context.Background(), scraper.Metadata{
		MovieID:        normalizedMovieID,
		Number:         strings.ToUpper(movieID),
		Title:          "Movie " + normalizedMovieID,
		Summary:        "Summary " + normalizedMovieID,
		Studio:         studio,
		Actors:         actors,
		Tags:           []string{"Tag A"},
		RuntimeMinutes: 120,
		Rating:         rating,
	}); err != nil {
		t.Fatalf("SaveMovieMetadata(%s) error = %v", movieID, err)
	}
}

func testMovieCode(i int) string {
	if i < 10 {
		return "M0" + string(rune('0'+i))
	}
	if i < 100 {
		tens := rune('0' + (i / 10))
		ones := rune('0' + (i % 10))
		return "M" + string(tens) + string(ones)
	}
	return "M100"
}

func TestGenerateHomepageDailyRecommendationsAvoidsYesterdayWhenInventoryAllows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newHomepageRecommendationFixture(t, 28)
	fixture.mustSaveSnapshot(t, "2026-04-14",
		[]string{"m01", "m02", "m03", "m04", "m05", "m06", "m07", "m08"},
		[]string{"m09", "m10", "m11", "m12", "m13", "m14"},
	)

	dto, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-04-15")
	if err != nil {
		t.Fatalf("GetOrCreateHomepageDailyRecommendations() error = %v", err)
	}

	got := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	disallowed := map[string]struct{}{
		"m01": {}, "m02": {}, "m03": {}, "m04": {}, "m05": {}, "m06": {}, "m07": {}, "m08": {},
		"m09": {}, "m10": {}, "m11": {}, "m12": {}, "m13": {}, "m14": {},
	}
	for _, id := range got {
		if _, ok := disallowed[id]; ok {
			t.Fatalf("today reused yesterday movie %q in %#v", id, got)
		}
	}
}

func TestGenerateHomepageDailyRecommendationsBackfillsYesterdayOnlyAfterExhaustingFreshTitles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newHomepageRecommendationFixture(t, 16)
	fixture.mustSaveSnapshot(t, "2026-04-14",
		[]string{"m01", "m02", "m03", "m04", "m05", "m06", "m07", "m08"},
		[]string{"m09", "m10", "m11", "m12", "m13", "m14"},
	)

	dto, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-04-15")
	if err != nil {
		t.Fatalf("GetOrCreateHomepageDailyRecommendations() error = %v", err)
	}

	all := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	if len(all) != 14 {
		t.Fatalf("len(all) = %d, want 14", len(all))
	}

	freshCount := 0
	for _, id := range all {
		if id == "m15" || id == "m16" {
			freshCount++
		}
	}
	if freshCount != 2 {
		t.Fatalf("freshCount = %d, want 2, all=%#v", freshCount, all)
	}
}

func TestGenerateHomepageDailyRecommendationsPenalizesRecentlyOverexposedTitles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newHomepageRecommendationFixture(t, 30)

	for i := 15; i <= 20; i++ {
		fixture.mustResaveMovieMetadata(t, testMovieCode(i), 4.9)
	}
	for i := 21; i <= 30; i++ {
		fixture.mustResaveMovieMetadata(t, testMovieCode(i), 4.4)
	}

	for _, dateUTC := range []string{"2026-04-10", "2026-04-11", "2026-04-12", "2026-04-13"} {
		fixture.mustSaveSnapshot(t, dateUTC,
			[]string{"m15", "m16", "m17", "m18", "m19", "m20", "m01", "m02"},
			[]string{"m03", "m04", "m05", "m06", "m07", "m08"},
		)
	}
	fixture.mustSaveSnapshot(t, "2026-04-14",
		[]string{"m01", "m02", "m03", "m04", "m05", "m06", "m07", "m08"},
		[]string{"m09", "m10", "m11", "m12", "m13", "m14"},
	)

	dto, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-04-15")
	if err != nil {
		t.Fatalf("GetOrCreateHomepageDailyRecommendations() error = %v", err)
	}

	all := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	selected := make(map[string]struct{}, len(all))
	for _, id := range all {
		selected[id] = struct{}{}
	}

	for i := 21; i <= 30; i++ {
		id := strings.ToLower(testMovieCode(i))
		if _, ok := selected[id]; !ok {
			t.Fatalf("underexposed movie %q missing from %#v", id, all)
		}
	}

	overexposedCount := 0
	for i := 15; i <= 20; i++ {
		if _, ok := selected[strings.ToLower(testMovieCode(i))]; ok {
			overexposedCount++
		}
	}
	if overexposedCount != 4 {
		t.Fatalf("overexposedCount = %d, want 4, all=%#v", overexposedCount, all)
	}
}

func TestGenerateHomepageDailyRecommendationsBalancesActorsAndStudiosAcrossSlate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newHomepageRecommendationFixture(t, 18)

	for i := 1; i <= 8; i++ {
		fixture.mustResaveMovieMetadataWithProfile(
			t,
			testMovieCode(i),
			4.9,
			"Studio Cluster",
			"Actor Cluster",
		)
	}
	for i := 9; i <= 18; i++ {
		fixture.mustResaveMovieMetadataWithProfile(
			t,
			testMovieCode(i),
			4.8,
			"Studio "+testMovieCode(i),
			"Actor "+testMovieCode(i),
		)
	}

	dto, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-04-15")
	if err != nil {
		t.Fatalf("GetOrCreateHomepageDailyRecommendations() error = %v", err)
	}

	all := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	selected := make(map[string]struct{}, len(all))
	for _, id := range all {
		selected[id] = struct{}{}
	}

	for i := 9; i <= 18; i++ {
		id := strings.ToLower(testMovieCode(i))
		if _, ok := selected[id]; !ok {
			t.Fatalf("diverse movie %q missing from %#v", id, all)
		}
	}

	clusterCount := 0
	for i := 1; i <= 8; i++ {
		if _, ok := selected[strings.ToLower(testMovieCode(i))]; ok {
			clusterCount++
		}
	}
	if clusterCount != 4 {
		t.Fatalf("clusterCount = %d, want 4, all=%#v", clusterCount, all)
	}
}
