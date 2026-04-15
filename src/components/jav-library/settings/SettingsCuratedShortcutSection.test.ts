import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsCuratedShortcutSection from "./SettingsCuratedShortcutSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

describe("SettingsCuratedShortcutSection", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it("captures and saves a valid single-key shortcut", async () => {
    const wrapper = mount(SettingsCuratedShortcutSection)

    expect(wrapper.get("[data-curated-shortcut-current]").text()).toContain("C")

    await wrapper.get("[data-curated-shortcut-record]").trigger("click")
    window.dispatchEvent(new KeyboardEvent("keydown", { code: "KeyX", key: "x" }))
    await wrapper.vm.$nextTick()

    expect(wrapper.get("[data-curated-shortcut-current]").text()).toContain("X")
    expect(localStorage.getItem("jav-curated-capture-key-code")).toBe("KeyX")
  })

  it("captures and saves Page Down as a valid shortcut", async () => {
    const wrapper = mount(SettingsCuratedShortcutSection)

    await wrapper.get("[data-curated-shortcut-record]").trigger("click")
    window.dispatchEvent(new KeyboardEvent("keydown", { code: "PageDown", key: "PageDown" }))
    await wrapper.vm.$nextTick()

    expect(wrapper.get("[data-curated-shortcut-current]").text()).toContain("PageDown")
    expect(localStorage.getItem("jav-curated-capture-key-code")).toBe("PageDown")
  })

  it("rejects reserved keys and keeps the previous shortcut", async () => {
    const wrapper = mount(SettingsCuratedShortcutSection)

    await wrapper.get("[data-curated-shortcut-record]").trigger("click")
    window.dispatchEvent(new KeyboardEvent("keydown", { code: "ArrowUp", key: "ArrowUp" }))
    await wrapper.vm.$nextTick()

    expect(wrapper.get("[data-curated-shortcut-error]").text()).toContain(
      "settings.curatedShortcutReserved",
    )
    expect(wrapper.get("[data-curated-shortcut-current]").text()).toContain("C")
  })

  it("cancels capture mode when escape is pressed", async () => {
    const wrapper = mount(SettingsCuratedShortcutSection)

    await wrapper.get("[data-curated-shortcut-record]").trigger("click")
    expect(wrapper.get("[data-curated-shortcut-status]").text()).toContain(
      "settings.curatedShortcutListening",
    )

    window.dispatchEvent(new KeyboardEvent("keydown", { code: "Escape", key: "Escape" }))
    await wrapper.vm.$nextTick()

    expect(wrapper.get("[data-curated-shortcut-status]").text()).toContain(
      "settings.curatedShortcutIdle",
    )
    expect(wrapper.get("[data-curated-shortcut-current]").text()).toContain("C")
  })

  it("resets the shortcut back to the default key", async () => {
    localStorage.setItem("jav-curated-capture-key-code", "F8")
    const wrapper = mount(SettingsCuratedShortcutSection)

    expect(wrapper.get("[data-curated-shortcut-current]").text()).toContain("F8")

    await wrapper.get("[data-curated-shortcut-reset]").trigger("click")

    expect(wrapper.get("[data-curated-shortcut-current]").text()).toContain("C")
    expect(localStorage.getItem("jav-curated-capture-key-code")).toBeNull()
  })
})
