import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"
import { createI18n } from "vue-i18n"
import PlayerSubtitleMenu from "./PlayerSubtitleMenu.vue"
import { defaultSubtitleDisplay } from "../../../electron/subtitle-format"

function fixture() {
  return mount(PlayerSubtitleMenu, {
    props: { modelValue: { ...defaultSubtitleDisplay }, available: ["zh-CN", "ja"], native: true },
    global: {
      plugins: [createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false })],
      stubs: {
        Popover: { template: "<div><slot /></div>" }, PopoverContent: { template: "<div><slot /></div>" },
        PopoverTrigger: { template: "<div><slot /></div>" }, SelectContent: { template: "<div />" },
        SettingsHint: { template: "<span><slot /></span>" },
      },
    },
  })
}

describe("subtitle numeric editing", () => {
  it("allows intermediate digits and commits only the completed font size", async () => {
    const wrapper = fixture()
    const size = wrapper.get('input[min="16"]')
    await size.setValue("")
    await size.setValue("3")
    expect(wrapper.emitted("update:modelValue")).toBeUndefined()
    await size.setValue("32")
    await size.trigger("keydown", { key: "Enter" })
    expect(wrapper.emitted("update:modelValue")).toEqual([[{ ...defaultSubtitleDisplay, size: 32 }]])
    wrapper.unmount()
  })

  it("restores empty values, accepts negative offsets and clamps only on commit", async () => {
    const wrapper = fixture()
    const offset = wrapper.get('input[min="-60"]')
    await offset.setValue("")
    await offset.trigger("blur")
    expect(wrapper.emitted("update:modelValue")).toBeUndefined()
    expect((offset.element as HTMLInputElement).value).toBe("0")
    await offset.setValue("-1.5")
    await offset.trigger("blur")
    expect(wrapper.emitted("update:modelValue")?.at(-1)).toEqual([{ ...defaultSubtitleDisplay, offset: -1.5 }])
    await offset.setValue("100")
    await offset.trigger("blur")
    expect(wrapper.emitted("update:modelValue")?.at(-1)).toEqual([{ ...defaultSubtitleDisplay, offset: 60 }])
    wrapper.unmount()
  })
})
