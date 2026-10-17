import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { HttpClientError } from "@/api/http-client"
import { i18n } from "@/i18n"
import BookOrganizationResults from "./BookOrganizationResults.vue"
import type { BookOrganizationJob, BookOrganizationItem } from "@/services/contracts/book-organization-service"

const api = vi.hoisted(() => ({ items: vi.fn(), review: vi.fn(), reloadComicsFromApi: vi.fn(), reloadPhotosFromApi: vi.fn() }))
vi.mock("@/services/book-organization-service", () => ({ useBookOrganizationService: () => api }))
vi.mock("@/services/comic-library-service", () => ({ useComicLibraryService: () => api }))
vi.mock("@/services/photo-library-service", () => ({ usePhotoLibraryService: () => api }))
const job: BookOrganizationJob = { id: "book-job", kind: "comic", locale: "zh-CN", status: "needs_confirmation", stage: "reviewing", total: 2, processed: 2, pendingReview: 2, applied: 0, failed: 0, createdAt: "", updatedAt: "" }
const rows: BookOrganizationItem[] = ["a", "b"].map(bookId => ({ bookId, title: `original ${bookId}`, status: "ready", revision: 2, before: { title: `original ${bookId}`, originalTitle: "原文", sourceFileName: "source.zip" }, metadata: { cleanTitle: "原文", translatedTitle: "译文" }, reason: "" }))
beforeEach(() => { vi.clearAllMocks(); i18n.global.locale.value = "zh-CN"; api.items.mockImplementation(async () => structuredClone(rows)); api.review.mockResolvedValue(undefined) })
function setup() { return mount(BookOrganizationResults, { props: { job: { ...job }, canWrite: true }, global: { plugins: [i18n] } }) }

it("only applies explicitly selected persisted revisions and refreshes the library", async () => {
  const wrapper = setup(); await flushPromises()
  expect(api.review).not.toHaveBeenCalled()
  await wrapper.findAll('[role="checkbox"]')[1]!.trigger("click")
  await wrapper.findAll("button").find(b => b.text() === "确认应用 1 本")!.trigger("click")
  await flushPromises()
  expect(api.review).toHaveBeenCalledExactlyOnceWith("book-job", { bookId: "b", revision: 2, decision: "apply" })
  expect(api.reloadComicsFromApi).toHaveBeenCalledOnce()
  expect(wrapper.emitted("updated")).toHaveLength(1)
  wrapper.unmount()
})

it("drops selection when a polled proposal revision changes", async () => {
  const wrapper = setup(); await flushPromises()
  await wrapper.findAll('[role="checkbox"]')[0]!.trigger("click")
  api.items.mockResolvedValue(rows.map(row => ({ ...row, revision: 3 })))
  await wrapper.setProps({ job: { ...job, updatedAt: "later" } }); await flushPromises()
  expect(wrapper.findAll("button").find(b => b.text() === "确认应用 0 本")!.attributes("disabled")).toBeDefined()
  expect(api.review).not.toHaveBeenCalled()
  wrapper.unmount()
})

it("reports partial batch failures and disables writes on read-only connections", async () => {
  const wrapper = setup(); await flushPromises()
  await wrapper.findAll("button").find(b => b.text() === "选择本页待确认项")!.trigger("click")
  api.review.mockRejectedValueOnce(new Error("AI_WRITE_CONFLICT"))
  await wrapper.findAll("button").find(b => b.text() === "确认应用 2 本")!.trigger("click"); await flushPromises()
  expect(api.review).toHaveBeenCalledTimes(2)
  expect(wrapper.text()).toContain("已处理 1 本，失败 1 本")
  await wrapper.setProps({ canWrite: false })
  expect(wrapper.findAll("button").find(b => b.text() === "选择本页待确认项")!.attributes("disabled")).toBeDefined()
  wrapper.unmount()
})

it.each([
  ["BOOK_METADATA_INVALID_RESULT", "角色或来源证据校验"],
  ["BOOK_METADATA_PROVIDER_FAILED", "AI 连接设置"],
  ["BOOK_METADATA_TRUNCATED", "模型输出被截断"],
  ["BOOK_RECOGNITION_FAILED", "识别未完成，请重试任务"],
])("shows the specific failure reason for %s", async (reason, message) => {
  api.items.mockResolvedValue([{ ...rows[0], status: "failed", metadata: {}, reason }])
  const wrapper = setup(); await flushPromises()
  expect(wrapper.text()).not.toContain(message)
  await wrapper.get("[data-processing-item]").trigger("click"); await flushPromises()
  await vi.waitFor(() => expect(wrapper.text()).toContain(message))
  if (reason !== "BOOK_METADATA_PROVIDER_FAILED") expect(wrapper.text()).not.toContain("检查 AI")
  wrapper.unmount()
})

afterEach(() => { vi.useRealTimers() })
const limited = () => new HttpClientError(429, { code: "AI_RATE_LIMITED", message: "write rate limit reached", retryable: true })
async function selectAndApply(wrapper: ReturnType<typeof setup>) {
  await wrapper.findAll("button").find(b => b.text() === "选择本页待确认项")!.trigger("click")
  await wrapper.findAll("button").find(b => b.text() === "确认应用 2 本")!.trigger("click")
  await flushPromises()
}

it("waits on write rate limits and retries the same revision before continuing the selection", async () => {
  const wrapper = setup(); await flushPromises(); vi.useFakeTimers()
  api.review.mockRejectedValueOnce(limited())
  await selectAndApply(wrapper)
  expect(wrapper.text()).toContain("60 秒后自动继续")
  expect(api.review).toHaveBeenCalledTimes(1)
  await vi.advanceTimersByTimeAsync(59000)
  expect(api.review).toHaveBeenCalledTimes(1)
  await vi.advanceTimersByTimeAsync(1000); await flushPromises()
  expect(api.review.mock.calls.map(call => call[1].bookId)).toEqual(["a", "a", "b"])
  expect(api.review.mock.calls[0]).toEqual(api.review.mock.calls[1])
  expect(wrapper.text()).toContain("已处理 2 本，失败 0 本")
  expect(wrapper.text()).not.toContain("原资料已变化")
  wrapper.unmount()
})

it.each(["stop", "close", "readonly", "switch"])("stops quota retries on %s and keeps unapplied results", async mode => {
  const wrapper = setup(); await flushPromises(); vi.useFakeTimers()
  api.review.mockRejectedValueOnce(limited())
  await selectAndApply(wrapper)
  if (mode === "stop") await wrapper.findAll("button").find(b => b.text() === "停止等待")!.trigger("click")
  else if (mode === "close") wrapper.unmount()
  else if (mode === "readonly") await wrapper.setProps({ canWrite: false })
  else await wrapper.setProps({ job: { ...job, id: "different" } })
  await vi.advanceTimersByTimeAsync(120000); await flushPromises()
  expect(api.review).toHaveBeenCalledTimes(1)
  if (mode === "stop") {
    expect(wrapper.text()).toContain("剩余项目保持待确认")
    expect(wrapper.findAll("button").some(b => b.text() === "确认应用 2 本")).toBe(true)
  }
  if (mode !== "close") wrapper.unmount()
})

it("pauses after bounded quota retries instead of looping indefinitely", async () => {
  const wrapper = setup(); await flushPromises(); vi.useFakeTimers()
  api.review.mockRejectedValue(limited())
  await selectAndApply(wrapper)
  await vi.advanceTimersByTimeAsync(180000); await flushPromises()
  expect(api.review).toHaveBeenCalledTimes(4)
  expect(wrapper.text()).toContain("剩余项目保持待确认")
  expect(wrapper.findAll("button").some(b => b.text() === "确认应用 2 本")).toBe(true)
  wrapper.unmount()
})

it.each(['comic', 'photo'] as const)("paginates compact %s items without mixing page selections", async kind => {
  const all = Array.from({ length: 51 }, (_, i) => ({ ...rows[0]!, bookId: `book-${i}`, title: `Book ${i}` }))
  api.items.mockImplementation(async (_id: string, offset = 0) => all.slice(offset, offset + 25))
  const wrapper = setup()
  await wrapper.setProps({ job: { ...job, kind, total: 51 } }); await flushPromises()
  expect(wrapper.findAll('[data-processing-item]')).toHaveLength(25)
  expect(wrapper.text()).not.toContain('译文')
  await wrapper.findAll('[data-processing-item]')[0]!.trigger('click'); await flushPromises()
  await vi.waitFor(() => expect(wrapper.get('[data-processing-item-detail]').text()).toContain('译文'))
  expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  expect(wrapper.findAll('[data-processing-item]')[0]!.attributes('aria-expanded')).toBe('true')
  await wrapper.findAll('[data-processing-item]')[0]!.trigger('click'); await flushPromises()
  expect(wrapper.findAll('[data-processing-item]')[0]!.attributes('aria-expanded')).toBe('false')
  expect(wrapper.text()).not.toContain('译文')
  await wrapper.findAll('[role="checkbox"]')[0]!.trigger('click')
  const next = () => wrapper.get('[data-items-pagination]').findAll('button')[1]!
  api.items.mockRejectedValueOnce(new Error('offline'))
  await next().trigger('click'); await flushPromises()
  expect(wrapper.get('[data-items-pagination]').text()).toContain('1 / 3')
  expect(wrapper.text()).toContain('确认应用 1 本')
  await next().trigger('click'); await flushPromises()
  expect(api.items).toHaveBeenLastCalledWith('book-job', 25)
  expect(wrapper.findAll('[data-processing-item]')[0]!.text()).toContain('Book 25')
  expect(wrapper.text()).toContain('确认应用 0 本')
  await wrapper.setProps({ job: { ...job, kind, total: 51, updatedAt: 'later' } }); await flushPromises()
  expect(api.items).toHaveBeenLastCalledWith('book-job', 25)
  await next().trigger('click'); await flushPromises()
  expect(wrapper.findAll('[data-processing-item]')).toHaveLength(1)
  expect(next().attributes('disabled')).toBeDefined()
  expect(api.review).not.toHaveBeenCalled()
  wrapper.unmount()
})

it("keeps the requested book page when polling overlaps navigation", async () => {
  const wrapper = setup(); await flushPromises()
  await wrapper.setProps({ job: { ...job, total: 51 } }); await flushPromises()
  let finish!: (items: BookOrganizationItem[]) => void
  api.items.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
  await wrapper.get('[data-items-pagination]').findAll('button')[1]!.trigger('click')
  const nextRows = [{ ...rows[0]!, bookId: 'next', title: 'Next page' }]
  api.items.mockResolvedValue(nextRows)
  await wrapper.setProps({ job: { ...job, total: 51, updatedAt: 'polled' } }); await flushPromises()
  expect(api.items).toHaveBeenLastCalledWith('book-job', 25)
  finish(rows); await flushPromises()
  expect(wrapper.get('[data-items-pagination]').text()).toContain('2 / 3')
  expect(wrapper.get('[data-processing-item]').text()).toContain('Next page')
  wrapper.unmount()
})
