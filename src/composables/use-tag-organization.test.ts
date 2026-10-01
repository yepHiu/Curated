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
import { startTagOrganizationTracking, stopTagOrganizationTracking, useTagOrganization, organizationProgressText, organizationErrorText } from "./use-tag-organization"

/** Build a minimal persistent job snapshot for lifecycle tests. */
function job(id: string, status: TagOrganizationJob["status"]): TagOrganizationJob {
 return { id, taskId: id, status, stage: "classifying", triggerReason: "manual", total: 6, processed: status === "completed" ? 6 : 0, succeeded: 6, unresolved: 0, failed: 0, revision: status === "completed" ? 2 : 1, createdAt: "2026-10-01", updatedAt: "2026-10-01" }
}

beforeEach(() => { vi.useFakeTimers(); vi.clearAllMocks(); api.listTagOrganizations.mockResolvedValue([]); useTagOrganization().quiet.value = false })
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
