import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"
import PlayerPlaylistCard from "./PlayerPlaylistCard.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function movie(): Movie {
  return {
    id: "movie-1",
    title: "未命名的夏天",
    code: "SSIS-448",
    studio: "S1",
    actors: ["天使もえ"],
    tags: [],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4,
    summary: "",
    isFavorite: false,
    addedAt: "2026-08-01T00:00:00Z",
    location: "D:/a.mp4",
    resolution: "1080p",
    year: 2026,
    tone: "",
    coverClass: "",
    coverUrl: "https://example.test/cover.jpg",
  }
}

describe("PlayerPlaylistCard", () => {
  it("puts the poster on the left and keeps text left-aligned", () => {
    const wrapper = mount(PlayerPlaylistCard, {
      props: { movie: movie(), current: true },
    })

    const img = wrapper.get("img")
    const title = wrapper.get("p")
    expect(wrapper.text()).toContain("未命名的夏天")
    expect(wrapper.text()).toContain("SSIS-448")
    expect(wrapper.text()).toContain("天使もえ")
    expect(img.element.compareDocumentPosition(title.element) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(wrapper.get("[data-player-playlist-item]").attributes("aria-current")).toBe("true")
    expect(wrapper.get("[data-player-playlist-poster]").classes()).toEqual(
      expect.arrayContaining(["aspect-[2.25/1]", "w-[min(68%,19.5rem)]"]),
    )
    expect(wrapper.get("[data-player-playlist-item]").classes()).not.toContain("p-2.5")
  })
})
