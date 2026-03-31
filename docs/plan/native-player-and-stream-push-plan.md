# Curated Native Player And Stream Push Plan

## Goal

Upgrade Curated from browser-only playback into a dual-path playback system:

1. browser playback remains available
2. a native player kernel can be launched from Curated
3. the backend can create playback sessions and expose pushed stream output for browser fallback

## Scope For This Iteration

This iteration targets the minimum stable version:

- native player kernel: external `mpv` process launched by the backend
- stream push: backend-managed HLS session produced by `ffmpeg` when needed
- browser path: current direct `/stream` path remains the default fallback
- frontend: add a clear "open in native player" action and consume richer playback descriptors

## Design

### Playback modes

Curated should support these playback modes in the descriptor/session layer:

- `direct`
  - browser plays `/api/library/movies/{id}/stream`
- `hls`
  - browser plays `/api/playback/sessions/{sessionId}/hls/index.m3u8`
- `native`
  - backend launches the native player kernel and returns launch state

### Backend components

Add these backend building blocks:

- `internal/nativeplayer`
  - resolves player executable path
  - launches `mpv`
  - supports direct file path or session URL input
- `internal/playback`
  - playback session manager
  - optional HLS transcoding lifecycle
  - session cleanup and output directory management
- HTTP routes
  - `POST /api/library/movies/{id}/playback-session`
  - `POST /api/library/movies/{id}/native-play`
  - `GET /api/playback/sessions/{sessionId}/hls/{file}`
  - `DELETE /api/playback/sessions/{sessionId}`

### Native player strategy

The native player kernel should be external-process based first, not embedded.

Reason:

- much smaller change
- stable on Windows
- aligns with Curated's current tray + local server architecture
- gives strong codec support immediately once `mpv` is present

### Stream push strategy

The pushed stream path should use `ffmpeg` to generate HLS into a managed session directory.

Rules:

- if direct play is good enough, keep using `/stream`
- if browser playback needs a safer format, create an HLS session
- if native playback is explicitly requested, launch native player and bypass browser decode

## Config Additions

Extend backend player config with optional fields:

- `nativePlayerEnabled`
- `nativePlayerCommand`
- `nativePlayerArgs`
- `streamPushEnabled`
- `ffmpegCommand`
- `streamSessionRoot`
- `preferNativePlayer`

All of these should be optional and default-safe so current installs keep working.

## Frontend Changes

### Player page

Add one native playback action:

- button: open in native player

Behavior:

- call backend native-play endpoint
- show a success or error toast/message
- keep current browser player available

### Playback descriptor usage

Frontend should continue using the playback descriptor seam and understand:

- `direct`
- `hls`
- `native`

For this iteration, browser UI will actively render `direct` and `hls`.
`native` is mainly exposed through the explicit launch action.

## Validation

Minimum validation for this phase:

- backend unit tests for session creation and native-launch endpoint
- browser player still works for direct playback
- native play endpoint returns clear error when `mpv` is unavailable
- HLS session routes serve playlist and segment files when `ffmpeg` is configured

## Follow-Up

After this iteration, the natural next upgrades are:

1. media capability inspection
2. auto decision between direct and HLS
3. native player IPC for progress sync and transport controls
4. better session cleanup and idle eviction
