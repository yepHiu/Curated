import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ComicBook } from "@/domain/comic/types"
import ComicCard from "./ComicCard.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { name: "Badge", template: "<span><slot /></span>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled"],
    template: "<button :disabled='disabled'><slot /></button>",
  },
}))

function makeComic(overrides: Partial<ComicBook> = {}): ComicBook {
  return {
    id: "comic-card-1",
    title: "Glass City Notebook",
    tags: ["作者:青井", "cyberpunk"],
    rating: 4.5,
    isFavorite: true,
    readStatus: "reading",
    pageCount: 24,
    currentPageIndex: 5,
    coverUrl: "https://example.com/cover.jpg",
    sourceFileName: "glass-city.cbz",
    location: "D:/Comics/glass-city.cbz",
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
    ...overrides,
  }
}

describe("ComicCard", () => {
  it("shows title, page count, rating, favorite, and read progress", () => {
    const wrapper = mount(ComicCard, {
      props: {
        comic: makeComic(),
      },
    })

    expect(wrapper.text()).toContain("Glass City Notebook")
    expect(wrapper.text()).toContain("24")
    expect(wrapper.text()).toContain("4.5")
    expect(wrapper.text()).toContain("6 / 24")
    expect(wrapper.get("[data-comic-favorite]").attributes("data-comic-favorite")).toBe("true")
  })
})
