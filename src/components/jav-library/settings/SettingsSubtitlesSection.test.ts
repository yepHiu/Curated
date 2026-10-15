import { mount, flushPromises } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import { Switch } from "@/components/ui/switch"
import SettingsSubtitlesSection from "./SettingsSubtitlesSection.vue"

const service = vi.hoisted(() => ({ settings: vi.fn(), saveSettings: vi.fn(), testProvider: vi.fn() }))
vi.mock("@/services/subtitle-service", () => ({ useSubtitleService: () => service }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe("subtitle provider enable visibility", () => {
  it("hides dependent controls while retaining drafts and saved configuration", async () => {
    const settings = { enabled: true, canManage: true, apiKeyConfigured: true, recognitionModel: "qwen3-asr-flash-filetrans", translationModel: "qwen-plus", configuredModel: "custom" }
    service.settings.mockResolvedValue({ ...settings })
    service.saveSettings.mockImplementation(async patch => ({ ...settings, ...patch }))
    const wrapper = mount(SettingsSubtitlesSection)
    await flushPromises()
    await wrapper.get("#subtitle-key").setValue("replacement-key")
    wrapper.getComponent(Switch).vm.$emit("update:modelValue", false)
    await flushPromises()
    expect(wrapper.find("#subtitle-key").exists()).toBe(false)
    expect(wrapper.find("#subtitle-recognition-model").exists()).toBe(false)
    expect(wrapper.find("#subtitle-model").exists()).toBe(false)
    expect(wrapper.text()).not.toContain("subtitles.testProvider")
    wrapper.getComponent(Switch).vm.$emit("update:modelValue", true)
    await flushPromises()
    expect((wrapper.get("#subtitle-key").element as HTMLInputElement).value).toBe("replacement-key")
    await wrapper.get("#subtitle-key").setValue("")
    wrapper.getComponent(Switch).vm.$emit("update:modelValue", false)
    await flushPromises()
    expect(service.saveSettings).toHaveBeenCalledWith({ enabled: false, recognitionModel: "qwen3-asr-flash-filetrans", translationModel: "qwen-plus" })
    expect(service.saveSettings.mock.calls.at(-1)![0]).not.toHaveProperty("apiKey")
    wrapper.unmount()
  })
  it("saves an entered key before testing without a manual save button", async () => {
    service.saveSettings.mockClear()
    service.settings.mockResolvedValue({ enabled:true, canManage:true, apiKeyConfigured:false, recognitionModel:"qwen3-asr-flash-filetrans", translationModel:"qwen-flash", configuredModel:"" })
    service.saveSettings.mockImplementation(async patch => ({ ...patch, canManage:true, apiKeyConfigured:true }))
    service.testProvider.mockResolvedValue(undefined)
    const wrapper=mount(SettingsSubtitlesSection)
    await flushPromises()
    expect(service.saveSettings).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain("subtitles.save")
    await wrapper.get("#subtitle-key").setValue("new-key")
    await wrapper.findAll("button").find(button=>button.text()==="subtitles.testProvider")!.trigger("click")
    await flushPromises()
    expect(service.saveSettings).toHaveBeenCalledWith(expect.objectContaining({apiKey:"new-key"}))
    expect(service.testProvider).toHaveBeenCalledOnce()
    expect(service.saveSettings.mock.invocationCallOrder[0]).toBeLessThan(service.testProvider.mock.invocationCallOrder[0]!)
    wrapper.unmount()
  })

})
