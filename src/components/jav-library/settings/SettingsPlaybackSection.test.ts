import { flushPromises, shallowMount } from "@vue/test-utils"
import { ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { PatchPlayerSettingsBody, PlayerSettingsDTO } from "@/api/types"
import SettingsPlaybackSection from "./SettingsPlaybackSection.vue"

const { health, service } = vi.hoisted(() => {
  // 保留真实本机判断，仅替换 HTTP 和资料库服务。
  return { health: vi.fn(), service: vi.fn() }
})
vi.mock("@/api/endpoints", () => ({ api: { health } }))
vi.mock("@/services/library-service", () => ({ useLibraryService: service }))
vi.mock("vue-i18n", () => ({
  /** 使用稳定翻译键断言选项可见性。 */
  useI18n: () => ({ t: (key: string) => { /* 无翻译副作用。 */ return key } }),
}))

const playerSettings = ref<PlayerSettingsDTO>({} as PlayerSettingsDTO)
const patchPlayerSettings = vi.fn(async (patch: PatchPlayerSettingsBody) => {
  // 模拟服务端部分更新后的响应，未提交字段保持现值。
  playerSettings.value = { ...playerSettings.value, ...patch }
})
const advancedLabels = [
  "settings.hardwareDecode", "settings.playbackHardwareEncoder",
  "settings.playbackStreamPushEnabled", "settings.playbackForceStreamPush",
  "settings.playbackFfmpegCommand",
]
let wrapper: ReturnType<typeof render> | undefined

/** 使用真实保存逻辑与组件模型事件，避免依赖 UI 基元实现。 */
function render() {
  return shallowMount(SettingsPlaybackSection, {
    props: { autoSaveReady: true },
    global: { renderStubDefaultSlot: true },
  })
}

beforeEach(() => {
  // 每例从同一份已保存配置开始，避免去抖和客户端存储相互污染。
  vi.useFakeTimers()
  vi.stubEnv("VITE_USE_WEB_API", "true")
  vi.stubEnv("VITE_API_BASE_URL", "http://localhost:8080/api")
  delete window.javLibrary
  localStorage.clear()
  health.mockReset().mockResolvedValue({ canManageLibraryPaths: true })
  patchPlayerSettings.mockClear()
  playerSettings.value = {
    hardwareDecode: true, hardwareEncoder: "nvenc", streamPushEnabled: true,
    forceStreamPush: true, ffmpegCommand: "/server/bin/ffmpeg",
    nativePlayerEnabled: false, nativePlayerPreset: "custom", nativePlayerCommand: "/server/bin/player",
    preferNativePlayer: false, seekForwardStepSec: 10, seekBackwardStepSec: 10,
  }
  service.mockReturnValue({ playerSettings, patchPlayerSettings })
})

afterEach(() => {
  // 卸载组件以取消计时器，再恢复全局运行环境。
  wrapper?.unmount()
  vi.useRealTimers()
  vi.unstubAllEnvs()
  delete window.javLibrary
})

describe("playback settings visibility and saving", () => {
  // 覆盖共享页面的两种本机入口，确认探测前不会闪现高级选项。
  it.each(["web", "desktop"])("shows server controls only after verifying local %s", async (client) => {
    if (client === "desktop") window.javLibrary = { getDesktopInfo: vi.fn().mockResolvedValue({ serverOrigin: "http://localhost:8080" }) }
    wrapper = render()
    for (const label of advancedLabels) expect(wrapper.text()).not.toContain(label)
    await flushPromises()
    for (const label of advancedLabels) expect(wrapper.text()).toContain(label)
    const inputs = wrapper.findAllComponents({ name: "Input" })
    inputs[0]!.vm.$emit("update:modelValue", "/new/ffmpeg")
    await vi.advanceTimersByTimeAsync(600)
    expect(patchPlayerSettings).toHaveBeenCalledWith(expect.objectContaining({ ffmpegCommand: "/new/ffmpeg" }))
  })

  // 远端、旧服务端和不可确认状态均采用同一精简界面与保存边界。
  it.each(["web", "desktop", "old-server", "offline"])("hides infrastructure and omits it from %s saves", async (client) => {
    if (client === "web") vi.stubEnv("VITE_API_BASE_URL", "http://192.168.1.20:8081/api")
    if (client === "desktop") window.javLibrary = { getDesktopInfo: vi.fn().mockResolvedValue({ serverOrigin: "http://192.168.1.20:8081" }) }
    if (client === "old-server") health.mockResolvedValue({})
    if (client === "offline") health.mockRejectedValue(new Error("offline"))
    wrapper = render()
    await flushPromises()
    for (const label of advancedLabels) expect(wrapper.text()).not.toContain(label)
    expect(wrapper.text()).toContain("settings.playbackNativePlayerEnabled")
    expect(wrapper.text()).toContain("settings.playbackSeekForwardStep")

    // 模拟另一端修改底层配置；远端保存步长不能覆盖这些新值。
    playerSettings.value = { ...playerSettings.value, hardwareEncoder: "amf", ffmpegCommand: "/updated/ffmpeg" }
    const inputs = wrapper.findAllComponents({ name: "Input" })
    inputs[2]!.vm.$emit("update:modelValue", "30")
    await vi.advanceTimersByTimeAsync(600)
    expect(patchPlayerSettings).toHaveBeenCalledTimes(1)
    expect(patchPlayerSettings).toHaveBeenCalledWith({
      nativePlayerPreset: "custom", nativePlayerEnabled: false, preferNativePlayer: false,
      seekForwardStepSec: 30, seekBackwardStepSec: 10,
    })
    expect(playerSettings.value.ffmpegCommand).toBe("/updated/ffmpeg")
    expect(playerSettings.value.hardwareEncoder).toBe("amf")
    expect(playerSettings.value.nativePlayerCommand).toBe("/server/bin/player")
  })

  it("does not autosave normalized hidden fields when opening remote settings", async () => {
    // 历史配置可能保留互相冲突的推流字段，远端打开页面不应顺带纠正它们。
    health.mockResolvedValue({ canManageLibraryPaths: false })
    playerSettings.value.streamPushEnabled = false
    wrapper = render()
    await flushPromises()
    await vi.advanceTimersByTimeAsync(600)
    expect(patchPlayerSettings).not.toHaveBeenCalled()
    expect(playerSettings.value.forceStreamPush).toBe(true)
  })
})
