import { flushPromises, mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import { ref } from "vue"

import SettingsDesktopPlayback from "./SettingsDesktopPlayback.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({ locale: ref("zh-CN"), t: (key: string) => key }),
}))

describe("desktop native playback availability", () => {
  it("explains a missing engine separately from a missing host", async () => {
    const capabilities = vi.fn()
      .mockResolvedValueOnce({ protocol: 1, available: false, reason: "engine", preferences: { preferNative: false, volume: 1, speed: 1 } })
    window.javLibrary = { playback: { capabilities, setPreferences: vi.fn() } as unknown as NonNullable<Window["javLibrary"]>["playback"] }
    const wrapper = mount(SettingsDesktopPlayback)
    await flushPromises()
    expect(wrapper.text()).toContain("desktopPlayback.unavailableEngine")
    capabilities.mockResolvedValueOnce({ protocol: 1, available: false, reason: "host", preferences: { preferNative: false, volume: 1, speed: 1 } })
    const host = mount(SettingsDesktopPlayback)
    await flushPromises()
    expect(host.text()).toContain("desktopPlayback.unavailableHost")
  })
})
