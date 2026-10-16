import { afterEach, describe, expect, it, vi } from "vitest"
import { flushPromises } from "@vue/test-utils"
import { startTaskCenterTracking, stopTaskCenterTracking, useTaskCenter } from "./use-task-center"
import { emptyTaskSummary } from "@/lib/task-center"
const api = vi.hoisted(() => ({ summary: vi.fn(), refresh: vi.fn(async () => {}) }))
vi.mock("@/services/task-center-service", () => ({ useTaskCenterService: () => api }))
vi.mock("./use-tag-organization", () => ({ useTagOrganization: () => api, stopTagOrganizationTracking: vi.fn() }))
vi.mock("./use-subtitle-jobs", () => ({ useSubtitleJobs: () => api, stopSubtitleTracking: vi.fn() }))
afterEach(() => { stopTaskCenterTracking(); vi.useRealTimers(); vi.clearAllMocks() })
describe("task center observation", () => {
  it("retains the last snapshot on disconnect and rejects replies from a previous lifecycle", async () => {
    let resolve!: (value: ReturnType<typeof emptyTaskSummary>) => void
    api.summary.mockImplementationOnce(() => new Promise(r => { resolve = r }))
    const stale = useTaskCenter().refresh()
    stopTaskCenterTracking()
    api.summary.mockResolvedValueOnce({ ...emptyTaskSummary(), total: 2 })
    await useTaskCenter().refresh()
    resolve({ ...emptyTaskSummary(), total: 99 }); await stale
    expect(useTaskCenter().summary.value.total).toBe(2)
    api.summary.mockRejectedValueOnce(new Error("offline")); await useTaskCenter().refresh()
    expect(useTaskCenter().connected.value).toBe(false)
    expect(useTaskCenter().summary.value.total).toBe(2)
  })
  it("coalesces wakeups and stops scheduling on unmount", async () => {
    vi.useFakeTimers()
    api.summary.mockResolvedValue({ ...emptyTaskSummary(), running: 1 })
    startTaskCenterTracking(); await flushPromises()
    window.dispatchEvent(new Event("curated:task-center-updated"))
    window.dispatchEvent(new Event("curated:task-center-updated"))
    await flushPromises()
    const calls = api.summary.mock.calls.length
    await vi.advanceTimersByTimeAsync(2000)
    expect(api.summary).toHaveBeenCalledTimes(calls + 1)
    stopTaskCenterTracking(); await vi.advanceTimersByTimeAsync(30000)
    expect(api.summary).toHaveBeenCalledTimes(calls + 1)
  })
})
