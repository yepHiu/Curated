# Backend Critical & High-Priority Fix Plan — 2026-04-29

Consolidated fix plan covering all **Critical** (C1–C3) and **High** (H1–H5) findings
from the backend audit. Medium/Low findings are documented separately in the audit report.

---

## Fix 1: PersistScanMovie — Don't overwrite trashed movies without restoring

**Severity:** Critical | **File:** `internal/storage/scan_repository.go` | **Effort:** Small

### Problem

When a scanned file matches an existing movie by `code`, the `SELECT` query (line 94) does
not check `trashed_at`. If the matched movie is in trash, the `UPDATE` at line 153
overwrites its `location` but leaves `trashed_at` set. The movie stays trashed but now
points to a new file — inconsistent state.

### Fix

**Option A (recommended):** Filter out trashed movies from the match query. A trashed
movie with the same `code` should not prevent a fresh import — the new file becomes a new
movie. Change line 92-97:

```go
queryErr := tx.QueryRowContext(
    ctx,
    `SELECT id, code, location FROM movies
     WHERE (code = ? OR location = ?)
       AND (trashed_at IS NULL OR TRIM(trashed_at) = '')
     LIMIT 1`,
    result.Number,
    result.Path,
).Scan(&movieID, &code, &location)
```

**Option B (alternative):** If the product intent is that re-scanning a trashed movie's
file should restore it, then clear `trashed_at` in the UPDATE:

```go
_, err = tx.ExecContext(
    ctx,
    `UPDATE movies SET location = ?, trashed_at = '', updated_at = ? WHERE id = ?`,
    result.Path,
    nowUTC(),
    movieID,
)
```

**Recommendation:** Option A. Scan imports a new file as a new movie. Restore from trash
should be an explicit user action (via the existing restore endpoint), not a side effect of
scanning.

### Verification

- Unit test: scan a file matching a trashed movie's code → verify a new movie is created
- Unit test: scan a file matching an active movie's code → existing update behavior preserved

---

## Fix 2: DeleteLibraryPathAndPruneOrphanMovies — Exclude trashed movies

**Severity:** Critical | **File:** `internal/storage/library_paths_repository.go` | **Effort:** Tiny

### Problem

Line 259 queries all movies regardless of trash status for orphan detection. When a library
path is removed, trashed movies that happen to reside under that path are permanently
deleted via `deleteMovieDatabaseTx`, bypassing the trash safety net entirely.

### Fix

Add the active-movie filter to the orphan query. Change line 258-259 from:

```go
rows, err := tx.QueryContext(ctx,
    `SELECT id, location FROM movies WHERE TRIM(COALESCE(location, '')) != ''`)
```

To:

```go
rows, err := tx.QueryContext(ctx,
    `SELECT id, location FROM movies
     WHERE TRIM(COALESCE(location, '')) != ''
       AND (trashed_at IS NULL OR TRIM(trashed_at) = '')`)
```

This ensures only active (non-trashed) movies are orphan-pruned. Trashed movies under the
removed path remain in trash — the user can still restore or permanently delete them later.

### Verification

- Unit test: remove library path that covers both active and trashed movies
- Verify only active movies are pruned; trashed movies remain untouched

---

## Fix 3: Add io.LimitReader to three handlers missing request body size limits

**Severity:** Critical | **Files:** `internal/server/server.go` | **Effort:** Tiny

### Problem

Three handlers use bare `io.ReadAll(r.Body)` without size limits:

| Line | Handler | Recommended Limit |
|---|---|---|
| 450 | `handlePatchActorUserTags` | 256KB |
| 500 | `handlePatchActorExternalLinks` | 256KB |
| 1226 | `handlePatchMovie` | 1MB |

### Fix

Change each instance from `io.ReadAll(r.Body)` to `io.ReadAll(io.LimitReader(r.Body, MAX))`.

**Line 450** (`handlePatchActorUserTags`):
```go
// Before
body, err := io.ReadAll(r.Body)

// After
body, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
```

**Line 500** (`handlePatchActorExternalLinks`):
```go
// Before
body, err := io.ReadAll(r.Body)

// After
body, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
```

**Line 1226** (`handlePatchMovie`):
```go
// Before
body, err := io.ReadAll(r.Body)

// After
body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
```

Consistent with existing patterns in the codebase: `handlePutMovieComment` uses 2MB,
`handlePatchCuratedFrameTags` uses 256KB.

### Verification

- Send oversized body to each endpoint → confirm 4xx or clean error (not OOM)

---

## Fix 4: Library Service — Add map index for O(1) lookups

**Severity:** High | **File:** `internal/library/service.go` | **Effort:** Medium

### Problem

`GetMovie`, `UpsertScannedMovie`, and `ApplyScrapedMetadata` all do O(n) linear scans
over `s.movies []MovieDetailDTO`. For a library with 10,000 movies, every detail-page
request and scan update pays this cost.

### Fix

Add a `map[string]int` index (movieID → slice index) alongside the existing slice.
Update the index in `UpsertScannedMovie` (append and update paths) and rebuild it
when the full list is reloaded.

**Step 1:** Add the index to the Service struct:

```go
type Service struct {
    mu     sync.RWMutex
    movies []contracts.MovieDetailDTO
    index  map[string]int  // movieID → position in movies slice
}
```

**Step 2:** Update `GetMovie` to use the index:

```go
func (s *Service) GetMovie(movieID string) (contracts.MovieDetailDTO, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    i, ok := s.index[movieID]
    if !ok {
        return contracts.MovieDetailDTO{}, errMovieNotFound
    }
    m := contracts.EffectiveMovieDetailDTO(s.movies[i])
    syncEffectiveRating(&m)
    return m, nil
}
```

**Step 3:** Update `UpsertScannedMovie` to maintain the index:

```go
func (s *Service) UpsertScannedMovie(result contracts.ScanFileResultDTO) {
    // ... existing validation ...

    s.mu.Lock()
    defer s.mu.Unlock()

    // Check index first
    if i, ok := s.index[result.MovieID]; ok {
        movie := &s.movies[i]
        movie.Location = result.Path
        // ... existing update logic ...
        syncEffectiveRating(movie)
        return
    }

    // Fall back to code/location match for edge cases
    for i := range s.movies {
        if s.movies[i].Code == result.Number || s.movies[i].Location == result.Path {
            // ... existing update logic ...
            s.index[s.movies[i].ID] = i
            return
        }
    }

    // Append new
    s.movies = append(s.movies, newMovie)
    s.index[newMovie.ID] = len(s.movies) - 1
}
```

**Step 4:** Update `ApplyScrapedMetadata` similarly to use the index.

**Step 5:** Add `RebuildIndex()` method called after loading the full list from storage:

```go
func (s *Service) RebuildIndex() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.index = make(map[string]int, len(s.movies))
    for i := range s.movies {
        s.index[s.movies[i].ID] = i
    }
}
```

### Consideration

The index adds memory (~16 bytes per movie for the map entry). For a 10,000-movie
library, that's ~160KB. Negligible cost for O(1) lookups.

### Verification

- Existing tests in `library/service_test.go` (if any) must still pass
- Manual: load library with 5000+ movies, verify detail page load time unchanged or improved

---

## Fix 5: Task Manager — Add TTL-based cleanup

**Severity:** High | **File:** `internal/tasks/manager.go` | **Effort:** Small

### Problem

The `tasks` map grows unbounded. Tasks are created and updated but never deleted.
`ListRecentFinished` returns terminal tasks but does not GC them.

### Fix

**Step 1:** Add a configurable max-age constant and a `PurgeOldTasks` method:

```go
const defaultTaskMaxAge = 24 * time.Hour

// PurgeOldTasks removes terminal tasks older than maxAge.
// Returns the number of tasks removed.
func (m *Manager) PurgeOldTasks(maxAge time.Duration) int {
    if maxAge <= 0 {
        maxAge = defaultTaskMaxAge
    }
    cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339)

    m.mu.Lock()
    defer m.mu.Unlock()

    removed := 0
    for id, t := range m.tasks {
        switch t.Status {
        case contracts.TaskCompleted, contracts.TaskFailed,
             contracts.TaskPartialFailed, contracts.TaskCancelled:
            if t.FinishedAt != "" && t.FinishedAt < cutoff {
                delete(m.tasks, id)
                removed++
            }
        }
    }
    return removed
}
```

**Step 2:** Also cap the total number of in-memory tasks. If the map exceeds a threshold
(e.g. 5000), purge aggressively:

```go
const maxInMemoryTasks = 5000

func (m *Manager) enforceCap() int {
    m.mu.Lock()
    defer m.mu.Unlock()

    if len(m.tasks) <= maxInMemoryTasks {
        return 0
    }

    // Collect terminal tasks sorted by FinishedAt, remove oldest half
    type entry struct {
        id         string
        finishedAt string
    }
    var terminal []entry
    for id, t := range m.tasks {
        switch t.Status {
        case contracts.TaskCompleted, contracts.TaskFailed,
             contracts.TaskPartialFailed, contracts.TaskCancelled:
            if t.FinishedAt != "" {
                terminal = append(terminal, entry{id, t.FinishedAt})
            }
        }
    }
    slices.SortFunc(terminal, func(a, b entry) int {
        if a.finishedAt < b.finishedAt { return -1 }
        if a.finishedAt > b.finishedAt { return 1 }
        return 0
    })

    toRemove := len(m.tasks) - maxInMemoryTasks/2
    if toRemove > len(terminal) {
        toRemove = len(terminal)
    }
    for i := 0; i < toRemove; i++ {
        delete(m.tasks, terminal[i].id)
    }
    return toRemove
}
```

**Step 3:** Call `enforceCap()` from `Create()` and call `PurgeOldTasks` periodically.

In `internal/app/app.go`, add a lightweight background goroutine:

```go
func (a *App) startTaskJanitor(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            n := a.tasks.PurgeOldTasks(24 * time.Hour)
            if n > 0 && a.logger != nil {
                a.logger.Debug("purged old tasks", zap.Int("count", n))
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### Verification

- Unit test: insert 100 terminal tasks with old FinishedAt timestamps, call PurgeOldTasks
  with 1-hour maxAge, verify all removed
- Unit test: insert 6000 tasks, verify enforceCap keeps map under 5000

---

## Fix 6: Homepage Recommendations — Reduce O(n²) to O(n·k)

**Severity:** High | **File:** `internal/storage/homepage_daily_recommendations.go` | **Effort:** Medium

### Problem

The inner candidate selection loop rebuilds a full weighted-candidate vector for each of
the k=14 selected items. Each rebuild iterates all n candidates (up to 10,000) and each
candidate triggers diversity penalty calculations with nested actor-set loops.

### Fix

**Step 1:** Pre-compute the actor-to-candidate mapping once before the selection loop,
so `homepageDiversityPenalty` can check against a pre-built set instead of scanning.

**Step 2:** Instead of rebuilding the weighted vector from scratch on each iteration,
compute weights once and adjust incrementally. After selecting a candidate, only reduce
the weight of remaining candidates that share actors/studios with the selected one,
rather than recomputing all weights:

```go
// Pseudocode for the replacement approach:
weights := make([]float64, len(candidates))
for i, c := range candidates {
    weights[i] = computeBaseWeight(c)
}

selected := make(map[int]bool)
for len(out) < limit {
    // Find best remaining candidate
    bestIdx := -1
    bestWeight := 0.0
    for i, w := range weights {
        if selected[i] { continue }
        if w > bestWeight {
            bestWeight = w
            bestIdx = i
        }
    }
    if bestIdx < 0 { break }

    out = append(out, candidates[bestIdx])
    selected[bestIdx] = true

    // Apply diversity penalty incrementally to remaining candidates
    for i := range weights {
        if selected[i] { continue }
        weights[i] *= diversityDiscount(candidates[bestIdx], candidates[i])
    }
}
```

This replaces O(n·k) full rebuilds with O(n·k) simple multiplications — each iteration
still scans all remaining candidates but avoids the allocation and actor-set construction
of a full rebuild.

### Verification

- Unit test: generate recommendations for a mock library of 2000 candidates
- Verify output is identical to the old algorithm (deterministic)
- Benchmark comparison: old vs new for n=2000, k=14

---

## Fix 7: Add Missing Database Indexes

**Severity:** High | **Files:** `internal/storage/migrations/` | **Effort:** Small

### Problem

Four important indexes are missing, causing full table scans on frequent query patterns.

### Fix

Create migration `0019_performance_indexes.sql`:

```sql
-- 0019_performance_indexes: add covering indexes for frequent query patterns

-- Speed up active/trash filtering on every movie list query.
-- Combined with the default ORDER BY added_at DESC, id ASC.
CREATE INDEX IF NOT EXISTS idx_movies_active_sort
    ON movies(added_at DESC, id ASC)
    WHERE trashed_at IS NULL OR TRIM(trashed_at) = '';

-- Speed up favorites mode filtering.
CREATE INDEX IF NOT EXISTS idx_movies_favorites
    ON movies(added_at DESC, id ASC)
    WHERE is_favorite = 1 AND (trashed_at IS NULL OR TRIM(trashed_at) = '');

-- Speed up actor-based filtering via the EXISTS subquery.
CREATE INDEX IF NOT EXISTS idx_movie_actors_actor_id
    ON movie_actors(actor_id);

-- Speed up asset lookups by movie + type (the dominant query pattern).
CREATE INDEX IF NOT EXISTS idx_media_assets_movie_type
    ON media_assets(movie_id, type);
```

**Note on partial indexes (WHERE clause):** SQLite supports partial indexes since 3.8.0.
The `WHERE` clause means the index only includes rows matching the condition, making it
smaller and faster than a full-column index.

**Alternative (simpler but slightly less optimal):** If avoiding partial indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_movies_trashed_added
    ON movies(trashed_at, added_at DESC, id ASC);
CREATE INDEX IF NOT EXISTS idx_movies_is_favorite
    ON movies(is_favorite);
CREATE INDEX IF NOT EXISTS idx_movie_actors_actor_id
    ON movie_actors(actor_id);
CREATE INDEX IF NOT EXISTS idx_media_assets_movie_type
    ON media_assets(movie_id, type);
```

### Consideration

Each index adds write overhead during INSERT/UPDATE. For a desktop application with
infrequent bulk inserts (scan runs periodically), this tradeoff is strongly in favor
of adding the indexes.

### Verification

- Run `EXPLAIN QUERY PLAN` before and after on the movie list query to confirm index usage
- Existing tests must pass (indexes are transparent to query results)

---

## Fix 8: Library Service — Cache joined strings for search

**Severity:** High | **File:** `internal/library/service.go` | **Effort:** Small

### Problem

`matchesQuery` at line 249-252 does three `strings.Join` allocations per movie per
list request:

```go
strings.Join(movie.Actors, " "),
strings.Join(movie.Tags, " "),
strings.Join(movie.UserTags, " "),
```

### Fix

Pre-compute a search text field on the DTO struct. Add a `SearchText` field to
`MovieDetailDTO` (or `MovieListItemDTO`), populated once when the movie is loaded
or scraped:

```go
func buildSearchText(m MovieListItemDTO) string {
    return strings.Join(m.Actors, " ") + " " +
           strings.Join(m.Tags, " ") + " " +
           strings.Join(m.UserTags, " ")
}
```

Call this in `UpsertScannedMovie`, `ApplyScrapedMetadata`, and the initial load path.
Then `matchesQuery` becomes:

```go
func matchesQuery(movie contracts.MovieDetailDTO, qLower string) bool {
    return strings.Contains(strings.ToLower(movie.SearchText), qLower)
}
```

If adding a field to the DTO is undesirable (it's in `contracts` and shared), an
alternative is to add a `searchTexts map[string]string` alongside the new `index` map
in the `Service` struct:

```go
type Service struct {
    mu          sync.RWMutex
    movies      []contracts.MovieDetailDTO
    index       map[string]int    // movieID → slice position
    searchTexts map[string]string // movieID → precomputed search string
}
```

### Verification

- Search behavior unchanged (same set of joined fields)
- Benchmark: compare allocations before/after in `ListMovies` with a 1000-movie set

---

## Implementation Order

Recommended order, accounting for dependencies and risk:

| Order | Fix | Risk | Effort |
|---|---|---|---|
| 1 | Fix 3 — Body size limits | Zero | 10 min |
| 2 | Fix 1 — Scan/trash filter | Low | 30 min |
| 3 | Fix 2 — Prune/trash filter | Low | 15 min |
| 4 | Fix 7 — Database indexes | Zero (additive) | 20 min |
| 5 | Fix 5 — Task cleanup | Low | 1 hr |
| 6 | Fix 4 — Library map index | Medium | 2 hr |
| 7 | Fix 8 — Search text cache | Low | 30 min |
| 8 | Fix 6 — Homepage alg optimization | Medium | 2 hr |

Fixes 1-4 can be done in a single session. Fixes 5-8 require more testing but are
independent of each other.

---

## Rollback Strategy

- **Fixes 1-3:** Simple code changes; revert commit to roll back.
- **Fix 7 (indexes):** Indexes are additive and transparent. They can be dropped
  with a follow-up migration if needed, but there is no reason to.
- **Fix 4 (map index):** The map is derived from the slice. If a bug is found,
  removing the index usage falls back to the current linear scan.
- **Fix 5 (task cleanup):** Tasks are in-memory only. The janitor can be disabled
  by not starting the goroutine.
- **Fix 6 (homepage):** The algorithm change must produce identical output to the
  current implementation. Verify before deploying.
