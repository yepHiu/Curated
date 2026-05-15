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

const gamepadFocusNavigationMock = vi.hoisted(() => vi.fn())
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

vi.mock("@/composables/use-library-storage-status-alerts", () => ({
  useLibraryStorageStatusAlerts: vi.fn(),
}))

vi.mock("@/composables/use-theme", () => ({
  useTheme: () => ({
    resolvedMode: { value: "light" },
    setThemePreference: vi.fn(),
  }),
}))

vi.mock("@/composables/use-gamepad-focus-navigation", () => ({
  useGamepadFocusNavigation: gamepadFocusNavigationMock,
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

vi.mock("@/components/jav-library/MovieImportDialog.vue", () => ({
  default: { name: "MovieImportDialog", template: "<button data-movie-import>import.trigger</button>" },
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
    gamepadFocusNavigationMock.mockClear()
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

  it("renders the movie import entry near the header actions", () => {
    const wrapper = shallowMount(AppShell)

    expect(wrapper.findComponent({ name: "MovieImportDialog" }).exists()).toBe(true)
  })

  it("mounts global gamepad focus navigation in the app shell", () => {
    shallowMount(AppShell)

    expect(gamepadFocusNavigationMock).toHaveBeenCalledTimes(1)
  })

  it("uses the tightened desktop sidebar grid transition", () => {
    const wrapper = shallowMount(AppShell)
    const split = wrapper.get('[data-shell-layout="split"]')

    expect(split.classes().join(" ")).toContain("lg:duration-200")
    expect(split.classes().join(" ")).not.toContain("lg:duration-300")
  })

  it("uses density variables for desktop shell dimensions", () => {
    const wrapper = shallowMount(AppShell)
    const split = wrapper.get('[data-shell-layout="split"]')
    const header = wrapper.get("[data-shell-header]")
    const splitClasses = split.classes().join(" ")
    const headerClasses = header.classes().join(" ")

    expect(splitClasses).toContain("lg:grid-cols-[var(--app-sidebar-width)_minmax(0,1fr)]")
    expect(splitClasses).not.toContain("lg:grid-cols-[304px_minmax(0,1fr)]")
    expect(headerClasses).toContain("min-h-[var(--app-header-min-height)]")
    expect(headerClasses).not.toContain("min-h-[4.5rem]")
  })

  it("does not wrap batch-toolbar routes in the global workspace padding", () => {
    routerMocks.route.name = "library"
    const wrapper = shallowMount(AppShell)

    const contentFrame = wrapper.get("[data-router-view-frame]")

    expect(contentFrame.classes().join(" ")).not.toContain("px-4")
    expect(contentFrame.classes().join(" ")).not.toContain("py-4")
  })
})
