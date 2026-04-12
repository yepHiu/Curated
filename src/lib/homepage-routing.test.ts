import { computed, reactive, ref } from "vue"
import { mount, shallowMount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import AppSidebar from "@/components/jav-library/AppSidebar.vue"
import AppShell from "@/layouts/AppShell.vue"
import router from "@/router"

const routerMocks = vi.hoisted(() => ({
  route: {
    fullPath: "/",
    name: "home",
    params: {},
    path: "/",
    query: {},
  },
  replace: vi.fn(),
}))

vi.mock("@vueuse/core", async () => {
  const actual = await vi.importActual<typeof import("@vueuse/core")>("@vueuse/core")
  const { ref } = await vi.importActual<typeof import("vue")>("vue")

  return {
    ...actual,
    useMediaQuery: () => ref(true),
  }
})

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("en"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", async () => {
  const actual = await vi.importActual<typeof import("vue-router")>("vue-router")

  return {
    ...actual,
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
  }
})

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: computed(() => []),
    trashedMovies: computed(() => []),
    getMovieById: vi.fn(),
  }),
}))

vi.mock("@/composables/use-library-watch-toasts", () => ({
  useLibraryWatchToasts: vi.fn(),
}))

vi.mock("@/composables/use-theme", () => ({
  useTheme: () => ({
    resolvedMode: { value: "light" },
    setThemePreference: vi.fn(),
  }),
}))

vi.mock("@/composables/use-backend-health", () => ({
  useBackendHealth: () => ({
    useWebApi: false,
    status: ref("mock"),
    probing: ref(false),
    versionDisplay: ref(""),
    checkNow: vi.fn(),
  }),
}))

vi.mock("@/lib/curated-frames/db", () => ({
  countCuratedFrames: vi.fn(async () => 0),
}))

vi.mock("@/lib/curated-frames/revision", () => ({
  curatedFramesRevision: ref(0),
}))

vi.mock("@/lib/playback-progress-storage", () => ({
  listSortedByUpdatedDesc: vi.fn(() => []),
  playbackProgressRevision: ref(0),
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
  Button: {
    name: "Button",
    props: ["asChild"],
    template: "<button><slot /></button>",
  },
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { name: "Badge", template: "<span><slot /></span>" },
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

describe("homepage routing", () => {
  beforeEach(() => {
    routerMocks.replace.mockClear()
    Object.assign(routerMocks.route, reactive({
      fullPath: "/",
      name: "home",
      params: {},
      path: "/",
      query: {},
    }))
  })

  it("resolves the root path to the homepage route", () => {
    expect(router.resolve("/").name).toBe("home")
  })

  it("renders the homepage entry in the browse sidebar group", () => {
    const wrapper = mount(AppSidebar)

    expect(wrapper.text()).toContain("nav.home")
  })

  it("treats the homepage as a primary route without back navigation padding", () => {
    const wrapper = shallowMount(AppShell)

    expect(wrapper.find('[data-router-view-frame]').classes().join(" ")).not.toContain("px-4")
    expect(wrapper.text()).not.toContain("shell.backLibrary")
  })
})
