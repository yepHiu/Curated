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

Curated is a local-first media library application built with a Vue 3 frontend and a Go + SQLite backend. The current repository ships a web-first architecture with Windows-friendly release packaging, tray-mode runtime support, metadata scraping, playback workflows, and curated-frame management.

The product name is **Curated**. The repository folder and npm package may still use **`jav-shadcn`**. The Go module is **`curated-backend`** and the server entrypoint is **`backend/cmd/curated`**.

## Highlights

- Local-first architecture with a Vue SPA frontend and a Go HTTP API backend.
- Real API mode and mock mode for fast UI iteration.
- SQLite-backed persistence for library data, playback progress, comments, ratings, and curated frames.
- UTC-based homepage daily recommendations in Web API mode, persisted in SQLite so the hero carousel and today's picks stay identical across browsers and devices, with recent-history exposure penalties plus actor and studio diversity balancing to reduce repeated titles across days.
- Packaged-app update checks in Settings -> About, backed by GitHub Releases with a lightweight sidebar badge when a newer installer is available.
- Windows release flow with tray-mode startup, local web serving, and installer packaging.
- Actor metadata, curated-frame export, and playback-session diagnostics already integrated into the current web phase.

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
- The Vite dev server proxies `/api` to `http://localhost:8080`.

## Features

### Library

- Virtualized poster-grid browsing for large libraries.
- Favorites, ratings, tags, and library organization controls.
- Real backend mode and mock adapter mode behind the same frontend service layer.
- Homepage hero and "today's recommendations" can be sourced from a backend-generated daily snapshot that rolls over automatically on the UTC day boundary and tries to avoid same-day actor/studio clustering.

### Playback

- Resume playback support with persisted progress in Web API mode.
- Browser playback, external player handoff, and HLS session support in the current playback pipeline.
- Session diagnostics and richer playback decision metadata for direct play, remux, and transcode paths.

### Actors

- Actor browsing, profile loading, and user-tag editing.
- Same-origin actor avatar delivery through backend-managed caching.

### Curated Frames

- Frame capture, browsing, tagging, filtering, and export workflows.
- WebP / PNG export with embedded metadata.

### Packaging

- Windows-oriented release workflow.
- Tray-mode release runtime with local frontend hosting beside the backend executable.
- Settings -> About can compare the current packaged installer version with the latest GitHub Release and open the official release page for manual upgrade.

## Configuration

Runtime configuration is split between frontend environment variables and backend settings.

### Frontend

- `VITE_USE_WEB_API=true`: use the real backend
- `VITE_API_BASE_URL`: override the API base URL
- `VITE_LOG_LEVEL`: optional browser log level default

### Backend

The backend reads its main runtime config from JSON and merges library-level settings from:

- `config/library-config.cfg`

Common library-level settings include:

- `organizeLibrary`
- `metadataMovieProvider`
- `metadataMovieStrategy`
- `autoLibraryWatch`
- `autoActorProfileScrape`
- `launchAtLogin`
- `curatedFrameExportFormat` (default `jpg`; accepted values: `jpg`, `webp`, `png`)
- `proxy`
- backend log directory and retention settings
  Empty `logDir` means "use the default log directory" rather than disabling file logging:
  release builds use `LOCALAPPDATA\\Curated\\logs`, while dev builds use `backend/runtime/logs`.

Release builds default to port `:8081` unless overridden by config.

## API

Curated exposes a Go HTTP API for library, playback, actor, settings, and curated-frame workflows.

See [API.md](API.md) for the full endpoint reference.

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
- The current base line is `1.1.0`.
- `pnpm release:*` is now backed by `python scripts/release/release_cli.py`.
- Release packaging assembles a Windows-oriented staging directory, portable zip, installer executable, and release manifest.
- The package build ledger now lives in `docs/ops/package-build-history.csv` and is written in UTF-8 with BOM for Excel / WPS compatibility.
- Windows release binaries default to tray mode and can host the built frontend locally when `frontend-dist/` is present beside the executable.
- The installer still uses Inno Setup under Python orchestration; `scripts/release/windows/Curated.iss.tpl` remains the template source.
- Settings can persist Windows login autostart for the current user; autostart launches Curated silently in tray mode without opening the browser on that login-triggered run.

Additional release references:

- [docs/plan/2026-03-31-production-packaging-and-config-strategy.md](docs/plan/2026-03-31-production-packaging-and-config-strategy.md)
- [docs/ops/package-build-history.csv](docs/ops/package-build-history.csv)
- [docs/ops/2026-04-02-package-build-history.md](docs/ops/2026-04-02-package-build-history.md)

## Documentation

- [API.md](API.md): public HTTP API reference
- [docs/product/2026-03-20-jav-libary.md](docs/product/2026-03-20-jav-libary.md): product design and target architecture
- [docs/reference/2026-03-20-project-memory.md](docs/reference/2026-03-20-project-memory.md): implementation facts and stable project memory
- [docs/reference/architecture-and-implementation.html](docs/reference/architecture-and-implementation.html): architecture overview
- [docs/reference/2026-03-21-library-organize.md](docs/reference/2026-03-21-library-organize.md): library organization notes
- [docs/reference/2026-03-24-frontend-ui-spec.md](docs/reference/2026-03-24-frontend-ui-spec.md): UI tokens and frontend patterns

## Notes

- The current repository is in the **web-first** implementation phase.
- Electron and mpv remain target-direction items, not shipped features in this repository.
- `docs/film-scanner/` contains reference material and fixtures rather than the production module layout.
