# Player Shortcuts And Curated Keybinding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the player arrow-key volume hotkey conflict with focused sliders, and add a single-key customizable curated-frame capture shortcut in Settings -> Curated.

**Architecture:** Extract keyboard-policy logic into a small shared utility so the player and settings UI use the same reserved-key and validation rules. Keep the curated capture shortcut frontend-local in curated settings storage, then wire the player to read the configured key instead of hard-coding `C`.

**Tech Stack:** Vue 3, TypeScript, Vitest, localStorage-backed UI settings, reka-ui Slider.

---

### Task 1: Shared Shortcut Policy Utilities

**Files:**
- Create: `src/lib/player-shortcuts.ts`
- Test: `src/lib/player-shortcuts.test.ts`

- [ ] Write failing tests for reserved curated keys, accepted single-key formats, display labels, and slider-target filtering.
- [ ] Run `pnpm test -- src/lib/player-shortcuts.test.ts` and verify the new tests fail for missing exports / missing behavior.
- [ ] Implement the minimal shared helpers used by both player and settings.
- [ ] Re-run `pnpm test -- src/lib/player-shortcuts.test.ts` and verify it passes.

### Task 2: Curated Shortcut Storage

**Files:**
- Modify: `src/lib/curated-frames/settings-storage.ts`
- Test: `src/lib/curated-frames/settings-storage.test.ts`

- [ ] Add failing tests for default curated capture key, persistence, normalization, and reset behavior.
- [ ] Run `pnpm test -- src/lib/curated-frames/settings-storage.test.ts` and verify failure first.
- [ ] Implement the minimal storage helpers around the curated capture key while preserving existing save-mode behavior.
- [ ] Re-run `pnpm test -- src/lib/curated-frames/settings-storage.test.ts` and verify it passes.

### Task 3: Settings -> Curated Shortcut UI

**Files:**
- Create: `src/components/jav-library/settings/SettingsCuratedShortcutSection.vue`
- Create: `src/components/jav-library/settings/SettingsCuratedShortcutSection.test.ts`
- Modify: `src/components/jav-library/SettingsPage.vue`
- Modify: `src/locales/en.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/ja.json`

- [ ] Write failing component tests for capture mode, valid save, reserved-key rejection, cancel with `Escape`, and reset-to-default.
- [ ] Run `pnpm test -- src/components/jav-library/settings/SettingsCuratedShortcutSection.test.ts` and verify failure first.
- [ ] Implement the shortcut section component and mount it inside the curated settings card.
- [ ] Re-run `pnpm test -- src/components/jav-library/settings/SettingsCuratedShortcutSection.test.ts` and verify it passes.

### Task 4: Player Hotkey Fix And Configured Capture Shortcut

**Files:**
- Modify: `src/components/jav-library/PlayerPage.vue`
- Test: `src/lib/player-shortcuts.test.ts`

- [ ] Extend failing tests, if needed, to cover the exact player hotkey policy: slider-focused events must not trigger global arrow-key volume shortcuts.
- [ ] Update the player keydown handler to ignore slider-focused targets and to use the configured curated capture shortcut.
- [ ] Verify focused tests still pass.

### Task 5: Verification

**Files:**
- No new files expected

- [ ] Run `pnpm typecheck`.
- [ ] Run `pnpm lint`.
- [ ] Run `pnpm test`.
- [ ] If docs/UI wording changed materially, confirm no additional public docs updates are required beyond this task’s local UI copy.
