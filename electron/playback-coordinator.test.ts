import { describe, expect, it, vi } from "vitest"
import { NativePlaybackCoordinator, validatePlaybackOpen, type PlaybackContext, type PlaybackSurface } from "./playback-coordinator"
import { NativeMpvPlayer } from "./native-mpv-player"
import type { NativeLabControl } from "./native-player-contract"
import { defaultPlaybackPreferences } from "./playback-preferences"

class Player extends NativeMpvPlayer {
  stopped = false
  captures = 0
  override async captureFrame() {
    this.captures++
    return { image: Buffer.from("fixture png"), positionSec: this.state.positionSec + 0.125, capturedAt: "2026-10-06T01:02:03.000Z" }
  }
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
  let captureStatus = 204
  const uploads: { metadata: Record<string, unknown>; image: Buffer; headers: Headers }[] = []
  const context: PlaybackContext = { origin: "http://127.0.0.1:12345", generation: "one", fetch: async (url, init) => {
    const path = new URL(url).pathname + new URL(url).search
    requests.push({ path, body: init?.body as string | undefined })
    if (path === "/api/auth/status") return Response.json({ unlocked }, { status: authStatus })
    if (path === "/api/curated-frames") {
      const form = init!.body as FormData
      const image = form.get("image") as Blob
      uploads.push({ metadata: JSON.parse(form.get("metadata") as string), image: Buffer.from(await image.arrayBuffer()), headers: new Headers(init?.headers) })
      return new Response(null, { status: captureStatus })
    }
    if (path.includes("/playback-session")) return Response.json({ mode: "direct", fileId: "p1", resumePositionSec: 30, durationSec: 100 })
    if (path.startsWith("/api/library/movies/")) return Response.json({ code: "TEST", title: "Fixture", actors: ["Fixture actor"],
      files: [{ id: "p1", fileName: "1.mp4" }, { id: "p2", fileName: "2.mp4" }] })
    return new Response(null, { status: 204 })
  } }
  const surface: PlaybackSurface = { createPlayer: () => { const player = new Player(); players.push(player); return player },
    setTitle: async () => {}, focus: async () => { focused++ }, action: async () => {}, fullscreen: () => false, dispose: async () => { disposed++ } }
  const coordinator = new NativePlaybackCoordinator("fixture", async () => surface, () => defaultPlaybackPreferences, () => {})
  return { coordinator, context, surface, requests, players, uploads, captureStatus: (status: number) => { captureStatus = status }, disposed: () => disposed, focused: () => focused,
    lock: (status = 200) => { unlocked = false; authStatus = status } }
}
describe("production native coordinator", () => {
  it("saves the selected part and actual frame time as multipart and publishes the gallery revision", async () => {
    const f = fixture()
    try {
      const session = await f.coordinator.open(f.context, { movieId: "a", fileId: "p2", startSec: 12, autoplay: false })
      const frame = await f.coordinator.capture(session.sessionId)
      expect(frame).toMatchObject({ movieId: "a", fileId: "p2", positionSec: 12.125, phase: "saved", preview: "data:image/png;base64," + Buffer.from("fixture png").toString("base64") })
      expect(f.uploads[0]!.metadata).toEqual({ id: frame.id, movieId: "a", fileId: "p2", title: "Fixture", code: "TEST", actors: ["Fixture actor"],
        positionSec: 12.125, capturedAt: "2026-10-06T01:02:03.000Z", tags: [] })
      // fetch 必须自行生成 multipart boundary，不能手工覆盖为 JSON。
      expect(f.uploads[0]!.headers.has("Content-Type")).toBe(false)
      expect(f.coordinator.snapshot()).toMatchObject({ captureRevision: 1, state: { status: "paused", positionSec: 12 } })
      await expect(f.coordinator.capture(session.sessionId, frame.id)).rejects.toThrow("STALE_CAPTURE")
    } finally { await f.coordinator.stop() }
  })
  it("retries identical metadata, image and ID without recapturing or incrementing a failed save", async () => {
    const f = fixture()
    try {
      const session = await f.coordinator.open(f.context, { movieId: "a", autoplay: true })
      f.captureStatus(500)
      const failed = await f.coordinator.capture(session.sessionId)
      expect(failed).toMatchObject({ phase: "error", error: "CAPTURE_SAVE_FAILED" })
      expect(f.coordinator.snapshot().captureRevision ?? 0).toBe(0)
      f.players[0]!.state.positionSec = 47
      f.captureStatus(204)
      const saved = await f.coordinator.capture(session.sessionId, failed.id)
      expect(saved).toMatchObject({ id: failed.id, positionSec: failed.positionSec, phase: "saved" })
      expect(f.uploads[1]).toEqual(f.uploads[0])
      expect(f.players[0]!.captures).toBe(1)
      expect(f.coordinator.snapshot().state.status).toBe("playing")
      expect(f.coordinator.snapshot().captureRevision).toBe(1)
    } finally { await f.coordinator.stop() }
  })
  it("rejects simultaneous captures and stale retries after replacing the part", async () => {
    const f = fixture()
    try {
      const session = await f.coordinator.open(f.context, { movieId: "a", autoplay: false })
      f.captureStatus(500)
      const first = f.coordinator.capture(session.sessionId)
      await expect(f.coordinator.capture(session.sessionId)).rejects.toThrow("CAPTURE_BUSY")
      const failed = await first
      await f.coordinator.command(session.sessionId, { action: "part", fileId: "p2" })
      await expect(f.coordinator.capture(session.sessionId, failed.id)).rejects.toThrow("STALE_PLAYBACK_SESSION")
      await expect(f.coordinator.capture(f.coordinator.snapshot().sessionId, failed.id)).rejects.toThrow("STALE_CAPTURE")
      expect(f.players[1]!.captures).toBe(0)
      expect(f.uploads).toHaveLength(1)
    } finally { await f.coordinator.stop() }
  })
  it.each([401, 403])("closes playback when the frame upload returns HTTP %i", async status => {
    const f = fixture()
    try {
      const session = await f.coordinator.open(f.context, { movieId: "a", autoplay: true })
      f.captureStatus(status)
      await expect(f.coordinator.capture(session.sessionId)).rejects.toThrow("SERVER_LOCKED")
      expect(f.players[0]!.stopped).toBe(true)
      expect(f.disposed()).toBe(1)
      expect(f.coordinator.snapshot()).toMatchObject({ windowOpen: false, state: { status: "error", error: "SERVER_LOCKED" } })
      expect(f.coordinator.snapshot().captureRevision ?? 0).toBe(0)
    } finally { await f.coordinator.stop() }
  })
  it("does not upload an unavailable frame and allows a later capture", async () => {
    const f = fixture()
    try {
      const session = await f.coordinator.open(f.context, { movieId: "a", autoplay: false })
      vi.spyOn(f.players[0]!, "captureFrame").mockRejectedValueOnce(new Error("CAPTURE_NOT_READY"))
      await expect(f.coordinator.capture(session.sessionId)).rejects.toThrow("CAPTURE_NOT_READY")
      expect(f.uploads).toHaveLength(0)
      expect((await f.coordinator.capture(session.sessionId)).phase).toBe("saved")
    } finally { await f.coordinator.stop() }
  })
  it("routes window actions to the owned surface and publishes only changed window state", async () => {
    const f = fixture()
    let maximized = false
    f.surface.maximized = () => maximized
    const action = vi.spyOn(f.surface, "action")
    try {
      const session = await f.coordinator.open(f.context, { movieId: "a", autoplay: false })
      await f.coordinator.command(session.sessionId, { action: "fullscreen" })
      await f.coordinator.command(session.sessionId, { action: "minimize" })
      expect(action.mock.calls).toEqual([["fullscreen"], ["minimize"]])
      await expect(f.coordinator.command("old-session", { action: "fullscreen" })).rejects.toThrow("STALE")
      f.coordinator.windowStateChanged()
      const revision = f.coordinator.snapshot().revision
      maximized = true
      f.coordinator.windowStateChanged()
      expect(f.coordinator.snapshot()).toMatchObject({ maximized: true, revision: revision + 1 })
      f.coordinator.windowStateChanged()
      expect(f.coordinator.snapshot().revision).toBe(revision + 1)
    } finally { await f.coordinator.stop() }
  })

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
