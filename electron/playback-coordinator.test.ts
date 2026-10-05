import { describe, expect, it, vi } from "vitest"
import { NativePlaybackCoordinator, validatePlaybackOpen, type PlaybackContext, type PlaybackSurface } from "./playback-coordinator"
import { NativeMpvPlayer } from "./native-mpv-player"
import type { NativeLabControl } from "./native-player-contract"
import { defaultPlaybackPreferences } from "./playback-preferences"

class Player extends NativeMpvPlayer {
  stopped = false
  override async start(_exe: string, _url: string, start: number, _headless = false, initiallyPaused = false) {
    this.state = { ...this.state, status: initiallyPaused ? "paused" : "playing", positionSec: start, durationSec: 100 }
    this.emit("state", this.state)
  }
  override async control(input: NativeLabControl) {
    if (input.action === "pause") this.state.status = "paused"
    if (input.action === "resume") this.state.status = "playing"
    if (input.action === "seek") this.state.positionSec = input.value!
    this.emit("state", this.state)
  }
  override async stop() { this.stopped = true; if (this.state.status !== "ended") this.state.status = "stopped" }
}
function fixture() {
  const requests: { path: string; body?: string }[] = []
  const players: Player[] = []
  let disposed = 0
  let focused = 0
  let unlocked = true
  let authStatus = 200
  const context: PlaybackContext = { origin: "http://127.0.0.1:12345", generation: "one", fetch: async (url, init) => {
    const path = new URL(url).pathname + new URL(url).search
    requests.push({ path, body: init?.body as string | undefined })
    if (path === "/api/auth/status") return Response.json({ unlocked }, { status: authStatus })
    if (path.includes("/playback-session")) return Response.json({ mode: "direct", fileId: "p1", resumePositionSec: 30, durationSec: 100 })
    if (path.startsWith("/api/library/movies/")) return Response.json({ code: "TEST", title: "Fixture",
      files: [{ id: "p1", fileName: "1.mp4" }, { id: "p2", fileName: "2.mp4" }] })
    return new Response(null, { status: 204 })
  } }
  const surface: PlaybackSurface = { createPlayer: () => { const player = new Player(); players.push(player); return player },
    focus: async () => { focused++ }, action: async () => {}, fullscreen: () => false, dispose: async () => { disposed++ } }
  const coordinator = new NativePlaybackCoordinator("fixture", async () => surface, () => defaultPlaybackPreferences, () => {})
  return { coordinator, context, requests, players, disposed: () => disposed, focused: () => focused,
    lock: (status = 200) => { unlocked = false; authStatus = status } }
}
describe("production native coordinator", () => {
  it("uses direct resume, focuses existing playback, serializes replacement and rejects stale commands", async () => {
    const f = fixture()
    try {
      const first = await f.coordinator.open(f.context, { movieId: "a", autoplay: true, sourceQuery: { back: "browse", browse: "fc2", q: "original" } })
      expect(first.state).toMatchObject({ fileId: "p1", positionSec: 30 })
      await f.coordinator.open(f.context, { movieId: "a", autoplay: true, startSec: 7, sourceQuery: { q: "changed" } })
      expect(f.coordinator.snapshot()).toMatchObject({ state: { positionSec: 30 }, sourceQuery: { q: "original", browse: "fc2" } })
      expect(f.players).toHaveLength(1)
      expect(f.focused()).toBe(2)
      const second = await f.coordinator.open(f.context, { movieId: "a", fileId: "p2", startSec: 12, autoplay: false })
      expect(f.players[0]!.stopped).toBe(true)
      expect(second.sessionId).not.toBe(first.sessionId)
      expect(second.state).toMatchObject({ fileId: "p2", positionSec: 12, status: "paused" })
      await expect(f.coordinator.command(first.sessionId, { action: "seek", value: 40 })).rejects.toThrow("STALE")
      const saves = f.requests.filter(request => request.path.startsWith("/api/playback/progress/"))
      expect(saves[0]?.path).toContain("fileId=p1")
      expect(JSON.parse(saves[0]!.body!)).toMatchObject({ fileId: "p1", positionSec: 30 })
      expect(f.requests.filter(request => request.path.includes("playback-session")).every(request => request.body === '{"mode":"direct"}')).toBe(true)
      await f.coordinator.stop()
      expect(f.disposed()).toBe(1)
      expect(f.players.every(player => player.stopped)).toBe(true)
    } finally { await f.coordinator.stop() }
  })
  it("refuses locked and foreign file requests without replacing valid playback", async () => {
    const f = fixture()
    try {
      const original = await f.coordinator.open(f.context, { movieId: "a", autoplay: true })
      await expect(f.coordinator.open(f.context, { movieId: "a", fileId: "foreign", autoplay: true })).rejects.toThrow("INVALID_MOVIE_FILE")
      expect(f.coordinator.snapshot().sessionId).toBe(original.sessionId)
      f.lock()
      await expect(f.coordinator.open(f.context, { movieId: "b", autoplay: true })).rejects.toThrow("SERVER_LOCKED")
      expect(f.players[0]!.stopped).toBe(true)
    } finally { await f.coordinator.stop() }
  })
  it("advances parts only on natural EOF and keeps the frozen queue", async () => {
    const f = fixture()
    try {
      await f.coordinator.open(f.context, { movieId: "a", autoplay: true, queue: ["a", "b"], autoAdvance: true, sourceQuery: { back: "actor", actor: "Fixture" } })
      f.players[0]!.state.status = "ended"
      f.players[0]!.emit("state", f.players[0]!.state)
      await f.coordinator.run(async () => {})
      expect(f.coordinator.snapshot().state.fileId).toBe("p2")
      expect(f.coordinator.snapshot().sourceQuery).toEqual({ back: "actor", actor: "Fixture" })
      const current = f.coordinator.snapshot()
      await f.coordinator.command(current.sessionId, { action: "stop" })
      expect(f.players).toHaveLength(2)
      expect(f.coordinator.snapshot().windowOpen).toBe(false)
    } finally { await f.coordinator.stop() }
  })
  it.each([401, 403])("stops and closes the native surface when periodic authentication returns HTTP %i", async status => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval"] })
    const f = fixture()
    try {
      await f.coordinator.open(f.context, { movieId: "a", autoplay: true })
      f.lock(status)
      await vi.advanceTimersByTimeAsync(5000)
      await f.coordinator.run(async () => {})
      expect(f.coordinator.snapshot()).toMatchObject({ windowOpen: false, state: { status: "error", error: "SERVER_LOCKED" } })
      expect(f.players[0]!.stopped).toBe(true)
      expect(f.disposed()).toBe(1)
    } finally { await f.coordinator.stop(); vi.useRealTimers() }
  })
  it("releases an ended session if the next part's detail preflight fails", async () => {
    const f = fixture()
    try {
      await f.coordinator.open(f.context, { movieId: "a", autoplay: true, autoAdvance: true })
      const fetch = f.context.fetch
      f.context.fetch = async (url, init) => {
        if (new URL(url).pathname === "/api/library/movies/a") throw new Error("NETWORK_FAILED")
        return fetch(url, init)
      }
      f.players[0]!.state.status = "ended"
      f.players[0]!.emit("state", f.players[0]!.state)
      await f.coordinator.run(async () => {})
      expect(f.players[0]!.stopped).toBe(true)
      expect(f.coordinator.snapshot()).toMatchObject({ windowOpen: false, state: { status: "error", error: "NETWORK_FAILED" } })
    } finally { await f.coordinator.stop() }
  })
  it("releases native resources before emitting a Web handoff at the actual position", async () => {
    const f = fixture()
    let fallback: unknown
    f.coordinator.on("web-fallback", input => { expect(f.disposed()).toBe(1); fallback = input })
    const state = await f.coordinator.open(f.context, { movieId: "a", startSec: 42, autoplay: true })
    await f.coordinator.command(state.sessionId, { action: "web" })
    expect(fallback).toEqual({ movieId: "a", fileId: "p1", startSec: 42 })
  })
  it("validates identities, time, locale and bounded queues", () => {
    expect(() => validatePlaybackOpen({ movieId: "", autoplay: true })).toThrow()
    expect(() => validatePlaybackOpen({ movieId: "a", autoplay: true, startSec: NaN })).toThrow()
    expect(() => validatePlaybackOpen({ movieId: "a", autoplay: true, queue: [""] })).toThrow()
    expect(() => validatePlaybackOpen({ movieId: "a", autoplay: true, locale: "unknown" })).toThrow()
    expect(() => validatePlaybackOpen({ movieId: "a", autoplay: true, sourceQuery: { q: { invalid: true } } })).toThrow()
    expect(() => validatePlaybackOpen({ movieId: "a", autoplay: true, sourceQuery: { q: "x".repeat(2049) } })).toThrow()
    expect(validatePlaybackOpen({ movieId: "a", autoplay: true, sourceQuery: { back: "browse", url: "https://untrusted", engine: "web" } }).sourceQuery).toEqual({ back: "browse" })
  })
})
