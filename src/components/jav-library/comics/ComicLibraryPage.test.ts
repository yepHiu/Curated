import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ComicBook } from "@/domain/comic/types"
import ComicLibraryPage from "./ComicLibraryPage.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("./VirtualComicGrid.vue", () => ({
  default: {
    name: "VirtualComicGrid",
    props: ["comics"],
    template: "<div data-virtual-comic-grid><article v-for='comic in comics' :key='comic.id'>{{ comic.title }}</article></div>",
  },
}))

function makeComic(id: string, title: string): ComicBook {
  return {
    id,
    title,
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

describe("ComicLibraryPage", () => {
  it("omits the page title and description from the comic wall header", () => {
    const wrapper = mount(ComicLibraryPage, {
      props: {
        comics: [makeComic("comic-1", "Glass City Notebook")],
        activeFilter: "all",
      },
    })

    expect(wrapper.text()).not.toContain("comics.title")
    expect(wrapper.text()).not.toContain("comics.subtitle")
    expect(wrapper.find("h1").exists()).toBe(false)
  })

  it("renders sample comics in the comic wall", () => {
    const wrapper = mount(ComicLibraryPage, {
      props: {
        comics: [
          makeComic("comic-1", "Glass City Notebook"),
          makeComic("comic-2", "Rain Garden"),
        ],
        activeFilter: "all",
      },
    })

    expect(wrapper.text()).toContain("Glass City Notebook")
    expect(wrapper.text()).toContain("Rain Garden")
  })
})
