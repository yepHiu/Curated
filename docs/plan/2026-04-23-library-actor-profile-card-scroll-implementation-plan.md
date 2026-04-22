# Library Actor Profile Card Scroll Implementation Plan

Date: 2026-04-23
Status: Implemented

## Goal

Fix the actor profile card so it scrolls together with the movie cards by rendering it inside the masonry component's real scroll container.

## Architecture Summary

Keep `VirtualMovieMasonry` as the sole scroll owner.

Implementation uses a narrow component seam:

- `LibraryPage.vue` passes actor card content into `VirtualMovieMasonry` via a new `#header` slot
- `VirtualMovieMasonry.vue` renders that slot inside the true scroll container
- when movies exist, the slot is rendered through `DynamicScroller`'s `before` slot
- when movies are empty, the slot is rendered in a fallback scroll shell above the empty-state card

## File Map

- Modify: `src/components/jav-library/LibraryPage.vue`
- Modify: `src/components/jav-library/VirtualMovieMasonry.vue`
- Modify: `src/components/jav-library/LibraryPage.test.ts`
- Add: `src/components/jav-library/VirtualMovieMasonry.test.ts`

## Implementation Steps

### 1. Add failing tests

- Update `LibraryPage.test.ts` to require that `ActorProfileCard` is passed into the masonry header area rather than rendered as an external sibling.
- Add `VirtualMovieMasonry.test.ts` to require:
  - header slot renders before movie items inside the scroller
  - header slot renders above the empty state when there are no movies

### 2. Add header-slot support to `VirtualMovieMasonry`

- Add a named `header` slot.
- Detect whether header content exists.
- For non-empty lists:
  - render the header through `DynamicScroller` `#before`
- For empty lists with header content:
  - render a scrollable fallback container
  - place header first
  - place the empty state card second

### 3. Route actor profile card through the new seam in `LibraryPage`

- Remove the direct sibling placement of `ActorProfileCard`.
- Pass `ActorProfileCard` into `VirtualMovieMasonry` using `#header`.
- Keep `tags` mode behavior unchanged by not providing header content there.

### 4. Verification

Run:

```bash
pnpm test -- src/components/jav-library/LibraryPage.test.ts src/components/jav-library/VirtualMovieMasonry.test.ts src/components/jav-library/ActorProfileCard.test.ts
pnpm typecheck
pnpm lint
```

Expected:

- all focused tests pass
- typecheck passes
- lint passes

## Result

The actor profile card is now part of the same scrolling content stream as the movie grid, while preserving the existing virtualization and scroll-preservation architecture.
