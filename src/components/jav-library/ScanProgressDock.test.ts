import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { TaskDTO } from "@/api/types"

const trackerState = vi.hoisted(() => ({
  activeTask: { value: null as TaskDTO | null },
  pollError: { value: null as string | null },
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
  it("renders scan progress labels from locale keys", async () => {
    trackerState.activeTask.value = scanTask("running", {
      scanProcessed: 7,
      scanTotal: 10,
      scanImported: 2,
      scanUpdated: 3,
      scanSkipped: 1,
    })
    trackerState.pollError.value = null

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
      trackerState.activeTask.value = scanTask(status)
      trackerState.pollError.value = null

      const wrapper = await mountDock()

      expect(wrapper.text()).toContain(titleKey)
    },
  )

  it("renders poll error title from locale keys", async () => {
    trackerState.activeTask.value = null
    trackerState.pollError.value = "scanTask.fetchFailed"

    const wrapper = await mountDock()

    expect(wrapper.text()).toContain("scan.statusLabel")
  })
})
