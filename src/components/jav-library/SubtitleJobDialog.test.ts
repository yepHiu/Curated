import { mount, flushPromises } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { ref } from "vue"
import type { Movie } from "@/domain/movie/types"
import type { SubtitleJob } from "@/services/contracts/subtitle-service"
import SubtitleJobDialog from "./SubtitleJobDialog.vue"
import SubtitleModelSelect from "./SubtitleModelSelect.vue"
import MoviePartSelect from "./MoviePartSelect.vue"
import TaskDetailDialog from "./TaskDetailDialog.vue"
import { stopSubtitleTracking } from "@/composables/use-subtitle-jobs"
vi.mock("./TaskDetailContent.vue", () => ({ __esModule: true, default: {
  name: "TaskDetailContent", props: ["taskKey"],
  template: '<section data-task-detail>{{ taskKey }}</section>',
} }))
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
  stopSubtitleTracking(); vi.clearAllMocks()
  mocks.settings.mockResolvedValue({ enabled: true, canManage: true, apiKeyConfigured: true, translationModel: "qwen-flash", configuredModel: "custom" })
  mocks.jobs.mockResolvedValue([]); mocks.audioTracks.mockResolvedValue([{ index: 0, language: "ja", title: "", codec: "aac", channels: 2 }]); mocks.create.mockResolvedValue(running)
})
afterEach(() => { stopSubtitleTracking() })
const button = (wrapper: ReturnType<typeof mount>, label: string) => wrapper.findAll("button").find(item => item.text() === label)!

describe("subtitle creation and shared task details", () => {
  it("starts only on request, switches the same dialog to task details, and never cancels on close", async () => {
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    expect(mocks.create).not.toHaveBeenCalled()
    await button(wrapper, "subtitles.start").trigger("click"); await flushPromises()
    expect(mocks.create).toHaveBeenCalledWith("movie", expect.objectContaining({ fileId: "part-1", translationModel: "qwen-flash", audioTrack: 0, requestId: expect.any(String) }))
    expect(wrapper.getComponent(TaskDetailDialog).props("taskKey")).toBe("subtitle:job")
    expect(wrapper.find('[data-subtitle-setup]').exists()).toBe(false)
    expect(wrapper.findAllComponents(TaskDetailDialog)).toHaveLength(1)
    await wrapper.setProps({ open: false })
    expect(mocks.action).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it("opens an active job in shared details and keeps its frozen model when returning to setup", async () => {
    mocks.jobs.mockResolvedValue([running])
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    expect(wrapper.getComponent(TaskDetailDialog).props("taskKey")).toBe("subtitle:job")
    await button(wrapper, "subtitles.setupAndHistory").trigger("click")
    expect(wrapper.getComponent(SubtitleModelSelect).props()).toMatchObject({ modelValue: "qwen-plus", disabled: true })
    expect(button(wrapper, "subtitles.start")).toBeUndefined()
    await button(wrapper, "tasks.viewTask").trigger("click")
    expect(wrapper.getComponent(TaskDetailDialog).props("taskKey")).toBe("subtitle:job")
    expect(mocks.create).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it("routes every historical generation, including failures, to its exact shared task details", async () => {
    const asset = { id: "asset", fileId: "part-1", name: "subtitle.srt", language: "ja", format: "srt", size: 12, hash: "hash", missing: true }
    mocks.jobs.mockResolvedValue([
      running,
      { ...running, id: "old-1", state: "completed", createdAt: "2026-10-15T12:00:00Z", assets: [asset] },
      { ...running, id: "old-2", state: "failed", createdAt: "2026-10-14T12:00:00Z" },
    ])
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    await button(wrapper, "subtitles.setupAndHistory").trigger("click")
    const rows = wrapper.get('[data-subtitle-history]').findAll('li > button')
    expect(rows).toHaveLength(3)
    await rows[1]!.trigger('click')
    expect(wrapper.getComponent(TaskDetailDialog).props("taskKey")).toBe("subtitle:old-1")
    await button(wrapper, "subtitles.setupAndHistory").trigger("click")
    await wrapper.get('[data-subtitle-history]').findAll('li > button')[2]!.trigger('click')
    expect(wrapper.getComponent(TaskDetailDialog).props("taskKey")).toBe("subtitle:old-2")
    expect(mocks.create).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it("disables remote creation but keeps saved tasks available for inspection", async () => {
    mocks.settings.mockResolvedValue({ enabled: true, canManage: false, apiKeyConfigured: true, translationModel: "qwen-plus", configuredModel: "custom" })
    mocks.jobs.mockResolvedValue([{ ...running, state: "completed" }])
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    expect(button(wrapper, "subtitles.start").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("subtitles.localOnly")
    await wrapper.get('[data-subtitle-history] button').trigger("click")
    expect(wrapper.getComponent(TaskDetailDialog).props("taskKey")).toBe("subtitle:job")
    wrapper.unmount()
  })

  it("keeps creation and history scoped to the selected file and ignores late audio probes", async () => {
    const multi = { ...movie, files: [...movie.files!, { id: "part-2", fileName: "DEMO-001.CD2.mp4", partIndex: 2 }] } as Movie
    let finish!: (value: unknown) => void
    mocks.audioTracks.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    mocks.jobs.mockResolvedValue([{ ...running, state: "completed" }])
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie: multi } }); await flushPromises()
    mocks.audioTracks.mockResolvedValue([{ index: 2, language: "ja", title: "", codec: "aac", channels: 2 }])
    wrapper.getComponent(MoviePartSelect).vm.$emit("update:modelValue", "part-2"); await flushPromises()
    finish([]); await flushPromises()
    expect(wrapper.find('[data-subtitle-history]').exists()).toBe(false)
    await button(wrapper, "subtitles.start").trigger("click"); await flushPromises()
    expect(mocks.create).toHaveBeenCalledWith("movie", expect.objectContaining({ fileId: "part-2", audioTrack: 2 }))
    wrapper.unmount()
  })

  it("keeps historical tasks accessible if loading settings fails", async () => {
    mocks.settings.mockRejectedValue(new Error("settings offline"))
    mocks.jobs.mockResolvedValue([{ ...running, state: "failed" }])
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    expect(wrapper.text()).toContain("settings offline")
    expect(button(wrapper, "subtitles.start").attributes("disabled")).toBeDefined()
    await wrapper.get('[data-subtitle-history] button').trigger("click")
    expect(wrapper.getComponent(TaskDetailDialog).props("taskKey")).toBe("subtitle:job")
    wrapper.unmount()
  })

  it("does not reopen details when a submitted job returns after closing", async () => {
    let finish!: (value: SubtitleJob) => void
    mocks.create.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const wrapper = mount(SubtitleJobDialog, { props: { open: true, movie } }); await flushPromises()
    await button(wrapper, "subtitles.start").trigger("click")
    await wrapper.setProps({ open: false })
    finish(running); await flushPromises()
    expect(wrapper.getComponent(TaskDetailDialog).props()).toMatchObject({ open: false, taskKey: "" })
    expect(mocks.action).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

vi.mock("vue-router", () => ({ RouterLink: { props: ["to"], template: '<a :href="String(to)"><slot /></a>' } }))

vi.mock("@/i18n", () => ({ i18n: { global: { t: (key: string) => key } } }))
