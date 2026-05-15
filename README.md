<p align="center">
  <img src="icon/curated-title-nobg.png" alt="Curated" width="520" />
</p>

<p align="center">
  English | <a href="README.zh-CN.md">简体中文</a> | <a href="README.ja-JP.md">日本語</a>
</p>

<p align="center">
  <a href="https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vuedotjs&logoColor=white"><img alt="Vue 3" src="https://img.shields.io/badge/Vue-3-42b883?style=flat-square&logo=vuedotjs&logoColor=white"></a>
  <a href="https://img.shields.io/badge/TypeScript-5.x-3178c6?style=flat-square&logo=typescript&logoColor=white"><img alt="TypeScript 5.x" src="https://img.shields.io/badge/TypeScript-5.x-3178c6?style=flat-square&logo=typescript&logoColor=white"></a>
  <a href="https://img.shields.io/badge/Vite-8.x-646cff?style=flat-square&logo=vite&logoColor=white"><img alt="Vite 8.x" src="https://img.shields.io/badge/Vite-8.x-646cff?style=flat-square&logo=vite&logoColor=white"></a>
  <a href="https://img.shields.io/badge/Go-1.25+-00add8?style=flat-square&logo=go&logoColor=white"><img alt="Go 1.25+" src="https://img.shields.io/badge/Go-1.25+-00add8?style=flat-square&logo=go&logoColor=white"></a>
  <a href="https://img.shields.io/badge/SQLite-modernc-003b57?style=flat-square&logo=sqlite&logoColor=white"><img alt="SQLite modernc" src="https://img.shields.io/badge/SQLite-modernc-003b57?style=flat-square&logo=sqlite&logoColor=white"></a>
  <a href="https://img.shields.io/badge/Tailwind_CSS-v4-06b6d4?style=flat-square&logo=tailwindcss&logoColor=white"><img alt="Tailwind CSS v4" src="https://img.shields.io/badge/Tailwind_CSS-v4-06b6d4?style=flat-square&logo=tailwindcss&logoColor=white"></a>
  <a href="https://img.shields.io/badge/shadcn--vue-ui-111111?style=flat-square"><img alt="shadcn-vue" src="https://img.shields.io/badge/shadcn--vue-ui-111111?style=flat-square"></a>
  <a href="https://img.shields.io/badge/Windows-tray_ready-0078d4?style=flat-square&logo=windows&logoColor=white"><img alt="Windows tray ready" src="https://img.shields.io/badge/Windows-tray_ready-0078d4?style=flat-square&logo=windows&logoColor=white"></a>
</p>

# Curated

Curated is a local-first media library application built with a Vue 3 frontend and a Go + SQLite backend. The current repository ships a web-first architecture with Windows-friendly release packaging, tray-mode runtime support, an Electron desktop-shell MVP, metadata scraping, playback workflows, curated-frame management, gamepad controls, and a comprehensive settings system.

See [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md) for the full catalog of implemented features.

The product name is **Curated**. The repository folder and npm package may still use **`jav-shadcn`**. The Go module is **`curated-backend`** and the server entrypoint is **`backend/cmd/curated`**.

## Highlights

- **Local-first** — Vue 3 SPA frontend + Go HTTP API backend + SQLite persistence.
- **Dual-mode development** — Real API mode (full backend) and mock mode (fast UI iteration) behind the same service layer.
- **Comprehensive library management** — Virtualized poster grid, favorites, ratings, tags, actor profiles, trash/restore, movie comments, and multi-root library paths with fsnotify-based auto-scan.
- **Movie import** — Drag-and-drop, file selection, or folder selection with progress tracking and resumable chunked upload for large files.
- **Storage presence checks** — Windows-first detection for configured library roots backed by external drives, with startup alerts, notification-center entries, scan/import blocking, and manual rebind when a volume changes.
- **Metadata scraping** — Multi-provider support with configurable strategies, provider health checks, and machine-readable failure categories for network troubleshooting.
- **Playback** — HTML5 video with Range streaming, resume playback, daily watch-time statistics, HLS session support with remux/transcode pipeline, external player handoff, and playback session diagnostics.
- **Homepage daily recommendations** — UTC-based hero carousel and recommendation rail persisted in SQLite for cross-device consistency, with weighted sampling, cooling windows, and actor/studio diversity balancing.
- **Curated frames** — Frame capture, browsing, tagging, filtering, and multi-format export (JPG/WebP/PNG) with embedded metadata.
- **Actor management** — Actor browsing, profile detail, user tags, external links, same-origin avatar caching, and async metadata scraping.
- **PIN App Lock** - Optional Web API app lock with Argon2id-hashed PIN storage, PIN-length metadata for the keyboard-first lock screen, HTTP-only unlock sessions, idle-timeout locking, PIN change controls, and an opt-in trusted-device mode that can stay unlocked until explicitly locked.
- **Gamepad controls** — Web Gamepad API support for standard controllers including DualSense: global focus navigation, library-grid selection, and player playback controls.
- **Windows release packaging** — Electron desktop app as the installed entrypoint, Inno Setup installer, portable zip, FFmpeg bundling, release manifest generation, Windows login autostart, and GitHub Releases-based update checks with in-app installer download, SHA256 verification, and explicit installer launch.
- **Electron shell MVP** — In-repo Electron main process that starts or reuses the Go HTTP backend, starts or reuses Vite in development, uses the Curated app icon and tray, hides to tray on window close, loads the existing Web UI, marks backend requests as `Curated Desktop` for connected-client visibility, and exposes only a narrow native directory-picker bridge instead of replacing REST APIs with IPC. Packaged releases install `Curated.exe` as the Electron shell and place the Go backend at `resources/app/curated.exe`.
- **Settings & configuration** — Full settings UI (Overview, General, Video storage, Metadata, Network, Curated frames, About, Maintenance) with library-level config persistence, proxy support, connected-client visibility, and logging controls.

## Quick Start

### Requirements

- **Node.js**: current LTS compatible with Vite 8
- **pnpm**: required; this repository uses `pnpm-lock.yaml`
- **Go**: `1.25.4+`

### Start The Backend

```bash
cd backend
go run ./cmd/curated
```

Development defaults:

- HTTP address: `:8080`
- health name: `curated-dev`

Windows development helper:

```bash
pnpm backend:build:dev
```

This produces `backend/runtime/curated-dev.exe`.

### Start The Frontend

```bash
pnpm install
pnpm dev
```

The Vite development server usually runs on `http://localhost:5173`.

### Real API vs Mock Mode

- Set `VITE_USE_WEB_API=true` in the repository root `.env` to use the real backend API.
- Any other value keeps the frontend in mock mode.
- In local loopback Web API development, the frontend connects directly to `http://127.0.0.1:8080` for API calls; the Vite `/api` proxy remains available for fallback and non-loopback development.

### Start The Electron Shell MVP

```powershell
pnpm dev:electron
```

The Electron shell builds `backend/runtime/curated-dev.exe`, compiles `electron-dist/`, starts or reuses the Go backend in `-mode http`, waits for `/api/health`, then starts or reuses the Vite frontend at `http://127.0.0.1:5173` and opens that URL in a secure BrowserWindow with the Curated app icon. The Vite renderer is launched with `VITE_USE_WEB_API=true` and points API calls at the Electron-managed backend; in packaged builds, the installed `Curated.exe` is the Electron shell, the bundled Go backend lives at `resources/app/curated.exe`, and Electron loads the backend-hosted static UI on `http://127.0.0.1:8081`. Electron marks backend requests with `X-Curated-Client: desktop-electron`, the app version, and desktop OS headers so the backend reports the client as `Curated Desktop` instead of plain Chrome and can display Windows 11 instead of Chromium's legacy `Windows NT 10.0` token. Closing the window hides it to the tray so the backend and Web entry keep running; use the tray menu to reopen Curated, open the Web UI in a browser, open Settings, or quit the app. Business APIs remain HTTP; preload exposes only `window.javLibrary.pickDirectory()` so existing folder-picking flows can use Electron's native directory dialog.

## Features

### Library

- Virtualized poster-grid browsing for large libraries with URL-backed selection.
- Standard gamepad navigation for the virtualized poster grid.
- Favorites, ratings (0-5), user tags, and metadata tags.
- Multi-root library paths: add, edit, delete, and reveal in OS file manager.
- Library organization with structured folder naming (`organizeLibrary` setting).
- Trash/restore workflow: soft-delete, restore, or permanent-delete.
- Movie comments/notes persisted per movie.
- Actor profile card overlay when browsing by actor.
- Search by query, actor, or tag.

### Scanning & Metadata

- Manual and auto-scan with background task tracking.
- fsnotify-based directory watch with debounced auto-scan (`autoLibraryWatch`).
- Movie metadata scraping via metatube-sdk-go with async task execution.
- Multiple metadata providers with configurable strategies: `auto-global`, `auto-cn-friendly`, `custom-chain`, `specified`.
- Provider health checks (ping single / ping all) with failure categories.
- Auto actor profile scrape on successful movie scrape (`autoActorProfileScrape`).

### Import

- Top-bar movie import via drag-and-drop, file selection, or folder selection.
- Progress tracking with per-file status and failure notifications.
- Resumable chunked upload for large files with commit/abort lifecycle.
- Conflict detection (existing target files are not overwritten).
- Configurable default import library path.
- Imports are blocked with a storage warning when the default target drive is offline or no longer matches the bound volume.

### Playback

- HTML5 video playback with HTTP Range streaming.
- Resume playback with persisted progress (SQLite in Web API mode, localStorage in mock mode).
- Playback descriptor seam for direct-play, remux, and transcode paths.
- HLS session support with session diagnostics and recent-session listing.
- External player handoff via configurable browser protocol template (PotPlayer preset).
- Daily watch-time statistics in Settings → Overview (91-day window).
- Player stats overlay, preview timeline thumbnails, and curated-frame capture.
- Route navigation context: timestamp (`?t=`) and return path (`?from=history`).
- Active playback sidebar return.

### Actors

- Actor browsing with search, tag filter, sort, and pagination.
- Actor profile detail with metadata display.
- User tag editing and external links management.
- Same-origin actor avatar delivery through backend-managed caching.
- Actor metadata scraping as async task.

### Curated Frames

- Frame capture from player during playback.
- Browsing with pagination, text search, and filtering by tag, actor, or movie.
- Tag editing and frame deletion.
- Stats overview, tag facets, and actor facets.
- Export in JPG (EXIF), WebP (EXIF), PNG (iTXt), or ZIP with embedded metadata (tags, schemaVersion, exportedAt, appName, appVersion).
- Configurable export format preference (`curatedFrameExportFormat`).

### Homepage & Recommendations

- UTC-based daily recommendation snapshot persisted in SQLite.
- Hero carousel and recommendation rail with cross-device consistency.
- Weighted sampling without replacement with cooling windows and count decay.
- Actor and studio diversity balancing.
- Force-refresh with hero preservation and recommendation exclusion.

### Security

- Optional PIN App Lock in Settings -> Security.
- PIN values are stored as Argon2id salted hashes in SQLite; plaintext PIN values are never persisted. The configured PIN length is stored separately so the lock screen renders the correct number of PIN cells.
- Unlock sessions are server-side and carried by the browser through an HTTP-only `curated_auth` cookie.
- Users can choose an idle-lock delay for regular devices. UI activity and protected API use extend the idle deadline, so Curated locks after inactivity rather than on a fixed countdown.
- After one successful unlock, users can trust the current device indefinitely until it is explicitly locked or the session is revoked.
- Settings -> Security opens PIN setup and PIN change in shadcn-vue dialogs from entry buttons; changing the configured PIN requires the current PIN plus the new PIN confirmation.
- The lock screen uses a compact shadcn-vue card with PIN cells based on the configured PIN length and keyboard input only; it does not show library artwork or other sensitive media context.
- In Web API mode, locked requests to protected `/api/*` endpoints return `423 AUTH_LOCKED`; mock mode keeps PIN disabled for fast UI iteration.

### Settings & Configuration

- Full settings UI: Overview, General, Security, Video storage, Metadata, Network, Curated frames, About, Maintenance.
- Library-level config persisted to `config/library-config.cfg` with atomic writes.
- Proxy configuration with JavBus and Google ping tests.
- Backend logging: configurable directory, retention, and level.
- App update checks against GitHub Releases with sidebar badge, in-app installer download, SHA256 verification, an opt-in General-settings auto-download toggle, and explicit user-confirmed installer launch.
- Windows login autostart (`launchAtLogin`).

### Gamepad Controls

- Web Gamepad API support for standard controllers including DualSense.
- Global focus navigation, library-grid selection, and player playback controls.
- Large seek jumps, curated-frame capture, and stats/chrome toggle.
- Browser-local settings toggle persisted in localStorage.

### Packaging & Release

- Windows release workflow: `pnpm release:publish` via Python CLI.
- Installed release entrypoint: `Curated.exe` is the Electron desktop shell; the release Go backend is bundled as `resources/app/curated.exe` and started with `-mode http` when Electron owns it.
- Tray-mode runtime with local frontend hosting on `:8081`.
- Inno Setup installer and portable zip distribution.
- FFmpeg bundling and release manifest generation.
- Package build history ledger (`docs/ops/package-build-history.csv`).

### Developer Experience

- Dual-mode development: real API mode and mock mode for fast UI iteration.
- Frontend: Vue 3 + TypeScript + Vite 8 + Tailwind CSS v4 + shadcn-vue.
- Backend: Go 1.25+ + SQLite (modernc) + Zap logging + clean architecture.
- i18n: English, 简体中文, 日本語 via vue-i18n.
- Dev performance monitor bar (dev builds only).
- Error boundary and client request timeout.
- Structured error codes across all backend domains.

## Configuration

Runtime configuration is split between frontend environment variables and backend settings.

### Frontend

- `VITE_USE_WEB_API=true`: use the real backend
- `VITE_API_BASE_URL`: override the API base URL; when unset, local loopback Web API development connects directly to dev backend `:8080` to avoid proxying large uploads through Vite, while release hosting on `:8081` and other modes use same-origin `/api`
- `VITE_LOG_LEVEL`: optional browser log level default

Gamepad controls are a browser-local preference saved in `localStorage` under `curated-gamepad-controls-v1`. They use the Web Gamepad API only; WebHID, node-hid, adaptive triggers, LED control, and Electron main-process controller integration remain future target-direction items.

### Backend

The backend reads its main runtime config from JSON and merges library-level settings from:

- `config/library-config.cfg`

Common library-level settings include:

- `organizeLibrary`
- `metadataMovieProvider`
- `metadataMovieStrategy`
- `defaultImportLibraryPathId`
- `autoLibraryWatch`
- `autoActorProfileScrape`
- `autoDownloadUpdates`
- `launchAtLogin`
- `curatedFrameExportFormat` (default `jpg`; accepted values: `jpg`, `webp`, `png`)
- `proxy`
- backend log directory and retention settings
  Empty `logDir` means "use the default log directory" rather than disabling file logging:
  release builds use `LOCALAPPDATA\\Curated\\logs`, while dev builds use `backend/runtime/logs`.

Release builds default to port `:8081` unless overridden by config. The bundled frontend also uses same-origin `/api` by default, so LAN clients opening `http://<host-ip>:8081` call the backend on that same host and port.

## API

Curated exposes a Go HTTP API for authentication/PIN App Lock, library, playback, actor, settings, connected-client visibility, storage presence, and curated-frame workflows.

See [API.md](API.md) for the full endpoint reference.

Movie import uses browser upload via `POST /api/import/movies` for drag/drop, file selection, and folder selection. Large uploads use resumable session endpoints under `/api/import/movies/uploads`, staging bytes under the target library root before commit. Imports use `defaultImportLibraryPathId` as the target and report progress through `import.movies` tasks.

Library storage presence uses endpoints under `/api/library/paths/storage-status` to detect offline or mismatched backing volumes. The current implementation is Windows-first; macOS and Linux use a fallback path probe and remain future adaptation targets.

## Repository Layout

```text
.
├── src/                    # Vue SPA: views, UI, domain components, API client, adapters
├── backend/
│   ├── cmd/curated/        # Backend entrypoint
│   └── internal/           # App, config, storage, server, scanner, scraper, tasks, desktop
├── config/                 # Library-level runtime config
├── docs/                   # See docs/README.md: reference, product, ops, plan, prd, release-notes
├── icon/                   # Brand design source assets
└── package.json            # pnpm scripts and dependencies
```

Root directory policy notes:

- `videos_test/` stays at the repository root as a fixed local test-fixture directory.
- `config/` stays at the repository root for library-level runtime config; do not merge it into `backend/internal/config`.
- `backend/runtime/` is the allowed dev-runtime output area.
- New local-only scratch state should prefer `.workspace/`.
- Go build caches should not be created inside the repository; release tooling now uses system temporary directories for backend build cache paths.

## Release And Packaging

Recommended release entrypoint:

```powershell
pnpm release:publish
```

Key notes:

- Production package versioning is owned by `scripts/release/version.json`.
- The current base line is `1.4.7`.
- `pnpm release:*` is now backed by `python scripts/release/release_cli.py`.
- `pnpm release:publish` builds the Vue frontend, the release Go backend, and the Electron main process before assembling artifacts.
- Release packaging assembles a Windows-oriented Electron staging directory, portable zip, installer executable, and release manifest.
- The assembled app copies the Electron runtime to `release/Curated`, renames `electron.exe` to `Curated.exe`, writes `resources/app/package.json`, places `electron-dist/` and `frontend-dist/` under `resources/app/`, and bundles the Go backend as `resources/app/curated.exe`.
- Release packaging bundles FFmpeg into `resources/app/third_party/ffmpeg/bin/`: it first uses `backend/third_party/ffmpeg/bin/`, then falls back to a real local FFmpeg installation discovered from Scoop or PATH, and fails fast if no runtime is available.
- The package build ledger now lives in `docs/ops/package-build-history.csv` and is written in UTF-8 with BOM for Excel / WPS compatibility.
- The installer still uses Inno Setup under Python orchestration; `scripts/release/windows/Curated.iss.tpl` remains the template source and launches `{app}\Curated.exe`.
- Settings can persist Windows login autostart for the current user; autostart launches Curated silently in tray mode without opening the browser on that login-triggered run.

Additional release references:

- [docs/plan/2026-03-31-production-packaging-and-config-strategy.md](docs/plan/2026-03-31-production-packaging-and-config-strategy.md)
- [docs/ops/package-build-history.csv](docs/ops/package-build-history.csv)
- [docs/ops/2026-04-02-package-build-history.md](docs/ops/2026-04-02-package-build-history.md)

## Documentation

- [API.md](API.md): public HTTP API reference
- [docs/features/2026-05-03-feature-inventory.md](docs/features/2026-05-03-feature-inventory.md): comprehensive feature catalog (all implemented features)
- [docs/product/2026-03-20-jav-libary.md](docs/product/2026-03-20-jav-libary.md): product design and target architecture
- [docs/reference/2026-03-20-project-memory.md](docs/reference/2026-03-20-project-memory.md): implementation facts and stable project memory
- [docs/reference/architecture-and-implementation.html](docs/reference/architecture-and-implementation.html): architecture overview
- [docs/reference/2026-03-21-library-organize.md](docs/reference/2026-03-21-library-organize.md): library organization notes
- [docs/reference/2026-03-24-frontend-ui-spec.md](docs/reference/2026-03-24-frontend-ui-spec.md): UI tokens and frontend patterns

## Notes

- The current repository is in the **web-first** implementation phase.
- Electron currently exists as a minimal desktop shell under `electron/`; it has tray lifecycle management and a narrow native directory-picker preload bridge, while deeper IPC bridges, mpv/process control, broader native file bridges, and controller hardware integrations remain future target-direction items.
- `docs/film-scanner/` contains reference material and fixtures rather than the production module layout.
