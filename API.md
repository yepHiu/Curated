# Curated API Reference

## Overview

Curated exposes a Go HTTP API for PIN App Lock/authentication, library browsing, playback workflows, actor metadata, settings, connected-client visibility, curated-frame management, storage presence checks, scans, and task tracking.

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

## Authentication And PIN App Lock

PIN App Lock is optional and is disabled by default. When enabled, protected `/api/*`
endpoints require a valid unlock session. Locked requests return HTTP `423` with
error code `AUTH_LOCKED`. `/api/health` and the auth endpoints below remain public
so the frontend can render the lock flow.

PINs are stored only as salted Argon2id hashes in SQLite. Browser unlock state is a
server-side `auth_sessions` row carried by the HTTP-only `curated_auth` cookie.
Regular sessions use an idle timeout: activity through the UI or protected API calls
extends `sessionExpiresAt`; inactivity past that deadline locks the session. The
`trustedForever` option creates a session without `expiresAt`; that device stays trusted
until the current session is explicitly locked or a future session-management flow revokes it.

### `GET /api/auth/status`

Purpose:

- return whether PIN lock is enabled and whether the current request has a valid session
- provide the current idle timeout, LAN PIN policy, restart-lock policy, and trusted-device status

Response shape:

- `pinEnabled`: whether PIN lock is active
- `unlocked`: whether the current cookie/session is valid, or true when PIN is disabled
- `setupRequired`: true until a PIN has been configured
- `pinLength`: configured PIN digit count when known; the lock screen uses this to render the correct number of PIN cells without storing or exposing the PIN itself
- `trustedForever`: true when the current session was unlocked with permanent device trust
- `sessionExpiresAt`: current idle deadline for regular sessions; empty for trusted-forever sessions
- `sessionTtlMinutes`: idle-lock delay in minutes
- `lanRequiresPin`, `lockOnRestart`

### `POST /api/auth/setup-pin`

Purpose:

- set the first PIN and create an unlock session for the current browser
- update non-secret PIN lock settings at the same time

Request body:

- `pin`: 4-8 numeric digits
- `confirmPin`: must match `pin`
- optional `sessionTtlMinutes`, `lanRequiresPin`, `lockOnRestart`
- optional `trustedForever`: trust the current browser indefinitely after setup

Important notes:

- when a PIN already exists, this endpoint requires the current request to be unlocked
- the backend persists only the PIN length as non-secret metadata, alongside the salted Argon2id hash
- the response sets an HTTP-only `curated_auth` cookie on success

### `POST /api/auth/unlock`

Purpose:

- verify the submitted PIN and create a new unlock session

Request body:

- `pin`: submitted PIN
- optional `trustedForever`: when true, create a non-expiring trusted-device session

Important notes:

- incorrect PIN returns HTTP `401` with error code `AUTH_INVALID_PIN`
- success sets an HTTP-only `curated_auth` cookie; regular sessions use a browser-session cookie while the server owns the idle deadline

### `POST /api/auth/change-pin`

Purpose:

- replace the configured PIN after the current request is already unlocked

Request body:

- `currentPin`: existing PIN
- `newPin`: new 4-8 digit numeric PIN
- `confirmPin`: must match `newPin`

Important notes:

- the endpoint is protected by the app-lock middleware and also verifies `currentPin`
- incorrect current PIN returns HTTP `401` with error code `AUTH_INVALID_PIN`
- plaintext PIN values are never persisted; the response includes the updated `pinLength`

### `POST /api/auth/lock`

Purpose:

- revoke the current session and clear the `curated_auth` cookie

Important notes:

- this also revokes a trusted-forever session for the current device

### `PATCH /api/auth/settings`

Purpose:

- update non-secret PIN lock settings after the current request is already unlocked

Request body:

- optional `pinEnabled`
- optional `sessionTtlMinutes`
- optional `lanRequiresPin`
- optional `lockOnRestart`

## Connected Clients

### `GET /api/connected-clients`

Purpose:

- list clients that have accessed the backend during the current backend process lifetime
- power the Settings -> Overview connected-clients card below the watch-time statistics

Response shape:

- `clients`: array of tracked client entries
- `total`: total tracked clients in the current process
- `localCount`: clients classified as local access
- `remoteCount`: clients classified as remote access
- `sampledAt`: server timestamp for the response

Client entry fields:

- `key`: stable in-memory key for the current process, normally derived from IP + User-Agent; Curated Desktop includes its desktop marker so it stays distinct from a normal Chrome tab with the same Chromium User-Agent
- `ip`, optional `port`, optional `hostname`
- `browser`, optional `browserVersion`, `os`, optional `osVersion`; Windows `osVersion` is shown only when the backend can determine a user-facing version such as `10` or `11`
- `deviceType`: `desktop`, `laptop`, `mobile`, `tablet`, `tool`, or `unknown`
- `accessKind`: `local` or `remote`
- `isLocalMachine`: true for loopback or known host-interface IPs
- `firstSeen`, `lastSeen`, `requestCount`

Important notes:

- tracking is in-memory only; restarting the backend clears the list
- clients are deduplicated by remote IP + User-Agent, not by TCP port
- Curated Desktop/Electron adds `X-Curated-Client: desktop-electron`, `X-Curated-Client-Version`, `X-Curated-OS`, and `X-Curated-OS-Version`; the backend reports those entries as `browser: "Curated Desktop"` instead of Chrome and can show Windows 11 even though Chromium's legacy User-Agent still contains `Windows NT 10.0`
- regular browsers may provide `Sec-CH-UA-Platform` and `Sec-CH-UA-Platform-Version`; when present, the backend uses those Client Hints to map Windows platform version 13+ to `Windows 11`
- the backend keeps at most 50 most-recent clients
- MAC addresses are not collected or exposed

## App Updates

### `GET /api/app-update/status`

Purpose:

- return the current app-update comparison result used by Settings -> About and the sidebar brand badge

Important notes:

- the backend compares the current runtime `installerVersion` with the latest GitHub Release for `yepHiu/Curated`
- successful release checks expose `installerDownloadUrl` when the latest release includes a Windows `.exe` installer asset
- when the release asset includes a SHA256 digest, the response also exposes `installerSha256`
- downloaded installer state is reflected through `artifactStatus`, `downloadedVersion`, `downloadedFileName`, `downloadedBytes`, `totalBytes`, `downloadProgress`, `signatureStatus`, `installReady`, `lastInstallAttemptAt`, and `lastInstallError`
- development runtimes use `0.0.0` as the local installed version so the full update-check path remains testable before packaging
- successful checks are cached in SQLite so routine reads do not hit GitHub on every app start
- response fields include `supported`, `status`, `installedVersion`, optional `latestVersion`, `hasUpdate`, `checkedAt`, `publishedAt`, `releaseName`, `releaseUrl`, `installerDownloadUrl`, `installerSha256`, `releaseNotesSnippet` (full GitHub release body text, newline-preserved, capped around 100k runes for cache size), artifact fields, and optional `errorMessage`

### `POST /api/app-update/check`

Purpose:

- force a fresh app-update check against the latest GitHub Release

Important notes:

- bypasses the cached status used by `GET /api/app-update/status`
- returns the same DTO shape as the status endpoint
- used by the manual "Check for updates" action in Settings -> About

### `POST /api/app-update/download`

Purpose:

- download the latest Windows installer into Curated's controlled update cache and verify it with SHA256

Important notes:

- requires the cached update state to be `update-available`
- requires `installerDownloadUrl` and `installerSha256`
- downloads to the backend cache under an `updates/<version>/` directory
- returns the same DTO shape as the status endpoint with `artifactStatus=verified` and `installReady=true` after verification
- unsigned installers can still be downloaded and SHA256 verified, but signature status remains `not_checked`

### `POST /api/app-update/install`

Purpose:

- launch the verified downloaded installer after explicit user action

Request body:

- `mode`: optional `interactive`, `silent`, or `verysilent`; default is `interactive`

Important notes:

- requires `installReady=true`
- `silent` and `verysilent` only pass Inno Setup quiet flags; Windows UAC can still appear when the install target requires administrator privileges
- returns the same DTO shape with `artifactStatus=install-launched` after the installer process is started

### `DELETE /api/app-update/downloaded-installer`

Purpose:

- remove the cached downloaded installer and clear artifact metadata from the app-update status row

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
- accepts an optional JSON body `{ "preserveHeroMovieIds": ["..."], "excludeRecommendationMovieIds": ["..."] }`; when provided, the backend keeps those hero IDs as the persisted `heroMovieIds` and avoids returning the caller's currently visible recommendation IDs while regenerating `recommendationMovieIds`
- the homepage "Today's Recommendations" refresh sends both `preserveHeroMovieIds` and `excludeRecommendationMovieIds` so clicking it refreshes only the recommendation rail, avoids the current rail where inventory allows, and does not change the current hero carousel
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

### `GET /api/library/paths/storage-status`

Purpose:

- return the current storage availability snapshot for each configured library root

Response:

- `items`: array of `LibraryPathStorageStatusDTO`

Status values:

- `online`: the configured directory is reachable, readable, and matches the stored backing-volume identity when one is known
- `offline`: the backing storage root appears unavailable or disconnected
- `volume_mismatch`: the path resolves to a different volume than the one previously bound to this library path
- `path_missing`: the backing storage is present, but the configured library directory is missing or is not a directory
- `permission_denied`: Curated can see the path, but cannot read it
- `unknown`: Curated could not classify the storage state

Important notes:

- Windows is the primary supported platform for volume identity detection in this phase
- macOS and Linux currently use the platform-neutral fallback and are future adaptation targets
- responses include `canRescan` and `canImport`; the frontend uses these to block scans or imports that would target unavailable storage
- responses can include `rootPath`, `driveType`, `volumeLabel`, `fileSystem`, `identityConfidence`, `expectedVolumeId`, and `currentVolumeId` for diagnostics

### `POST /api/library/paths/storage-status/check`

Purpose:

- perform a fresh storage availability probe for configured library roots

Request:

- optional JSON body `{ "libraryPathIds": ["..."] }`
- omit the body or pass an empty list to check all configured library roots

Response:

- same shape as `GET /api/library/paths/storage-status`

Important notes:

- online checks persist the current backing-volume binding for the library path
- the frontend calls this during app startup in Web API mode and before storage-sensitive actions such as full scans and movie imports

### `POST /api/library/paths/{id}/storage-binding/rebind`

Purpose:

- replace the stored backing-volume binding for one configured library root with the currently detected volume identity

Response:

- one `LibraryPathStorageStatusDTO`

Important notes:

- intended for deliberate recovery from `volume_mismatch`, for example after replacing a disk or moving a library root to a new volume
- the new binding is persisted only when the current path is classified as `online`

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
- `autoDownloadUpdates` is an opt-in library-level setting surfaced in Settings -> General; when enabled, the startup background update check may download and SHA256-verify a newer installer, while installation still requires explicit user confirmation
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
