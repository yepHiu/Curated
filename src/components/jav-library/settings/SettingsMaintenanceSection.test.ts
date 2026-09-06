import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsMaintenanceSection from "./SettingsMaintenanceSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/lib/pick-directory", () => ({
  pickLibraryDirectory: vi.fn(),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    backupDirectory: { value: "" },
    createBackup: vi.fn(),
    setBackupDirectory: vi.fn(),
    verifyBackup: vi.fn(),
    preflightBackupRestore: vi.fn(),
  }),
}))

describe("SettingsMaintenanceSection", () => {
  it("renders maintenance actions and emits full scan requests", async () => {
    const wrapper = mount(SettingsMaintenanceSection, {
      props: {
        fullScanBusy: false,
        backupSupported: false,
        healthSupported: false,
      },
    })

    expect(wrapper.text()).toContain("settings.triggerFullScan")
    expect(wrapper.text()).not.toContain("settings.configCardTitle")
    expect(wrapper.findAll('[data-slot="card"]')).toHaveLength(1)

    const maintenanceBlocks = wrapper.findAll("[data-settings-maintenance-block]")
    // The health block stays asynchronously loaded; the two synchronous
    // blocks lock the parent card composition while health has its own suite.
    expect(maintenanceBlocks).toHaveLength(2)
    expect(maintenanceBlocks.every((block) => block.classes().includes("p-4"))).toBe(true)

    await wrapper.get("[data-settings-full-scan]").trigger("click")

    expect(wrapper.emitted("runFullScan")).toHaveLength(1)
  })

  it("keeps primary maintenance actions comfortable on narrow screens", () => {
    const wrapper = mount(SettingsMaintenanceSection, {
      props: {
        fullScanBusy: false,
        backupSupported: false,
        healthSupported: false,
      },
    })

    const fullScan = wrapper.get("[data-settings-full-scan]")
    expect(fullScan.attributes("data-settings-comfortable-control")).toBeDefined()
    expect(fullScan.classes()).toContain("min-h-11")
    expect(fullScan.classes()).toContain("w-full")
    expect(fullScan.classes()).toContain("sm:w-auto")
  })

  it("disables full scan while a scan is busy", () => {
    const wrapper = mount(SettingsMaintenanceSection, {
      props: {
        fullScanBusy: true,
        backupSupported: false,
        healthSupported: false,
      },
    })

    expect(wrapper.get("[data-settings-full-scan]").attributes("disabled")).toBeDefined()
  })
})
