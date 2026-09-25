import { flushPromises, mount } from "@vue/test-utils"
import { computed } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
import ComicsView from "./ComicsView.vue"

function makeComic(overrides: Partial<ComicBook> = {}): ComicBook {
  const id = overrides.id ?? "comic-1"
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
    ...overrides,
  }
}

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  route: {
    fullPath: "/comics",
    name: "comics",
    query: {},
  },
}))

const serviceState = vi.hoisted(() => ({
  comics: [] as ComicBook[],
  loadError: null as string | null,
}))

const serviceMocks = vi.hoisted(() => ({
  refreshSettings: vi.fn(),
  reloadComicsFromApi: vi.fn(),
  ensureComicsLoaded: vi.fn(),
  patchComic: vi.fn<(comicId: string, patch: ComicPatch) => Promise<ComicBook | undefined>>(),
  deleteComic: vi.fn<(comicId: string) => Promise<void>>(),
}))

const pushAppToastMock = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routerMocks.route,
  useRouter: () => ({
    push: routerMocks.push,
    replace: routerMocks.replace,
  }),
}))

vi.mock("@/services/comic-library-service", () => ({
  useComicLibraryService: () => ({
    comics: computed(() => serviceState.comics),
    comicsLoaded: computed(() => true),
    loadError: computed(() => serviceState.loadError),
    refreshSettings: serviceMocks.refreshSettings,
    reloadComicsFromApi: serviceMocks.reloadComicsFromApi,
    ensureComicsLoaded: serviceMocks.ensureComicsLoaded,
    patchComic: serviceMocks.patchComic,
    deleteComic: serviceMocks.deleteComic,
  }),
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: pushAppToastMock,
}))

vi.mock("@/components/jav-library/comics/ComicLibraryPage.vue", () => ({
  default: {
    name: "ComicLibraryPage",
    props: ["comics", "batchMode", "batchSelectedIds"],
    emits: [
      "enterBatchMode",
      "exitBatchMode",
      "selectAllVisibleInBatch",
      "toggleBatchSelect",
      "openDetails",
      "openReader",
      "toggleFavorite",
      "updateSearch",
      "update:sort",
    ],
    template: `
      <section
        data-comic-library-page
        :data-batch-mode="batchMode ? 'true' : 'false'"
        :data-selected="(batchSelectedIds || []).join(',')"
      >
        <button data-enter-batch @click="$emit('enterBatchMode')" />
        <button data-select-all @click="$emit('selectAllVisibleInBatch')" />
        <button
          v-for="comic in comics"
          :key="comic.id"
          :data-select-comic="comic.id"
          @click="$emit('toggleBatchSelect', comic.id)"
        />
        <button
          v-for="comic in comics"
          :key="'reader-' + comic.id"
          :data-open-reader="comic.id"
          @click="$emit('openReader', comic.id, comic.currentPageIndex)"
        />
      </section>
    `,
  },
}))

vi.mock("@/components/jav-library/MediaBatchActionBar.vue", () => ({
  default: {
    name: "MediaBatchActionBar",
    props: ["selectedCount", "operationBusy"],
    emits: [
      "exit",
      "clearSelection",
      "selectAllVisible",
      "addFavorite",
      "removeFavorite",
      "addTag",
      "deleteSelection",
    ],
    template: `
      <div
        data-comic-batch-action-bar
        :data-selected-count="selectedCount"
        :data-operation-busy="operationBusy ? 'true' : 'false'"
      >
        <button data-batch-add-favorite @click="$emit('addFavorite')" />
        <button data-batch-remove-favorite @click="$emit('removeFavorite')" />
        <button data-batch-add-tag @click="$emit('addTag', 'batch-tag')" />
        <button data-batch-delete @click="$emit('deleteSelection')" />
        <button data-batch-select-all @click="$emit('selectAllVisible')" />
        <button data-batch-clear @click="$emit('clearSelection')" />
        <button data-batch-exit @click="$emit('exit')" />
      </div>
    `,
  },
}))

describe("ComicsView batch management", () => {
  beforeEach(() => {
    routerMocks.push.mockReset()
    routerMocks.replace.mockReset()
    routerMocks.route.fullPath = "/comics"
    routerMocks.route.query = {}
    serviceState.comics = [
      makeComic({ id: "comic-1", tags: ["existing"] }),
      makeComic({ id: "comic-2", tags: ["batch-tag"] }),
    ]
    serviceState.loadError = null
    serviceMocks.refreshSettings.mockReset()
    serviceMocks.refreshSettings.mockResolvedValue(undefined)
    serviceMocks.reloadComicsFromApi.mockReset()
    serviceMocks.reloadComicsFromApi.mockResolvedValue(undefined)
    serviceMocks.ensureComicsLoaded.mockReset()
    serviceMocks.ensureComicsLoaded.mockResolvedValue(undefined)
    serviceMocks.patchComic.mockReset()
    serviceMocks.patchComic.mockImplementation(async (comicId, patch) => {
      const comic = serviceState.comics.find((item) => item.id === comicId)
      return comic ? { ...comic, ...patch, isFavorite: patch.favorite ?? comic.isFavorite } : undefined
    })
    serviceMocks.deleteComic.mockReset()
    serviceMocks.deleteComic.mockResolvedValue(undefined)
    pushAppToastMock.mockReset()
  })

  it("selects visible comics and batch-updates favorite state through the comic service", async () => {
    const wrapper = mount(ComicsView)
    await flushPromises()

    await wrapper.get("[data-enter-batch]").trigger("click")
    await wrapper.get("[data-select-all]").trigger("click")

    expect(wrapper.get("[data-comic-batch-action-bar]").attributes("data-selected-count")).toBe("2")

    await wrapper.get("[data-batch-add-favorite]").trigger("click")
    await flushPromises()

    expect(serviceMocks.patchComic).toHaveBeenNthCalledWith(1, "comic-1", { favorite: true })
    expect(serviceMocks.patchComic).toHaveBeenNthCalledWith(2, "comic-2", { favorite: true })
  })

  it("appends a batch tag only to selected comics that do not already have it", async () => {
    const wrapper = mount(ComicsView)
    await flushPromises()

    await wrapper.get("[data-enter-batch]").trigger("click")
    await wrapper.get("[data-select-all]").trigger("click")
    await wrapper.get("[data-batch-add-tag]").trigger("click")
    await flushPromises()

    expect(serviceMocks.patchComic).toHaveBeenCalledTimes(1)
    expect(serviceMocks.patchComic).toHaveBeenCalledWith("comic-1", {
      tags: ["existing", "batch-tag"],
    })
  })

  it("batch-deletes selected comics through the comic service and exits batch mode", async () => {
    const wrapper = mount(ComicsView)
    await flushPromises()

    await wrapper.get("[data-enter-batch]").trigger("click")
    await wrapper.get('[data-select-comic="comic-1"]').trigger("click")
    await wrapper.get('[data-select-comic="comic-2"]').trigger("click")
    await wrapper.get("[data-batch-delete]").trigger("click")
    await flushPromises()

    expect(serviceMocks.deleteComic).toHaveBeenNthCalledWith(1, "comic-1")
    expect(serviceMocks.deleteComic).toHaveBeenNthCalledWith(2, "comic-2")
    expect(wrapper.get("[data-comic-library-page]").attributes("data-batch-mode")).toBe("false")
  })

  it("opens the reader with the current comic wall route as the return target", async () => {
    routerMocks.route.fullPath = "/comics?q=alpha&sort=fileName"
    routerMocks.route.query = { q: "alpha", sort: "fileName" }
    serviceState.comics = [
      makeComic({ id: "comic-1", title: "Alpha Comic" }),
      makeComic({ id: "comic-2", title: "Beta Comic" }),
    ]
    const wrapper = mount(ComicsView)
    await flushPromises()

    await wrapper.get('[data-open-reader="comic-1"]').trigger("click")

    expect(routerMocks.push).toHaveBeenCalledWith({
      name: "comic-reader",
      params: { id: "comic-1", pageIndex: "0" },
      query: { returnTo: "/comics?q=alpha&sort=fileName" },
    })
  })

  it("filters by exact tag instead of treating the tag as a title substring", async () => {
    serviceState.comics = [
      makeComic({ id: "tagged", tags: ["作者:青井"] }),
      makeComic({ id: "title-only", title: "作者:青井笔记", tags: [] }),
    ]
    routerMocks.route.query = { tag: "作者:青井" }
    const wrapper = mount(ComicsView)
    await flushPromises()

    expect(wrapper.find('[data-select-comic="tagged"]').exists()).toBe(true)
    expect(wrapper.find('[data-select-comic="title-only"]').exists()).toBe(false)
  })

  it("uses the same overflow-hidden page content frame as the movie library", async () => {
    const wrapper = mount(ComicsView)
    await flushPromises()

    expect(serviceMocks.ensureComicsLoaded).toHaveBeenCalled()
    expect(serviceMocks.refreshSettings).not.toHaveBeenCalled()
    expect(serviceMocks.reloadComicsFromApi).not.toHaveBeenCalled()
    expect(wrapper.get("[data-comics-view-content]").classes()).toEqual(
      expect.arrayContaining([
        "flex",
        "min-h-0",
        "min-w-0",
        "flex-1",
        "flex-col",
        "overflow-hidden",
      ]),
    )
  })
})
