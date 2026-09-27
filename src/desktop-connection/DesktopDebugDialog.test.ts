import { flushPromises, mount } from "@vue/test-utils"
import { createI18n } from "vue-i18n"
import { afterEach, expect, it, vi } from "vitest"
import DesktopDebugDialog from "./DesktopDebugDialog.vue"
import { connectionMessages } from "./messages"

const wrappers: ReturnType<typeof mount>[] = []
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()); document.body.innerHTML = "" })

function render(probe = vi.fn().mockResolvedValue({ url: "http://localhost:8081", name: "Curated Server", version: "1.7.2", serverId: "id", legacy: false, latencyMs: 12 })) {
  const api = {
    readDebugInfo: vi.fn().mockResolvedValue({ version: "0.2.0", buildStamp: "20260927.143000", platform: "windows", arch: "x64", savedCount: 0, proxyMode: "system" }),
    probeDebugServer: probe,
    openDebugDevTools: vi.fn().mockResolvedValue(undefined),
  }
  const wrapper = mount(DesktopDebugDialog, { props: { open: true, api, initialAddress: "http://localhost:8081" }, attachTo: document.body,
    global: { plugins: [createI18n({ legacy: false, locale: "zh-CN", messages: connectionMessages })] } })
  wrappers.push(wrapper)
  return { wrapper, api }
}

it("shows local status without probing or changing a connection on open", async () => {
  const { api } = render()
  await flushPromises()
  expect(document.body.textContent).toContain("0.2.0")
  expect(document.body.textContent).toContain("当前服务器")
  expect(api.readDebugInfo).toHaveBeenCalledOnce()
  expect(api.probeDebugServer).not.toHaveBeenCalled()
})

it("probes a server without saving it and opens the local DevTools", async () => {
  const { api } = render()
  await flushPromises()
  const checks = [...document.querySelectorAll<HTMLButtonElement>("button")].find(button => button.textContent?.includes("检查"))!
  checks.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }))
  await flushPromises()
  document.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  await flushPromises()
  expect(api.probeDebugServer).toHaveBeenCalledWith("http://localhost:8081")
  expect(document.body.textContent).toContain("Curated Server 可用")
  const tools = [...document.querySelectorAll<HTMLButtonElement>("button")].find(button => button.textContent?.includes("打开 DevTools"))!
  tools.click()
  await flushPromises()
  expect(api.openDebugDevTools).toHaveBeenCalledOnce()
})

it("keeps the address and shows probe failures in the dialog", async () => {
  render(vi.fn().mockRejectedValue(new Error("Connection refused")))
  await flushPromises()
  const checks = [...document.querySelectorAll<HTMLButtonElement>("button")].find(button => button.textContent?.includes("检查"))!
  checks.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }))
  await flushPromises()
  document.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  await flushPromises()
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("Connection refused")
  expect(document.querySelector<HTMLInputElement>("#debug-server-address")?.value).toBe("http://localhost:8081")
})
