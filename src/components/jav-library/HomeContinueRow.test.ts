import { mount } from "@vue/test-utils"
import { ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"
import HomeContinueRow from "./HomeContinueRow.vue"

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

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("@/components/jav-library/MovieCard.vue", () => ({
  default: {
    name: "MovieCard",
    props: ["movie"],
    template: "<article class='movie-card-stub'>{{ movie.title }}</article>",
  },
}))

describe("HomeContinueRow", () => {
  it("renders continue watching as poster movie cards", () => {
    const wrapper = mount(HomeContinueRow, {
      props: {
        entries: [
          {
            movie: makeMovie("m1"),
            progressPercent: 25,
            remainingMinutes: 88,
            updatedAt: "2026-04-12T10:00:00.000Z",
          },
        ],
      },
    })

    expect(wrapper.findAll(".movie-card-stub")).toHaveLength(1)
    expect(wrapper.findAll(".playback-history-card-stub")).toHaveLength(0)
    expect(wrapper.text()).toContain("home.sectionContinueTitle")
    expect(wrapper.text()).not.toContain("home.continueRemaining")
    expect(wrapper.text()).not.toContain("home.heroDetailsAction")
  })
})
