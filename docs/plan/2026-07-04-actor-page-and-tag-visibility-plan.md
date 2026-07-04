# Actor Page and Actor Tag Visibility Plan

## Background

The actor tag system exists in the backend, service contract, mock adapter, and UI. Current user feedback is that actor-specific tags have low day-to-day value, while actor profile and actor filmography deserve a stronger dedicated experience.

The preferred direction is to hide actor tag UI first, without deleting backend storage or API behavior. This keeps existing data recoverable and avoids unnecessary backend churn.

## Requirements Summary

- Hide actor user tags from actor-facing UI for now.
- Keep backend actor tag endpoints, DTO fields, database tables, and service methods intact unless a future cleanup explicitly targets them.
- Remove or hide tag editing and tag-filter affordances from actor cards and actor profile cards.
- Avoid showing actor tags in the actor profile card that appears above actor-filtered movie lists.
- Improve actor library cards visually, especially by making actor avatars larger and less cramped.
- Avoid relying on the general Library page actor filter as the long-term actor filmography experience.
- Introduce, in a later slice, a dedicated actor detail route that presents actor information first and that actor's movies below it.
- Keep actor filmography paginated or independently scoped, rather than merely adding another filter control to the all-movies page.

## Current Implementation Notes

- `src/components/jav-library/ActorsPage.vue` currently supports `actorTag` route query, active actor-tag filter banners, and passes tag suggestions/events to actor cards.
- `src/components/jav-library/ActorLibraryCard.vue` currently renders actor user tags, supports adding/removing them, and clicking a tag filters the actor library.
- `src/components/jav-library/ActorLibraryCard.vue` currently uses a small `size-12` square avatar. Actor portraits feel too small and visually cramped in the current card.
- `src/components/jav-library/ActorProfileCard.vue` currently renders actor user tags on the profile card shown in actor-filtered Library views, and supports editing/removing/filtering those tags.
- `src/views/LibraryView.vue` currently resolves `actor=` or exact actor search into an actor profile card plus a filtered version of the main movie grid.
- `src/api/types.ts`, `src/api/endpoints.ts`, `src/services/contracts/library-service.ts`, and the Web/Mock adapters expose actor tag methods and fields; these should stay in place for this change.

## Recommended Approach

### Phase 1: Hide Actor Tags in the UI

This is the first implementation slice.

- Remove actor tag display, add/remove controls, suggestion dropdowns, and tag-filter click actions from `ActorLibraryCard`.
- Remove actor tag display and editing controls from `ActorProfileCard`.
- Remove `actorTag` filter banner and actor-tag event handling from `ActorsPage`.
- Use the space freed by actor-tag controls to improve `ActorLibraryCard` visual hierarchy.
- Enlarge actor avatars from the current small square treatment to a more portrait-forward treatment. Recommended direction: a compact card with a larger rounded portrait area, enough to recognize the actor without turning the actor library into a huge hero grid.
- Keep cards dense enough for browsing: do not create oversized marketing-style cards, nested cards, or large decorative footers.
- Keep actor list search (`actorsQ`) as-is. Backend may still include tags in search until a later decision, but the UI should no longer promote actor tags.
- Keep service methods such as `patchActorUserTags` and DTO fields such as `userTags` untouched.
- Update tests so they assert tags are not rendered in actor cards/profile cards, and that the actor card keeps the avatar visible with the expected image/fallback behavior.
- Leave locale keys in place for now unless they become directly unused by lint or tests. This keeps rollback cheap.

Trade-off: this is conservative and reversible. Some hidden backend behavior remains, but it avoids risky migrations and large API cleanup.

#### Actor Card Visual Options

Recommended: **larger portrait-led compact card**.

- Replace the current tiny square avatar emphasis with a larger portrait block, likely around `size-20` to `size-24` visually on desktop while staying responsive on mobile.
- Keep actor name and movie count close to the portrait, with clear line-clamp behavior.
- Use existing theme tokens, `Avatar`, `Card`, and restrained borders/shadows.
- This is the best balance between recognizability and browse density.

Alternative A: **simple square avatar upscale**.

- Increase `size-12` to `size-16` or `size-20` while keeping the current horizontal header layout.
- Lowest implementation risk, but less improvement for portrait photos because the crop still feels cramped.

Alternative B: **large top media tile**.

- Put a large image area across the card top, with actor name/count below.
- Strongest visual impact, but it reduces browsing density and may make the actor page feel too much like a promotional gallery.

The preferred first pass is the recommended portrait-led compact card, then validate with desktop and mobile screenshots.

### Phase 2: Add Dedicated Actor Detail Page

This is the current implementation slice. The product decision is to stop treating actor filmography as a thin `library?actor=...` filter wrapper. The new route should make the actor the page subject, with that actor's movies presented as page content below the actor profile.

- Add a route such as `/actors/:name` or `/actor/:name` with a stable encoded actor name.
- Make actor cards navigate to this actor detail page instead of pushing to `library?actor=...`.
- Render actor information at the top using the existing actor profile loading/scrape logic from `ActorProfileCard`, preferably extracted into a composable or smaller profile component to avoid duplicating fetch/poll behavior.
- Render that actor's movies below the profile in a page-owned movie section.
- Keep detail/player navigation browse context actor-aware so returning from detail/player lands back on the actor page, not the generic Library page.
- Decide whether the first version can use already-loaded `libraryService.movies` filtered by actor, or whether it should call the backend movie list with `actor=` and pagination. The product direction prefers an independently scoped actor page; backend pagination can be a follow-up if the current service contract does not already expose list pagination cleanly.

Trade-off: reusing the existing movie grid is efficient, but the route, page layout, and navigation state should be actor-page specific so it does not feel like a thin Library filter wrapper.

#### Phase 2 Implementation Plan

**Goal:** build an actor-owned detail route now, while keeping backend changes out of scope.

**Architecture:** add a child route under the existing actor namespace, `actors/:actorName`, backed by a new `ActorDetailView` and `ActorDetailPage`. The actor page can reuse `ActorProfileCard` for the profile area and `VirtualMovieMasonry` for the movie grid, but it should own its own route state, title, empty state, and navigation rather than mounting `LibraryView`.

**Files:**

- Modify `src/router/index.ts` to add `name: "actor-detail"` at `path: "actors/:actorName"`.
- Modify `src/components/jav-library/ActorLibraryCard.vue` so the card click pushes `actor-detail` with `actorName`.
- Create `src/views/ActorDetailView.vue` as the route wrapper that reads and decodes the route param.
- Create `src/components/jav-library/ActorDetailPage.vue` to render profile, movie section, empty state, and movie interactions.
- Add focused tests for `ActorLibraryCard`, router registration, and `ActorDetailPage`.
- Update this plan document as the durable reminder that backend actor-movie endpoints remain a later option.

**TDD checkpoints:**

1. Add a failing `ActorLibraryCard` test that expects actor card clicks to navigate to `actor-detail`.
2. Add a failing router test that resolves `/actors/Mina%20Kaze` to `actor-detail`.
3. Add a failing `ActorDetailPage` test that filters movies by exact actor and passes them to `VirtualMovieMasonry`.
4. Implement the route, card navigation, view wrapper, and page component minimally.
5. Verify focused tests, then run `pnpm typecheck`, `pnpm lint`, and browser QA on desktop and mobile.

**First-version behavior:**

- Actor cards open `/actors/<encoded actor name>`.
- Actor detail pages show actor profile first.
- The filmography section below shows only movies whose `actors` include that exact actor name.
- The first version reuses the already-loaded service movie cache, matching current frontend data boundaries and avoiding backend churn.
- Movie detail/player buttons from this page may initially use existing detail/player routes; actor-specific return routing can be refined as a follow-up if the current navigation intent helpers require broader changes.

**Not in this slice:**

- No backend route changes.
- No actor tag cleanup.
- No new filter drawer or reuse of the entire Library page.
- No batch management on the actor filmography page in the first pass.

### Phase 3: Optional Backend/API Cleanup

This is explicitly not part of the initial change.

- If actor tags stay unused for a while, decide whether to deprecate UI-only, API-only, or database-level support.
- Remove actor tag query support only after confirming there is no need to recover existing actor tag data.
- Update `project-facts.mdc`, `API.md`, `CLAUDE.md`, and public docs only if backend/API behavior changes.

### Phase 3: Optional Dedicated Actor-Movie Backend API

This is recorded as a future reminder, not current scope.

- Add an explicit backend endpoint such as `GET /api/library/actors/{name}/movies` or `GET /api/library/actors/movies?name=...`.
- Return a paged movie response with actor-specific totals, limit, offset, and sort options.
- Let the frontend actor detail page load filmography independently from the global movie cache.
- Keep the existing `GET /api/library/movies?actor=...` behavior for compatibility until callers migrate.
- Revisit docs (`project-facts.mdc`, `API.md`, `CLAUDE.md`, README API summaries) only when this backend contract is actually introduced.

## Open Decision

The main implementation decision before Phase 2 is route shape:

- Recommended: `/actors/:name` reuses the existing actor namespace and reads as a detail child of the actor library.
- Alternative: `/actor/:name` is singular and semantically precise, but adds a new top-level route family.

## Acceptance Criteria

### Phase 1

- Actor library cards no longer show actor tags or tag edit controls.
- Actor library cards have visibly larger, better-proportioned actor portraits than the current `size-12` avatar.
- Actor cards remain compact and scannable across mobile and desktop grids.
- Actor profile cards no longer show actor tags or tag edit controls.
- Actor library no longer exposes actor-tag filtering UI.
- Existing backend actor tag APIs remain untouched.
- Existing movie user tags and metadata tags are unaffected.
- Unit tests cover that actor tags are hidden from actor UI.

### Phase 2

- Clicking an actor opens a dedicated actor detail page.
- The actor detail page shows actor identity/profile information and the actor's movies below it.
- The route is shareable and reload-safe.
- Detail/player navigation from actor filmography preserves an actor-page return path.
- The page does not present itself as just the generic all-movies page with an actor filter.

## Verification Plan

- Run focused component tests for `ActorLibraryCard`, `ActorProfileCard`, `ActorsPage`, and new actor detail components.
- For the actor card visual refresh, verify rendered desktop and mobile screenshots: portraits should not look tiny, names/counts should not overlap, and fallback initials should still look intentional.
- Run router/navigation tests for the new actor route and browse-return behavior.
- Run `pnpm typecheck`.
- Run `pnpm lint`.
- Run focused Vitest tests first, then broader `pnpm test` if route/navigation behavior changes.
