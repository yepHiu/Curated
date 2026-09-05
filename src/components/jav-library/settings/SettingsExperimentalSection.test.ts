import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsExperimentalSection from "./SettingsExperimentalSection.vue"
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))
describe("SettingsExperimentalSection", () => {
  it("links to formal AI settings without a local-only enable switch", () => {
    const wrapper = mount(SettingsExperimentalSection, { props: { useWebApi: true }, global: { stubs: { RouterLink: { name: "RouterLink", props: ["to"], template: '<a><slot /></a>' } } } })
    expect(wrapper.text()).toContain("aiSettings.moved")
    expect(wrapper.find('[role="switch"]').exists()).toBe(false)
    expect(wrapper.findComponent({ name: "RouterLink" }).props("to")).toEqual({ name: "settings", query: { section: "ai" } })
  })
})
