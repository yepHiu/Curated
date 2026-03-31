# Curated Player Pipeline Evolution Plan

## Current State

Curated currently uses a simple playback pipeline:

1. The browser opens the player route.
2. The frontend generates `/api/library/movies/{id}/stream`.
3. The Go backend opens the local video file.
4. The backend returns the file through HTTP with Range support.
5. The browser `<video>` element performs decoding and playback.

This is a good minimum implementation, but it has clear limits:

- playback capability depends entirely on browser codec support
- there is no transcoding
- there is no bitrate adaptation
- subtitle and audio track control are limited to what the browser exposes
- future desktop-player ambitions are constrained by browser media behavior

## Recommended Evolution Path

The most natural path is:

### Phase 1: Keep browser playback, add a richer playback contract

Do not replace the browser player immediately.

Instead, add a formal backend playback session contract so the frontend stops treating playback as "just one file URL".

Recommended additions:

- `GET /api/library/movies/{id}/playback`
  - returns a playback descriptor instead of only a raw stream URL
- descriptor fields:
  - primary stream URL
  - mime / container / codec hints
  - subtitle tracks
  - audio tracks
  - duration
  - resume point
  - direct-play vs transcode capability flags

This phase gives us a stable seam for future upgrades while keeping current behavior working.

### Phase 2: Introduce playback capability analysis in backend

Before changing the player engine, teach the backend how to classify files:

- direct-play safe in browser
- direct-play risky
- requires transcode

This can be based on media metadata such as:

- container
- video codec
- audio codec
- subtitle type
- bitrate / resolution

This phase does not yet require transcoding. It only lets Curated make smarter decisions.

### Phase 3: Add optional transcoding pipeline

Once the backend can classify playback capability, the next natural step is:

- add an FFmpeg-based transcoding service
- keep direct play as the default fast path
- fall back to transcode only when the browser cannot reliably play the source

Recommended shape:

- `GET /api/library/movies/{id}/stream` remains the direct-play path
- `POST /api/library/movies/{id}/playback-session` creates a session
- the backend decides:
  - direct play
  - remux
  - transcode
- the frontend consumes the returned session descriptor

This is the cleanest path because it preserves current behavior and adds power only when needed.

### Phase 4: Decide whether desktop playback should remain browser-based

At this point there are two realistic directions.

#### Option A: Continue with browser player as the main playback shell

Best when:

- Curated stays mainly a browser-first local app
- direct play plus selective transcoding is enough
- you want lower implementation complexity

Pros:

- smallest architecture change
- keeps current UI intact
- easiest to evolve incrementally

Cons:

- browser codec limits still exist
- advanced player features remain harder

#### Option B: Add a native desktop playback engine

Best when:

- you want stronger local playback capability
- you want better subtitle / audio / hardware decode control
- you want a Jellyfin / Plex desktop-app style experience

The most natural engine candidate is not Electron video playback itself, but:

- native external engine such as `mpv`
- or an embedded native player surface later

Recommended first step if choosing native playback:

- keep the current web UI
- launch a local native player for playback only
- pass a structured playback session payload from backend to the native player

This avoids trying to reimplement the entire application shell too early.

## Recommended Near-Term Strategy

For Curated's current architecture, the most natural next step is:

1. Keep the browser player for now.
2. Add a backend playback descriptor API.
3. Add playback capability detection.
4. Introduce optional FFmpeg transcoding only for unsupported media.
5. Re-evaluate native player integration after that seam is stable.

This path is the best fit because Curated already has:

- a local backend service
- a browser UI shell
- a tray-managed desktop runtime
- file-based direct streaming

So the system is already shaped like:

- desktop entry
- local API server
- browser UI
- media service

That makes "backend playback orchestration" the natural next layer, not "replace everything with a different shell".

## Suggested Milestones

### Milestone 1

Add a playback descriptor API and refactor frontend playback to consume it.

### Milestone 2

Add media capability inspection and direct-play eligibility rules.

### Milestone 3

Add optional FFmpeg remux / transcode sessions.

### Milestone 4

Evaluate native desktop playback integration for advanced users or release builds.

## Final Recommendation

The most natural evolution is:

browser direct-play -> backend playback descriptor -> capability-aware routing -> optional transcoding -> optional native player

That gives Curated a strong media pipeline without forcing an early rewrite of the current product architecture.
