import { flushPromises, mount } from "@vue/test-utils"
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
    props: ["comics", "batchMode", "batchSelectedIds"],
    emits: ["toggleBatchSelect"],
    template:
      "<div data-virtual-comic-grid :data-batch-mode=\"batchMode ? 'true' : 'false'\" :data-selected=\"(batchSelectedIds || []).join(',')\"><button v-for='comic in comics' :key='comic.id' data-comic-select @click=\"$emit('toggleBatchSelect', comic.id)\">{{ comic.title }}</button></div>",
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
        activeSort: "addedAt",
      },
    })

    expect(wrapper.text()).toContain("nav.comics")
    expect(wrapper.text()).not.toContain("comics.subtitle")
    expect(wrapper.find("h1").exists()).toBe(true)
  })

  it("renders sample comics in the comic wall", () => {
    const wrapper = mount(ComicLibraryPage, {
      props: {
        comics: [
          makeComic("comic-1", "Glass City Notebook"),
          makeComic("comic-2", "Rain Garden"),
        ],
        activeSort: "addedAt",
      },
    })

    expect(wrapper.text()).toContain("Glass City Notebook")
    expect(wrapper.text()).toContain("Rain Garden")
  })

  it("keeps content toolbar aligned with movie tabs and leaves search to the shell", () => {
    const wrapper = mount(ComicLibraryPage, {
      props: {
        comics: [makeComic("comic-1", "Glass City Notebook")],
        activeSort: "addedAt",
        searchQuery: "rain",
      },
    })

    expect(wrapper.get("[data-comic-library-toolbar]").classes()).toEqual(
      expect.arrayContaining(["flex-wrap", "items-center", "justify-between", "pb-1"]),
    )
    expect(wrapper.find("[data-book-sort-trigger]").exists()).toBe(true)
    expect(wrapper.find("[data-comic-library-toolbar] input").exists()).toBe(false)
    expect(wrapper.text()).not.toContain("comics.searchPlaceholder")
    expect(wrapper.text()).toContain("comics.sortByAdded")
    expect(wrapper.text()).not.toContain("comics.filterUnread")
    expect(wrapper.text()).not.toContain("comics.filterReading")
    expect(wrapper.text()).not.toContain("comics.filterRead")
  })

  it("uses the same scroll gutter as the movie grid area", () => {
    const wrapper = mount(ComicLibraryPage, {
      props: {
        comics: [makeComic("comic-1", "Glass City Notebook")],
        activeSort: "addedAt",
      },
    })

    expect(wrapper.get("[data-comic-grid-scroll]").classes()).toEqual(
      expect.arrayContaining(["min-h-0", "flex-1", "overflow-y-auto", "pr-2"]),
    )
  })

  it("emits the selected comic sort when a sort tab is clicked", async () => {
    const wrapper = mount(ComicLibraryPage, {
      props: {
        comics: [makeComic("comic-1", "Glass City Notebook")],
        activeSort: "addedAt",
      },
    })

    wrapper.findComponent({ name: "BookLibraryToolbar" }).vm.$emit("sort", "fileName")
    await flushPromises()

    expect(wrapper.emitted("update:sort")?.[0]).toEqual(["fileName"])
  })

  it("renders movie-style batch management controls for the comic wall", async () => {
    const wrapper = mount(ComicLibraryPage, {
      props: {
        comics: [makeComic("comic-1", "Glass City Notebook")],
        activeSort: "addedAt",
      },
    })

    expect(wrapper.text()).toContain("comics.batchManage")

    await wrapper.get("[data-comic-enter-batch]").trigger("click")

    expect(wrapper.emitted("enterBatchMode")).toHaveLength(1)

    await wrapper.setProps({
      batchMode: true,
      batchSelectedIds: ["comic-1"],
    })

    expect(wrapper.text()).toContain("comics.batchSelectVisible")
    expect(wrapper.text()).toContain("comics.batchExitToolbar")
    expect(wrapper.get("[data-virtual-comic-grid]").attributes("data-batch-mode")).toBe("true")
    expect(wrapper.get("[data-virtual-comic-grid]").attributes("data-selected")).toBe("comic-1")

    await wrapper.get("[data-comic-select-visible]").trigger("click")
    await wrapper.get("[data-comic-exit-batch]").trigger("click")
    await wrapper.get("[data-comic-select]").trigger("click")

    expect(wrapper.emitted("selectAllVisibleInBatch")).toHaveLength(1)
    expect(wrapper.emitted("exitBatchMode")).toHaveLength(1)
    expect(wrapper.emitted("toggleBatchSelect")).toEqual([["comic-1"]])
  })
})
