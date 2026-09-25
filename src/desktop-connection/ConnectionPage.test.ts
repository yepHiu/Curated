import { mount, flushPromises } from "@vue/test-utils"
import { createI18n } from "vue-i18n"
import { afterEach, expect, it, vi } from "vitest"
import ConnectionPage from "./ConnectionPage.vue"
import { connectionMessages } from "./messages"

// 主题系统另有独立覆盖，本测试聚焦连接状态和操作边界。
vi.mock("@/composables/use-theme", () => ({
  /** 提供稳定主题避免系统媒体查询影响状态测试。 */
  useTheme: () => ({ resolvedMode: "dark", setThemePreference: vi.fn() }),
}))
const wrappers: ReturnType<typeof mount>[] = []
// 每例释放发现定时器及模拟桥接。
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()); vi.unstubAllGlobals() })

/** 用真实 UI 基元和本地词典挂载，模拟 Electron 的受限连接桥。 */
function render(overrides: Record<string, unknown> = {}) {
  const api = {
    list: vi.fn().mockResolvedValue({ connections: [], desktopVersion: "1.0.0", suggestedUrl: "http://localhost:8081" }),
    discover: vi.fn().mockResolvedValue([]),
    connect: vi.fn().mockResolvedValue({ ok: false, error: "Server unavailable" }),
    cancel: vi.fn().mockResolvedValue(undefined),
    forget: vi.fn().mockResolvedValue(undefined),
    checkUpdate: vi.fn().mockResolvedValue(""),
    ...overrides,
  }
  Object.defineProperty(window, "curatedConnection", { configurable: true, value: api })
  const wrapper = mount(ConnectionPage, { global: { plugins: [createI18n({ legacy: false, locale: "zh-CN", messages: connectionMessages })] } })
  wrappers.push(wrapper)
  return { wrapper, api }
}

// 多播失败不能把手动地址表单标为无效，也不能禁用手动连接。
it("keeps discovery errors local and manual connection available", async () => {
  const { wrapper, api } = render({ discover: vi.fn().mockRejectedValue(new Error("Multicast unavailable")) })
  await flushPromises()
  expect(wrapper.get('[aria-labelledby="discovery-title"]').text()).toContain("Multicast unavailable")
  expect(wrapper.find("form [role=alert]").exists()).toBe(false)
  await wrapper.get("form").trigger("submit")
  await flushPromises()
  expect(api.connect).toHaveBeenCalledWith("http://localhost:8081")
  expect(wrapper.get("form [role=alert]").text()).toContain("Server unavailable")
  expect((wrapper.get("input").element as HTMLInputElement).value).toBe("http://localhost:8081")
})

// 当前连接不应出现可执行的忘记按钮，以免用户误判为普通历史档案。
it("marks the active connection and prevents forgetting it", async () => {
  const url = "http://localhost:8081"
  const { wrapper } = render({ list: vi.fn().mockResolvedValue({ connections: [{ url, name: "Home", serverId: "id" }], activeUrl: url, desktopVersion: "1.0.0" }) })
  await flushPromises()
  expect(wrapper.get('[aria-labelledby="recent-title"]').text()).toContain("当前连接")
  expect(wrapper.get('button[aria-label="忘记 Home"]').attributes("disabled")).toBeDefined()
})

// 连接过程中提供取消并防止重复提交，结束后恢复可编辑地址。
it("keeps cancellation available during a pending connection", async () => {
  let finish!: (value: { ok: boolean; error: string }) => void
  const pending = new Promise(resolve => { finish = resolve })
  const { wrapper, api } = render({ connect: vi.fn().mockReturnValue(pending) })
  await flushPromises()
  await wrapper.get("form").trigger("submit")
  expect(wrapper.get('button[type="submit"]').attributes("disabled")).toBeDefined()
  const cancel = wrapper.findAll("button").find(button => button.text() === "取消")!
  await cancel.trigger("click")
  expect(api.cancel).toHaveBeenCalledOnce()
  finish({ ok: false, error: "Cancelled" })
  await flushPromises()
  expect(wrapper.get("input").attributes("disabled")).toBeUndefined()
})
