import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsComicReaderSection from "./SettingsComicReaderSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  PanelsTopLeft: { name: "PanelsTopLeft", template: "<span />" },
}))

vi.mock("@/components/ui/select", () => ({
  Select: {
    name: "Select",
    props: ["modelValue", "disabled"],
    emits: ["update:modelValue"],
    template:
      "<div class='select-stub' :data-model-value='modelValue' :data-disabled='String(!!disabled)'><slot /></div>",
  },
  SelectContent: { name: "SelectContent", template: "<div><slot /></div>" },
  SelectItem: { name: "SelectItem", props: ["value"], template: "<div><slot /></div>" },
  SelectTrigger: { name: "SelectTrigger", template: "<div><slot /></div>" },
  SelectValue: {
    name: "SelectValue",
    props: ["placeholder"],
    template: "<span><slot>{{ placeholder }}</slot></span>",
  },
}))

const baseProps = {
  reader: {
    mode: "page" as const,
    fit: "contain" as const,
    direction: "ltr" as const,
  },
  saving: false,
  error: "",
}

describe("SettingsComicReaderSection", () => {
  it("lays out each reader default as a left label with the control on the right", () => {
    const wrapper = mount(SettingsComicReaderSection, {
      props: baseProps,
    })

    const list = wrapper.get("[data-comic-reader-settings-list]")
    expect(list.classes()).toContain("flex")
    expect(list.classes()).not.toContain("md:grid-cols-3")

    const fields = [
      ["mode", "settings.comicReaderMode", "page"],
      ["fit", "settings.comicReaderFit", "contain"],
      ["direction", "settings.comicReaderDirection", "ltr"],
    ] as const

    for (const [field, label, value] of fields) {
      const row = wrapper.get(`[data-comic-reader-setting-row="${field}"]`)
      const labelEl = row.get("[data-comic-reader-setting-label]")
      const trigger = row.get("[data-comic-reader-setting-trigger]")
      const select = row.get(".select-stub")

      expect(row.classes()).toEqual(
        expect.arrayContaining([
          "grid",
          "grid-cols-[minmax(4rem,1fr)_minmax(8rem,10rem)]",
          "items-center",
        ]),
      )
      expect(labelEl.text()).toBe(label)
      expect(trigger.classes()).toEqual(expect.arrayContaining(["h-9", "w-full"]))
      expect(select.attributes("data-model-value")).toBe(value)
    }
  })

  it("emits reader preference patches from each select", () => {
    const wrapper = mount(SettingsComicReaderSection, {
      props: baseProps,
    })
    const selects = wrapper.findAllComponents({ name: "Select" })

    selects[0].vm.$emit("update:modelValue", "scroll")
    selects[1].vm.$emit("update:modelValue", "width")
    selects[2].vm.$emit("update:modelValue", "rtl")

    expect(wrapper.emitted("patchReader")).toEqual([
      [{ mode: "scroll" }],
      [{ fit: "width" }],
      [{ direction: "rtl" }],
    ])
  })
})
