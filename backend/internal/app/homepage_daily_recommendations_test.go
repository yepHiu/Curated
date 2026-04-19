package app

import (
	"context"
	"fmt"
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
	return fmt.Sprintf("M%02d", i)
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

func TestGenerateHomepageDailyRecommendationsAvoidsLast7DaysWhenInventoryAllows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newHomepageRecommendationFixture(t, 112)
	for i := 1; i <= 98; i++ {
		fixture.mustResaveMovieMetadata(t, testMovieCode(i), 5.0)
	}
	for i := 99; i <= 112; i++ {
		fixture.mustResaveMovieMetadata(t, testMovieCode(i), 1.0)
	}

	for offset := 1; offset <= 7; offset++ {
		dateUTC := fmt.Sprintf("2026-04-%02d", 15-offset)
		start := (offset-1)*14 + 1
		heroIDs := make([]string, 0, 8)
		recommendationIDs := make([]string, 0, 6)
		for i := 0; i < 8; i++ {
			heroIDs = append(heroIDs, strings.ToLower(testMovieCode(start+i)))
		}
		for i := 8; i < 14; i++ {
			recommendationIDs = append(recommendationIDs, strings.ToLower(testMovieCode(start+i)))
		}
		fixture.mustSaveSnapshot(t, dateUTC, heroIDs, recommendationIDs)
	}

	dto, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-04-15")
	if err != nil {
		t.Fatalf("GetOrCreateHomepageDailyRecommendations() error = %v", err)
	}

	recentIDs := make(map[string]struct{}, 98)
	for i := 1; i <= 98; i++ {
		recentIDs[strings.ToLower(testMovieCode(i))] = struct{}{}
	}

	all := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	if len(all) != 14 {
		t.Fatalf("len(all) = %d, want 14", len(all))
	}
	for _, id := range all {
		if _, ok := recentIDs[id]; ok {
			t.Fatalf("last-7-day movie %q reappeared in %#v", id, all)
		}
	}
}

func TestGenerateHomepageDailyRecommendationsFallsBackToShorterRecentWindow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fixture := newHomepageRecommendationFixture(t, 35)

	groupA := []string{"m01", "m02", "m03", "m04", "m05", "m06", "m07"}
	groupB := []string{"m08", "m09", "m10", "m11", "m12", "m13", "m14"}
	groupC := []string{"m15", "m16", "m17", "m18", "m19", "m20", "m21"}
	groupD := []string{"m22", "m23", "m24", "m25", "m26", "m27", "m28"}
	groupE := []string{"m29", "m30", "m31", "m32", "m33", "m34", "m35"}

	for _, id := range groupC {
		fixture.mustResaveMovieMetadata(t, strings.ToUpper(id), 5.0)
	}
	for _, id := range append(append([]string{}, groupD...), groupE...) {
		fixture.mustResaveMovieMetadata(t, strings.ToUpper(id), 3.5)
	}

	fixture.mustSaveSnapshot(t, "2026-04-14",
		append(append([]string{}, groupA...), groupB[:1]...),
		append([]string{}, groupB[1:]...),
	)
	fixture.mustSaveSnapshot(t, "2026-04-13",
		append(append([]string{}, groupB...), groupC[:1]...),
		append([]string{}, groupC[1:]...),
	)
	fixture.mustSaveSnapshot(t, "2026-04-12",
		append(append([]string{}, groupC...), groupA[:1]...),
		append([]string{}, groupA[1:]...),
	)
	fixture.mustSaveSnapshot(t, "2026-04-11",
		append(append([]string{}, groupD...), groupA[:1]...),
		append([]string{}, groupA[1:]...),
	)
	fixture.mustSaveSnapshot(t, "2026-04-10",
		append(append([]string{}, groupD...), groupB[:1]...),
		append([]string{}, groupB[1:]...),
	)
	fixture.mustSaveSnapshot(t, "2026-04-09",
		append(append([]string{}, groupE...), groupC[:1]...),
		append([]string{}, groupC[1:]...),
	)
	fixture.mustSaveSnapshot(t, "2026-04-08",
		append(append([]string{}, groupE...), groupA[:1]...),
		append([]string{}, groupA[1:]...),
	)

	dto, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-04-15")
	if err != nil {
		t.Fatalf("GetOrCreateHomepageDailyRecommendations() error = %v", err)
	}

	all := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	if len(all) != 14 {
		t.Fatalf("len(all) = %d, want 14", len(all))
	}

	disallowed := make(map[string]struct{}, 21)
	for _, id := range append(append([]string{}, groupA...), groupB...) {
		disallowed[id] = struct{}{}
	}
	for _, id := range groupC {
		disallowed[id] = struct{}{}
	}

	for _, id := range all {
		if _, ok := disallowed[id]; ok {
			t.Fatalf("movie from the last 3 days %q reappeared in %#v", id, all)
		}
	}
}
