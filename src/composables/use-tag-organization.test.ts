// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { TagOrganizationJob } from "@/services/contracts/topic-service"

const api = vi.hoisted(() => ({ listTagOrganizations: vi.fn(), startTagOrganization: vi.fn(), reloadMoviesFromApi: vi.fn().mockResolvedValue(undefined) }))
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

beforeEach(() => { vi.useFakeTimers(); vi.clearAllMocks(); api.listTagOrganizations.mockResolvedValue([]); api.startTagOrganization.mockResolvedValue(job("started", "queued")); useTagOrganization().quiet.value = false })
afterEach(() => { stopTagOrganizationTracking(); vi.useRealTimers() })

describe("organization observation", () => {
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
  await state.start()
  expect(state.selection.value).toBeNull()
  expect(api.startTagOrganization).toHaveBeenCalledWith("all", undefined)
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
