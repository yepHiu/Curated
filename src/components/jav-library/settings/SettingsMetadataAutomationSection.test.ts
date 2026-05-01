import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsMetadataAutomationSection from "./SettingsMetadataAutomationSection.vue"

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

vi.mock("@/components/ui/switch", () => ({
  Switch: {
    name: "Switch",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template:
      "<button class=\"switch-stub\" @click=\"$emit('update:modelValue', !modelValue)\"><slot /></button>",
  },
}))

const baseProps = {
  useWebApi: true,
  providerPingAllBusy: false,
  providerPingOneName: null,
  providerHealthPingAllSummary: "",
  providerHealthPingError: "",
  autoLibraryWatch: false,
  autoLibraryWatchSaving: false,
  autoLibraryWatchError: "",
  autoActorProfileScrape: true,
  autoActorProfileScrapeSaving: false,
  autoActorProfileScrapeError: "",
}

describe("SettingsMetadataAutomationSection", () => {
  it("renders provider health and automation controls", () => {
    const wrapper = mount(SettingsMetadataAutomationSection, {
      props: {
        ...baseProps,
        providerHealthPingAllSummary: "2 providers ok",
        autoLibraryWatchError: "movie save failed",
      },
    })

    expect(wrapper.text()).toContain("settings.providerHealthTitle")
    expect(wrapper.text()).toContain("settings.autoScrape")
    expect(wrapper.text()).toContain("settings.autoActorProfileScrape")
    expect(wrapper.text()).toContain("2 providers ok")
    expect(wrapper.text()).toContain("movie save failed")
  })

  it("emits ping and switch changes", async () => {
    const wrapper = mount(SettingsMetadataAutomationSection, {
      props: baseProps,
    })

    await wrapper.get("[data-provider-ping-all]").trigger("click")
    const switches = wrapper.findAll(".switch-stub")
    await switches[0]?.trigger("click")
    await switches[1]?.trigger("click")

    expect(wrapper.emitted("pingAllProviders")).toHaveLength(1)
    expect(wrapper.emitted("changeAutoLibraryWatch")).toEqual([[true]])
    expect(wrapper.emitted("changeAutoActorProfileScrape")).toEqual([[false]])
  })

  it("renders mock hint outside Web API mode", () => {
    const wrapper = mount(SettingsMetadataAutomationSection, {
      props: {
        ...baseProps,
        useWebApi: false,
      },
    })

    expect(wrapper.text()).toContain("settings.providerHealthMockHint")
    expect(wrapper.find("[data-provider-ping-all]").exists()).toBe(false)
  })
})
