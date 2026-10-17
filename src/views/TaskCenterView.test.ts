import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { createMemoryHistory, createRouter } from "vue-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { i18n } from "@/i18n"
import TaskCenterView from "./TaskCenterView.vue"
import { Select } from "@/components/ui/select"
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
async function setup(path = "/tasks?state=active&kind=subtitle&pages=2&scroll=320") {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: "/tasks", component: TaskCenterView },
    { path: "/detail/:id", name: "detail", component: { template: "<div />" } },
  ] })
  await router.push(path)
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


describe("task center pagination", () => {
  function pagedTasks() {
    const tasks = Array.from({ length: 125 }, (_, index) => ({ ...structuredClone(job), key: `subtitle:${index}`, id: String(index) }))
    api.list.mockImplementation(async ({ cursor, limit }) => {
      const start = Number(cursor || 0)
      return { items: tasks.slice(start, start + limit), nextCursor: start + limit < tasks.length ? String(start + limit) : "" }
    })
  }
  it("replaces pages, returns to previous pages and refreshes only the current cursor", async () => {
    pagedTasks()
    const { wrapper } = await setup("/tasks?state=history")
    expect(api.list).toHaveBeenLastCalledWith({ kind: "all", state: "history", cursor: undefined, limit: 20 })
    expect(wrapper.findAll('button[aria-haspopup="dialog"]')).toHaveLength(20)
    expect(wrapper.get("[data-task-previous]").attributes("disabled")).toBeDefined()
    await wrapper.get("[data-task-next]").trigger("click"); await flushPromises()
    expect(api.list).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "20", limit: 20 }))
    expect(wrapper.findAll('button[aria-haspopup="dialog"]')).toHaveLength(20)
    const list = wrapper.get("[data-task-center]").element as HTMLElement
    list.scrollTop = 180
    api.list.mockClear(); revision.value++; await flushPromises()
    expect(api.list).toHaveBeenCalledExactlyOnceWith(expect.objectContaining({ cursor: "20", limit: 20 }))
    expect(list.scrollTop).toBe(180)
    await wrapper.get("[data-task-previous]").trigger("click"); await flushPromises()
    expect(api.list).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: undefined, limit: 20 }))
    expect(list.scrollTop).toBe(0)
  })
  it("supports 50 and 100 per page, resets page on size/filter changes and stops at the last page", async () => {
    pagedTasks()
    const { wrapper, router } = await setup("/tasks")
    const size = wrapper.findAllComponents(Select).find(select => select.props("modelValue") === "20")!
    size.vm.$emit("update:modelValue", "50"); await flushPromises()
    expect(router.currentRoute.value.query.pageSize).toBe("50")
    expect(wrapper.findAll('button[aria-haspopup="dialog"]')).toHaveLength(50)
    await wrapper.get("[data-task-next]").trigger("click"); await flushPromises()
    expect(api.list).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "50", limit: 50 }))
    size.vm.$emit("update:modelValue", "100"); await flushPromises()
    expect(api.list).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: undefined, limit: 100 }))
    expect(wrapper.findAll('button[aria-haspopup="dialog"]')).toHaveLength(100)
    await wrapper.get("[data-task-next]").trigger("click"); await flushPromises()
    expect(wrapper.findAll('button[aria-haspopup="dialog"]')).toHaveLength(25)
    expect(wrapper.get("[data-task-next]").attributes("disabled")).toBeDefined()
    await router.replace({ query: { pageSize: "100", kind: "organize" } }); await flushPromises()
    expect(api.list).toHaveBeenLastCalledWith(expect.objectContaining({ kind: "organize", cursor: undefined, limit: 100 }))
    expect(wrapper.get("[data-task-previous]").attributes("disabled")).toBeDefined()
  })
  it("ignores a late page response after filters change", async () => {
    pagedTasks()
    const { wrapper, router } = await setup("/tasks?pageSize=invalid")
    let finish!: (value: unknown) => void
    api.list.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    await wrapper.get("[data-task-next]").trigger("click")
    await router.replace({ query: { kind: "organize" } }); await flushPromises()
    finish({ items: [{ ...job, key: "late" }], nextCursor: "late" }); await flushPromises()
    expect(wrapper.findAll('button[aria-haspopup="dialog"]')).toHaveLength(20)
    expect(wrapper.get("[data-task-previous]").attributes("disabled")).toBeDefined()
  })
})
