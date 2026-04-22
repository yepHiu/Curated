# Library Actor Profile Card Scroll Design

Date: 2026-04-23
Status: Approved for spec review

## Goal

Adjust the actor profile card placement in the library page so that, when an actor filter is active, the actor profile card scrolls together with the main content area.

Confirmed scope:

- Only change the normal library page layout.
- Do not change `tags` mode.
- Do not change `trash` mode.
- Place the actor profile card below the sort/tab row that currently shows the "入库时间 / 发售日期 / 评分" style switching controls.
- The actor profile card should remain above the movie grid.

## Current Context

Today, `ActorProfileCard` is rendered near the top of `LibraryPage.vue`, before the main mode-specific content blocks. That makes it visually feel separate from the main content flow.

In the normal library mode, the page structure is effectively:

1. actor profile card
2. optional studio filter bar
3. sort/tab row
4. movie masonry

This means the actor card appears higher than the sort controls, which is not the intended reading order when the actor card is contextual content for the current filtered result set.

## Options

### Option A: Move the actor profile card into the normal library content flow

Render `ActorProfileCard` only inside the normal library mode section, directly below the sort/tab row and above `VirtualMovieMasonry`.

Pros:

- Matches the desired scroll behavior naturally.
- Keeps the actor card visually attached to the movie results it explains.
- Smallest structural change.
- Lowest risk to `VirtualMovieMasonry` sizing and scroll-preserve behavior.

Cons:

- Requires one small conditional layout split in `LibraryPage.vue`.

Recommendation: use this option.

### Option B: Keep the card where it is but change scroll container boundaries

Rebuild the layout so the current top area and the content area share one larger scrolling container.

Pros:

- Preserves current DOM order.

Cons:

- Higher layout risk.
- More likely to affect masonry height calculations and scroll preserve behavior.
- Harder to reason about than simply moving the card.

### Option C: Make the card sticky under the sort row

Keep the card visible while the list scrolls.

Pros:

- Actor info stays available.

Cons:

- Does not match the requested behavior.
- Consumes vertical space.
- More intrusive on smaller screens.

Not recommended.

## Recommended Design

### 1. Placement

In `LibraryPage.vue`:

- Remove the top-level `ActorProfileCard` placement from the shared page header flow.
- Re-render it only in the normal library branch.
- Insert it after the normal-library sort/tab row and before the movie masonry area.

Resulting normal library flow:

1. optional studio filter bar
2. sort/tab row
3. actor profile card
4. movie masonry

This makes the actor card part of the same content stream as the filtered movie results.

### 2. Mode behavior

Normal library mode:

- show actor card below the sort/tab row when `activeActorTrimmed` is non-empty

Tags mode:

- no placement change
- keep current tags browsing card behavior unchanged

Trash mode:

- no placement change
- actor card remains absent from this layout treatment

### 3. Scrolling behavior

No new scroll container should be introduced.

The actor card should simply become a normal block in the same vertical flow above `VirtualMovieMasonry`. This gives the desired "scroll with the content" behavior without changing the scroll ownership model.

That is the key design choice to avoid side effects in:

- masonry sizing
- preserved list scroll position
- existing page height assumptions

### 4. Risk control

Files touched should stay narrow:

- `src/components/jav-library/LibraryPage.vue`
- possibly one focused component test if coverage is added
- no change to `ActorProfileCard.vue` behavior
- no change to `LibraryView.vue` filtering logic

This is intentionally a composition-only change, not a data or behavior change.

### 5. Verification

Manual verification target:

- in normal library mode with `actor=` active, the sort/tab row appears first
- the actor profile card appears directly below it
- scrolling the page scrolls the actor profile card away together with the movie list
- `tags` mode and `trash` mode stay visually unchanged

Automated verification target:

- a focused `LibraryPage` layout test can assert that `ActorProfileCard` is rendered in the normal library content section and not before the shared top structure

## Out of Scope

- redesigning the actor profile card itself
- sticky actor profile behavior
- changing sort/tab semantics
- changing actor card behavior in `tags` or `trash` mode
- refactoring scroll-preserve logic
