import { flushPromises, mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"
import LibraryView from "./LibraryView.vue"

const pushAppToastMock = vi.hoisted(() => vi.fn())
const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
}))
const serviceState = vi.hoisted(() => ({
  loadError: null as string | null,
}))
const serviceMocks = vi.hoisted(() => ({
  toggleFavorite: vi.fn(),
  ensureTrashLoaded: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => ({
    name: "library",
    path: "/library",
    query: {},
  }),
  useRouter: () => routerMocks,
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: pushAppToastMock,
}))

vi.mock("@/composables/use-scan-task-tracker", () => ({
  useScanTaskTracker: () => ({
    activeTask: ref(null),
    start: vi.fn(),
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: computed(() => []),
    trashedMovies: computed(() => []),
    loadError: computed(() => serviceState.loadError),
    ensureTrashLoaded: serviceMocks.ensureTrashLoaded,
    toggleFavorite: serviceMocks.toggleFavorite,
  }),
}))

vi.mock("@/components/jav-library/LibraryPage.vue", () => ({
  default: {
    name: "LibraryPage",
    emits: ["toggleFavorite"],
    template:
      '<button type="button" data-toggle-favorite @click="$emit(\'toggleFavorite\', { movieId: \'movie-1\', nextValue: true })">Favorite</button>',
  },
}))

vi.mock("@/components/jav-library/LibraryBatchActionBar.vue", () => ({
  default: { name: "LibraryBatchActionBar", template: "<div />" },
}))

vi.mock("@/components/jav-library/MovieDeleteConfirmDialog.vue", () => ({
  default: { name: "MovieDeleteConfirmDialog", template: "<div />" },
}))

vi.mock("@/components/jav-library/MovieEditDialog.vue", () => ({
  default: { name: "MovieEditDialog", template: "<div />" },
}))

vi.mock("@/components/jav-library/MovieLibraryContextMenu.vue", () => ({
  default: { name: "MovieLibraryContextMenu", template: "<div />" },
}))

afterEach(() => {
  serviceState.loadError = null
  serviceMocks.toggleFavorite.mockReset()
  serviceMocks.ensureTrashLoaded.mockReset()
  pushAppToastMock.mockReset()
  routerMocks.push.mockReset()
  routerMocks.replace.mockReset()
  vi.restoreAllMocks()
})

describe("LibraryView feedback", () => {
  it("shows a library load error banner when the service reports one", () => {
    serviceState.loadError = "Failed to load library"

    const wrapper = mount(LibraryView)

    expect(wrapper.get("[data-library-load-error]").text()).toBe("Failed to load library")
  })

  it("shows a destructive toast when favorite toggling fails", async () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    serviceMocks.toggleFavorite.mockRejectedValueOnce(new Error("favorite failed"))
    const wrapper = mount(LibraryView)

    await wrapper.get("[data-toggle-favorite]").trigger("click")
    await flushPromises()

    expect(pushAppToastMock).toHaveBeenCalledWith("favorite failed", {
      variant: "destructive",
    })
  })
})
