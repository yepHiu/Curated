import { flushPromises, mount } from "@vue/test-utils"
import { defineComponent } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"
import type { TaskDTO } from "@/api/types"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"

const mocks = vi.hoisted(() => ({
  getTaskStatus: vi.fn(),
  pushAppToast: vi.fn(),
  reloadMoviesFromApi: vi.fn(),
}))

vi.mock("@/api/endpoints", () => ({
  api: {
    getTaskStatus: mocks.getTaskStatus,
  },
}))

vi.mock("@/composables/use-app-toast", async () => {
  const actual = await vi.importActual<typeof import("@/composables/use-app-toast")>(
    "@/composables/use-app-toast",
  )
  return {
    ...actual,
    pushAppToast: mocks.pushAppToast,
  }
})

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      t: (key: string) => key,
    },
  },
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    reloadMoviesFromApi: mocks.reloadMoviesFromApi,
  }),
}))

function makeTask(status: TaskDTO["status"]): TaskDTO {
  return {
    taskId: "task-1",
    type: "scan.library",
    status,
    createdAt: "2026-04-30T00:00:00.000Z",
    progress: status === "completed" ? 1 : 0.2,
    message: "Scanning",
  }
}

function makeImportTask(status: TaskDTO["status"]): TaskDTO {
  return {
    ...makeTask(status),
    type: "import.movies",
    progress: 100,
    message: "Movie import completed",
    metadata: {
      completedFiles: 2,
      failedFiles: 0,
    },
  }
}

function makeMovieScrapeTask(status: TaskDTO["status"]): TaskDTO {
  return {
    ...makeTask(status),
    taskId: "scrape-1",
    type: "scrape.movie",
    progress: status === "completed" ? 100 : 1,
    message: "Metadata updated",
    metadata: {
      movieId: "movie-1",
    },
  }
}

afterEach(() => {
  vi.clearAllTimers()
  vi.useRealTimers()
  vi.clearAllMocks()
})

describe("useScanTaskTracker", () => {
  it("stops polling when the owning component unmounts", async () => {
    vi.useFakeTimers()
    mocks.getTaskStatus.mockResolvedValue(makeTask("running"))

    const Harness = defineComponent({
      setup() {
        const tracker = useScanTaskTracker()
        tracker.start("task-1")
        return () => null
      },
    })

    const wrapper = mount(Harness)
    await Promise.resolve()
    expect(mocks.getTaskStatus).toHaveBeenCalledTimes(1)

    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(500)

    expect(mocks.getTaskStatus).toHaveBeenCalledTimes(1)
  })

  it("uses a localized fallback when polling fails without an Error object", async () => {
    mocks.getTaskStatus.mockRejectedValueOnce("network down")

    const Harness = defineComponent({
      setup() {
        const { pollError, start } = useScanTaskTracker()
        start("task-1")
        return { pollError }
      },
      template: "<p>{{ pollError }}</p>",
    })

    const wrapper = mount(Harness)
    await flushPromises()

    expect(wrapper.text()).toBe("scanTask.fetchFailed")
  })

  it("toasts and reloads the library when movie import completes", async () => {
    vi.useFakeTimers()
    mocks.getTaskStatus.mockResolvedValueOnce(makeImportTask("completed"))

    const Harness = defineComponent({
      setup() {
        const tracker = useScanTaskTracker()
        tracker.start("import-1")
        return () => null
      },
    })

    const wrapper = mount(Harness)
    await flushPromises()

    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      "toasts.movieImportDone",
      expect.objectContaining({
        variant: "success",
        notification: expect.objectContaining({
          type: "system",
          title: "notificationCenter.titles.importDone",
          source: { taskId: "task-1", route: "/settings?section=library" },
        }),
      }),
    )
    expect(mocks.reloadMoviesFromApi).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it("persists failed movie import task notifications", async () => {
    vi.useFakeTimers()
    mocks.getTaskStatus.mockResolvedValueOnce({
      ...makeImportTask("failed"),
      errorMessage: "Disk is full",
      metadata: {
        completedFiles: 1,
        failedFiles: 2,
      },
    })

    const Harness = defineComponent({
      setup() {
        const tracker = useScanTaskTracker()
        tracker.start("import-2")
        return () => null
      },
    })

    const wrapper = mount(Harness)
    await flushPromises()

    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      "toasts.movieImportFailed",
      expect.objectContaining({
        variant: "destructive",
        notification: expect.objectContaining({
          type: "system",
          title: "notificationCenter.titles.importFailed",
          source: { taskId: "task-1", route: "/settings?section=library" },
        }),
      }),
    )

    wrapper.unmount()
  })

  it("shows a persisted start notification when a manual scan opts in", async () => {
    vi.useFakeTimers()
    mocks.getTaskStatus.mockResolvedValueOnce(makeTask("running"))

    const Harness = defineComponent({
      setup() {
        const tracker = useScanTaskTracker()
        tracker.start("task-1", { notifyScanStart: true, hideProgressDock: true })
        return () => null
      },
    })

    const wrapper = mount(Harness)
    await flushPromises()

    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      "toasts.manualLibraryScanStarted",
      expect.objectContaining({
        variant: "default",
        notification: expect.objectContaining({
          type: "scan",
          title: "notificationCenter.titles.scanStarted",
          source: { taskId: "task-1", route: "/settings?section=library" },
        }),
      }),
    )

    wrapper.unmount()
  })

  it("keeps hidden progress tasks out of the progress dock state", async () => {
    vi.useFakeTimers()
    mocks.getTaskStatus.mockResolvedValueOnce(makeTask("running"))

    const Harness = defineComponent({
      setup() {
        const { activeTask, progressTask, start } = useScanTaskTracker()
        start("task-1", { hideProgressDock: true })
        return { activeTask, progressTask }
      },
      template: `
        <div>
          <span data-active>{{ activeTask?.taskId ?? "" }}</span>
          <span data-progress>{{ progressTask?.taskId ?? "" }}</span>
        </div>
      `,
    })

    const wrapper = mount(Harness)
    await flushPromises()

    expect(wrapper.get("[data-active]").text()).toBe("task-1")
    expect(wrapper.get("[data-progress]").text()).toBe("")

    wrapper.unmount()
  })

  it("routes manual scan terminal notifications back to settings", async () => {
    vi.useFakeTimers()
    mocks.getTaskStatus.mockResolvedValueOnce(makeTask("completed"))

    const Harness = defineComponent({
      setup() {
        const tracker = useScanTaskTracker()
        tracker.start("task-1")
        return () => null
      },
    })

    const wrapper = mount(Harness)
    await flushPromises()

    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      "toasts.manualLibraryScanDone",
      expect.objectContaining({
        variant: "success",
        notification: expect.objectContaining({
          type: "scan",
          title: "notificationCenter.titles.scanDone",
          source: { taskId: "task-1", route: "/settings?section=library" },
        }),
      }),
    )

    wrapper.unmount()
  })

  it("persists single movie scrape task notifications when requested", async () => {
    vi.useFakeTimers()
    mocks.getTaskStatus.mockResolvedValueOnce(makeMovieScrapeTask("completed"))

    const Harness = defineComponent({
      setup() {
        const tracker = useScanTaskTracker()
        tracker.start("scrape-1", { notifyMovieScrape: true })
        return () => null
      },
    })

    const wrapper = mount(Harness)
    await flushPromises()

    expect(mocks.pushAppToast).toHaveBeenCalledWith(
      "toasts.manualMovieScrapeDone",
      expect.objectContaining({
        variant: "success",
        notification: expect.objectContaining({
          type: "scrape",
          title: "notificationCenter.titles.scrapeDone",
          source: {
            taskId: "scrape-1",
            movieId: "movie-1",
            route: "/detail/movie-1",
          },
        }),
      }),
    )

    wrapper.unmount()
  })

  it("keeps movie scrape task notifications opt-in for batch refreshes", async () => {
    vi.useFakeTimers()
    mocks.getTaskStatus.mockResolvedValueOnce(makeMovieScrapeTask("completed"))

    const Harness = defineComponent({
      setup() {
        const tracker = useScanTaskTracker()
        tracker.start("scrape-1")
        return () => null
      },
    })

    const wrapper = mount(Harness)
    await flushPromises()

    expect(mocks.pushAppToast).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})
