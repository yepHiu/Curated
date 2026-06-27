import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type { TaskDTO } from "@/api/types"

const trackerState = vi.hoisted(() => ({
  activeTask: { value: null as TaskDTO | null },
  progressTask: { value: null as TaskDTO | null },
  pollError: { value: null as string | null },
  progressPollError: { value: null as string | null },
  dismiss: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/composables/use-scan-task-tracker", () => ({
  useScanTaskTracker: () => trackerState,
}))

function scanTask(status: TaskDTO["status"], metadata?: Record<string, unknown>): TaskDTO {
  return {
    taskId: "task-1",
    type: "scan.library",
    status,
    createdAt: "2026-04-30T00:00:00.000Z",
    progress: 42,
    message: "Scanning files",
    metadata,
  }
}

function importTask(status: TaskDTO["status"], metadata?: Record<string, unknown>): TaskDTO {
  return {
    ...scanTask(status, metadata),
    type: "import.movies",
    progress: 62,
    message: "Copying IMP-001.mp4",
  }
}

function comicImportTask(status: TaskDTO["status"], metadata?: Record<string, unknown>): TaskDTO {
  return {
    ...scanTask(status, metadata),
    type: "import.comics",
    progress: 62,
    message: "Copying Book One.cbz",
  }
}

function comicScanTask(status: TaskDTO["status"], metadata?: Record<string, unknown>): TaskDTO {
  return {
    ...scanTask(status, metadata),
    type: "scan.comics",
    progress: 42,
    message: "Scanning comics",
  }
}

async function mountDock() {
  const { default: ScanProgressDock } = await import("./ScanProgressDock.vue")
  return mount(ScanProgressDock, {
    global: {
      stubs: {
        Teleport: true,
      },
    },
  })
}

describe("ScanProgressDock", () => {
  beforeEach(() => {
    trackerState.activeTask.value = null
    trackerState.progressTask.value = null
    trackerState.pollError.value = null
    trackerState.progressPollError.value = null
    trackerState.dismiss.mockClear()
  })

  it("renders scan progress labels from locale keys", async () => {
    const task = scanTask("running", {
      scanProcessed: 7,
      scanTotal: 10,
      scanImported: 2,
      scanUpdated: 3,
      scanSkipped: 1,
    })
    trackerState.activeTask.value = task
    trackerState.progressTask.value = task

    const wrapper = await mountDock()

    expect(wrapper.text()).toContain("scan.scanning")
    expect(wrapper.text()).toContain("scan.processed")
    expect(wrapper.text()).toContain("scan.newItems")
    expect(wrapper.text()).toContain("scan.updated")
    expect(wrapper.text()).toContain("scan.skipped")
    expect(wrapper.get("button").attributes("aria-label")).toBe("scan.close")
  })

  it.each([
    ["completed", "scan.completed"],
    ["failed", "scan.finished"],
    ["partial_failed", "scan.finished"],
  ] satisfies Array<[TaskDTO["status"], string]>)(
    "renders %s title from locale keys",
    async (status, titleKey) => {
      const task = scanTask(status)
      trackerState.activeTask.value = task
      trackerState.progressTask.value = task

      const wrapper = await mountDock()

      expect(wrapper.text()).toContain(titleKey)
    },
  )

  it("renders poll error title from locale keys", async () => {
    trackerState.activeTask.value = null
    trackerState.pollError.value = "scanTask.fetchFailed"
    trackerState.progressPollError.value = "scanTask.fetchFailed"

    const wrapper = await mountDock()

    expect(wrapper.text()).toContain("scan.statusLabel")
  })

  it("renders import progress metadata instead of scan counters", async () => {
    const task = importTask("running", {
      currentFileName: "IMP-001.mp4",
      completedFiles: 1,
      totalFiles: 3,
      failedFiles: 0,
      copiedBytes: 1048576,
      totalBytes: 2097152,
    })
    trackerState.activeTask.value = task
    trackerState.progressTask.value = task

    const wrapper = await mountDock()

    expect(wrapper.text()).toContain("import.scanning")
    expect(wrapper.text()).toContain("import.currentFile")
    expect(wrapper.text()).toContain("IMP-001.mp4")
    expect(wrapper.text()).toContain("import.files")
    expect(wrapper.text()).toContain("import.copiedBytes")
    expect(wrapper.text()).not.toContain("scan.newItems")
  })

  it("renders comic import progress with comic-specific title", async () => {
    const task = comicImportTask("running", {
      currentFileName: "Book One.cbz",
      completedFiles: 1,
      totalFiles: 2,
      failedFiles: 0,
      copiedBytes: 1024,
      totalBytes: 2048,
    })
    trackerState.activeTask.value = task
    trackerState.progressTask.value = task

    const wrapper = await mountDock()

    expect(wrapper.text()).toContain("import.comicScanning")
    expect(wrapper.text()).toContain("Book One.cbz")
    expect(wrapper.text()).toContain("import.currentFile")
    expect(wrapper.text()).not.toContain("scan.newItems")
  })

  it("renders comic scan metadata instead of movie scan counters", async () => {
    const task = comicScanTask("running", {
      filesDiscovered: 9,
      imported: 2,
      updated: 3,
      skipped: 4,
    })
    trackerState.activeTask.value = task
    trackerState.progressTask.value = task

    const wrapper = await mountDock()

    expect(wrapper.text()).toContain("scan.comicScanning")
    expect(wrapper.text()).toContain("scan.discovered")
    expect(wrapper.text()).toContain("scan.newItems")
    expect(wrapper.text()).toContain("2")
    expect(wrapper.text()).toContain("3")
    expect(wrapper.text()).toContain("4")
  })

  it("does not render when the active task is hidden from progress UI", async () => {
    trackerState.activeTask.value = scanTask("running")
    trackerState.progressTask.value = null

    const wrapper = await mountDock()

    expect(wrapper.text()).toBe("")
  })
})
