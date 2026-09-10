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

vi.mock("@/components/ui/toggle", () => ({
  Toggle: {
    name: "Toggle",
    props: ["pressed"],
    emits: ["update:pressed"],
    template:
      "<button :aria-pressed='pressed' @click=\"$emit('update:pressed', !pressed)\"><slot /></button>",
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
  it("shows title, page count, rating, and read progress without source or favorite controls", () => {
    const wrapper = mount(ComicCard, {
      props: {
        comic: makeComic(),
      },
    })

    expect(wrapper.text()).toContain("Glass City Notebook")
    expect(wrapper.text()).toContain("24")
    expect(wrapper.text()).toContain("4.5")
    expect(wrapper.text()).toContain("6 / 24")
    expect(wrapper.text()).not.toContain("glass-city.cbz")
    expect(wrapper.find("[data-comic-favorite]").exists()).toBe(false)
  })

  it("uses the same compact poster-first layout tokens as movie cards", () => {
    const wrapper = mount(ComicCard, {
      props: {
        comic: makeComic(),
      },
    })

    expect(wrapper.get("[data-comic-card]").classes()).toEqual(
      expect.arrayContaining(["rounded-[1.2rem]", "bg-card/80", "shadow-md"]),
    )
    expect(wrapper.get("[data-comic-poster]").classes()).toEqual(
      expect.arrayContaining(["aspect-[358/537]"]),
    )
    expect(wrapper.get("[data-comic-card-body]").classes()).toEqual(
      expect.arrayContaining([
        "min-h-[var(--movie-card-body-min-height)]",
        "gap-[var(--movie-card-body-gap)]",
        "p-[var(--movie-card-padding)]",
      ]),
    )
    expect(wrapper.text()).not.toContain("comics.startReading")
  })

  it("toggles batch selection instead of opening details in batch mode", async () => {
    const wrapper = mount(ComicCard, {
      props: {
        comic: makeComic(),
        batchMode: true,
        batchChecked: true,
      },
    })

    expect(wrapper.get("[data-comic-batch-checkbox]").attributes("checked")).toBeDefined()

    await wrapper.get("[data-comic-card-open]").trigger("click")

    expect(wrapper.emitted("openDetails")).toBeUndefined()
    expect(wrapper.emitted("toggleBatchSelect")).toEqual([["comic-card-1"]])
  })
})
