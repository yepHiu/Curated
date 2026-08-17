import { mount, type VueWrapper } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"

import type { Movie } from "@/domain/movie/types"

import MovieCard, { MOVIE_CARD_OPEN_DETAILS_DELAY_MS } from "./MovieCard.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function makeMovie(overrides: Partial<Movie> = {}): Movie {
  return {
    id: "movie-card-1",
    code: "MOV-001",
    title: "Movie 1",
    studio: "Studio",
    actors: ["Actor"],
    tags: [],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4,
    summary: "",
    isFavorite: false,
    addedAt: "2026-05-03",
    location: "D:/Media/MOV-001.mp4",
    resolution: "1080p",
    year: 2026,
    tone: "",
    coverClass: "",
    thumbUrl: "/api/library/movies/movie-card-1/asset/thumb?v=test-movie-card",
    ...overrides,
  }
}

function dispatchCardClick(wrapper: VueWrapper, detail: number) {
  wrapper.get("[data-movie-card-id] button").element.dispatchEvent(
    new MouseEvent("click", { bubbles: true, cancelable: true, detail }),
  )
}

describe("MovieCard", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("prefers the card thumbnail over the detail cover", () => {
    const wrapper = mount(MovieCard, {
      props: {
        movie: makeMovie({
          coverUrl: "/api/library/movies/movie-card-1/asset/cover?v=cover",
          thumbUrl: "/api/library/movies/movie-card-1/asset/thumb?v=thumb",
        }),
      },
    })

    expect(wrapper.get("img").attributes("src")).toContain("/asset/thumb")
  })

  it("keeps the poster overlay visible when a loaded poster card is remounted", async () => {
    const movie = makeMovie()
    const first = mount(MovieCard, {
      props: {
        movie,
      },
    })

    await first.get("img").trigger("load")
    expect(first.find(".bg-gradient-to-t").exists()).toBe(true)
    first.unmount()

    const second = mount(MovieCard, {
      props: {
        movie,
      },
    })

    expect(second.find(".bg-gradient-to-t").exists()).toBe(true)
  })

  it("reserves visible space for the tag overflow badge on narrow cards", () => {
    const wrapper = mount(MovieCard, {
      props: {
        movie: makeMovie({
          userTags: ["favorite", "performer"],
          tags: ["subtitle", "collection"],
        }),
        showFavorite: true,
      },
    })

    expect(wrapper.get("[data-movie-tag-row]").classes()).toEqual(
      expect.arrayContaining(["min-w-0", "overflow-hidden"]),
    )
    for (const tag of wrapper.findAll("[data-movie-card-tag]")) {
      expect(tag.classes()).toEqual(expect.arrayContaining(["min-w-0", "shrink"]))
    }
    const overflow = wrapper.get("[data-movie-tag-overflow]")
    expect(overflow.text()).toBe("+1")
    expect(overflow.classes()).toContain("shrink-0")
    expect(wrapper.get("[data-movie-favorite-toggle]").classes()).toContain("size-11")
  })

  it("uses a theme border instead of a checkbox for batch selection", async () => {
    const wrapper = mount(MovieCard, {
      props: {
        movie: makeMovie(),
        batchMode: true,
        batchChecked: true,
      },
    })

    expect(wrapper.find("input[type=checkbox]").exists()).toBe(false)
    expect(wrapper.get("[data-movie-card-selected]").classes()).toEqual(
      expect.arrayContaining(["border-2", "border-primary"]),
    )
    const toggle = wrapper.get("[data-movie-batch-toggle]")
    expect(toggle.attributes("aria-pressed")).toBe("true")
    await toggle.trigger("click")
    expect(wrapper.emitted("toggleBatchSelect")).toEqual([
      [{ movieId: "movie-card-1", shiftKey: false }],
    ])
  })

  it("emits a shift-click batch selection payload", async () => {
    const wrapper = mount(MovieCard, {
      props: {
        movie: makeMovie(),
        batchMode: true,
      },
    })

    await wrapper.get("[data-movie-batch-toggle]").trigger("click", { shiftKey: true })
    expect(wrapper.emitted("toggleBatchSelect")).toEqual([
      [{ movieId: "movie-card-1", shiftKey: true }],
    ])
  })

  it("opens details after a delayed single click", async () => {
    vi.useFakeTimers()
    const wrapper = mount(MovieCard, {
      props: {
        movie: makeMovie(),
      },
    })

    dispatchCardClick(wrapper, 1)
    expect(wrapper.emitted("openDetails")).toBeUndefined()
    await vi.advanceTimersByTimeAsync(MOVIE_CARD_OPEN_DETAILS_DELAY_MS)
    expect(wrapper.emitted("openDetails")).toEqual([["movie-card-1"]])
    expect(wrapper.emitted("openPlayer")).toBeUndefined()
  })

  it("opens the player on double-click without opening details", async () => {
    vi.useFakeTimers()
    const wrapper = mount(MovieCard, {
      props: {
        movie: makeMovie(),
      },
    })

    dispatchCardClick(wrapper, 1)
    dispatchCardClick(wrapper, 2)
    await vi.runOnlyPendingTimersAsync()
    expect(wrapper.emitted("openPlayer")).toEqual([["movie-card-1"]])
    expect(wrapper.emitted("openDetails")).toBeUndefined()
  })

  it("opens details immediately from a keyboard activation", async () => {
    const wrapper = mount(MovieCard, {
      props: {
        movie: makeMovie(),
      },
    })

    dispatchCardClick(wrapper, 0)
    expect(wrapper.emitted("openDetails")).toEqual([["movie-card-1"]])
    expect(wrapper.emitted("openPlayer")).toBeUndefined()
  })
})
