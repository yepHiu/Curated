# Home Detail Sidebar Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore homepage scroll only for the `home -> detail -> home` return path, add expandable movie summaries on the detail page, and reduce desktop sidebar collapse jank without changing the app's navigation model.

**Architecture:** Keep each improvement local to the surface that owns it. Homepage scroll restore uses a small session-scoped composable plus an explicit "arm restore before leaving home" call from `HomeView`. Detail summary folding is a focused presentational component consumed by `DetailPanel`. Sidebar performance work keeps the current shell and routes intact but shifts `AppSidebar` from branch replacement toward a mostly stable DOM tree with collapse-by-visibility semantics.

**Tech Stack:** Vue 3 Composition API, TypeScript, vue-router, Vitest, Vue Test Utils, Tailwind utility classes.

---

## File Map

- Create: `src/composables/use-home-scroll-preserve.ts`
  - Own the one-shot homepage scroll snapshot and restore flag for the `home -> detail -> home` path.
- Create: `src/composables/use-home-scroll-preserve.test.ts`
  - Unit-test arming, consuming, and clearing homepage scroll restore state.
- Create: `src/components/jav-library/ExpandableText.vue`
  - Render folded / expanded long text with clamp, overflow detection, and toggle button.
- Create: `src/components/jav-library/ExpandableText.test.ts`
  - Verify short-text pass-through and long-text expand / collapse behavior.
- Create: `src/components/jav-library/AppSidebar.test.ts`
  - Lock in the desktop sidebar’s stable navigation structure during compact toggles.
- Modify: `src/views/HomeView.vue`
  - Arm homepage restore before pushing to detail.
- Modify: `src/components/jav-library/HomepagePortal.vue`
  - Attach the homepage scroll restore composable to the existing `data-home-scroll-region`.
- Modify: `src/views/HomeView.test.ts`
  - Verify homepage restore arming on detail navigation and no arming on other browse interactions.
- Modify: `src/components/jav-library/DetailPanel.vue`
  - Replace raw summary paragraph rendering with `ExpandableText`.
- Modify: `src/components/jav-library/AppSidebar.vue`
  - Refactor the desktop sidebar to preserve a stable navigation DOM and hide text/badges instead of swapping major branches.
- Modify: `src/layouts/AppShell.vue`
  - Tighten desktop sidebar transition scope and duration so shell layout changes do less work during collapse.
- Modify: `docs/plan/2026-04-16-home-detail-sidebar-optimization-proposal.md`
  - Mark the proposal as implemented/planned in more concrete terms if the implementation reveals a small boundary adjustment.

---

### Task 1: Homepage Detail-Return Scroll Restore

**Files:**
- Create: `src/composables/use-home-scroll-preserve.ts`
- Create: `src/composables/use-home-scroll-preserve.test.ts`
- Modify: `src/views/HomeView.vue`
- Modify: `src/components/jav-library/HomepagePortal.vue`
- Modify: `src/views/HomeView.test.ts`

- [ ] **Step 1: Write the failing composable test**

Create `src/composables/use-home-scroll-preserve.test.ts` with:

```ts
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  armHomeDetailReturnRestore,
  consumeHomeDetailReturnRestore,
  readHomeScrollSnapshot,
  resetHomeScrollRestoreState,
  saveHomeScrollSnapshot,
} from "./use-home-scroll-preserve"

describe("use-home-scroll-preserve", () => {
  beforeEach(() => {
    resetHomeScrollRestoreState()
  })

  it("stores a snapshot and returns it only once after the detail-return restore is armed", () => {
    saveHomeScrollSnapshot(428)
    armHomeDetailReturnRestore()

    expect(readHomeScrollSnapshot()).toBe(428)
    expect(consumeHomeDetailReturnRestore()).toBe(428)
    expect(consumeHomeDetailReturnRestore()).toBeNull()
  })

  it("does not restore when the home detail-return flag was never armed", () => {
    saveHomeScrollSnapshot(512)

    expect(consumeHomeDetailReturnRestore()).toBeNull()
    expect(readHomeScrollSnapshot()).toBe(512)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
pnpm test -- src/composables/use-home-scroll-preserve.test.ts
```

Expected: FAIL because `src/composables/use-home-scroll-preserve.ts` does not exist yet.

- [ ] **Step 3: Write the minimal composable implementation**

Create `src/composables/use-home-scroll-preserve.ts` with:

```ts
import { nextTick, onBeforeUnmount, onMounted, type Ref } from "vue"

const HOME_SCROLL_TOP_KEY = "curated:home-scroll-top"
const HOME_DETAIL_RETURN_ARMED_KEY = "curated:home-detail-return-armed"

function readNumber(key: string): number | null {
  if (typeof window === "undefined") return null
  const raw = window.sessionStorage.getItem(key)
  if (!raw) return null
  const value = Number(raw)
  return Number.isFinite(value) ? value : null
}

function writeNumber(key: string, value: number) {
  if (typeof window === "undefined") return
  window.sessionStorage.setItem(key, String(Math.max(0, Math.round(value))))
}

export function saveHomeScrollSnapshot(scrollTop: number) {
  writeNumber(HOME_SCROLL_TOP_KEY, scrollTop)
}

export function readHomeScrollSnapshot(): number | null {
  return readNumber(HOME_SCROLL_TOP_KEY)
}

export function armHomeDetailReturnRestore() {
  if (typeof window === "undefined") return
  window.sessionStorage.setItem(HOME_DETAIL_RETURN_ARMED_KEY, "1")
}

export function consumeHomeDetailReturnRestore(): number | null {
  if (typeof window === "undefined") return null
  const armed = window.sessionStorage.getItem(HOME_DETAIL_RETURN_ARMED_KEY) === "1"
  window.sessionStorage.removeItem(HOME_DETAIL_RETURN_ARMED_KEY)
  if (!armed) return null
  return readHomeScrollSnapshot()
}

export function resetHomeScrollRestoreState() {
  if (typeof window === "undefined") return
  window.sessionStorage.removeItem(HOME_SCROLL_TOP_KEY)
  window.sessionStorage.removeItem(HOME_DETAIL_RETURN_ARMED_KEY)
}

export function useHomeScrollPreserve(options: { scrollElRef: Ref<HTMLElement | null> }) {
  const { scrollElRef } = options

  const persist = () => {
    const el = scrollElRef.value
    if (!el) return
    saveHomeScrollSnapshot(el.scrollTop)
  }

  onMounted(async () => {
    const restoreTop = consumeHomeDetailReturnRestore()
    if (restoreTop === null) return

    await nextTick()
    requestAnimationFrame(() => {
      const el = scrollElRef.value
      if (!el) return
      el.scrollTop = restoreTop
      setTimeout(() => {
        if (scrollElRef.value) {
          scrollElRef.value.scrollTop = restoreTop
        }
      }, 60)
    })
  })

  onBeforeUnmount(() => {
    persist()
  })

  return { persist }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:

```bash
pnpm test -- src/composables/use-home-scroll-preserve.test.ts
```

Expected: PASS

- [ ] **Step 5: Wire homepage navigation and scroll region**

Update `src/views/HomeView.vue`:

```ts
import { armHomeDetailReturnRestore } from "@/composables/use-home-scroll-preserve"

function openDetails(movieId: string) {
  armHomeDetailReturnRestore()
  void router.push({
    name: "detail",
    params: { id: movieId },
    query: { back: "home" },
  })
}
```

Update `src/components/jav-library/HomepagePortal.vue`:

```ts
import { ref } from "vue"
import { useHomeScrollPreserve } from "@/composables/use-home-scroll-preserve"

const homeScrollRegionRef = ref<HTMLElement | null>(null)
const { persist } = useHomeScrollPreserve({ scrollElRef: homeScrollRegionRef })

function onHomeScroll(event: Event) {
  const target = event.target
  if (!(target instanceof HTMLElement)) return
  persist()
}
```

And bind the container:

```vue
<div
  ref="homeScrollRegionRef"
  data-home-scroll-region
  class="h-full min-h-0 overflow-y-auto bg-background text-foreground"
  @scroll.passive="onHomeScroll"
>
```

- [ ] **Step 6: Add the view-level regression test**

Update `src/views/HomeView.test.ts` with:

```ts
const armHomeDetailReturnRestoreMock = vi.hoisted(() => vi.fn())

vi.mock("@/composables/use-home-scroll-preserve", () => ({
  armHomeDetailReturnRestore: armHomeDetailReturnRestoreMock,
  useHomeScrollPreserve: () => ({ persist: vi.fn() }),
}))
```

And add:

```ts
it("arms homepage restore before opening detail", async () => {
  const wrapper = mount(HomeView)

  await wrapper.get('[data-home-hero-open-details="m1"]').trigger("click")

  expect(armHomeDetailReturnRestoreMock).toHaveBeenCalledTimes(1)
  expect(routerPushMock).toHaveBeenCalledWith({
    name: "detail",
    params: { id: "m1" },
    query: { back: "home" },
  })
})
```

If the current hero/detail trigger selector differs, adapt the selector to the existing emitted detail-open element instead of adding a new behavior.

- [ ] **Step 7: Run targeted homepage tests**

Run:

```bash
pnpm test -- src/composables/use-home-scroll-preserve.test.ts src/views/HomeView.test.ts
```

Expected: PASS

- [ ] **Step 8: Commit**

Run:

```bash
git add src/composables/use-home-scroll-preserve.ts src/composables/use-home-scroll-preserve.test.ts src/views/HomeView.vue src/components/jav-library/HomepagePortal.vue src/views/HomeView.test.ts
git commit -m "feat: restore homepage scroll on detail return"
```

---

### Task 2: Fold Long Detail Summaries

**Files:**
- Create: `src/components/jav-library/ExpandableText.vue`
- Create: `src/components/jav-library/ExpandableText.test.ts`
- Modify: `src/components/jav-library/DetailPanel.vue`

- [ ] **Step 1: Write the failing component test**

Create `src/components/jav-library/ExpandableText.test.ts` with:

```ts
import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"
import ExpandableText from "./ExpandableText.vue"

describe("ExpandableText", () => {
  it("does not render a toggle for short text", () => {
    const wrapper = mount(ExpandableText, {
      props: { text: "Short summary.", collapsedLines: 5 },
    })

    expect(wrapper.text()).toContain("Short summary.")
    expect(wrapper.find("[data-expandable-toggle]").exists()).toBe(false)
  })

  it("expands and collapses long text", async () => {
    const wrapper = mount(ExpandableText, {
      props: {
        text: "Long summary ".repeat(80),
        collapsedLines: 5,
        forceExpandable: true,
      },
    })

    expect(wrapper.get("[data-expandable-content]").classes()).toContain("line-clamp-5")

    await wrapper.get("[data-expandable-toggle]").trigger("click")
    expect(wrapper.get("[data-expandable-content]").classes()).not.toContain("line-clamp-5")

    await wrapper.get("[data-expandable-toggle]").trigger("click")
    expect(wrapper.get("[data-expandable-content]").classes()).toContain("line-clamp-5")
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
pnpm test -- src/components/jav-library/ExpandableText.test.ts
```

Expected: FAIL because `src/components/jav-library/ExpandableText.vue` does not exist yet.

- [ ] **Step 3: Write the minimal expandable text component**

Create `src/components/jav-library/ExpandableText.vue` with:

```vue
<script setup lang="ts">
import { computed, ref, watch } from "vue"

const props = withDefaults(
  defineProps<{
    text: string
    collapsedLines?: number
    forceExpandable?: boolean
    expandLabel?: string
    collapseLabel?: string
  }>(),
  {
    collapsedLines: 5,
    forceExpandable: false,
    expandLabel: "展开简介",
    collapseLabel: "收起简介",
  },
)

const expanded = ref(false)

watch(
  () => props.text,
  () => {
    expanded.value = false
  },
)

const normalizedText = computed(() => props.text.trim())
const shouldShowToggle = computed(
  () => props.forceExpandable || normalizedText.value.length > 180,
)
</script>

<template>
  <div v-if="normalizedText" class="min-w-0">
    <p
      data-expandable-content
      class="text-pretty text-sm leading-6 text-muted-foreground"
      :class="!expanded && shouldShowToggle ? `line-clamp-${collapsedLines}` : ''"
    >
      {{ normalizedText }}
    </p>
    <button
      v-if="shouldShowToggle"
      data-expandable-toggle
      type="button"
      class="mt-2 inline-flex text-sm font-medium text-primary underline-offset-4 hover:underline"
      @click="expanded = !expanded"
    >
      {{ expanded ? collapseLabel : expandLabel }}
    </button>
  </div>
</template>
```

- [ ] **Step 4: Run the test to verify it passes**

Run:

```bash
pnpm test -- src/components/jav-library/ExpandableText.test.ts
```

Expected: PASS

- [ ] **Step 5: Replace the raw detail summary paragraph**

Update `src/components/jav-library/DetailPanel.vue`:

```ts
import ExpandableText from "@/components/jav-library/ExpandableText.vue"
```

Replace:

```vue
<p
  v-if="summaryDisplay"
  class="text-pretty text-sm leading-6 text-muted-foreground"
>
  {{ summaryDisplay }}
</p>
```

With:

```vue
<ExpandableText
  v-if="summaryDisplay"
  :text="summaryDisplay"
  :collapsed-lines="5"
  :expand-label="t('detailPanel.expandSummary')"
  :collapse-label="t('detailPanel.collapseSummary')"
/>
```

If the i18n keys do not exist yet, add them in the same change set to the project’s active locale messages instead of hard-coding the labels in `DetailPanel`.

- [ ] **Step 6: Run the focused tests**

Run:

```bash
pnpm test -- src/components/jav-library/ExpandableText.test.ts src/views/HomeView.test.ts
```

Expected: PASS

- [ ] **Step 7: Commit**

Run:

```bash
git add src/components/jav-library/ExpandableText.vue src/components/jav-library/ExpandableText.test.ts src/components/jav-library/DetailPanel.vue
git commit -m "feat: fold long movie summaries"
```

---

### Task 3: Reduce Desktop Sidebar Collapse Jank

**Files:**
- Create: `src/components/jav-library/AppSidebar.test.ts`
- Modify: `src/components/jav-library/AppSidebar.vue`
- Modify: `src/layouts/AppShell.vue`
- Modify: `src/layouts/AppShell.test.ts`

- [ ] **Step 1: Write the failing sidebar structure test**

Create `src/components/jav-library/AppSidebar.test.ts` with:

```ts
import { mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import AppSidebar from "./AppSidebar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  RouterLink: {
    name: "RouterLink",
    props: ["to"],
    template: "<a :data-to=\"JSON.stringify(to)\"><slot /></a>",
  },
  useRoute: () => ({ name: "home", query: {} }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: computed(() => []),
    trashedMovies: computed(() => []),
  }),
}))

vi.mock("@/composables/use-backend-health", () => ({
  useBackendHealth: () => ({
    useWebApi: true,
    status: ref("online"),
    probing: ref(false),
    versionDisplay: ref("1.2.4"),
    checkNow: vi.fn(),
  }),
}))

describe("AppSidebar", () => {
  it("keeps the same core nav link count when toggling compact mode", async () => {
    const wrapper = mount(AppSidebar, { props: { compact: false } })
    const expandedLinks = wrapper.findAll("[data-sidebar-nav-link]")

    await wrapper.setProps({ compact: true })
    const compactLinks = wrapper.findAll("[data-sidebar-nav-link]")

    expect(expandedLinks).toHaveLength(compactLinks.length)
  })
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
pnpm test -- src/components/jav-library/AppSidebar.test.ts
```

Expected: FAIL because `AppSidebar.vue` does not expose a stable nav link structure yet.

- [ ] **Step 3: Refactor `AppSidebar` toward a mostly stable navigation tree**

Update `src/components/jav-library/AppSidebar.vue` so each nav item uses one shared `RouterLink` structure for both expanded and compact states:

```vue
<RouterLink
  v-for="item in sidebarNavGroups.browse"
  :key="item.page"
  :to="getNavigationTarget(item.page)"
  data-sidebar-nav-link
  class="group flex min-h-10 w-full min-w-0 items-center gap-2 rounded-2xl px-3 transition-colors"
  :class="[
    isActive(item.page) ? 'bg-sidebar-accent text-sidebar-accent-foreground' : 'hover:bg-sidebar-accent/60',
    props.compact ? 'justify-center px-2' : 'justify-between',
  ]"
>
  <span class="flex min-w-0 items-center gap-2 overflow-hidden">
    <component :is="item.icon" class="size-5 shrink-0" />
    <span
      class="truncate transition-[opacity,max-width] duration-200 motion-reduce:transition-none"
      :class="props.compact ? 'pointer-events-none max-w-0 opacity-0' : 'max-w-[10rem] opacity-100'"
      :aria-hidden="props.compact"
    >
      {{ item.label }}
    </span>
  </span>
  <Badge
    v-if="item.hint"
    class="transition-opacity duration-150 motion-reduce:transition-none"
    :class="props.compact ? 'pointer-events-none opacity-0' : 'opacity-100'"
  >
    {{ item.hint }}
  </Badge>
</RouterLink>
```

Apply the same pattern to the `yours` section and keep the settings entry structurally aligned with the same collapse-by-visibility approach. Do not preserve the current "large expanded branch vs icon-only branch" split for desktop navigation items.

- [ ] **Step 4: Tighten shell/sidebar animation scope**

Update `src/layouts/AppShell.vue`:

```ts
const shellGridClass = computed(() => {
  const base =
    "grid h-full min-h-0 grid-cols-1 lg:transition-[grid-template-columns] lg:duration-200 lg:ease-out motion-reduce:lg:transition-none"
  if (!isLgUp.value) {
    return base
  }
  return `${base} ${desktopSidebarCollapsed.value ? "lg:grid-cols-[4.75rem_minmax(0,1fr)]" : "lg:grid-cols-[304px_minmax(0,1fr)]"}`
})
```

And reduce transition scope in `AppSidebar.vue`:

```vue
<aside
  class="flex h-full min-h-0 w-full min-w-0 flex-col overflow-x-hidden bg-sidebar text-sidebar-foreground motion-reduce:transition-none"
  :class="props.compact ? 'items-center px-2 pb-3 pt-0' : 'px-3.5 pb-3.5 pt-0'"
>
```

That is: remove broad `transition-[padding] duration-300 ease-in-out` from the sidebar root, and let the shell width transition plus text fade carry the interaction instead.

- [ ] **Step 5: Update the shell test to lock in the new transition contract**

Update `src/layouts/AppShell.test.ts` with:

```ts
it("uses the tightened desktop sidebar grid transition", () => {
  const wrapper = shallowMount(AppShell)
  const split = wrapper.get('[data-shell-layout="split"]')

  expect(split.classes().join(" ")).toContain("lg:duration-200")
  expect(split.classes().join(" ")).not.toContain("lg:duration-300")
})
```

- [ ] **Step 6: Run the focused sidebar tests**

Run:

```bash
pnpm test -- src/components/jav-library/AppSidebar.test.ts src/layouts/AppShell.test.ts
```

Expected: PASS

- [ ] **Step 7: Run full frontend verification for the combined feature**

Run:

```bash
pnpm test -- src/composables/use-home-scroll-preserve.test.ts src/views/HomeView.test.ts src/components/jav-library/ExpandableText.test.ts src/components/jav-library/AppSidebar.test.ts src/layouts/AppShell.test.ts
pnpm typecheck
pnpm lint
pnpm build
```

Expected:

- targeted Vitest command: PASS
- `pnpm typecheck`: PASS
- `pnpm lint`: PASS
- `pnpm build`: PASS

- [ ] **Step 8: Commit**

Run:

```bash
git add src/components/jav-library/AppSidebar.vue src/components/jav-library/AppSidebar.test.ts src/layouts/AppShell.vue src/layouts/AppShell.test.ts src/components/jav-library/DetailPanel.vue src/components/jav-library/ExpandableText.vue src/components/jav-library/ExpandableText.test.ts src/composables/use-home-scroll-preserve.ts src/composables/use-home-scroll-preserve.test.ts src/views/HomeView.vue src/components/jav-library/HomepagePortal.vue src/views/HomeView.test.ts docs/plan/2026-04-16-home-detail-sidebar-optimization-proposal.md
git commit -m "feat: optimize home detail and sidebar interactions"
```

---

## Self-Review

### Spec Coverage

- Homepage scroll restore limited to `home -> detail -> home`: covered in Task 1
- Long detail summary fold/unfold behavior: covered in Task 2
- Desktop sidebar collapse jank reduction without mobile drawer changes: covered in Task 3

### Placeholder Scan

- No `TODO` / `TBD` placeholders remain
- Every code-changing step contains concrete code or replacement snippets
- Every verification step names exact commands

### Type / Naming Consistency

- Homepage restore naming is consistent across:
  - `armHomeDetailReturnRestore`
  - `consumeHomeDetailReturnRestore`
  - `useHomeScrollPreserve`
- Expandable text naming is consistent across:
  - `ExpandableText.vue`
  - `data-expandable-content`
  - `data-expandable-toggle`
- Sidebar test contract is consistent around:
  - `data-sidebar-nav-link`
  - `compact`
  - `lg:duration-200`
