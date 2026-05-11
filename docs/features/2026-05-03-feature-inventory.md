# Curated Feature Inventory

2026-05-03 · Current as of v1.4.2

This document catalogs all features implemented in the current **web-first** phase. Features marked `[Target]` are documented future direction, not shipped.

---

## 1. Library Management

### 1.1 Browsing & Discovery

| Feature | Status | Notes |
|---|---|---|
| Virtualized poster-grid browsing | Shipped | `vue-virtual-scroller` for large libraries |
| URL-backed grid selection | Shipped | Selection survives navigation; works without DOM rendering |
| Search by query | Shipped | `q` parameter on library endpoint |
| Tag-based filtering | Shipped | `tag` query, `userTags` + `metadataTags` |
| Actor-based filtering | Shipped | `actor` query with `ActorProfileCard` overlay |
| Favorites view | Shipped | `favorites` route, toggle via `PATCH` |
| Recent additions view | Shipped | `recent` route |
| Tags browse view | Shipped | `tags` route |
| Trash view | Shipped | `mode=trash` query, restore or permanent-delete |
| History view | Shipped | Watch history grouped by local calendar date |

### 1.2 Movie Detail

| Feature | Status | Notes |
|---|---|---|
| Full detail page | Shipped | `detail/:id` route |
| User rating (0-5) | Shipped | Persisted via `PATCH /api/library/movies/{id}` |
| Favorite toggle | Shipped | Same endpoint |
| User tags editing | Shipped | `userTags` field |
| Metadata tags | Shipped | `metadataTags` field |
| User comment/notes | Shipped | `GET/PUT /api/library/movies/{id}/comment`; SQLite in Web API, localStorage in Mock |
| Preview stills | Shipped | `GET /api/library/movies/{id}/asset/preview/{index}` |
| Cover/thumbnail assets | Shipped | `GET /api/library/movies/{id}/asset/{kind}` |

### 1.3 Library Organization

| Feature | Status | Notes |
|---|---|---|
| Structured folder organization | Shipped | `organizeLibrary` setting; renames/moves files into structured dirs |
| Library path management | Shipped | Add, edit title, delete, reveal in OS file manager |
| Multi-root library support | Shipped | Multiple library paths under one database |
| Directory watch (fsnotify) | Shipped | `autoLibraryWatch` setting; debounced scan on new files |
| Soft-delete (trash) | Shipped | Sets `trashedAt`; restore or permanent-delete |
| Reveal in file manager | Shipped | Server-side file manager open for movie files and library roots |

### 1.4 Scanning & Metadata

| Feature | Status | Notes |
|---|---|---|
| Manual library scan | Shipped | `POST /api/scans`, task-tracked |
| Auto-scan loop | Shipped | Periodic background scan |
| fsnotify-triggered scan | Shipped | New file detection with debounce |
| Movie metadata scraping | Shipped | Via metatube-sdk-go; `POST /api/library/movies/{id}/scrape` |
| Actor metadata scraping | Shipped | `POST /api/library/actors/scrape` |
| Auto actor profile scrape | Shipped | `autoActorProfileScrape` setting; enqueues on movie scrape success |
| Batch metadata refresh | Shipped | `POST /api/library/metadata-scrape` by library paths |
| Multi-provider support | Shipped | `metadataMovieProvider` + `metadataMovieStrategy` |
| Provider strategies | Shipped | `auto-global`, `auto-cn-friendly`, `custom-chain`, `specified` |
| Provider health checks | Shipped | `POST /api/providers/ping` + `/ping-all` |
| Failure categories | Shipped | Machine-readable error categories for network troubleshooting |

---

## 2. Playback

### 2.1 Core Playback

| Feature | Status | Notes |
|---|---|---|
| HTML5 video playback | Shipped | `<video>` with HTTP Range streaming |
| Resume playback | Shipped | Persisted progress; SQLite in Web API, localStorage in Mock |
| Playback descriptor seam | Shipped | `GET /api/library/movies/{id}/playback` returns structured descriptor |
| Direct-play path | Shipped | Raw stream URL for browser-compatible content |
| HLS session support | Shipped | Remux-first; transcode fallback; `POST /api/library/movies/{id}/playback-session` |
| Playback session diagnostics | Shipped | `GET /api/playback/sessions/recent` + `/{id}` |
| HLS segment serving | Shipped | `GET /api/playback/sessions/{id}/hls/{file}` |

### 2.2 External Player

| Feature | Status | Notes |
|---|---|---|
| PotPlayer protocol handoff | Shipped | Configurable browser protocol template (`potplayer:{url}`) |
| Legacy native-play hook | Shipped | `POST /api/library/movies/{id}/native-play` (not default path) |
| Unsafe protocol rejection | Shipped | Player validates launch protocols for safety |

### 2.3 Watch Statistics

| Feature | Status | Notes |
|---|---|---|
| Daily watch-time tracking | Shipped | Bounded deltas via `POST /api/playback/watch-time/daily` |
| Watch-time overview | Shipped | Settings → Overview; 91-day window |
| Played-movie tracking | Shipped | `GET /api/library/played-movies`, `POST .../played-movies/{id}` |
| Movie import insights | Shipped | Import and watch-time statistics in Settings |

### 2.4 Player UI

| Feature | Status | Notes |
|---|---|---|
| Player stats overlay | Shipped | Codec, resolution, bitrate, dropped frames |
| Timeline helpers | Shipped | Preview thumbnails on hover |
| Curated frame capture from player | Shipped | Hotkey/button capture during playback |
| Active playback sidebar return | Shipped | Return to player from sidebar while playing |
| Hidden player action feedback | Shipped | Visual feedback for player actions |
| Route navigation context | Shipped | `?t=` for timestamp, `?from=history` for return path |

---

## 3. Actors

| Feature | Status | Notes |
|---|---|---|
| Actor list browsing | Shipped | `GET /api/library/actors` with `q`, `actorTag`, `sort`, pagination |
| Actor profile detail | Shipped | `GET /api/library/actors/profile` by name |
| User tag editing | Shipped | `PATCH /api/library/actors/tags` |
| External links management | Shipped | `PATCH /api/library/actors/external-links` |
| Avatar caching & delivery | Shipped | `GET /api/library/actors/{name}/asset/avatar` (same-origin) |
| Actor scraping | Shipped | Async task via `POST /api/library/actors/scrape` |
| Actor profile card on library | Shipped | Shown when browsing library with `actor=` filter |

---

## 4. Curated Frames

| Feature | Status | Notes |
|---|---|---|
| Frame capture | Shipped | From player during playback; base64 or multipart upload |
| Frame browsing | Shipped | Paginated grid with `GET /api/curated-frames` |
| Text search | Shipped | `q` parameter |
| Tag filtering | Shipped | `tag` parameter; facet via `/curated-frames/tags` |
| Actor filtering | Shipped | `actor` parameter; facet via `/curated-frames/actors` |
| Movie filtering | Shipped | `movieId` parameter |
| Tag editing | Shipped | `PATCH /api/curated-frames/{id}/tags` |
| Frame deletion | Shipped | `DELETE /api/curated-frames/{id}` |
| Thumbnail delivery | Shipped | `GET /api/curated-frames/{id}/thumbnail` |
| Full image delivery | Shipped | `GET /api/curated-frames/{id}/image` |
| Stats overview | Shipped | `GET /api/curated-frames/stats` |
| JPG export | Shipped | Single/batch; EXIF `UserComment` metadata |
| WebP export | Shipped | Single/batch; EXIF metadata |
| PNG export | Shipped | Single/batch; iTXt metadata |
| ZIP multi-frame export | Shipped | 1-20 frames in archive |
| Embedded export metadata | Shipped | `tags`, `schemaVersion`, `exportedAt`, `appName`, `appVersion` |
| Export format preference | Shipped | `curatedFrameExportFormat` setting (default `jpg`) |
| Near-duplicate handling | Shipped | Allowed on create; reviewed/cleaned in library UI |

---

## 5. Movie Import

| Feature | Status | Notes |
|---|---|---|
| Drag-and-drop import | Shipped | Top-bar action; `POST /api/import/movies` (multipart) |
| File selection import | Shipped | Browser file picker |
| Folder selection import | Shipped | Preserves relative paths via `relativePath` fields |
| Progress tracking | Shipped | `import.movies` task with file-level progress |
| Resumable upload | Shipped | Chunked upload for large files; `POST /api/import/movies/uploads` |
| Upload commit | Shipped | `POST /api/import/movies/uploads/{id}/commit` |
| Upload abort | Shipped | `DELETE /api/import/movies/uploads/{id}` |
| Upload status query | Shipped | `GET /api/import/movies/uploads/{id}` |
| Conflict detection | Shipped | Existing target files are not overwritten |
| Default import path | Shipped | `defaultImportLibraryPathId` setting |
| Staging isolation | Shipped | Files hidden until commit; `.curated-import/` staging dir |

---

## 6. Homepage & Recommendations

| Feature | Status | Notes |
|---|---|---|
| Daily recommendation snapshot | Shipped | UTC day key; persisted in SQLite |
| Hero carousel | Shipped | `heroMovieIds` in snapshot |
| Recommendation rail | Shipped | `recommendationMovieIds` in snapshot |
| Cross-device consistency | Shipped | Same snapshot for all browsers/devices |
| Generation versioning | Shipped | Reuses snapshot only when algorithm version matches |
| Weighted sampling | Shipped | Without replacement; weight based on recency and count |
| Hard cooling | Shipped | Recently scraped movies are temporarily excluded |
| Recovery cooling window | Shipped | 14-day exclusion window with staged fallback (14→10→7→5→3→1→0) |
| Recommendation count decay | Shipped | Logarithmic penalty for repeated recommendations |
| Actor diversity balancing | Shipped | Penalty for reusing same actors in one slate |
| Studio diversity balancing | Shipped | Penalty for reusing same studios in one slate |
| Force-refresh | Shipped | `POST /api/homepage/recommendations/refresh` |
| Hero preservation on refresh | Shipped | `preserveHeroMovieIds` body parameter |
| Recommendation exclusion on refresh | Shipped | `excludeRecommendationMovieIds` body parameter |

---

## 7. Settings & Configuration

### 7.1 Settings UI

| Feature | Status | Notes |
|---|---|---|
| Overview dashboard | Shipped | Stats cards (movies, tags, frames); watch-time summary |
| General settings | Shipped | Library organization, auto-watch, language |
| Video storage | Shipped | Library paths, default import path, auto-scan |
| Metadata settings | Shipped | Provider selection, strategy, provider chain, auto-scrape |
| Network settings | Shipped | Proxy configuration with ping tests |
| Curated frames settings | Shipped | Export format preference |
| About page | Shipped | Version info, update checks, dev tools |
| Maintenance | Shipped | Database path, log settings |

### 7.2 Configuration System

| Feature | Status | Notes |
|---|---|---|
| Library config file | Shipped | `config/library-config.cfg` (JSON) |
| Atomic config writes | Shipped | `PATCH /api/settings` writes atomically |
| Frontend env vars | Shipped | `VITE_USE_WEB_API`, `VITE_API_BASE_URL`, `VITE_LOG_LEVEL` |
| Backend JSON config | Shipped | Main runtime config with library config merge |

### 7.3 App Updates

| Feature | Status | Notes |
|---|---|---|
| Update status check | Shipped | `GET /api/app-update/status`; compares with GitHub Releases |
| Manual update check | Shipped | `POST /api/app-update/check`; bypasses cache |
| Update caching | Shipped | Results cached in SQLite |
| Sidebar update badge | Shipped | Lightweight badge (expanded) or dot (compact) in sidebar |
| Direct installer download | Shipped | `installerDownloadUrl` from release assets; Settings → About download button |
| Release page fallback | Shipped | Falls back to release page URL when no `.exe` asset |

### 7.4 Proxy

| Feature | Status | Notes |
|---|---|---|
| HTTP proxy configuration | Shipped | Persisted via `proxy` setting |
| Environment variable sync | Shipped | `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY` set from config |
| Proxy ping test (JavBus) | Shipped | `POST /api/proxy/ping-javbus` |
| Proxy ping test (Google) | Shipped | `POST /api/proxy/ping-google` |

### 7.5 Logging

| Feature | Status | Notes |
|---|---|---|
| Zap structured logging | Shipped | File + console sinks |
| Configurable log directory | Shipped | `logDir` setting; empty = default |
| Log retention | Shipped | `logMaxAgeDays` setting |
| Log level control | Shipped | `logLevel` setting; Settings UI + config |
| Dev log path | Shipped | `backend/runtime/logs` |
| Release log path | Shipped | `LOCALAPPDATA\Curated\logs` |

---

## 8. Gamepad Controls

| Feature | Status | Notes |
|---|---|---|
| Web Gamepad API support | Shipped | Standard gamepad API; no WebHID/node-hid |
| DualSense standard mapping | Shipped | Recognizes DualSense in standard mode |
| Global focus navigation | Shipped | Navigate between sidebar, content, player |
| Library grid navigation | Shipped | D-pad + stick navigation in virtualized poster grid |
| Player playback controls | Shipped | Play/pause, seek, volume, mute, fullscreen exit |
| Large seek jumps | Shipped | Shoulder-button large jumps |
| Curated frame capture (gamepad) | Shipped | Capture button during playback |
| Stats/chrome toggle | Shipped | Show/hide player overlays |
| Route-back behavior | Shipped | Return from player to previous page |
| Browser-local toggle | Shipped | `localStorage` key `curated-gamepad-controls-v1` |
| Rumble support typing | Shipped | Type-level support for future rumble |

---

## 9. Packaging & Release

| Feature | Status | Notes |
|---|---|---|
| Windows release workflow | Shipped | `pnpm release:publish` via Python CLI |
| Version management | Shipped | `scripts/release/version.json` |
| Tray-mode runtime | Shipped | `-mode tray` startup; system tray icon |
| Local frontend hosting | Shipped | Release binary serves `frontend-dist/` on `:8081` |
| Inno Setup installer | Shipped | `.iss` template rendered by Python |
| Portable zip | Shipped | Standalone zip distribution |
| FFmpeg bundling | Shipped | Bundled in `third_party/ffmpeg/bin/` |
| Release manifest | Shipped | Generated with each build |
| Package build history | Shipped | `docs/ops/package-build-history.csv` (UTF-8 BOM) |
| Windows login autostart | Shipped | `launchAtLogin` setting; silent tray on autostart |
| Dev binary naming | Shipped | `curated-dev.exe` (dev) vs `curated.exe` (release) |

---

## 10. Frontend Architecture

| Feature | Status | Notes |
|---|---|---|
| Vue 3 + Composition API | Shipped | Full SPA |
| TypeScript | Shipped | Strict mode |
| Vite 8 build | Shipped | Dev server with HMR |
| Tailwind CSS v4 | Shipped | Utility-first CSS |
| shadcn-vue UI kit | Shipped | Accessible component primitives |
| Service layer + adapter pattern | Shipped | `WebAdapter` / `MockAdapter` behind `LibraryService` contract |
| i18n (3 languages) | Shipped | `en`, `ja`, `zh-CN` via `vue-i18n` |
| Virtual scrolling | Shipped | `vue-virtual-scroller` for poster grids |
| Toast notifications | Shipped | `vue-sonner` via `pushAppToast()` |
| Router-based navigation | Shipped | 8 routes: library, favorites, recent, tags, actors, history, detail, player, settings |
| Error boundary | Shipped | Root-level error boundary |
| Dev performance monitor | Shipped | Fixed bottom overlay bar; dev-only |
| Loading state management | Shipped | Shallow refs for large state |
| Library first-load caching | Shipped | Performance optimization |

---

## 11. Backend Architecture

| Feature | Status | Notes |
|---|---|---|
| Go HTTP server | Shipped | `net/http` with middleware |
| SQLite (modernc) | Shipped | Pure-Go SQLite; no CGO |
| Clean architecture | Shipped | Repository pattern; contracts layer |
| Async task system | Shipped | `pending → running → completed/partial_failed/failed/cancelled` |
| Task types | Shipped | `scan.library`, `scrape.movie`, `scrape.actor`, `import.movies` |
| Task polling API | Shipped | `GET /api/tasks/{taskId}` + `GET /api/tasks/recent` |
| Database migrations | Shipped | Auto-run on startup |
| Structured error codes | Shipped | `COMMON_*`, `LIBRARY_*`, `SCAN_*`, `SCRAPER_*`, `PLAYER_*`, `SETTINGS_*`, `CURATED_*`, `PROVIDER_*` |
| HTTP client timeout | Shipped | Request-level timeout |
| Graceful shutdown | Shipped | Barrier for in-flight scrape goroutines |
| Scanner cancellation | Shipped | `filepath.Walk` respects context cancellation |
| Path leak prevention | Shipped | Validates paths in reveal error messages |
| Playback validation | Shipped | Progress cache validation |

---

## 12. Security

| Feature | Status | Notes |
|---|---|---|
| Unsafe protocol rejection | Shipped | Player launch URL validation |
| Path traversal prevention | Shipped | Reveal endpoint validates paths |
| Filesystem path sanitization | Shipped | Error messages don't leak server paths |
| Client request timeout | Shipped | Prevents hanging connections |

---

## 13. Current Architecture Phase

**Web-first** — Vue SPA + Go HTTP API + SQLite.

### Shipped
- Go HTTP backend with SQLite
- File scanning, metadata scraping, task system
- REST API at `/api`
- Frontend connects via HTTP (`VITE_USE_WEB_API=true`)
- HTML5 `<video>` with HTTP Range streaming
- Web Gamepad API controls
- Trash/restore workflow
- HLS playback sessions
- Curated frames with export
- Windows release packaging with tray mode

### Target (not yet implemented)
- Electron shell, preload script, main process IPC
- mpv player integration with named pipes
- Desktop file system bridge
- WebHID / node-hid controller depth (touchpad gestures, adaptive triggers, LED)
- Multi-device playback progress sync
- Real-time event push (WebSocket/SSE)

---

## 14. Routes Reference

| Route | Page | Notes |
|---|---|---|
| `/library` | Library | Default landing; query: `q`, `tag`, `actor`, `tab` |
| `/favorites` | Favorites | Filtered library view |
| `/recent` | Recent | Recently added |
| `/tags` | Tags | Tag browse |
| `/actors` | Actors | Actor list; query: `q`, `actorTag`, `sort` |
| `/history` | History | Watch history by date |
| `/detail/:id` | Movie Detail | Full metadata, comments, previews |
| `/player/:id` | Player | Query: `?t=`, `?from=history` |
| `/settings` | Settings | Overview, General, Library, Metadata, Network, Curated, About, Maintenance |
