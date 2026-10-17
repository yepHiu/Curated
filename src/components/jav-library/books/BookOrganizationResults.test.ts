import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, expect, it, vi } from "vitest"
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
