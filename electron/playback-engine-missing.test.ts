import { describe, expect, it, vi } from "vitest"
import { NativePlaybackCoordinator } from "./playback-coordinator"
import { defaultPlaybackPreferences } from "./playback-preferences"

describe("missing native engine", () => {
  it("stops before the server or player window is touched", async () => {
    const fetch = vi.fn()
    const createSurface = vi.fn()
    const coordinator = new NativePlaybackCoordinator("", createSurface, () => defaultPlaybackPreferences, () => {})
    await expect(coordinator.open(
      { origin: "http://127.0.0.1:8080", generation: "generation", fetch },
      { movieId: "movie", autoplay: true },
    )).rejects.toThrow("NATIVE_ENGINE_MISSING")
    expect(fetch).not.toHaveBeenCalled()
    expect(createSurface).not.toHaveBeenCalled()
    expect(coordinator.snapshot().state).toMatchObject({ status: "error", error: "NATIVE_ENGINE_MISSING" })
  })
})
