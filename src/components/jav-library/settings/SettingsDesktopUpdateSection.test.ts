import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"
import { createI18n } from "vue-i18n"
import zh from "@/locales/zh-CN.json"
import SettingsDesktopUpdateSection from "./SettingsDesktopUpdateSection.vue"

function render(props: InstanceType<typeof SettingsDesktopUpdateSection>["$props"]) {
  return mount(SettingsDesktopUpdateSection, { props, global: { plugins: [createI18n({ legacy: false, locale: "zh-CN", messages: { "zh-CN": zh } })] } })
}

describe("Desktop version and updates", () => {
  it("shows numeric version, development badge and result without a separate check button", () => {
    const wrapper = render({
      info: { version: "0.1.0", buildStamp: "20260926.123433", development: true, distribution: "legacy", platform: "macos", arch: "arm64" },
      infoError: false,
      result: { status: "development" },
    })
    expect(wrapper.get("[data-desktop-version]").text()).toBe("0.1.0")
    expect(wrapper.get("[data-desktop-build-stamp]").text()).toContain("20260926.123433")
    expect(wrapper.text()).toContain("开发版")
    expect(wrapper.text()).not.toContain("beta")
    expect(wrapper.find("[data-desktop-update-check]").exists()).toBe(false)
    expect(wrapper.get("[data-desktop-update-status]").text()).toContain("开发版不参与正式更新")
  })

  it("shows bridge failures without claiming up to date", () => {
    const wrapper = render({ info: null, infoError: true, result: { status: "error" } })
    expect(wrapper.text()).toContain("无法读取桌面版本")
    expect(wrapper.text()).toContain("无法检查 Desktop 更新")
    expect(wrapper.text()).not.toContain("已是最新")
  })
})
