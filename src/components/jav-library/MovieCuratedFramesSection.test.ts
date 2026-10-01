import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import MovieCuratedFramesSection from "./MovieCuratedFramesSection.vue"
import { listCuratedFramesPage, type CuratedFrameDbRow, type CuratedFramePageResult } from "@/lib/curated-frames/db"
import { bumpCuratedFramesRevision } from "@/lib/curated-frames/revision"

vi.mock("@/lib/curated-frames/db", () => ({ listCuratedFramesPage: vi.fn() }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const list = vi.mocked(listCuratedFramesPage)
function row(id: string, movieId = "movie-a"): CuratedFrameDbRow {
  return { id, movieId, code: "CODE-A", title: "Movie A", actors: [], tags: [], positionSec: 65, capturedAt: "2026-10-01T00:00:00Z" }
}
function page(items: CuratedFrameDbRow[], total = items.length, nextCursor?: string): CuratedFramePageResult {
  return { items, total, nextCursor, limit: 12, offset: 0 }
}
const wrappers: ReturnType<typeof mount>[] = []
function render(movieId = "movie-a") {
  const wrapper = mount(MovieCuratedFramesSection, {
    props: { movieId },
    global: { stubs: {
      Dialog: { props: ["open"], template: '<div v-if="open"><slot /></div>' },
      DialogContent: { template: '<div><slot /></div>' },
      DialogTitle: { template: '<h2><slot /></h2>' },
      DialogDescription: { template: '<p><slot /></p>' },
      MediaStill: { props: ["src", "fit"], template: '<img :src="src" />' },
      FrameImageViewer: { props: ["src"], template: '<img data-original :src="src" />' },
    } },
  })
  wrappers.push(wrapper)
  return wrapper
}
beforeEach(() => list.mockReset())
afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.unstubAllGlobals()
})

describe("MovieCuratedFramesSection", () => {
  it("filters by movie, loads thumbnails, and opens only the selected original with navigation", async () => {
    list.mockResolvedValue(page([row("frame-a"), row("frame-b")]))
    const wrapper = render()
    await flushPromises()
    expect(list).toHaveBeenCalledWith(expect.objectContaining({ movieId: "movie-a", limit: 12, offset: 0 }))
    expect(wrapper.find("[data-original]").exists()).toBe(false)
    expect(wrapper.get('[data-movie-frame="frame-a"] img').attributes("src")).toContain("/frame-a/thumbnail")
    expect(wrapper.text()).toContain("01:05")
    await wrapper.get('[data-movie-frame="frame-a"]').trigger("click")
    expect(wrapper.get("[data-original]").attributes("src")).toContain("/frame-a/image")
    await wrapper.get('[aria-label="curated.nextFrame"]').trigger("click")
    expect(wrapper.get("[data-original]").attributes("src")).toContain("/frame-b/image")
  })

  it("retains loaded frames on pagination failure and retries the same cursor without duplicates", async () => {
    list.mockResolvedValueOnce(page([row("frame-a")], 3, "cursor-a"))
    const wrapper = render()
    await flushPromises()
    list.mockRejectedValueOnce(new Error("offline"))
    const more = wrapper.findAll("button").find((button) => button.text() === "curated.loadMore")!
    await more.trigger("click")
    await flushPromises()
    expect(wrapper.findAll("[data-movie-frame]")).toHaveLength(1)
    expect(wrapper.get('[role="alert"]').text()).toContain("detailPage.curatedFramesLoadError")
    list.mockResolvedValueOnce(page([row("frame-a"), row("frame-b"), row("frame-c")], -1))
    await wrapper.get('[role="alert"] button').trigger("click")
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ movieId: "movie-a", offset: 1, cursor: "cursor-a", skipTotal: true }))
    expect(wrapper.findAll("[data-movie-frame]")).toHaveLength(3)
    expect(wrapper.text()).not.toContain("curated.loadMore")
  })

  it("discards a late response from the previous movie", async () => {
    let resolveOld!: (value: CuratedFramePageResult) => void
    list.mockReturnValueOnce(new Promise((resolve) => { resolveOld = resolve }))
    const wrapper = render()
    list.mockResolvedValueOnce(page([row("frame-b", "movie-b")]))
    await wrapper.setProps({ movieId: "movie-b" })
    await flushPromises()
    resolveOld(page([row("frame-a")]))
    await flushPromises()
    expect(wrapper.find('[data-movie-frame="frame-a"]').exists()).toBe(false)
    expect(wrapper.find('[data-movie-frame="frame-b"]').exists()).toBe(true)
  })

  it("shows loading, retry and empty states independently", async () => {
    list.mockRejectedValueOnce(new Error("offline"))
    const wrapper = render()
    expect(wrapper.get('[role="status"]').text()).toBe("common.loading")
    await flushPromises()
    expect(wrapper.text()).not.toContain("detailPage.curatedFramesEmpty")
    list.mockResolvedValueOnce(page([]))
    await wrapper.get('[role="alert"] button').trigger("click")
    await flushPromises()
    expect(wrapper.text()).toContain("detailPage.curatedFramesEmpty")
  })

  it("releases Mock image URLs on refresh and unmount", async () => {
    const createObjectURL = vi.fn().mockReturnValueOnce("blob:first").mockReturnValueOnce("blob:second")
    const revokeObjectURL = vi.fn()
    vi.stubGlobal("URL", { createObjectURL, revokeObjectURL })
    list.mockResolvedValue(page([{ ...row("frame-a"), imageBlob: new Blob(["image"]) }]))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('[data-movie-frame] img').attributes("src")).toBe("blob:first")
    bumpCuratedFramesRevision()
    await flushPromises()
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:first")
    wrapper.unmount()
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:second")
  })
})
