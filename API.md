# Curated API Reference

## Overview

Curated exposes a Go HTTP API for library browsing, playback workflows, actor metadata, settings, curated-frame management, scans, and task tracking.

This document is the single public API reference for the repository.

Implementation references:

- backend routes: `backend/internal/server/server.go`
- backend DTOs and error contracts: `backend/internal/contracts/contracts.go`
- frontend API types: `src/api/types.ts`

## Base URLs

Common local development URLs:

- frontend dev server: `http://localhost:5173`
- backend dev API base: `http://127.0.0.1:8080/api`
- backend release API base: `http://127.0.0.1:8081/api`

The Vite development server proxies `/api` to `http://127.0.0.1:8080`.

## Conventions

### Transport

- API routes use the `/api` prefix.
- Responses are JSON unless the endpoint explicitly serves media or stream content.

### Async Tasks

Long-running operations such as scans and metadata scraping use task-based async execution.

Typical pattern:

1. create or trigger a task-oriented endpoint
2. receive a task identifier
3. poll the task status endpoint

Related endpoints:

- `GET /api/tasks/recent`
- `GET /api/tasks/{taskId}`

### Pagination

List-style endpoints commonly support pagination fields such as:

- `limit`
- `offset`

Some responses also include total-count metadata.

### Runtime Configuration

Library-level settings are persisted through `config/library-config.cfg`, while broader runtime config can come from backend JSON config files and environment setup.

## Scrape And Provider Notes

- actor avatars can be cached locally and served through the backend instead of relying on direct browser requests to remote image hosts
- movie preview images prefer local cache and can fall back to backend fetch behavior when only source-side URLs are available
- settings support the higher-level `metadataMovieStrategy` in addition to legacy scrape-mode fields
- provider health responses and task payloads can include machine-readable failure categories for troubleshooting

## Health

### `GET /api/health`

Purpose:

- returns backend health and release identity information

Important notes:

- development mode reports the `curated-dev` backend identity
- release mode reports `curated`
- `version` is the backend build identifier / build stamp shown in Settings -> About
- `installerVersion` is an optional installer/package version, embedded into release backend binaries at packaging time
- development runtimes expose `installerVersion: "0.0.0"` as a stable fallback when no packaged version was injected
- release builds should continue exposing stable version and channel information

### `GET /api/dev/performance`

Purpose:

- returns development-only CPU summary information used by the frontend performance monitor bar

Important notes:

- intended for development diagnostics
- not a core product-facing endpoint

## App Updates

### `GET /api/app-update/status`

Purpose:

- return the current app-update comparison result used by Settings -> About and the sidebar brand badge

Important notes:

- the backend compares the current runtime `installerVersion` with the latest GitHub Release for `yepHiu/Curated`
- development runtimes use `0.0.0` as the local installed version so the full update-check path remains testable before packaging
- successful checks are cached in SQLite so routine reads do not hit GitHub on every app start
- response fields include `supported`, `status`, `installedVersion`, optional `latestVersion`, `hasUpdate`, `checkedAt`, `publishedAt`, `releaseName`, `releaseUrl`, `releaseNotesSnippet`, and optional `errorMessage`

### `POST /api/app-update/check`

Purpose:

- force a fresh app-update check against the latest GitHub Release

Important notes:

- bypasses the cached status used by `GET /api/app-update/status`
- returns the same DTO shape as the status endpoint
- used by the manual "Check for updates" action in Settings -> About

## Homepage

### `GET /api/homepage/recommendations`

Purpose:

- return the persisted homepage daily recommendation snapshot used by the homepage hero and today's recommendation rail

Important notes:

- the backend uses the current UTC date as the snapshot key
- the first request for a UTC day generates and persists the snapshot in SQLite; later requests reuse the same result only when `generationVersion` matches the current algorithm
- the snapshot contains `heroMovieIds` and `recommendationMovieIds`, plus `dateUtc`, `generatedAt`, and `generationVersion`
- recommendation memory is persisted separately per movie in `homepage_recommendation_states`, tracking `last_recommended_at`, `recommend_count`, and `skip_until`
- when enough inventory is available, the backend avoids reusing the combined hero/recommendation slate from the last 14 UTC days, then falls back through `14 -> 10 -> 7 -> 5 -> 3 -> 1 -> 0` day exclusion windows only as needed
- generation uses weighted sampling without replacement: hard-cooling movies are skipped, recently recommended movies recover weight over a 14-day cooling window, long-unseen movies gain weight, and repeatedly recommended movies receive a logarithmic count penalty
- the slate builder also applies actor and studio diversity penalties while picking the hero and recommendation set, so the same actors or studios are less likely to dominate one day's slate
- after a new daily slate is persisted, the selected movies' recommendation states are updated so later days have persistent memory beyond the daily snapshot itself

### `POST /api/homepage/recommendations/refresh`

Purpose:

- force-regenerate and overwrite the persisted homepage daily recommendation snapshot for the current UTC day

Important notes:

- intended primarily for development and verification workflows
- returns the same DTO shape as `GET /api/homepage/recommendations`
- uses the same UTC day key and persistence table, but bypasses reuse of the existing snapshot for that day
- accepts an optional JSON body `{ "preserveHeroMovieIds": ["..."] }`; when provided, the backend keeps those IDs as the persisted `heroMovieIds` while regenerating `recommendationMovieIds`
- the homepage "Today's Recommendations" refresh uses `preserveHeroMovieIds` so clicking it refreshes only the recommendation rail and does not change the current hero carousel
- the frontend exposes this through a development-only button in Settings -> About when running in dev mode with `VITE_USE_WEB_API=true`

## Movies

### `GET /api/library/movies`

Purpose:

- list movies in the library

Common filters:

- `q`
- `tag`
- `actor`
- pagination fields

### `GET /api/library/movies/{id}`

Purpose:

- fetch a single movie detail payload

### `PATCH /api/library/movies/{id}`

Purpose:

- update user-facing movie state

Common fields:

- favorite flag
- user rating
- `userTags`
- `metadataTags`

### `DELETE /api/library/movies/{id}`

Purpose:

- remove a movie from the library

### `GET /api/library/movies/{id}/stream`

Purpose:

- stream the primary video file

Important notes:

- supports Range requests

### `GET /api/library/movies/{id}/asset/{kind}`

Purpose:

- fetch movie assets such as cover and thumb variants

### `GET /api/library/movies/{id}/asset/preview/{index}`

Purpose:

- fetch indexed preview stills when available

### `POST /api/library/movies/{id}/scrape`

Purpose:

- trigger metadata re-scrape for one movie

Important notes:

- handled as an async task

### `POST /api/library/movies/{id}/reveal`

Purpose:

- reveal the media file in the server machine's file manager

### `GET /api/library/movies/{id}/comment`

Purpose:

- fetch the saved user comment for one movie

### `PUT /api/library/movies/{id}/comment`

Purpose:

- upsert the saved user comment for one movie

## Library Paths

### `POST /api/library/paths`

Purpose:

- add a configured library root

### `PATCH /api/library/paths/{id}`

Purpose:

- update the display title for one configured library root

### `DELETE /api/library/paths/{id}`

Purpose:

- remove one configured library root from the database configuration

Important notes:

- this unbinds or prunes database records that are no longer covered by any configured root
- it does not delete the actual on-disk library directory

### `POST /api/library/paths/{id}/reveal`

Purpose:

- open the configured library root directory in the server machine's file manager

Important notes:

- the backend validates that the stored path still exists on disk and is a directory
- this opens the folder only; it does not modify library configuration or disk files

## Movie Imports

### `POST /api/import/movies`

Purpose:

- copy browser-uploaded movie files into the configured default library root

Request:

- `multipart/form-data`
- repeated `files` parts contain uploaded movie files
- repeated `relativePath` fields may be supplied before each file to preserve folder-relative names from browser folder selection
- optional `totalBytes` is used for progress metadata

Important notes:

- requires `defaultImportLibraryPathId` to be configured through `GET/PATCH /api/settings`
- only video-like extensions are accepted by the import handler
- imported files are copied into the target library root; source files are not moved or deleted
- existing target files are treated as per-file conflicts and are not overwritten
- the response is a `TaskDTO` with type `import.movies`
- task metadata can include `targetLibraryPathId`, `targetPath`, `stage`, `totalFiles`, `completedFiles`, `failedFiles`, `copiedBytes`, `totalBytes`, `currentFileName`, `errorItems`, and optional `scanTaskId` / `scanError`
- a successful or partially successful copy triggers the existing restricted library scan for the target root

### `POST /api/import/movies/uploads`

Purpose:

- create a resumable movie import upload session for large browser uploads

Request:

- JSON body with `files`
- each file manifest includes `relativePath`, `size`, and optional `lastModified`

Response:

- `uploadId`
- `chunkSize`
- `targetPath`
- per-file `fileId`, `relativePath`, `size`, `bytesReceived`, and `complete`
- an `import.movies` task snapshot

Important notes:

- requires `defaultImportLibraryPathId`
- staging files are created under `<target-library-root>/.curated-import/<uploadId>/`
- the first implementation keeps upload session state in backend memory; interrupted browser uploads can resume while the backend process is still running
- final movie files are not visible in the library until commit succeeds

### `GET /api/import/movies/uploads/{uploadId}`

Purpose:

- return resumable upload status, including per-file received bytes and completion flags

### `PUT /api/import/movies/uploads/{uploadId}/files/{fileId}/chunks/{chunkIndex}`

Purpose:

- upload one raw binary chunk for a resumable movie import file

Request:

- raw `application/octet-stream` body
- required header `X-Curated-Offset`
- optional header `X-Curated-Chunk-Size`

Important notes:

- chunk bytes are written by offset into the file's staging `.part` file
- duplicate chunk indexes with the same range are treated as idempotent status reads
- duplicate chunk indexes with a different range are rejected

### `POST /api/import/movies/uploads/{uploadId}/commit`

Purpose:

- validate that all upload files are complete, flush staging files, commit them without overwriting existing target files, and start the restricted library scan

Response:

- `TaskDTO` with type `import.movies`

### `DELETE /api/import/movies/uploads/{uploadId}`

Purpose:

- abort a resumable upload session and remove its staging directory

## Playback

### `GET /api/library/movies/{id}/playback`

Purpose:

- return the playback descriptor used by the frontend player pipeline

Important notes:

- this is the primary playback planning seam
- current responses are still direct-play oriented in many paths
- playback descriptors may include diagnostics such as session kind and reason codes

### `POST /api/library/movies/{id}/playback-session`

Purpose:

- create an explicit playback session, for example for HLS push workflows

### `GET /api/playback/sessions/recent`

Purpose:

- list active and recently archived playback sessions

### `GET /api/playback/sessions/{id}`

Purpose:

- fetch a playback session status snapshot

### `GET /api/playback/sessions/{id}/hls/{file}`

Purpose:

- serve HLS playlists and segments for pushed playback sessions

### `POST /api/library/movies/{id}/native-play`

Purpose:

- legacy backend-side native-player launch hook

Important notes:

- still present for legacy or native-shell integration
- no longer the default path for the player page's local-player action

### `GET /api/playback/progress`

Purpose:

- list playback progress records

### `PUT /api/playback/progress/{movieId}`

Purpose:

- update playback progress for one movie

### `DELETE /api/playback/progress/{movieId}`

Purpose:

- clear playback progress for one movie

### `GET /api/playback/watch-time/daily`

Purpose:

- list daily watch-time totals for Settings -> Overview statistics

Query:

- `days` (optional): number of local calendar days to include; defaults to the 91-day statistics window and is capped at 91

### `POST /api/playback/watch-time/daily`

Purpose:

- add one bounded watch-time delta for a movie and local day

Body:

- `movieId`: movie id
- `dayKey`: local day key in `YYYY-MM-DD`
- `watchedSec`: positive watch-time delta in seconds; backend rejects unusually large single deltas

### `GET /api/library/played-movies`

Purpose:

- list played-movie statistics or records

### `POST /api/library/played-movies/{id}`

Purpose:

- mark one movie as played

## Actors

### `GET /api/library/actors`

Purpose:

- list actors in the library

Common filters:

- `q`
- `actorTag`
- `sort`
- pagination fields

### `GET /api/library/actors/profile`

Purpose:

- fetch a single actor profile by actor name

### `PATCH /api/library/actors/tags`

Purpose:

- replace user tags for a specific actor

Important notes:

- actor identity is typically passed through the actor name query field

### `PATCH /api/library/actors/external-links`

Purpose:

- replace the full user-managed external link list for a specific actor

Important notes:

- actor identity is passed through the actor name query field
- request body shape is `{ "externalLinks": string[] }`
- links are separate from the scraped `homepage` field and are returned by `GET /api/library/actors/profile`

### `POST /api/library/actors/scrape`

Purpose:

- trigger actor metadata scraping

Important notes:

- handled as an async task

### `GET /api/library/actors/{name}/asset/avatar`

Purpose:

- serve a same-origin cached avatar image for an actor

Important notes:

- backend-owned avatar caching reduces direct browser dependency on remote image hosts

## Settings

### `GET /api/settings`

Purpose:

- return the current effective settings DTO used by the frontend

### `PATCH /api/settings`

Purpose:

- partially update persisted settings

Important notes:

- updates are written back to `config/library-config.cfg` for library-level keys
- `defaultImportLibraryPathId` stores the default target library path used by `POST /api/import/movies`
- playback runtime preferences are also surfaced through this settings contract
- `curatedFrameExportFormat` is a persisted library-level setting; accepted values are `jpg`, `webp`, and `png`, with `jpg` as the default
- `autoActorProfileScrape` is an opt-in library-level setting; when enabled, successful movie metadata scrapes may enqueue missing actor-profile scrape tasks for actors that still lack both avatar and summary
- some backend logging changes require restart before file sinks fully reflect new values

## Curated Frames

### `GET /api/curated-frames`

Purpose:

- query curated frames with filtering and pagination

Common filters:

- `q`
- `actor`
- `movieId`
- `tag`
- `limit`
- `offset`

Important notes:

- responses include pagination metadata such as total count

### `GET /api/curated-frames/stats`

Purpose:

- fetch curated frame aggregate counts

### `GET /api/curated-frames/tags`

Purpose:

- fetch curated frame tag facets

### `GET /api/curated-frames/actors`

Purpose:

- fetch curated frame actor facets

### `POST /api/curated-frames`

Purpose:

- create a curated frame record

Supported request styles:

- legacy JSON with `imageBase64`
- multipart form data with `metadata` and `image`

Important notes:

- near-duplicate captures are allowed
- review and cleanup are expected in the curated-frames library UI

### `GET /api/curated-frames/{id}/image`

Purpose:

- fetch the full curated frame image

### `GET /api/curated-frames/{id}/thumbnail`

Purpose:

- fetch the curated frame thumbnail

### `PATCH /api/curated-frames/{id}/tags`

Purpose:

- update curated frame tags

### `DELETE /api/curated-frames/{id}`

Purpose:

- delete a curated frame

### `POST /api/curated-frames/export`

Purpose:

- export curated frames as WebP, PNG, or ZIP bundles

Important notes:

- current export range is 1 to 20 frames
- exported metadata includes fields such as `tags`, `schemaVersion`, `exportedAt`, `appName`, and `appVersion`

## Scans And Tasks

### `POST /api/scans`

Purpose:

- start a library scan

### `GET /api/tasks/recent`

Purpose:

- list recently finished tasks for UI tracking and notifications

### `GET /api/tasks/{taskId}`

Purpose:

- fetch task status and progress for an async operation

## Type References

Primary type sources:

- backend contracts: `backend/internal/contracts/contracts.go`
- frontend API types: `src/api/types.ts`

For exact field-level payload structure, consult those source files together with the server route handlers in `backend/internal/server/server.go`.
### `POST /api/curated-frames/export`

Purpose:

- export 1-20 curated frames in a single image format or a ZIP archive when multiple files are returned

Important notes:

- accepts `jpg`, `webp`, and `png`
- JPG exports use `.jpg` filenames and `image/jpeg`
- frontend single-export and batch-export actions now use the persisted `curatedFrameExportFormat` setting
