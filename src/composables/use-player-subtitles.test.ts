import { mount, flushPromises } from "@vue/test-utils"
import { defineComponent, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { usePlayerSubtitles } from "./use-player-subtitles"
import { readSubtitlePreferences, saveSubtitlePreferences } from "@/lib/subtitle-preferences"
import { defaultSubtitleDisplay } from "../../electron/subtitle-format"

const service = vi.hoisted(() => ({ assets: vi.fn(), content: vi.fn() }))
vi.mock("@/services/subtitle-service", () => ({ useSubtitleService: () => service }))
const subtitle = "1\n00:10:00,000 --> 00:10:02,000\n中文\n"
beforeEach(() => { localStorage.clear(); vi.clearAllMocks() })

describe("file-bound subtitles on the absolute video timeline", () => {
  it("uses absolute HLS time with positive delay and clears cues immediately on file changes", async () => {
    service.assets.mockResolvedValue([{ id: "v1", language: "zh-CN", format: "srt" }])
    service.content.mockResolvedValue(subtitle)
    const file = ref<string | undefined>("part-1"), time = ref(600.5)
    let player!: ReturnType<typeof usePlayerSubtitles>
    const wrapper = mount(defineComponent({ setup() { player = usePlayerSubtitles(ref("movie"), file, time); return () => null } }))
    await flushPromises()
    player.preferences.value = { ...defaultSubtitleDisplay, mode: "zh-CN", offset: 1 }
    expect(player.text.value).toBe("")
    time.value = 601.5
    expect(player.text.value).toBe("中文")
    service.assets.mockImplementation(() => new Promise(() => {}))
    file.value = "part-2"
    await flushPromises()
    expect(player.text.value).toBe("")
    expect(player.preferences.value.offset).toBe(0)
    wrapper.unmount()
  })
  it("ignores content arriving after a different file has loaded", async () => {
    let resolveOld!: (value: string) => void
    service.assets.mockImplementation((_movie: string, file: string) => Promise.resolve([{ id: file, language: "zh-CN", format: "srt" }]))
    service.content.mockImplementation((id: string) => id === "old" ? new Promise<string>(resolve => { resolveOld = resolve }) : Promise.resolve(subtitle.replace("中文", "新文件")))
    const file = ref<string | undefined>("old")
    let player!: ReturnType<typeof usePlayerSubtitles>
    const wrapper = mount(defineComponent({ setup() { player = usePlayerSubtitles(ref("movie"), file, ref(601)); return () => null } }))
    await flushPromises()
    file.value = "new"; await flushPromises()
    player.preferences.value.mode = "zh-CN"
    expect(player.text.value).toBe("新文件")
    resolveOld(subtitle); await flushPromises()
    expect(player.text.value).toBe("新文件")
    wrapper.unmount()
  })
  it("stores track delay per server/file/version and shares only appearance defaults", () => {
    saveSubtitlePreferences({ ...defaultSubtitleDisplay, offset: 2, size: 36 }, "server-a:file:v1")
    expect(readSubtitlePreferences("server-a:file:v1")).toMatchObject({ offset: 2, size: 36 })
    expect(readSubtitlePreferences("server-b:file:v1")).toMatchObject({ offset: 0, size: 36 })
    expect(readSubtitlePreferences("server-a:file:v2").offset).toBe(0)
  })
})
