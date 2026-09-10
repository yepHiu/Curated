import { flushPromises, mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import PhotoLibraryPage from "./PhotoLibraryPage.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("./VirtualPhotoGrid.vue", () => ({
  default: {
    name: "VirtualPhotoGrid",
    props: ["photos"],
    template: "<div data-virtual-photo-grid><article v-for='photo in photos' :key='photo.id'>{{ photo.title }}</article></div>",
  },
}))

function makePhoto(id: string, title: string): PhotoBook {
  return {
    id,
    title,
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

describe("PhotoLibraryPage", () => {
  it("keeps content toolbar aligned with movie tabs and leaves search to the shell", () => {
    const wrapper = mount(PhotoLibraryPage, {
      props: {
        photos: [makePhoto("photo-1", "Summer Frame")],
        activeSort: "addedAt",
        searchQuery: "summer",
      },
    })

    expect(wrapper.find("h1").exists()).toBe(true)
    expect(wrapper.get("[data-photo-library-toolbar]").classes()).toEqual(
      expect.arrayContaining(["flex-wrap", "items-center", "justify-between", "pb-1"]),
    )
    expect(wrapper.find("[data-photo-library-toolbar] input").exists()).toBe(false)
    expect(wrapper.text()).not.toContain("photos.searchPlaceholder")
    expect(wrapper.get("[data-book-sort-trigger]").text()).toBe("library.savedViewSort")
  })

  it("renders photo books in the photo wall", () => {
    const wrapper = mount(PhotoLibraryPage, {
      props: {
        photos: [
          makePhoto("photo-1", "Summer Frame"),
          makePhoto("photo-2", "Night Portrait"),
        ],
        activeSort: "addedAt",
      },
    })

    expect(wrapper.text()).toContain("Summer Frame")
    expect(wrapper.text()).toContain("Night Portrait")
  })

  it("emits the selected photo sort when a sort tab is clicked", async () => {
    const wrapper = mount(PhotoLibraryPage, {
      props: {
        photos: [makePhoto("photo-1", "Summer Frame")],
        activeSort: "addedAt",
      },
    })

    wrapper.findComponent({ name: "BookLibraryToolbar" }).vm.$emit("sort", "fileName")
    await flushPromises()

    expect(wrapper.emitted("update:sort")?.[0]).toEqual(["fileName"])
  })
})
