# Player Shortcuts And Curated Keybinding Design

Date: 2026-04-16

## Scope

This change covers two related UX issues:

1. Fix the player bug where pressing volume hotkeys can make the progress slider twitch.
2. Add a single-key customizable curated-frame capture hotkey in Settings -> Curated.

## Root Cause Summary

The player currently listens to `window`-level `keydown` events in `PlayerPage.vue`.
At the same time, both the progress slider and the volume slider are `reka-ui` sliders, which keep their own keyboard behavior when a slider thumb has focus.

Current effect:

- `ArrowUp` / `ArrowDown` are handled globally as volume hotkeys.
- A focused slider can also react to the same arrow keys locally.
- Because the global handler does not exclude slider-focused targets, one key press can drive two independent UI updates.
- The visible symptom is that the progress or volume UI can "twitch" when arrow-key volume shortcuts are used.

## Options Considered

### Option A: Disable slider keyboard behavior

Change the slider wrapper or component usage so the sliders no longer respond to arrow keys.

Pros:

- Removes the conflict at the component level.

Cons:

- Regresses accessibility and standard slider keyboard behavior.
- Too broad for a focused bug fix.

### Option B: Keep slider keyboard support, but ignore global shortcuts when focus is on slider controls

Detect focused interactive slider elements and skip the player-level hotkey handler for those events.

Pros:

- Preserves normal slider accessibility behavior.
- Minimal blast radius.
- Matches expected media-player behavior: focused control owns its keys.

Cons:

- Requires careful target detection.

### Option C: Move all player hotkeys from `window` to a narrower focus scope

Attach key handling to the player surface only when the player area is focused.

Pros:

- Cleaner event ownership model long term.

Cons:

- Larger behavioral shift.
- Higher regression risk for existing keyboard-only usage.

## Recommended Approach

Use Option B.

This keeps the player hotkeys intact for normal playback, but prevents them from hijacking arrow-key input when the user is actively focused inside slider controls.

## Curated Capture Hotkey Design

### Requirements

- Only a single key is supported.
- If the chosen key conflicts with reserved player shortcuts, saving is blocked.
- The UI must clearly explain why a key cannot be used.

### Reserved Keys

The curated capture shortcut cannot be set to any key already reserved by player controls, including:

- `Space`
- `ArrowLeft`
- `ArrowRight`
- `ArrowUp`
- `ArrowDown`
- `Escape`
- `J`
- `K`
- `L`
- `M`
- `F`
- `P`

The default curated capture key remains `C`.

### Storage

Persist the curated capture key in browser-local settings beside existing curated-frame settings storage.

This remains frontend-local state, similar to other browser/device-specific UX preferences.

### Settings UI

Add a dedicated "Capture Shortcut" subsection inside Settings -> Curated.

The interaction:

1. Show the current shortcut as a `kbd`-style value.
2. Provide a "Set shortcut" button that enters a capture mode.
3. In capture mode, the next valid single key press becomes the candidate shortcut.
4. If the key is reserved, do not save and show an inline validation message.
5. Provide a "Reset to default" action.

### Interaction Rules

- Only printable letter/number/function-style single keys are accepted.
- Modifier combinations are ignored.
- Pressing `Escape` while in capture mode cancels capture mode; it does not become the shortcut.
- The UI must show which keys are reserved.
- Successful save should update the displayed shortcut immediately.

## Player Behavior Changes

### Volume Shortcut Fix

In `onPlaybackKeydown`:

- Continue to ignore typing targets.
- Also ignore keyboard shortcuts when the event target is inside a slider control.
- Result: focused sliders keep their own arrow-key behavior and no longer fight with the global volume hotkeys.

### Curated Capture Shortcut

Replace the hard-coded `KeyC` branch with a lookup against the configured curated capture key.

Behavior:

- If the current key matches the configured capture key, run the same capture flow.
- The visible curated button in the player remains unchanged.
- Settings text should reflect the live configured key instead of hard-coding `C`.

## Testing Plan

Add or update tests for:

1. Player keyboard handling ignores global volume hotkeys when a slider target is focused.
2. Configured curated capture key triggers capture.
3. Reserved keys are rejected by validation.
4. Settings shortcut capture UI saves a valid single key.
5. Reset-to-default restores `C`.

## Files Likely Affected

- `src/components/jav-library/PlayerPage.vue`
- `src/components/jav-library/SettingsPage.vue`
- `src/lib/curated-frames/settings-storage.ts`
- `src/locales/en.json`
- `src/locales/zh-CN.json`
- `src/locales/ja.json`

Possible new files:

- `src/lib/curated-frames/shortcut-settings.ts`
- `src/components/jav-library/settings/SettingsCuratedShortcutSection.vue`
- related test files
