# Curated HLS Stream Pipeline Plan

## Goal

Implement an in-page HLS playback path so Curated can keep playback inside the current UI without opening a separate native-player window.

## Scope

This plan focuses only on the HLS pipeline:

- backend creates HLS playback sessions with `ffmpeg`
- frontend can play `.m3u8` output inside the existing player page
- direct `/stream` playback remains available as the fallback
- native player work stays optional and secondary

## Implementation

### Backend

- keep `GET /api/library/movies/{id}/playback` as the single playback entry
- when the source container is browser-risky and HLS is enabled:
  - create a managed HLS session
  - return a descriptor with:
    - `mode = hls`
    - `sessionId`
    - HLS playlist URL
- serve generated HLS files from `/api/playback/sessions/{id}/hls/{file}`
- clean up session directories when playback ends or the page switches sources

### Frontend

- keep the existing player UI and controls
- add `hls.js` for browsers that do not natively support HLS
- when the descriptor mode is `hls`:
  - attach `hls.js` to the existing `<video>`
  - load the playlist URL
  - keep the same progress, resume, and keyboard controls
- when the descriptor mode is `direct`:
  - keep the current plain `<video src=...>` path

## Defaults

- direct playback still wins for browser-friendly formats
- HLS is preferred for containers such as `.mkv`, `.avi`, `.ts`, `.m2ts`, `.wmv`, `.mov`
- if `ffmpeg` is missing or HLS session creation fails, Curated falls back to direct playback instead of breaking playback entirely

## Validation

- player page can open and play an HLS descriptor
- session cleanup runs on source switch and component unmount
- direct playback still works unchanged
- backend tests continue to pass
- frontend typecheck continues to pass

## Follow-Up

After the HLS path is stable, the next natural steps are:

1. capability detection based on real media metadata instead of extension-only heuristics
2. HLS bitrate ladder variants
3. live progress/health reporting for active transcoding sessions
4. settings UI for enabling/disabling HLS stream push and choosing the `ffmpeg` command
