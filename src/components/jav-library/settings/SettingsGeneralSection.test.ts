import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsGeneralSection from "./SettingsGeneralSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Languages: { name: "Languages", template: "<span />" },
  Power: { name: "Power", template: "<span />" },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<div><slot /></div>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<div><slot /></div>" },
  CardHeader: { name: "CardHeader", template: "<div><slot /></div>" },
  CardTitle: { name: "CardTitle", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/select", () => {
  const Select = {
    name: "Select",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<div class=\"select-stub\" :data-model-value=\"modelValue\"><slot /></div>",
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

vi.mock("./SettingsLoggingSection.vue", () => ({
  default: {
    name: "SettingsLoggingSection",
    props: ["autoSaveReady"],
    template: "<div data-logging :data-auto-save-ready=\"String(autoSaveReady)\">logging</div>",
  },
}))

const baseProps = {
  locale: "zh-CN",
  themePreference: "system" as const,
  launchAtLogin: false,
  launchAtLoginSaving: false,
  launchAtLoginDisabled: false,
  launchAtLoginUnavailableHint: "",
  launchAtLoginError: "",
  autoSaveReady: true,
}

describe("SettingsGeneralSection", () => {
  it("renders locale, appearance, launch-at-login and logging controls", () => {
    const wrapper = mount(SettingsGeneralSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.generalSubsectionLocaleAppearance")
    expect(wrapper.text()).toContain("settings.language")
    expect(wrapper.text()).toContain("settings.appearance")
    expect(wrapper.text()).toContain("settings.launchAtLoginTitle")
    expect(wrapper.get("[data-logging]").attributes("data-auto-save-ready")).toBe("true")
  })

  it("emits setting changes from controls", async () => {
    const wrapper = mount(SettingsGeneralSection, {
      props: baseProps,
    })

    const selects = wrapper.findAllComponents({ name: "Select" })
    selects[0]?.vm.$emit("update:modelValue", "en")
    selects[1]?.vm.$emit("update:modelValue", "dark")
    await wrapper.get(".switch-stub").trigger("click")

    expect(wrapper.emitted("update:locale")).toEqual([["en"]])
    expect(wrapper.emitted("changeTheme")).toEqual([["dark"]])
    expect(wrapper.emitted("changeLaunchAtLogin")).toEqual([[true]])
  })

  it("renders launch-at-login transient states", () => {
    const wrapper = mount(SettingsGeneralSection, {
      props: {
        ...baseProps,
        launchAtLoginSaving: true,
        launchAtLoginDisabled: true,
        launchAtLoginUnavailableHint: "unsupported",
        launchAtLoginError: "save failed",
      },
    })

    expect(wrapper.text()).toContain("settings.launchAtLoginSyncing")
    expect(wrapper.text()).not.toContain("unsupported")
    expect(wrapper.text()).toContain("save failed")
    expect(wrapper.get(".switch-stub").attributes("disabled")).toBeDefined()
  })
})
