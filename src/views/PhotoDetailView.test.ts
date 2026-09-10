import { flushPromises, mount } from "@vue/test-utils"
import { computed } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import PhotoDetailView from "./PhotoDetailView.vue"

function makePhoto(overrides: Partial<PhotoBook> = {}): PhotoBook {
  return {
    id: "photo-1",
    title: "Summer Frame",
    tags: ["portrait"],
    rating: 4.5,
    isFavorite: false,
    pageCount: 12,
    currentPageIndex: 3,
    sourceFileName: "summer-frame.cbz",
    location: "D:/Photos/summer-frame.cbz",
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
    pages: [
      {
        photoId: "photo-1",
        index: 0,
        entryPath: "001.jpg",
        fileName: "001.jpg",
        imageUrl: "https://example.com/page-1.jpg",
        thumbUrl: "https://example.com/thumb-1.jpg",
      },
    ],
    ...overrides,
  }
}

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  route: {
    fullPath: "/photos/photo-1",
    name: "photo-detail",
    params: { id: "photo-1" } as Record<string, string>,
    query: {},
  },
}))

const serviceState = vi.hoisted(() => ({
  photo: undefined as PhotoBook | undefined,
}))

const serviceMocks = vi.hoisted(() => ({
  refreshSettings: vi.fn(),
  reloadPhotosFromApi: vi.fn(),
  getPhotoById: vi.fn(),
  loadPhotoDetail: vi.fn(),
  replacePhotoTags: vi.fn(),
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
  }),
}))

vi.mock("@/services/photo-library-service", () => ({
  usePhotoLibraryService: () => ({
    photosLoaded: computed(() => true),
    refreshSettings: serviceMocks.refreshSettings,
    reloadPhotosFromApi: serviceMocks.reloadPhotosFromApi,
    getPhotoById: serviceMocks.getPhotoById,
    loadPhotoDetail: serviceMocks.loadPhotoDetail,
    replacePhotoTags: serviceMocks.replacePhotoTags,
  }),
}))

vi.mock("@/components/jav-library/photos/PhotoDetailPanel.vue", () => ({
  default: {
    name: "PhotoDetailPanel",
    props: ["photo"],
    emits: ["startBrowsing", "browseByTag"],
    template: `
      <section data-photo-detail-panel>
        <h2>{{ photo.title }}</h2>
        <button data-start-browsing @click="$emit('startBrowsing', photo.currentPageIndex)" />
        <button data-browse-tag @click="$emit('browseByTag', { tag: photo.tags[0] })" />
      </section>
    `,
  },
}))

vi.mock("@/components/jav-library/photos/PhotoPagePreviewGrid.vue", () => ({
  default: {
    name: "PhotoPagePreviewGrid",
    props: ["photo"],
    emits: ["openViewer"],
    template: `
      <section data-photo-page-preview-grid>
        <button data-open-preview @click="$emit('openViewer', 0)" />
      </section>
    `,
  },
}))

describe("PhotoDetailView", () => {
  it("saves tags through the photo service and updates detail state", async () => {
    serviceMocks.replacePhotoTags.mockResolvedValueOnce(makePhoto({ tags: ['portrait', 'landscape'] }))
    const wrapper = mount(PhotoDetailView)
    await flushPromises()
    const done = vi.fn()
    wrapper.findComponent({ name: 'PhotoDetailPanel' }).vm.$emit('addTag', 'landscape', done)
    await flushPromises()
    expect(serviceMocks.replacePhotoTags).toHaveBeenCalledWith('photo-1', ['portrait', 'landscape'])
    expect(wrapper.findComponent({ name: 'PhotoDetailPanel' }).props('photo').tags).toEqual(['portrait', 'landscape'])
    expect(done).toHaveBeenCalledWith()
  })
  beforeEach(() => {
    routerMocks.push.mockReset()
    routerMocks.route.fullPath = "/photos/photo-1"
    routerMocks.route.params = { id: "photo-1" }
    serviceState.photo = makePhoto()
    serviceMocks.refreshSettings.mockReset()
    serviceMocks.refreshSettings.mockResolvedValue(undefined)
    serviceMocks.reloadPhotosFromApi.mockReset()
    serviceMocks.reloadPhotosFromApi.mockResolvedValue(undefined)
    serviceMocks.getPhotoById.mockReset()
    serviceMocks.getPhotoById.mockImplementation((id?: string) =>
      id === serviceState.photo?.id ? serviceState.photo : undefined,
    )
    serviceMocks.loadPhotoDetail.mockReset()
    serviceMocks.loadPhotoDetail.mockImplementation(async (id: string) =>
      id === serviceState.photo?.id ? serviceState.photo : undefined,
    )
  })

  it("loads a photo book into the detail panel and preview grid", async () => {
    const wrapper = mount(PhotoDetailView)
    await flushPromises()

    expect(serviceMocks.refreshSettings).toHaveBeenCalled()
    expect(serviceMocks.loadPhotoDetail).toHaveBeenCalledWith("photo-1")
    expect(serviceMocks.reloadPhotosFromApi).not.toHaveBeenCalled()
    expect(serviceMocks.getPhotoById).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain("Summer Frame")
    expect(wrapper.find("[data-photo-page-preview-grid]").exists()).toBe(true)
  })

  it("opens the viewer from the browse button and page preview with return intent", async () => {
    const wrapper = mount(PhotoDetailView)
    await flushPromises()

    await wrapper.get("[data-start-browsing]").trigger("click")
    await wrapper.get("[data-open-preview]").trigger("click")

    expect(routerMocks.push).toHaveBeenNthCalledWith(1, {
      name: "photo-viewer",
      params: { id: "photo-1", pageIndex: "3" },
      query: { returnTo: "/photos/photo-1" },
    })
    expect(routerMocks.push).toHaveBeenNthCalledWith(2, {
      name: "photo-viewer",
      params: { id: "photo-1", pageIndex: "0" },
      query: { returnTo: "/photos/photo-1" },
    })
  })

  it("opens the photo wall filtered by tag", async () => {
    const wrapper = mount(PhotoDetailView)
    await flushPromises()

    await wrapper.get("[data-browse-tag]").trigger("click")

    expect(routerMocks.push).toHaveBeenCalledWith({
      name: "photos",
      query: { q: "portrait" },
    })
  })
})
