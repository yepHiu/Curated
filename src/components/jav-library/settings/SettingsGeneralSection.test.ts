import { setDevRemoteSimulation } from "@/lib/dev-remote-simulation"
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
  Card: { name: "Card", template: "<section data-card><slot /></section>" },
  CardContent: { name: "CardContent", template: "<div data-card-content><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<p data-card-description><slot /></p>" },
  CardHeader: { name: "CardHeader", template: "<header data-card-header><slot /></header>" },
  CardTitle: { name: "CardTitle", template: "<h3 data-card-title><slot /></h3>" },
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

const baseProps = {
  locale: "zh-CN",
  themePreference: "system" as const,
  launchAtLogin: false,
  launchAtLoginSaving: false,
  launchAtLoginDisabled: false,
  launchAtLoginError: "",
}

describe("SettingsGeneralSection", () => {
  it("keeps remote client controls without server administration", async () => {
    setDevRemoteSimulation(true)
    try {
      const wrapper = mount(SettingsGeneralSection, { props: baseProps })
      expect(wrapper.text()).toContain("settings.appearance")
      expect(wrapper.findComponent({ name: "Switch" }).exists()).toBe(false)
      wrapper.unmount()
    } finally { setDevRemoteSimulation(false) }
  })

  it("uses the settings card title and hint layout contract", () => {
    const wrapper = mount(SettingsGeneralSection, {
      props: baseProps,
    })

    const firstCard = wrapper.get("[data-card]")
    const firstHeader = wrapper.get("[data-card-header]")
    const firstTitle = wrapper.get("[data-card-title]")
    const firstContent = wrapper.get("[data-card-content]")

    expect(firstCard.classes()).toContain("gap-2")
    expect(firstHeader.classes()).toContain("pb-0")
    expect(firstHeader.classes()).toContain("gap-y-1")
    expect(firstHeader.classes()).toContain("grid-cols-[auto_minmax(0,1fr)]")
    expect(firstTitle.classes()).toEqual(
      expect.arrayContaining(["min-w-0", "text-lg", "tracking-tight"]),
    )
    expect(firstContent.classes()).toEqual(
      expect.arrayContaining(["flex", "flex-col", "gap-3", "pt-0"]),
    )
  })

  it("renders locale, appearance and launch-at-login controls", () => {
    const wrapper = mount(SettingsGeneralSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.generalSubsectionLocaleAppearance")
    expect(wrapper.text()).toContain("settings.language")
    expect(wrapper.text()).toContain("settings.appearance")
    expect(wrapper.text()).toContain("settings.launchAtLoginTitle")
    expect(wrapper.text()).not.toContain("settings.languageHint")
    expect(wrapper.text()).not.toContain("settings.appearanceHint")
    expect(wrapper.text()).not.toContain("settings.launchAtLoginDesc")
    expect(wrapper.text()).not.toContain("settings.launchAtLoginHint")
    const selectedValues = wrapper.findAllComponents({ name: "SelectValue" })
    expect(selectedValues[0]?.text()).toBe("settings.langZh")
    expect(selectedValues[1]?.text()).toBe("settings.themeSystem")
  })

  it("updates translated selected-value labels when locale and theme props change", async () => {
    const wrapper = mount(SettingsGeneralSection, { props: baseProps })

    await wrapper.setProps({ locale: "en", themePreference: "dark" })

    const selectedValues = wrapper.findAllComponents({ name: "SelectValue" })
    expect(selectedValues[0]?.text()).toBe("settings.langEn")
    expect(selectedValues[1]?.text()).toBe("settings.themeDark")
  })

  it("emits setting changes from controls", async () => {
    const wrapper = mount(SettingsGeneralSection, {
      props: baseProps,
    })

    const selects = wrapper.findAllComponents({ name: "Select" })
    selects[0]?.vm.$emit("update:modelValue", "en")
    selects[1]?.vm.$emit("update:modelValue", "dark")
    const switches = wrapper.findAll(".switch-stub")
    await switches[0]?.trigger("click")

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
        launchAtLoginError: "save failed",
      },
    })

    expect(wrapper.text()).toContain("settings.launchAtLoginSyncing")
    expect(wrapper.text()).toContain("save failed")
    expect(wrapper.findAll(".switch-stub")[0]?.attributes("disabled")).toBeDefined()
  })
})
