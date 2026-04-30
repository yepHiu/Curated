import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsMaintenanceSection from "./SettingsMaintenanceSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("SettingsMaintenanceSection", () => {
  it("renders maintenance actions and emits full scan requests", async () => {
    const wrapper = mount(SettingsMaintenanceSection, {
      props: {
        fullScanBusy: false,
      },
    })

    expect(wrapper.text()).toContain("settings.manualCardTitle")
    expect(wrapper.text()).toContain("settings.configCardTitle")

    await wrapper.get("[data-settings-full-scan]").trigger("click")

    expect(wrapper.emitted("runFullScan")).toHaveLength(1)
  })

  it("disables full scan while a scan is busy", () => {
    const wrapper = mount(SettingsMaintenanceSection, {
      props: {
        fullScanBusy: true,
      },
    })

    expect(wrapper.get("[data-settings-full-scan]").attributes("disabled")).toBeDefined()
  })
})
