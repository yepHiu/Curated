# Backend Code Audit Report — 2026-04-29

Full audit of the Go backend (`curated-backend`), covering correctness, performance,
robustness, and maintainability.

---

## Scope

| Package | Files Audited |
|---|---|
| `server` | server.go (2436L), playback_curated_handlers.go, curated_export_handler.go, movie_poster_local.go, actor_avatar_local.go, app_update_handlers.go, homepage_daily_recommendations_handlers.go, library_played_handlers.go, access_log.go |
| `storage` | sqlite.go, library_repository.go, metadata_repository.go, scan_repository.go, library_paths_repository.go, movie_user_prefs_repository.go, movie_trash_repository.go, movie_comment_repository.go, movie_delete_repository.go, movie_asset_file.go, movie_stream.go, actor_repository.go, actor_external_links_repository.go, user_tags_repository.go, asset_repository.go, playback_curated.go, played_movies.go, homepage_daily_recommendations.go, app_update_status.go, 19 migration files |
| `library` | service.go |
| `scanner` | service.go, number.go |
| `scraper/metatube` | service.go |
| `tasks` | manager.go |
| `config` | config.go, library_settings.go |
| `app` | app.go (2453L) |

---

## Findings by Severity

### Critical — Correctness / Data Safety

#### C1. PersistScanMovie silently overwrites trashed movies

**File:** `internal/storage/scan_repository.go:94`

```
SELECT id, code, location FROM movies WHERE code = ? OR location = ? LIMIT 1
```

When scanning, if a movie with the same `code` exists but is in trash (`trashed_at` is set),
the code at line 153-158 updates its `location` to the new file path **without clearing
`trashed_at`**. The movie stays in trash state but now points to a new file — an inconsistent
state where the movie is simultaneously trashed and re-imported.

**Impact:** User scans a new file matching a previously-trashed code. The movie silently
remains in trash, invisible in the library, but its file pointer has changed.

#### C2. DeleteLibraryPathAndPruneOrphanMovies bypasses trash for permanent deletion

**File:** `internal/storage/library_paths_repository.go:259`

```sql
SELECT id, location FROM movies WHERE TRIM(COALESCE(location, '')) != ''
```

This query does not filter by `trashed_at` — it selects **both active and trashed** movies.
When a library root is removed, all orphaned movies under that root are permanently deleted
via `deleteMovieDatabaseTx`, completely bypassing the trash safety net.

**Impact:** Removing a library path permanently deletes trashed movies that happen to reside
under that path. No recovery possible.

#### C3. Unbounded request body reads (DoS vector)

Three handlers use `io.ReadAll(r.Body)` without `io.LimitReader`:

| File | Line | Handler |
|---|---|---|
| `internal/server/server.go` | 450 | `handlePatchActorUserTags` |
| `internal/server/server.go` | 500 | `handlePatchActorExternalLinks` |
| `internal/server/server.go` | 1226 | `handlePatchMovie` |

All other handlers that read bodies use `io.LimitReader(r.Body, <MAX>)` (e.g. 2MB for
comments, 256KB for curated frame tags). These three are inconsistent omissions.

**Impact:** A malicious client can send a multi-GB body, exhausting server memory.

---

### High — Performance / Resource Leaks

#### H1. Library service: O(n) linear search on every mutation and lookup

**File:** `internal/library/service.go:108-110`

```go
func (s *Service) GetMovie(movieID string) (contracts.MovieDetailDTO, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for _, movie := range s.movies {  // O(n) scan
        if movie.ID == movieID {
```

`GetMovie`, `UpsertScannedMovie` (line 130), and `ApplyScrapedMetadata` (line 178) all do
full-slice scans. The `movies` slice grows unbounded (one entry per library movie). A
`map[string]*MovieDetailDTO` index would reduce all three operations to O(1).

**Impact:** For libraries with thousands of movies, every detail-page request and every
scan-step update pays a linear scan cost.

#### H2. Task Manager: unbounded map growth, no cleanup

**File:** `internal/tasks/manager.go:14-15`

```go
type Manager struct {
    mu    sync.RWMutex
    tasks map[string]contracts.TaskDTO
}
```

Tasks are created via `Create()` but never deleted. `ListRecentFinished` returns terminal
tasks but does not remove them. No background GC, no TTL, no size cap.

**Impact:** Running for days/weeks accumulates hundreds or thousands of finished tasks in
memory. Each task DTO carries full metadata maps.

#### H3. Homepage recommendations: O(n^2) candidate selection

**File:** `internal/storage/homepage_daily_recommendations.go:442-494`

The inner loop rebuilds a weighted candidate vector from scratch for each selected item
(limit=14). Each rebuild iterates all candidates (up to 10,000 from line 115). Each
candidate also triggers `homepageDiversityPenalty` with an inner actor-set loop.

**Impact:** For large libraries, recommendation generation is noticeably slow. Combined
with the fact this runs at least once per UTC day, it's a user-visible delay on homepage
load.

#### H4. Missing database indexes

| Missing Index | Query Pattern Affected |
|---|---|
| `movies(trashed_at)` or better: use `NULL` for active | Every movie list query filters on `trashed_at IS NULL OR TRIM(trashed_at) = ''` (full scan). Store `trashed_at` as NULL for active movies and use a simple `IS NULL` / `IS NOT NULL` check. |
| `movies(added_at DESC, id ASC)` | Default sort order has no index coverage. A compound index `(trashed_at, added_at DESC, id)` would cover both filter and sort. |
| `media_assets(movie_id, type)` | The most frequent asset query is `WHERE movie_id = ? AND type = ?` but only `movie_id` is indexed (from 0001). |
| `movie_actors(actor_id)` | EXISTS subquery in `buildMovieFilters` joins via `ma.actor_id`. Only `movie_actors(movie_id)` is indexed. |

**Impact:** Every movie list request does a full table scan. For thousands of movies and
a UI that refreshes the list frequently, this is the single biggest database performance
bottleneck.

#### H5. Per-request string allocations in hot list loop

**File:** `internal/library/service.go:249-252`

```go
strings.Join(movie.Actors, " "),
strings.Join(movie.Tags, " "),
strings.Join(movie.UserTags, " "),
```

Three `strings.Join` calls allocate new strings for every movie in every list request.
These could be pre-computed and cached on the DTO struct.

---

### Medium — Architecture / Robustness

#### M1. App struct is a 2453-line God Object

**File:** `internal/app/app.go`

The `App` struct holds 30+ fields and 20+ distinct concerns: scan orchestration, scrape
orchestration, asset downloads, actor profiles, settings persistence, playback resolution,
homepage recommendations, task management, library watching, update checking, and HTTP
routing. Multiple independent subsystems are co-located in a single file.

#### M2. Scrape goroutines have no shutdown barrier

**File:** `internal/app/app.go:1346-1351`

```go
func (a *App) enqueueScrape(...) {
    go func(r contracts.ScanFileResultDTO, parent string) {
        a.scrapeSem <- struct{}{}  // blocks here if semaphore is full
```

Six `go func()` calls across app.go have no `sync.WaitGroup` tracking. During shutdown,
goroutines can be stuck waiting on the semaphore. No barrier waits for them to drain.

#### M3. Atomic file rename fails on cross-volume paths (Windows)

**File:** `internal/config/library_settings.go:372-391`

The retry loop uses `os.Rename` with a fallback `os.Remove` + `os.Rename`. On Windows,
`os.Rename` across volumes returns `syscall.EXDEV` — the retry loop can never succeed
because it retries the same doomed approach.

#### M4. Duplicate migration numbers

**Files:** `migrations/0009_actor_user_tags.sql` and `migrations/0009_movie_display_overrides.sql`

Two migration files share the prefix `0009`. Order depends on alphabetical filename sort,
which currently happens to work (`0009_actor_user_tags` < `0009_movie_display_overrides`),
but is fragile.

#### M5. Hardcoded health-check keyword

**File:** `internal/scraper/metatube/service.go:683`

```go
testKeyword := "SSIS"
```

All providers are health-checked with the same `SSIS` keyword. Providers that don't carry
this series will fail the check and enter cooldown — despite being perfectly healthy.

#### M6. filepath.Walk not context-aware

**File:** `internal/scanner/service.go:128`

The `filepath.Walk` callback has no mechanism to abort mid-walk for context cancellation.
If a library root contains millions of subdirectories but no video files, Walk runs to
completion before `ctx.Done()` is checked.

#### M7. Filesystem paths leaked to HTTP clients

**File:** `internal/server/server.go:908, 968`

`handleRevealMovieInFileManager` and `handleRevealLibraryPathInFileManager` include the
absolute server-side filesystem path in error response messages.

#### M8. Fragile SQLite UNIQUE constraint detection

**File:** `internal/server/playback_curated_handlers.go:365`

```go
if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
```

String-matching on error text to classify a constraint violation. The storage package
should return a sentinel error (e.g. `ErrCuratedFrameIDExists`) that the handler can
check with `errors.Is`.

---

### Low — Code Quality

#### L1. handlePatchSettings: 331-line single function

**File:** `internal/server/server.go:1637-1968`

10 near-identical `if body.X != nil` blocks for settings fields. Should be refactored
into per-field-group functions returning `settingsPatchOperation`.

#### L2. config/library_settings.go: 200+ lines of copy-paste key parsing

**File:** `internal/config/library_settings.go:37-241`

Every setting key follows the same pattern (check existence, parse type, validate, assign)
but is written as a manually repeated block. A table-driven approach would eliminate ~180
lines of repetitive code and remove the risk of copy-paste bugs.

#### L3. parseYear integer overflow

**File:** `internal/library/service.go:224-233`

Long numeric strings (e.g. corrupted data) produce unchecked integer overflow.
No upper bound validation on the resulting year value.

#### L4. sanitizeTaskType produces consecutive hyphens

**File:** `internal/tasks/manager.go:152-162`

Input like `scan..library` produces `scan--library`. Purely cosmetic.

#### L5. nowUTC() duplicated across two packages

**Files:** `internal/tasks/manager.go:148`, `internal/app/app.go:2451`

#### L6. 53 nil-logger guard checks across server handlers

Every log call wraps the log site in `if h.logger != nil`. If `h.logger` is always
non-nil (it is set at construction), use `zap.NewNop()` for the zero-value case instead.

#### L7. Curated frame full-text search does full table scan

**File:** `internal/storage/playback_curated.go:173-176`

```sql
LOWER(title || ' ' || code || ' ' || movie_id || ' ' || actors_json || ' ' || tags_json
  || ' ' || captured_at || ' ' || CAST(position_sec AS TEXT)) LIKE ?
```

Concatenation across multiple columns with `LIKE` cannot use any index. Consider SQLite FTS5.

#### L8. Dev test paths in production config defaults

**File:** `internal/config/config.go:260-276`

Hardcoded `../videos_test` paths are baked into production defaults. Harmless in practice
(they won't exist on user machines) but conceptually wrong.

#### L9. Config DisallowUnknownFields blocks forward compat

**File:** `internal/config/config.go:187`

A config file written by a newer version with an added field will fail to load on an older
version, blocking rollbacks unnecessarily.

#### L10. Missing COLLATE NOCASE on actors.name and tags.name

Actor and tag names have case-sensitive UNIQUE constraints but `ListActors` uses
`COLLATE NOCASE` for ordering — inconsistent.

---

## Non-Findings

The following were examined and found to have **no issues**:

- **SQL injection:** All data access goes through parameterized queries via the storage
  abstraction. Zero raw SQL string building.
- **Race conditions in server layer:** Handler struct fields are set once at construction
  and never mutated.
- **Context cancellation in HTTP server:** Graceful shutdown via `ctx.Done()` is properly
  handled in `ListenAndServe`.
- **Transaction discipline:** Multi-step mutations consistently use `BeginTx` + defer
  rollback + explicit commit.
- **rows.Err() after iteration:** Checked correctly in all 20+ locations.

---

## Summary Table

| # | Severity | Category | Location | Description |
|---|---|---|---|---|
| C1 | Critical | Data correctness | scan_repository.go:94 | PersistScanMovie overwrites trashed movie location without restoring |
| C2 | Critical | Data correctness | library_paths_repository.go:259 | DeleteLibraryPath permanently deletes trashed movies |
| C3 | Critical | DoS | server.go:450,500,1226 | Three handlers lack request body size limits |
| H1 | High | Performance | library/service.go:108 | O(n) linear search for every movie lookup/mutation |
| H2 | High | Memory leak | tasks/manager.go:15 | Tasks map grows unbounded, never cleaned |
| H3 | High | Performance | homepage_daily_recommendations.go:442 | O(n²) candidate selection loop |
| H4 | High | Performance | Multiple locations | 4 missing database indexes |
| H5 | High | Performance | library/service.go:249 | Per-request strings.Join allocations |
| M1 | Medium | Architecture | app/app.go | 2453-line God Object |
| M2 | Medium | Goroutine leak | app/app.go:1346 | Scrape goroutines lack shutdown barrier |
| M3 | Medium | Robustness | config/library_settings.go:372 | Cross-volume rename fails on Windows |
| M4 | Medium | Robustness | migrations/ | Duplicate migration numbers (0009) |
| M5 | Medium | Robustness | metatube/service.go:683 | Hardcoded health-check keyword |
| M6 | Medium | Robustness | scanner/service.go:128 | Walk not context-aware |
| M7 | Medium | Privacy | server.go:908,968 | Filesystem paths leaked to clients |
| M8 | Medium | Reliability | playback_curated_handlers.go:365 | Fragile string-match error classification |
| L1-L10 | Low | Code quality | Various | See Low section above |
