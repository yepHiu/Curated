import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsMetadataProviderSelectSection from "./SettingsMetadataProviderSelectSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Activity: { name: "Activity", template: "<span />" },
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { name: "Badge", template: "<span><slot /></span>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button :disabled=\"$attrs.disabled\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/select", () => {
  const Select = {
    name: "Select",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<div class=\"select-stub\" :data-model-value=\"modelValue\" :data-disabled=\"String(!!disabled)\"><slot /></div>",
  }
  return {
    Select,
    SelectContent: { name: "SelectContent", template: "<div><slot /></div>" },
    SelectItem: { name: "SelectItem", props: ["value"], template: "<div><slot /></div>" },
    SelectTrigger: { name: "SelectTrigger", template: "<div><slot /></div>" },
    SelectValue: { name: "SelectValue", template: "<div><slot /></div>" },
  }
})

const baseProps = {
  useWebApi: true,
  metadataMovieProvider: "",
  metadataMovieSelectOptions: ["javbus", "javdb"],
  metadataMovieSaving: false,
  providerPingAllBusy: false,
  providerPingOneName: null,
  currentProviderHealth: null,
}

describe("SettingsMetadataProviderSelectSection", () => {
  it("renders selected provider options", () => {
    const wrapper = mount(SettingsMetadataProviderSelectSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.metadataMovieProviderSelectLabel")
    expect(wrapper.text()).toContain("javbus")
    expect(wrapper.text()).toContain("javdb")
    expect(wrapper.get(".select-stub").attributes("data-model-value")).toBe("javbus")
  })

  it("emits provider select and ping actions", async () => {
    const wrapper = mount(SettingsMetadataProviderSelectSection, {
      props: baseProps,
    })

    wrapper.getComponent({ name: "Select" }).vm.$emit("update:modelValue", "javdb")
    await wrapper.get("[data-provider-ping-current]").trigger("click")

    expect(wrapper.emitted("selectProvider")).toEqual([["javdb"]])
    expect(wrapper.emitted("pingProvider")).toEqual([["javbus"]])
  })

  it("renders current provider health", () => {
    const wrapper = mount(SettingsMetadataProviderSelectSection, {
      props: {
        ...baseProps,
        metadataMovieProvider: "javdb",
        currentProviderHealth: {
          name: "javdb",
          status: "ok",
          latencyMs: 123,
          message: "healthy",
        },
      },
    })

    expect(wrapper.text()).toContain("settings.providerHealthStatusOk")
    expect(wrapper.text()).toContain("123ms")
    expect(wrapper.text()).toContain("healthy")
  })
})
