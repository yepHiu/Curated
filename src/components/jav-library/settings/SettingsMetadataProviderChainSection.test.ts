import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsMetadataProviderChainSection from "./SettingsMetadataProviderChainSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params?.name ? `${key}:${params.name}` : key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Activity: { name: "Activity", template: "<span />" },
  GripVertical: { name: "GripVertical", template: "<span />" },
  Loader2: { name: "Loader2", template: "<span />" },
  Plus: { name: "Plus", template: "<span />" },
  X: { name: "X", template: "<span />" },
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
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template:
      "<div class=\"select-stub\" :data-model-value=\"modelValue\"><slot /></div>",
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
  canPickSpecifiedMetadata: true,
  providerChainDraft: ["javbus", "javdb"],
  availableProvidersForChain: ["mgstage"],
  selectedProviderToAdd: "",
  chainDragFromIndex: null,
  metadataMovieChainSaving: false,
  metadataMovieChainError: "",
  providerPingAllBusy: false,
  providerPingOneName: null,
  providerHealthByName: {
    javbus: {
      name: "javbus",
      status: "ok" as const,
      latencyMs: 123,
      message: "",
    },
  },
}

describe("SettingsMetadataProviderChainSection", () => {
  it("renders chain rows with health and available provider choices", () => {
    const wrapper = mount(SettingsMetadataProviderChainSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.metadataMovieProviderChainLabel")
    expect(wrapper.text()).toContain("settings.metadataMovieProviderChainDragHint")
    expect(wrapper.text()).toContain("javbus")
    expect(wrapper.text()).toContain("javdb")
    expect(wrapper.text()).toContain("123ms")
    expect(wrapper.text()).toContain("mgstage")
  })

  it("emits row, add, save, and select-provider actions", async () => {
    const wrapper = mount(SettingsMetadataProviderChainSection, {
      props: baseProps,
    })

    wrapper.getComponent({ name: "Select" }).vm.$emit("update:modelValue", "mgstage")
    await wrapper.setProps({ selectedProviderToAdd: "mgstage" })
    await wrapper.get("[data-provider-chain-add]").trigger("click")
    await wrapper.get("[data-provider-chain-save]").trigger("click")
    await wrapper.get("[data-provider-chain-ping='javbus']").trigger("click")
    await wrapper.get("[data-provider-chain-remove='javdb']").trigger("click")

    expect(wrapper.emitted("update:selectedProviderToAdd")).toEqual([["mgstage"]])
    expect(wrapper.emitted("addProvider")).toHaveLength(1)
    expect(wrapper.emitted("saveProviderChain")).toHaveLength(1)
    expect(wrapper.emitted("pingProvider")).toEqual([["javbus"]])
    expect(wrapper.emitted("removeProvider")).toEqual([[1]])
  })

  it("renders empty, warning, saving, and error states", () => {
    const wrapper = mount(SettingsMetadataProviderChainSection, {
      props: {
        ...baseProps,
        canPickSpecifiedMetadata: false,
        providerChainDraft: [],
        availableProvidersForChain: [],
        metadataMovieChainSaving: true,
        metadataMovieChainError: "save failed",
      },
    })

    expect(wrapper.text()).toContain("settings.metadataMovieProviderChainNoList")
    expect(wrapper.text()).toContain("settings.metadataMovieProviderChainEmpty")
    expect(wrapper.text()).toContain("settings.metadataMovieProviderSyncing")
    expect(wrapper.text()).toContain("save failed")
    expect(wrapper.find("[data-provider-chain-add]").exists()).toBe(false)
  })
})
