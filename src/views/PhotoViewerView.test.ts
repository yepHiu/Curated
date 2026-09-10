import { flushPromises, mount } from "@vue/test-utils"
import { computed } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { PhotoBook, PhotoViewerSettings } from "@/domain/photo/types"
import PhotoViewerView from "./PhotoViewerView.vue"

function makePhoto(): PhotoBook {
  return {
    id: "photo-1",
    title: "Summer Frame",
    tags: [],
    rating: null,
    isFavorite: false,
    pageCount: 12,
    currentPageIndex: 0,
    sourceFileName: "summer-frame.cbz",
    location: "D:/Photos/summer-frame.cbz",
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
  }
}

const routeMock = vi.hoisted(() => ({
  params: { id: "photo-1", pageIndex: "4" } as Record<string, string>,
}))

const serviceState = vi.hoisted(() => ({
  photo: undefined as PhotoBook | undefined,
  viewerDefaults: {
    mode: "page",
    fit: "contain",
    direction: "ltr",
  } as PhotoViewerSettings,
}))

const serviceMocks = vi.hoisted(() => ({
  refreshSettings: vi.fn(),
  reloadPhotosFromApi: vi.fn(),
  getPhotoById: vi.fn(),
  loadPhotoDetail: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routeMock,
}))

vi.mock("@/services/photo-library-service", () => ({
  usePhotoLibraryService: () => ({
    photoViewer: computed(() => serviceState.viewerDefaults),
    refreshSettings: serviceMocks.refreshSettings,
    reloadPhotosFromApi: serviceMocks.reloadPhotosFromApi,
    getPhotoById: serviceMocks.getPhotoById,
    loadPhotoDetail: serviceMocks.loadPhotoDetail,
  }),
}))

vi.mock("@/components/jav-library/photos/PhotoViewer.vue", () => ({
  default: {
    name: "PhotoViewer",
    props: ["photo", "viewerDefaults", "initialPageIndex"],
    template:
      "<section data-photo-viewer :data-photo-id='photo.id' :data-initial-page='initialPageIndex' :data-viewer-mode='viewerDefaults.mode'>{{ photo.title }}</section>",
  },
}))

describe("PhotoViewerView", () => {
  beforeEach(() => {
    routeMock.params = { id: "photo-1", pageIndex: "4" }
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

  it("loads the routed photo book into the photo viewer", async () => {
    const wrapper = mount(PhotoViewerView)
    await flushPromises()

    expect(serviceMocks.refreshSettings).toHaveBeenCalled()
    expect(serviceMocks.loadPhotoDetail).toHaveBeenCalledWith("photo-1")
    expect(serviceMocks.reloadPhotosFromApi).not.toHaveBeenCalled()
    expect(serviceMocks.getPhotoById).not.toHaveBeenCalled()
    expect(wrapper.get("[data-photo-viewer]").attributes("data-photo-id")).toBe("photo-1")
    expect(wrapper.get("[data-photo-viewer]").attributes("data-initial-page")).toBe("4")
    expect(wrapper.get("[data-photo-viewer]").attributes("data-viewer-mode")).toBe("page")
  })
})
