import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import VirtualPhotoGrid from "./VirtualPhotoGrid.vue"

vi.mock("@/components/jav-library/photos/PhotoCard.vue", () => ({
  default: {
    name: "PhotoCard",
    props: ["photo"],
    template: "<article data-photo-card>{{ photo.id }}</article>",
  },
}))

function makePhoto(id: string): PhotoBook {
  return {
    id,
    title: `Photo ${id}`,
    tags: [],
    rating: null,
    isFavorite: false,
    pageCount: 12,
    currentPageIndex: 0,
    sourceFileName: `${id}.cbz`,
    location: `D:/Photos/${id}.cbz`,
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
  }
}

describe("VirtualPhotoGrid", () => {
  it("uses the same grid track and gap variables as the movie poster grid", () => {
    const wrapper = mount(VirtualPhotoGrid, {
      props: {
        photos: [makePhoto("p1")],
      },
    })

    const style = wrapper.get("[data-virtual-photo-grid]").attributes("style")

    expect(style).toContain(
      "grid-template-columns: repeat(auto-fill, minmax(min(100%, var(--movie-grid-min-track)), 1fr))",
    )
    expect(style).toContain("column-gap: var(--movie-grid-gap)")
    expect(style).toContain("row-gap: var(--movie-grid-gap)")
    expect(style).toContain("padding-bottom: var(--movie-grid-gap)")
  })

  it("centers photo cards inside the same max-width frame as the movie grid", () => {
    const wrapper = mount(VirtualPhotoGrid, {
      props: {
        photos: [makePhoto("p1")],
      },
    })

    const cardShell = wrapper.get("[data-photo-card-shell]")
    const cardFrame = wrapper.get("[data-photo-card-frame]")

    expect(cardShell.classes()).toEqual(expect.arrayContaining(["flex", "min-w-0", "justify-center"]))
    expect(cardFrame.classes()).toEqual(expect.arrayContaining(["w-full", "min-w-0"]))
    expect(cardFrame.attributes("style")).toContain("max-width: min(100%, var(--movie-card-max-width))")
  })
})
