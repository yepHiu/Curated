import { computed } from "vue"
import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsNetworkSection from "./SettingsNetworkSection.vue"

const {
  lanEnabledState,
  lanListeningState,
  lanAccessUrlsState,
  setLANEnabled,
} = vi.hoisted(() => ({
  lanEnabledState: { value: false },
  lanListeningState: { value: false },
  lanAccessUrlsState: { value: [] as string[] },
  setLANEnabled: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  ChevronDown: { name: "ChevronDown", template: "<span />" },
  Globe: { name: "Globe", template: "<span />" },
  Loader2: { name: "Loader2", template: "<span />" },
  Network: { name: "Network", template: "<span />" },
}))

vi.mock("@/composables/use-settings-scroll-preserve", () => ({
  useSettingsScrollPreserve: () => ({
    withPreservedScroll: async <T>(fn: () => Promise<T> | T) => await fn(),
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    lanEnabled: computed(() => lanEnabledState.value),
    lanListening: computed(() => lanListeningState.value),
    lanAccessUrls: computed(() => lanAccessUrlsState.value),
    setLANEnabled,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button :disabled=\"$attrs.disabled\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<div><slot /></div>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<div><slot /></div>" },
  CardHeader: { name: "CardHeader", template: "<div><slot /></div>" },
  CardTitle: { name: "CardTitle", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<input v-bind=\"$attrs\" :value=\"modelValue\" :disabled=\"disabled\" @input=\"$emit('update:modelValue', $event.target.value)\" />",
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

vi.mock("@/components/ui/switch", () => ({
  Switch: {
    name: "Switch",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<button class=\"switch-stub\" :disabled=\"disabled\" @click=\"$emit('update:modelValue', !modelValue)\"><slot /></button>",
  },
}))

const baseProps = {
  useWebApi: true,
  proxyEnabled: true,
  proxyScheme: "http" as const,
  proxyHost: "127.0.0.1",
  proxyPort: "7890",
  proxyUsername: "",
  proxyPassword: "",
  proxyAuthExpanded: false,
  proxySaving: false,
  proxyOutboundPingBusy: false,
  proxyJavbusBusy: false,
  proxyGoogleBusy: false,
  proxyJavbusStatusMessage: null,
  proxyGoogleStatusMessage: null,
  proxyStatusMessage: null,
}

describe("SettingsNetworkSection", () => {
  beforeEach(() => {
    lanEnabledState.value = false
    lanListeningState.value = false
    lanAccessUrlsState.value = []
    setLANEnabled.mockReset()
  })

  it("renders proxy form and web api ping actions", () => {
    const wrapper = mount(SettingsNetworkSection, {
      props: {
        ...baseProps,
        proxyStatusMessage: {
          text: "connected",
          className: "text-success",
        },
      },
    })

    expect(wrapper.text()).toContain("settings.proxyTitle")
    expect(wrapper.text()).toContain("settings.proxyEnabled")
    expect(wrapper.text()).toContain("settings.proxyPingJavbus")
    expect(wrapper.text()).toContain("settings.proxyPingGoogle")
    expect(wrapper.text()).toContain("connected")
    expect(wrapper.get(".select-stub").attributes("data-model-value")).toBe("http")
  })

  it("emits field updates", async () => {
    const wrapper = mount(SettingsNetworkSection, {
      props: baseProps,
    })

    await wrapper.get("[data-proxy-enabled]").trigger("click")
    wrapper.getComponent({ name: "Select" }).vm.$emit("update:modelValue", "socks5")
    await wrapper.get("[data-proxy-host]").setValue("10.0.0.1")
    await wrapper.get("[data-proxy-port]").setValue("7897")
    await wrapper.get("[data-proxy-auth-toggle]").trigger("click")

    expect(wrapper.emitted("update:proxyEnabled")).toEqual([[false]])
    expect(wrapper.emitted("update:proxyScheme")).toEqual([["socks5"]])
    expect(wrapper.emitted("update:proxyHost")).toEqual([["10.0.0.1"]])
    expect(wrapper.emitted("update:proxyPort")).toEqual([["7897"]])
    expect(wrapper.emitted("update:proxyAuthExpanded")).toEqual([[true]])
  })

  it("emits save and ping actions", async () => {
    const wrapper = mount(SettingsNetworkSection, {
      props: baseProps,
    })

    await wrapper.get("[data-proxy-save]").trigger("click")
    await wrapper.get("[data-proxy-javbus]").trigger("click")
    await wrapper.get("[data-proxy-google]").trigger("click")

    expect(wrapper.emitted("saveProxy")).toHaveLength(1)
    expect(wrapper.emitted("testProxyJavbus")).toHaveLength(1)
    expect(wrapper.emitted("testProxyGoogle")).toHaveLength(1)
  })

  it("renders mock hint and hides ping actions outside Web API mode", () => {
    const wrapper = mount(SettingsNetworkSection, {
      props: {
        ...baseProps,
        useWebApi: false,
      },
    })

    expect(wrapper.text()).toContain("settings.proxyMockHint")
    expect(wrapper.text()).toContain("settings.lanAccessMockHint")
    expect(wrapper.get("[data-lan-access-switch]").attributes("disabled")).toBeDefined()
    expect(wrapper.find("[data-proxy-javbus]").exists()).toBe(false)
    expect(wrapper.find("[data-proxy-google]").exists()).toBe(false)
  })

  it("saves LAN access without requiring PIN", async () => {
    setLANEnabled.mockResolvedValue(undefined)
    const wrapper = mount(SettingsNetworkSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.lanAccessTitle")
    expect(wrapper.text()).not.toContain("settings.lanAccessPinRequired")
    expect(wrapper.find("[data-lan-access-open-security]").exists()).toBe(false)
    expect(wrapper.get("[data-lan-access-switch]").attributes("disabled")).toBeUndefined()

    await wrapper.get("[data-lan-access-switch]").trigger("click")
    expect(setLANEnabled).toHaveBeenCalledWith(true)
  })

  it.each([
    { enabled: true, listening: false, restartRequired: true },
    { enabled: false, listening: true, restartRequired: true },
    { enabled: true, listening: true, restartRequired: false },
    { enabled: false, listening: false, restartRequired: false },
  ])("shows restart hint=$restartRequired for LAN enabled=$enabled, listening=$listening", ({
    enabled,
    listening,
    restartRequired,
  }) => {
    lanEnabledState.value = enabled
    lanListeningState.value = listening
    const wrapper = mount(SettingsNetworkSection, { props: baseProps })

    expect(wrapper.text().includes("settings.lanAccessRestartHint")).toBe(restartRequired)
  })

  it("hides the restart hint in Mock mode even if LAN states differ", () => {
    lanEnabledState.value = false
    lanListeningState.value = true
    const wrapper = mount(SettingsNetworkSection, {
      props: { ...baseProps, useWebApi: false },
    })

    expect(wrapper.text()).not.toContain("settings.lanAccessRestartHint")
  })

  it("shows a full-quit hint when LAN access is saved but not yet listening", () => {
    lanEnabledState.value = true
    lanListeningState.value = false
    lanAccessUrlsState.value = ["http://192.168.1.8:8080"]
    const wrapper = mount(SettingsNetworkSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.lanAccessRestartHint")
    expect(wrapper.text()).toContain("http://192.168.1.8:8080")
  })
})
