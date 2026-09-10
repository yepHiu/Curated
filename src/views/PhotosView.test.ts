import { flushPromises, mount } from "@vue/test-utils"
import { computed } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import PhotosView from "./PhotosView.vue"

function makePhoto(overrides: Partial<PhotoBook> = {}): PhotoBook {
  const id = overrides.id ?? "photo-1"
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
    ...overrides,
  }
}

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  route: {
    fullPath: "/photos",
    name: "photos",
    query: {},
  },
}))

const serviceState = vi.hoisted(() => ({
  photos: [] as PhotoBook[],
  loadError: null as string | null,
}))

const serviceMocks = vi.hoisted(() => ({
  refreshSettings: vi.fn(),
  reloadPhotosFromApi: vi.fn(),
}))

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

vi.mock("@/services/photo-library-service", () => ({
  usePhotoLibraryService: () => ({
    photos: computed(() => serviceState.photos),
    loadError: computed(() => serviceState.loadError),
    refreshSettings: serviceMocks.refreshSettings,
    reloadPhotosFromApi: serviceMocks.reloadPhotosFromApi,
  }),
}))

vi.mock("@/components/jav-library/photos/PhotoLibraryPage.vue", () => ({
  default: {
    name: "PhotoLibraryPage",
    props: ["photos", "activeSort"],
    emits: ["openDetails", "openViewer", "update:sort"],
    template: `
      <section data-photo-library-page :data-active-sort="activeSort">
        <article v-for="photo in photos" :key="photo.id" :data-photo-id="photo.id">{{ photo.title }}</article>
        <button data-sort-filename @click="$emit('update:sort', 'fileName')" />
        <button v-for="photo in photos" :key="'open-' + photo.id" :data-open-photo="photo.id" @click="$emit('openDetails', photo.id)" />
        <button v-for="photo in photos" :key="'view-' + photo.id" :data-view-photo="photo.id" @click="$emit('openViewer', photo.id, photo.currentPageIndex)" />
      </section>
    `,
  },
}))

describe("PhotosView", () => {
  beforeEach(() => {
    routerMocks.push.mockReset()
    routerMocks.replace.mockReset()
    routerMocks.route.fullPath = "/photos"
    routerMocks.route.query = {}
    serviceState.photos = [
      makePhoto({ id: "photo-1", title: "Summer Frame", sourceFileName: "b.cbz" }),
      makePhoto({ id: "photo-2", title: "Night Portrait", sourceFileName: "a.cbz" }),
    ]
    serviceState.loadError = null
    serviceMocks.refreshSettings.mockReset()
    serviceMocks.refreshSettings.mockResolvedValue(undefined)
    serviceMocks.reloadPhotosFromApi.mockReset()
    serviceMocks.reloadPhotosFromApi.mockResolvedValue(undefined)
  })

  it("loads photo settings and mock photo books into the photo wall", async () => {
    const wrapper = mount(PhotosView)
    await flushPromises()

    expect(serviceMocks.refreshSettings).toHaveBeenCalled()
    expect(serviceMocks.reloadPhotosFromApi).toHaveBeenCalled()
    expect(wrapper.text()).toContain("Summer Frame")
    expect(wrapper.text()).toContain("Night Portrait")
  })

  it("filters by shell query and sorts by filename query", async () => {
    routerMocks.route.query = { q: "night", sort: "fileName" }
    const wrapper = mount(PhotosView)
    await flushPromises()

    expect(wrapper.text()).not.toContain("Summer Frame")
    expect(wrapper.text()).toContain("Night Portrait")
    expect(wrapper.get("[data-photo-library-page]").attributes("data-active-sort")).toBe("fileName")
  })

  it("updates the sort query and opens photo detail/viewer routes", async () => {
    const wrapper = mount(PhotosView)
    await flushPromises()

    await wrapper.get("[data-sort-filename]").trigger("click")
    await wrapper.get('[data-open-photo="photo-1"]').trigger("click")
    await wrapper.get('[data-view-photo="photo-1"]').trigger("click")

    expect(routerMocks.replace).toHaveBeenCalledWith({
      name: "photos",
      query: {
        sort: "fileName",
      },
    })
    expect(routerMocks.push).toHaveBeenNthCalledWith(1, {
      name: "photo-detail",
      params: { id: "photo-1" },
    })
    expect(routerMocks.push).toHaveBeenNthCalledWith(2, {
      name: "photo-viewer",
      params: { id: "photo-1", pageIndex: "0" },
      query: { returnTo: "/photos" },
    })
  })
})
