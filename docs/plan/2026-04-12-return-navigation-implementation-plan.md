# Return Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement navigation intent based return behavior for P1 and P2, centralizing back-target resolution and unifying route builders for browse/detail/player/history/curated frame flows.

**Architecture:** Keep route query as the shareable browse-state carrier, but move return semantics into a single frontend navigation helper layer. `AppShell`, `LibraryView`, `DetailView`, `HistoryView`, and curated-frame playback entry points should consume the same navigation helpers instead of rebuilding `from`/query logic ad hoc.

**Tech Stack:** Vue 3, TypeScript, Vue Router, Vitest

---

### Task 1: Add failing tests for navigation intent helpers

**Files:**
- Create: `src/lib/navigation-intent.test.ts`
- Modify: `src/lib/player-route.test.ts`

- [ ] **Step 1: Write failing tests for centralized back-target resolution**
- [ ] **Step 2: Run `pnpm test -- src/lib/navigation-intent.test.ts src/lib/player-route.test.ts` and verify failures**
- [ ] **Step 3: Implement the minimal helper layer to satisfy the tests**
- [ ] **Step 4: Re-run `pnpm test -- src/lib/navigation-intent.test.ts src/lib/player-route.test.ts` and verify pass**

### Task 2: Move shell/page navigation logic onto the helper layer

**Files:**
- Create: `src/lib/navigation-intent.ts`
- Modify: `src/layouts/AppShell.vue`
- Modify: `src/views/LibraryView.vue`
- Modify: `src/views/DetailView.vue`
- Modify: `src/views/HistoryView.vue`
- Modify: `src/components/jav-library/CuratedFramesLibrary.vue`
- Modify: `src/lib/player-route.ts`

- [ ] **Step 1: Replace `AppShell` inline back-target branching with a shared resolver**
- [ ] **Step 2: Replace per-page route assembly with unified helper builders**
- [ ] **Step 3: Keep behavior-compatible browse context carry-over while making `player` and `detail` consume a shared intent model**
- [ ] **Step 4: Run targeted tests again to confirm no regressions**

### Task 3: Verify the affected frontend surface

**Files:**
- Modify: `docs/plan/2026-04-12-return-navigation-review.md`

- [ ] **Step 1: Update the review doc with implementation status for P1/P2**
- [ ] **Step 2: Run `pnpm test -- src/lib/navigation-intent.test.ts src/lib/player-route.test.ts src/views/HistoryView.test.ts`**
- [ ] **Step 3: Run `pnpm typecheck`**
- [ ] **Step 4: Report actual results with command evidence**
