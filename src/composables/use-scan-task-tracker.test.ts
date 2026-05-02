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
      expect.objectContaining({ variant: "success" }),
    )
    expect(mocks.reloadMoviesFromApi).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })
})
