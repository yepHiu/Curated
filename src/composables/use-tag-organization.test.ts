// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { TagOrganizationJob } from "@/services/contracts/topic-service"

const api = vi.hoisted(() => ({ getTagOrganizationStats: vi.fn(), listTagOrganizations: vi.fn(), startTagOrganization: vi.fn(), reloadMoviesFromApi: vi.fn().mockResolvedValue(undefined) }))
const toast = vi.hoisted(() => vi.fn())
const notify = vi.hoisted(() => vi.fn())
vi.mock("@/services/ai-service", () => ({ useAIService: () => api }))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => api }))
vi.mock("@/i18n", () => ({ i18n: { global: { t: (key: string, values?: Record<string, unknown>) => values ? `${key}:${JSON.stringify(values)}` : key, te: (key: string) => key === "topics.errors.AI_ORGANIZATION_TIMEOUT" } } }))
vi.mock("@/composables/use-app-toast", () => ({ pushAppToast: toast }))
vi.mock("@/composables/use-notification-center", () => ({ useNotificationCenter: () => ({ addNotification: notify }) }))
import { startTagOrganizationTracking, stopTagOrganizationTracking, useTagOrganization, organizationProgressText, organizationProgressValue, organizationErrorText } from "./use-tag-organization"

/** Build a minimal persistent job snapshot for lifecycle tests. */
function job(id: string, status: TagOrganizationJob["status"]): TagOrganizationJob {
 return { id, taskId: id, status, stage: "classifying", triggerReason: "manual", total: 6, processed: status === "completed" ? 6 : 0, succeeded: 6, unresolved: 0, failed: 0, revision: status === "completed" ? 2 : 1, createdAt: "2026-10-01", updatedAt: "2026-10-01" }
}

beforeEach(() => { vi.useFakeTimers(); vi.clearAllMocks(); api.getTagOrganizationStats.mockResolvedValue({ total: 6, organized: 2, unorganized: 3, outdated: 1, needsAttention: 0, unresolved: 1 }); api.listTagOrganizations.mockResolvedValue([]); api.startTagOrganization.mockResolvedValue(job("started", "queued")); useTagOrganization().quiet.value = false })
afterEach(() => { stopTagOrganizationTracking(); vi.useRealTimers() })

describe("organization observation", () => {
 it("reports skipped movies once after completion, never during processing", async () => {
  api.listTagOrganizations.mockResolvedValue([{ ...job('skip', 'running'), failed: 1 }])
  await useTagOrganization().refresh()
  expect(notify).not.toHaveBeenCalled()
  api.listTagOrganizations.mockResolvedValue([{ ...job('skip', 'partial_failed'), succeeded: 5, failed: 1 }])
  await useTagOrganization().refresh(); await useTagOrganization().refresh()
  expect(notify).toHaveBeenCalledOnce()
  expect(notify.mock.calls[0]![0].message).toContain('topics.completedWithIssues')
  expect(notify.mock.calls[0]![0].message).toContain('"failed":1')
  expect(api.startTagOrganization).not.toHaveBeenCalled()
 })
 it("only reads when browsing, emits once on completion and does not replay after remount", async () => {
  api.listTagOrganizations.mockResolvedValue([job("one","running")])
  startTagOrganizationTracking(); await useTagOrganization().refresh()
  expect(api.startTagOrganization).not.toHaveBeenCalled()
  api.listTagOrganizations.mockResolvedValue([job("one","completed")])
  await useTagOrganization().refresh(); await useTagOrganization().refresh()
  expect(toast).toHaveBeenCalledTimes(1); expect(notify).toHaveBeenCalledTimes(1)
  stopTagOrganizationTracking(); startTagOrganizationTracking(); await useTagOrganization().refresh()
  expect(toast).toHaveBeenCalledTimes(1)
 })
 it("keeps results in the center but stays quiet during playback", async () => {
  api.listTagOrganizations.mockResolvedValue([job("two","running")])
  await useTagOrganization().refresh()
  useTagOrganization().quiet.value = true
  api.listTagOrganizations.mockResolvedValue([job("two","completed")])
  await useTagOrganization().refresh()
  expect(notify).toHaveBeenCalledTimes(1); expect(toast).not.toHaveBeenCalled()
 })
 it("does not toast completion learned from a reconnect snapshot", async () => {
  api.listTagOrganizations.mockResolvedValue([job("three","running")]); await useTagOrganization().refresh()
  api.listTagOrganizations.mockRejectedValueOnce(new Error("offline")); await useTagOrganization().refresh()
  api.listTagOrganizations.mockResolvedValue([job("three","completed")]); await useTagOrganization().refresh()
  expect(toast).not.toHaveBeenCalled(); expect(useTagOrganization().connected.value).toBe(true)
 })
 it("ignores a stale request after the observer lifecycle is replaced", async () => {
  let resolveOld!: (jobs: TagOrganizationJob[]) => void
  api.listTagOrganizations.mockReturnValueOnce(new Promise<TagOrganizationJob[]>((resolve) => { resolveOld = resolve }))
  const old = useTagOrganization().refresh(); stopTagOrganizationTracking()
  api.listTagOrganizations.mockResolvedValue([job("new","running")]); await useTagOrganization().refresh()
  resolveOld([job("stale","completed")]); await old
  expect(useTagOrganization().jobs.value[0]?.id).toBe("new")
 })
})


describe("organization diagnostics", () => {
 it("separates preparation progress from applied movies and explains quota waits", () => {
  const preparing = { ...job("progress", "running"), total: 100, stage: "vocabulary", vocabularyProcessed: 50 }
  expect(organizationProgressText(preparing)).toBe('topics.vocabularyProgress:{"done":50,"total":100}')
  expect(organizationProgressText({ ...preparing, vocabularyProcessed: undefined })).toBe('topics.vocabularyProgress:{"done":0,"total":100}')
  expect(organizationProgressText({ ...preparing, stage: "waiting_quota", processed: 10 })).toBe('topics.waitingQuota:{"done":10,"total":100}')
  expect(organizationProgressText({ ...preparing, stage: "classifying", processed: 15 })).toBe('topics.progress:{"done":15,"total":100}')
 })
 it("translates known failures and never exposes unknown provider messages", () => {
  expect(organizationErrorText("AI_ORGANIZATION_TIMEOUT")).toBe("topics.errors.AI_ORGANIZATION_TIMEOUT")
  expect(organizationErrorText("private provider response")).toBe("topics.needsAttention")
 })
})

describe("organization scope", () => {
 it("retries a problem movie or the problem queue only after explicit invocation", async () => {
  const state = useTagOrganization()
  api.getTagOrganizationStats.mockResolvedValue({ total: 1, organized: 0, unorganized: 0, outdated: 0, needsAttention: 1, unresolved: 0 })
  await state.refreshStats()
  await state.start()
  expect(api.startTagOrganization).not.toHaveBeenCalled()
  await state.retryMovie('problem')
  expect(api.startTagOrganization).toHaveBeenLastCalledWith('selected', ['problem'])
  await state.retryIssues()
  expect(api.startTagOrganization).toHaveBeenLastCalledWith('issues')
 })
 it("requires available coverage and never falls back to all movies", async () => {
  const state = useTagOrganization()
  state.openAll(); await state.start()
  expect(api.startTagOrganization).not.toHaveBeenCalled()
  api.getTagOrganizationStats.mockRejectedValueOnce(new Error("offline"))
  await state.refreshStats(); await state.start()
  expect(state.statsError.value).toBe(true)
  expect(api.startTagOrganization).not.toHaveBeenCalled()
  api.getTagOrganizationStats.mockResolvedValue({ total: 6, organized: 5, unorganized: 0, outdated: 1, needsAttention: 0, unresolved: 2 })
  await state.refreshStats(); await state.start()
  expect(api.startTagOrganization).not.toHaveBeenCalled()
  await state.start("outdated")
  expect(api.startTagOrganization).toHaveBeenLastCalledWith("outdated", undefined)
  await state.start("all")
  expect(api.startTagOrganization).toHaveBeenLastCalledWith("all", undefined)
 })
 it("does not let a late coverage response leak across server sessions", async () => {
  let resolveOld!: (value: { total: number; organized: number; unorganized: number; outdated: number; needsAttention: number; unresolved: number }) => void
  api.getTagOrganizationStats.mockReturnValueOnce(new Promise((resolve) => { resolveOld = resolve }))
  const pending = useTagOrganization().refreshStats()
  stopTagOrganizationTracking()
  resolveOld({ total: 100, organized: 100, unorganized: 0, outdated: 0, needsAttention: 0, unresolved: 0 })
  await pending
  expect(useTagOrganization().stats.value).toBeNull()
 })
 it("captures selected IDs, removes duplicates, and starts only after the explicit action", async () => {
  const state = useTagOrganization()
  const ids = ["one", "two", "one"]
  state.openSelected(ids)
  ids.push("outside")
  expect(state.dialogOpen.value).toBe(true)
  expect(api.startTagOrganization).not.toHaveBeenCalled()
  await state.start()
  expect(api.startTagOrganization).toHaveBeenCalledWith("selected", ["one", "two"])
 })
 it("captures a single movie title and switches back to the full-library scope", async () => {
  const state = useTagOrganization()
  state.openSelected(["one"], "A movie")
  expect(state.selection.value?.title).toBe("A movie")
  state.openAll()
  await state.refreshStats()
  await state.start()
  expect(state.selection.value).toBeNull()
  expect(api.startTagOrganization).toHaveBeenCalledWith("unorganized", undefined)
 })
 it("rejects empty selections, excessive selections and duplicate active jobs", async () => {
  const state = useTagOrganization()
  state.openSelected([]); await state.start()
  state.openSelected(Array.from({ length: 601 }, (_, i) => String(i))); await state.start()
  state.jobs.value = [job("active", "running")]
  state.openSelected(["one"]); await state.start()
  expect(api.startTagOrganization).not.toHaveBeenCalled()
 })
 it("shows stage-specific progress and bounds invalid counts", () => {
  const running = { ...job("progress", "running"), total: 100, processed: 20, vocabularyProcessed: 60 }
  expect(organizationProgressValue({ ...running, stage: "vocabulary" })).toBe(60)
  expect(organizationProgressValue(running)).toBe(20)
  expect(organizationProgressValue({ ...running, total: 0 })).toBe(0)
  expect(organizationProgressValue({ ...running, processed: 200 })).toBe(100)
 })
})
