import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsMetadataModeSection from "./SettingsMetadataModeSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Info: { name: "Info", template: "<span />" },
}))

vi.mock("reka-ui", () => ({
  TooltipContent: { name: "TooltipContent", template: "<div><slot /></div>" },
  TooltipPortal: { name: "TooltipPortal", template: "<div><slot /></div>" },
  TooltipProvider: { name: "TooltipProvider", template: "<div><slot /></div>" },
  TooltipRoot: { name: "TooltipRoot", template: "<div><slot /></div>" },
  TooltipTrigger: { name: "TooltipTrigger", template: "<div><slot /></div>" },
}))

const baseProps = {
  metadataMovieModeUi: "auto" as const,
  metadataMovieSaving: false,
  metadataMovieChainSaving: false,
  providerPingAllBusy: false,
  canPickSpecifiedMetadata: true,
  canUseMetadataChainMode: true,
}

describe("SettingsMetadataModeSection", () => {
  it("renders provider mode choices", () => {
    const wrapper = mount(SettingsMetadataModeSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.metadataMovieProviderMode")
    expect(wrapper.text()).toContain("settings.metadataMovieProviderAuto")
    expect(wrapper.text()).toContain("settings.metadataMovieProviderSpecified")
    expect(wrapper.text()).toContain("settings.metadataMovieProviderChain")
    expect(wrapper.get('input[value="auto"]').attributes("checked")).toBeDefined()
  })

  it("emits mode changes", async () => {
    const wrapper = mount(SettingsMetadataModeSection, {
      props: baseProps,
    })

    await wrapper.get('input[value="specified"]').setValue(true)
    await wrapper.get('input[value="chain"]').setValue(true)
    await wrapper.get('input[value="auto"]').setValue(true)

    expect(wrapper.emitted("selectSpecified")).toHaveLength(1)
    expect(wrapper.emitted("selectChain")).toHaveLength(1)
    expect(wrapper.emitted("selectAuto")).toHaveLength(1)
  })

  it("disables unavailable modes", () => {
    const wrapper = mount(SettingsMetadataModeSection, {
      props: {
        ...baseProps,
        metadataMovieSaving: true,
        canPickSpecifiedMetadata: false,
        canUseMetadataChainMode: false,
      },
    })

    expect(wrapper.get('input[value="auto"]').attributes("disabled")).toBeDefined()
    expect(wrapper.get('input[value="specified"]').attributes("disabled")).toBeDefined()
    expect(wrapper.get('input[value="chain"]').attributes("disabled")).toBeDefined()
  })
})
