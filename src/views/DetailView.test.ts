import { flushPromises, mount } from "@vue/test-utils"
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
const routerReplaceMock = vi.hoisted(() => vi.fn())
const serviceState = vi.hoisted(() => ({
  movie: undefined as Movie | undefined,
  movies: [] as Movie[],
}))
const serviceMocks = vi.hoisted(() => ({
  loadMovieDetail: vi.fn(),
  getRelatedMovies: vi.fn(() => []),
  toggleFavorite: vi.fn(),
  patchMovie: vi.fn(),
  deleteMovie: vi.fn(),
  restoreMovie: vi.fn(),
  deleteMoviePermanently: vi.fn(),
  refreshMovieMetadata: vi.fn(),
  revealMovieInFileManager: vi.fn(),
}))
const scanTrackerStartMock = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    push: routerPushMock,
    replace: routerReplaceMock,
  }),
}))

vi.mock("@/components/jav-library/DetailPage.vue", () => ({
  default: {
    name: "DetailPage",
    props: ["movie"],
    emits: [
      "toggleFavorite",
      "updateUserRating",
      "deleteMovie",
      "refreshMetadata",
    ],
    template: `
      <section
        data-detail-page-stub
        :data-movie-id="movie.id"
        :data-favorite="String(movie.isFavorite)"
      >
        <button
          data-toggle-favorite
          @click="$emit('toggleFavorite', { movieId: movie.id, nextValue: !movie.isFavorite })"
        />
        <button
          data-update-rating
          @click="$emit('updateUserRating', { movieId: movie.id, value: 3.5 })"
        />
        <button data-delete-movie @click="$emit('deleteMovie', movie.id)" />
        <button data-refresh-metadata @click="$emit('refreshMetadata', movie.id)" />
      </section>
    `,
  },
}))

vi.mock("@/components/jav-library/NotFoundState.vue", () => ({
  default: {
    name: "NotFoundState",
    props: ["title", "description"],
    template: "<div data-not-found-stub>{{ title }} {{ description }}</div>",
  },
}))

vi.mock("@/services/library-service", () => {
  return {
    useLibraryService: () => ({
      movies: computed(() => serviceState.movies),
      getMovieById: (id: string) => (id === serviceState.movie?.id ? serviceState.movie : undefined),
      loadMovieDetail: serviceMocks.loadMovieDetail,
      getRelatedMovies: serviceMocks.getRelatedMovies,
      toggleFavorite: serviceMocks.toggleFavorite,
      patchMovie: serviceMocks.patchMovie,
      deleteMovie: serviceMocks.deleteMovie,
      restoreMovie: serviceMocks.restoreMovie,
      deleteMoviePermanently: serviceMocks.deleteMoviePermanently,
      refreshMovieMetadata: serviceMocks.refreshMovieMetadata,
      revealMovieInFileManager: serviceMocks.revealMovieInFileManager,
    }),
  }
})

vi.mock("@/composables/use-scan-task-tracker", () => ({
  useScanTaskTracker: () => ({
    activeTask: ref(null),
    start: scanTrackerStartMock,
  }),
}))

describe("DetailView", () => {
  beforeEach(() => {
    routeState.name = "detail"
    routeState.params = { id: "movie-1" }
    routeState.query = {
      browse: "favorites",
      q: "star",
      tab: "top-rated",
      selected: "movie-1",
    }
    serviceState.movie = makeMovie()
    serviceState.movies = [serviceState.movie]
    serviceMocks.loadMovieDetail.mockReset()
    serviceMocks.loadMovieDetail.mockResolvedValue(serviceState.movie)
    serviceMocks.getRelatedMovies.mockReset()
    serviceMocks.getRelatedMovies.mockReturnValue([])
    serviceMocks.toggleFavorite.mockReset()
    serviceMocks.patchMovie.mockReset()
    serviceMocks.deleteMovie.mockReset()
    serviceMocks.restoreMovie.mockReset()
    serviceMocks.deleteMoviePermanently.mockReset()
    serviceMocks.refreshMovieMetadata.mockReset()
    serviceMocks.revealMovieInFileManager.mockReset()
    scanTrackerStartMock.mockReset()
    routerPushMock.mockReset()
    routerReplaceMock.mockReset()
  })

  it("loads the detail movie and renders the detail page", async () => {
    const wrapper = mount(DetailView)
    await flushPromises()

    expect(serviceMocks.loadMovieDetail).toHaveBeenCalledWith("movie-1")
    expect(wrapper.get("[data-detail-page-stub]").attributes("data-movie-id")).toBe("movie-1")
  })

  it("renders NotFoundState with a load error when the movie cannot be loaded", async () => {
    serviceState.movie = undefined
    serviceState.movies = []
    serviceMocks.loadMovieDetail.mockResolvedValueOnce(undefined)

    const wrapper = mount(DetailView)
    await flushPromises()

    expect(wrapper.find("[data-detail-page-stub]").exists()).toBe(false)
    expect(wrapper.find("[data-not-found-stub]").exists()).toBe(true)
    expect(wrapper.text()).toContain("detail.loadError")
  })

  it("forwards favorite and rating updates to the library service", async () => {
    const favoriteMovie = makeMovie({ isFavorite: true })
    const ratedMovie = makeMovie({ userRating: 3.5, rating: 3.5 })
    serviceMocks.toggleFavorite.mockResolvedValueOnce(favoriteMovie)
    serviceMocks.patchMovie.mockResolvedValueOnce(ratedMovie)
    const wrapper = mount(DetailView)
    await flushPromises()

    await wrapper.get("[data-toggle-favorite]").trigger("click")
    await flushPromises()
    expect(serviceMocks.toggleFavorite).toHaveBeenCalledWith("movie-1", true)
    expect(wrapper.get("[data-detail-page-stub]").attributes("data-favorite")).toBe("true")

    await wrapper.get("[data-update-rating]").trigger("click")
    await flushPromises()
    expect(serviceMocks.patchMovie).toHaveBeenCalledWith("movie-1", { rating: 3.5 })
  })

  it("moves a movie to trash and returns to the browse route", async () => {
    serviceMocks.deleteMovie.mockResolvedValueOnce(undefined)
    const wrapper = mount(DetailView)
    await flushPromises()

    await wrapper.get("[data-delete-movie]").trigger("click")
    await flushPromises()

    expect(serviceMocks.deleteMovie).toHaveBeenCalledWith("movie-1")
    expect(routerReplaceMock).toHaveBeenCalledWith({
      name: "favorites",
      query: {
        q: "star",
        tab: "top-rated",
      },
    })
  })

  it("starts scan tracking when metadata refresh returns a task", async () => {
    serviceMocks.refreshMovieMetadata.mockResolvedValueOnce({
      taskId: "task-1",
      type: "scrape.movie",
      status: "pending",
      createdAt: "2026-04-01T00:00:00.000Z",
      progress: 0,
    })
    const wrapper = mount(DetailView)
    await flushPromises()

    await wrapper.get("[data-refresh-metadata]").trigger("click")
    await flushPromises()

    expect(serviceMocks.refreshMovieMetadata).toHaveBeenCalledWith("movie-1")
    expect(scanTrackerStartMock).toHaveBeenCalledWith("task-1", { notifyMovieScrape: true })
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
