import { flushPromises, mount } from "@vue/test-utils"
import { computed } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
import ComicDetailView from "./ComicDetailView.vue"

function makeComic(overrides: Partial<ComicBook> = {}): ComicBook {
  return {
    id: "comic-1",
    title: "Comic 1",
    tags: ["author:alpha"],
    rating: 3,
    isFavorite: false,
    readStatus: "reading",
    pageCount: 12,
    currentPageIndex: 2,
    sourceFileName: "comic-1.cbz",
    location: "D:/Comics/comic-1.cbz",
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
    ...overrides,
  }
}

const routeState = vi.hoisted(() => ({
  fullPath: "/comics/comic-1",
  params: { id: "comic-1" as string | undefined },
}))

const routerPushMock = vi.hoisted(() => vi.fn())
const routerReplaceMock = vi.hoisted(() => vi.fn())
const serviceState = vi.hoisted(() => ({
  comic: undefined as ComicBook | undefined,
}))
const serviceMocks = vi.hoisted(() => ({
  loadComicDetail: vi.fn(),
  patchComic: vi.fn(),
  deleteComic: vi.fn(),
  revealComicSource: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    push: routerPushMock,
    replace: routerReplaceMock,
  }),
}))

vi.mock("@/services/comic-library-service", () => ({
  useComicLibraryService: () => ({
    comics: computed(() => (serviceState.comic ? [serviceState.comic] : [])),
    loadComicDetail: serviceMocks.loadComicDetail,
    patchComic: serviceMocks.patchComic,
    deleteComic: serviceMocks.deleteComic,
    revealComicSource: serviceMocks.revealComicSource,
  }),
}))

vi.mock("@/components/jav-library/comics/ComicDetailPanel.vue", () => ({
  default: {
    name: "ComicDetailPanel",
    props: ["comic", "busy"],
    emits: ["patch", "startReading", "deleteComic", "revealSource", "browseByTag"],
    methods: {
      markPatchDone(err: unknown) {
        globalThis.__comicPatchDone = err ?? null
      },
    },
    template: `
      <section data-comic-detail-panel-stub :data-comic-id="comic.id">
        <button
          data-comic-patch
          @click="$emit('patch', { title: 'Edited Comic' }, markPatchDone)"
        />
        <button
          data-comic-patch-rating
          @click="$emit('patch', { rating: 3.5 })"
        />
        <button data-comic-start-reading @click="$emit('startReading', comic.currentPageIndex)" />
        <button data-comic-browse-tag @click="$emit('browseByTag', { tag: comic.tags[0] })" />
        <button data-comic-delete @click="$emit('deleteComic', comic.id)" />
        <button data-comic-reveal @click="$emit('revealSource', comic.id)" />
      </section>
    `,
  },
}))

vi.mock("@/components/jav-library/comics/ComicPagePreviewGrid.vue", () => ({
  default: {
    name: "ComicPagePreviewGrid",
    props: ["comic"],
    emits: ["openReader"],
    template: "<section data-comic-preview-stub />",
  },
}))

vi.mock("@/components/jav-library/books/BookCommentSection.vue", () => ({
  default: {
    name: "BookCommentSection",
    props: ["kind", "entityId"],
    template: '<section data-book-comment-section :data-kind="kind" :data-entity-id="entityId" />',
  },
}))

declare global {
  var __comicPatchDone: unknown
  interface Window {
    __comicPatchDone?: unknown
  }
}

describe("ComicDetailView", () => {
  beforeEach(() => {
    routeState.fullPath = "/comics/comic-1"
    routeState.params = { id: "comic-1" }
    serviceState.comic = makeComic()
    serviceMocks.loadComicDetail.mockReset()
    serviceMocks.loadComicDetail.mockResolvedValue(serviceState.comic)
    serviceMocks.patchComic.mockReset()
    serviceMocks.patchComic.mockResolvedValue(makeComic({ title: "Edited Comic" }))
    serviceMocks.deleteComic.mockReset()
    serviceMocks.deleteComic.mockResolvedValue(undefined)
    serviceMocks.revealComicSource.mockReset()
    serviceMocks.revealComicSource.mockResolvedValue(undefined)
    routerPushMock.mockReset()
    routerReplaceMock.mockReset()
    delete globalThis.__comicPatchDone
  })

  it("loads and renders the comic detail", async () => {
    const wrapper = mount(ComicDetailView)
    await flushPromises()

    expect(serviceMocks.loadComicDetail).toHaveBeenCalledWith("comic-1")
    expect(wrapper.get("[data-comic-detail-panel-stub]").attributes("data-comic-id")).toBe(
      "comic-1",
    )
    expect(wrapper.get("[data-book-comment-section]").attributes("data-kind")).toBe("comics")
    expect(wrapper.get("[data-book-comment-section]").attributes("data-entity-id")).toBe("comic-1")
    // 备注卡片必须排在页面预览之后。
    expect(
      wrapper.html().indexOf("data-comic-preview-stub"),
    ).toBeLessThan(wrapper.html().indexOf("data-book-comment-section"))
  })

  it("patches comic metadata and reports callback success to the edit dialog", async () => {
    const wrapper = mount(ComicDetailView)
    await flushPromises()

    await wrapper.get("[data-comic-patch]").trigger("click")
    await flushPromises()

    expect(serviceMocks.patchComic).toHaveBeenCalledWith("comic-1", {
      title: "Edited Comic",
    } satisfies ComicPatch)
    expect(globalThis.__comicPatchDone).toBeNull()
  })

  it("saves a local comic rating through the comic service", async () => {
    // 详情评分卡经通用 patch 通道写入本地分。
    serviceMocks.patchComic.mockResolvedValueOnce(makeComic({ rating: 3.5 }))
    const wrapper = mount(ComicDetailView)
    await flushPromises()

    await wrapper.get("[data-comic-patch-rating]").trigger("click")
    await flushPromises()

    expect(serviceMocks.patchComic).toHaveBeenCalledWith("comic-1", {
      rating: 3.5,
    } satisfies ComicPatch)
  })

  it("deletes a comic and replaces the route with the comic library", async () => {
    const wrapper = mount(ComicDetailView)
    await flushPromises()

    await wrapper.get("[data-comic-delete]").trigger("click")
    await flushPromises()

    expect(serviceMocks.deleteComic).toHaveBeenCalledWith("comic-1")
    expect(routerReplaceMock).toHaveBeenCalledWith({ name: "comics" })
  })

  it("reveals the comic source through the comic service and surfaces failures", async () => {
    serviceMocks.revealComicSource.mockRejectedValueOnce(new Error("reveal failed"))
    const wrapper = mount(ComicDetailView)
    await flushPromises()

    await wrapper.get("[data-comic-reveal]").trigger("click")
    await flushPromises()

    expect(serviceMocks.revealComicSource).toHaveBeenCalledWith("comic-1")
    expect(wrapper.get('[role="alert"]').text()).toContain("reveal failed")
  })

  it("opens the comic wall filtered by the exact tag", async () => {
    const wrapper = mount(ComicDetailView)
    await flushPromises()

    await wrapper.get("[data-comic-browse-tag]").trigger("click")

    expect(routerPushMock).toHaveBeenCalledWith({
      name: "comics",
      query: { tag: "author:alpha" },
    })
  })

  it("opens the reader from the current page", async () => {
    const wrapper = mount(ComicDetailView)
    await flushPromises()

    await wrapper.get("[data-comic-start-reading]").trigger("click")

    expect(routerPushMock).toHaveBeenCalledWith({
      name: "comic-reader",
      params: { id: "comic-1", pageIndex: "2" },
      query: { returnTo: "/comics/comic-1" },
    })
  })
})
