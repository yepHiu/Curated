import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsOrganizeSection from "./SettingsOrganizeSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  Layers: { name: "Layers", template: "<span />" },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<div><slot /></div>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<div><slot /></div>" },
  CardHeader: { name: "CardHeader", template: "<div><slot /></div>" },
  CardTitle: { name: "CardTitle", template: "<div><slot /></div>" },
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

describe("SettingsOrganizeSection", () => {
  it("renders organize settings and saving/error states", () => {
    const wrapper = mount(SettingsOrganizeSection, {
      props: {
        organizeLibrary: true,
        organizeLibrarySaving: true,
        organizeLibraryError: "save failed",
      },
    })

    expect(wrapper.text()).toContain("settings.organizeTitle")
    expect(wrapper.text()).toContain("settings.organizeSwitch")
    expect(wrapper.text()).toContain("settings.organizeSyncing")
    expect(wrapper.text()).toContain("save failed")
  })

  it("emits switch changes", async () => {
    const wrapper = mount(SettingsOrganizeSection, {
      props: {
        organizeLibrary: false,
        organizeLibrarySaving: false,
        organizeLibraryError: "",
      },
    })

    await wrapper.get(".switch-stub").trigger("click")

    expect(wrapper.emitted("changeOrganizeLibrary")).toEqual([[true]])
  })
})
