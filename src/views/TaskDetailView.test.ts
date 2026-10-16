import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { createMemoryHistory, createRouter } from "vue-router"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { i18n } from "@/i18n"
import TaskDetailView from "./TaskDetailView.vue"
import { projectSubtitle } from "@/lib/task-center"
const api = vi.hoisted(() => ({ get: vi.fn(), action: vi.fn(), refresh: vi.fn(), reloadMoviesFromApi: vi.fn() }))
const revision = ref(0)
vi.mock("@/services/task-center-service", () => ({ useTaskCenterService: () => api }))
vi.mock("@/composables/use-task-center", () => ({ useTaskCenter: () => ({ revision, refresh: api.refresh }) }))
vi.mock("@/composables/use-tag-organization", () => ({ useTagOrganization: () => api, organizationErrorText: (s: string) => s }))
vi.mock("@/composables/use-subtitle-jobs", () => ({ useSubtitleJobs: () => api }))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => api }))
vi.mock("@/services/ai-service", () => ({ useAIService: () => ({ getTagOrganizationItems: vi.fn(async () => []) }) }))
vi.mock("@/services/subtitle-service", () => ({ useSubtitleService: () => ({}) }))
vi.mock("@/lib/experimental-agent", () => ({ useExperimentalAgent: () => ({ writeEnabled: ref(true) }) }))
vi.mock("@/components/ui/dialog", () => ({ Dialog: { props: ["open"], template: '<section v-if="open" role="dialog"><slot /></section>' }, ...Object.fromEntries(["DialogContent", "DialogHeader", "DialogTitle", "DialogDescription", "DialogFooter"].map(name => [name, { template: '<div><slot /></div>' }])) }))
const job = projectSubtitle({ id: "a", movieId: "movie", fileId: "file", fileName: "part.mp4", state: "running", stage: "translating", completed: 8, total: 12, assets: [], createdAt: "2026-10-17T00:00:00Z", updatedAt: "2026-10-17T00:01:00Z", translationModel: "qwen-flash" })
async function setup() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: "/tasks/:taskKey", component: TaskDetailView }, { path: "/tasks", component: { template: '<div />' } }, { path: "/detail/:id", name: "detail", component: { template: '<div />' } }] })
  await router.push("/tasks/subtitle:a")
  const wrapper = mount(TaskDetailView, { global: { plugins: [router, i18n] } }); await flushPromises()
  return { wrapper, router }
}
beforeEach(() => { vi.clearAllMocks(); i18n.global.locale.value = "zh-CN"; api.get.mockResolvedValue(structuredClone(job)) })
describe("task detail behavior", () => {
  it("moves the active stage from loading to success or error and stops on disconnect", async () => {
    const { wrapper } = await setup()
    const states = () => wrapper.findAll('[data-node-state]').map(n => n.attributes('data-node-state'))
    expect(states()).toEqual(["success", "success", "running", "pending"])
    api.get.mockResolvedValue({ ...job, state: "failed" }); revision.value++; await flushPromises()
    expect(states()).toEqual(["success", "success", "error", "pending"])
    api.get.mockResolvedValue({ ...job, stage: "uploading" }); revision.value++; await flushPromises()
    expect(states()).toEqual(["success", "running", "pending", "pending"])
    api.get.mockRejectedValueOnce(new Error("offline")); revision.value++; await flushPromises()
    expect(states()).toEqual(["success", "paused", "pending", "pending"])
    api.get.mockResolvedValue({ ...job, state: "cancelled" }); revision.value++; await flushPromises()
    expect(states()).toEqual(["success", "success", "paused", "pending"])
    api.get.mockResolvedValue({ ...job, state: "succeeded", stage: "done" }); revision.value++; await flushPromises()
    expect(states()).toEqual(["success", "success", "success", "success"])
    wrapper.unmount()
  })
  it("requires confirmation before cancellation and respects a read-only connection", async () => {
    const { wrapper } = await setup()
    const cancel = wrapper.findAll("button").find(b => b.text() === "取消任务")!
    await cancel.trigger("click"); expect(api.action).not.toHaveBeenCalled()
    await wrapper.find('[role="dialog"]').findAll("button").find(b => b.text() === "确认")!.trigger("click")
    await flushPromises(); expect(api.action).toHaveBeenCalledWith(expect.objectContaining({ key: "subtitle:a" }), "cancel")
    api.get.mockResolvedValue({ ...job, canManage: false }); revision.value++; await flushPromises()
    expect(wrapper.findAll("button").find(b => b.text() === "取消任务")!.attributes("disabled")).toBeDefined()
    wrapper.unmount()
  })
  it("keeps progress on network failure and discards late details after navigating", async () => {
    const { wrapper, router } = await setup()
    api.get.mockRejectedValueOnce(new Error("offline")); revision.value++; await flushPromises()
    expect(wrapper.text()).toContain("part.mp4")
    expect(wrapper.text()).toContain("最近状态")
    expect(wrapper.findAll("button").find(b => b.text() === "取消任务")!.attributes("disabled")).toBeDefined()
    let finish!: (v: typeof job) => void
    api.get.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    revision.value++; await flushPromises()
    api.get.mockResolvedValue({ ...job, key: "subtitle:b", id: "b", subtitle: { ...job.subtitle!, fileName: "other.mp4" } })
    await router.push("/tasks/subtitle:b"); await flushPromises()
    finish(job); await flushPromises()
    expect(wrapper.text()).toContain("other.mp4"); expect(wrapper.text()).not.toContain("part.mp4")
    wrapper.unmount()
  })
})
