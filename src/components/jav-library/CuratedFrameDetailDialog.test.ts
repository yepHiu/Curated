import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import CuratedFrameDetailDialog from "./CuratedFrameDetailDialog.vue"
import type { CuratedFrameDialogItem, CuratedFrameDialogNavigationEntry } from "@/lib/curated-frames/dialog-navigation"

const { updateTags, deleteFrame, exportFrames, push } = vi.hoisted(() => ({
  updateTags: vi.fn(), deleteFrame: vi.fn(), exportFrames: vi.fn(), push: vi.fn(),
}))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key, locale: ref("en") }) }))
vi.mock("vue-router", () => ({ useRoute: () => ({ query: {} }), useRouter: () => ({ push }) }))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => ({
  curatedFrameExportFormat: ref("png"), curatedFrameExportMode: ref("raw"),
}) }))
vi.mock("@/composables/use-curated-frame-export", () => ({ useCuratedFrameExport: () => ({ exportFrames }) }))
vi.mock("@/composables/use-app-toast", () => ({ pushAppToast: vi.fn() }))
vi.mock("@/lib/curated-frames/db", () => ({
  updateCuratedFrameTags: updateTags, deleteCuratedFrame: deleteFrame,
  listCuratedFrameTagSuggestions: vi.fn().mockResolvedValue([]),
}))

function item(id: string): CuratedFrameDialogItem {
  return { url: `thumb-${id}`, row: {
    id, movieId: "movie-a", code: `CODE-${id}`, title: `Movie ${id}`, actors: ["Actor A", "Actor B"],
    tags: ["warm"], positionSec: 65, capturedAt: "2026-10-01T00:00:00Z",
  } }
}
function entries(...items: CuratedFrameDialogItem[]): CuratedFrameDialogNavigationEntry<CuratedFrameDialogItem>[] {
  return items.map((item) => ({ item, sectionActor: null }))
}
const wrappers: ReturnType<typeof mount>[] = []
function render(navigation = entries(item("a"), item("b"), item("c"), item("d"))) {
  const wrapper = mount(CuratedFrameDetailDialog, {
    props: { entries: navigation },
    attachTo: document.body,
    global: { stubs: {
      Dialog: { props: ["open"], template: '<div v-if="open" data-dialog><slot /></div>' },
      DialogContent: { template: '<div><slot /></div>' },
      DialogHeader: { template: '<header><slot /></header>' },
      DialogTitle: { template: '<h2><slot /></h2>' },
      DialogDescription: { template: '<p><slot /></p>' },
      Carousel: { template: '<div><slot /></div>' },
      CarouselContent: { template: '<div><slot /></div>' },
      CarouselItem: { template: '<div><slot /></div>' },
      FrameImageViewer: { props: ["src", "alt", "active"], template: '<img data-original :data-active="active" :src="src" />' },
      CuratedFrameDeleteConfirmDialog: { props: ["open", "error"], emits: ["confirm"], template: '<div v-if="open" data-delete-confirm><p>{{ error }}</p><button @click="$emit(\'confirm\')">confirm</button></div>' },
    } },
  })
  wrappers.push(wrapper)
  return wrapper
}
async function open(wrapper: ReturnType<typeof render>, entry = wrapper.props("entries")[0]!) {
  wrapper.vm.open(entry.item, entry.sectionActor)
  await flushPromises()
}
function button(wrapper: ReturnType<typeof render>, label: string) {
  return wrapper.findAll("button").find((candidate) => candidate.text() === label)!
}
function currentImage(wrapper: ReturnType<typeof render>) {
  return wrapper.get('[data-original][data-active="true"]').attributes("src")
}
beforeEach(() => {
  updateTags.mockReset().mockResolvedValue(undefined)
  deleteFrame.mockReset().mockResolvedValue(undefined)
  exportFrames.mockReset().mockResolvedValue(undefined)
  push.mockReset().mockResolvedValue(undefined)
})
afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.useRealTimers()
})

describe("CuratedFrameDetailDialog", () => {
  it("shows metadata and loads originals only for the selected and adjacent frames", async () => {
    const wrapper = render()
    await open(wrapper)
    expect(wrapper.text()).toContain("Movie a")
    expect(wrapper.text()).toContain("Actor A、Actor B")
    expect(currentImage(wrapper)).toContain("/a/image")
    expect(wrapper.findAll("[data-original]").map((image) => image.attributes("src"))).toEqual([
      expect.stringContaining("/a/image"), expect.stringContaining("/b/image"), "thumb-c", "thumb-d",
    ])
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true }))
    await flushPromises()
    expect(currentImage(wrapper)).toContain("/b/image")
    await wrapper.get('[aria-label="curated.previousFrame"]').trigger("click")
    await flushPromises()
    expect(currentImage(wrapper)).toContain("/a/image")
  })

  it("keeps tag drafts and the current frame when saving fails, then supports retry", async () => {
    const wrapper = render()
    await open(wrapper)
    updateTags.mockRejectedValueOnce(new Error("offline"))
    await wrapper.get('[aria-label="curated.ariaRemoveTag"]').trigger("click")
    await wrapper.get('[aria-label="curated.nextFrame"]').trigger("click")
    await flushPromises()
    expect(updateTags).toHaveBeenCalledWith("a", [])
    expect(currentImage(wrapper)).toContain("/a/image")
    expect(wrapper.text()).toContain("curated.tagSaveFailed")
    await button(wrapper, "curated.tagSaveRetry").trigger("click")
    await flushPromises()
    expect(wrapper.emitted("tagsSaved")?.at(-1)).toEqual([{ id: "a", tags: [] }])
    await wrapper.get('[aria-label="curated.nextFrame"]').trigger("click")
    await flushPromises()
    expect(currentImage(wrapper)).toContain("/b/image")
  })

  it("keeps actor-group identity for duplicated frames and exports using the active actor", async () => {
    const shared = item("shared")
    const navigation = [
      { item: shared, sectionActor: "Actor A" },
      { item: shared, sectionActor: "Actor B" },
      { item: item("last"), sectionActor: "Actor B" },
    ]
    const wrapper = render(navigation)
    await open(wrapper, navigation[1]!)
    await button(wrapper, "curated.exportWatermarked").trigger("click")
    await flushPromises()
    expect(exportFrames).toHaveBeenCalledWith(expect.any(Array), ["shared"], "Actor B", "png", "watermarked")
    await wrapper.get('[aria-label="curated.nextFrame"]').trigger("click")
    await flushPromises()
    expect(wrapper.text()).toContain("Movie last")
  })

  it("previews motion and routes playback to the captured media time", async () => {
    const frame = item("motion")
    frame.row.motion = { status: "ready", artifactUrl: "motion.gif", contentType: "image/gif", durationSec: 2, width: 320, height: 180, fps: 12, fileSize: 100 }
    const wrapper = render(entries(frame))
    await open(wrapper)
    await button(wrapper, "curated.playMotion").trigger("click")
    expect(wrapper.find('img[src="motion.gif"]').exists()).toBe(true)
    await button(wrapper, "curated.playFromTime").trigger("click")
    await flushPromises()
    expect(push).toHaveBeenCalledWith(expect.objectContaining({ name: "player", params: { id: "movie-a" }, query: expect.objectContaining({ t: "65" }) }))
    expect(wrapper.find("[data-dialog]").exists()).toBe(false)
  })

  it("requires delete confirmation, preserves failed deletes, and reports successful removal", async () => {
    const wrapper = render()
    await open(wrapper)
    await button(wrapper, "curated.deleteThisFrame").trigger("click")
    expect(deleteFrame).not.toHaveBeenCalled()
    deleteFrame.mockRejectedValueOnce(new Error("offline"))
    await wrapper.get("[data-delete-confirm] button").trigger("click")
    await flushPromises()
    expect(wrapper.text()).toContain("curated.deleteFailed")
    expect(wrapper.find("[data-dialog]").exists()).toBe(true)
    await wrapper.get("[data-delete-confirm] button").trigger("click")
    await flushPromises()
    expect(wrapper.emitted("deleted")).toEqual([[["a"]]])
    expect(wrapper.find("[data-dialog]").exists()).toBe(false)
  })
})
