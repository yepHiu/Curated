import { mount, flushPromises } from "@vue/test-utils"
import { createI18n } from "vue-i18n"
import { afterEach, expect, it, vi } from "vitest"
import DesktopSettingsDialog from "./DesktopSettingsDialog.vue"
import { connectionMessages } from "./messages"

const wrappers: ReturnType<typeof mount>[] = []
// 每例释放真实 Dialog 的 portal 和焦点管理。
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()); document.body.innerHTML = "" })

/** 仅模拟 IPC；使用真实 Dialog/Input/Switch 检查用户操作。 */
function render(saveSettings = vi.fn().mockResolvedValue({ proxyMode: "manual", proxyUrl: "http://localhost:7890", launchAtLogin: false, loginSupported: true, restartRequired: true })) {
  const api = { readSettings: vi.fn().mockResolvedValue({ proxyMode: "manual", proxyUrl: "http://localhost:7890", launchAtLogin: false, loginSupported: true, restartRequired: false }), saveSettings }
  const wrapper = mount(DesktopSettingsDialog, { props: { open: true, api }, attachTo: document.body, global: { plugins: [createI18n({ legacy: false, locale: "zh-CN", messages: connectionMessages })] } })
  wrappers.push(wrapper)
  return { wrapper, api }
}

// IPC 返回失败时不能清除代理草稿或显示成功。
it("keeps edited proxy values when saving fails", async () => {
  const { api } = render(vi.fn().mockRejectedValue(new Error("Write denied")))
  await flushPromises()
  const input = document.querySelector<HTMLInputElement>("#desktop-proxy-url")!
  input.value = "socks5://localhost:1080"; input.dispatchEvent(new Event("input", { bubbles: true }))
  await flushPromises()
  document.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  await flushPromises()
  expect(api.saveSettings).toHaveBeenCalledWith({ proxyMode: "manual", proxyUrl: "socks5://localhost:1080", launchAtLogin: false })
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("Write denied")
  expect(input.value).toBe("socks5://localhost:1080")
})

// 代理为下次启动配置，保存反馈必须清楚说明重启要求。
it("shows restart feedback below the settings after saving", async () => {
  render()
  await flushPromises()
  document.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  await flushPromises()
  expect(document.querySelector('[role="status"]')?.textContent).toContain("重启 Desktop")
})
