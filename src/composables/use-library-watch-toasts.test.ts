import { flushPromises, mount } from "@vue/test-utils"
import { defineComponent } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"
import type { TaskDTO } from "@/api/types"

const mocks = vi.hoisted(() => ({
  getRecentTasks: vi.fn(),
  pushAppToast: vi.fn(),
  reloadMoviesFromApi: vi.fn(),
  bumpMovieImageVersion: vi.fn(),
}))

vi.mock("@/api/endpoints", () => ({
  api: {
    getRecentTasks: mocks.getRecentTasks,
  },
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: mocks.pushAppToast,
  taskTerminalToastVariant: (status: TaskDTO["status"]) => {
    if (status === "completed") return "success"
    if (status === "partial_failed") return "warning"
    if (status === "failed") return "destructive"
    return "default"
  },
}))

vi.mock("@/lib/image-version", () => ({
  bumpMovieImageVersion: mocks.bumpMovieImageVersion,
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    reloadMoviesFromApi: mocks.reloadMoviesFromApi,
  }),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      key === "toasts.libraryWatchScanDone" ? `${key}: ${params?.message ?? ""}` : key,
  }),
}))

function makeFsnotifyScanTask(): TaskDTO {
  return {
    taskId: "scan-1",
    type: "scan.library",
    status: "completed",
    createdAt: "2026-05-02T00:00:00.000Z",
    startedAt: "2026-05-02T00:00:01.000Z",
    finishedAt: "2026-05-02T00:00:02.000Z",
    progress: 100,
    message: "Scan finished: 24 discovered, 0 imported, 0 updated, 24 skipped",
    metadata: {
      trigger: "fsnotify",
      scanTotal: 24,
      scanProcessed: 24,
      scanImported: 0,
      scanUpdated: 0,
      scanSkipped: 24,
    },
  }
}

async function mountLibraryWatchHarness() {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", "true")
  const { useLibraryWatchToasts } = await import("@/composables/use-library-watch-toasts")
  const Harness = defineComponent({
    setup() {
      useLibraryWatchToasts()
      return () => null
    },
  })
  return mount(Harness)
}

afterEach(() => {
  vi.clearAllTimers()
  vi.useRealTimers()
  vi.clearAllMocks()
  vi.unstubAllEnvs()
  sessionStorage.clear()
})

describe("useLibraryWatchToasts", () => {
  it("summarizes fsnotify scan completion without exposing the raw backend message", async () => {
    vi.useFakeTimers()
    mocks.getRecentTasks.mockResolvedValueOnce({ tasks: [makeFsnotifyScanTask()] })

    const wrapper = await mountLibraryWatchHarness()
    await flushPromises()

    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      "toasts.libraryWatchScanDoneNoChanges",
      expect.objectContaining({ variant: "success" }),
    )
    expect(mocks.pushAppToast).not.toHaveBeenCalledWith(
      expect.stringContaining("Scan finished:"),
      expect.anything(),
    )
    expect(mocks.reloadMoviesFromApi).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })
})
