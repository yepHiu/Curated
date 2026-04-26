# Player Playback Settings Design

Date: 2026-04-26
Status: Proposed

## Goal

Add a player-local "Playback Settings" control on the player page so users can change, for the current playback session only:

- playback speed
- playback mode (`Direct` or `HLS`)

The control should open a small menu-like surface rather than a full page or large modal.

## Requirement Understanding

The requested behavior is scoped to the current player page / current playback session, not the global settings page:

- playback speed is a local per-session override
- playback mode is a local per-session override
- the entry point should be a dedicated playback-settings button inside the player controls
- clicking the button opens a compact settings surface
- users can choose a speed preset and a playback mode inside that surface

## Current Code Reality

### Playback speed

Playback speed is not currently surfaced in the player UI, but the browser video element already supports it naturally through `HTMLVideoElement.playbackRate`.

Implication:

- this part is mostly a frontend-local state problem
- it should be relatively low risk

### Playback mode

The current player already understands `direct` and `hls` session modes:

- `PlaybackDescriptorDTO.mode`
- `CreatePlaybackSessionBody.mode`
- `libraryService.createPlaybackSession(movieId, mode, startPositionSec?)`

Implication:

- we already have a backend contract for per-session mode creation
- mode switching can likely reuse the existing playback-session creation path
- we do not need to invent a brand-new backend concept for mode override

### Important product boundary

This feature should not silently mutate the global playback settings from the Settings page. It should stay local to the current player instance / current route session.

## UI Approaches

### Approach A: Compact dropdown-style playback settings menu

Interaction:

- add a small settings icon button in the player control cluster
- clicking opens a compact floating panel anchored to the button
- panel sections:
  - playback speed
  - playback mode

Pros:

- matches the requested "small menu" shape closely
- minimal interruption to playback
- compact enough for immersive player controls
- can reuse existing dropdown / popover design language

Cons:

- must stay carefully sized on mobile and narrow widths

Recommended.

### Approach B: Wider popover card

Interaction:

- same settings button entry
- opens a wider popover card with more breathing room
- speed shown as chips or segmented options
- mode shown as radio rows with description text

Pros:

- clearer information hierarchy
- more room for explanations like "HLS may be slower but more compatible"

Cons:

- feels slightly heavier than a menu
- more visual weight in the player chrome

### Approach C: Inline expandable controls

Interaction:

- tapping the settings button expands an inline controls strip directly in the bottom toolbar

Pros:

- no floating layer
- fast access

Cons:

- increases toolbar complexity
- more likely to crowd the player controls
- weaker fit for the current immersive direction

Not recommended for the first pass.

## Recommended UI

Use Approach A with a compact anchored panel.

### Recommended control contents

#### 1. Speed section

Use compact preset chips or radio-style items:

- `0.75x`
- `1.0x`
- `1.25x`
- `1.5x`
- `2.0x`

Reason:

- quick, low-friction, familiar
- no need for a free-form input in the first version

#### 2. Mode section

Use radio-style choices:

- `Direct`
- `HLS`

Optional helper text:

- `Direct`: lower overhead, best when browser direct playback is supported
- `HLS`: more compatible / resumable, but may need remux or transcode

Important rule:

- if direct play is not available for the current item, `Direct` should either be disabled or show a clear unavailable state

## Proposed Behavior

### Playback speed

- default to `1.0x` when entering the player
- changing the speed applies immediately to the current media element
- the choice lives only for the current player session
- leaving the player resets it unless later we explicitly choose to persist it

### Playback mode

- current mode should be reflected in the menu
- switching mode should recreate or swap the playback session using the selected mode
- keep the current playback time when switching modes
- if the player was playing before the switch, resume playback after the switch where possible

### Failure handling

- if mode switching fails, keep the previous playback session
- show a player toast / inline error message
- do not leave the player in a broken half-switched state

## Technical Plan

### Frontend

Likely changes:

- `src/components/jav-library/PlayerPage.vue`
  - add playback-settings button
  - add local state for:
    - current session speed
    - current session mode override
    - settings menu open / close
  - apply `video.playbackRate`
  - trigger mode switch through `libraryService.createPlaybackSession(...)`

- reuse existing UI primitives
  - likely `DropdownMenu` or a popover-like anchored surface

### Backend / contract

The existing contract already supports `mode` on `createPlaybackSession`, so the first implementation may not require a new endpoint.

Possible backend work may still be needed for polish:

- ensure forcing `direct` behaves predictably when the source is not direct-playable
- ensure error payloads are clear enough for frontend feedback

## Recommended First Version Scope

Keep the first version deliberately tight:

1. Add playback-settings button.
2. Open compact anchored panel.
3. Support speed presets only, not custom numeric speed.
4. Support `Direct` / `HLS` mode selection.
5. Preserve current playback time on mode switch.
6. Keep all changes local to the current player session.

## Acceptance Criteria

- player has a dedicated playback-settings button
- clicking it opens a compact menu-like surface
- users can change playback speed for the current session
- users can switch between `Direct` and `HLS` for the current session
- switching mode keeps current playback position as closely as possible
- the feature does not mutate the global Settings page values
