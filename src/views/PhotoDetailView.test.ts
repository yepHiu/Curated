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
  patchPhoto: vi.fn(),
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
    patchPhoto: serviceMocks.patchPhoto,
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

vi.mock("@/components/jav-library/books/BookCommentSection.vue", () => ({
  default: {
    name: "BookCommentSection",
    props: ["kind", "entityId"],
    template: '<section data-book-comment-section :data-kind="kind" :data-entity-id="entityId" />',
  },
}))

describe("PhotoDetailView", () => {
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
    serviceMocks.replacePhotoTags.mockReset()
    serviceMocks.patchPhoto.mockReset()
    serviceMocks.patchPhoto.mockImplementation(async (_id: string, patch: { rating?: number | null; title?: string }) => {
      // 把评分或标题补丁应用到当前详情夹具，供视图回写断言。
      serviceState.photo = makePhoto({
        ...serviceState.photo,
        rating: patch.rating !== undefined ? patch.rating : serviceState.photo?.rating ?? null,
        title: patch.title !== undefined ? patch.title : serviceState.photo?.title ?? "Summer Frame",
      })
      return serviceState.photo
    })
  })

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

  it("saves a local photo rating through the photo service", async () => {
    // 详情评分卡经独立 patchPhoto 写入本地分。
    const wrapper = mount(PhotoDetailView)
    await flushPromises()
    wrapper.findComponent({ name: "PhotoDetailPanel" }).vm.$emit("updateRating", 3.5)
    await flushPromises()
    expect(serviceMocks.patchPhoto).toHaveBeenCalledWith("photo-1", { rating: 3.5 })
    expect(wrapper.findComponent({ name: "PhotoDetailPanel" }).props("photo").rating).toBe(3.5)
  })

  it("saves a photo display title through the photo service", async () => {
    serviceMocks.patchPhoto.mockResolvedValueOnce(makePhoto({ title: "展示写真" }))
    const wrapper = mount(PhotoDetailView)
    await flushPromises()
    const done = vi.fn()
    wrapper.findComponent({ name: "PhotoDetailPanel" }).vm.$emit("saveTitle", "展示写真", done)
    await flushPromises()
    expect(serviceMocks.patchPhoto).toHaveBeenCalledWith("photo-1", { title: "展示写真" })
    expect(wrapper.findComponent({ name: "PhotoDetailPanel" }).props("photo").title).toBe("展示写真")
    expect(done).toHaveBeenCalledWith()
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
    expect(wrapper.get("[data-book-comment-section]").attributes("data-kind")).toBe("photos")
    expect(wrapper.get("[data-book-comment-section]").attributes("data-entity-id")).toBe("photo-1")
    // 备注卡片必须排在页面预览之后。
    expect(
      wrapper.html().indexOf("data-photo-page-preview-grid"),
    ).toBeLessThan(wrapper.html().indexOf("data-book-comment-section"))
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
      query: { tag: "portrait" },
    })
  })
})
