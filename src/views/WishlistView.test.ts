import { flushPromises, shallowMount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import WishlistView from "./WishlistView.vue"

const mocks = vi.hoisted(() => ({
  list: vi.fn(), replace: vi.fn(),
  route: { query: {} as Record<string, string | undefined>, fullPath: "/wishlist" },
}))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock("vue-router", () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ replace: mocks.replace, push: vi.fn() }),
}))
vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({ wishlist: { list: mocks.list } }),
}))
beforeEach(() => {
  vi.clearAllMocks()
  mocks.route.query = {}
  mocks.list.mockResolvedValue({ items: [], total: 0, pendingCount: 0 })
})
function render() {
  return shallowMount(WishlistView, { global: { renderStubDefaultSlot: true } })
}
describe("wishlist empty state", () => {
  it("checks the entire collection before showing the first-use message", async () => {
    const wrapper = render()
    expect(wrapper.findComponent({ name: "MediaEmptyState" }).exists()).toBe(false)
    await flushPromises()
    expect(mocks.list).toHaveBeenCalledWith({ status: "all", limit: 1 })
    expect(wrapper.findComponent({ name: "MediaEmptyState" }).props("filtered")).toBe(false)
    wrapper.unmount()
  })
  it("offers a way to reveal wishes in other states", async () => {
    mocks.list.mockResolvedValueOnce({ items: [], total: 0, pendingCount: 0 })
      .mockResolvedValueOnce({ items: [], total: 2, pendingCount: 0 })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.findComponent({ name: "MediaEmptyState" }).props("filtered")).toBe(true)
    const button = wrapper.findAllComponents({ name: "Button" }).find(item => item.text() === "bookBrowser.clearFilters")!
    button.vm.$emit("click")
    expect(mocks.replace).toHaveBeenCalledWith({ query: { q: undefined, cursor: undefined, status: "all" } })
    wrapper.unmount()
  })
  it.each([{ q: "missing" }, { status: "completed" }, { cursor: "next" }])("uses filtered copy for %j", async (query) => {
    mocks.route.query = query
    const wrapper = render()
    await flushPromises()
    expect(wrapper.findComponent({ name: "MediaEmptyState" }).props("filtered")).toBe(true)
    expect(mocks.list).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
  it("never presents a load failure as an empty collection", async () => {
    mocks.list.mockRejectedValue(new Error("offline"))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.findComponent({ name: "MediaEmptyState" }).exists()).toBe(false)
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    wrapper.unmount()
  })
})
