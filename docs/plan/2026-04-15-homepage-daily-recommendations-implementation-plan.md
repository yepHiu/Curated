# Homepage Daily Recommendations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace homepage frontend-only `Hero` and `今日推荐` selection with backend-persisted UTC daily recommendation snapshots that are identical across devices, rotate automatically across UTC day boundaries, avoid repeating the previous UTC day when inventory allows, and degrade gracefully when inventory is too small.

**Architecture:** Add a backend-owned daily snapshot pipeline in `backend/internal/storage`, `backend/internal/app`, and `backend/internal/server`, exposed through `GET /api/homepage/recommendations`. Keep `recentMovies`, `continueWatching`, and `tasteRadar` in the existing frontend homepage model, but feed `heroMovies` and `recommendations` from the backend snapshot via a small frontend loader and UTC rollover refresh seam.

**Tech Stack:** Go, SQLite, existing `backend/internal/*` store/server/contracts layers, Vue 3, TypeScript, Vite, Vitest, pnpm.

---

### Task 1: Add SQLite Snapshot Persistence

**Files:**
- Create: `backend/internal/storage/migrations/0015_homepage_daily_recommendations.sql`
- Create: `backend/internal/storage/homepage_daily_recommendations.go`
- Create: `backend/internal/storage/homepage_daily_recommendations_test.go`

- [ ] **Step 1: Write the failing storage tests**

```go
package storage

import (
	"context"
	"testing"
)

func TestDailyHomepageRecommendationSnapshotLifecycle(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)

	snapshot := HomepageDailyRecommendationSnapshot{
		DateUTC:               "2026-04-15",
		HeroMovieIDs:          []string{"m1", "m2"},
		RecommendationMovieIDs: []string{"m3", "m4"},
		GeneratedAt:           "2026-04-15T00:00:00Z",
		GenerationVersion:     "v1",
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
}

func TestHomepageDailyRecommendationSnapshotReturnsMissingForUnknownDay(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLiteStore(t)

	_, ok, err := store.GetHomepageDailyRecommendationSnapshot(ctx, "2026-04-16")
	if err != nil {
		t.Fatalf("GetHomepageDailyRecommendationSnapshot() error = %v", err)
	}
	if ok {
		t.Fatalf("expected missing snapshot")
	}
}
```

- [ ] **Step 2: Run the storage tests to verify RED**

Run:

```powershell
cd backend; go test ./internal/storage -run "TestDailyHomepageRecommendationSnapshot|TestHomepageDailyRecommendationSnapshotReturnsMissing"
```

Expected: FAIL because the migration, snapshot row type, and store methods do not exist yet.

- [ ] **Step 3: Add the migration and storage methods**

```sql
CREATE TABLE IF NOT EXISTS homepage_daily_recommendations (
  date_utc TEXT PRIMARY KEY,
  hero_movie_ids_json TEXT NOT NULL,
  recommendation_movie_ids_json TEXT NOT NULL,
  generated_at TEXT NOT NULL,
  generation_version TEXT NOT NULL
);
```

```go
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
)

type HomepageDailyRecommendationSnapshot struct {
	DateUTC                string
	HeroMovieIDs           []string
	RecommendationMovieIDs []string
	GeneratedAt            string
	GenerationVersion      string
}

func (s *SQLiteStore) GetHomepageDailyRecommendationSnapshot(ctx context.Context, dateUTC string) (HomepageDailyRecommendationSnapshot, bool, error) {
	var heroJSON string
	var recoJSON string
	var row HomepageDailyRecommendationSnapshot
	err := s.db.QueryRowContext(ctx, `
		SELECT date_utc, hero_movie_ids_json, recommendation_movie_ids_json, generated_at, generation_version
		FROM homepage_daily_recommendations
		WHERE date_utc = ?
	`, dateUTC).Scan(&row.DateUTC, &heroJSON, &recoJSON, &row.GeneratedAt, &row.GenerationVersion)
	if err == sql.ErrNoRows {
		return HomepageDailyRecommendationSnapshot{}, false, nil
	}
	if err != nil {
		return HomepageDailyRecommendationSnapshot{}, false, err
	}
	if err := json.Unmarshal([]byte(heroJSON), &row.HeroMovieIDs); err != nil {
		return HomepageDailyRecommendationSnapshot{}, false, err
	}
	if err := json.Unmarshal([]byte(recoJSON), &row.RecommendationMovieIDs); err != nil {
		return HomepageDailyRecommendationSnapshot{}, false, err
	}
	return row, true, nil
}

func (s *SQLiteStore) UpsertHomepageDailyRecommendationSnapshot(ctx context.Context, row HomepageDailyRecommendationSnapshot) error {
	heroJSON, err := json.Marshal(row.HeroMovieIDs)
	if err != nil {
		return err
	}
	recoJSON, err := json.Marshal(row.RecommendationMovieIDs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO homepage_daily_recommendations (
			date_utc,
			hero_movie_ids_json,
			recommendation_movie_ids_json,
			generated_at,
			generation_version
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(date_utc) DO UPDATE SET
			hero_movie_ids_json = excluded.hero_movie_ids_json,
			recommendation_movie_ids_json = excluded.recommendation_movie_ids_json,
			generated_at = excluded.generated_at,
			generation_version = excluded.generation_version
	`, row.DateUTC, string(heroJSON), string(recoJSON), row.GeneratedAt, row.GenerationVersion)
	return err
}
```

- [ ] **Step 4: Run the storage tests to verify GREEN**

Run:

```powershell
cd backend; go test ./internal/storage -run "TestDailyHomepageRecommendationSnapshot|TestHomepageDailyRecommendationSnapshotReturnsMissing"
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/storage/migrations/0015_homepage_daily_recommendations.sql backend/internal/storage/homepage_daily_recommendations.go backend/internal/storage/homepage_daily_recommendations_test.go
git commit -m "feat: add homepage daily recommendation snapshot storage"
```

### Task 2: Add Backend Recommendation Generation Logic

**Files:**
- Create: `backend/internal/app/homepage_daily_recommendations.go`
- Create: `backend/internal/app/homepage_daily_recommendations_test.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/storage/library_repository.go`

- [ ] **Step 1: Write the failing app tests**

```go
package app

import (
	"context"
	"testing"
)

func TestGenerateHomepageDailyRecommendationsAvoidsYesterdayWhenInventoryAllows(t *testing.T) {
	ctx := context.Background()
	fixture := newHomepageRecommendationFixture(t)

	yesterday := fixture.mustSaveSnapshot("2026-04-14",
		[]string{"m01", "m02", "m03", "m04", "m05", "m06", "m07", "m08"},
		[]string{"m09", "m10", "m11", "m12", "m13", "m14"},
	)
	_ = yesterday

	dto, err := fixture.app.GetOrCreateHomepageDailyRecommendations(ctx, "2026-04-15")
	if err != nil {
		t.Fatalf("GetOrCreateHomepageDailyRecommendations() error = %v", err)
	}

	got := append(append([]string{}, dto.HeroMovieIDs...), dto.RecommendationMovieIDs...)
	for _, disallowed := range []string{"m01", "m02", "m03", "m04", "m05", "m06", "m07", "m08", "m09", "m10", "m11", "m12", "m13", "m14"} {
		for _, id := range got {
			if id == disallowed {
				t.Fatalf("today reused yesterday movie %q in %#v", disallowed, got)
			}
		}
	}
}

func TestGenerateHomepageDailyRecommendationsBackfillsYesterdayOnlyAfterExhaustingFreshTitles(t *testing.T) {
	ctx := context.Background()
	fixture := newHomepageRecommendationFixtureWithMovieCount(t, 16)

	fixture.mustSaveSnapshot("2026-04-14",
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
	if all[0] == "m01" && all[1] == "m02" {
		t.Fatalf("expected fresh titles to be used before backfill, got %#v", all)
	}
}
```

- [ ] **Step 2: Run the app tests to verify RED**

Run:

```powershell
cd backend; go test ./internal/app -run "TestGenerateHomepageDailyRecommendations"
```

Expected: FAIL because the generator, helper fixture, and app entrypoint do not exist yet.

- [ ] **Step 3: Implement the generator and app seam**

```go
package app

import (
	"context"
	"time"

	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

const homepageDailyRecommendationGenerationVersion = "v1"

func (a *App) GetOrCreateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error) {
	if dateUTC == "" {
		dateUTC = time.Now().UTC().Format("2006-01-02")
	}

	if snapshot, ok, err := a.store.GetHomepageDailyRecommendationSnapshot(ctx, dateUTC); err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	} else if ok {
		return contracts.HomepageDailyRecommendationsDTO{
			DateUTC:                snapshot.DateUTC,
			GeneratedAt:            snapshot.GeneratedAt,
			GenerationVersion:      snapshot.GenerationVersion,
			HeroMovieIDs:           snapshot.HeroMovieIDs,
			RecommendationMovieIDs: snapshot.RecommendationMovieIDs,
		}, nil
	}

	generated, err := a.generateHomepageDailyRecommendations(ctx, dateUTC)
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}
	if err := a.store.UpsertHomepageDailyRecommendationSnapshot(ctx, storage.HomepageDailyRecommendationSnapshot{
		DateUTC:                generated.DateUTC,
		HeroMovieIDs:           generated.HeroMovieIDs,
		RecommendationMovieIDs: generated.RecommendationMovieIDs,
		GeneratedAt:            generated.GeneratedAt,
		GenerationVersion:      generated.GenerationVersion,
	}); err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}
	return generated, nil
}
```

```go
func (a *App) generateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error) {
	movies, err := a.store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 10000, Offset: 0})
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	yesterdayUTC := mustPreviousUTCDate(dateUTC)
	yesterday, _, err := a.store.GetHomepageDailyRecommendationSnapshot(ctx, yesterdayUTC)
	if err != nil {
		return contracts.HomepageDailyRecommendationsDTO{}, err
	}

	state := buildHomepageRecommendationState(movies, yesterday)
	heroIDs := selectHomepageDailySlate(state, homepageSectionHero, 8)
	recoIDs := selectHomepageDailySlate(state, homepageSectionRecommendation, 6)

	return contracts.HomepageDailyRecommendationsDTO{
		DateUTC:                dateUTC,
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		GenerationVersion:      homepageDailyRecommendationGenerationVersion,
		HeroMovieIDs:           heroIDs,
		RecommendationMovieIDs: recoIDs,
	}, nil
}
```

- [ ] **Step 4: Run the app tests to verify GREEN**

Run:

```powershell
cd backend; go test ./internal/app -run "TestGenerateHomepageDailyRecommendations"
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/app/homepage_daily_recommendations.go backend/internal/app/homepage_daily_recommendations_test.go backend/internal/app/app.go backend/internal/storage/library_repository.go
git commit -m "feat: add homepage daily recommendation generation"
```

### Task 3: Expose Snapshot DTO and HTTP Endpoint

**Files:**
- Modify: `backend/internal/contracts/contracts.go`
- Create: `backend/internal/server/homepage_daily_recommendations_handlers.go`
- Create: `backend/internal/server/homepage_daily_recommendations_test.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/app/app.go`

- [ ] **Step 1: Write the failing server tests**

```go
package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHomepageRecommendationsReturnsPersistedSnapshot(t *testing.T) {
	fixture := newHomepageRecommendationsServerFixture(t)
	fixture.mustSaveSnapshot("2026-04-15",
		[]string{"m01", "m02", "m03", "m04", "m05", "m06", "m07", "m08"},
		[]string{"m09", "m10", "m11", "m12", "m13", "m14"},
	)

	resp, err := http.Get(fixture.server.URL + "/api/homepage/recommendations")
	if err != nil {
		t.Fatalf("GET homepage recommendations: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
	}

	var dto struct {
		DateUTC string   `json:"dateUtc"`
		Hero    []string `json:"heroMovieIds"`
		Reco    []string `json:"recommendationMovieIds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if dto.DateUTC != "2026-04-15" {
		t.Fatalf("DateUTC = %q, want %q", dto.DateUTC, "2026-04-15")
	}
	if len(dto.Hero) != 8 || len(dto.Reco) != 6 {
		t.Fatalf("hero=%d reco=%d", len(dto.Hero), len(dto.Reco))
	}
}
```

- [ ] **Step 2: Run the server tests to verify RED**

Run:

```powershell
cd backend; go test ./internal/server -run "TestHomepageRecommendationsReturnsPersistedSnapshot"
```

Expected: FAIL because the DTO, handler, route wiring, and app dependency seam do not exist yet.

- [ ] **Step 3: Add the DTO, server dependency, and endpoint**

```go
type HomepageDailyRecommendationsDTO struct {
	DateUTC                string   `json:"dateUtc"`
	GeneratedAt            string   `json:"generatedAt"`
	GenerationVersion      string   `json:"generationVersion,omitempty"`
	HeroMovieIDs           []string `json:"heroMovieIds"`
	RecommendationMovieIDs []string `json:"recommendationMovieIds"`
}
```

```go
type HomepageRecommendationsProvider interface {
	GetOrCreateHomepageDailyRecommendations(ctx context.Context, dateUTC string) (contracts.HomepageDailyRecommendationsDTO, error)
}
```

```go
func (h *Handler) handleGetHomepageRecommendations(w http.ResponseWriter, r *http.Request) {
	dto, err := h.homepageRecommendations.GetOrCreateHomepageDailyRecommendations(r.Context(), "")
	if err != nil {
		h.logger.Error("get homepage recommendations", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load homepage recommendations")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}
```

```go
mux.HandleFunc("GET /api/homepage/recommendations", h.handleGetHomepageRecommendations)
```

- [ ] **Step 4: Run the server tests to verify GREEN**

Run:

```powershell
cd backend; go test ./internal/server -run "TestHomepageRecommendationsReturnsPersistedSnapshot"
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/contracts/contracts.go backend/internal/server/homepage_daily_recommendations_handlers.go backend/internal/server/homepage_daily_recommendations_test.go backend/internal/server/server.go backend/internal/app/app.go
git commit -m "feat: expose homepage daily recommendations endpoint"
```

### Task 4: Add Frontend API and Snapshot Loader Seam

**Files:**
- Modify: `src/api/types.ts`
- Modify: `src/api/endpoints.ts`
- Modify: `src/services/contracts/library-service.ts`
- Modify: `src/services/adapters/web/web-library-service.ts`
- Modify: `src/services/adapters/mock/mock-library-service.ts`
- Create: `src/composables/use-homepage-daily-recommendations.ts`
- Create: `src/composables/use-homepage-daily-recommendations.test.ts`

- [ ] **Step 1: Write the failing frontend seam tests**

```ts
import { describe, expect, it, vi } from "vitest"
import { nextTick, ref } from "vue"
import { useHomepageDailyRecommendations } from "./use-homepage-daily-recommendations"

describe("useHomepageDailyRecommendations", () => {
  it("loads backend snapshot on mount and exposes ids", async () => {
    const fetchSnapshot = vi.fn().mockResolvedValue({
      dateUtc: "2026-04-15",
      generatedAt: "2026-04-15T00:00:00Z",
      heroMovieIds: ["m1", "m2"],
      recommendationMovieIds: ["m3", "m4"],
    })

    const state = useHomepageDailyRecommendations({
      fetchSnapshot,
      currentUtcDateRef: ref("2026-04-15"),
    })

    await state.refresh()
    expect(fetchSnapshot).toHaveBeenCalledTimes(1)
    expect(state.snapshot.value?.heroMovieIds).toEqual(["m1", "m2"])
  })

  it("re-fetches when the UTC day key changes", async () => {
    const currentUtcDateRef = ref("2026-04-15")
    const fetchSnapshot = vi.fn()
      .mockResolvedValueOnce({
        dateUtc: "2026-04-15",
        generatedAt: "2026-04-15T00:00:00Z",
        heroMovieIds: ["m1"],
        recommendationMovieIds: ["m2"],
      })
      .mockResolvedValueOnce({
        dateUtc: "2026-04-16",
        generatedAt: "2026-04-16T00:00:00Z",
        heroMovieIds: ["m3"],
        recommendationMovieIds: ["m4"],
      })

    const state = useHomepageDailyRecommendations({
      fetchSnapshot,
      currentUtcDateRef,
    })

    await state.refresh()
    currentUtcDateRef.value = "2026-04-16"
    await nextTick()

    expect(fetchSnapshot).toHaveBeenCalledTimes(2)
    expect(state.snapshot.value?.dateUtc).toBe("2026-04-16")
  })
})
```

- [ ] **Step 2: Run the frontend seam tests to verify RED**

Run:

```powershell
pnpm test -- src/composables/use-homepage-daily-recommendations.test.ts
```

Expected: FAIL because the DTO, endpoint wrapper, service seam, and composable do not exist yet.

- [ ] **Step 3: Add API types, service method, and composable**

```ts
export interface HomepageDailyRecommendationsDTO {
  dateUtc: string
  generatedAt: string
  generationVersion?: string
  heroMovieIds: string[]
  recommendationMovieIds: string[]
}
```

```ts
getHomepageDailyRecommendations(): Promise<HomepageDailyRecommendationsDTO> {
  return httpClient.get<HomepageDailyRecommendationsDTO>("/homepage/recommendations")
}
```

```ts
export function useHomepageDailyRecommendations(opts: {
  fetchSnapshot: () => Promise<HomepageDailyRecommendationsDTO>
  currentUtcDateRef: Ref<string>
}) {
  const snapshot = ref<HomepageDailyRecommendationsDTO | null>(null)
  const loading = ref(false)
  const error = ref<unknown>(null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      snapshot.value = await opts.fetchSnapshot()
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  watch(opts.currentUtcDateRef, async (next, prev) => {
    if (next !== prev) {
      await refresh()
    }
  })

  return { snapshot, loading, error, refresh }
}
```

- [ ] **Step 4: Run the frontend seam tests to verify GREEN**

Run:

```powershell
pnpm test -- src/composables/use-homepage-daily-recommendations.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add src/api/types.ts src/api/endpoints.ts src/services/contracts/library-service.ts src/services/adapters/web/web-library-service.ts src/services/adapters/mock/mock-library-service.ts src/composables/use-homepage-daily-recommendations.ts src/composables/use-homepage-daily-recommendations.test.ts
git commit -m "feat: add frontend homepage recommendation snapshot seam"
```

### Task 5: Wire Homepage View to Backend Snapshot and Preserve Fallback

**Files:**
- Modify: `src/lib/homepage-portal.ts`
- Modify: `src/lib/homepage-portal.test.ts`
- Modify: `src/views/HomeView.vue`
- Modify: `src/views/HomeView.test.ts`

- [ ] **Step 1: Write the failing homepage integration tests**

```ts
import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import HomeView from "./HomeView.vue"

describe("HomeView", () => {
  it("prefers backend snapshot ids for hero and recommendation sections", async () => {
    const wrapper = mount(HomeView)
    await vi.dynamicImportSettled()

    expect(wrapper.text()).toContain("Movie m8")
    expect(wrapper.text()).toContain("Movie m14")
  })

  it("falls back to local homepage assembly when backend snapshot loading fails", async () => {
    const wrapper = mount(HomeView)
    await vi.dynamicImportSettled()

    expect(wrapper.find("[data-home-hero]").exists()).toBe(true)
    expect(wrapper.findAll("[data-hero-progress-item]").length).toBeGreaterThan(0)
  })
})
```

- [ ] **Step 2: Run the homepage tests to verify RED**

Run:

```powershell
pnpm test -- src/views/HomeView.test.ts src/lib/homepage-portal.test.ts
```

Expected: FAIL because `HomeView` does not yet read the backend snapshot seam and `homepage-portal.ts` still builds `heroMovies` and `recommendations` internally only.

- [ ] **Step 3: Refactor the homepage portal model to accept snapshot overrides**

```ts
export interface HomepageDailyRecommendationSelection {
  heroMovieIds: string[]
  recommendationMovieIds: string[]
}

export interface BuildHomepagePortalInput {
  movies: readonly Movie[]
  playbackEntries?: readonly PlaybackProgressEntry[]
  dailySelection?: HomepageDailyRecommendationSelection | null
  daySeed?: string
  heroLimit?: number
  recentLimit?: number
  recommendationLimit?: number
  continueLimit?: number
  tasteLimitPerKind?: number
}
```

```ts
const heroMovies = dailySelection
  ? mapMovieIDsToMovies(dailySelection.heroMovieIds, activeMovies)
  : fillHeroMovies(seededHeroOrder(heroPool, stableDateSeed(daySeed)), heroLimit)

const recommendations = dailySelection
  ? mapRecommendationEntriesFromIDs(dailySelection.recommendationMovieIds, activeMovies, preference)
  : buildRecommendationEntriesFallback(/* existing logic */)
```

```ts
const snapshotState = useHomepageDailyRecommendations({
  fetchSnapshot: () => libraryService.getHomepageDailyRecommendations(),
  currentUtcDateRef: useCurrentUtcDayKey({ intervalMs: 60_000 }),
})

const portalModel = computed(() =>
  buildHomepagePortalModel({
    movies: libraryService.movies.value,
    playbackEntries: listSortedByUpdatedDesc(),
    dailySelection: snapshotState.snapshot.value
      ? {
          heroMovieIds: snapshotState.snapshot.value.heroMovieIds,
          recommendationMovieIds: snapshotState.snapshot.value.recommendationMovieIds,
        }
      : null,
    heroLimit: 8,
  }),
)
```

- [ ] **Step 4: Run the homepage tests to verify GREEN**

Run:

```powershell
pnpm test -- src/views/HomeView.test.ts src/lib/homepage-portal.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add src/lib/homepage-portal.ts src/lib/homepage-portal.test.ts src/views/HomeView.vue src/views/HomeView.test.ts
git commit -m "feat: drive homepage hero and recommendations from daily snapshots"
```

### Task 6: Add UTC Rollover Refresh and Endpoint Failure Fallback

**Files:**
- Create: `src/lib/current-utc-day-key.ts`
- Create: `src/lib/current-utc-day-key.test.ts`
- Modify: `src/views/HomeView.vue`
- Modify: `src/views/HomeView.test.ts`
- Modify: `src/services/adapters/web/web-library-service.ts`

- [ ] **Step 1: Write the failing UTC rollover tests**

```ts
import { describe, expect, it, vi } from "vitest"
import { nextTick, ref } from "vue"
import { mount } from "@vue/test-utils"
import HomeView from "./HomeView.vue"

describe("HomeView UTC rollover", () => {
  it("re-fetches daily snapshot after the UTC day changes", async () => {
    vi.useFakeTimers()
    const wrapper = mount(HomeView)

    vi.setSystemTime(new Date("2026-04-16T00:00:30.000Z"))
    vi.advanceTimersByTime(60_000)
    await nextTick()

    expect(wrapper.find("[data-home-hero]").exists()).toBe(true)
  })
})
```

- [ ] **Step 2: Run the UTC rollover tests to verify RED**

Run:

```powershell
pnpm test -- src/lib/current-utc-day-key.test.ts src/views/HomeView.test.ts
```

Expected: FAIL because the UTC day key helper and rollover-triggered refresh are not implemented yet.

- [ ] **Step 3: Add the UTC day key helper and fallback rules**

```ts
import { onBeforeUnmount, ref } from "vue"

export function useCurrentUtcDayKey(opts: { intervalMs?: number } = {}) {
  const intervalMs = opts.intervalMs ?? 60_000
  const key = ref(new Date().toISOString().slice(0, 10))

  const timer = window.setInterval(() => {
    const next = new Date().toISOString().slice(0, 10)
    if (next !== key.value) {
      key.value = next
    }
  }, intervalMs)

  onBeforeUnmount(() => {
    window.clearInterval(timer)
  })

  return key
}
```

```ts
async function refresh() {
  loading.value = true
  try {
    snapshot.value = await opts.fetchSnapshot()
    fallbackMode.value = false
  } catch (err) {
    error.value = err
    if (!snapshot.value) {
      fallbackMode.value = true
    }
  } finally {
    loading.value = false
  }
}
```

- [ ] **Step 4: Run the UTC rollover tests to verify GREEN**

Run:

```powershell
pnpm test -- src/lib/current-utc-day-key.test.ts src/views/HomeView.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add src/lib/current-utc-day-key.ts src/lib/current-utc-day-key.test.ts src/views/HomeView.vue src/views/HomeView.test.ts src/services/adapters/web/web-library-service.ts
git commit -m "feat: refresh homepage daily snapshots on UTC rollover"
```

### Task 7: Sync Docs and Verify End-to-End

**Files:**
- Modify: `docs/plan/2026-04-15-homepage-daily-recommendations-design.md`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `README.md`
- Modify: `API.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update docs for the new endpoint and behavior**

```md
- Homepage Hero and 今日推荐 now come from backend-owned UTC daily snapshots.
- Endpoint: `GET /api/homepage/recommendations`
- Snapshot results are consistent across devices and browsers when `VITE_USE_WEB_API=true`.
- If the endpoint is unavailable on first load, the frontend falls back to local assembly so the homepage stays usable.
```

- [ ] **Step 2: Run targeted frontend verification**

Run:

```powershell
pnpm test -- src/composables/use-homepage-daily-recommendations.test.ts src/lib/current-utc-day-key.test.ts src/lib/homepage-portal.test.ts src/views/HomeView.test.ts
```

Expected: PASS.

- [ ] **Step 3: Run targeted backend verification**

Run:

```powershell
cd backend; go test ./internal/storage ./internal/app ./internal/server
```

Expected: PASS.

- [ ] **Step 4: Run repo-level guardrails**

Run:

```powershell
pnpm typecheck
pnpm lint
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add docs/plan/2026-04-15-homepage-daily-recommendations-design.md .cursor/rules/project-facts.mdc README.md API.md CLAUDE.md
git commit -m "docs: document homepage daily recommendations"
```

## Self-Review

- Spec coverage check:
  - Backend persistence: Task 1
  - Backend generator and no-repeat policy: Task 2
  - API exposure: Task 3
  - Frontend snapshot seam: Task 4
  - Homepage UI wiring: Task 5
  - UTC auto-refresh and fallback: Task 6
  - Docs and verification: Task 7
- Placeholder scan:
  - No `TBD`, `TODO`, or “similar to previous task” shortcuts remain.
- Type consistency:
  - Backend DTO name is `HomepageDailyRecommendationsDTO`.
  - Frontend DTO name is also `HomepageDailyRecommendationsDTO`.
  - API path is consistently `GET /api/homepage/recommendations`.
  - Frontend override input is consistently `dailySelection`.
