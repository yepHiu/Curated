# Library Actor Profile Card Scroll Design

Date: 2026-04-23
Status: Implemented

## Goal

Make the actor profile card truly participate in the same scroll as the movie cards on the normal library content area.

Confirmed scope:

- Keep the actor profile card below the sort/tab row.
- The actor profile card must scroll together with the movie grid, not stay outside the list scroll area.
- Do not change `tags` mode behavior.
- Do not introduce a new page-level scroll container.

## Root Cause

The first layout-only adjustment was insufficient.

`LibraryPage.vue` places the actor profile card above `VirtualMovieMasonry`, but the real scroll owner is not `LibraryPage`. The actual scroll container lives inside `VirtualMovieMasonry.vue`, where `DynamicScroller` uses `overflow-y-auto`.

That means any actor card rendered outside `VirtualMovieMasonry` will never be part of the same scroll stream as the movie cards, even if it appears visually close to them.

## Options

### Option A: Add a header slot to `VirtualMovieMasonry`

Render pre-grid content inside the masonry scroll container, before the virtualized movie chunks.

Pros:

- Keeps one real scroll owner.
- Minimal behavioral change.
- Preserves `useLibraryScrollPreserve`.
- Keeps virtualization logic intact.

Cons:

- Requires a small component API change.

Recommendation: use this option.

### Option B: Move scroll ownership from `VirtualMovieMasonry` to `LibraryPage`

Make the parent page own scrolling and make masonry a pure content renderer.

Pros:

- Centralizes layout ownership.

Cons:

- Higher risk to virtual scrolling.
- More likely to break scroll preservation and sizing assumptions.
- Larger change surface than needed.

Not recommended.

### Option C: Fake shared scrolling with sticky or overlay behavior

Keep the actor card outside the list but simulate the visual result.

Pros:

- Small DOM change.

Cons:

- Does not satisfy the requirement.
- Introduces UI inconsistency.

Not recommended.

## Recommended Design

### 1. Scroll ownership stays in `VirtualMovieMasonry`

Do not move scroll ownership out of the masonry component.

`VirtualMovieMasonry` remains responsible for:

- the scrollable container
- virtualized chunk rendering
- scroll preservation
- back-to-top behavior

### 2. Add a `header` slot to `VirtualMovieMasonry`

Expose a named `header` slot that renders inside the masonry scroll container and before the virtualized movie content.

Implementation details:

- When movies exist, render the slot through `DynamicScroller`'s `before` slot.
- When the list is empty but the header exists, render a fallback scroll container that still contains:
  - the header slot first
  - the empty-state card second

This preserves the shared scrolling behavior even for empty filtered results.

### 3. Pass `ActorProfileCard` through the new slot from `LibraryPage`

`LibraryPage.vue` should stop rendering `ActorProfileCard` as a sibling above the masonry.

Instead, it should pass the actor card into:

- `<VirtualMovieMasonry>`
- via `#header`

This keeps the existing visual order:

1. sort/tab row
2. actor profile card
3. movie cards

But now all three belong to the same content scroll.

### 4. Mode behavior

Normal library content flow:

- render the actor profile card through the masonry header slot when an actor filter is active

Tags mode:

- do not provide header content

Trash mode:

- keep existing layout behavior unchanged

## Verification Strategy

Automated verification:

- `LibraryPage.test.ts` should assert the actor profile card is passed into the masonry header slot.
- `VirtualMovieMasonry.test.ts` should assert the header slot renders before movie items inside the scroller.
- `VirtualMovieMasonry.test.ts` should also cover the empty-list fallback case.

Runtime verification:

- actor card stays below the sort/tab row
- actor card scrolls away together with movie cards
- tags mode remains unchanged
- empty-state layout still shows actor card above the empty card when relevant

## Files Affected

- `src/components/jav-library/LibraryPage.vue`
- `src/components/jav-library/VirtualMovieMasonry.vue`
- `src/components/jav-library/LibraryPage.test.ts`
- `src/components/jav-library/VirtualMovieMasonry.test.ts`
