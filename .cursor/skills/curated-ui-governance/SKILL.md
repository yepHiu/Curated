---
name: curated-ui-governance
description: Establish or review Curated UI direction before implementing material Vue UI changes. Use for page, layout, component-system, interaction, or visual-consistency work; not for isolated text-only edits.
---

# Curated UI Governance

Use this Skill to make UI decisions *before* implementation and to keep those decisions consistent after implementation. It is a project design-governance workflow, not a generic styling checklist.

## Constitutional principles

Every material UI decision must preserve these principles unless the user explicitly requests a justified exception:

1. **Product surface before component.** Identify whether the work belongs to browse, detail, playback, settings, shell navigation, or feedback. Do not transplant a layout language from one surface into another without a reason.
2. **One hierarchy, one primary job.** A screen or panel has a clear user task, visual starting point, and primary action. Secondary controls must not compete with media, metadata, or the main task.
3. **System over local invention.** Use semantic tokens, existing `shadcn-vue` primitives, and established Curated patterns before creating a component, color, spacing scale, or interaction idiom.
4. **State completeness is part of design.** Loading, empty, error, locked, unavailable, destructive-confirmation, and narrow-width states are product states—not deferred engineering details.
5. **Density is intentional.** Curated is desktop-first and media-focused; compact browse surfaces and low-chrome playback are deliberate. Provide 44px touch targets for mobile primary actions without turning desktop controls into a generic dashboard.
6. **Accessible interaction is non-negotiable.** Interactive elements are semantic and keyboard reachable; focus is visible; text and inputs remain readable on the root dark palette; decorative layers stay out of the accessibility tree.
7. **Rules are living, but exceptions stay local.** A repeated decision becomes a documented system rule. A justified exception is labelled and contained; it must not silently become a new visual system.

## When to use it

Apply this Skill before a change that alters a route, a major section or dialog, shell/layout behavior, a shared UI primitive, design tokens, responsive behavior, or a recurring interaction pattern. Use it for UI reviews that need a concrete corrective direction.

For an isolated copy correction or a narrow implementation that faithfully follows an existing component pattern, use the relevant implementation rules directly instead.

## Start with a design frame

Before coding a material change, read the relevant parts of:

- `docs/reference/2026-03-24-frontend-ui-spec.md`
- `.cursor/rules/ui-component-spec.mdc`
- `.cursor/rules/vue-frontend-standards.mdc`
- `.cursor/rules/jav-library-frontend-patterns.mdc`
- the current route, closest analogous component, and `src/style.css` when tokens or density are involved.

State a concise design frame in the work summary:

| Item | Decide before implementation |
| --- | --- |
| Surface and job | Which Curated surface this belongs to, the user job, and the page/panel's primary action. |
| Existing precedent | The component or route that should be reused or deliberately diverged from. |
| Hierarchy and density | What users notice first, what is secondary, and whether the interaction is browse, inspect, configure, or act. |
| State matrix | Applicable loading, empty, error, unavailable/locked, confirmation, and narrow-width states. |
| System impact | Whether the work is local, a reusable business component, a UI primitive, or a token/rule change. |

Do not fabricate a full design document for a small established-pattern change. For a new surface or a cross-page rule, turn the frame into a short decision record in the relevant `docs/plan/` document before implementation.

## Design and implementation rules

- Reuse `src/components/ui` primitives before adding a new base component. Put Curated domain composition in `src/components/jav-library`; keep route assembly in `src/views` and `AppShell`.
- Use semantic token classes and existing spacing/shape language. Do not use raw palette colors or one-off CSS as a shortcut around a missing system decision.
- Keep the current surface contracts intact: browse is poster-first and compact; details are metadata-first; playback is video-first and low-chrome; settings use the centered, stacked-card model; `AppShell` owns the internal scrolling shell.
- Treat a new recurring status color, spacing tier, card archetype, or interaction pattern as a system decision. First check whether an existing semantic token or primitive can carry it; only then extend the system deliberately.
- Never bypass `useLibraryService()` and its contracts to make a UI demo work. Web, Mock, and Electron availability differences must have intentional UI states.
- Do not add instructional copy under ordinary library toolbars, filters, popovers, or section headings. Explain consequential or irreversible operations at the decision point.
- Keep user-visible text localizable. Use the existing toast and dialog primitives instead of adding parallel feedback systems.

## Validate proportionally

Run the narrowest relevant typecheck, unit, and E2E coverage. Inspect the states changed, not only the successful path. When changing shell layout, grid/card geometry, settings, dialogs, HUD, global typography, or spacing, follow `docs/reference/frontend-display-scaling-checklist.md`.

Do not run `pnpm test:display` without the user's explicit approval. Mention it as an unrun validation item when it is relevant.

## Maintain the constitution

When a change modifies global tokens, a shared primitive, or a recurring Curated UI pattern, update the canonical sources in the same change:

- `docs/reference/2026-03-24-frontend-ui-spec.md` for durable UI facts and conventions;
- `.cursor/rules/ui-component-spec.mdc` for concise implementation constraints;
- an existing relevant `docs/plan/` document for the decision's context and rationale, or a new dated plan document when it establishes a new cross-surface direction.

Do not promote a one-off visual preference to a rule. Record the scope and exception instead.

## Completion report

Summarize the design frame, precedent reused or reason for divergence, affected states, systemic rules updated (if any), validation run, and validation intentionally not run. This makes the design decision reviewable rather than leaving it implicit in Tailwind classes.
