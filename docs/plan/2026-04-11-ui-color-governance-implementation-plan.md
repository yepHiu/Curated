# UI Color Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. This repository is being edited directly on master because the user explicitly requested it.

**Goal:** Start enforcing Curated's semantic color system by adding reusable status-tone class carriers and converting the highest-confidence raw status colors in business UI.

**Architecture:** Keep the existing theme tokens in `src/style.css`. Add a small pure helper under `src/lib/ui/` for status text, dot, badge, and panel class strings, then consume those helpers in high-exposure business components. Treat player HUD and dev tooling as documented exception areas rather than forcing them through normal page styling.

**Tech Stack:** Vue 3, Tailwind CSS v4, shadcn-vue components, class-variance-authority, Vitest.

---

### Task 1: Status Tone Helper

**Files:**
- Create: `src/lib/ui/status-tone.ts`
- Create: `src/lib/ui/status-tone.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, it } from "vitest"
import {
  statusBadgeClass,
  statusDotClass,
  statusPanelClass,
  statusTextClass,
} from "./status-tone"

describe("status tone classes", () => {
  it("maps success, warning, danger, and info to semantic tokens", () => {
    expect(statusTextClass("success")).toBe("text-success")
    expect(statusTextClass("warning")).toBe("text-warning")
    expect(statusTextClass("danger")).toBe("text-danger")
    expect(statusTextClass("info")).toBe("text-info")

    expect(statusDotClass("success")).toBe("bg-success shadow-sm ring-1 ring-success/40")
    expect(statusBadgeClass("warning")).toBe("border-warning/35 bg-warning/10 text-warning")
    expect(statusPanelClass("info")).toBe(
      "rounded-2xl border border-info/25 border-l-[3px] border-l-info/60 bg-info/[0.07] px-4 py-3",
    )
  })
})
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `pnpm test -- src/lib/ui/status-tone.test.ts`

Expected: FAIL because `src/lib/ui/status-tone.ts` does not exist.

- [ ] **Step 3: Add the minimal helper implementation**

Create `src/lib/ui/status-tone.ts` with the four status tones and explicit class maps.

- [ ] **Step 4: Run the test and verify it passes**

Run: `pnpm test -- src/lib/ui/status-tone.test.ts`

Expected: PASS.

### Task 2: Badge Status Variants

**Files:**
- Modify: `src/components/ui/badge/index.ts`
- Modify: `src/lib/ui/status-tone.ts`
- Modify: `src/lib/ui/status-tone.test.ts`

- [ ] **Step 1: Extend the failing test**

Add assertions that `statusBadgeClass("success")`, `statusBadgeClass("warning")`, `statusBadgeClass("danger")`, and `statusBadgeClass("info")` return class strings using semantic status tokens.

- [ ] **Step 2: Run the focused test**

Run: `pnpm test -- src/lib/ui/status-tone.test.ts`

Expected: PASS after Task 1, then continue to Badge implementation.

- [ ] **Step 3: Add `success`, `warning`, `danger`, and `info` variants to `badgeVariants`**

Use the same class strings exported by `statusBadgeClass` so the component carrier and helper stay aligned.

- [ ] **Step 4: Run focused verification**

Run: `pnpm test -- src/lib/ui/status-tone.test.ts`

Expected: PASS.

### Task 3: First Business UI Replacements

**Files:**
- Modify: `src/components/jav-library/AppSidebar.vue`
- Modify: `src/components/jav-library/SettingsPage.vue`

- [ ] **Step 1: Replace backend status dot raw colors**

Import `statusDotClass` and replace `bg-emerald-*`, `bg-amber-*`, and destructive dot variants with semantic status-dot helpers.

- [ ] **Step 2: Replace settings provider and proxy status raw colors**

Import `statusBadgeClass`, `statusDotClass`, `statusPanelClass`, and `statusTextClass`. Replace provider health badges/dots, proxy success text, local warning panel, unsupported directory warning text, and the CORS info panel with semantic status helpers.

- [ ] **Step 3: Keep explicit exceptions untouched**

Do not rewrite player immersive HUD colors or dev performance/watermark colors in this first slice.

### Task 4: Documentation and Verification

**Files:**
- Modify: `docs/2026-03-24-frontend-ui-spec.md`
- Modify: `docs/plan/2026-04-11-ui-color-governance-plan.md`

- [ ] **Step 1: Update the UI spec**

Add the semantic status color rule, raw Tailwind status color restriction, and exception areas.

- [ ] **Step 2: Update the governance plan status**

Mark the plan as implementation-started or equivalent.

- [ ] **Step 3: Run verification**

Run:

```powershell
pnpm test -- src/lib/ui/status-tone.test.ts
pnpm typecheck
pnpm build
```

Expected: all commands pass. If `pnpm build` repeats typecheck internally, still run it as the final production build gate.
