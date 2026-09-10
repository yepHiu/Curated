import { shallowMount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"
import SettingsExperimentalSection from "./SettingsExperimentalSection.vue"
describe("SettingsExperimentalSection", () => {
  it("shows media Beta sections without an AI migration card", () => {
    const wrapper = shallowMount(SettingsExperimentalSection, { global: { stubs: {
      SettingsComicLibrarySection: { template: '<div data-comic-beta />' },
      SettingsPhotoLibrarySection: { template: '<div data-photo-beta />' },
    } } })
    expect(wrapper.find('[data-comic-beta]').exists()).toBe(true)
    expect(wrapper.find('[data-photo-beta]').exists()).toBe(true)
    expect(wrapper.find('a').exists()).toBe(false)
    expect(wrapper.text()).not.toContain("aiSettings")
  })
})
