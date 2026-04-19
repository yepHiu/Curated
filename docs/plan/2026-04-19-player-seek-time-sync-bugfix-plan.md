# Player Seek Time Sync Bugfix Plan

## Problem Summary

- Symptom reported by user: while the in-page player is already playing, pressing keyboard seek shortcuts (`ArrowLeft` / `ArrowRight`) can sometimes fail to move playback.
- After the bug triggers, the on-screen playback time/progress can stay stuck at one timestamp even though the underlying video keeps playing forward.

## Current Evidence

- `src/components/jav-library/PlayerPage.vue` updates UI playback time from the real media clock mainly in event handlers such as `onTimeUpdate()` and `onVideoSeeked()`.
- Keyboard seek handling (`onPlaybackKeydown -> seekDelta -> startOptimisticSeek`) eagerly overwrites `currentTime` before the media element confirms that the seek actually landed.
- There is no independent "authoritative clock reconciliation" loop while playback is running. If a seek-related media event is skipped, coalesced, or delayed, the UI can keep showing the optimistic timestamp instead of the current `video.currentTime`.

## Options

### Option A: Add authoritative clock reconciliation while playing

- Keep the current optimistic seek UX so the UI still reacts immediately.
- Add a lightweight sync loop while playback is active that periodically re-reads `video.currentTime` and forces `currentTime` back to the authoritative media clock.
- Clear stale optimistic seek state both when the media clock reaches the target and when the media clock is clearly progressing somewhere else.

Why this is recommended:

- It addresses the observed freeze directly instead of only masking one seek path.
- It is robust across direct playback, HLS local seek, and HLS session swap timing.
- It keeps the current responsive feel.

### Option B: Disable repeated/in-flight keyboard seeks

- Ignore arrow-key seeks while a prior seek/session swap is still pending.

Trade-off:

- Lower implementation risk, but it does not address the deeper "UI clock can drift away from the media clock" issue.
- It may still leave edge cases where the UI becomes stale for other reasons.

### Option C: Remove optimistic time updates entirely

- Only update the on-screen time after the browser emits seek/time events.

Trade-off:

- Simplest mental model, but UX becomes noticeably less responsive.
- Does not help with event-loss scenarios if media events remain unreliable during some transitions.

## Recommended Design

Implement Option A.

### Scope

- Extract or add a small, testable helper for deciding when optimistic seek state should be cleared or overridden by the real media clock.
- Add a playback-time reconciliation loop that runs only while the player is active and a source is loaded.
- Cover the regression with focused Vitest tests before changing production logic.

### Non-Goals

- No changes to backend playback session APIs.
- No new playback controls or shortcut behavior changes.
- No large player refactor beyond the clock/seek synchronization path.
