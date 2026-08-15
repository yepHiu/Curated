import { defineComponent } from "vue"
import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { TaskDTO } from "@/api/types"

function taskFixture(partial: Partial<TaskDTO> & Pick<TaskDTO, "taskId" | "type" | "status">): TaskDTO {
  return {
    createdAt: "2026-08-16T00:00:00Z",
    progress: 0,
    ...partial,
  }
}

const trackerState = vi.hoisted(() => ({
  activeTask: { value: null as TaskDTO | null },
  progressTask: { value: null as TaskDTO | null },
}))

vi.mock("@/composables/use-scan-task-tracker", () => ({
  useScanTaskTracker: () => trackerState,
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

async function mountNow() {
  const { useMessageCenterNow } = await import("./use-message-center-now")
  const Harness = defineComponent({
    setup() {
      const { nowItems } = useMessageCenterNow()
      return { nowItems }
    },
    template: `<div data-items>{{ nowItems.map((item) => item.id).join(",") }}</div>`,
  })
  return mount(Harness)
}

describe("useMessageCenterNow", () => {
  it("summarizes an in-progress import without using the scan dock copy", async () => {
    trackerState.activeTask.value = taskFixture({
      taskId: "import-1",
      type: "import.movies",
      status: "running",
      message: "Copying",
      metadata: { completedFiles: 2, totalFiles: 5 },
    })
    trackerState.progressTask.value = trackerState.activeTask.value

    const wrapper = await mountNow()
    expect(wrapper.get("[data-items]").text()).toBe("MSG-0002")
  })

  it("hides terminal tasks from the Now section", async () => {
    trackerState.activeTask.value = taskFixture({
      taskId: "scan-1",
      type: "scan.library",
      status: "completed",
      message: "done",
    })
    trackerState.progressTask.value = trackerState.activeTask.value

    const wrapper = await mountNow()
    expect(wrapper.get("[data-items]").text()).toBe("")
  })
})
