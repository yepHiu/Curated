# Active Playback Sidebar Return Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task in this session. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a sidebar "继续播放" entry that appears after the user leaves the player and returns to the same movie at the last known position.

**Architecture:** Use a small module-scoped Vue composable as the single active playback session boundary. `PlayerPage` writes the current playback session, `AppSidebar` consumes it and renders an expanded card or compact icon entry above backend status.

**Tech Stack:** Vue 3 Composition API, vue-router route targets, TypeScript, Vitest, shadcn-vue primitives already present in the repo.

---

## File Structure

- Create `src/composables/use-active-playback-session.ts`: shared active playback session state, route target generation, stale/end visibility rules, manual dismiss.
- Create `src/composables/use-active-playback-session.test.ts`: unit tests for state saving, route generation, hidden ended/near-end sessions, dismiss behavior.
- Modify `src/components/jav-library/AppSidebar.vue`: render expanded/compact "继续播放" entry above backend status.
- Modify `src/components/jav-library/AppSidebar.test.ts`: assert sidebar rendering and current-player suppression.
- Modify `src/components/jav-library/PlayerPage.vue`: update active session on descriptor load, play/pause/time updates, flush, end/error cleanup.
- Modify `src/components/jav-library/PlayerPage.loading.test.ts`: verify descriptor load and time update write active playback state.
- Modify locale JSON files under `src/locales/`: add sidebar playback labels in `zh-CN`, `en`, and `ja`.

## Tasks

### Task 1: Shared Active Playback Session

**Files:**
- Create: `src/composables/use-active-playback-session.ts`
- Test: `src/composables/use-active-playback-session.test.ts`

- [ ] **Step 1: Write failing tests**

Cover these behaviors:

- saving a session exposes `activePlaybackSession`
- generated route preserves the original player query and overwrites `t` / `autoplay`
- sessions under 5 seconds and near the end are hidden
- `dismissActivePlaybackSession()` hides the current session until a newer update arrives
- `clearActivePlaybackSession(movieId)` clears only the matching session

Run:

```powershell
pnpm vitest run --configLoader native --pool threads src/composables/use-active-playback-session.test.ts
```

Expected: fail because the composable does not exist.

- [ ] **Step 2: Implement minimal composable**

Add types and functions:

- `type ActivePlaybackStatus = "playing" | "paused" | "waiting" | "ended" | "error"`
- `updateActivePlaybackSession(input)`
- `clearActivePlaybackSession(movieId?)`
- `dismissActivePlaybackSession(movieId?)`
- `useActivePlaybackSession()`

The public session should include `resumeRouteTarget` with `name: "player"`, `params.id`, `query.autoplay = "1"`, and `query.t` formatted to integer seconds.

- [ ] **Step 3: Verify tests pass**

Run:

```powershell
pnpm vitest run --configLoader native --pool threads src/composables/use-active-playback-session.test.ts
```

Expected: pass.

### Task 2: Sidebar UI

**Files:**
- Modify: `src/components/jav-library/AppSidebar.vue`
- Modify: `src/components/jav-library/AppSidebar.test.ts`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/en.json`
- Modify: `src/locales/ja.json`

- [ ] **Step 1: Write failing sidebar tests**

Add tests that:

- render expanded `data-active-playback-card` when an active session exists and route is not the same player
- render compact `data-active-playback-compact` in compact mode
- hide the entry when current route is `player` for the same movie

Run:

```powershell
pnpm vitest run --configLoader native --pool threads src/components/jav-library/AppSidebar.test.ts
```

Expected: fail because the sidebar does not consume the active playback composable yet.

- [ ] **Step 2: Implement sidebar rendering**

Use existing `RouterLink`, `Button`, `Separator`, lucide icons, semantic tokens, and data attributes:

- expanded card: `data-active-playback-card`
- compact icon: `data-active-playback-compact`
- close button: `data-active-playback-dismiss`

Keep it above backend status.

- [ ] **Step 3: Add locale labels**

Add translation keys under `nav`:

- `continuePlayback`
- `continuePlaybackAt`
- `continuePlaybackAria`
- `dismissContinuePlayback`

- [ ] **Step 4: Verify sidebar tests pass**

Run:

```powershell
pnpm vitest run --configLoader native --pool threads src/components/jav-library/AppSidebar.test.ts
```

Expected: pass.

### Task 3: Player Wiring

**Files:**
- Modify: `src/components/jav-library/PlayerPage.vue`
- Modify: `src/components/jav-library/PlayerPage.loading.test.ts`

- [ ] **Step 1: Write failing PlayerPage test**

Mock `@/composables/use-active-playback-session` and assert:

- `updateActivePlaybackSession` is called after a playback descriptor resolves
- after `loadedmetadata` and `timeupdate`, the update includes current playback position and duration
- after `ended`, `clearActivePlaybackSession(movieId)` is called

Run:

```powershell
pnpm vitest run --configLoader native --pool threads src/components/jav-library/PlayerPage.loading.test.ts
```

Expected: fail because `PlayerPage` does not write active playback state.

- [ ] **Step 2: Implement PlayerPage active session updates**

Import:

```ts
import {
  clearActivePlaybackSession,
  updateActivePlaybackSession,
} from "@/composables/use-active-playback-session"
```

Update on descriptor load, play, pause, waiting, time update/flush, error, and ended. Clear on end/error; do not clear merely on unmount because the sidebar entry needs to remain after route navigation.

- [ ] **Step 3: Verify PlayerPage tests pass**

Run:

```powershell
pnpm vitest run --configLoader native --pool threads src/components/jav-library/PlayerPage.loading.test.ts
```

Expected: pass.

### Task 4: Integrated Verification

**Files:**
- No new files unless tests expose a real issue.

- [ ] **Step 1: Run focused tests**

```powershell
pnpm vitest run --configLoader native --pool threads src/composables/use-active-playback-session.test.ts src/components/jav-library/AppSidebar.test.ts src/components/jav-library/PlayerPage.loading.test.ts
```

Expected: pass.

- [ ] **Step 2: Run typecheck**

```powershell
pnpm typecheck
```

Expected: pass.

- [ ] **Step 3: Review diff**

```powershell
git diff -- src/composables/use-active-playback-session.ts src/composables/use-active-playback-session.test.ts src/components/jav-library/AppSidebar.vue src/components/jav-library/AppSidebar.test.ts src/components/jav-library/PlayerPage.vue src/components/jav-library/PlayerPage.loading.test.ts src/locales/zh-CN.json src/locales/en.json src/locales/ja.json docs/plan/2026-05-01-active-playback-sidebar-return-implementation-plan.md
```

Expected: only this feature and the implementation plan are present.

## Self Review

- Spec coverage: matches方案 B, sidebar placement, compact/expanded/mobile behavior, route target preservation, dismiss, and end cleanup.
- Placeholder scan: no placeholders.
- Type consistency: route target, movie id, and status names are defined in Task 1 before use elsewhere.
