# Player Seek Time Sync Bugfix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent the player time/progress UI from getting stuck after keyboard seek shortcuts when the underlying media clock keeps advancing.

**Architecture:** Keep the current optimistic-seek UX, but add a small playback clock reconciliation helper that compares the displayed absolute time, the authoritative media clock, and the pending optimistic seek target. `PlayerPage.vue` will use that helper from both media events and a lightweight runtime reconciliation loop so the UI can recover even when a seek-related media event is skipped or delayed.

**Tech Stack:** Vue 3, TypeScript, Vitest, existing player/HLS playback stack.

---

### Task 1: Add regression tests for playback clock reconciliation

**Files:**
- Create: `src/lib/player-playback-clock.test.ts`
- Create: `src/lib/player-playback-clock.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, it } from "vitest"
import {
  clearOptimisticSeekTargetIfSettled,
  shouldReconcileDisplayedPlaybackTime,
} from "@/lib/player-playback-clock"

describe("clearOptimisticSeekTargetIfSettled", () => {
  it("clears the optimistic seek once the authoritative clock reaches the target", () => {
    expect(clearOptimisticSeekTargetIfSettled(120, 120.4)).toBeNull()
  })

  it("clears stale optimistic seek state when playback has progressed away from the target", () => {
    expect(clearOptimisticSeekTargetIfSettled(120, 126)).toBeNull()
  })

  it("keeps the optimistic seek while the authoritative clock is still near the pending target", () => {
    expect(clearOptimisticSeekTargetIfSettled(120, 118.8)).toBe(120)
  })
})

describe("shouldReconcileDisplayedPlaybackTime", () => {
  it("reconciles when the displayed time drifts behind the authoritative media clock", () => {
    expect(
      shouldReconcileDisplayedPlaybackTime({
        displayedTimeSec: 120,
        authoritativeTimeSec: 126,
        optimisticSeekTargetSec: 120,
        isScrubbingProgress: false,
        isPlaybackWaiting: false,
      }),
    ).toBe(true)
  })

  it("does not reconcile while the user is actively scrubbing the progress slider", () => {
    expect(
      shouldReconcileDisplayedPlaybackTime({
        displayedTimeSec: 120,
        authoritativeTimeSec: 126,
        optimisticSeekTargetSec: 120,
        isScrubbingProgress: true,
        isPlaybackWaiting: false,
      }),
    ).toBe(false)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/lib/player-playback-clock.test.ts`
Expected: FAIL with module-not-found or missing exports for `@/lib/player-playback-clock`

- [ ] **Step 3: Write minimal implementation**

```ts
export function clearOptimisticSeekTargetIfSettled(
  optimisticSeekTargetSec: number | null,
  authoritativeTimeSec: number,
  toleranceSec: number = 1,
): number | null {
  if (optimisticSeekTargetSec == null) return null
  const normalizedAuthoritative = Number.isFinite(authoritativeTimeSec) ? authoritativeTimeSec : 0
  if (Math.abs(normalizedAuthoritative - optimisticSeekTargetSec) <= toleranceSec) return null
  if (normalizedAuthoritative > optimisticSeekTargetSec + Math.max(2, toleranceSec)) return null
  return optimisticSeekTargetSec
}

export function shouldReconcileDisplayedPlaybackTime(input: {
  displayedTimeSec: number
  authoritativeTimeSec: number
  optimisticSeekTargetSec: number | null
  isScrubbingProgress: boolean
  isPlaybackWaiting: boolean
  toleranceSec?: number
}): boolean {
  if (input.isScrubbingProgress) return false
  const toleranceSec = Math.max(0, input.toleranceSec ?? 0.75)
  const displayed = Number.isFinite(input.displayedTimeSec) ? input.displayedTimeSec : 0
  const authoritative = Number.isFinite(input.authoritativeTimeSec) ? input.authoritativeTimeSec : 0
  if (Math.abs(authoritative - displayed) <= toleranceSec) return false
  if (input.isPlaybackWaiting && input.optimisticSeekTargetSec != null) {
    return authoritative > input.optimisticSeekTargetSec + Math.max(2, toleranceSec)
  }
  return true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/lib/player-playback-clock.test.ts`
Expected: PASS with all tests green

- [ ] **Step 5: Commit**

```bash
git add src/lib/player-playback-clock.ts src/lib/player-playback-clock.test.ts
git commit -m "test: cover player playback clock reconciliation"
```

### Task 2: Wire the helper into the player runtime

**Files:**
- Modify: `src/components/jav-library/PlayerPage.vue`
- Modify: `src/lib/player-playback-clock.ts`
- Test: `src/lib/player-playback-clock.test.ts`

- [ ] **Step 1: Update `PlayerPage.vue` to use the helper**

```ts
import {
  clearOptimisticSeekTargetIfSettled,
  shouldReconcileDisplayedPlaybackTime,
} from "@/lib/player-playback-clock"
```

Add a reconciliation runner that:
- reads `videoRef.value.currentTime`
- converts it with `getAbsolutePlaybackTime(...)`
- clears stale optimistic seek state
- forces `currentTime.value` back to the authoritative clock when the helper says reconciliation is needed

- [ ] **Step 2: Add a lightweight playback clock loop**

Implement a loop using `requestAnimationFrame` while the player is active so reconciliation still happens when `timeupdate`/`seeked` events are not emitted at the right moment.

- [ ] **Step 3: Re-run the focused regression test**

Run: `pnpm test -- src/lib/player-playback-clock.test.ts`
Expected: PASS

- [ ] **Step 4: Run existing nearby tests**

Run: `pnpm test -- src/lib/playback-targets.test.ts src/lib/player-shortcuts.test.ts src/lib/playback-progress-storage.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/components/jav-library/PlayerPage.vue src/lib/player-playback-clock.ts src/lib/player-playback-clock.test.ts
git commit -m "fix: reconcile player time after keyboard seek"
```

### Task 3: Verify the shipped behavior

**Files:**
- Modify: `docs/plan/2026-04-19-player-seek-time-sync-implementation-plan.md`

- [ ] **Step 1: Run typecheck**

Run: `pnpm typecheck`
Expected: PASS

- [ ] **Step 2: Record verification notes**

Document that the regression test covers:
- optimistic seek settling near the target
- stale optimistic seek clearing when authoritative playback moves far ahead
- reconciliation suppression during slider scrubbing

- [ ] **Step 3: Optional manual verification**

Run the app and verify:
1. Open a movie in the in-page player
2. Press `ArrowRight` / `ArrowLeft` repeatedly during playback
3. Confirm the visible time and progress bar continue advancing with the actual video

- [ ] **Step 4: Commit**

```bash
git add docs/plan/2026-04-19-player-seek-time-sync-implementation-plan.md
git commit -m "docs: record player seek time sync verification"
```

## Follow-up Root Cause Found After Manual Reproduction

The first implementation still allowed reproduction because `isPlaybackWaiting` was overloaded:

- `isPlaybackWaiting=true` represented real media buffering.
- The same flag also represented "a keyboard/slider seek was just requested and is waiting for confirmation."
- Direct playback used this same path, so `ArrowLeft` / `ArrowRight` eagerly displayed `player.bufferingSeek` before the media element had reported real buffering.
- The reconciliation loop also used too large a per-sample movement threshold (`0.2s`). Browser timer cadence can produce smaller samples during normal playback, causing the loop to repeatedly reset its baseline without ever treating the media clock as moving.

Follow-up fix:

- `shouldEnterSeekWaitingState("direct")` returns `false`; Direct seeks no longer pre-enter seek-buffering state.
- `shouldEnterSeekWaitingState("hls")` returns `true`; HLS seeks still get seek-buffering treatment because stream windows may need refill or replacement.
- Direct playback may still enter ordinary buffering when the media element emits `waiting`, but it no longer shows the HLS-oriented "seeking and refilling buffer" label.
- `hasAuthoritativeClockMoved()` lowers the movement threshold so the reconciliation loop can detect normal playback-sized clock progress.

## Second Follow-up Root Cause Found From 100% Repro

Reproduction:

1. Click the progress slider with the mouse.
2. Press `ArrowLeft` / `ArrowRight`.
3. The time/progress appears stuck and subsequent arrow presses do not perform player seek.

Cause:

- After mouse interaction, the Reka slider thumb keeps keyboard focus.
- `shouldIgnoreGlobalPlaybackHotkeysForTarget()` intentionally ignores events whose target is inside `[data-slot="slider"]`.
- The focused progress slider consumes arrow keys as slider keyboard input.
- For the progress slider, `@update:model-value` only calls `onProgressSliderInput()`, which updates preview/scrub state but does not commit `seekToAbsolutePlaybackTime()`.
- Therefore arrow keys after a progress-slider click can move the slider preview instead of seeking the video.

Second follow-up fix:

- Add `shouldBlurPlaybackSliderAfterCommit()` to identify active focus inside the committed progress slider.
- Add `progressSliderRootRef` on the progress slider wrapper.
- After `onProgressSliderCommit()` performs the actual seek, blur the focused progress slider element immediately and once more on a 0ms timer to handle focus timing differences.
- Do not apply this blur behavior to the volume slider; keyboard arrows on the volume slider remain valid local slider behavior.
