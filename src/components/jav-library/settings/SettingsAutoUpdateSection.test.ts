import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsAutoUpdateSection from "./SettingsAutoUpdateSection.vue"

vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe("SettingsAutoUpdateSection", () => {
  it("shows save state and emits a changed preference", async () => {
    const wrapper = mount(SettingsAutoUpdateSection, {
      props: { enabled: false, saving: true, error: "save failed" },
    })

    expect(wrapper.text()).toContain("settings.autoDownloadUpdatesTitle")
    expect(wrapper.text()).toContain("settings.autoDownloadUpdatesSyncing")
    expect(wrapper.get('[role="alert"]').text()).toBe("save failed")

    wrapper.findComponent({ name: "Switch" }).vm.$emit("update:modelValue", true)
    expect(wrapper.emitted("change")).toEqual([[true]])
  })
})
