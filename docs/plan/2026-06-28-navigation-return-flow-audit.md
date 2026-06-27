# Curated Navigation Return Flow Audit

Date: 2026-06-28

## Background

User-reported flow:

1. Open a movie detail page.
2. Click an actor block in the cast area.
3. Curated navigates to the library with an exact `actor=` filter.
4. The app shell no longer offers an application-level way back to the movie detail page.
5. The user must clear the actor filter, return to the full library, and find/open the same movie again.

This report focuses on application-level return affordances inside Curated, not browser-native history.

## Root Cause

The current navigation model separates:

- browse context: `browse`, `q`, `tag`, `actor`, `studio`, `tab`, `selected`
- return intent: `back`

Movie detail and player routes use this model through `src/lib/navigation-intent.ts`.

The broken path is in `src/views/DetailView.vue`:

- `browseByActor`
- `browseByTag`
- `browseByStudio`

These handlers push a library route with `mergeLibraryQuery(...)`. That helper intentionally removes transient navigation keys such as `browse`, `back`, `from`, `autoplay`, and `t`. The resulting filtered library route has no `back=detail` intent.

Then `src/layouts/AppShell.vue` treats `library`, `favorites`, `tags`, and `actors` as primary browse routes through `isPrimaryBrowseRoute`, so it hides the header back button on the filtered library page.

Result: the URL history may still contain the detail page, but the Curated UI does not expose a clear return path, which is especially problematic in the desktop shell and gamepad-oriented flows.

## Healthy Existing Flows

- Library/favorites/tags/trash -> detail: detail route receives `browse` and `selected`; header/Escape returns to the original browse route.
- Library/favorites/tags/trash -> player: player receives `back=browse`; player back returns to the browse route.
- Detail -> player: player receives `back=detail`; player back returns to the detail page.
- Home -> detail/player: routes receive `back=home`; app back returns to home.
- History -> player: player receives `back=history`; app back returns to history.
- Curated frames -> player: player receives `back=curated-frames`; app back returns to curated frames.

## Unfriendly Or Risky Flows

1. Detail -> actor/tag/studio filtered library

   This is the reported issue. It should keep a return intent to the originating detail page.

2. Home taste chips -> actor/tag/studio filtered library

   `HomeView.vue` opens taste drill-downs as plain filtered library routes. This may be acceptable as a browse transition, but it is inconsistent with home -> detail/player, which preserves `back=home`.

3. Actor library card -> actor filmography

   `ActorLibraryCard.vue` opens the library with `actor=` using an empty source query, so the filmography page cannot return to the actors page or the actors page search/tag filter state.

4. Actor filmography profile card -> actor-tag filtered actors page

   `ActorProfileCard.vue` can jump from a library actor filmography into `/actors?actorTag=...` without preserving the filmography as a return target.

5. Notification center source routes

   `NotificationCenter.vue` pushes raw source route strings. This is acceptable for system notifications, but if a source opens a filtered page, the previous in-app context is not represented explicitly.

## Recommended Design

Implement a small "route-origin return intent" extension rather than relying on browser history.

For the first fix slice:

1. Add a helper in `src/lib/navigation-intent.ts` to build filtered-library routes from a detail page:

   - Keep the existing target filter behavior.
   - Preserve the source browse mode.
   - Preserve the source movie id as `selected`.
   - Add `back=detail`.

2. Update `DetailView.vue` `browseByActor`, `browseByTag`, and `browseByStudio` to use that helper.

3. Update `AppShell.vue` so a primary browse route with `back=detail` and a usable `selected` movie id shows the same header back button as non-primary routes.

4. Keep filter clear behavior unchanged. Clearing the actor/tag/studio filter should not be required for returning to detail.

5. Add focused tests:

   - `navigation-intent.test.ts`: filtered route from detail carries `back=detail`, source browse mode, selected movie id, and the exact filter.
   - `AppShell.test.ts`: a library route with `back=detail&selected=movie-1` renders a detail back target.
   - `DetailView.test.ts`: emitting `browseByActor` pushes the filtered route with a detail return intent.

Second slice after the first fix is validated:

1. Decide whether home taste chips should use `back=home`.
2. Decide whether actor filmography should use `back=actors`, which would require adding `actors` to the supported `NavigationBackTarget` list.
3. Decide whether notification routes should support optional return intent metadata instead of raw route strings only.

## Alternatives Considered

### A. Use `router.back()`

Pros: minimal code.

Cons: unreliable for deep links, reloads, desktop-shell entry points, gamepad flows, and notification jumps. It also cannot choose between "back to detail" and "back to previous browse state" when browser history is missing or noisy.

### B. Add an in-page "Return to detail" banner only on actor filters

Pros: visually explicit.

Cons: duplicates app-shell navigation, solves only actor filter pages, and creates another one-off navigation surface.

### C. Extend the existing `back` query model

Pros: matches current player/detail navigation architecture, testable in route helpers, survives reload/deep link as long as query contains the originating selected movie id.

Cons: requires making `AppShell` show a back button on a primary route when the primary route is a drill-down.

Recommended: C.

## Verification Plan

Run focused frontend tests first:

```bash
pnpm test -- src/lib/navigation-intent.test.ts src/views/DetailView.test.ts src/layouts/AppShell.test.ts
```

Then run broader frontend checks if the first slice modifies shared route helpers:

```bash
pnpm typecheck
pnpm test
```

Manual QA:

1. Open a movie detail page from library/favorites/tags.
2. Click an actor in the cast area.
3. Confirm the filtered library page shows an app-level "Back to details" button.
4. Click it and confirm the original detail page opens.
5. Repeat with metadata tag and studio links.
6. Confirm normal library/favorites/tags/trash browse pages still do not show a back button.

