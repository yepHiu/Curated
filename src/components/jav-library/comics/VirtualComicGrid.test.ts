import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ComicBook } from "@/domain/comic/types"
import VirtualComicGrid from "./VirtualComicGrid.vue"

vi.mock("@/components/jav-library/comics/ComicCard.vue", () => ({
  default: {
    props: ["comic"],
    template: "<article data-comic-card>{{ comic.id }}</article>",
  },
}))

function makeComic(id: string): ComicBook {
  return {
    id,
    title: `Comic ${id}`,
    tags: [],
    rating: null,
    isFavorite: false,
    readStatus: "unread",
    pageCount: 12,
    currentPageIndex: 0,
    sourceFileName: `${id}.cbz`,
    location: `D:/Comics/${id}.cbz`,
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
  }
}

describe("VirtualComicGrid", () => {
  it("uses the same grid track and gap variables as the movie poster grid", () => {
    const wrapper = mount(VirtualComicGrid, {
      props: {
        comics: [makeComic("c1")],
      },
    })

    const style = wrapper.get("[data-virtual-comic-grid]").attributes("style")

    expect(style).toContain(
      "grid-template-columns: repeat(auto-fill, minmax(min(100%, var(--movie-grid-min-track)), 1fr))",
    )
    expect(style).toContain("column-gap: var(--movie-grid-gap)")
    expect(style).toContain("row-gap: var(--movie-grid-gap)")
  })
})
