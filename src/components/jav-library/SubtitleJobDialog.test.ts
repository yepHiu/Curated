import { mount, flushPromises } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { ref } from "vue"
import type { Movie } from "@/domain/movie/types"
import type { SubtitleJob } from "@/services/contracts/subtitle-service"
import SubtitleJobDialog from "./SubtitleJobDialog.vue"
import SubtitleModelSelect from "./SubtitleModelSelect.vue"
const mocks = vi.hoisted(() => ({ settings: vi.fn(), jobs: vi.fn(), audioTracks: vi.fn(), create: vi.fn(), action: vi.fn() }))
vi.mock("@/services/subtitle-service", () => ({ useSubtitleService: () => mocks }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ locale: ref("en"), t: (key: string) => key }) }))
vi.mock("@/components/ui/dialog", () => ({
  Dialog: { props: ["open"], emits: ["update:open"], template: '<div v-if="open"><slot /></div>' },
  ...Object.fromEntries(["DialogContent", "DialogHeader", "DialogTitle", "DialogDescription", "DialogFooter"].map(name => [name, { template: "<div><slot /></div>" }])),
}))
const movie = { id: "movie", code: "DEMO-001", files: [{ id: "part-1", fileName: "DEMO-001.CD1.mp4", partIndex: 1 }] } as Movie
const running: SubtitleJob = { id: "job", movieId: "movie", fileId: "part-1", fileName: "DEMO-001.CD1.mp4", state: "running", stage: "translating", translationModel: "qwen-plus", completed: 10, total: 30, createdAt: "", updatedAt: "", assets: [] }
beforeEach(() => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] }); vi.clearAllMocks()
  mocks.settings.mockResolvedValue({ enabled: true, canManage: true, apiKeyConfigured: true, translationModel: "qwen-flash", configuredModel: "custom" })
  mocks.jobs.mockResolvedValue([]); mocks.audioTracks.mockResolvedValue([{ index: 0, language: "ja", title: "", codec: "aac", channels: 2 }]); mocks.create.mockResolvedValue(running)
})
afterEach(() => { vi.useRealTimers() })
describe("subtitle dialog owns interaction while Server owns execution", () => {
  it("requires an explicit start and does not cancel when closed", async () => {
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    expect(mocks.create).not.toHaveBeenCalled()
    const start = wrapper.findAll("button").find(button => button.text() === "subtitles.start")!
    expect(start.attributes("disabled")).toBeUndefined(); await start.trigger("click"); await flushPromises()
    expect(mocks.create).toHaveBeenCalledWith("movie", expect.objectContaining({ fileId: "part-1", translationModel: "qwen-flash", audioTrack: 0, requestId: expect.any(String) }))
    await wrapper.setProps({ open: false }); expect(mocks.action).not.toHaveBeenCalled(); wrapper.unmount()
  })
  it("restores the active job and its frozen model instead of the newer default", async () => {
    mocks.jobs.mockResolvedValue([running])
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    expect(wrapper.getComponent(SubtitleModelSelect).props()).toMatchObject({ modelValue: "qwen-plus", disabled: true })
    expect(wrapper.text()).toContain("10 / 30"); expect(mocks.create).not.toHaveBeenCalled(); wrapper.unmount()
  })
  it("disables remote task creation", async () => {
    mocks.settings.mockResolvedValue({ enabled: true, canManage: false, apiKeyConfigured: true, translationModel: "qwen-plus", configuredModel: "custom" })
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    expect(wrapper.findAll("button").find(button => button.text() === "subtitles.start")?.attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("subtitles.localOnly"); wrapper.unmount()
  })
})
