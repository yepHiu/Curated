# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Curated** (product name; repo folder `jav-shadcn`) is a desktop-oriented media library application for managing, browsing, scraping, and playing video collections. It consists of a Vue 3 frontend with a Go backend, using SQLite for persistence and metatube-sdk-go for metadata scraping.

**Current Architecture Phase:** Local-first Electron delivery around a shared Vue SPA + Go HTTP API boundary. Electron currently starts or reuses the Go backend, starts or reuses Vite in development, uses the Curated app icon, hides to tray on window close, marks backend requests as Curated Desktop with `X-Curated-Client: desktop-electron` plus desktop OS headers, versions its renderer entry URL across app upgrades, and exposes only a narrow `window.javLibrary.pickDirectory()` preload bridge for native directory selection. Production release packaging installs `Curated.exe` as the Electron desktop shell, bundles the Go backend under `resources/app/curated.exe`, serves entry HTML without caching while keeping hashed assets immutable, and cleans only managed `frontend-dist` directories before installer upgrades; deeper Electron IPC bridges and mpv player integration remain target-direction work.

**Public docs rule:** Root `README.md` is the short English entry, `README.zh-CN.md` and `README.ja-JP.md` are full translations, `docs/guide.md` is the detailed handbook and documentation index, and root `API.md` is the single public API reference. Do not rebuild the full API table or operational long-form content inside the README.

## Tech Stack

- **Frontend:** Vue 3 + TypeScript + Vite + Tailwind CSS v4 + shadcn-vue + vue-i18n
- **Backend:** Go 1.25+ with SQLite (modernc.org/sqlite), Zap logging
- **Metadata:** metatube-sdk-go for adult video metadata scraping
- **Testing:** Vitest (frontend), Go test (backend)

## Commands

### Frontend

```bash
# Install dependencies
pnpm install

# Start development server (proxies /api to 127.0.0.1:8080)
pnpm dev

# Build for production
pnpm build

# Run linter
pnpm lint

# Run tests
pnpm test

# Run single test file
pnpm test -- src/path/to/file.test.ts

# Type check only
pnpm typecheck
```

### Backend

```bash
# Build the development backend binary
cd backend
go build -o curated-dev.exe ./cmd/curated

# Run backend HTTP server (default mode)
./curated-dev.exe

# Run with specific config
./curated-dev.exe -config path/to/config.yaml

# Run in stdio mode (for future Electron bridge)
./curated-dev.exe -mode stdio

# Run the Electron shell MVP from the repo root
# Development Electron starts/reuses backend :8080 and Vite :5173.
pnpm dev:electron

# Build Electron main-process output for release assembly
pnpm release:electron-main

# Build the release backend binary
go build -tags release -o curated.exe ./cmd/curated

# Run tests
go test ./...

# Run tests for specific package
go test ./internal/storage/...

# Run with verbose output
go test -v ./internal/storage/...

# Create / verify / preflight a backup package
go run ./cmd/curated -maintenance backup-create -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-verify -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-preflight -backup-path C:\Backups\curated.curated-backup

# Restore only after Curated is fully stopped and preflight succeeds
go run ./cmd/curated -maintenance backup-restore -backup-path C:\Backups\curated.curated-backup -confirm-restore

# Plan first; apply only while Curated is fully stopped and with a verified pre-migration backup
go run ./cmd/curated -maintenance path-migrate-plan -path-from D:\Media -path-to E:\Media
go run ./cmd/curated -maintenance path-migrate-apply -path-from D:\Media -path-to E:\Media -backup-path D:\Backups\before-path-migration.curated-backup -confirm-path-migration
```

Backup format v1 includes the SQLite snapshot and optional `library-config.cfg`, but not media or user asset files. Normal runtime holds `<databasePath>.runtime.lock`; offline restore and path migration must acquire the same cross-process lock. Restore retains `.pre-restore-*` rollback files. Path migration uses a read-only plan, a strict path-column whitelist, segment-aware Windows/Unix mapping, target/conflict checks, an automatically created and verified backup, one transaction for updates plus `path_migration_audits`, and storage-binding reset. Cross-platform missing/unchecked targets require explicit `-allow-missing-paths`; target type errors and conflicts cannot be overridden. In Web API mode, Settings -> Maintenance calls the PIN-protected create, verify, and preflight endpoints with absolute paths on the backend machine; it intentionally exposes no online restore or path-migration action.

Windows binary naming rule:
- Development backend builds must use `curated-dev.exe`.
- Release/package backend builds use `curated.exe`.
- Do not generate a dev Windows backend binary named `curated.exe`, because it can conflict with the installed production backend on the same machine.

### Full Stack Development

```bash
# Terminal 1: Start backend (from repo root or backend/)
cd backend && go run ./cmd/curated
# Or build a Windows dev binary with: pnpm backend:build:dev
# Then run: ./backend/runtime/curated-dev.exe

# Terminal 2: Start frontend dev server
pnpm dev
```

**Environment Variables:**
- `VITE_USE_WEB_API=true` - Use real backend API (set in root `.env` by default)
- `VITE_API_BASE_URL` - Override API base URL; when unset, loopback Web API dev connects directly to dev backend `:8080` to avoid Vite proxying large uploads, while release hosting on `:8081` and other modes default to same-origin `/api`
- `VITE_LOG_LEVEL` - Optional default for browser `loglevel` (`trace`|`debug`|`info`|`warn`|`error`|`silent`); overridden by `localStorage['curated-client-log-level']` when set (Settings → Logging)

### Library Behavior Configuration

Library-specific settings are persisted to `config/library-config.cfg` (JSON) and merged on startup:

- **`organizeLibrary`** - Whether to organize library files into structured folders
- **`autoLibraryWatch`** - Whether to auto-scan when files change via fsnotify (default: `true`)
- **`autoActorProfileScrape`** - Whether missing actor profiles (no avatar and no summary) are auto-queued after successful movie scrapes and by a bounded background library sweep (default: `false`)
- **`autoDownloadUpdates`** - Whether startup background update checks may automatically download and SHA256-verify a newer installer after one is discovered (default: `false`); exposed in Settings -> General, and installation still remains an explicit user action
- **`launchAtLogin`** - Whether Windows login autostart is enabled for the current user (default: `false`); supported runtimes sync `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` to launch `curated(.exe) -mode tray -autostart`, which starts silently in tray mode without opening the browser on that login launch
- **`metadataMovieProvider`** - Primary metadata provider for movie scraping
- **`metadataMovieStrategy`** - Higher-level provider scheduling strategy (`auto-global` | `auto-cn-friendly` | `custom-chain` | `specified`)
- **`defaultImportLibraryPathId`** - Library path id used as the target for top-bar movie imports; persisted by Settings -> Video storage and consumed by `POST /api/import/movies` and resumable upload endpoints under `/api/import/movies/uploads`
- **`backupDirectory`** - Last directory used by a successful Settings -> Maintenance backup creation. Only the directory is stored; each backup still receives a new UTC-timestamped filename. Empty means no remembered destination
- **`logDir`** / **`logFilePrefix`** / **`logMaxAgeDays`** / **`logLevel`** - Backend Zap log file output (merged into the same fields as the main `-config` JSON); empty **`logDir`** means "use the default log directory" instead of disabling file logging: dev builds default to **`backend/runtime/logs`**, while release builds default to **`LOCALAPPDATA\\Curated\\logs`**. **`PATCH /api/settings`** field **`backendLog`** updates **`logDir`** / **`logMaxAgeDays`** / **`logLevel`** from the settings UI (omits **`logFilePrefix`** so manual `library-config.cfg` or the default `curated-dev` in dev / `curated` in release applies); **restart the backend** for new log directory/level to apply to file sinks
- **`proxy`** - Outbound HTTP proxy for the Curated backend (Metatube scraping, asset downloads); persisted here and applied as process `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` via `backend/internal/proxyenv` so `http.ProxyFromEnvironment` picks it up

Update via `PATCH /api/settings`; changes are written atomically to the config file.

## Architecture

### Frontend Structure (`src/`)

```
src/
  api/              # HTTP client and endpoint definitions
  components/
    jav-library/    # Domain-specific components
    ui/             # shadcn-vue UI components
  composables/      # Vue composables (e.g., use-scan-task-tracker.ts, use-app-toast.ts)
  domain/           # Domain types and logic
    library/
    movie/
  lib/              # Utilities and typed mock data (jav-library.ts)
  locales/          # i18n translation files (en.json, ja.json, zh-CN.json)
  router/           # Vue Router configuration
  services/         # Frontend service layer
    adapters/       # Mock adapter, future HTTP/Electron adapters
    contracts/      # Service interfaces
  views/            # Page-level components
```

**Key Frontend Patterns:**
- Use `@/` alias for imports from `src/`
- shadcn-vue components are in `src/components/ui/`
- Domain components are in `src/components/jav-library/`
- Mock data and types are in `src/lib/jav-library.ts`
- Service layer with adapter pattern for backend communication
- State management defaults to composables plus the service layer; Pinia may be introduced later through a small, bounded service/new feature, not a broad upfront migration
- Routes: `library`, `favorites`, `recent` (redirects to library), `tags` (redirects to library, preserving query), `actors`, `history`, `detail/:id`, `player/:id`, `settings`, `lock`
- PIN App Lock: Web API mode can enable a backend-enforced PIN gate. `src/services/auth-lock-service.ts` owns auth status including `pinLength`, `src/services/auth-idle-lock-service.ts` keeps regular sessions alive on user activity and redirects after idle expiry, `/lock` renders the keyboard-first lock screen with the configured number of PIN cells, and the route guard redirects locked pages to `/lock?redirect=...`.
- Playback progress: dual storage (backend SQLite in Web API mode, `localStorage` in Mock mode)
- Daily watch-time: Settings -> Overview statistics use player-reported watch-time deltas; Web API mode stores per-day/per-movie aggregates in SQLite, Mock mode stores them in `localStorage`
- History page: `src/views/HistoryView.vue` displays watch history grouped by date
- Virtual scrolling: uses `vue-virtual-scroller` for large poster grids
- i18n: uses `vue-i18n` with locale files in `src/locales/`
- Toast notifications: use `pushAppToast()` from `@/composables/use-app-toast` (backed by vue-sonner)

### Backend Structure (`backend/`)

```
backend/
  cmd/curated/      # Application entry point (Go module: curated-backend)
  internal/
    app/            # Application lifecycle and wiring
    config/         # Configuration management
    contracts/      # DTOs, error codes, shared interfaces
    library/        # Library domain service
    logging/        # Zap logger setup
    proxyenv/       # Sync library proxy config to HTTP_PROXY/HTTPS_PROXY for outbound HTTP
    scanner/        # File scanning service
    scraper/        # Metadata scraping adapter
    server/         # HTTP server and handlers
    storage/        # SQLite repository layer
      migrations/   # Database migrations
    tasks/          # Async task management
```

**Key Backend Patterns:**
- Clean architecture with repository pattern
- Contracts define DTOs shared between layers
- Storage layer handles all database access
- Services contain business logic
- Server layer handles HTTP transport

### API Routes

The backend exposes these HTTP endpoints:

```
GET    /api/health                          # Health check (name, version/build stamp, optional installerVersion, channel, databasePath)
GET    /api/auth/status                     # PIN App Lock status for the current browser/session, including pinLength
POST   /api/auth/setup-pin                  # Create or replace the app PIN; stores only an Argon2id salted hash
POST   /api/auth/unlock                     # Verify PIN and issue HTTP-only curated_auth cookie; trustedForever creates a non-expiring trusted-device session
POST   /api/auth/change-pin                 # Change the PIN after an unlocked session verifies the current PIN
POST   /api/auth/lock                       # Revoke the current auth session and clear curated_auth
PATCH  /api/auth/settings                   # Update non-secret PIN lock settings; requires an unlocked session
GET    /api/auth/sessions                   # List active trusted-forever sessions by safe public ID (never exposes bearer tokens)
DELETE /api/auth/sessions/{publicId}        # Revoke one trusted-forever session
POST   /api/auth/sessions/revoke-others     # Revoke all trusted-forever sessions except the current session
GET    /api/connected-clients               # Process-lifetime client visibility for Settings overview (IP/UA metadata, no MAC)
GET    /api/dev/performance                 # Dev-only CPU summary for the frontend monitor bar
GET    /api/app-update/status               # Cached packaged-app update status vs latest GitHub Release
POST   /api/app-update/check                # Force a fresh packaged-app update check
POST   /api/app-update/download             # Download and SHA256-verify the latest installer
POST   /api/app-update/install              # Launch a verified downloaded installer after explicit user action
DELETE /api/app-update/downloaded-installer # Clear cached downloaded installer metadata/file
POST   /api/maintenance/backups             # Create a consistent package without overwriting an existing destination
POST   /api/maintenance/backups/verify      # Verify manifest, hashes, files, SQLite integrity, and migrations
POST   /api/maintenance/backups/preflight   # Assess compatibility, targets, and capacity without writing live data
POST   /api/library/health/scan              # Read-only SQLite/storage/file/asset/metadata/staging diagnostics with stable findings
POST   /api/library/health/repairs           # Start an explicitly confirmed bounded metadata repair queue
GET    /api/library/health/repairs/{repairId} # Read persisted parent/child repair progress and per-item results
POST   /api/library/health/actions           # Start confirmed audited orphan-state or strict import-staging cleanup
GET    /api/homepage/recommendations        # Persisted UTC-day homepage hero + recommendation snapshot
POST   /api/homepage/recommendations/refresh # Force-regenerate the current UTC-day homepage recommendation snapshot; optional body preserveHeroMovieIds keeps the hero slate and excludeRecommendationMovieIds avoids the current rail
GET    /api/homepage/recommendations/feedback # List active explicit recommendation feedback
POST   /api/homepage/recommendations/feedback # Create idempotent movie exclusion/snooze or actor/studio/tag down-ranking feedback
DELETE /api/homepage/recommendations/feedback/{feedbackId} # Remove one feedback item and restore future candidate eligibility
GET    /api/library/movies                  # List movies (mode/q/tag/actor/studio/playState/userRating/resolution/addedAfter/limit/offset); tag and actor may be comma-separated AND; studio may be comma-separated OR
GET    /api/library/saved-views             # List ordered versioned Saved Views
POST   /api/library/saved-views             # Create one Saved View (maximum 50)
PUT    /api/library/saved-views/order       # Transactionally replace the complete Saved View order
PATCH  /api/library/saved-views/{id}        # Rename and/or replace canonical filters
DELETE /api/library/saved-views/{id}        # Delete only the view definition
GET    /api/library/movies/{id}             # Get movie detail
GET    /api/library/movies/{id}/playback    # Playback descriptor (direct-play metadata now; future remux/transcode seam); optional `clientVideoCodecs=h264,hevc,av1` query narrows mp4-family direct play to browser-reported codecs
POST   /api/library/movies/{id}/playback-session  # Create explicit playback session (for example HLS stream push)
GET    /api/playback/sessions/recent        # List active + recently archived playback sessions for diagnostics
GET    /api/playback/sessions/{id}          # Get playback session status snapshot
PATCH  /api/library/movies/{id}             # Update: isFavorite, rating (0-5), userTags, metadataTags, user* overrides
DELETE /api/library/movies/{id}             # Delete movie (move to trash)
DELETE /api/library/movies/{id}?permanent=true  # Permanently delete (must be in trash)
POST   /api/library/movies/{id}/restore     # Restore from trash
GET    /api/library/movies/{id}/stream      # Video stream (HTML5 video/Range requests)
POST   /api/library/movies/{id}/native-play # Legacy backend-side native player launch hook
GET    /api/playback/sessions/{id}/hls/{file} # Serve HLS playlists and segments for pushed playback sessions
POST   /api/library/movies/{id}/reveal      # Open OS file manager at primary video (server machine; path rules same as stream)
POST   /api/library/movies/{id}/scrape      # Re-scrape metadata (async task)
GET    /api/library/movies/{id}/comment     # Get user comment for movie
PUT    /api/library/movies/{id}/comment     # Upsert user comment for movie
GET    /api/library/actors                  # List actors (query: q, actorTag, sort, limit, offset)
GET    /api/library/actors/profile          # Get actor profile (query: name)
GET    /api/library/actors/{name}/asset/avatar # Get same-origin cached actor avatar
PATCH  /api/library/actors/tags             # Update actor user tags (query: name)
PATCH  /api/library/actors/external-links   # Update actor user external links (query: name)
POST   /api/library/actors/scrape           # Scrape actor metadata (async task)
POST   /api/library/actors/merge-preview    # Read-only canonical merge preview with a complete-state token
POST   /api/library/actors/merge            # Confirm and transactionally apply an actor merge
GET    /api/library/actors/merge-audits     # List persisted immutable actor merge audits
GET    /api/insights/overview               # Bounded local watch/completion/rating aggregate for 30d/90d/365d/all
GET    /api/insights/breakdown              # Bounded canonical actor/studio/tag full-attribution ranking (max 25)
GET    /api/library/played-movies           # List played movies with timestamps
POST   /api/library/played-movies/{id}      # Mark movie as played
POST   /api/library/paths                   # Add library path
POST   /api/library/paths/{id}/reveal       # Open configured library root in OS file manager
GET    /api/library/paths/storage-status    # List storage availability for configured library roots
POST   /api/library/paths/storage-status/check # Fresh storage probe; optional body libraryPathIds narrows scope
POST   /api/library/paths/{id}/storage-binding/rebind # Bind a library path to the currently detected backing volume
PATCH  /api/library/paths/{id}              # Update library path
DELETE /api/library/paths/{id}              # Delete library path
POST   /api/library/metadata-scrape         # Batch metadata refresh by library paths
POST   /api/import/movies                   # Copy uploaded movie files into the configured default library path (returns import.movies task)
POST   /api/import/movies/uploads           # Create resumable movie import upload session for large browser uploads
GET    /api/import/movies/uploads/{id}      # Get resumable upload status
PUT    /api/import/movies/uploads/{id}/files/{fileId}/chunks/{chunkIndex} # Upload one raw binary chunk
POST   /api/import/movies/uploads/{id}/commit # Commit staged chunks into the library and trigger scan
DELETE /api/import/movies/uploads/{id}      # Abort resumable upload and remove staging files
GET    /api/settings                        # Get settings (includes backupDirectory / autoDownloadUpdates / launchAtLogin / launchAtLoginSupported)
PATCH  /api/settings                        # Partial update (persisted to config/library-config.cfg)
POST   /api/proxy/ping-javbus               # Test proxy: GET https://www.javbus.com/ (body.proxy optional = use form draft; omit = use persisted proxy)
POST   /api/proxy/ping-google               # Test proxy: GET https://www.google.com/ (same body as ping-javbus)
POST   /api/scans                           # Start scan task
GET    /api/events                          # SSE backend events; currently streams task.updated snapshots
GET    /api/tasks/recent                    # Recently finished tasks (for UI toasts)
GET    /api/tasks/{taskId}                  # Get task status
POST   /api/library/movies/{movieId}/clips # Queue bounded GIF clip generation; curatedFrameId persists it with a curated frame (0.4–6 seconds)
GET    /api/tasks/{taskId}/artifact         # Download completed GIF clip artifact
GET    /api/playback/progress               # List all playback progress
PUT    /api/playback/progress/{movieId}     # Update playback progress
DELETE /api/playback/progress/{movieId}     # Delete playback progress
GET    /api/playback/watch-time/daily       # List daily watch-time totals for Settings overview
POST   /api/playback/watch-time/daily       # Add one bounded watch-time delta
GET    /api/curated-frames                  # List curated frames (q, actor, movieId, tag, limit, offset; returns total/limit/offset)
GET    /api/curated-frames/stats            # Curated frames total count
GET    /api/curated-frames/tags             # Curated frame tag facets
GET    /api/curated-frames/actors           # Curated frame actor facets
POST   /api/curated-frames                  # Create curated frame (legacy JSON imageBase64 or multipart metadata + image); near-duplicates are allowed and reviewed in the library UI
POST   /api/curated-frames/export           # Export 1–20 frames as JPG/WebP/PNG with embedded tags/schemaVersion/exportedAt/appName/appVersion or ZIP
GET    /api/curated-frames/{id}/image       # Get curated frame image
GET    /api/curated-frames/{id}/thumbnail   # Get curated frame thumbnail
GET    /api/curated-frames/{id}/motion      # Stream persisted GIF motion artifact when ready
PATCH  /api/curated-frames/{id}/tags        # Update frame tags
DELETE /api/curated-frames/{id}             # Delete curated frame
POST   /api/providers/ping                  # Ping a single provider
POST   /api/providers/ping-all              # Ping all providers
```

**Async Task Pattern:** Long-running operations (scan, movie scrape, actor scrape) return a task ID. Poll `GET /api/tasks/{taskId}` for progress. Frontend uses `useScanTaskTracker()` composable for this.

**PIN App Lock:** PIN lock is disabled by default. When enabled, all protected `/api/*` routes are guarded by backend middleware and return `423 AUTH_LOCKED` without a valid `curated_auth` HTTP-only cookie. PIN values are stored in SQLite only as Argon2id salted hashes; the non-secret PIN length is stored separately and returned as `pinLength` so `/lock` can render the correct number of keyboard-entry cells. Curated now uses one global PIN policy for local and LAN clients; the old `lanRequiresPin` status field remains fixed at `true` only for compatibility and is no longer writable or shown as a switch. Regular unlock sessions use `sessionTtlMinutes` as an idle-lock delay: protected API use and frontend activity refresh `/api/auth/status`, extending `sessionExpiresAt` instead of locking on a fixed countdown. Unlock can also use `{ "trustedForever": true }`, which leaves `sessionExpiresAt` empty and survives backend restart-lock cleanup until the current device is explicitly locked or the session is revoked. Trusted sessions have a separate random `public_id`, allowing list/revoke APIs and a Settings -> Security device/session manager without exposing the bearer token stored in the HTTP-only cookie; the UI marks the current device and confirms single/other-device revocation. `/api/health`, `/api/auth/status`, `/api/auth/setup-pin`, `/api/auth/unlock`, and `/api/auth/lock` remain public so the lock UI can render and recover; `POST /api/auth/change-pin` is protected and additionally verifies the current PIN. Invalid setup/unlock attempts are limited by source IP and hashed client key: after five consecutive failures the backend applies exponential backoff and returns `429 AUTH_RATE_LIMITED` with `Retry-After` plus `details.retryAfterSeconds`; successful setup/unlock clears the counters.

**Library directory watch (fsnotify):** When the main config allows it (`libraryWatchEnabled`, default on) and **`autoLibraryWatch`** is true (default, persisted in `library-config.cfg`), the backend watches library roots for new files and, after debounce, queues a scan with `trigger: fsnotify`. Turning **`autoLibraryWatch`** off stops the watch loop and ignores watch-driven enqueue; manual or interval full scans are unchanged. When **`autoActorProfileScrape`** is true, successful movie metadata scrapes also enqueue `scrape.actor` tasks for actors that still lack both avatar and summary, and a bounded background sweep backfills the same missing-profile actors after startup and every 15 minutes (batch of 50, 24-hour failed-attempt cooldown).

**Homepage daily recommendations and feedback:** `GET /api/homepage/recommendations` returns the UTC-day snapshot used by the homepage hero and today's recommendations. Generation version `v8` persists both the compatibility `recommendationMovieIds` list and same-order `recommendations` items with truthful reason codes plus optional `feedbackEffects`; the renderer only localizes those codes and does not invent Web API explanations. In Web API mode SQLite stores the display snapshot, long-lived per-movie memory in `homepage_recommendation_states`, and explicit feedback in `homepage_recommendation_feedback` (migrations `0033`/`0034`). `not_interested` permanently excludes a movie until feedback removal, `snooze` excludes it for 1-365 days, and `less` reduces actor/studio/tag weight while retaining a 0.1 exploration floor. Actor feedback and diversity use the same canonical Unicode identity normalization as actor aliases. Feedback targets must belong to the active source movie, duplicate submissions are idempotent, expired snoozes are ignored/cleaned, and the active set is bounded to 500. Mock mode mirrors the behavior with isolated `localStorage` key `curated-homepage-recommendation-feedback-v1`. Creating/removing feedback leaves the current card visible for immediate undo; the next explicit generation applies it. `POST /api/homepage/recommendations/refresh` only regenerates and never creates negative feedback; optional `preserveHeroMovieIds` and `excludeRecommendationMovieIds` retain the current hero and avoid the current rail where inventory allows.

**Saved Views:** The library toolbar can save, apply, update, rename, reorder, and delete versioned reusable filters. Web API mode persists canonical definitions in SQLite table `library_saved_views` (migration `0032`); Mock mode uses isolated `localStorage` key `curated-library-saved-views-v1` and never clears `jav-library-movie-prefs`. Filter v1 includes library mode, search, exact tag/actor/studio (tag and actor may be comma-separated AND; studio may be comma-separated OR), tab, play state, exact local user rating, normalized resolution, and a relative added-within-days window. Applying a view rebuilds a canonical route from an empty query; `selected`, `from`, `browse`, `back`, `autoplay`, and `t` are navigation-only and are never persisted. Deleting a Saved View never deletes media, ratings, playback history, or tags.

**Actor canonical identities and audited merges:** Migrations `0035`/`0036` add `actors.normalized_name`, globally unique `actor_aliases`, and immutable `actor_merge_audits` whose ID/name snapshots do not pin live actor rows. Identity comparison uses Unicode NFKC, case folding, trim, and whitespace folding. Profile lookup, list search, exact library filtering, avatar/tag/link operations, scraping, and metadata ingestion resolve aliases to the canonical actor. `POST /api/library/actors/merge-preview` is read-only and returns an opaque token covering complete relevant state; `POST /api/library/actors/merge` requires `confirm:true`, rejects stale tokens and unresolved profile conflicts, and atomically preserves/deduplicates movie links, actor tags, ordered external links, profile/avatar state, recommendation feedback, aliases, and curated-frame actor JSON before deleting the source. A canonical target can later merge again without losing earlier audits. The actor detail UI provides the preview/decision/confirmation flow and canonicalizes old alias routes. Mock mode mirrors alias/audit persistence with `curated-actor-merges-v1`.

**Personal Insights:** The lazy `/insights` route calls `GET /api/insights/overview` plus three parallel `GET /api/insights/breakdown` requests for `actor`, `studio`, and `tag`. Ranges are fixed to `30d`, `90d`, `365d`, and `all`, with explicit inclusive `from`/`to` local calendar days and an IANA timezone; Go embeds `time/tzdata` for packaged Windows builds. `watchedSeconds` and started movies come from bounded `playback_daily_watch_time` aggregation. Completed means a movie watched in the selected range whose **current** saved progress reaches 90%; rated/average use the **current** local user rating for started movies, not historical completion/rating event times. Empty denominators return JSON `null`. Breakdowns are capped at 25, use stable ordering and `full-per-entity` attribution, so totals across actors/tags may exceed 100%; canonical actor aliases do not create separate rows. Mock mode performs the same bounded aggregate inside the adapter from local watch-time/progress state and never exposes raw rows to the page. Range changes reject stale responses, and null rates render as `—`, not a misleading `0%`.

**Library storage presence:** `GET /api/library/paths/storage-status`, `POST /api/library/paths/storage-status/check`, and `POST /api/library/paths/{id}/storage-binding/rebind` report whether configured library roots are online, offline, mismatched to their previously bound volume, missing, permission denied, or unknown. Windows is the primary supported volume-identity implementation; macOS/Linux currently rely on the fallback path probe and are future adaptation targets. The frontend checks storage on Web API startup, surfaces abnormal paths through the new toast/notification center, blocks scan/import actions when `canRescan` or `canImport` is false, and exposes manual rebind in Settings -> Video storage for deliberate disk replacement or path migration.

**Curated Desktop client marker:** Electron injects `X-Curated-Client: desktop-electron`, `X-Curated-Client-Version`, `X-Curated-OS`, and `X-Curated-OS-Version` on requests to the backend origin. `backend/internal/clienttracker` uses that marker before User-Agent parsing, reports the browser name as `Curated Desktop`, includes the marker in its in-memory client key so the desktop shell is not merged with a normal Chrome tab on the same machine, and trusts the desktop OS headers to show Windows 11 instead of Chromium's legacy `Windows NT 10.0` token. For regular browsers, the backend also uses `Sec-CH-UA-Platform` and `Sec-CH-UA-Platform-Version` when present.

**App update checks/download/install:** `GET /api/app-update/status` returns the packaged-app update state used by Settings -> About and the sidebar brand badge, while `POST /api/app-update/check` forces a refresh. The backend compares the current runtime `installerVersion` with the latest GitHub Release for `yepHiu/Curated`, returns `installerDownloadUrl` and `installerSha256` when the release includes a Windows `.exe` installer asset with a digest, caches the result in SQLite, reuses the process proxy settings for outbound requests, and uses `0.0.0` as the dev-runtime fallback when no packaged installer version was injected. Settings -> General exposes persisted `autoDownloadUpdates`; when enabled, the startup background check may automatically download and SHA256-verify a newer installer, but it never auto-installs one. `POST /api/app-update/download` downloads to the backend update cache and verifies SHA256; `POST /api/app-update/install` launches the verified installer only after explicit user action. Current local packages are unsigned, so this flow relies on SHA256 integrity and keeps silent auto-install disabled by default.

**Backup maintenance controls:** `POST /api/maintenance/backups`, `/verify`, and `/preflight` are protected by the existing PIN middleware, accept strict JSON with backend-machine absolute paths, and cap request bodies at 64 KiB. Creation never overwrites an existing destination; it uses an atomic hard-link commit where supported and an `O_EXCL` copy fallback for filesystems such as exFAT. Settings -> Maintenance exposes create-and-verify, verify, and preflight in Web API mode, disables them in Mock mode, and keeps the actual restore exclusively in the offline maintenance CLI. After creation succeeds, the Web adapter persists only the selected directory through `PATCH /api/settings` as `backupDirectory`; subsequent settings loads prefill it while generating a new timestamped filename for every package.

**Offline path migration:** `path-migrate-plan` and `path-migrate-apply` are CLI-only maintenance actions. They map absolute Windows/UNC/Unix prefixes without arbitrary text replacement and only touch `library_paths.path`, `movies.location`, `scan_items.path`, `media_assets.local_path`, `actors.avatar_local_path`, `library_path_storage_bindings.root_path`, and `app_update_status.downloaded_file_path`. Plan is database-read-only and returns structured affected counts, samples, target status, conflicts, errors, warnings, and `canApply`. Apply requires `-confirm-path-migration` plus a new `-backup-path`, creates and verifies that backup first, then re-plans and commits path changes, binding deletion, integrity checks, and `path_migration_audits` atomically.

**Library Health and repair:** `POST /api/library/health/scan` is read-only and reports stable category counts/findings for SQLite quick/foreign-key checks, storage roots, source files, registered assets, duplicates, metadata attempts, orphan user state, and first-level `.curated-import` residue. Offline or mismatched roots produce one root finding and skip per-file checks. `POST /api/library/health/repairs` accepts only freshly revalidated `metadata_missing` / `metadata_failed` findings, requires `confirm:true`, is bounded to 100 backend items / 25 Settings items, persists the parent run and every child `scrape.movie` result, and marks restart interruptions explicitly. `POST /api/library/health/actions` similarly revalidates exact findings before whitelisted orphan-state deletion or strict non-symlink `upload_<16hex>` staging cleanup; both paths write audit evidence and never target final movie files. Settings -> Maintenance exposes scan/export, confirmation dialogs, task progress, and per-item results in Web API mode; Mock mode stays disabled. The health workspace and non-default locale payloads are dynamically loaded so the verified production bundle remains below hard budgets.

## Architecture Boundaries

**Implemented (Current State):**
- Go HTTP backend with SQLite database
- File scanning, metadata scraping, task system
- REST API at `/api`
- Frontend connects via HTTP when `VITE_USE_WEB_API=true`; backend defaults are loopback-only (`127.0.0.1:8080` dev, `127.0.0.1:8081` release). A non-loopback main-config `httpAddr` also requires `lanEnabled: true` and an initialized application PIN.
- Electron shell MVP under `electron/`: starts or reuses the Go HTTP backend, waits for `/api/health`, starts or reuses Vite at `http://127.0.0.1:5173` in development, loads the Web UI in BrowserWindow with the Curated app icon and a versioned renderer query, keeps the app running in the tray when the window is closed, and exposes only `window.javLibrary.pickDirectory()` through preload for native directory selection. Packaged releases install `Curated.exe` as the Electron shell, package Electron app files under `resources/app/electron-dist`, place the Go backend at `resources/app/curated.exe`, load the backend-hosted static UI on `http://127.0.0.1:8081`, serve entry HTML with `no-store`, and remove managed old `frontend-dist` trees before installer upgrades.
- Library storage presence checks for configured roots are implemented Windows-first, with macOS/Linux kept as fallback/future adaptation targets
- Playback uses HTML5 `<video>` with HTTP Range streaming
- Trash/restore functionality (soft delete with `trashedAt` timestamp)

**Not Yet Implemented (Documented as Targets):**
- Deep Electron preload/main process IPC for business APIs or broad native desktop bridges
- mpv player integration with named pipes
- Desktop file system bridge
- WebHID / node-hid controller depth features such as touchpad gestures, adaptive triggers, LED control, and Electron main-process controller integration

**Design Principle:** Frontend code should not assume Electron, mpv, PotPlayer, or any native-player executable exists. All business logic goes through composables plus the service layer (`useLibraryService()`, service contracts, and adapter-owned `src/api`) to allow swapping transport (HTTP now, IPC later). Shared state defaults to composables and services; Pinia can be evaluated later from a small, bounded service/new feature when it reduces complexity, rather than as a broad upfront migration.

## Key Documentation

Reference these docs in `docs/` for detailed specifications:

- `API.md` - Public HTTP API reference; update this when public API behavior changes
- `docs/guide.md` - Detailed handbook and documentation index; keep operational long-form content here instead of the root README

- `docs/product/2026-03-20-jav-libary.md` - Complete product design document (domain models, UI design, task system)
- `docs/reference/2026-03-21-backend-go-standards.md` - Go coding standards and directory structure
- `docs/reference/2026-03-21-backend-contract-constraints.md` - API contract design (commands, events, DTOs, error codes)
- `docs/reference/2026-03-24-frontend-ui-spec.md` - Frontend UI design tokens and component specifications
- `docs/reference/2026-03-20-project-memory.md` - Current implementation facts and architectural decisions
- `docs/film-scanner/CLAUDE.md` - Reference implementation for metadata scraping (if present)

Additional guidance in `.cursor/rules/`:
- `architecture-boundaries.mdc` - Current vs target architecture
- `backend-task-patterns.mdc` - Background task design
- `jav-library-frontend-patterns.mdc` - Frontend patterns
- `project-facts.mdc` - Detailed project implementation facts

## Testing

### Frontend Tests

- Uses Vitest with jsdom environment
- Test files: `*.test.ts` in `src/`
- Vue components tested with `@vue/test-utils`

### Backend Tests

- Uses standard Go testing
- Repository tests use in-memory SQLite
- Test files: `*_test.go` alongside source files

## Important Conventions

### Video ID (番号) Parsing

Video IDs are extracted from filenames using patterns like:
- `ABC-123`, `ABC_123` → `ABC-123`
- `abc123` → `ABC-123`
- Supports special prefixes: `FC2`, `heyzo`, `tokyo-hot`, `1pondo`, `caribbeancom`

### Error Codes

Backend uses stable error codes (see `backend/internal/contracts/contracts.go`):
- `COMMON_*` - General errors
- `LIBRARY_*` - Library operations
- `SCAN_*` - Scanning errors
- `SCRAPER_*` - Metadata scraping errors
- `PLAYER_*` - Player control errors
- `SETTINGS_*` - Configuration errors
- `CURATED_*` - Curated frames errors
- `PROVIDER_*` - Provider health check errors

### Database Migrations

Migrations are in `backend/internal/storage/migrations/` and run automatically on startup.

Migration `0029_movie_import_upload_sessions.sql` persists resumable movie-import sessions, ordered files, synchronized chunk-range ledgers, per-file commit markers, expiry/diagnostics, and cleanup audits. `movie_import_upload_repository.go` owns transactional counters and state transitions; `movie_import_upload_runtime.go` restores `uploading` / `committing` sessions and original task IDs on startup, reconciles interrupted commits, and runs the scoped upload janitor. Chunk bytes are `Sync`/`Close`d before SQLite records the range, and startup derives counters from chunk rows rather than preallocated file size.

Upload DTOs expose optional `expiresAt`, `recoveryStatus`, `recoveryError`, and per-file `state`. `recoveryStatus` is `ready`, `unavailable`, or `unrecoverable`; offline target storage is retained for later reconciliation. The 24-hour sliding TTL and 15-minute janitor are safety bounded: cleanup is audited, only strict old `.curated-import/upload_<16 lowercase hex>` or registered terminal/expired staging is eligible, symlinks/out-of-scope paths are skipped, and final destination files are never deleted. Stable errors include `IMPORT_UPLOAD_PERSIST_FAILED`, `IMPORT_UPLOAD_UNRECOVERABLE`, and `IMPORT_UPLOAD_EXPIRED`.

### Backend Task System

All long-running operations (scan, scrape, asset download, movie import) are modeled as background tasks:

- **Task lifecycle:** `pending` → `running` → `completed` | `partial_failed` | `failed` | `cancelled`
- **Task types:** `scan.library`, `scrape.movie`, `scrape.actor`, `import.movies`
- **SSE events:** `GET /api/events` streams non-blocking `task.updated` snapshots; frontend task tracking and library-watch toasts consume it in Web API mode
- **Polling fallback:** Frontend still polls `GET /api/tasks/{taskId}` for progress updates when SSE is unavailable
- **Recent tasks:** `GET /api/tasks/recent` returns recently completed tasks for UI toast notifications
- **Idempotency:** Tasks are designed to be safely retryable without duplicates

### Playback Progress Sync

Playback progress has dual storage depending on mode:

- **Web API mode (`VITE_USE_WEB_API=true`):** Synced to backend SQLite via `GET/PUT/DELETE /api/playback/progress`
- **Mock mode:** Stored in browser `localStorage` (key: `jav-library-playback-progress-v1`)

Daily watch-time uses a separate aggregate path:

- **Web API mode (`VITE_USE_WEB_API=true`):** Player reports bounded wall-clock deltas to `POST /api/playback/watch-time/daily`; Settings reads `GET /api/playback/watch-time/daily?days=91`
- **Mock mode:** Stored in browser `localStorage` (key: `curated-playback-watch-time-daily-v1`)
- The Settings overview renders watch-time summary metrics for a fixed 91-day window; heatmap visualization is not shown.
- The separate lazy `/insights` page provides 30/90/365/all-time overview plus actor/studio/tag breakdowns through the `LibraryService` boundary; it does not replace the fixed Settings operational summary.
- Settings -> Overview also renders connected clients directly below the watch-time summary. Web API mode polls `GET /api/connected-clients` while Overview is active; Mock mode returns representative local/LAN examples. The backend tracker is process-lifetime in memory, deduplicates by IP + User-Agent, caps at 50 recent clients, and does not collect MAC addresses.

### Playback Descriptor Seam

Player startup should consume `GET /api/library/movies/{id}/playback` instead of assuming playback is only a raw `/stream` URL.

- Current behavior: backend returns a direct-play descriptor pointing at `/api/library/movies/{id}/stream`
- Descriptor also carries resume position, filename, mime type, and future track/session fields
- Descriptor now also carries structured playback diagnostics: `sessionKind`, `reasonCode`, `reasonMessage`, `sourceContainer`, `sourceVideoCodec`, `sourceAudioCodec`
- Purpose: preserve current browser playback while creating the expansion seam for remux/transcode/native playback later
- Browser playback may now move onto a backend-managed HLS session when stream push is enabled
- HLS startup is remux-first when the source is already HLS-friendly, with fallback to hardware/software transcode profiles
- The frontend keeps HLS playback inside the existing player page and loads the npm-bundled official `hls.js/light` build on demand when the browser lacks native HLS support; Curated's current single-rendition local stream does not require the full build's subtitle, EME/DRM, alternate-audio, or CMCD controllers, and packaged desktop builds no longer rely on a CDN HLS script
- The backend now keeps a bounded in-memory archive for recent playback sessions so `GET /api/playback/sessions/recent` and `GET /api/playback/sessions/{id}` can diagnose recently stopped or expired HLS sessions
- The current player page prefers browser-side local-player handoff for external playback. With the PotPlayer preset, the frontend uses a browser protocol template (default `potplayer:{url}`) instead of depending on backend process launch.
- `POST /api/library/movies/{id}/native-play` still exists as a legacy/native-shell hook, but it is no longer the default path for the player page button

### Movie Comments

User comments/notes per movie:

- **Web API mode:** Stored in backend via `GET/PUT /api/library/movies/{id}/comment` (table `library_movie_comments`)
- **Mock mode:** Stored in `localStorage` (key: `jav-library-movie-comment-v1`)

### Curated Frames

Frame extraction and management:

- **Web API mode:** paginated `GET /api/curated-frames`, `GET /api/curated-frames/stats`, `GET /api/curated-frames/tags`, `GET /api/curated-frames/actors`, `POST/GET/PATCH/DELETE /api/curated-frames`, with `GET /api/curated-frames/{id}/image` and `GET /api/curated-frames/{id}/thumbnail`
- **Export:** `POST /api/curated-frames/export` supports JPG (EXIF `UserComment`), WebP (EXIF metadata), or PNG (iTXt metadata) formats and embeds `tags`, `schemaVersion`, `exportedAt`, `appName`, and `appVersion`
- **Mock mode:** Stored in IndexedDB

### Trash/Restore

Movies support soft-delete (trash) workflow:

- **Delete:** Moves movie to trash (sets `trashedAt` timestamp); can be restored
- **Permanent delete:** Only allowed for movies already in trash
- **Restore:** Removes `trashedAt` timestamp, movie returns to library
- **Trash view:** Access via `mode=trash` query parameter on library endpoint

## Frontend Patterns

### Service Layer Usage

All data access and mutations go through the service layer:

```typescript
// Use the library service composable
const service = useLibraryService()
const movies = await service.getMovies({ limit: 50 })
```

Do not bypass the service layer for library actions. Views/components should not import concrete Web/Mock adapters directly; add a service-contract method when UI needs a new business capability.

### State Management

Use Vue composables and service-layer refs/computed values as the default shared-state model. Pinia or another store may be introduced incrementally for a small, bounded service or new feature when it removes real complexity; avoid starting with an app-wide migration or duplicate global store.

### Toast Notifications

Use the global toast system:

```typescript
import { pushAppToast } from '@/composables/use-app-toast'

pushAppToast('success', 'Operation completed')
pushAppToast('error', 'Something went wrong', { duration: 5000 })
```

### Player Route Navigation

The player page supports navigation context:
- `?t=123` - Start at specific timestamp (seconds)
- `?from=history` - Return to history page instead of detail page
- Build player route with helper: `buildPlayerRouteFromBrowse(movieId, options)`

### shadcn-vue Components

- Use `@/components/ui/input` `Input` component for form text fields (with explicit import)
- Follow dark mode contrast guidelines for form controls on dark surfaces
- Use existing theme tokens; avoid raw color values

### Actor Profile Card

When viewing library with `actor=` query param and `VITE_USE_WEB_API=true`, the `LibraryPage` displays an `ActorProfileCard` at the top showing actor info from `GET /api/library/actors/profile` with scrape capability via `POST /api/library/actors/scrape`.

## Development Notes

- The frontend Vite dev server proxies `/api` to `http://127.0.0.1:8080` (backend)
- Backend supports three modes: `http` (default), `stdio`, `both`
- Dev builds now expose backend name `curated-dev`; release builds keep `curated`
- Windows dev backend binary naming is an explicit constraint: keep dev builds as `curated-dev.exe` and reserve `curated.exe` for release/package builds only
- Current state: Frontend uses web adapter when `VITE_USE_WEB_API=true` (default in `.env`), mock adapter otherwise
- Settings -> About now includes packaged-app update status, a manual update-check action, in-app latest `.exe` installer download with SHA256 verification, explicit installer launch when ready, and a release-page fallback link; Settings -> General adds persisted `autoDownloadUpdates` for opt-in startup background download-and-verify behavior; when an update is available, the sidebar shows a lightweight `New` badge (expanded) or dot (compact) that links to `Settings -> About`, while the `Curated` brand text/icon links to the home page
- In development only, `src/layouts/AppShell.vue` mounts a fixed bottom overlay `DevPerformanceBar.vue`. It does not participate in page layout and aggregates frontend runtime sampling, request stats from `src/api/http-client.ts`, backend health, and `GET /api/dev/performance`.
- Auto-scan loop runs in background when backend starts
- Library organization (`organizeLibrary`), directory-watch-driven auto scan (`autoLibraryWatch`), missing actor profile scraping after movie scrapes plus a bounded library sweep (`autoActorProfileScrape`), background installer auto-download (`autoDownloadUpdates`), default movie import target (`defaultImportLibraryPathId`), remembered backup directory (`backupDirectory`), Windows login autostart (`launchAtLogin`), curated-frame export format (`curatedFrameExportFormat`, default `jpg`), and default curated-frame export style (`curatedFrameExportMode`, default `raw`, `raw` or `watermarked`) can be updated via `PATCH /api/settings` (persisted in `config/library-config.cfg`)
- Async tasks (scan, scrape): use `useScanTaskTracker()` composable to poll task status
- Task / provider diagnostics now carry machine-readable failure categories (`errorCategory`) for mainland-network troubleshooting
- i18n locale files are in `src/locales/` (en.json, ja.json, zh-CN.json)
