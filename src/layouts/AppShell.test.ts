import { mount, shallowMount } from "@vue/test-utils"
import { nextTick, reactive } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import AppShell from "./AppShell.vue"

const routerMocks = vi.hoisted(() => {
  const replace = vi.fn()
  const route = {
    fullPath: "/library",
    name: "library",
    params: {},
    path: "/library",
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
    props: ["to"],
    template: "<a :data-to=\"JSON.stringify(to)\"><slot /></a>",
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

vi.mock("@/components/notification-center/NotificationCenter.vue", () => ({
  default: { name: "NotificationCenter", template: "<div data-notification-center />" },
}))

vi.mock("@/components/jav-library/MovieImportDialog.vue", () => ({
  default: { name: "MovieImportDialog", template: "<button data-movie-import>import.trigger</button>" },
}))

vi.mock("@/components/jav-library/ImportMenu.vue", () => ({
  default: { name: "ImportMenu", template: "<div data-import-menu />" },
}))

vi.mock("@/components/ui/sonner", () => ({
  Toaster: { name: "Toaster", template: "<div />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button><slot /></button>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template:
      "<input :value=\"modelValue\" @input=\"$emit('update:modelValue', $event.target.value)\" />",
  },
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
      fullPath: "/library",
      name: "library",
      params: {},
      path: "/library",
      query: {},
    })
  })

  it("keeps an active tag filter instead of rewriting it into a text search", async () => {
    shallowMount(AppShell)

    await nextTick()
    routerMocks.replace.mockClear()
    routerMocks.route.fullPath = "/library?tag=Drama"
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

  it("renders the import menu near the header actions", () => {
    const wrapper = shallowMount(AppShell)

    expect(wrapper.findComponent({ name: "ImportMenu" }).exists()).toBe(true)
  })

  it("shows a detail back link on primary browse drill-down routes", () => {
    routerMocks.route.fullPath =
      "/library?actor=Actor%20A&back=detail&browse=favorites&selected=movie-1"
    routerMocks.route.name = "library"
    routerMocks.route.path = "/library"
    routerMocks.route.query = {
      actor: "Actor A",
      back: "detail",
      browse: "favorites",
      selected: "movie-1",
    }

    const wrapper = mount(AppShell)

    expect(wrapper.text()).toContain("shell.backDetail")
    expect(JSON.parse(wrapper.get("a[data-to]").attributes("data-to") ?? "{}")).toEqual(
      {
        name: "detail",
        params: { id: "movie-1" },
        query: {
          actor: "Actor A",
          browse: "favorites",
          selected: "movie-1",
        },
      },
    )
  })

  it("does not show the global back-to-library link on the comic library root", () => {
    routerMocks.route.fullPath = "/comics"
    routerMocks.route.name = "comics"
    routerMocks.route.path = "/comics"
    routerMocks.route.query = {}

    const wrapper = mount(AppShell)

    expect(wrapper.text()).not.toContain("shell.backLibrary")
    expect(wrapper.find('a[data-to*="\\"name\\":\\"library\\""]').exists()).toBe(false)
  })

  it("returns from comic detail pages to the comic library", () => {
    routerMocks.route.fullPath = "/comics/comic-1"
    routerMocks.route.name = "comic-detail"
    routerMocks.route.params = { id: "comic-1" }
    routerMocks.route.path = "/comics/comic-1"
    routerMocks.route.query = {}

    const wrapper = mount(AppShell)

    expect(wrapper.text()).toContain("shell.backComics")
    expect(JSON.parse(wrapper.get("a[data-to]").attributes("data-to") ?? "{}")).toEqual({
      name: "comics",
    })
  })

  it("returns from comic reader pages with a flush workspace and no shell header", () => {
    routerMocks.route.fullPath =
      "/comics/comic-1/read/3?returnTo=%2Fcomics%2Fcomic-1"
    routerMocks.route.name = "comic-reader"
    routerMocks.route.params = { id: "comic-1", pageIndex: "3" }
    routerMocks.route.path = "/comics/comic-1/read/3"
    routerMocks.route.query = { returnTo: "/comics/comic-1" }

    const wrapper = mount(AppShell)
    const contentFrame = wrapper.get("[data-router-view-frame]")

    expect(wrapper.find("[data-shell-header]").exists()).toBe(false)
    expect(contentFrame.classes().join(" ")).not.toContain("px-[var(--app-page-px)]")
    expect(contentFrame.classes().join(" ")).not.toContain("py-[var(--app-page-py)]")
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

  it.each(["library", "comics", "photos", "comic-reader", "photo-viewer"])(
    "does not wrap %s routes in the global workspace padding",
    (routeName) => {
      routerMocks.route.name = routeName
      routerMocks.route.path = `/${routeName}`
      routerMocks.route.fullPath = `/${routeName}`
      routerMocks.route.query = {}

      const wrapper = shallowMount(AppShell)

      const contentFrame = wrapper.get("[data-router-view-frame]")
      const contentFrameClasses = contentFrame.classes().join(" ")

      expect(contentFrameClasses).not.toContain("px-[var(--app-page-px)]")
      expect(contentFrameClasses).not.toContain("py-[var(--app-page-py)]")
    },
  )

  it("uses the shell search for the comic library route", () => {
    routerMocks.route.fullPath = "/comics?q=rain"
    routerMocks.route.name = "comics"
    routerMocks.route.path = "/comics"
    routerMocks.route.query = { q: "rain" }

    const wrapper = shallowMount(AppShell)

    const searchInput = wrapper.get('[placeholder="comics.searchPlaceholder"]')

    expect(searchInput.attributes("modelvalue")).toBe("rain")
  })

  it("uses the shell search for the photo library route", () => {
    routerMocks.route.fullPath = "/photos?q=beach"
    routerMocks.route.name = "photos"
    routerMocks.route.path = "/photos"
    routerMocks.route.query = { q: "beach" }

    const wrapper = shallowMount(AppShell)

    const searchInput = wrapper.get('[placeholder="photos.searchPlaceholder"]')

    expect(searchInput.attributes("modelvalue")).toBe("beach")
  })

  it("treats actor detail as an owned page with actor-library back navigation", () => {
    routerMocks.route.fullPath = "/actors/Mina%20Kaze"
    routerMocks.route.name = "actor-detail"
    routerMocks.route.path = "/actors/Mina%20Kaze"
    routerMocks.route.params = { actorName: "Mina Kaze" }
    routerMocks.route.query = {}

    const wrapper = mount(AppShell)
    const contentFrame = wrapper.get("[data-router-view-frame]")

    expect(wrapper.text()).toContain("shell.backActors")
    expect(JSON.parse(wrapper.get("a[data-to]").attributes("data-to") ?? "{}")).toEqual({
      name: "actors",
    })
    expect(contentFrame.classes().join(" ")).not.toContain("px-4")
    expect(contentFrame.classes().join(" ")).not.toContain("py-4")
  })

  it("keeps detail-origin actor pages wired back to the originating detail", () => {
    routerMocks.route.fullPath =
      "/actors/Mina%20Kaze?back=detail&browse=favorites&selected=movie-1"
    routerMocks.route.name = "actor-detail"
    routerMocks.route.path = "/actors/Mina%20Kaze"
    routerMocks.route.params = { actorName: "Mina Kaze" }
    routerMocks.route.query = {
      back: "detail",
      browse: "favorites",
      selected: "movie-1",
    }

    const wrapper = mount(AppShell)

    expect(wrapper.text()).toContain("shell.backDetail")
    expect(JSON.parse(wrapper.get("a[data-to]").attributes("data-to") ?? "{}")).toEqual({
      name: "detail",
      params: { id: "movie-1" },
      query: {
        browse: "favorites",
        selected: "movie-1",
      },
    })
  })
})
