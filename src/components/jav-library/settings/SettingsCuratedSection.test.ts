import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsCuratedSection from "./SettingsCuratedSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  FolderOpen: { name: "FolderOpen", template: "<span />" },
  ImageDown: { name: "ImageDown", template: "<span />" },
  Info: { name: "Info", template: "<span />" },
}))

vi.mock("reka-ui", () => ({
  TooltipContent: { name: "TooltipContent", template: "<div><slot /></div>" },
  TooltipPortal: { name: "TooltipPortal", template: "<div><slot /></div>" },
  TooltipProvider: { name: "TooltipProvider", template: "<div><slot /></div>" },
  TooltipRoot: { name: "TooltipRoot", template: "<div><slot /></div>" },
  TooltipTrigger: { name: "TooltipTrigger", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
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

vi.mock("./SettingsCuratedShortcutSection.vue", () => ({
  default: {
    name: "SettingsCuratedShortcutSection",
    template: "<div data-shortcut-section />",
  },
}))

const baseProps = {
  captureShortcutLabel: "C",
  curatedSaveMode: "app" as const,
  directorySupported: true,
  curatedFrameExportFormat: "jpg" as const,
  curatedExportFormatOptions: [
    { value: "jpg" as const, label: "JPG" },
    { value: "png" as const, label: "PNG" },
  ],
  curatedExportFormatSaving: false,
  curatedExportDirLabel: "",
  curatedExportPickBusy: false,
  curatedExportError: "",
  curatedExportFormatError: "",
}

describe("SettingsCuratedSection", () => {
  it("renders curated save policy, shortcut and export format controls", () => {
    const wrapper = mount(SettingsCuratedSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.curatedCardTitle")
    expect(wrapper.text()).toContain("C")
    expect(wrapper.text()).toContain("settings.savePolicy")
    expect(wrapper.find("[data-shortcut-section]").exists()).toBe(true)
    expect(wrapper.get(".select-stub").attributes("data-model-value")).toBe("jpg")
  })

  it("emits save mode and export format changes", async () => {
    const wrapper = mount(SettingsCuratedSection, {
      props: baseProps,
    })

    await wrapper.get('input[value="download"]').setValue(true)
    wrapper.getComponent({ name: "Select" }).vm.$emit("update:modelValue", "png")

    expect(wrapper.emitted("update:curatedSaveMode")).toEqual([["download"]])
    expect(wrapper.emitted("changeExportFormat")).toEqual([["png"]])
  })

  it("renders export directory controls and emits actions", async () => {
    const wrapper = mount(SettingsCuratedSection, {
      props: {
        ...baseProps,
        curatedSaveMode: "directory",
        curatedExportDirLabel: "Frames",
        curatedExportError: "pick failed",
        curatedExportFormatError: "format failed",
      },
    })

    expect(wrapper.text()).toContain('settings.exportChosen:{"name":"Frames"}')
    expect(wrapper.text()).toContain("pick failed")
    expect(wrapper.text()).toContain("format failed")

    await wrapper.get("[data-curated-pick-directory]").trigger("click")
    await wrapper.get("[data-curated-clear-directory]").trigger("click")

    expect(wrapper.emitted("pickExportDirectory")).toHaveLength(1)
    expect(wrapper.emitted("clearExportDirectory")).toHaveLength(1)
  })

  it("shows unsupported directory save mode state", () => {
    const wrapper = mount(SettingsCuratedSection, {
      props: {
        ...baseProps,
        directorySupported: false,
      },
    })

    expect(wrapper.get('input[value="directory"]').attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("settings.curatedDirUnsupported")
  })
})
