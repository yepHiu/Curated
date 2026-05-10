import { flushPromises, mount } from "@vue/test-utils"
import { computed, defineComponent } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"

const mocks = vi.hoisted(() => ({
  pushAppToast: vi.fn(),
  checkLibraryPathStorageStatus: vi.fn(),
  statuses: [] as Array<Record<string, unknown>>,
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    libraryPathStorageStatuses: computed(() => mocks.statuses),
    checkLibraryPathStorageStatus: mocks.checkLibraryPathStorageStatus,
  }),
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: mocks.pushAppToast,
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

async function mountStorageStatusHarness() {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", "true")
  const { useLibraryStorageStatusAlerts } = await import(
    "@/composables/use-library-storage-status-alerts"
  )
  const Harness = defineComponent({
    setup() {
      useLibraryStorageStatusAlerts()
      return () => null
    },
  })
  return mount(Harness)
}

afterEach(() => {
  vi.clearAllMocks()
  vi.unstubAllEnvs()
  sessionStorage.clear()
  mocks.statuses = []
})

describe("useLibraryStorageStatusAlerts", () => {
  it("pushes a storage notification for one offline library path", async () => {
    mocks.statuses = [
      {
        libraryPathId: "library-b",
        title: "Cold storage",
        path: "F:/Offline/Collections",
        status: "offline",
        message: "The storage device may be offline or disconnected.",
        canRescan: false,
        canImport: false,
      },
    ]
    mocks.checkLibraryPathStorageStatus.mockResolvedValueOnce(undefined)

    const wrapper = await mountStorageStatusHarness()
    await flushPromises()

    expect(mocks.checkLibraryPathStorageStatus).toHaveBeenCalledTimes(1)
    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      expect.stringContaining("toasts.storagePathOffline"),
      expect.objectContaining({
        variant: "warning",
        notification: expect.objectContaining({
          type: "storage",
          title: "notificationCenter.titles.storageOffline",
          source: { route: "/settings?section=library", libraryPathId: "library-b" },
        }),
      }),
    )
    wrapper.unmount()
  })

  it("aggregates multiple abnormal paths into one toast", async () => {
    mocks.statuses = [
      {
        libraryPathId: "library-a",
        title: "Archive A",
        path: "D:/Media/A",
        status: "offline",
        message: "offline",
      },
      {
        libraryPathId: "library-b",
        title: "Archive B",
        path: "E:/Media/B",
        status: "volume_mismatch",
        message: "wrong disk",
      },
    ]
    mocks.checkLibraryPathStorageStatus.mockResolvedValueOnce(undefined)

    const wrapper = await mountStorageStatusHarness()
    await flushPromises()

    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      expect.stringContaining("toasts.storagePathsAbnormal"),
      expect.objectContaining({
        notification: expect.objectContaining({ type: "storage" }),
      }),
    )
    wrapper.unmount()
  })

  it("does not toast when all library paths are online", async () => {
    mocks.statuses = [
      {
        libraryPathId: "library-a",
        title: "Archive A",
        path: "D:/Media/A",
        status: "online",
        message: "online",
      },
    ]
    mocks.checkLibraryPathStorageStatus.mockResolvedValueOnce(undefined)

    const wrapper = await mountStorageStatusHarness()
    await flushPromises()

    expect(mocks.pushAppToast).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
