import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { createMemoryHistory, createRouter } from "vue-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { i18n } from "@/i18n"
import TaskCenterView from "./TaskCenterView.vue"
import { projectSubtitle } from "@/lib/task-center"
const api = vi.hoisted(() => ({ list: vi.fn(), get: vi.fn(), action: vi.fn(), refresh: vi.fn(), reloadMoviesFromApi: vi.fn() }))
const revision = ref(0)
vi.mock("@/services/task-center-service", () => ({ useTaskCenterService: () => api }))
vi.mock("@/composables/use-task-center", () => ({ useTaskCenter: () => ({ revision, refresh: api.refresh, connected: ref(true), updatedAt: ref(0) }) }))
vi.mock("@/composables/use-tag-organization", () => ({ useTagOrganization: () => api, organizationErrorText: (s: string) => s }))
vi.mock("@/composables/use-subtitle-jobs", () => ({ useSubtitleJobs: () => api }))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => api }))
vi.mock("@/services/ai-service", () => ({ useAIService: () => ({ getTagOrganizationItems: vi.fn(async () => []) }) }))
vi.mock("@/services/subtitle-service", () => ({ useSubtitleService: () => ({}) }))
vi.mock("@/lib/experimental-agent", () => ({ useExperimentalAgent: () => ({ writeEnabled: ref(true) }) }))
const job = projectSubtitle({ id: "a", movieId: "movie", fileId: "file", fileName: "part.mp4", state: "running", stage: "translating", completed: 8, total: 12, assets: [], createdAt: "2026-10-17T00:00:00Z", updatedAt: "2026-10-17T00:01:00Z", translationModel: "qwen-flash" })

let wrapper: ReturnType<typeof mount> | undefined
async function setup() {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: "/tasks", component: TaskCenterView },
    { path: "/detail/:id", name: "detail", component: { template: "<div />" } },
  ] })
  await router.push("/tasks?state=active&kind=subtitle&pages=2&scroll=320")
  wrapper = mount(TaskCenterView, { attachTo: document.body, global: { plugins: [router, i18n] } })
  await flushPromises()
  return { wrapper, router }
}
beforeEach(() => {
  vi.clearAllMocks()
  i18n.global.locale.value = "zh-CN"
  api.get.mockResolvedValue(structuredClone(job))
  api.list.mockResolvedValue({ items: [structuredClone(job)], nextCursor: "" })
})
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = "" })

describe("task center detail dialog", () => {
  it("opens details without navigation and restores list context and focus on close", async () => {
    const { wrapper, router } = await setup()
    const originalRoute = router.currentRoute.value.fullPath
    const list = wrapper.get("[data-task-center]").element as HTMLElement
    list.scrollTop = 320
    const row = wrapper.get('button[aria-haspopup="dialog"]')
    await row.trigger("click")
    await vi.waitFor(() => expect(document.querySelector("[data-task-detail]")).not.toBeNull())
    await flushPromises()
    expect(api.get).toHaveBeenCalledWith("subtitle:a")
    expect(router.currentRoute.value.fullPath).toBe(originalRoute)
    expect(document.querySelector("[data-task-detail]")?.textContent).toContain("part.mp4")
    const dialog = document.querySelector('[data-slot="dialog-content"]')!
    ;(dialog.querySelector('[data-slot="dialog-close"]') as HTMLElement).click()
    await flushPromises()
    await vi.waitFor(() => expect(document.querySelector("[data-task-detail]")).toBeNull())
    expect(list.scrollTop).toBe(320)
    expect(router.currentRoute.value.fullPath).toBe(originalRoute)
    expect(document.activeElement).toBe(row.element)
    expect(api.list).toHaveBeenCalledTimes(1)
  })

  it("discards pending detail responses when closed and loads fresh details when reopened", async () => {
    let finish!: (value: typeof job) => void
    api.get.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const { wrapper } = await setup()
    const row = wrapper.get('button[aria-haspopup="dialog"]')
    await row.trigger("click")
    await vi.waitFor(() => expect(api.get).toHaveBeenCalledTimes(1))
    ;(document.querySelector('[data-slot="dialog-close"]') as HTMLElement).click()
    await flushPromises()
    finish(job)
    await flushPromises()
    expect(document.querySelector("[data-task-detail]")).toBeNull()
    await row.trigger("click")
    await vi.waitFor(() => expect(api.get).toHaveBeenCalledTimes(2))
    await flushPromises()
    expect(document.querySelector("[data-task-detail]")?.textContent).toContain("part.mp4")
    expect(api.action).not.toHaveBeenCalled()
  })
})
