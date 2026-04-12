# Homepage Portal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a real homepage portal to Curated, make it the default `/` entry, and keep its content driven by deterministic library data instead of hard-coded view logic.

**Architecture:** Introduce a dedicated homepage view and homepage-specific UI components under `src/components/jav-library`, but keep selection and recommendation logic in a pure `src/lib` module so it can be tested independently. The homepage should read from existing library, playback-progress, and played-movies state, while routing and sidebar navigation remain aligned with the current AppShell structure.

**Tech Stack:** Vue 3 SFCs, TypeScript, vue-router, vue-i18n, shadcn-vue primitives, Vitest, existing local storage helpers and library service adapters.

---

### Task 1: Homepage Data Assembly

**Files:**
- Create: `src/lib/homepage-portal.ts`
- Create: `src/lib/homepage-portal.test.ts`
- Read: `src/domain/movie/types.ts`
- Read: `src/lib/playback-progress-storage.ts`
- Read: `src/lib/played-movies-storage.ts`
- Read: `src/lib/random-sample.ts`

- [x] **Step 1: Write the failing test**

Write tests for:
- daily hero chooses exactly 8 movies with deterministic date-seeded ordering
- continue watching only includes unfinished progress rows
- recommendation scoring favors favorite / user-rated / recent-signal matches over unrelated movies
- recent imports are sorted by `addedAt` descending

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/lib/homepage-portal.test.ts`
Expected: FAIL because `src/lib/homepage-portal.ts` does not exist yet.

- [x] **Step 3: Write minimal implementation**

Implement pure helpers that:
- normalize and sort candidate movie pools
- build homepage sections from existing movie records plus playback / played state
- keep deterministic behavior from date seed and stable tie-breaking

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/lib/homepage-portal.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/lib/homepage-portal.ts src/lib/homepage-portal.test.ts
git commit -m "feat: add homepage portal data assembly"
```

### Task 2: Homepage View Composition

**Files:**
- Create: `src/views/HomeView.vue`
- Create: `src/components/jav-library/HomepagePortal.vue`
- Create: `src/components/jav-library/HomeHeroCarousel.vue`
- Create: `src/components/jav-library/HomeSectionRow.vue`
- Create: `src/components/jav-library/HomeContinueRow.vue`
- Read: `src/components/jav-library/MovieCard.vue`
- Read: `src/components/jav-library/MediaStill.vue`
- Read: `src/layouts/AppShell.vue`

- [x] **Step 1: Write the failing test**

Add a view-level smoke test that mounts `HomeView` with mocked library and playback state and asserts:
- the hero region renders
- the progress rail renders 8 segments
- recent and recommendation sections render expected headings

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/views/HomeView.test.ts`
Expected: FAIL because `HomeView.vue` and homepage components do not exist yet.

- [x] **Step 3: Write minimal implementation**

Build the homepage as:
- one full-bleed hero region with internal text column
- supporting section rows below the hero
- section components that reuse existing `MovieCard` or `MediaStill` rather than inventing a second card system

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/views/HomeView.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/views/HomeView.vue src/views/HomeView.test.ts src/components/jav-library/HomepagePortal.vue src/components/jav-library/HomeHeroCarousel.vue src/components/jav-library/HomeSectionRow.vue src/components/jav-library/HomeContinueRow.vue
git commit -m "feat: add homepage portal view"
```

### Task 3: Router And Sidebar Integration

**Files:**
- Modify: `src/router/index.ts`
- Modify: `src/domain/library/types.ts`
- Modify: `src/components/jav-library/AppSidebar.vue`
- Modify: `src/layouts/AppShell.vue`

- [x] **Step 1: Write the failing test**

Add a router / navigation test that verifies:
- `/` resolves to homepage instead of redirecting to library
- sidebar includes the homepage item
- homepage is treated as a primary browse surface for header behavior

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/lib/homepage-routing.test.ts`
Expected: FAIL because route names and sidebar metadata are not updated yet.

- [x] **Step 3: Write minimal implementation**

Update:
- `AppPage` union with `"home"`
- router root child route to `HomeView`
- sidebar browse group so homepage appears first
- shell header logic so homepage does not show the back button like a detail page

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/lib/homepage-routing.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/router/index.ts src/domain/library/types.ts src/components/jav-library/AppSidebar.vue src/layouts/AppShell.vue src/lib/homepage-routing.test.ts
git commit -m "feat: wire homepage into app navigation"
```

### Task 4: Localization And Finish Pass

**Files:**
- Modify: `src/locales/en.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/ja.json`
- Modify: `docs/plan/2026-04-12-homepage-portal-design.md`

- [x] **Step 1: Write the failing test**

Extend homepage smoke coverage so translated keys used by `HomeView` are asserted to exist via rendered output for the default locale.

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- src/views/HomeView.test.ts`
Expected: FAIL due to missing locale keys.

- [x] **Step 3: Write minimal implementation**

Add locale entries for:
- sidebar / nav homepage label
- hero labels
- section labels and empty states
- recommendation rationale and continue-watching wording

- [x] **Step 4: Run test to verify it passes**

Run: `pnpm test -- src/views/HomeView.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/locales/en.json src/locales/zh-CN.json src/locales/ja.json docs/plan/2026-04-12-homepage-portal-design.md
git commit -m "feat: localize homepage portal"
```

### Final Verification

- [x] Run homepage lib tests: `pnpm test -- src/lib/homepage-portal.test.ts`
- [x] Run homepage view tests: `pnpm test -- src/views/HomeView.test.ts`
- [x] Run targeted navigation tests: `pnpm test -- src/lib/homepage-routing.test.ts`
- [x] Run typecheck: `pnpm typecheck`
- [x] Run full frontend tests if targeted tests stay green: `pnpm test`

### Incremental Refinement: Hero Carousel Motion

- [x] Replace the single-image hero body with a slide track that keeps the active frame centered while exposing previous and next frames on the left and right edges.
- [x] Keep rail navigation below the hero container and drive the same animated horizontal transform for autoplay and manual selection.
- [x] Move visible movie metadata into each frame, limiting the frame copy to `title` and `code` so long titles truncate cleanly without pushing the layout.
- [x] Cover the motion structure with `src/components/jav-library/HomeHeroCarousel.test.ts` and re-run homepage view and routing regressions.
- [x] Remove the visible outer hero card container so the slide track uses a full-width stage and begins from the page edges instead of scrolling inside a narrow centered box.
- [x] Make the hero carousel loop seamlessly by prepending and appending clone slides, then snapping the track back to the canonical index after wraparound transitions complete.
- [x] Restore the progress rail to a smaller centered footprint instead of stretching it across the entire full-width hero stage.
- [x] Fix wraparound stutter by deriving slide visual state from the logical movie index instead of the physical track index, so the clone slide and canonical slide keep the same active/adjacent styling during the post-wrap snap.
- [x] Split hero transition timing so autoplay can stay smoother and slower while manual rail / preview clicks react faster.
- [x] Rebalance hero preview depth for light mode by removing hard-coded black drop shadows and using theme-aware shadows plus brightness / saturation falloff on side previews.
- [x] Make homepage taste radar chips clickable and route them into the appropriate browse filters (`tags` for exact tags, `library` for actor/studio filters).
