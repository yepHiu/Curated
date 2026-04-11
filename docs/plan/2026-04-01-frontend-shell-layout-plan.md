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

## 2026-04-11 Update: Container Removal Direction

The selected direction is **Option A: Split Shell** from
`docs/plan/2026-04-11-shell-container-removal-prototype.html`.

**Goal:** remove the "large rounded container wrapping the sidebar and content area" impression and make the app read as a shadcn-style sidebar plus workspace shell.

**Architecture:** keep `AppShell.vue` as the global shell owner. The desktop grid should directly contain `AppSidebar` and the right workspace. `AppSidebar.vue` should read as a persistent navigation surface instead of an independent card, while mobile drawer behavior can keep its overlay/surface treatment.

**Implementation steps:**

1. Add a shell structure test in `src/layouts/AppShell.test.ts` that asserts the desktop shell uses a full-height split layout and no longer relies on the previous rounded right-panel container class.
2. Update `src/layouts/AppShell.vue` so the shell grid has no outer card gap on desktop, the sidebar column owns the right divider, and the right workspace is a plain flex column with a toolbar border and content scroll area.
3. Update `src/components/jav-library/AppSidebar.vue` so the root aside drops its card border/radius and uses sidebar surface tokens suitable for a fixed rail.
4. Run the targeted AppShell test, then run `pnpm typecheck` and `pnpm build` if the targeted test passes.

**Non-goals:**

- Do not redesign page-level movie cards, settings cards, dialogs, or player HUD.
- Do not change route/search behavior, drawer behavior, or backend status behavior.
- Do not introduce new color tokens; keep existing semantic Tailwind/shadcn-vue tokens.
