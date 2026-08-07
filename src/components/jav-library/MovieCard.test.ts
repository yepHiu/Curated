import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

import type { Movie } from "@/domain/movie/types"

import MovieCard from "./MovieCard.vue"

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

describe("MovieCard", () => {
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

  it("keeps the batch selection visual compact inside its touch target", () => {
    const wrapper = mount(MovieCard, {
      props: {
        movie: makeMovie(),
        batchMode: true,
      },
    })

    expect(wrapper.get("[data-movie-batch-toggle]").classes()).toContain("size-11")
    expect(wrapper.get("[data-movie-batch-toggle-visual]").classes()).toContain("size-7")
  })
})
