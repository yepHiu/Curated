---
name: curated-ui-acceptance
description: Verify Curated Vue UI changes against design tokens, interaction states, accessibility, and responsive behavior. Use after UI implementation or for focused UI review.
---

# Curated UI Acceptance

Use this Skill to turn UI implementation into an explicit acceptance pass. For new directions or system decisions, start with `curated-ui-governance`.

## Inspect what changed

Read the relevant route/component and `docs/reference/2026-03-24-frontend-ui-spec.md`. Check the applicable surface contract, existing precedent, semantic tokens, primitive reuse, information hierarchy, and intentional density. Do not assess every generic checklist item when a change cannot affect it.

## Acceptance states

Review the applicable success, loading, empty, error, locked/unavailable, destructive-confirmation, keyboard/focus, and narrow-width states. Confirm:

- semantic interactive elements and visible `focus-visible` behavior;
- meaningful image `alt` and decorative `aria-hidden` use;
- readable fields on the root dark palette, without relying on `dark:` alone;
- localized text and the existing toast/dialog system;
- Web/Mock differences represented intentionally;
- 44px mobile targets for the documented primary actions.

For shell, grid/card geometry, settings, dialogs, HUD, global typography, or spacing, follow `docs/reference/frontend-display-scaling-checklist.md`. Do not run `pnpm test:display` unless the user expressly authorizes it; report it as unrun when relevant.

Conclude with concrete findings, changed or recommended fixes, verification executed, and any systemic convention that needs `curated-ui-governance` documentation.
