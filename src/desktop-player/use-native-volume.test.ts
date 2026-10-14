import { effectScope, ref } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import type { DesktopPlaybackSnapshot } from "../../electron/playback-contract"
import { emptyNativePlayerState } from "../../electron/native-player-contract"
import { useNativeVolume } from "./use-native-volume"

function fixture() {
  const snapshot = ref<DesktopPlaybackSnapshot>({ sessionId: "one", revision: 1, engine: "native", windowOpen: true,
    fullscreen: false, queue: [], autoAdvance: false, locale: "zh-CN",
    state: { ...emptyNativePlayerState(), status: "paused", volume: 30 } })
  const bridge = { command: vi.fn().mockResolvedValue(undefined), snapshot: vi.fn(async () => snapshot.value) }
  const receive = vi.fn((next: DesktopPlaybackSnapshot) => { snapshot.value = next })
  const error = vi.fn()
  const scope = effectScope()
  const volume = scope.run(() => useNativeVolume(bridge, snapshot, receive, error))!
  return { snapshot, bridge, receive, error, volume, scope }
}

describe("native volume dragging", () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it("updates immediately, ignores old engine samples while dragging and sends the trailing value", async () => {
    const f = fixture()
    f.volume.update([45])
    expect(f.volume.percent.value).toBe(45)
    f.snapshot.value = { ...f.snapshot.value, revision: 2, state: { ...f.snapshot.value.state, volume: 30 } }
    expect(f.volume.percent.value).toBe(45)
    await vi.advanceTimersByTimeAsync(100)
    f.volume.update([80])
    expect(f.volume.percent.value).toBe(80)
    await vi.advanceTimersByTimeAsync(119)
    expect(f.bridge.command).not.toHaveBeenCalled()
    f.bridge.snapshot.mockResolvedValueOnce({ ...f.snapshot.value, revision: 3, state: { ...f.snapshot.value.state, volume: 80 } })
    await vi.advanceTimersByTimeAsync(1)
    expect(f.bridge.command).toHaveBeenCalledExactlyOnceWith("one", { action: "volume", value: 80 })
    expect(f.volume.percent.value).toBe(80)
    f.scope.stop()
  })

  it("keeps a newer drag value when an earlier command finishes", async () => {
    const f = fixture()
    let complete!: () => void
    f.bridge.command.mockImplementationOnce(() => new Promise<void>(resolve => { complete = resolve }))
    f.volume.update([50])
    await vi.advanceTimersByTimeAsync(120)
    f.volume.update([85])
    complete()
    await vi.advanceTimersByTimeAsync(0)
    expect(f.volume.percent.value).toBe(85)
    f.scope.stop()
  })

  it("cancels pending commands on session change and disposal", async () => {
    const f = fixture()
    f.volume.update([80])
    f.snapshot.value = { ...f.snapshot.value, sessionId: "two", state: { ...f.snapshot.value.state, volume: 40 } }
    expect(f.volume.percent.value).toBe(40)
    await vi.advanceTimersByTimeAsync(120)
    expect(f.bridge.command).not.toHaveBeenCalled()
    f.volume.update([70])
    f.scope.stop()
    await vi.advanceTimersByTimeAsync(120)
    expect(f.bridge.command).not.toHaveBeenCalled()
  })

  it("restores the engine value and reports a failed volume command", async () => {
    const f = fixture()
    f.bridge.command.mockRejectedValueOnce(new Error("MPV_FAILED"))
    f.volume.update([80])
    await vi.advanceTimersByTimeAsync(120)
    expect(f.volume.percent.value).toBe(30)
    expect(f.error).toHaveBeenCalledOnce()
    f.scope.stop()
  })
})
