# Player Progress Hit Area Design

Date: 2026-04-26
Status: Proposed

## Goal

Improve the player seek-bar usability so users do not need to click the progress bar with overly strict precision before a jump is triggered.

## Current Behavior

Current player progress interaction is wired in:

- `src/components/jav-library/PlayerPage.vue`
- `src/components/ui/slider/Slider.vue`

The current slider track uses a very thin visual height (`h-1.5`), while the player expects click-to-seek behavior on that same surface. In practice, this makes the effective seek target feel too narrow and too precise.

## User Need

- users should be able to click near the progress bar and still seek successfully
- visual style should remain restrained and immersive
- seek precision should remain stable
- drag behavior should not regress

## Approaches

### Approach A: Enlarge the interactive hit area only

Changes:

- keep the track visually thin
- enlarge the slider root / effective interactive height to roughly `20px ~ 28px`
- vertically center the thin track inside that larger interaction zone
- optionally enlarge the thumb hot zone without making the thumb itself look larger

Pros:

- lowest implementation risk
- preserves the current visual design
- improves click-to-seek immediately
- does not require custom time-mapping logic

Cons:

- still bounded by the slider's interaction box
- not a true fuzzy-distance interpretation

Recommended as the first pass.

### Approach B: Add a near-track click tolerance layer

Changes:

- add a transparent click-capture layer around the seek bar
- if the pointer lands within a configured vertical tolerance from the track center, treat it as a valid seek click
- compute target time from horizontal position

Pros:

- more closely matches a "fuzzy hit" model
- stronger usability improvement if the current slider library still feels too strict after Approach A

Cons:

- more custom pointer logic
- must be carefully coordinated with drag behavior and existing slider events
- higher regression risk than Approach A

### Approach C: Hover-proximity expansion

Changes:

- keep the default slim appearance
- expand the visual or interactive zone when the pointer approaches or hovers the seek area

Pros:

- polished feel
- can make the seek bar look more intentionally interactive

Cons:

- introduces a more visible behavior change
- higher chance of conflicting with immersive playback goals

Not recommended for the first pass.

## Recommended Implementation

Use Approach A first, with conservative sizing:

1. Keep the current thin progress-bar appearance.
2. Increase the effective seek interaction height to around `24px`.
3. Keep the track vertically centered inside that larger hit area.
4. Leave current seek precision and drag behavior unchanged.
5. Manually verify whether click-to-seek becomes comfortable enough.

If that still feels too strict, follow with a small vertical-tolerance layer from Approach B using a limited threshold such as `8px ~ 12px` from the track center.

## Acceptance Criteria

- users no longer need to click exactly on the thin visible line to seek
- the seek bar still looks visually slim
- drag behavior remains stable
- no accidental large-area false-positive seeking appears in the control bar
