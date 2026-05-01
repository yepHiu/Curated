# Library Scroll Production Performance Review

Date: 2026-04-12

## Symptom

In production builds with a large movie library, scrolling down the library can outrun card rendering and poster loading:

- the scrollbar keeps moving
- the list still reserves space below
- one or more rows/chunks show as blank or skeleton-like gaps for a noticeable time
- cards appear later, which makes the browse experience feel laggy

This is especially visible when the content drive is an HDD and poster assets are served from local files.

## Current implementation

Relevant files:

- `src/components/jav-library/VirtualMovieMasonry.vue`
- `src/components/jav-library/MovieCard.vue`
- `src/components/jav-library/MediaStill.vue`
- `backend/internal/server/movie_poster_local.go`

Key observations:

1. The library uses `vue-virtual-scroller` `DynamicScroller`, but it virtualizes by chunk rather than per card.
2. Each chunk renders 4 rows worth of cards at a time.
3. The scroller buffer is currently modest:
   - `BUFFER_CHUNKS = 5`
   - `BUFFER_PX = 600`
4. Library card posters are currently requested with `loading="eager"` in `MovieCard.vue`.
5. Poster URLs are same-origin backend asset routes like `/api/library/movies/{id}/asset/thumb`, and the server serves bytes from local files using `http.ServeContent`.
6. Response headers are `Cache-Control: private, no-cache`, which means the browser tends to revalidate instead of treating posters as strongly cacheable immutable assets.

## Root-cause analysis

This is likely a combined scheduling problem rather than a single bug.

### 1. Virtual window is not far enough ahead

The scroller can move into space whose next chunks are not yet rendered or measured far enough in advance.
That creates the “scrollbar already moved, content area still empty” feeling.

### 2. Poster requests start too late for the browsing pattern

Because cards only exist when their virtual chunk becomes active, poster requests for upcoming rows are delayed until near-visibility.
On HDD-backed local assets, many small reads have poor random-read latency, so “just-in-time” image fetches are too late.

### 3. `eager` on every visible card is not actually helping enough

`eager` is useful for above-the-fold hero images, but for a long virtualized library it can create bursty image fetches for an entire chunk at once.
That competes for decode and I/O budget precisely when the user is scrolling.

### 4. Asset caching policy is conservative

`private, no-cache` is safe for freshness, but it increases revalidation pressure on repeated browse sessions.
For stable local poster files, this policy is likely more conservative than needed.

## Recommended optimization order

### Option A: Tune the existing virtual scroller first

Changes:

- increase `BUFFER_PX`
- increase `pool-size`
- optionally increase `BUFFER_CHUNKS`
- slightly over-estimate chunk height rather than under-estimate it

Expected effect:

- fewer white gaps during fast scroll
- better “next screen already prepared” behavior

Tradeoff:

- somewhat more DOM nodes alive at once
- somewhat more memory usage

Recommendation:

Do this first. It is the smallest, lowest-risk change and directly targets the visible gap.

### Option B: Change library poster loading from unconditional `eager` to staged loading

Changes:

- keep near-viewport chunks/cards as eager or high-priority
- make farther virtualized cards `lazy`
- optionally add a tiny prefetch band for the next 1 to 2 chunks

Expected effect:

- smoother decode / request scheduling
- less burst contention when a chunk enters view

Tradeoff:

- needs a small “distance-to-viewport” policy
- slightly more code than plain buffer tuning

Recommendation:

Do this second. It addresses the real production workload better than forcing all visible cards to eager-load.

### Option C: Improve local poster cache headers / derivative strategy

Changes:

- switch poster asset responses from `private, no-cache` to a stronger cache strategy when image versioning is present
- ensure library card browsing always prefers small thumbnail derivatives over larger covers
- optionally add a smaller “list-thumb” derivative specifically for library browsing

Expected effect:

- less repeat revalidation
- fewer expensive local reads and decodes

Tradeoff:

- touches backend asset-serving semantics
- requires care so re-scrape freshness still works

Recommendation:

Do this after A/B unless profiling shows backend file serving is the main bottleneck.

### Option D: Structural fallback if A/B/C are still not enough

Changes:

- move from chunk-virtualization toward per-row or more granular virtualization
- or preload upcoming chunk data based on scroll direction / velocity

Expected effect:

- best long-term control over large libraries

Tradeoff:

- highest complexity
- larger regression surface

Recommendation:

Do not start here unless simpler changes fail.

## Suggested implementation sequence

1. Raise virtual scroller buffer aggressively and measure on a production-like library.
2. Add lightweight instrumentation in production/dev builds:
   - chunk activation timing
   - image load count and latency
   - scroll velocity vs chunk render delay
3. Change library card poster loading to a staged strategy:
   - current chunk: eager
   - next chunk(s): prefetch / eager
   - farther chunks: lazy
4. Revisit backend asset cache headers for local poster routes.
5. Only then decide whether the chunk-based virtualization model itself needs redesign.

## Concrete code targets

- `src/components/jav-library/VirtualMovieMasonry.vue`
  - buffer size
  - pool size
  - chunk-height safety margin
  - optional chunk prewarm logic
- `src/components/jav-library/MovieCard.vue`
  - poster loading strategy
  - optional fetch priority policy
- `src/components/jav-library/MediaStill.vue`
  - support per-card fetch policy cleanly
- `backend/internal/server/movie_poster_local.go`
  - cache policy for stable local poster assets

## 2026-05-01 Loading Regression Follow-Up

### Additional symptom

Home, the full library page, and navigation into movie detail can feel slower than older builds even when the UI itself is not doing an obviously expensive render.

The slow path is a combination of startup data loading and image revalidation:

- Web mode imports `webLibraryService`, which begins loading the active movie cache immediately.
- That cache loader pages through the full library in `500` item batches up to `10000` movies.
- `HomeView` uses `moviesLoaded` to leave the homepage skeleton, so it was waiting for the entire paged prefetch instead of the first usable page.
- Movie detail can be opened while the full list prefetch is still occupying the API/database path.
- Local cover, thumb, preview, and actor avatar routes used `Cache-Control: private, no-cache`, so repeated visits still revalidated many stable image URLs.
- Homepage daily recommendations are cached after generation, but a stale or missing snapshot can still trigger a backend `ListMovies(limit=10000)` candidate pass. That remains a secondary contributor when the daily snapshot regenerates.

### Implemented slice

The first optimization slice landed on 2026-05-01:

- `src/services/adapters/web/web-library-service.ts` now marks the active movie list loaded after the first page is received, then continues loading remaining pages in the background.
- The full cache is still completed after the remaining pages return, so library-wide search and sort behavior continues to converge to the full local cache.
- Trash loading uses the same first-page update behavior for its own cache.
- `backend/internal/server/movie_poster_local.go` now serves movie cover, thumb, and locally cached preview asset responses with reusable private cache headers.
- `backend/internal/server/actor_avatar_local.go` uses the same reusable cache policy for actor avatars.
- Local movie image and actor avatar URLs now include a file-version query parameter based on local asset mtime and size, so browsers can reuse stable bytes without pinning stale images after a rescrape or avatar refresh.
- Existing frontend image versioning remains an additional in-session freshness mechanism after metadata refresh tasks call `bumpMovieImageVersion`.
- Remote preview proxy fallback responses remain conservative (`private, no-cache`) because those bytes are not guaranteed to be stable local cache files.

### Regression coverage

New tests cover the intended behavior:

- `src/services/adapters/web/web-library-service.test.ts` verifies `moviesLoaded` becomes true after the first page while the next page is still pending, then the cache completes when the background page resolves.
- `backend/internal/server/server_test.go` verifies local movie cover and preview assets are not served with `no-cache`, include a reusable `max-age`, and local poster URLs include asset versions.
- `backend/internal/storage/movie_asset_file_test.go` verifies local poster, preview, and actor avatar cache hits expose URL versions.

### Remaining follow-up

If large-library users still report slow first load after this slice, the next step should be server-side pagination, filtering, and sorting for the library route instead of requiring the browser to own a complete `10000` item cache before every workflow is fully accurate.

## Original Scroll Recommendation to Execute Next

For the original scroll blank-gap problem, the highest-value first slice remains:

1. increase virtual scroller headroom
2. stop treating every library poster as unconditional eager-load
3. measure again on the real production package with HDD-backed assets

This should improve the visible “blank gap while scrolling” problem without forcing a large architecture rewrite.
