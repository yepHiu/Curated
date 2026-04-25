# Player Immersive Edge-To-Edge Plan

Date: 2026-04-26
Status: Proposed

## Goal

Only adjust the player experience so playback becomes as edge-to-edge and immersive as possible, without changing curated-frame views.

## Current Gap Sources

Code inspection shows player whitespace comes from multiple layers:

- `src/views/PlayerView.vue`
  - outer wrapper uses `pr-2`, which adds right-side breathing room
- `src/components/jav-library/PlayerPage.vue`
  - the main playback surface uses `p-4 sm:p-6 lg:p-8`, which creates explicit inset margins on all sides
  - the shutter overlay mirrors that inset with `absolute inset-4 sm:inset-6 lg:inset-8`
- the `<video>` itself uses `object-contain`
  - this preserves the full frame without cropping
  - if the video aspect ratio differs from the viewport aspect ratio, some top/bottom or left/right bars are mathematically unavoidable

## Approaches

### Approach A: Remove layout padding, keep `object-contain`

Changes:

- remove outer player-page wrapper padding
- remove the video-surface padding in `PlayerPage.vue`
- keep top and bottom control overlays as floating overlays with their own internal padding
- keep `object-contain`

Pros:

- removes all artificial gaps introduced by layout
- preserves the full video frame without cropping
- lowest risk

Cons:

- when viewport ratio and video ratio differ, some bars still remain

Recommended as the first pass.

### Approach B: Full bleed with `object-cover`

Changes:

- same padding removals as Approach A
- switch the video from `object-contain` to `object-cover`

Pros:

- visually fills the player area almost all the time
- strongest "immersive" effect

Cons:

- crops the picture on one axis
- risky for subtitle-safe areas and framed content

Not recommended as the default unless explicit cropping is acceptable.

## Recommended Implementation

Use Approach A first:

1. In `src/views/PlayerView.vue`, remove the outer `pr-2` so the route itself is flush.
2. In `src/components/jav-library/PlayerPage.vue`, remove the main video-area padding (`p-4 sm:p-6 lg:p-8`).
3. Update the shutter overlay inset to `inset-0` so it matches the flush surface.
4. Keep `object-contain` for now.
5. Leave top/bottom chrome overlays intact, since they are overlay layers and do not need to consume layout width/height.

## Expected Result

- all human-added spacing around the playback surface is removed
- the video becomes edge-to-edge within the available player canvas
- any remaining bars come only from aspect-ratio mismatch, not layout padding

## Verification

- run `pnpm typecheck`
- manually inspect player route on desktop
- confirm top/bottom overlay controls still align correctly

## Immersive Chrome Idle-Hide Refinement

Date: 2026-04-26
Status: Proposed

### New Requirements

- the page should enter immersive mode after 5 seconds of no mouse movement on the player page, not only inside the playback surface
- keyboard shortcuts should continue working while chrome is hidden
- keyboard shortcuts should not force chrome back into view
- chrome should only come back when the mouse moves again
- when chrome is hidden, seek backward / seek forward should still show lightweight feedback without breaking immersion

### Current Behavior

Current code in `src/components/jav-library/PlayerPage.vue` behaves like this:

- idle-hide is driven by mouse activity on the player surface only
- `mousemove`, `mousedown`, and `mouseenter` call the same visibility-reset path
- global playback hotkeys currently also call the visibility-reset path before handling the shortcut
- as a result, using `ArrowLeft`, `ArrowRight`, `J`, `L`, `Space`, or other player hotkeys can bring chrome back

### Approaches

#### Approach A: Minimal patch on top of current surface logic

Changes:

- keep the existing surface-level mouse listeners as the main source of idle detection
- remove the forced `onChromePointerActivity()` call from keyboard shortcut handling
- add a small centered seek hint when seeking while chrome is hidden

Pros:

- smallest code change
- lowest regression risk

Cons:

- does not meet the new requirement exactly, because "page idle" is still effectively "surface idle"
- if future player-page overlays or side zones grow, the behavior will feel inconsistent

#### Approach B: Page-scoped mouse-idle detection with separate chrome and feedback channels

Changes:

- move immersive idle detection from "playback surface only" to "entire player page while this route is active"
- only mouse movement re-shows chrome
- mouse down, keyboard shortcuts, and seek operations no longer re-show chrome automatically
- keep keyboard shortcuts functional while chrome is hidden
- add a lightweight transient seek HUD for backward / forward actions when chrome is hidden

Pros:

- matches the requirement closely
- keeps immersive mode stable during keyboard-driven playback control
- gives feedback for seeks without reintroducing the full title/progress/control chrome

Cons:

- slightly larger change than Approach A
- needs careful event scoping so non-mouse interactions do not accidentally wake chrome

Recommended.

#### Approach C: Fully split chrome state machine

Changes:

- introduce separate states for `mouse-active`, `chrome-visible`, `feedback-visible`, `modal-visible`, and `controls-locked`
- unify pointer, keyboard, context menu, and playback state through a dedicated controller layer

Pros:

- strongest long-term extensibility
- easiest to evolve if later we add different immersive modes

Cons:

- too heavy for the current request
- higher risk of touching unrelated player behavior

Not recommended for this round.

### Recommended Implementation

Use Approach B with conservative boundaries:

1. Keep the existing 5-second timeout constant.
2. Switch the idle-detection scope from the playback surface to the whole player page container while the player route is mounted.
3. Treat only real mouse movement as a wake-up signal for chrome.
4. Remove the current shortcut path that treats keyboard usage as activity.
5. Keep pointer-driven UI interactions explicit:
   - opening context menu can still show chrome because that is a deliberate pointer interaction
   - clicking progress or controls can still keep chrome visible because the user is already in direct UI interaction
6. Add a small transient seek feedback HUD for backward / forward while chrome is hidden:
   - anchored to the top-right corner
   - text/icon only, low-opacity, auto-dismiss
   - no background card heavier than needed
   - examples: `-10s`, `+10s`
7. Reuse the existing toast system only for errors, not for immersive seek feedback.
8. When curated capture succeeds while chrome is hidden, reuse the same top-right HUD channel with a lighter theme-colored treatment to distinguish it from seek actions.

### Suggested Feedback Design

For seek feedback while chrome is hidden:

- show a compact transient HUD in the top-right corner
- duration around 500-800 ms
- content:
  - backward: left icon + `-10s`
  - forward: right icon + `+10s`
  - curated capture: localized curated label + `+1`
- styling:
  - soft white text
  - subtle dark translucent backing or blurred chip
  - no full-width banner
  - no toast stacking
  - keep it inset slightly from the edge so it feels attached to the viewport, not floating over the subject center
  - curated capture keeps the same transparency and motion, but swaps to a pale primary-tinted treatment so action types are distinguishable by color

### Acceptance Criteria

- after 5 seconds without mouse movement on the player page, title, progress, toolbar, and cursor are hidden
- pressing playback hotkeys while hidden does not show chrome
- moving the mouse shows chrome again and restarts the 5-second timer
- backward / forward shortcuts still provide lightweight feedback when hidden
- direct pointer interaction with visible controls still behaves normally
