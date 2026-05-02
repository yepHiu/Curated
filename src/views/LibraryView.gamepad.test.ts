import { mount } from "@vue/test-utils"
import { computed, nextTick, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { GamepadDirection, StandardGamepadButtonName } from "@/lib/gamepad/standard-gamepad"
import type { Movie } from "@/domain/movie/types"
import LibraryView from "./LibraryView.vue"

const gamepadMocks = vi.hoisted(() => ({
  useGamepad: vi.fn(),
  buttonHandlers: new Map<string | number, () => void>(),
  directionHandlers: new Map<GamepadDirection, () => void>(),
  enabled: { value: true, __v_isRef: true },
}))

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  route: {
    name: "library",
    path: "/library",
    query: { selected: "movie-2" } as Record<string, string>,
  },
}))

const serviceState = vi.hoisted(() => ({
  movies: [] as Movie[],
}))

vi.mock("@/composables/use-gamepad", () => ({
  useGamepad: gamepadMocks.useGamepad,
}))

vi.mock("@/lib/gamepad/gamepad-settings", () => ({
  useGamepadControlsPreference: () => ({
    gamepadControlsEnabled: gamepadMocks.enabled,
    setGamepadControlsEnabled: vi.fn(),
  }),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routerMocks.route,
  useRouter: () => routerMocks,
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: vi.fn(),
}))

vi.mock("@/composables/use-scan-task-tracker", () => ({
  useScanTaskTracker: () => ({
    activeTask: ref(null),
    start: vi.fn(),
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: computed(() => serviceState.movies),
    trashedMovies: computed(() => []),
    loadError: computed(() => null),
    ensureTrashLoaded: vi.fn(),
    toggleFavorite: vi.fn(),
  }),
}))

vi.mock("@/components/jav-library/LibraryPage.vue", () => ({
  default: {
    name: "LibraryPage",
    props: ["selectedMovie", "batchMode", "batchSelectedIds"],
    emits: ["columnsChange"],
    mounted(this: { $emit: (event: string, ...args: unknown[]) => void }) {
      this.$emit("columnsChange", 5)
    },
    template: `
      <div data-library-page :data-batch-mode="batchMode ? 'true' : 'false'">
        <p data-selected>{{ selectedMovie?.id }}</p>
      </div>
    `,
  },
}))

vi.mock("@/components/jav-library/LibraryBatchActionBar.vue", () => ({
  default: {
    name: "LibraryBatchActionBar",
    props: ["selectedCount"],
    template: "<div data-batch-bar>{{ selectedCount }}</div>",
  },
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

function makeMovie(id: string, index: number): Movie {
  return {
    id,
    code: id,
    title: `Movie ${id}`,
    studio: "",
    actors: [],
    tags: [],
    userTags: [],
    runtimeMinutes: 0,
    rating: 0,
    summary: "",
    isFavorite: false,
    addedAt: `2026-01-${String(31 - index).padStart(2, "0")}`,
    location: "",
    resolution: "",
    year: 0,
    tone: "",
    coverClass: "",
  }
}

function pressDirection(direction: GamepadDirection) {
  const handler = gamepadMocks.directionHandlers.get(direction)
  if (!handler) throw new Error(`missing direction handler for ${direction}`)
  handler()
}

function pressButton(button: StandardGamepadButtonName) {
  const handler = gamepadMocks.buttonHandlers.get(button)
  if (!handler) throw new Error(`missing button handler for ${button}`)
  handler()
}

describe("LibraryView gamepad grid navigation", () => {
  beforeEach(() => {
    serviceState.movies = Array.from({ length: 12 }, (_, index) => makeMovie(`movie-${index}`, index))
    routerMocks.route.name = "library"
    routerMocks.route.query = { selected: "movie-2" }
    routerMocks.push.mockReset()
    routerMocks.replace.mockReset()
    gamepadMocks.buttonHandlers.clear()
    gamepadMocks.directionHandlers.clear()
    gamepadMocks.enabled.value = true
    gamepadMocks.useGamepad.mockReset()
    gamepadMocks.useGamepad.mockReturnValue({
      connected: ref(true),
      gamepadId: ref("DualSense"),
      lastInputAt: ref(0),
      supported: ref(true),
      onButtonPress: vi.fn((button: string | number, handler: () => void) => {
        gamepadMocks.buttonHandlers.set(button, handler)
        return vi.fn()
      }),
      onDirectionPress: vi.fn((direction: GamepadDirection, handler: () => void) => {
        gamepadMocks.directionHandlers.set(direction, handler)
        return vi.fn()
      }),
      rumble: vi.fn(),
      stop: vi.fn(),
    })
  })

  it("moves the selected movie by the current masonry column count", async () => {
    mount(LibraryView)
    await nextTick()

    pressDirection("down")

    expect(routerMocks.replace).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "library",
        query: expect.objectContaining({ selected: "movie-7" }),
      }),
    )
  })

  it("opens details for the currently selected movie with Cross", () => {
    mount(LibraryView)

    pressButton("cross")

    expect(routerMocks.push).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "detail",
        params: { id: "movie-2" },
      }),
    )
  })

  it("uses Square to enter batch mode and select the current movie", async () => {
    const wrapper = mount(LibraryView)

    pressButton("square")
    await nextTick()

    expect(wrapper.get("[data-batch-bar]").text()).toBe("1")
  })

  it("uses R2 to jump a page through the selected movie list", () => {
    mount(LibraryView)

    pressButton("r2")

    expect(routerMocks.replace).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "library",
        query: expect.objectContaining({ selected: "movie-11" }),
      }),
    )
  })
})
