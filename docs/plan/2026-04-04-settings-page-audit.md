# Settings Page Audit 2026-04-04

## Scope

This audit focused on the Curated settings page save / refresh interactions, autosave stability, request concurrency, and the most visible UX/performance risks around the current implementation in `src/components/jav-library/SettingsPage.vue`.

## High-Confidence Problems Confirmed

### 1. Concurrent writes to `library-config.cfg`

- Multiple settings groups ultimately persist to the same backend file: `config/library-config.cfg`.
- Before this audit, writes were not globally serialized at the config helper layer.
- Resulting risk:
  - Two fast `PATCH /api/settings` requests could both perform read-modify-write against stale snapshots.
  - On Windows development environments, temp-file rename replacement could also fail with `Access is denied`.

### 2. High-frequency autosave paths could overlap

- Playback settings autosave and backend log autosave are both driven by debounced draft watchers.
- Before this audit, repeated edits could enqueue overlapping saves against the same backend config file.
- The UX symptom range includes:
  - save flicker
  - stale value bounce-back
  - backend rename conflicts
  - apparent “settings page refresh / conflict” behavior

### 3. Settings page is very large and heavily stateful

- `src/components/jav-library/SettingsPage.vue` is currently over 4,000 lines.
- This increases the chance of:
  - unrelated save flows interfering with each other
  - expensive re-renders
  - harder-to-debug interaction regressions
  - slower future iteration

## Fixes Applied In This Pass

### Backend stability

- Added a process-level serialized write lock around `WriteLibrarySettingsMerge(...)`.
- Strengthened config-file replacement so temp-file rename retries more safely on Windows.
- Added tests covering:
  - replacing an existing `library-config.cfg`
  - concurrent writers preserving all merged keys

### Frontend autosave stability

- Playback settings autosave now runs through a serialized save queue instead of overlapping requests.
- Backend log autosave now uses the same serialized save pattern.

### Playback-adjacent correctness

- HLS fallback to direct playback no longer blindly marks fallback media as browser-direct-playable.
- This reduces the chance of falling back into a silent black-screen state after stream-path changes.

## Current Architecture / Behavior Notes

### Scroll preservation

- The scroll-preserve composable is currently intentionally reduced to a no-op implementation.
- This is a pragmatic stability tradeoff after earlier restore logic proved too aggressive and likely contributed to rendering instability.
- Current state is safer, but it means some “save without scroll movement” polish is intentionally deferred.

### Save strategy inconsistency

- The page currently mixes three models:
  - autosave
  - explicit save button
  - immediate toggle-save actions
- This works, but it is cognitively inconsistent and can make users unsure whether a field has already persisted.

## Performance Opportunities

### 1. Split `SettingsPage.vue` into section components

- Highest-value maintainability optimization.
- Recommended split:
  - `SettingsOverviewSection`
  - `SettingsLibrarySection`
  - `SettingsMetadataSection`
  - `SettingsProxySection`
  - `SettingsPlaybackSection`
  - `SettingsLoggingSection`
  - `SettingsAboutSection`

Benefits:

- smaller reactive graphs per section
- reduced accidental coupling
- easier targeted tests
- simpler save-flow ownership

### 2. Lazy-mount non-visible sections

- Many settings sections can be mounted only when first visited.
- Good targets:
  - provider health / about health
  - proxy diagnostics
  - advanced playback
  - long library management forms

Benefits:

- faster initial settings-page render
- less watcher churn
- lower memory / DOM cost

### 3. Reduce request chatter for text-field autosave

- Current debounced autosave is functional, but text fields such as:
  - backend log directory
  - ffmpeg command
  - native player browser template
  - seek step values
  can still produce frequent save attempts while the user is still editing.

Recommended follow-up:

- keep switches as autosave
- move text-heavy fields to either:
  - blur-save
  - Enter-to-save
  - grouped section save

### 4. Avoid full settings refresh when only local section state changed

- On mount, `refreshSettings()` also refreshes curated-frame counts.
- For revisits to settings, this can be optimized toward stale-while-revalidate or section-level refresh instead of full-page re-hydration.

## UX Opportunities

### 1. Unify persistence mental model

- Pick one of these as the long-term direction:
  - “Everything autosaves”
  - “Each section saves explicitly”
- Current hybrid model is serviceable but ambiguous.

### 2. Add clearer section-level save state

- Recommended status line pattern:
  - `Saving...`
  - `Saved just now`
  - `Save failed`
- This should live at section scope, not only at field scope.

### 3. Surface save-source conflicts more clearly

- If backend persistence fails, show a section toast or inline banner that mentions which section failed, not only a generic field error.

### 4. Differentiate destructive / advanced options more clearly

- Good candidates for stronger visual hierarchy:
  - proxy network behavior
  - HLS / force HLS testing options
  - metadata provider strategy / chain behavior

## Suggested Next Refactor Order

1. Split `SettingsPage.vue` into section components without changing behavior.
2. Move playback + backend log + proxy into section-owned save composables.
3. Convert text-heavy autosave fields to blur-save or explicit section save.
4. Lazy-load low-frequency advanced sections.
5. Add targeted Vitest coverage for section save queues and draft-sync behavior.

## Validation Performed In This Pass

- `pnpm typecheck`
- `go test ./internal/config ./internal/app ./internal/playback ./internal/server`

Vitest could not be completed in this environment because Vite/Vitest startup hit a local Windows `spawn EPERM` process error while loading `vitest.config.ts`.
