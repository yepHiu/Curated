import { afterEach, describe, expect, it, vi } from "vitest"

const comicServiceState = vi.hoisted(() => ({
  enabled: true,
  refreshSettings: vi.fn(),
}))

const photoServiceState = vi.hoisted(() => ({
  enabled: false,
  refreshSettings: vi.fn(),
}))

afterEach(() => {
  vi.resetModules()
  vi.clearAllMocks()
  comicServiceState.enabled = true
  photoServiceState.enabled = false
})

function mockRouteDependencies() {
  vi.doMock("@/services/auth-lock-service", () => ({
    authLockService: { refreshStatus: vi.fn() },
    isAuthLockEnabled: () => false,
  }))
  vi.doMock("@/services/comic-library-service", () => ({
    useComicLibraryService: () => ({
      comicLibraryEnabled: {
        get value() {
          return comicServiceState.enabled
        },
      },
      refreshSettings: comicServiceState.refreshSettings,
    }),
  }))
  vi.doMock("@/services/photo-library-service", () => ({
    usePhotoLibraryService: () => ({
      photoLibraryEnabled: {
        get value() {
          return photoServiceState.enabled
        },
      },
      refreshSettings: photoServiceState.refreshSettings,
    }),
  }))
  vi.doMock("@/layouts/AppShell.vue", () => ({
    default: { name: "MockAppShell", template: "<div />" },
  }))
  vi.doMock("@/views/PhotosView.vue", () => ({
    default: { name: "MockPhotosView", template: "<div />" },
  }))
  vi.doMock("@/views/PhotoDetailView.vue", () => ({
    default: { name: "MockPhotoDetailView", template: "<div />" },
  }))
  vi.doMock("@/views/PhotoViewerView.vue", () => ({
    default: { name: "MockPhotoViewerView", template: "<div />" },
  }))
  vi.doMock("@/views/SettingsView.vue", () => ({
    default: { name: "MockSettingsView", template: "<div />" },
  }))
}

describe("photo library routes", () => {
  it("redirects the photo wall to photo settings when photos are disabled", async () => {
    photoServiceState.enabled = false
    mockRouteDependencies()

    const { default: router } = await import("@/router")
    await router.push("/photos")
    await router.isReady()

    expect(photoServiceState.refreshSettings).toHaveBeenCalled()
    expect(router.currentRoute.value.name).toBe("settings")
    expect(router.currentRoute.value.query.section).toBe("experimental")
  })

  it("allows the photo wall route when photos are enabled", async () => {
    photoServiceState.enabled = true
    mockRouteDependencies()

    const { default: router } = await import("@/router")
    await router.push("/photos")
    await router.isReady()

    expect(photoServiceState.refreshSettings).toHaveBeenCalled()
    expect(router.currentRoute.value.name).toBe("photos")
  })

  it("resolves photo detail routes when photos are enabled", async () => {
    photoServiceState.enabled = true
    mockRouteDependencies()

    const { default: router } = await import("@/router")
    await router.push("/photos/photo-1")
    await router.isReady()

    expect(router.currentRoute.value.name).toBe("photo-detail")
    expect(router.currentRoute.value.params.id).toBe("photo-1")
  })

  it("resolves photo viewer routes when photos are enabled", async () => {
    photoServiceState.enabled = true
    mockRouteDependencies()

    const { default: router } = await import("@/router")
    await router.push("/photos/photo-1/view/3")
    await router.isReady()

    expect(router.currentRoute.value.name).toBe("photo-viewer")
    expect(router.currentRoute.value.params.id).toBe("photo-1")
    expect(router.currentRoute.value.params.pageIndex).toBe("3")
  })
})
