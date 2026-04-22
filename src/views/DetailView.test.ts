import { mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import DetailView from "./DetailView.vue"
import type { Movie } from "@/domain/movie/types"

function makeMovie(overrides: Partial<Movie> = {}): Movie {
  return {
    id: "movie-1",
    title: "Movie 1",
    code: "CODE-1",
    studio: "Studio",
    actors: ["Actor A"],
    tags: ["meta-a"],
    userTags: ["user-a"],
    runtimeMinutes: 120,
    rating: 4.5,
    metadataRating: 4.5,
    userRating: undefined,
    summary: "Summary",
    isFavorite: false,
    addedAt: "2026-04-01T00:00:00.000Z",
    location: "D:/Library/movie-1.mp4",
    resolution: "1080p",
    year: 2026,
    releaseDate: "2026-04-01",
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
    ...overrides,
  }
}

const routeState = vi.hoisted(() => ({
  name: "detail",
  params: { id: "movie-1" },
  query: {
    browse: "favorites",
    q: "star",
    tab: "top-rated",
    selected: "movie-1",
  },
}))

const routerPushMock = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    push: routerPushMock,
    replace: vi.fn(),
  }),
}))

vi.mock("@/components/jav-library/DetailPage.vue", () => ({
  default: {
    name: "DetailPage",
    template: "<div data-detail-page-stub />",
  },
}))

vi.mock("@/components/jav-library/NotFoundState.vue", () => ({
  default: {
    name: "NotFoundState",
    template: "<div data-not-found-stub />",
  },
}))

vi.mock("@/services/adapters/web/web-library-service", () => ({
  loadMovieDetail: vi.fn(),
}))

vi.mock("@/services/library-service", () => {
  const movie = makeMovie()
  return {
    useLibraryService: () => ({
      movies: computed(() => [movie]),
      getMovieById: (id: string) => (id === movie.id ? movie : undefined),
      getRelatedMovies: () => [],
      toggleFavorite: vi.fn(),
      patchMovie: vi.fn(),
      deleteMovie: vi.fn(),
      restoreMovie: vi.fn(),
      deleteMoviePermanently: vi.fn(),
      refreshMovieMetadata: vi.fn(),
      revealMovieInFileManager: vi.fn(),
    }),
  }
})

vi.mock("@/composables/use-scan-task-tracker", () => ({
  useScanTaskTracker: () => ({
    activeTask: ref(null),
    start: vi.fn(),
  }),
}))

describe("DetailView", () => {
  beforeEach(() => {
    routerPushMock.mockReset()
  })

  it("returns to the browse route when Escape is pressed", async () => {
    mount(DetailView)

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }))

    expect(routerPushMock).toHaveBeenCalledWith({
      name: "favorites",
      query: {
        q: "star",
        tab: "top-rated",
        selected: "movie-1",
      },
    })
  })
})
