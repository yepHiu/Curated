import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import SettingsScopeBadge from "./SettingsScopeBadge.vue"

vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))
afterEach(() => { delete window.javLibrary })

describe("settings ownership labels", () => {
  it("identifies browser preferences without claiming they are Desktop settings", () => {
    const wrapper = mount(SettingsScopeBadge, { props: { scope: "client" } })
    expect(wrapper.text()).toBe("settings.scope.browser")
    expect(wrapper.get('[data-scope="client"]').attributes("title")).toBe("settings.scope.clientHint")
  })
  it("shows both server and Desktop ownership in mixed desktop sections", () => {
    window.javLibrary = { windowChrome: "macos" }
    const wrapper = mount(SettingsScopeBadge, { props: { scope: "mixed" } })
    expect(wrapper.findAll('[data-slot="badge"]').map(badge => badge.text())).toEqual([
      "settings.scope.server", "settings.scope.desktop",
    ])
  })
  it("keeps explicit server and desktop ownership independent of renderer context", async () => {
    const wrapper = mount(SettingsScopeBadge, { props: { scope: "desktop" } })
    expect(wrapper.text()).toBe("settings.scope.desktop")
    await wrapper.setProps({ scope: "server" })
    expect(wrapper.text()).toBe("settings.scope.server")
  })
})
