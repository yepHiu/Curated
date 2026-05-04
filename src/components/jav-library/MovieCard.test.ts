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
})
