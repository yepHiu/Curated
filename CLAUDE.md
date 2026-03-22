# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Curated** (product name; repo folder `jav-shadcn`) is a desktop-oriented media library application for managing, browsing, scraping, and playing video collections. It consists of a Vue 3 frontend with a Go backend, using SQLite for persistence and metatube-sdk-go for metadata scraping.

**Current Architecture Phase:** Web phase (Vue SPA + Go HTTP API). Future target is Electron desktop app with mpv player integration.

## Tech Stack

- **Frontend:** Vue 3 + TypeScript + Vite + Tailwind CSS v4 + shadcn-vue
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
go build -o javd.exe ./cmd/javd

# Run backend HTTP server (default mode)
./javd.exe

# Run with specific config
./javd.exe -config path/to/config.yaml

# Run in stdio mode (for future Electron bridge)
./javd.exe -mode stdio

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
cd backend && go run ./cmd/javd
# Or use pre-built binary: ./backend/javd.exe

# Terminal 2: Start frontend dev server
pnpm dev
```

**Environment Variables:**
- `VITE_USE_WEB_API=true` - Use real backend API (set in root `.env` by default)
- `VITE_API_BASE_URL` - Override API base URL (default: `/api`)

## Architecture

### Frontend Structure (`src/`)

```
src/
  api/              # HTTP client and endpoint definitions
  components/
    jav-library/    # Domain-specific components
    ui/             # shadcn-vue UI components
  composables/      # Vue composables (e.g., use-scan-task-tracker.ts)
  domain/           # Domain types and logic
    library/
    movie/
  lib/              # Utilities and typed mock data (jav-library.ts)
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
- Routes: `library`, `favorites`, `recent`, `tags`, `history`, `detail/:id`, `player/:id`, `settings`
- Playback progress: stored in browser `localStorage` only (key: `jav-library-playback-progress-v1`), not synced to backend
- History page: `src/views/HistoryView.vue` displays watch history grouped by date (local browser only)

### Backend Structure (`backend/`)

```
backend/
  cmd/javd/         # Application entry point
  internal/
    app/            # Application lifecycle and wiring
    config/         # Configuration management
    contracts/      # DTOs, error codes, shared interfaces
    library/        # Library domain service
    logging/        # Zap logger setup
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
GET    /api/library/movies                  # List movies (query: mode, q, limit, offset)
GET    /api/library/movies/{id}             # Get movie detail
PATCH  /api/library/movies/{id}             # Update: isFavorite, rating (0-5), userTags, metadataTags
DELETE /api/library/movies/{id}             # Delete movie
GET    /api/library/movies/{id}/stream      # Video stream (HTML5 video/Range requests)
POST   /api/library/movies/{id}/scrape      # Re-scrape metadata (async task)
POST   /api/library/paths                   # Add library path
PATCH  /api/library/paths/{id}              # Update library path
DELETE /api/library/paths/{id}              # Delete library path
GET    /api/settings                       # Get settings (libraryPaths, player, organizeLibrary, autoLibraryWatch, metadataMovieProvider[s], …)
PATCH  /api/settings                       # Partial update: organizeLibrary, autoLibraryWatch, metadataMovieProvider (persisted to config/library-config.cfg where applicable)
POST   /api/scans                          # Start scan task
GET    /api/tasks/recent                   # Recently finished tasks (for UI toasts)
GET    /api/tasks/{taskId}                  # Get task status
```

**Async Task Pattern:** Long-running operations (scan, scrape) return a task ID. Poll `GET /api/tasks/{taskId}` for progress. Frontend uses `useScanTaskTracker()` composable for this.

**Library directory watch (fsnotify):** When the main config allows it (`libraryWatchEnabled`, default on) and **`autoLibraryWatch`** is true (default, persisted in `library-config.cfg`), the backend watches library roots for new files and, after debounce, queues a scan with `trigger: fsnotify`. Turning **`autoLibraryWatch`** off stops the watch loop and ignores watch-driven enqueue; manual or interval full scans are unchanged.

## Architecture Boundaries

**Implemented (Current State):**
- Go HTTP backend with SQLite database
- File scanning, metadata scraping, task system
- REST API at `/api`
- Frontend connects via HTTP when `VITE_USE_WEB_API=true`
- Playback uses HTML5 `<video>` with HTTP Range streaming

**Not Yet Implemented (Documented as Targets):**
- Electron shell, preload script, main process IPC
- mpv player integration with named pipes
- Desktop file system bridge
- Server-side playback history (currently localStorage-only)

**Design Principle:** Frontend code should not assume Electron/mpv exists. All business logic goes through the service layer (`useLibraryService()`, `src/api/`) to allow swapping transport (HTTP now, IPC later).

## Key Documentation

Reference these docs in `docs/` for detailed specifications:

- `jav-libary.md` - Complete product design document (domain models, UI design, task system)
- `backend-go-standards.md` - Go coding standards and directory structure
- `backend-contract-constraints.md` - API contract design (commands, events, DTOs, error codes)
- `film-scanner/CLAUDE.md` - Reference implementation for metadata scraping

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

### Database Migrations

Migrations are in `backend/internal/storage/migrations/` and run automatically on startup.

## Development Notes

- The frontend Vite dev server proxies `/api` to `http://localhost:8080` (backend)
- Backend supports three modes: `http` (default), `stdio`, `both`
- Current state: Frontend uses web adapter when `VITE_USE_WEB_API=true` (default in `.env`), mock adapter otherwise
- Auto-scan loop runs in background when backend starts
- Library organization (`organizeLibrary`) and directory-watch-driven auto scan (`autoLibraryWatch`) can be toggled via `PATCH /api/settings` (persisted in `config/library-config.cfg`)
- Async tasks (scan, scrape): use `useScanTaskTracker()` composable to poll task status
