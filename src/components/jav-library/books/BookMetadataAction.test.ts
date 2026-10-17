import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, expect, it, vi } from "vitest"
import { i18n } from "@/i18n"
import BookMetadataAction from "./BookMetadataAction.vue"

const api = vi.hoisted(() => ({ start: vi.fn() }))
vi.mock("@/services/book-organization-service", () => ({ useBookOrganizationService: () => api }))
vi.mock("@/lib/experimental-agent", () => ({ useExperimentalAgent: () => ({ writeEnabled: ref(true) }) }))
vi.mock("@/composables/use-server-local-access", () => ({ useServerLocalAccess: () => ({ isServerLocal: ref(true) }) }))
vi.mock("@/composables/use-app-toast", () => ({ pushAppToast: vi.fn() }))
vi.mock("@/components/ui/dialog", () => ({ Dialog: { props: ["open"], template: '<section v-if="open" role="dialog"><slot /></section>' }, ...Object.fromEntries(["DialogContent", "DialogHeader", "DialogTitle", "DialogDescription", "DialogFooter"].map(name => [name, { template: '<div><slot /></div>' }])) }))
beforeEach(() => { vi.clearAllMocks(); i18n.global.locale.value = "zh-CN" })
const books = [{ id: "a", title: "原文标题", sourceFileName: "source.zip" }]
function setup() { return mount(BookMetadataAction, { props: { kind: "comic", books }, global: { plugins: [i18n], stubs: { RouterLink: { template: '<a><slot /></a>' } } } }) }

it("opening or cancelling the scope dialog never submits work", async () => {
  const wrapper = setup()
  await wrapper.get("button").trigger("click")
  expect(wrapper.get('[role="dialog"]').text()).toContain("原文标题")
  expect(wrapper.text()).toContain("任务中心")
  await wrapper.findAll("button").find(b => b.text() === "取消")!.trigger("click")
  expect(api.start).not.toHaveBeenCalled()
  wrapper.unmount()
})

it("submits the frozen selection and reuses its request ID after a failed response", async () => {
  const wrapper = setup()
  api.start.mockRejectedValueOnce(new Error("connection lost")).mockResolvedValueOnce({ id: "job" })
  await wrapper.get("button").trigger("click")
  await wrapper.setProps({ books: [{ id: "b", title: "different", sourceFileName: "b.zip" }] })
  await wrapper.findAll("button").find(b => b.text() === "确认并开始")!.trigger("click"); await flushPromises()
  expect(wrapper.text()).toContain("任务提交失败")
  await wrapper.findAll("button").find(b => b.text() === "确认并开始")!.trigger("click"); await flushPromises()
  expect(api.start.mock.calls[0]![0]).toEqual(api.start.mock.calls[1]![0])
  expect(api.start).toHaveBeenLastCalledWith(expect.objectContaining({ bookIds: ["a"], kind: "comic", locale: "zh-CN" }))
  expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  expect(wrapper.text()).toContain("查看任务")
  wrapper.unmount()
})
