import { mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import HomeView from "./HomeView.vue"
import type { Movie } from "@/domain/movie/types"

function makeMovie(id: string, overrides: Partial<Movie> = {}): Movie {
  return {
    id,
    title: `Movie ${id}`,
    code: `CODE-${id}`,
    studio: "Studio A",
    actors: ["Actor A"],
    tags: ["tag-a"],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4.0,
    metadataRating: 4.0,
    userRating: undefined,
    summary: `Summary ${id}`,
    isFavorite: false,
    addedAt: "2026-04-01T00:00:00.000Z",
    location: `D:/Library/${id}.mp4`,
    resolution: "1080p",
    year: 2026,
    releaseDate: "2026-04-01",
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
    ...overrides,
  }
}

const mockMovies = vi.hoisted(() => [
  makeMovie("m1", { isFavorite: true, rating: 4.9, userRating: 5, userTags: ["User Tag A"] }),
  makeMovie("m2", { rating: 4.8, actors: ["Actor B"], tags: ["tag-b"], studio: "Studio B" }),
  makeMovie("m3", { rating: 4.6 }),
  makeMovie("m4", { rating: 4.5 }),
  makeMovie("m5", { rating: 4.4 }),
  makeMovie("m6", { rating: 4.3 }),
  makeMovie("m7", { rating: 4.2 }),
  makeMovie("m8", { rating: 4.1 }),
  makeMovie("m9", { rating: 4.0 }),
  makeMovie("m10", { rating: 3.9 }),
  makeMovie("m11", { rating: 3.8 }),
])

const routerPushMock = vi.hoisted(() => vi.fn())
const homepageSnapshotState = vi.hoisted(() => ({
  value: null as null | {
    dateUtc: string
    generatedAt: string
    generationVersion?: string
    heroMovieIds: string[]
    recommendationMovieIds: string[]
  },
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("en"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRouter: () => ({
    push: routerPushMock,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: computed(() => mockMovies),
    trashedMovies: computed(() => []),
  }),
}))

vi.mock("@/lib/playback-progress-storage", () => ({
  listSortedByUpdatedDesc: vi.fn(() => [
    {
      movieId: "m1",
      positionSec: 180,
      durationSec: 1200,
      updatedAt: "2026-04-12T08:00:00.000Z",
    },
    {
      movieId: "m2",
      positionSec: 300,
      durationSec: 1200,
      updatedAt: "2026-04-12T10:00:00.000Z",
    },
  ]),
  getProgress: vi.fn(),
  playbackProgressRevision: ref(0),
}))

vi.mock("@/composables/use-homepage-daily-recommendations", () => ({
  useHomepageDailyRecommendations: () => ({
    snapshot: homepageSnapshotState,
    loading: ref(false),
    error: ref<unknown>(null),
    refresh: vi.fn(),
  }),
}))

vi.mock("@/components/jav-library/MovieCard.vue", () => ({
  default: {
    name: "MovieCard",
    props: ["movie"],
    template: "<article class='movie-card-stub'>{{ movie.title }}</article>",
  },
}))

vi.mock("@/components/jav-library/MediaStill.vue", () => ({
  default: {
    name: "MediaStill",
    template: "<div class='media-still-stub' />",
  },
}))

vi.mock("@/components/jav-library/PlaybackHistoryCard.vue", () => ({
  default: {
    name: "PlaybackHistoryCard",
    props: ["movie", "entry"],
    template: "<article class='playback-history-card-stub'>{{ movie.title }} {{ entry.movieId }}</article>",
  },
}))

describe("HomeView", () => {
  beforeEach(() => {
    routerPushMock.mockReset()
    homepageSnapshotState.value = null
  })

  it("renders the homepage hero and section rows", () => {
    const wrapper = mount(HomeView)
    const heroFrame = wrapper.get("[data-home-hero-frame]")
    const heroProgressRail = wrapper.get("[data-home-hero-progress-rail]")

    expect(wrapper.find("[data-home-hero]").exists()).toBe(true)
    expect(wrapper.get("[data-home-hero-shell]").classes()).toContain("px-0")
    expect(wrapper.get("[data-home-scroll-region]").classes()).toContain("overflow-y-auto")
    expect(heroFrame.classes()).not.toContain("rounded-[2rem]")
    expect(heroFrame.classes()).not.toContain("bg-card/35")
    expect(heroProgressRail.classes()).toContain("bg-background/72")
    expect(heroProgressRail.classes()).toContain("mt-3")
    expect(heroProgressRail.classes()).toContain("mx-auto")
    expect(heroProgressRail.classes()).toContain("max-w-[54rem]")
    expect(heroFrame.find("[data-home-hero-progress-rail]").exists()).toBe(false)
    expect(wrapper.get("[data-home-hero-stage]").classes()).toContain("h-[clamp(22rem,44vw,40rem)]")
    expect(wrapper.get("[data-home-hero-stage]").classes()).toContain("sm:h-[clamp(25rem,46vw,44rem)]")
    expect(wrapper.get('[data-hero-progress-item-active="true"]').classes()).toContain("bg-primary")
    expect(wrapper.findAll("[data-hero-progress-item]")).toHaveLength(8)
    expect(wrapper.text()).toContain("home.sectionRecentTitle")
    expect(wrapper.text()).toContain("home.sectionRecommendTitle")
    expect(wrapper.text()).toContain("home.sectionContinueTitle")
    expect(wrapper.findAll(".playback-history-card-stub")).toHaveLength(2)
  })

  it("prefers backend daily snapshot ids for hero and recommendations", () => {
    homepageSnapshotState.value = {
      dateUtc: "2026-04-15",
      generatedAt: "2026-04-15T00:00:01Z",
      generationVersion: "v1",
      heroMovieIds: ["m9", "m8", "m7", "m6", "m5", "m4", "m3", "m2"],
      recommendationMovieIds: ["m11", "m10", "m1"],
    }

    const wrapper = mount(HomeView)
    const portal = wrapper.getComponent({ name: "HomepagePortal" })
    const model = portal.props("model") as {
      heroMovies: Movie[]
      recommendations: { movie: Movie }[]
    }

    expect(model.heroMovies.map((movie) => movie.id)).toEqual([
      "m9",
      "m8",
      "m7",
      "m6",
      "m5",
      "m4",
      "m3",
      "m2",
    ])
    expect(model.recommendations.map((entry) => entry.movie.id)).toEqual(["m11", "m10", "m1"])
  })

  it("opens library filters from taste radar chips", async () => {
    const wrapper = mount(HomeView)

    await wrapper.get('[data-home-taste-chip-kind="actor"]').trigger("click")
    expect(routerPushMock).toHaveBeenLastCalledWith({
      name: "library",
      query: {
        actor: "Actor A",
      },
    })

    await wrapper.get('[data-home-taste-chip-kind="tag"]').trigger("click")
    expect(routerPushMock).toHaveBeenLastCalledWith({
      name: "tags",
      query: {
        tag: "tag-a",
      },
    })

    await wrapper.get('[data-home-taste-chip-kind="studio"]').trigger("click")
    expect(routerPushMock).toHaveBeenLastCalledWith({
      name: "library",
      query: {
        studio: "Studio A",
      },
    })
  })
})
