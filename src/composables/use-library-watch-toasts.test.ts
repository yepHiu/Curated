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

function makeFsnotifyScanTask(overrides: Partial<TaskDTO> = {}): TaskDTO {
  const metadata = {
    trigger: "fsnotify",
    paths: ["E:/JAV/Curated"],
    scanTotal: 24,
    scanProcessed: 24,
    scanImported: 0,
    scanUpdated: 0,
    scanSkipped: 24,
    ...(overrides.metadata ?? {}),
  }
  return {
    taskId: "scan-1",
    type: "scan.library",
    status: "completed",
    createdAt: "2026-05-02T00:00:00.000Z",
    startedAt: "2026-05-02T00:00:01.000Z",
    finishedAt: "2026-05-02T00:00:02.000Z",
    progress: 100,
    message: "Scan finished: 24 discovered, 0 imported, 0 updated, 24 skipped",
    ...overrides,
    metadata,
  }
}

function makeLinkedScrapeTask(): TaskDTO {
  return {
    taskId: "scrape-1",
    type: "scrape.movie",
    status: "completed",
    createdAt: "2026-05-02T00:00:03.000Z",
    startedAt: "2026-05-02T00:00:04.000Z",
    finishedAt: "2026-05-02T00:00:05.000Z",
    progress: 100,
    message: "Metadata saved for ABC-100",
    metadata: {
      movieId: "movie-1",
      parentScanTaskId: "scan-1",
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

  it("suppresses a redundant no-change scan after a changed scan for the same paths", async () => {
    vi.useFakeTimers()
    const changed = makeFsnotifyScanTask({
      taskId: "scan-changed",
      finishedAt: "2026-05-02T00:00:02.000Z",
      message: "Scan finished: 2 discovered, 2 imported, 0 updated, 0 skipped",
      metadata: {
        scanTotal: 2,
        scanProcessed: 2,
        scanImported: 2,
        scanUpdated: 0,
        scanSkipped: 0,
      },
    })
    const noChange = makeFsnotifyScanTask({
      taskId: "scan-no-change",
      finishedAt: "2026-05-02T00:00:07.000Z",
      message: "Scan finished: 2 discovered, 0 imported, 0 updated, 2 skipped",
      metadata: {
        scanTotal: 2,
        scanProcessed: 2,
        scanImported: 0,
        scanUpdated: 0,
        scanSkipped: 2,
      },
    })
    mocks.getRecentTasks.mockResolvedValueOnce({ tasks: [noChange, changed] })

    const wrapper = await mountLibraryWatchHarness()
    await flushPromises()

    expect(mocks.pushAppToast).toHaveBeenCalledTimes(1)
    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      "toasts.libraryWatchScanDoneWithChanges",
      expect.objectContaining({ variant: "success" }),
    )
    expect(mocks.pushAppToast).not.toHaveBeenCalledWith(
      "toasts.libraryWatchScanDoneNoChanges",
      expect.anything(),
    )

    wrapper.unmount()
  })

  it("reloads the library when a linked scrape completion appears without its parent scan", async () => {
    vi.useFakeTimers()
    mocks.getRecentTasks.mockResolvedValueOnce({ tasks: [makeLinkedScrapeTask()] })

    const wrapper = await mountLibraryWatchHarness()
    await flushPromises()

    expect(mocks.reloadMoviesFromApi).toHaveBeenCalledTimes(1)
    expect(mocks.bumpMovieImageVersion).toHaveBeenCalledWith("movie-1")
    expect(mocks.pushAppToast).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})
