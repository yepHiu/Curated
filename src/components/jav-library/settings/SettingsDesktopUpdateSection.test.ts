import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import { createI18n } from "vue-i18n"
import zh from "@/locales/zh-CN.json"
import SettingsDesktopUpdateSection from "./SettingsDesktopUpdateSection.vue"

function render() {
  return mount(SettingsDesktopUpdateSection, { global: { plugins: [createI18n({ legacy: false, locale: "zh-CN", messages: { "zh-CN": zh } })] } })
}

afterEach(() => { delete window.javLibrary })

describe("Desktop version and updates", () => {
  it("hides Desktop in browsers, even with a spoofed URL version", () => {
    window.history.replaceState({}, "", "/?curatedDesktopVersion=99.0.0")
    expect(render().find("[data-desktop-update-section]").exists()).toBe(false)
    window.history.replaceState({}, "", "/")
  })

  it("shows the trusted numeric version and separate development badge; checks via the bridge", async () => {
    const check = vi.fn().mockResolvedValue({ status: "development" })
    window.javLibrary = {
      getDesktopInfo: vi.fn().mockResolvedValue({ version: "0.1.0", development: true, distribution: "legacy", platform: "macos", arch: "arm64" }),
      checkDesktopUpdate: check,
    }
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get("[data-desktop-version]").text()).toBe("0.1.0")
    expect(wrapper.text()).toContain("开发版")
    expect(wrapper.text()).not.toContain("beta")
    await wrapper.get("[data-desktop-update-check]").trigger("click")
    await flushPromises()
    expect(check).toHaveBeenCalledOnce()
    expect(wrapper.get("[data-desktop-update-status]").text()).toContain("开发版不参与正式更新")
  })

  it("shows bridge failures without claiming up to date", async () => {
    window.javLibrary = { getDesktopInfo: vi.fn().mockRejectedValue(new Error("closed")), checkDesktopUpdate: vi.fn().mockRejectedValue(new Error("closed")) }
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain("无法读取桌面版本")
    await wrapper.get("[data-desktop-update-check]").trigger("click")
    await flushPromises()
    expect(wrapper.text()).toContain("无法检查 Desktop 更新")
    expect(wrapper.text()).not.toContain("已是最新")
  })
})
