# Curated Feature Inventory

Updated 2026-07-21 · Current repository state

This document catalogs features implemented in the current **Electron desktop shell + Web UI + Go API** architecture. Features marked `[Target]` are documented future direction, not shipped. Source code and `.cursor/rules/project-facts.mdc` take precedence if this inventory becomes stale.

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
| Advanced library filters | Shipped | Play state, exact local user rating, normalized resolution, and relative import window; canonical URL query |
| Saved Views | Implemented; browser QA pending | Ordered create/apply/update/rename/reorder/delete; SQLite in Web API, isolated versioned localStorage in Mock; transient navigation fields excluded |
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
| Storage presence and binding health | Shipped | Detects offline, missing, permission-denied, and volume-mismatch roots; blocks unsafe scan/import and supports deliberate rebind |
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
| Unified Library Health | Shipped | Read-only `POST /api/library/health/scan`; stable category counts/findings for SQLite, storage, source files, assets, duplicates, metadata attempts, orphan state, and strict import staging; offline roots skip per-file checks |
| Bounded metadata repair | Shipped | Confirmed `POST /api/library/health/repairs`, persisted `library.health.repair` parent plus per-movie `scrape.movie` child results, 100-item backend / 25-item Settings limits, explicit restart interruption |
| Audited targeted cleanup | Shipped | Confirmed `POST /api/library/health/actions`; freshly revalidated findings, whitelisted orphan-state deletion or strict non-symlink upload staging removal, persistent audit evidence, never final movie files |

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
| Canonical actor identities | Implemented; browser QA pending | `actors.normalized_name` plus globally unique aliases; NFKC, Unicode case fold, and whitespace fold shared by lookup, recommendations, Web, and Mock comparison |
| Alias-aware actor lookup | Implemented; browser QA pending | Profile, list search, exact library filter, avatar, tags, links, scraping, metadata ingestion, and old actor routes resolve to the canonical actor |
| Read-only merge preview | Implemented; browser QA pending | Reports movie/tag/link/profile/feedback/frame impact, conflicts, blockers, and a complete-state opaque token without writes |
| Confirmed transactional merge | Implemented; browser QA pending | `confirm:true`, stale-token rejection, explicit profile decisions, one SQLite transaction, zero partial writes, and association dedupe |
| Actor merge audits | Shipped | Queryable immutable snapshots; migrations `0035`/`0036` preserve audit history while allowing later chained canonical merges |
| Mock actor merge persistence | Shipped | Isolated `curated-actor-merges-v1` localStorage stores aliases and audits; generated movie actors canonicalize after reload |

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
| Restart persistence | Shipped | SQLite-backed session/file/chunk ledger restores the original task after backend restart |
| Chunk range ledger | Shipped | Synced non-overlapping chunk ranges are the source of truth for received-byte counters |
| Upload commit | Shipped | `POST /api/import/movies/uploads/{id}/commit` |
| Interrupted commit recovery | Shipped | Reconciles staging/final files and per-file commit markers without overwriting conflicts |
| Upload abort | Shipped | `DELETE /api/import/movies/uploads/{id}` |
| Upload status query | Shipped | `GET /api/import/movies/uploads/{id}` |
| Offline storage deferral | Shipped | `recoveryStatus=unavailable` retains the session until its target storage returns |
| Upload janitor | Shipped | Sliding TTL plus narrowly scoped terminal/expired/orphan staging cleanup |
| Cleanup audits | Shipped | Cleanup attempts and outcomes persist in `movie_import_upload_cleanup_audits` |
| Conflict detection | Shipped | Existing target files are not overwritten |
| Default import path | Shipped | `defaultImportLibraryPathId` setting |
| Staging isolation | Shipped | Files hidden until commit; `.curated-import/` staging dir |

---

## 6. Homepage, Recommendations & Personal Insights

| Feature | Status | Notes |
|---|---|---|
| Daily recommendation snapshot | Shipped | UTC day key; persisted in SQLite |
| Hero carousel | Shipped | `heroMovieIds` in snapshot |
| Recommendation rail | Shipped | Compatibility `recommendationMovieIds` plus same-order explanation items in snapshot |
| Cross-device consistency | Shipped | Same snapshot for all browsers/devices |
| Generation versioning | Shipped | Current `v8`; reuses snapshot only when algorithm version matches and shares canonical actor identity normalization |
| Weighted sampling | Shipped | Without replacement; weight based on recency and count |
| Hard cooling | Shipped | Recently scraped movies are temporarily excluded |
| Recovery cooling window | Shipped | 14-day exclusion window with staged fallback (14→10→7→5→3→1→0) |
| Recommendation count decay | Shipped | Logarithmic penalty for repeated recommendations |
| Actor diversity balancing | Shipped | Penalty for reusing same actors in one slate |
| Studio diversity balancing | Shipped | Penalty for reusing same studios in one slate |
| Force-refresh | Shipped | `POST /api/homepage/recommendations/refresh` |
| Hero preservation on refresh | Shipped | `preserveHeroMovieIds` body parameter |
| Recommendation exclusion on refresh | Shipped | `excludeRecommendationMovieIds` body parameter |
| Explainable recommendation items | Implemented; browser QA pending | Persisted truthful reason codes; renderer only localizes them |
| Movie not-interested feedback | Implemented; browser QA pending | Explicit permanent exclusion until feedback removal |
| Bounded snooze | Implemented; browser QA pending | 1-365 days; expired entries are ignored and cleaned |
| Actor/studio/tag down-ranking | Implemented; browser QA pending | Multiplies matching candidate weight while retaining bounded exploration |
| Feedback undo and management | Implemented; browser QA pending | Current-card undo plus dialog for active feedback removal |
| Web feedback persistence | Shipped | SQLite `homepage_recommendation_feedback`, migrations `0033`/`0034` |
| Mock feedback persistence | Shipped | Isolated `curated-homepage-recommendation-feedback-v1` localStorage |
| Refresh isolation | Shipped | Refresh regenerates recommendations but never creates feedback |
| Lazy Personal Insights route | Implemented; browser QA pending | `/insights` loads separately and is linked from the Yours sidebar group |
| Explicit insight ranges | Shipped | Inclusive local-calendar `30d`, `90d`, `365d`, and `all`, with IANA timezone and returned boundaries |
| Viewing overview aggregate | Shipped | Watch time, distinct started movies, current 90%-progress completed count/rate, current local rating count/average; empty denominators are null |
| Bounded preference breakdowns | Shipped | Stable top 1-25 canonical actor/studio/deduplicated-tag rows with `full-per-entity` attribution |
| Raw-history privacy boundary | Shipped | Web returns only SQLite aggregates; Mock aggregation stays inside the service adapter and does not expose source rows to the page |
| Responsive and localized insights UI | Implemented; browser QA pending | One semantic h1, keyboard radio range selection, loading/error/retry/empty states, stale-response guard, zh-CN/en/ja, 375px-safe grid |

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
| Maintenance | Shipped | Library Health scan/export, bounded metadata repair, confirmed audited cleanup, backup create-and-verify, package verification, restore preflight, audited offline path migration, full scan, and maintenance guidance |
| Security | Shipped | PIN setup/change, idle-lock policy, lock-now, and trusted-session review/revocation |

### 7.2 Configuration System

| Feature | Status | Notes |
|---|---|---|
| Library config file | Shipped | `config/library-config.cfg` (JSON) |
| Atomic config writes | Shipped | `PATCH /api/settings` writes atomically |
| Frontend env vars | Shipped | `VITE_USE_WEB_API`, `VITE_API_BASE_URL`, `VITE_LOG_LEVEL` |
| Backend JSON config | Shipped | Main runtime config with library config merge |

### 7.3 Backup, Recovery & Path Migration

| Feature | Status | Notes |
|---|---|---|
| Consistent SQLite package | Shipped | `VACUUM INTO`; optional `library-config.cfg`; no media or user assets in v1 |
| Manifest and integrity verification | Shipped | Size, SHA-256, declared entries, `quick_check`, `foreign_key_check`, migrations |
| Settings maintenance controls | Shipped | PIN-protected create-and-verify, verify, and restore preflight in Web API mode |
| Library Health workspace | Shipped | Web API-only diagnostics, semantic status/counts, stable finding/path detail, confirmation dialogs, task progress, per-item results; Mock disabled; 44px mobile maintenance actions |
| Offline restore | Shipped | Explicit CLI confirmation, runtime lock, compatibility/capacity preflight, `.pre-restore-*` rollback files |
| No-overwrite destination | Shipped | Atomic hard-link commit or `O_EXCL` fallback for filesystems without hard links |
| Read-only path migration plan | Shipped | Whitelisted columns; segment-aware Windows/UNC/Unix mapping; affected counts, samples, target status, conflicts, errors/warnings, and `canApply` |
| Audited path migration apply | Shipped | Runtime lock, explicit confirmation, automatic verified backup, single-transaction path updates plus audit, pre-commit SQLite checks, and storage-binding reset |
| Cross-platform path mapping | Shipped | Windows case/separator semantics and Windows→Unix mapping; missing/unchecked targets require explicit override while wrong type/errors/conflicts always block |

### 7.4 App Updates

| Feature | Status | Notes |
|---|---|---|
| Update status check | Shipped | `GET /api/app-update/status`; compares with GitHub Releases |
| Manual update check | Shipped | `POST /api/app-update/check`; bypasses cache |
| Update caching | Shipped | Results cached in SQLite |
| Sidebar update badge | Shipped | Lightweight badge (expanded) or dot (compact) in sidebar |
| Direct installer download | Shipped | `installerDownloadUrl` from release assets; Settings → About download button |
| Verified installer staging | Shipped | Downloads installer into the update cache and verifies SHA256 before it becomes installable |
| Explicit installer launch | Shipped | `POST /api/app-update/install`; never silently installs without a user action |
| Opt-in background download | Shipped | `autoDownloadUpdates`; startup may download and verify, but does not auto-install |
| Release page fallback | Shipped | Falls back to release page URL when no `.exe` asset |

### 7.5 Proxy

| Feature | Status | Notes |
|---|---|---|
| HTTP proxy configuration | Shipped | Persisted via `proxy` setting |
| Environment variable sync | Shipped | `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY` set from config |
| Proxy ping test (JavBus) | Shipped | `POST /api/proxy/ping-javbus` |
| Proxy ping test (Google) | Shipped | `POST /api/proxy/ping-google` |

### 7.6 Logging

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
| Electron installed entrypoint | Shipped | Top-level `Curated.exe` is the Electron shell; Go backend is `resources/app/curated.exe` |
| Narrow native directory bridge | Shipped | Preload exposes only `window.javLibrary.pickDirectory()`; business APIs remain HTTP REST |
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
| Router-based navigation | Shipped | 15 named records including home, lock, trash, actor detail, Curated Frames, redirect, and 404 routes |
| Error boundary | Shipped | Root-level error boundary |
| Dev performance monitor | Shipped | Fixed bottom overlay bar; dev-only |
| Loading state management | Shipped | Shallow refs for large state |
| Library first-load caching | Shipped | Performance optimization |
| Active-adapter bootstrap | Shipped | Only the selected Web adapter starts; Mock navigation performs no backend requests |
| Protected-state bootstrap | Shipped | Web mode waits for authoritative auth status and hydrates progress/played state only after unlock |
| Bundle hard budgets | Shipped | `pnpm build` enforces initial/total/named chunk limits and emits `dist/bundle-analysis.json` |

---

## 11. Backend Architecture

| Feature | Status | Notes |
|---|---|---|
| Go HTTP server | Shipped | `net/http` with middleware |
| SQLite (modernc) | Shipped | Pure-Go SQLite; no CGO |
| Clean architecture | Shipped | Repository pattern; contracts layer |
| Async task system | Shipped | `pending → running → completed/partial_failed/failed/cancelled` |
| Task types | Shipped | `scan.library`, `scrape.movie`, `scrape.actor`, `import.movies` |
| SSE task event stream | Shipped | `GET /api/events` sends `task.updated` snapshots and heartbeat; polling remains fallback |
| Task polling API | Shipped | `GET /api/tasks/{taskId}` + `GET /api/tasks/recent` |
| Database migrations | Shipped | Auto-run on startup |
| SQLite referential integrity | Shipped | Every SQLite connection verifies foreign keys; historical orphan rows are quarantined before cleanup |
| Structured error codes | Shipped | `COMMON_*`, `LIBRARY_*`, `ACTOR_MERGE_*`, `SCAN_*`, `SCRAPER_*`, `PLAYER_*`, `SETTINGS_*`, `CURATED_*`, `PROVIDER_*` |
| HTTP client timeout | Shipped | Request-level timeout |
| Graceful shutdown | Shipped | Barrier for in-flight scrape goroutines |
| Scanner cancellation | Shipped | `filepath.Walk` respects context cancellation |
| Path leak prevention | Shipped | Validates paths in reveal error messages |
| Playback validation | Shipped | Progress cache validation |

---

## 12. Security

| Feature | Status | Notes |
|---|---|---|
| Loopback-safe defaults | Shipped | Development and release listeners default to `127.0.0.1`; non-loopback requires explicit `lanEnabled` and an initialized PIN |
| Origin and Host validation | Shipped | Credentialed browser access is restricted to same-origin, loopback development origins, and exact configured origins |
| PIN App Lock | Shipped | Argon2id PIN hash, HTTP-only `curated_auth` cookie, router lock screen, and protected `/api/*` middleware |
| PIN attempt throttling | Shipped | Repeated setup/unlock failures trigger exponential backoff and `429 AUTH_RATE_LIMITED` with `Retry-After` |
| Sliding regular sessions | Shipped | Protected activity refreshes idle expiry; restart-lock behavior is configurable |
| Trusted-forever sessions | Shipped | Safe public IDs support list, single revoke, and revoke-others without exposing bearer tokens |
| Sanitized release configuration | Shipped | Packaging stages tracked `library-config.example.cfg` and rejects sensitive machine-local values |
| Unsafe protocol rejection | Shipped | Player launch URL validation |
| Path traversal prevention | Shipped | Reveal endpoint validates paths |
| Filesystem path sanitization | Shipped | Error messages don't leak server paths |
| Client request timeout | Shipped | Prevents hanging connections |

---

## 13. Current Architecture Phase

**Local-first desktop delivery with shared Web UI** — Electron shell + Vue SPA + Go HTTP API + SQLite. Browser access and Mock mode remain supported, while business APIs deliberately stay on the HTTP service boundary.

### Shipped
- Electron shell, tray lifecycle, installed Electron entrypoint, and narrow directory-picker preload bridge
- Go HTTP backend with SQLite
- File scanning, metadata scraping, task system
- REST API at `/api`
- Frontend connects via HTTP (`VITE_USE_WEB_API=true`)
- Mock mode with no inactive Web adapter startup requests
- PIN lock, throttling, trusted-session management, loopback/LAN guard, and strict browser origin policy
- SSE `task.updated` events with polling fallback
- HTML5 `<video>` with HTTP Range streaming
- Web Gamepad API controls
- Trash/restore workflow
- HLS playback sessions
- Curated frames with export
- Windows release packaging with tray mode
- SQLite-backed playback progress in Web API mode; localStorage fallback in Mock mode
- Storage presence health, connected-client visibility, and verified installer update workflow

### Target (not yet implemented)
- Deep Electron business IPC or a broad native desktop bridge
- mpv player integration with named pipes
- Broader desktop file-system and permission bridge beyond directory picking
- WebHID / node-hid controller depth (touchpad gestures, adaptive triggers, LED)
- Account-based multi-user synchronization beyond the current shared local SQLite library
- Native mobile clients and remote-server pairing

---

## 14. Routes Reference

| Route | Page | Notes |
|---|---|---|
| `/lock` | App Lock | PIN unlock and redirect recovery |
| `/` | Home | Daily recommendation hero and rail |
| `/library` | Library | Default landing; query: `q`, `tag`, `actor`, `tab` |
| `/favorites` | Favorites | Filtered library view |
| `/recent` | Recent redirect | Redirects into the library query model |
| `/tags` | Tags | Tag browse |
| `/trash` | Trash | Restore or permanently delete trashed movies |
| `/actors` | Actors | Actor list; query: `q`, `actorTag`, `sort` |
| `/actors/:actorName` | Actor Detail | Canonical profile and movie results; old alias routes normalize to canonical and expose the merge workbench |
| `/history` | History | Watch history by date |
| `/curated-frames` | Curated Frames | Search, filter, edit, delete, and export captured frames |
| `/detail/:id` | Movie Detail | Full metadata, comments, previews |
| `/player/:id?` | Player | Query: `?t=`, `?from=history` |
| `/settings` | Settings | Overview, General, Library, Metadata, Network, Curated, About, Maintenance |
| `/:pathMatch(.*)*` | Not Found | In-shell 404 state |

---

## 15. Delivery Quality

| Capability | Status | Notes |
|---|---|---|
| Pull-request CI workflow | Shipped | Frontend quality, Go quality, production audit, builds, Python release tests, and runtime e2e jobs |
| Runtime Chromium e2e | Shipped | Covers Mock isolation, locked startup hydration ordering, and 375px mobile controls |
| Mobile touch baseline | Shipped | Primary phone controls use at least 44×44 CSS px targets and the library route has one semantic `h1` |
| Production dependency audit | Shipped | CI blocks high-severity production advisories |
| Display-scaling suite | Manual | Long-running cross-browser/display test remains explicit and is not part of normal CI |
