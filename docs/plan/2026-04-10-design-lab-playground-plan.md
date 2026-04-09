# Curated Design Lab Playground Spec And Implementation Plan

## Context

- Product name is `Curated`.
- Current frontend stack is `Vue 3 + TypeScript + Vite + Tailwind v4 + shadcn-vue`.
- Current global theme tokens already live in `src/style.css`.
- Current settings page already has an `About` section and already distinguishes development mode through `import.meta.env.DEV`.
- `SettingsPage.vue` is already very large, so the playground should not be implemented as a giant extra block inside the existing About tab content.

## Goal

Create a `Curated Design Lab` playground that acts as the UI/UX sandbox and the practical design-system source of truth for frontend work. It should allow fast prototyping, token inspection, component state review, and code extraction without polluting production UX.

## Recommended Product Shape

### Entry strategy

- Keep the user-facing entry inside `Settings > About`.
- Show the entry only in development builds.
- The entry should navigate to a dedicated route instead of rendering the entire playground inside the Settings page.
- This is now confirmed.

### Route strategy

- Add a dedicated route at `/design-lab`.
- Guard it with `import.meta.env.DEV` in the router and in the UI entry point.
- Do not add it to the main sidebar in phase 1.
- Allow direct deep-linking while developing.
- This route naming is now confirmed.

### Why this shape

- Matches the user's requirement that the entry lives in About.
- Avoids making `SettingsPage.vue` even larger and more stateful.
- Gives the playground room for split panes, responsive sandboxes, and live code previews.
- Keeps production UI clean and low-risk.

## Alternatives Considered

### Option A: Build the playground directly inside `Settings > About`

Pros:

- Fastest entry path for developers.
- No new route/view wiring.

Cons:

- Conflicts with the current settings-page size and complexity.
- Hard to support split panes, responsive canvas, and larger demos.
- Makes settings maintenance worse.

### Option B: Add a dedicated dev-only route, linked from `Settings > About`

Pros:

- Best balance of isolation and discoverability.
- Easier to scale from simple token viewer to full playground.
- Clear separation between settings concerns and design-lab concerns.

Cons:

- Slightly more routing and shell work.

Recommendation:

- Choose this option.
- This is now confirmed.

### Option C: Create a standalone internal app outside the main router

Pros:

- Maximum isolation.
- Can evolve independently.

Cons:

- Overkill for the current repository.
- Duplicates shell, theme, and routing patterns.
- Weakens the “same environment as the product UI” goal.

## UI Structure Draft

## Page layout

### 1. Top bar

- Left: `Curated Design Lab` title, environment badge, optional current branch badge later.
- Center: module switcher or quick search for tokens/components.
- Right: theme toggle, viewport preset switcher, code panel toggle, reset sandbox button.

### 2. Left navigation rail

- `Tokens`
- `Components`
- `Playground`
- `Motion`
- `Accessibility`

Each section should support anchor navigation and lazy rendering.

### 3. Main content canvas

The main area should be a two-column workspace on desktop:

- Left 65-70%: visual preview canvas
- Right 30-35%: inspector, prop controls, generated code

On smaller widths:

- stack vertically
- move inspector below preview

### 4. Bottom utility dock

Optional in phase 2:

- copy status
- active token path
- contrast summary
- responsive width readout

## Section Drafts

### A. Tokens

Subsections:

- Color matrix
- Typography scale
- Radius and shadow scale
- Spacing scale
- Current semantic token mapping

Each token card should show:

- visual swatch or sample
- token name
- CSS variable
- value
- usage note
- contrast score where applicable

### B. Components

Subsections:

- Foundations
- Form controls
- Navigation
- Feedback
- Overlays

Each component block should show:

- default specimen
- state strip: default / hover / active / focus-visible / loading / disabled
- size variants
- tone variants
- notes for intended usage

### C. Playground

Split into:

- live preview canvas
- props panel
- code output panel

Phase 1 focus:

- Button
- Input
- Tag
- Card

These give the highest leverage for shaping the rest of the system.

### D. Motion

Show:

- fade / slide / zoom presets
- duration tokens
- easing tokens
- reduced-motion notes

### E. Accessibility

Show:

- contrast checks
- keyboard focus examples
- focus order examples
- state labels and aria notes

## Token Strategy

Use a three-layer token model.

### Layer 1: Primitive palette tokens

These are raw scales and should not be consumed directly by feature code except in the token lab itself.

- `--color-rose-50` to `--color-rose-900`
- `--color-slate-50` to `--color-slate-950`
- `--color-green-50` to `--color-green-900`
- `--color-amber-50` to `--color-amber-900`
- `--color-red-50` to `--color-red-900`
- `--color-blue-50` to `--color-blue-900`

### Layer 2: Semantic app tokens

These are the main app-facing tokens and should back Tailwind semantic classes.

- `--background`
- `--foreground`
- `--surface`
- `--surface-elevated`
- `--surface-muted`
- `--border`
- `--border-strong`
- `--primary`
- `--primary-foreground`
- `--success`
- `--success-foreground`
- `--warning`
- `--warning-foreground`
- `--danger`
- `--danger-foreground`
- `--info`
- `--info-foreground`
- `--focus-ring`

### Layer 3: Component decision tokens

Use sparingly where the semantic layer is still too coarse.

- `--button-radius`
- `--input-height-md`
- `--card-shadow-hover`
- `--dialog-backdrop`

## Proposed Core Color Tokens

This proposal intentionally extends the current Curated color language instead of replacing it.

### Brand

- Brand primary: `#FE628E`
- Brand deep accent: `#C93C68`
- Brand soft tint: `#FFD7E2`

### Semantic support colors

- Success: `#2F9E78`
- Warning: `#D89A1B`
- Danger: `#E14B6D`
- Info: `#5B6FD4`

### Neutral light scale

- Neutral 0: `#FFFFFF`
- Neutral 50: `#F8FAFC`
- Neutral 100: `#F4F6FC`
- Neutral 200: `#EBEEF5`
- Neutral 300: `#DCE2EC`
- Neutral 400: `#B8C0D1`
- Neutral 500: `#8B95A8`
- Neutral 600: `#5A6378`
- Neutral 700: `#3A4254`
- Neutral 800: `#1F2634`
- Neutral 900: `#0F1219`

### Neutral dark scale

- Dark 950: `#0D0F1A`
- Dark 900: `#101423`
- Dark 800: `#141826`
- Dark 700: `#1B2234`
- Dark 600: `#252D43`
- Dark 500: `#4A556E`
- Dark 400: `#7E89A3`
- Dark 300: `#A2ABC2`
- Dark 200: `#CBD3E5`
- Dark 100: `#F8F7FB`

## Proposed Typography Tokens

Phase 1 should avoid a large app-wide font migration. Keep the current brand font approach and formalize the scale first.

### Font families

- `--font-brand`: `Outfit`, for brand and showcase headings only
- `--font-sans`: existing app sans stack
- `--font-mono`: existing monospace stack

### Font weights

- `--font-weight-regular`: `400`
- `--font-weight-medium`: `500`
- `--font-weight-semibold`: `600`
- `--font-weight-bold`: `700`

### Type scale

- `--text-h1`: `32px / 40px / 700`
- `--text-h2`: `28px / 36px / 700`
- `--text-h3`: `24px / 32px / 600`
- `--text-h4`: `20px / 28px / 600`
- `--text-h5`: `18px / 26px / 600`
- `--text-h6`: `16px / 24px / 600`
- `--text-body-lg`: `16px / 28px / 400`
- `--text-body`: `14px / 24px / 400`
- `--text-body-sm`: `13px / 20px / 400`
- `--text-caption`: `12px / 18px / 500`
- `--text-code`: `13px / 18px / 500`

### Typography principle

- Headings should prioritize hierarchy clarity over decorative styling.
- Long descriptions in the playground should use body or body-sm.
- Generated code and token values should consistently use the mono scale.

## Component Scope Recommendation

### Phase 1

- Button
- Input
- Tag / Badge
- Card
- Checkbox
- Switch
- Skeleton

### Phase 2

- Select
- Breadcrumb
- Pagination
- Steps
- Dropdown
- Toast
- Modal
- Drawer
- Avatar

### Why this order

- Phase 1 covers the most reused interaction patterns.
- It creates enough token pressure to validate color, radius, focus, spacing, and code-generation rules.
- Overlays and navigation components become more reliable after the core states are stabilized.

## Engineering Shape

## Suggested files

- `src/views/DesignLabView.vue`
- `src/components/design-lab/DesignLabShell.vue`
- `src/components/design-lab/DesignLabSectionTokens.vue`
- `src/components/design-lab/DesignLabSectionComponents.vue`
- `src/components/design-lab/DesignLabSectionPlayground.vue`
- `src/components/design-lab/DesignLabSectionMotion.vue`
- `src/components/design-lab/DesignLabSectionA11y.vue`
- `src/components/design-lab/playground/*`
- `src/lib/design-lab/*`
- `src/domain/design-lab/*`

### Entry touchpoints

- `src/components/jav-library/SettingsPage.vue`
- `src/lib/settings-nav.ts`
- `src/router/index.ts`

### Token touchpoints

- `src/style.css`
- optional later: `docs/frontend-ui-spec.md`

## Data and state model

- No backend dependency in phase 1.
- Entire playground can run in frontend-only mode.
- Use local reactive state for controls.
- Optionally persist local playground preferences in `localStorage` later:
  - active theme
  - active viewport
  - selected component
  - code panel collapsed state

## Code generation strategy

Phase 1 should use deterministic string templates, not AST generation.

- Safer
- Easier to audit
- Enough for Vue + Tailwind snippets

Each playground definition should contain:

- component id
- label
- prop schema
- default props
- preview renderer mapping
- code renderer mapping

## Delivery Plan

### Phase 0: Design approval

- Confirm route and entry strategy.
- Confirm initial token model.
- Confirm phase-1 component set.
- Confirm playground code output scope.

### Phase 1: Shell and entry

- Add dev-only About entry.
- Add dedicated route and empty shell.
- Add section navigation and desktop/mobile layout.
- Keep the route hidden from production and from the main sidebar.

### Phase 2: Token lab

- Refactor current `src/style.css` token story into primitive + semantic documentation.
- Build color, typography, radius, shadow, spacing showcases.
- Add contrast scoring utilities.
- Preserve current Curated brand direction while lightly extending semantic tokens.

### Phase 3: Component state gallery

- Build state strips for the initial component set.
- Reuse existing `ui` components wherever possible.
- Add explicit focus-visible and disabled examples.
- Keep each component showcase aligned to real in-repo variants, not invented variants.

### Phase 4: Live playground

- Add props inspector for Button, Input, Tag, Card.
- Add live code generation.
- Add preview width presets and custom width slider.
- Output Vue snippets plus token usage notes.

### Phase 5: Motion and a11y lab

- Add motion preview blocks.
- Add reduced-motion notes.
- Add keyboard walkthrough and accessibility summary blocks.

### Phase 6: Documentation sync

- Update `docs/frontend-ui-spec.md` once the token and component rules are validated.
- If the final route or dev-entry behavior becomes stable project fact, update `.cursor/rules/project-facts.mdc`.

## Engineering Rollout Detail

### Proposed file map

- Create `src/views/DesignLabView.vue`
- Create `src/components/design-lab/DesignLabShell.vue`
- Create `src/components/design-lab/DesignLabTopbar.vue`
- Create `src/components/design-lab/DesignLabNav.vue`
- Create `src/components/design-lab/sections/DesignLabTokensSection.vue`
- Create `src/components/design-lab/sections/DesignLabComponentsSection.vue`
- Create `src/components/design-lab/sections/DesignLabPlaygroundSection.vue`
- Create `src/components/design-lab/sections/DesignLabMotionSection.vue`
- Create `src/components/design-lab/sections/DesignLabA11ySection.vue`
- Create `src/components/design-lab/playground/PlaygroundPreviewCanvas.vue`
- Create `src/components/design-lab/playground/PlaygroundInspector.vue`
- Create `src/components/design-lab/playground/PlaygroundCodePanel.vue`
- Create `src/lib/design-lab/tokens.ts`
- Create `src/lib/design-lab/playground-definitions.ts`
- Create `src/lib/design-lab/contrast.ts`
- Modify `src/router/index.ts`
- Modify `src/components/jav-library/SettingsPage.vue`
- Modify locale files for About-entry copy if needed
- Modify `src/style.css` in the token-extension phase only

### State boundaries

- Route visibility is controlled by `import.meta.env.DEV`.
- Playground section navigation is local page state.
- Theme toggle can reuse the current document theme mechanism if available; otherwise use local document-class control inside the page shell.
- Component inspector state is page-local and should not leak into product settings.
- Persist only low-risk developer preferences in localStorage later if needed.

### Reuse strategy

- Reuse existing `src/components/ui` building blocks first.
- Build design-lab-only wrapper components only when the current UI primitives need a showcase adapter or playground-only control surface.
- Do not fork existing production components just to make the lab easier to build.

## Verification Strategy

### During implementation

- Frontend typecheck after each substantial phase with `pnpm typecheck`
- Frontend lint before claiming completion with `pnpm lint`
- Frontend tests where added with `pnpm test`
- Production build sanity check before final completion with `pnpm build`

### Manual verification checklist

- In development mode, `Settings > About` shows the Design Lab entry.
- In non-development mode, the entry is absent.
- Direct navigation to `#/design-lab` is blocked or redirected outside development mode.
- Light and dark mode both render token cards and component states correctly.
- Button, Input, Tag, and Card playground controls update the preview and generated code immediately.
- Mobile, tablet, and desktop preview widths all work inside the sandbox container.
- Focus-visible states are clearly visible with keyboard navigation.
- Contrast scores render for token showcase rows.

## Risks and Guardrails

### Main risks

- The scope expands into a mini Storybook replacement.
- The lab invents tokens or variants that the product does not really use.
- The settings page gains too much extra branching logic even though the main content is elsewhere.
- Dark-mode behavior diverges from the actual app shell.

### Guardrails

- Keep the About entry thin: title, description, and open action only.
- Treat the lab as an internal engineering surface, not a polished marketing page.
- Derive component examples from current `ui` and Curated component usage patterns.
- Keep token changes additive in phase 1.

## Branch Strategy

- Do not implement on `master`.
- After the design doc and plan are approved, create a dedicated development branch before code changes.
- Recommended branch name: `feat/design-lab-playground`
- Keep commits atomic:
  - route and entry
  - shell layout
  - token lab
  - component gallery
  - live playground
  - docs sync

## Approval Snapshot

Confirmed so far:

- Dev-only entry lives in `Settings > About`
- Main playground lives on a dedicated route
- Route name is `/design-lab`
- Phase-1 token work is a light extension of the current token system
- Code output should be Vue snippets plus token usage notes
- Phase-1 component scope is centered on Button, Input, Tag, Card, Checkbox, Switch, and Skeleton

---

# Curated Design Lab Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a dev-only `Design Lab` route with a Settings About entry, a token showcase, a first-pass component gallery, and a live playground for high-leverage UI primitives.

**Architecture:** Keep the product-facing entry thin inside `Settings > About`, but move the actual playground onto a dedicated dev-only route at `/design-lab`. Build the lab as a separate frontend surface under `src/components/design-lab`, reuse existing `ui` primitives and current theme tokens, and keep phase 1 frontend-only with no backend dependency.

**Tech Stack:** Vue 3 Composition API, TypeScript, Vite, Tailwind CSS v4, shadcn-vue, project theme tokens in `src/style.css`

---

## File Structure Map

- `src/router/index.ts`
  - add the dev-only `/design-lab` route
- `src/components/jav-library/SettingsPage.vue`
  - add the dev-only About entry card or action
- `src/views/DesignLabView.vue`
  - route-level view wrapper
- `src/components/design-lab/DesignLabShell.vue`
  - page shell and layout composition
- `src/components/design-lab/DesignLabTopbar.vue`
  - theme toggle, viewport presets, reset action
- `src/components/design-lab/DesignLabNav.vue`
  - local section navigation
- `src/components/design-lab/sections/DesignLabTokensSection.vue`
  - color, typography, radius, shadow, spacing showcase
- `src/components/design-lab/sections/DesignLabComponentsSection.vue`
  - first-pass static component gallery
- `src/components/design-lab/sections/DesignLabPlaygroundSection.vue`
  - live preview, inspector, code panel
- `src/components/design-lab/sections/DesignLabMotionSection.vue`
  - motion preview cards
- `src/components/design-lab/sections/DesignLabA11ySection.vue`
  - contrast and focus demos
- `src/components/design-lab/playground/PlaygroundPreviewCanvas.vue`
  - responsive preview container
- `src/components/design-lab/playground/PlaygroundInspector.vue`
  - props controls
- `src/components/design-lab/playground/PlaygroundCodePanel.vue`
  - live snippet output
- `src/lib/design-lab/tokens.ts`
  - typed token definitions used by the token showcase
- `src/lib/design-lab/playground-definitions.ts`
  - typed component playground definitions and defaults
- `src/lib/design-lab/contrast.ts`
  - WCAG contrast helpers
- `src/locales/en.json`
  - optional Design Lab entry copy
- `src/locales/zh-CN.json`
  - optional Design Lab entry copy
- `src/locales/ja.json`
  - optional Design Lab entry copy
- `src/style.css`
  - additive token extension only, not a full theme rewrite
- `docs/frontend-ui-spec.md`
  - post-validation documentation sync

### Task 1: Add The Dev-Only Route And About Entry

**Files:**
- Create: `src/views/DesignLabView.vue`
- Modify: `src/router/index.ts`
- Modify: `src/components/jav-library/SettingsPage.vue`
- Modify: `src/locales/en.json`
- Modify: `src/locales/zh-CN.json`
- Modify: `src/locales/ja.json`
- Test: manual route and visibility verification

- [ ] **Step 1: Add the failing route visibility expectation**

Document the expected behavior before implementation:

- In development mode, `#/design-lab` resolves to the new view and the About section exposes an entry action.
- Outside development mode, the About entry is hidden and direct navigation to `#/design-lab` redirects to `settings?section=about` or `library`.

- [ ] **Step 2: Add the route wrapper view**

Create [`src/views/DesignLabView.vue`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/views/DesignLabView.vue) with a minimal route wrapper:

```vue
<script setup lang="ts">
import DesignLabShell from "@/components/design-lab/DesignLabShell.vue"
</script>

<template>
  <DesignLabShell />
</template>
```

- [ ] **Step 3: Add the router entry with development gating**

Update [`src/router/index.ts`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/router/index.ts) to include a guarded route:

```ts
const isDev = import.meta.env.DEV

{
  path: "design-lab",
  name: "design-lab",
  redirect: () => (isDev ? undefined : { name: "settings", query: { section: "about" } }),
  component: isDev ? () => import("@/views/DesignLabView.vue") : () => import("@/views/SettingsView.vue"),
}
```

Then normalize the final implementation to the router pattern already used in the file. The key requirement is that non-dev access never lands on the lab.

- [ ] **Step 4: Add the dev-only About entry**

In [`src/components/jav-library/SettingsPage.vue`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/components/jav-library/SettingsPage.vue), add a new card or action block inside the About section under the existing dev-only content:

```vue
<template v-if="isViteDev">
  <div class="rounded-lg border border-border/50 bg-muted/5 px-4 py-3">
    <p class="font-medium text-foreground">{{ t("settings.designLabEntryTitle") }}</p>
    <p class="mt-1.5 text-sm text-muted-foreground">
      {{ t("settings.designLabEntryDesc") }}
    </p>
    <Button
      class="mt-3 h-10 rounded-xl px-4"
      variant="secondary"
      @click="router.push({ name: 'design-lab' })"
    >
      {{ t("settings.openDesignLab") }}
    </Button>
  </div>
</template>
```

Match the existing About card visual language rather than introducing a new style direction.

- [ ] **Step 5: Add locale strings**

Add matching keys for the three locale files:

```json
"designLabEntryTitle": "Design Lab",
"designLabEntryDesc": "Open the internal UI playground for token review, component prototyping, and interaction testing.",
"openDesignLab": "Open Design Lab"
```

Translate the strings appropriately in Chinese and Japanese instead of leaving English placeholders.

- [ ] **Step 6: Verify the route and entry behavior manually**

Run: `pnpm typecheck`
Expected: PASS

Manual checks:

- In dev, the About section shows the Design Lab entry.
- In dev, clicking the entry opens `#/design-lab`.
- In non-dev builds, there is no visible entry and the route is not usable.

- [ ] **Step 7: Commit**

```bash
git add src/router/index.ts src/views/DesignLabView.vue src/components/jav-library/SettingsPage.vue src/locales/en.json src/locales/zh-CN.json src/locales/ja.json
git commit -m "feat: add dev-only design lab entry"
```

### Task 2: Build The Design Lab Shell And Navigation

**Files:**
- Create: `src/components/design-lab/DesignLabShell.vue`
- Create: `src/components/design-lab/DesignLabTopbar.vue`
- Create: `src/components/design-lab/DesignLabNav.vue`
- Test: `pnpm typecheck`

- [ ] **Step 1: Create the shell component**

Create [`src/components/design-lab/DesignLabShell.vue`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/components/design-lab/DesignLabShell.vue) with desktop-first layout composition:

```vue
<script setup lang="ts">
import DesignLabNav from "@/components/design-lab/DesignLabNav.vue"
import DesignLabTopbar from "@/components/design-lab/DesignLabTopbar.vue"
import DesignLabTokensSection from "@/components/design-lab/sections/DesignLabTokensSection.vue"
import DesignLabComponentsSection from "@/components/design-lab/sections/DesignLabComponentsSection.vue"
import DesignLabPlaygroundSection from "@/components/design-lab/sections/DesignLabPlaygroundSection.vue"
import DesignLabMotionSection from "@/components/design-lab/sections/DesignLabMotionSection.vue"
import DesignLabA11ySection from "@/components/design-lab/sections/DesignLabA11ySection.vue"
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <DesignLabTopbar />
    <div class="flex min-h-0 flex-1">
      <DesignLabNav class="hidden w-56 shrink-0 lg:flex" />
      <main class="min-h-0 flex-1 overflow-y-auto p-6 lg:p-8">
        <div class="mx-auto flex w-full max-w-7xl flex-col gap-10">
          <DesignLabTokensSection />
          <DesignLabComponentsSection />
          <DesignLabPlaygroundSection />
          <DesignLabMotionSection />
          <DesignLabA11ySection />
        </div>
      </main>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Create the top bar**

Include:

- page title
- environment badge
- theme toggle placeholder
- viewport preset placeholder
- reset action placeholder

Use existing `Button`, `Badge`, and semantic token classes rather than hard-coded colors.

- [ ] **Step 3: Create the section nav**

Use a typed local list:

```ts
export const DESIGN_LAB_SECTIONS = [
  { id: "tokens", label: "Tokens" },
  { id: "components", label: "Components" },
  { id: "playground", label: "Playground" },
  { id: "motion", label: "Motion" },
  { id: "accessibility", label: "Accessibility" },
] as const
```

Render them as anchor links to section ids instead of adding nested router state.

- [ ] **Step 4: Verify the shell compiles**

Run: `pnpm typecheck`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/components/design-lab/DesignLabShell.vue src/components/design-lab/DesignLabTopbar.vue src/components/design-lab/DesignLabNav.vue
git commit -m "feat: add design lab shell layout"
```

### Task 3: Add Typed Token Definitions And The Token Showcase

**Files:**
- Create: `src/lib/design-lab/tokens.ts`
- Create: `src/lib/design-lab/contrast.ts`
- Create: `src/components/design-lab/sections/DesignLabTokensSection.vue`
- Modify: `src/style.css`
- Test: `pnpm typecheck`

- [ ] **Step 1: Add typed token metadata**

Create [`src/lib/design-lab/tokens.ts`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/lib/design-lab/tokens.ts) with display metadata:

```ts
export type ColorTokenSpec = {
  name: string
  cssVar: string
  value: string
  usage: string
  onColorVar?: string
}

export const semanticColorTokens: ColorTokenSpec[] = [
  {
    name: "Primary",
    cssVar: "--primary",
    value: "#FE628E",
    usage: "Primary action, selected state, brand emphasis",
    onColorVar: "--primary-foreground",
  },
]
```

Add sections for:

- semantic colors
- neutral scale
- typography scale
- radius scale
- shadow scale
- spacing scale

- [ ] **Step 2: Add contrast helpers**

Create [`src/lib/design-lab/contrast.ts`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/lib/design-lab/contrast.ts) with pure helpers:

```ts
export function getContrastRatio(foregroundHex: string, backgroundHex: string): number {
  return 1
}
```

Implement real WCAG contrast math before completing the task. Keep the helper framework-agnostic and unit-testable.

- [ ] **Step 3: Add additive theme tokens to style.css**

Update [`src/style.css`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/style.css) by adding only the approved extensions:

- semantic support colors: success, warning, danger, info
- stronger border token
- optional surface aliases

Do not replace the current `--primary`, `--background`, or overall Curated visual language.

- [ ] **Step 4: Create the token showcase section**

Create [`src/components/design-lab/sections/DesignLabTokensSection.vue`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/components/design-lab/sections/DesignLabTokensSection.vue) with:

- color cards
- typography specimens
- radius chips
- shadow panels
- spacing ruler

Each token row should show:

- token name
- css variable
- value
- usage note
- contrast score when applicable

- [ ] **Step 5: Verify token section behavior**

Run: `pnpm typecheck`
Expected: PASS

Manual checks:

- Light and dark values both remain readable.
- Token cards use the actual theme variables, not parallel hard-coded values.

- [ ] **Step 6: Commit**

```bash
git add src/lib/design-lab/tokens.ts src/lib/design-lab/contrast.ts src/components/design-lab/sections/DesignLabTokensSection.vue src/style.css
git commit -m "feat: add design lab token showcase"
```

### Task 4: Add The Static Component Gallery

**Files:**
- Create: `src/components/design-lab/sections/DesignLabComponentsSection.vue`
- Test: `pnpm typecheck`

- [ ] **Step 1: Build the static gallery section**

Create the section with blocks for:

- Button
- Input
- Tag or Badge
- Card
- Checkbox
- Switch
- Skeleton

- [ ] **Step 2: Show the required states**

For the supported components, explicitly render:

- default
- hover
- active
- focus-visible
- loading where applicable
- disabled

Prefer using deterministic classes or wrappers for visualized states instead of requiring actual mouse interaction for the gallery.

- [ ] **Step 3: Add short usage notes**

Under each gallery block, add concise text describing intended use so the gallery also serves as living documentation.

- [ ] **Step 4: Verify the section compiles**

Run: `pnpm typecheck`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/components/design-lab/sections/DesignLabComponentsSection.vue
git commit -m "feat: add design lab component gallery"
```

### Task 5: Add The Live Playground And Code Output

**Files:**
- Create: `src/lib/design-lab/playground-definitions.ts`
- Create: `src/components/design-lab/playground/PlaygroundPreviewCanvas.vue`
- Create: `src/components/design-lab/playground/PlaygroundInspector.vue`
- Create: `src/components/design-lab/playground/PlaygroundCodePanel.vue`
- Create: `src/components/design-lab/sections/DesignLabPlaygroundSection.vue`
- Test: `pnpm typecheck`

- [ ] **Step 1: Add typed playground definitions**

Create [`src/lib/design-lab/playground-definitions.ts`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/lib/design-lab/playground-definitions.ts) with typed component configs for:

- Button
- Input
- Tag
- Card

Each definition should include:

- id
- label
- supported controls
- default props
- code renderer
- usage-note renderer

- [ ] **Step 2: Build the preview canvas**

Create a preview component that supports:

- mobile preset `375px`
- tablet preset `768px`
- desktop preset `1280px`
- custom width

The canvas should center the specimen inside a bordered surface and never depend on app-level route layout widths.

- [ ] **Step 3: Build the inspector**

Provide practical controls only:

- component selector
- variant selector
- size selector
- radius selector
- text inputs
- toggle switches

Use strongly typed local state instead of generic schema editors.

- [ ] **Step 4: Build the code panel**

The code panel must output:

- a Vue snippet
- token usage notes

Example shape:

```ts
export type PlaygroundCodeOutput = {
  vueSnippet: string
  tokenNotes: string[]
}
```

- [ ] **Step 5: Compose the playground section**

Create [`src/components/design-lab/sections/DesignLabPlaygroundSection.vue`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/src/components/design-lab/sections/DesignLabPlaygroundSection.vue) that wires:

- preview canvas
- inspector
- code panel

Desktop should be two-column or three-column depending on available width. Smaller screens should stack vertically.

- [ ] **Step 6: Verify live updates**

Run: `pnpm typecheck`
Expected: PASS

Manual checks:

- Changing controls immediately updates the preview.
- Changing controls immediately updates the code output.
- Width presets visibly affect the specimen layout.

- [ ] **Step 7: Commit**

```bash
git add src/lib/design-lab/playground-definitions.ts src/components/design-lab/playground/PlaygroundPreviewCanvas.vue src/components/design-lab/playground/PlaygroundInspector.vue src/components/design-lab/playground/PlaygroundCodePanel.vue src/components/design-lab/sections/DesignLabPlaygroundSection.vue
git commit -m "feat: add interactive design lab playground"
```

### Task 6: Add Motion And Accessibility Sections

**Files:**
- Create: `src/components/design-lab/sections/DesignLabMotionSection.vue`
- Create: `src/components/design-lab/sections/DesignLabA11ySection.vue`
- Test: `pnpm typecheck`

- [ ] **Step 1: Add the motion section**

Render cards for:

- fade
- slide
- zoom

Each card should show:

- preview
- duration
- easing description
- reduced-motion note

- [ ] **Step 2: Add the accessibility section**

Render:

- contrast summary examples
- focus-visible examples
- keyboard-navigation reminder notes

Do not attempt a full automated a11y engine in phase 1.

- [ ] **Step 3: Verify the sections compile**

Run: `pnpm typecheck`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add src/components/design-lab/sections/DesignLabMotionSection.vue src/components/design-lab/sections/DesignLabA11ySection.vue
git commit -m "feat: add design lab motion and accessibility sections"
```

### Task 7: Final Verification And Documentation Sync

**Files:**
- Modify: `docs/frontend-ui-spec.md`
- Modify: `docs/plan/2026-04-10-design-lab-playground-plan.md`
- Test: `pnpm typecheck`
- Test: `pnpm lint`
- Test: `pnpm test`
- Test: `pnpm build`

- [ ] **Step 1: Update the frontend UI spec**

Document the validated token additions and the existence of the internal Design Lab in [`docs/frontend-ui-spec.md`](C:/Users/wujiahui/code/jav-lib/jav-shadcn/docs/frontend-ui-spec.md).

- [ ] **Step 2: Re-read this plan and mark any implementation drift**

If the implementation differed from the approved plan, update the plan doc inline so the repository does not retain stale intent.

- [ ] **Step 3: Run the verification sequence**

Run: `pnpm typecheck`
Expected: PASS

Run: `pnpm lint`
Expected: PASS

Run: `pnpm test`
Expected: PASS, or note any pre-existing failure unrelated to the Design Lab

Run: `pnpm build`
Expected: PASS

- [ ] **Step 4: Manual acceptance check**

Verify:

- Design Lab entry is visible only in dev
- `/design-lab` route is available only in dev
- token showcase reflects actual theme values
- Button, Input, Tag, and Card playground outputs are usable
- dark mode and responsive sandbox both work

- [ ] **Step 5: Commit**

```bash
git add docs/frontend-ui-spec.md docs/plan/2026-04-10-design-lab-playground-plan.md
git commit -m "docs: sync design lab implementation notes"
```

## Key implementation rules

- Dev-only exposure should be hard-gated in UI and routing.
- The playground must not increase backend coupling.
- Reuse current semantic tokens first; do not fork a second theme system.
- Prefer modular `design-lab` components over growing `SettingsPage.vue`.
- Generated code should reflect actual in-repo usage patterns, not idealized examples that the project does not use.

## Open decisions for review

1. Token ambition for phase 1:
   - confirmed: lightly extend current tokens
   - specifically add semantic support colors, neutral scale, and typography scale without replacing the current Curated visual language

2. Code output scope:
   - confirmed: Vue snippet output plus token usage notes
   - keep output practical for in-repo component usage rather than generating full page scaffolds

## Recommendation Summary

- Use a dev-only dedicated route linked from `Settings > About`.
- Keep phase 1 frontend-only.
- Start with tokens plus 4 high-leverage components in the live playground.
- Treat this as a structured internal lab, not a public showcase page.
