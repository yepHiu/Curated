# Curated

**Curated** is the product name. The repository folder and npm package may still be **`jav-shadcn`**. The Go module is **`curated-backend`**; the server entrypoint is **`backend/cmd/curated`**.

Local-first media library: **Vue 3** SPA plus a **Go** HTTP API (SQLite, folder scanning, Metatube scraping, asset cache). The current focus is a **web app**; a future **Electron** shell is documented separately.

## Stack

| Layer | Technologies |
|--------|----------------|
| Frontend | Vue 3, TypeScript, Vite 8, Tailwind CSS v4, shadcn-vue, vue-router, vue-virtual-scroller, **embla-carousel-vue** (preview image viewer) |
| Backend | Go (`backend/go.mod`), SQLite (modernc), zap, metatube-sdk-go |
| Dev proxy | Vite forwards `/api` → `http://localhost:8080` (`vite.config.ts`) |

## Repository layout

```text
.
├── src/                    # SPA: views, jav-library components, API client, library-service + Web/Mock adapters
├── backend/
│   ├── cmd/curated/        # Entry: HTTP / stdio / both
│   └── internal/           # Config, storage, scan, scrape, tasks, HTTP routes
├── config/
│   └── library-config.cfg  # Library JSON merged at startup (organizeLibrary, metadataMovieProvider, autoLibraryWatch, …)
├── docs/                   # Product notes, plans, UI spec
└── package.json            # pnpm scripts & deps (lockfile: pnpm-lock.yaml)
```

## Requirements

- **Node.js**: current LTS (compatible with Vite 8)
- **pnpm**: required; use `pnpm-lock.yaml`
- **Go**: `1.25.4+` (see `backend/go.mod`)

## Quick start

### 1. Backend (default `:8080` dev, `:8081` release builds)

Run from repo root or `backend/`. Without `-config`, built-in defaults apply (DB, cache, sample library paths).

**Development binary:** `pnpm backend:build:dev` produces **`backend/runtime/curated-dev.exe`** on Windows so the dev backend does not collide with the installed release backend.

**Production / release binary:** `go build -tags release ./cmd/curated` uses default **`httpAddr` `:8081`** (see `backend/internal/config/default_http_addr_*.go`) and still packages as **`curated.exe`**. Override anytime with JSON `httpAddr` or `-config`.

```bash
cd backend
go run ./cmd/curated
```

Useful flags:

- `-config <path>` — JSON config (`Config` in `backend/internal/config/config.go`)
- `-mode http` — HTTP only
- `-mode stdio` — stdio bridge only
- `-mode both` — HTTP + stdio
- `-mode tray` — Windows tray mode (the default for Windows release builds)

Health (dev): `GET http://localhost:8080/api/health` returns backend name **`curated-dev`**. Release build default port is **8081** and reports **`curated`**.

On Windows `release` builds, launching `curated.exe` now defaults to tray mode:

- Curated starts the local HTTP server in the background
- a Windows tray icon is created
- the first launch opens the browser automatically
- a second launch reuses the existing instance and opens the browser again
- the tray menu provides: open home, open settings, open logs, quit

### 2. Frontend

```bash
pnpm install
pnpm dev
```

Open the URL printed in the terminal (usually `http://localhost:5173`).

### 3. Mock vs Web API

- **Real backend**: set `VITE_USE_WEB_API=true` in a root `.env` (often already enabled).
- **Mock mode**: any other value — in-memory data, no Go process.

Optional: `VITE_API_BASE_URL` overrides the API base. Dev default is `/api` (Vite proxy → 8080). **`pnpm build`** uses **`http://127.0.0.1:8081/api`** when unset, matching the release backend default.

## Backend configuration (short)

If you omit `-config`, defaults typically include:

- **HTTP**: `:8080` (`go run` / dev, health name `curated-dev`); **`:8081`** for `go build -tags release` (health name `curated`, override with `httpAddr` in JSON)
- **Database**: `backend/runtime/curated.db` (path is relative to the working directory; see `databasePath` in JSON if you need a custom file).
- **Cache**: `backend/runtime/cache`
- **Initial library roots**: e.g. `videos_test`, `docs/film-scanner/videos_test` — adjust `libraryPaths` in JSON as needed.

Main JSON may include: `logLevel`, `httpAddr`, `databasePath`, `cacheDir`, `libraryPaths`, `autoScanIntervalSeconds`, **`libraryWatchEnabled`** / **`libraryWatchDebounceMs`**, scraper/timeouts, etc.

`config/library-config.cfg` merges **library-level** keys (`organizeLibrary`, `metadataMovieProvider`, **`autoLibraryWatch`**, …). **`autoLibraryWatch`**: when `false`, **fsnotify**-driven scans are disabled; manual or interval scans still run. Values can be updated via **`PATCH /api/settings`** and are written back to that file.

## HTTP API (summary)

See `backend/internal/server/server.go` for the full route table. Highlights:

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/health` | Health |
| GET | `/api/library/movies` | List (filters: `q`, `tag`, `actor`, …) |
| GET | `/api/library/movies/{id}` | Detail |
| GET | `/api/library/movies/{id}/playback` | Playback descriptor (direct-play metadata, resume position, future transcode seam) |
| POST | `/api/library/movies/{id}/playback-session` | Create playback session (for example HLS stream push) |
| GET | `/api/playback/sessions/recent` | List active and recently archived playback sessions for diagnostics |
| GET | `/api/playback/sessions/{id}` | Get playback session status snapshot |
| POST | `/api/library/movies/{id}/native-play` | Legacy backend-side native player launch hook |
| PATCH | `/api/library/movies/{id}` | Favorites, user rating, `userTags`, `metadataTags` |
| DELETE | `/api/library/movies/{id}` | Remove from library |
| GET | `/api/library/movies/{id}/stream` | Video stream (Range) |
| GET | `/api/playback/sessions/{id}/hls/{file}` | Serve HLS playlists and segments for pushed playback sessions |
| GET | `/api/library/movies/{id}/asset/...` | Cover, thumb, preview stills |
| POST | `/api/library/movies/{id}/scrape` | Re-scrape (async task) |
| GET/PATCH | `/api/settings` | Settings; library keys persisted to `library-config.cfg` |
| GET | `/api/library/actors` | Actor list (filters, tags) |
| GET | `/api/library/actors/profile` | Actor profile |
| GET | `/api/library/actors/{name}/asset/avatar` | Same-origin cached actor avatar |
| PATCH | `/api/library/actors/tags` | Replace actor user tags |
| POST | `/api/library/actors/scrape` | Scrape actor metadata (task) |
| GET/POST | `/api/library/played-movies` … | Played stats |
| GET/PUT/DELETE | `/api/playback/progress` … | Playback progress (SQLite) |
| POST/GET/… | `/api/curated-frames` … | Curated frames + export |
| POST | `/api/scans` | Start library scan |
| GET | `/api/tasks/recent` | Recent finished tasks |
| GET | `/api/tasks/{taskId}` | Task status |

DTOs and error codes: `backend/internal/contracts/contracts.go`, `src/api/types.ts`.

Scrape stability additions:

- actor avatars can now be cached locally and served from the backend instead of direct browser requests to remote image hosts
- movie preview images still prefer local cache, and now have a backend fetch fallback when only `source_url` is known
- settings now support a higher-level `metadataMovieStrategy` alongside legacy `metadataMovieScrapeMode`
- provider health responses and task payloads include richer machine-readable failure categories for mainland-China troubleshooting

## Frontend behavior (short)

- **Library**: virtualized poster grid; **batch mode** for multi-select actions (favorites, tags, trash, metadata refresh where supported).
- **Detail**: metadata, user tags, **preview gallery**; fullscreen viewer uses **Embla** main carousel + synced **thumbnail strip** (arrow keys, buttons, drag, click thumb).
- **Actor filter** (`actor=` on library): **Actor profile card** with Metatube refresh and **actor user tags** (same API as actor library cards).
- **Actors** page: actor cards with tag editing; **History** for playback history grouping.
- **Settings**: overview, paths, scraping, proxy, library behavior, curated frames, playback, maintenance.

## Playback & history

- **Web API mode** (`VITE_USE_WEB_API=true`): progress syncs to SQLite via **`/api/playback/progress`**; played markers via **`/api/library/played-movies`**.
- Player startup now has a dedicated playback descriptor seam via **`GET /api/library/movies/{id}/playback`**; the current implementation still returns direct-play metadata and `/stream`, but this is the planned expansion point for remux/transcode later.
- Playback descriptors now carry structured decision diagnostics such as **`sessionKind`**, **`reasonCode`**, **`reasonMessage`**, **`sourceContainer`**, **`sourceVideoCodec`**, and **`sourceAudioCodec`** so the player page can explain why direct play, remux HLS, or transcode HLS was chosen.
- When backend stream push is enabled, Curated now prefers browser playback through **HLS** session output under **`/api/playback/sessions/{id}/hls/...`** by default.
- HLS startup is now **remux-first** when the source is already browser-friendly enough for stream copy, with automatic fallback to hardware/software transcode profiles when remux startup fails.
- The current frontend HLS path keeps playback inside the existing player page and loads `hls.js` on demand for browsers without native HLS support.
- HLS session governance now includes an idle-session janitor plus status endpoints under **`/api/playback/sessions/recent`** and **`/api/playback/sessions/{id}`**. Those endpoints expose active sessions and a bounded in-memory archive of recently closed/expired sessions with `state`, `transcodeProfile`, `lastAccessedAt`, `expiresAt`, `finishedAt`, and `lastError`.
- The player page now prefers a **browser-side local-player handoff** for external playback. In the current UI, the PotPlayer preset uses a local browser protocol template (default `potplayer:{url}`) so the backend does not need to execute a player binary directly.
- The older backend route **`POST /api/library/movies/{id}/native-play`** still exists as a legacy/native-shell hook, but it is no longer the default path for the player page's local-player button.
- **Mock mode**: progress in **`localStorage`** (`jav-library-playback-progress-v1`).
- History UI: sidebar **History** → `history` route; player can use `?t=` and `?from=history` for return navigation.

## Scripts

```bash
pnpm dev        # Dev server
pnpm build      # typecheck + production build
pnpm preview    # Preview build
pnpm typecheck  # vue-tsc only
pnpm lint       # ESLint
pnpm test       # Vitest
```

## Release Packaging

The repository now includes a Windows-oriented release script scaffold under `scripts/release/`.

Recommended entrypoint:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/release/publish.ps1 -Version 0.1.0
```

This flow currently:

- builds the frontend production bundle
- builds the Go backend with `-tags release`
- injects `internal/version.BuildStamp`
- assembles a `release/Curated` staging directory
- packages a portable zip
- generates an Inno Setup installer script and builds the installer when `ISCC.exe` is available
- writes a release manifest to `release/manifest/release.json`

Release runtime notes:

- the assembled package includes `frontend-dist/` next to `curated.exe`
- the assembled package also copies `backend/third_party/` to `third_party/` next to `curated.exe`
- the backend serves that local frontend bundle directly when present
- when `third_party/ffmpeg/bin/ffmpeg.exe` is present, bundled FFmpeg is preferred automatically for HLS stream push while the default `player.ffmpegCommand` remains `ffmpeg`
- Windows release binaries default to tray mode, so launching `curated.exe` starts the background service and tray shell

Individual scripts:

```powershell
scripts/release/build-frontend.ps1
scripts/release/build-backend.ps1
scripts/release/assemble-release.ps1
scripts/release/package-portable.ps1
scripts/release/package-installer.ps1
scripts/release/publish.ps1
```

Installer notes:

- The installer path uses an Inno Setup template at `scripts/release/windows/Curated.iss.tpl`.
- If `ISCC.exe` is not installed, `package-installer.ps1` will generate `release/installer/Curated.iss` and stop with a warning instead of failing the whole plan.
- Release binaries must continue to expose version information via `GET /api/health` (`version` + `channel`) so the UI can display a stable release identifier.

## Backend tests

```bash
cd backend
go test ./...
```

## Documentation

- [`docs/jav-libary.md`](docs/jav-libary.md) — product design and target architecture
- [`docs/project-memory.md`](docs/project-memory.md) — implementation facts (if present)
- [`docs/architecture-and-implementation.html`](docs/architecture-and-implementation.html) — architecture overview (open in a browser)
- [`docs/library-organize.md`](docs/library-organize.md) — library organization notes
- [`docs/frontend-ui-spec.md`](docs/frontend-ui-spec.md) — UI tokens and patterns

## Notes

- **Electron / mpv** are **not** implemented in this repo; they remain design targets.
- **`docs/film-scanner/`** holds reference material and fixtures, not the production module layout (see `.cursor/rules/backend-go-standards.mdc`).
