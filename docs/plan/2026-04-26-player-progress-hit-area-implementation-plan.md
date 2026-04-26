# Player Progress Hit Area Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the player progress bar easier to click by enlarging the effective interaction zone while keeping the visible track slim.

**Architecture:** Keep the change inside the shared `Slider` UI component so the player benefits without introducing custom seek math or pointer-capture logic in `PlayerPage.vue`. Use a focused component test to lock the larger hit area and centered visual track before changing the slider classes.

**Tech Stack:** Vue 3, shadcn-vue/reka-ui slider wrapper, Vitest, Vue Test Utils, Tailwind utility classes.

---

### Task 1: Lock Slider Hit-Area Expectations

**Files:**
- Create: `src/components/ui/slider/Slider.test.ts`
- Modify: `src/components/ui/slider/Slider.vue`

- [ ] **Step 1: Write the failing test**

Create `src/components/ui/slider/Slider.test.ts` with a focused structure assertion for the expanded interaction zone:

```ts
import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"
import Slider from "@/components/ui/slider/Slider.vue"

describe("Slider", () => {
  it("keeps a slim visual track while exposing a larger click target", () => {
    const wrapper = mount(Slider, {
      props: {
        modelValue: [25],
        max: 100,
      },
    })

    const root = wrapper.get('[data-slot="slider"]')
    const track = wrapper.get('[data-slot="slider-track"]')

    expect(root.classes()).toContain("h-6")
    expect(track.classes()).toContain("absolute")
    expect(track.classes()).toContain("top-1/2")
    expect(track.classes()).toContain("-translate-y-1/2")
    expect(track.classes()).toContain("h-1.5")
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
pnpm test -- src/components/ui/slider/Slider.test.ts
```

Expected: FAIL because the current slider root does not include `h-6` and the track is not absolutely centered inside a larger hit zone.

- [ ] **Step 3: Write the minimal implementation**

Update `src/components/ui/slider/Slider.vue` so:

- the horizontal slider root gets a larger effective height such as `h-6`
- the track becomes absolutely positioned and vertically centered inside that larger root
- the visible track height stays `h-1.5`
- vertical slider behavior remains unchanged

- [ ] **Step 4: Run the targeted test to verify it passes**

Run:

```bash
pnpm test -- src/components/ui/slider/Slider.test.ts
```

Expected: PASS.

- [ ] **Step 5: Run typecheck**

Run:

```bash
pnpm typecheck
```

Expected: PASS.
