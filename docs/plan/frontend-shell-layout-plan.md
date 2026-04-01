# Curated Frontend Shell Layout Plan

## Goal

Align the app shell with the sidebar-plus-workspace distribution used by shadcn-vue dashboard blocks while keeping Curated's existing theme tokens, route structure, and page-level behavior intact.

## Implementation Summary

- Treat `src/layouts/AppShell.vue` as the single global shell entrypoint for layout changes.
- Keep the two-column model, but make it read as a fixed navigation rail plus a full workspace panel instead of a page card nested inside another card.
- Preserve mobile drawer behavior and all existing routing/search/state logic.
- Limit page-level adaptation to spacing and rhythm adjustments in the library browse workspace.

## Planned Changes

- `AppShell.vue`
  - Increase desktop sidebar width to roughly 304px and keep the compact rail aligned with the expanded shell rhythm.
  - Reduce outer shell padding and right-panel card emphasis.
  - Split the right side into a stable toolbar band and a dedicated content canvas with independent overflow handling.
  - Slightly widen the header search lane to better match the reference layout proportions.
- `AppSidebar.vue`
  - Tighten the outer container padding and corner treatment so it reads more like a persistent navigation surface.
  - Keep internal scrolling, status panel, and settings entry unchanged in behavior.
- `LibraryPage.vue`
  - Reduce top-level gaps and tighten the relationship between tabs/actions and the main browse area.
  - Keep virtualized content behavior and batch actions unchanged.

## Validation

- Verify desktop expand/collapse transitions and independent content scrolling.
- Verify mobile drawer open/close behavior and header wrapping.
- Check `library`, `actors`, `history`, `curated-frames`, `detail`, `player`, and `settings` under the new shell.
- Run `pnpm build` to catch template/type regressions.

## Notes

- This document records the layout implementation intent required by the workspace rules for plan-style changes.
- Visual parity is limited to spatial distribution only; no theme or brand styling should be copied from the reference block.
