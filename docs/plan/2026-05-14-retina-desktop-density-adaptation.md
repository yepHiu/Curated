# Retina Desktop Density Adaptation

## Goal

Make Curated show more content on MacBook Air and other desktop Retina screens whose CSS viewport is smaller but `devicePixelRatio` is `2`, without changing the existing layout density for DPR `1.5` external displays.

## Scope

- Add a desktop Retina density layer for pointer/hover desktop viewports at `min-width: 1024px` and `min-resolution: 2dppx`.
- Keep DPR `1`, `1.25`, and `1.5` on the default density variables.
- Avoid OS or browser user-agent checks. The layout remains CSS-pixel based with one media-query density override.
- Do not run `pnpm test:display` unless the user explicitly asks, because the project display-scaling checklist marks it as a long-running opt-in suite.

## Implementation

- Define app density variables in `src/style.css` for sidebar width, shell/header padding, movie grid minimum track width, grid gap, related-card width, and movie-card compact spacing.
- Override only those variables inside `(hover: hover) and (pointer: fine) and (min-width: 1024px) and (min-resolution: 2dppx)`.
- Wire `AppShell.vue`, `AppSidebar.vue`, `LibraryView.vue`, `VirtualMovieMasonry.vue`, `MovieGrid.vue`, and `MovieCard.vue` to use the variables instead of hard-coded desktop dimensions.
- Add a small TypeScript density helper for virtual-list estimates so the rendered grid and virtual-scroll math use the same default-vs-Retina decision.

## Tests

- Unit-test the Retina density query to ensure it starts at `2dppx` and does not mention `1.5dppx`.
- Unit-test default and compact movie-grid density numbers.
- Unit-test that `VirtualMovieMasonry` uses compact virtual-scroll gap estimates only when the Retina media query matches.
- Unit-test that `AppShell` uses CSS-variable sidebar and header sizing instead of fixed desktop sizes.

