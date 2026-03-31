# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Curated** (product name; repo folder `jav-shadcn`) is a desktop-oriented media library application for managing, browsing, scraping, and playing video collections. It consists of a Vue 3 frontend with a Go backend, using SQLite for persistence and metatube-sdk-go for metadata scraping.

**Current Architecture Phase:** Web phase (Vue SPA + Go HTTP API). Future target is Electron desktop app with mpv player integration.

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

# Start development server (proxies /api to localhost:8080)
pnpm dev

# Build for production
pnpm build

# Run linter
pnpm lint

# Run tests
pnpm test

# Run single test file
pnpm vitest run src/path/to/file.test.ts

# Type check only
pnpm typecheck
```

### Backend

```bash
# Build the backend binary
cd backend
go build -o curated.exe ./cmd/curated

# Run backend HTTP server (default mode)
./curated.exe

# Run with specific config
./curated.exe -config path/to/config.yaml

# Run in stdio mode (for future Electron bridge)
./curated.exe -mode stdio

# Run tests
go test ./...

# Run tests for specific package
go test ./internal/storage/...

# Run with verbose output
go test -v ./internal/storage/...
```

### Full Stack Development

```bash
# Terminal 1: Start backend (from repo root or backend/)
cd backend && go run ./cmd/curated
# Or use pre-built binary: ./backend/curated.exe

# Terminal 2: Start frontend dev server
pnpm dev
```

**Environment Variables:**
- `VITE_USE_WEB_API=true` - Use real backend API (set in root `.env` by default)
- `VITE_API_BASE_URL` - Override API base URL (default: `/api`)
- `VITE_LOG_LEVEL` - Optional default for browser `loglevel` (`trace`|`debug`|`info`|`warn`|`error`|`silent`); overridden by `localStorage['curated-client-log-level']` when set (Settings → Logging)

### Library Behavior Configuration

Library-specific settings are persisted to `config/library-config.cfg` (JSON) and merged on startup:

- **`organizeLibrary`** - Whether to organize library files into structured folders
- **`autoLibraryWatch`** - Whether to auto-scan when files change via fsnotify (default: `true`)
- **`metadataMovieProvider`** - Primary metadata provider for movie scraping
- **`logDir`** / **`logFilePrefix`** / **`logMaxAgeDays`** / **`logLevel`** - Backend Zap log file output (merged into the same fields as the main `-config` JSON); **`PATCH /api/settings`** field **`backendLog`** updates **`logDir`** / **`logMaxAgeDays`** / **`logLevel`** from the settings UI (omits **`logFilePrefix`** so manual `library-config.cfg` or default `curated` applies); **restart the backend** for new log directory/level to apply to file sinks
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
- Routes: `library`, `favorites`, `recent`, `tags`, `actors`, `history`, `detail/:id`, `player/:id`, `settings`
- Playback progress: dual storage (backend SQLite in Web API mode, `localStorage` in Mock mode)
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
GET    /api/health                          # Health check
GET    /api/library/movies                  # List movies (query: mode, q, limit, offset, actor, tag)
GET    /api/library/movies/{id}             # Get movie detail
GET    /api/library/movies/{id}/playback    # Playback descriptor (direct-play metadata now; future remux/transcode seam)
POST   /api/library/movies/{id}/playback-session  # Create explicit playback session (for example HLS stream push)
PATCH  /api/library/movies/{id}             # Update: isFavorite, rating (0-5), userTags, metadataTags, user* overrides
DELETE /api/library/movies/{id}             # Delete movie (move to trash)
DELETE /api/library/movies/{id}?permanent=true  # Permanently delete (must be in trash)
POST   /api/library/movies/{id}/restore     # Restore from trash
GET    /api/library/movies/{id}/stream      # Video stream (HTML5 video/Range requests)
POST   /api/library/movies/{id}/native-play # Launch external native player kernel when configured
GET    /api/playback/sessions/{id}/hls/{file} # Serve HLS playlists and segments for pushed playback sessions
POST   /api/library/movies/{id}/reveal      # Open OS file manager at primary video (server machine; path rules same as stream)
POST   /api/library/movies/{id}/scrape      # Re-scrape metadata (async task)
GET    /api/library/movies/{id}/comment     # Get user comment for movie
PUT    /api/library/movies/{id}/comment     # Upsert user comment for movie
GET    /api/library/actors                  # List actors (query: q, actorTag, sort, limit, offset)
GET    /api/library/actors/profile          # Get actor profile (query: name)
PATCH  /api/library/actors/tags             # Update actor user tags (query: name)
POST   /api/library/actors/scrape           # Scrape actor metadata (async task)
GET    /api/library/played-movies           # List played movies with timestamps
POST   /api/library/played-movies/{id}      # Mark movie as played
POST   /api/library/paths                   # Add library path
PATCH  /api/library/paths/{id}              # Update library path
DELETE /api/library/paths/{id}              # Delete library path
POST   /api/library/metadata-scrape         # Batch metadata refresh by library paths
GET    /api/settings                        # Get settings
PATCH  /api/settings                        # Partial update (persisted to config/library-config.cfg)
POST   /api/proxy/ping-javbus               # Test proxy: GET https://www.javbus.com/ (body.proxy optional = use form draft; omit = use persisted proxy)
POST   /api/proxy/ping-google               # Test proxy: GET https://www.google.com/ (same body as ping-javbus)
POST   /api/scans                           # Start scan task
GET    /api/tasks/recent                    # Recently finished tasks (for UI toasts)
GET    /api/tasks/{taskId}                  # Get task status
GET    /api/playback/progress               # List all playback progress
PUT    /api/playback/progress/{movieId}     # Update playback progress
DELETE /api/playback/progress/{movieId}     # Delete playback progress
GET    /api/curated-frames                  # List curated frames
POST   /api/curated-frames                  # Create curated frame
POST   /api/curated-frames/export           # Export 1–20 frames as WebP (EXIF JSON) or ZIP
GET    /api/curated-frames/{id}/image       # Get curated frame image
PATCH  /api/curated-frames/{id}/tags        # Update frame tags
DELETE /api/curated-frames/{id}             # Delete curated frame
POST   /api/providers/ping                  # Ping a single provider
POST   /api/providers/ping-all              # Ping all providers
```

**Async Task Pattern:** Long-running operations (scan, movie scrape, actor scrape) return a task ID. Poll `GET /api/tasks/{taskId}` for progress. Frontend uses `useScanTaskTracker()` composable for this.

**Library directory watch (fsnotify):** When the main config allows it (`libraryWatchEnabled`, default on) and **`autoLibraryWatch`** is true (default, persisted in `library-config.cfg`), the backend watches library roots for new files and, after debounce, queues a scan with `trigger: fsnotify`. Turning **`autoLibraryWatch`** off stops the watch loop and ignores watch-driven enqueue; manual or interval full scans are unchanged.

## Architecture Boundaries

**Implemented (Current State):**
- Go HTTP backend with SQLite database
- File scanning, metadata scraping, task system
- REST API at `/api`
- Frontend connects via HTTP when `VITE_USE_WEB_API=true`
- Playback uses HTML5 `<video>` with HTTP Range streaming
- Trash/restore functionality (soft delete with `trashedAt` timestamp)

**Not Yet Implemented (Documented as Targets):**
- Electron shell, preload script, main process IPC
- mpv player integration with named pipes
- Desktop file system bridge

**Design Principle:** Frontend code should not assume Electron/mpv exists. All business logic goes through the service layer (`useLibraryService()`, `src/api/`) to allow swapping transport (HTTP now, IPC later).

## Key Documentation

Reference these docs in `docs/` for detailed specifications:

- `jav-library.md` - Complete product design document (domain models, UI design, task system)
- `backend-go-standards.md` - Go coding standards and directory structure
- `backend-contract-constraints.md` - API contract design (commands, events, DTOs, error codes)
- `frontend-ui-spec.md` - Frontend UI design tokens and component specifications
- `project-memory.md` - Current implementation facts and architectural decisions
- `film-scanner/CLAUDE.md` - Reference implementation for metadata scraping

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

### Backend Task System

All long-running operations (scan, scrape, asset download) are modeled as background tasks:

- **Task lifecycle:** `pending` → `running` → `completed` | `partial_failed` | `failed` | `cancelled`
- **Task types:** `scan.library`, `scrape.movie`, `scrape.actor`
- **Polling:** Frontend polls `GET /api/tasks/{taskId}` for progress updates
- **Recent tasks:** `GET /api/tasks/recent` returns recently completed tasks for UI toast notifications
- **Idempotency:** Tasks are designed to be safely retryable without duplicates

### Playback Progress Sync

Playback progress has dual storage depending on mode:

- **Web API mode (`VITE_USE_WEB_API=true`):** Synced to backend SQLite via `GET/PUT/DELETE /api/playback/progress`
- **Mock mode:** Stored in browser `localStorage` (key: `jav-library-playback-progress-v1`)

### Playback Descriptor Seam

Player startup should consume `GET /api/library/movies/{id}/playback` instead of assuming playback is only a raw `/stream` URL.

- Current behavior: backend returns a direct-play descriptor pointing at `/api/library/movies/{id}/stream`
- Descriptor also carries resume position, filename, mime type, and future track/session fields
- Purpose: preserve current browser playback while creating the expansion seam for remux/transcode/native playback later
- Browser playback may now move onto a backend-managed HLS session when stream push is enabled
- The frontend keeps HLS playback inside the existing player page and loads `hls.js` on demand when the browser lacks native HLS support
- Native playback can be launched through `POST /api/library/movies/{id}/native-play` when `mpv` is configured

### Movie Comments

User comments/notes per movie:

- **Web API mode:** Stored in backend via `GET/PUT /api/library/movies/{id}/comment` (table `library_movie_comments`)
- **Mock mode:** Stored in `localStorage` (key: `jav-library-movie-comment-v1`)

### Curated Frames

Frame extraction and management:

- **Web API mode:** `POST/GET/PATCH/DELETE /api/curated-frames` with `GET /api/curated-frames/{id}/image`
- **Export:** `POST /api/curated-frames/export` supports WebP (EXIF metadata) or PNG (iTXt metadata) formats
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

Do not bypass the service layer for library actions.

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

- The frontend Vite dev server proxies `/api` to `http://localhost:8080` (backend)
- Backend supports three modes: `http` (default), `stdio`, `both`
- Current state: Frontend uses web adapter when `VITE_USE_WEB_API=true` (default in `.env`), mock adapter otherwise
- Auto-scan loop runs in background when backend starts
- Library organization (`organizeLibrary`) and directory-watch-driven auto scan (`autoLibraryWatch`) can be toggled via `PATCH /api/settings` (persisted in `config/library-config.cfg`)
- Async tasks (scan, scrape): use `useScanTaskTracker()` composable to poll task status
- i18n locale files are in `src/locales/` (en.json, ja.json, zh-CN.json)
