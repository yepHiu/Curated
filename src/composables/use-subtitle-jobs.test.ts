import { afterEach, describe, expect, it, vi } from "vitest"
import type { SubtitleJob } from "@/services/contracts/subtitle-service"
import { flushPromises } from "@vue/test-utils"
import { startSubtitleTracking, stopSubtitleTracking, useSubtitleJobs } from "./use-subtitle-jobs"
const api = vi.hoisted(() => ({ jobs: vi.fn(), loadMovieDetail: vi.fn() }))
vi.mock("@/services/subtitle-service", () => ({ useSubtitleService: () => api }))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => api }))
const job: SubtitleJob = { id: "job", movieId: "movie", fileId: "part", fileName: "movie.mp4", state: "running", stage: "translating", completed: 1, total: 3, assets: [], translationModel: "qwen-plus", createdAt: "", updatedAt: "" }

afterEach(() => { stopSubtitleTracking(); vi.useRealTimers(); vi.clearAllMocks() })
describe("global subtitle tracking", () => {
  it("restores server tasks on startup, keeps polling after close, and reopens the exact file", async () => {
    vi.useFakeTimers()
    api.jobs.mockResolvedValue([job])
    api.loadMovieDetail.mockResolvedValue({ id: "movie", files: [{ id: "part" }] })
    startSubtitleTracking(); await flushPromises()
    const state = useSubtitleJobs()
    expect(state.active.value).toEqual([job])
    await state.openJob(job)
    expect(state.dialogOpen.value).toBe(true)
    expect(state.fileId.value).toBe("part")
    state.dialogOpen.value = false
    api.jobs.mockResolvedValue([{ ...job, state: "completed" }])
    await vi.advanceTimersByTimeAsync(2000)
    expect(state.current.value?.state).toBe("completed")
    await state.openJob(state.current.value!)
    expect(state.dialogOpen.value).toBe(true)
    expect(api.loadMovieDetail).toHaveBeenCalledWith("movie")
  })
  it("retains the last snapshot on disconnect and ignores responses after shell teardown", async () => {
    api.jobs.mockResolvedValue([job])
    const state = useSubtitleJobs()
    await state.refresh()
    api.jobs.mockRejectedValue(new Error("offline"))
    await state.refresh()
    expect(state.connected.value).toBe(false)
    expect(state.current.value?.id).toBe("job")
    let resolve!: (value: unknown) => void
    api.jobs.mockReturnValue(new Promise(done => { resolve = done }))
    const pending = state.refresh()
    stopSubtitleTracking()
    resolve([job]); await pending
    expect(state.jobs.value).toEqual([])
  })
})
