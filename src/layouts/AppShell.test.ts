import { shallowMount } from "@vue/test-utils"
import { nextTick, reactive } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import AppShell from "./AppShell.vue"

const routerMocks = vi.hoisted(() => {
  const replace = vi.fn()
  const route = {
    fullPath: "/tags",
    name: "tags",
    params: {},
    path: "/tags",
    query: {},
  }

  return { replace, route }
})

const mediaQueryMatches = vi.hoisted(() => ({ value: true, __v_isRef: true }))

vi.mock("@vueuse/core", async () => {
  const actual = await vi.importActual<typeof import("@vueuse/core")>("@vueuse/core")

  return {
    ...actual,
    useMediaQuery: () => mediaQueryMatches,
  }
})

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: { value: "zh-CN" },
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  RouterLink: {
    name: "RouterLink",
    template: "<a><slot /></a>",
  },
  RouterView: {
    name: "RouterView",
    template: "<div />",
  },
  useRoute: () => routerMocks.route,
  useRouter: () => ({
    replace: routerMocks.replace,
  }),
}))

vi.mock("@/services/library-service", async () => {
  const { ref } = await vi.importActual<typeof import("vue")>("vue")

  return {
    useLibraryService: () => ({
      getMovieById: vi.fn(),
      movies: ref([]),
    }),
  }
})

vi.mock("@/composables/use-library-watch-toasts", () => ({
  useLibraryWatchToasts: vi.fn(),
}))

vi.mock("@/composables/use-theme", () => ({
  useTheme: () => ({
    resolvedMode: { value: "light" },
    setThemePreference: vi.fn(),
  }),
}))

vi.mock("@/components/jav-library/AppSidebar.vue", () => ({
  default: {
    name: "AppSidebar",
    props: ["compact", "showCollapseToggle"],
    template: `
      <aside
        :data-compact="compact ? 'true' : 'false'"
        :data-show-collapse-toggle="showCollapseToggle ? 'true' : 'false'"
      />
    `,
  },
}))

vi.mock("@/components/dev/DevEnvironmentBadge.vue", () => ({
  default: { name: "DevEnvironmentBadge", template: "<div />" },
}))

vi.mock("@/components/dev/DevPerformanceBar.vue", () => ({
  default: { name: "DevPerformanceBar", template: "<div />" },
}))

vi.mock("@/components/jav-library/ScanProgressDock.vue", () => ({
  default: { name: "ScanProgressDock", template: "<div />" },
}))

vi.mock("@/components/ui/sonner", () => ({
  Toaster: { name: "Toaster", template: "<div />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button><slot /></button>" },
}))

vi.mock("@/components/ui/switch", () => ({
  Switch: { name: "Switch", template: "<button />" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: { name: "Input", template: "<input />" },
}))

vi.mock("@/components/ui/scroll-area", () => ({
  ScrollArea: { name: "ScrollArea", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/separator", () => ({
  Separator: { name: "Separator", template: "<hr />" },
}))

describe("AppShell library search route sync", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mediaQueryMatches.value = true
    routerMocks.replace.mockClear()
    routerMocks.route = reactive({
      fullPath: "/tags",
      name: "tags",
      params: {},
      path: "/tags",
      query: {},
    })
  })

  it("keeps an active tag filter instead of rewriting it into a text search", async () => {
    shallowMount(AppShell)

    await nextTick()
    routerMocks.replace.mockClear()
    routerMocks.route.fullPath = "/tags?tag=Drama"
    routerMocks.route.query = { tag: "Drama" }

    await nextTick()
    await vi.advanceTimersByTimeAsync(300)

    expect(routerMocks.replace).not.toHaveBeenCalled()
  })

  it("renders a split shell instead of the previous rounded workspace container", () => {
    const wrapper = shallowMount(AppShell)

    expect(wrapper.find('[data-shell-layout="split"]').exists()).toBe(true)
    expect(wrapper.html()).not.toContain(
      "rounded-[1.75rem] border border-border/60 bg-background/95",
    )
  })

  it("moves the desktop sidebar collapse toggle into the content header", () => {
    const wrapper = shallowMount(AppShell)

    expect(wrapper.find('[data-sidebar-toggle="desktop"]').exists()).toBe(true)
    expect(wrapper.find('[data-show-collapse-toggle="true"]').exists()).toBe(false)
  })

  it("does not wrap batch-toolbar routes in the global workspace padding", () => {
    routerMocks.route.name = "library"
    const wrapper = shallowMount(AppShell)

    const contentFrame = wrapper.get("[data-router-view-frame]")

    expect(contentFrame.classes().join(" ")).not.toContain("px-4")
    expect(contentFrame.classes().join(" ")).not.toContain("py-4")
  })
})
