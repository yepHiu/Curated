import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsMetadataTriggerScrapeSection from "./SettingsMetadataTriggerScrapeSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  RefreshCw: { name: "RefreshCw", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button :disabled=\"$attrs.disabled\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

const baseProps = {
  busy: false,
  success: "",
  error: "",
}

describe("SettingsMetadataTriggerScrapeSection", () => {
  it("renders trigger scrape copy and emits run action", async () => {
    const wrapper = mount(SettingsMetadataTriggerScrapeSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.triggerScrape")
    expect(wrapper.text()).toContain("settings.triggerScrapeHint")
    expect(wrapper.text()).toContain("settings.triggerScrapeRunButton")

    await wrapper.get("[data-trigger-scrape-run]").trigger("click")

    expect(wrapper.emitted("run")).toHaveLength(1)
  })

  it("renders busy, success, and error states", () => {
    const wrapper = mount(SettingsMetadataTriggerScrapeSection, {
      props: {
        busy: true,
        success: "queued 2",
        error: "failed",
      },
    })

    expect(wrapper.get("[data-trigger-scrape-run]").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("settings.triggerScrapeRunning")
    expect(wrapper.text()).toContain("queued 2")
    expect(wrapper.text()).toContain("failed")
    expect(wrapper.get("[role='alert']").text()).toContain("failed")
  })
})
