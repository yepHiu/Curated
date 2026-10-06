import { describe, expect, it, vi } from "vitest"
import { NativePlayerLab } from "./native-player-lab"
import { NativeMpvPlayer } from "./native-mpv-player"
import type { NativeLabControl } from "./native-player-contract"

/** 替代引擎以精确检查业务身份和保存路径，不模拟视频质量。 */
class FixturePlayer extends NativeMpvPlayer {
  url = ""
  stopped = false
  /** 记录代理来源并建立有效播放时间。 */
  override async start(_executable: string, url: string, start: number): Promise<void> {
    this.url = url
    this.state = { ...this.state, status: "playing", positionSec: start + 1, durationSec: 100 }
  }
  /** 控制只改变 fixture 状态，真实管道另由集成测试验证。 */
  override async control(input: NativeLabControl): Promise<void> {
    if (input.action === "pause") this.state.status = "paused"
  }
  /** 标记自己的回收，便于检查换片没有残留。 */
  override async stop(): Promise<void> { this.stopped = true; this.state.status = "stopped" }
}

describe("native player lab identity", () => {
  // 第二片保存必须同时有 query fileId 与正文 fileId。
  it("keeps file identity on raw playback and progress; replaces and stops only its player", async () => {
    const requests: { url: string; body?: string; method?: string }[] = []
    const players: FixturePlayer[] = []
    const lab = new NativePlayerLab(() => {
      // fixture API 与业务断言共享请求记录。
      return async (url, init) => {
        requests.push({ url, body: init?.body as string | undefined, method: init?.method })
        if (url.endsWith("/health")) return Response.json({ name: "curated-dev" })
        if (url.endsWith("/auth/status")) return Response.json({ unlocked: true, pinEnabled: false })
        if (url.includes("/playback/progress/")) return new Response(null, { status: 204 })
        if (url.includes("/playback-session")) return Response.json({ mode: "direct", resumePositionSec: 42, durationSec: 100 })
        return Response.json({ id: "movie", code: "TEST", title: "Fixture", files: [
          { id: "part-1", fileName: "1.mp4", location: "private/path" }, { id: "part-2", fileName: "2.mp4" },
        ] })
      }
    }, () => { // 每次播放实例都使用不同的 fixture 引擎。
      const player = new FixturePlayer(); players.push(player); return player
    })
    lab.setExecutable("fixture-mpv.exe")
    try {
      await lab.connect("http://127.0.0.1:43210")
      const detail = await lab.detail("movie")
      expect(JSON.stringify(detail)).not.toContain("private/path")
      expect(await lab.resume({ movieId: "movie", fileId: "part-2" })).toBe(42)
      const resume = requests.find((request) => { return request.url.includes("playback-session") })
      expect(resume?.body).toBe('{"mode":"direct"}')
      await lab.start({ movieId: "movie", fileId: "part-2", startSec: 10 })
      expect(lab.status().state.fileId).toBe("part-2")
      expect(players[0]!.url).toMatch(/^http:\/\/127\.0\.0\.1:\d+\/[a-f0-9]+\/media$/)
      expect(JSON.stringify(lab.status())).not.toContain(players[0]!.url)
      await lab.control({ action: "pause" })
      await lab.control({ action: "pause" })
      await lab.start({ movieId: "movie", fileId: "part-1", startSec: 0 })
      expect(players[0]!.stopped).toBe(true)
      const saves = requests.filter((request) => { return request.url.includes("/playback/progress/") })
      expect(saves).toHaveLength(1)
      expect(saves[0]!.url).toContain("?fileId=part-2")
      expect(JSON.parse(saves[0]!.body!)).toMatchObject({ fileId: "part-2", positionSec: 11, durationSec: 100 })
      await expect(lab.start({ movieId: "movie", fileId: "foreign-file", startSec: 0 })).rejects.toThrow("INVALID_MOVIE_FILE")
      expect(lab.status().state.fileId).toBe("part-1")
      await lab.stop()
      expect(players[1]!.stopped).toBe(true)
    } finally { await lab.stop() }
  })

  // 缓存过的媒体也必须响应 Server 会话被锁定的事实。
  it("stops native playback when authentication becomes locked", async () => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval"] })
    let unlocked = true
    const player = new FixturePlayer()
    const lab = new NativePlayerLab(() => { return async (url) => {
      // 控制 fixture 认证状态，保持其它业务响应有效。
      if (url.endsWith("/health")) return Response.json({ name: "curated" })
      if (url.endsWith("/auth/status")) return Response.json({ unlocked, pinEnabled: true })
      if (url.includes("/progress/")) return new Response(null, { status: unlocked ? 204 : 403 })
      return Response.json({ id: "movie", code: "TEST", title: "Fixture", files: [{ id: "part-1", fileName: "1.mp4" }] })
    } }, () => { /* 此测试只构造一个待回收的会话。 */ return player })
    try {
      lab.setExecutable("fixture-mpv.exe")
      await lab.connect("http://127.0.0.1:43210")
      await lab.start({ movieId: "movie", fileId: "part-1", startSec: 2 })
      unlocked = false
      await vi.advanceTimersByTimeAsync(5000)
      expect(player.stopped).toBe(true)
      expect(lab.status().connection?.unlocked).toBe(false)
      expect(lab.status().state.error).toBe("SERVER_LOCKED")
    } finally { await lab.stop(); vi.useRealTimers() }
  })

  // 验证认证准入而不是测试 fixture 本身。
  it("does not read library data until unlocked; rejects non-root origins", async () => {
    const paths: string[] = []
    const lab = new NativePlayerLab(() => { return async (url) => {
      paths.push(url)
      return Response.json(url.endsWith("/health") ? { name: "curated" } : { unlocked: false, pinEnabled: true })
    } })
    await expect(lab.connect("http://127.0.0.1/api")).rejects.toThrow()
    await lab.connect("http://127.0.0.1:43210")
    await expect(lab.search("test")).rejects.toThrow("SERVER_LOCKED")
    expect(paths.some((url) => { return url.includes("library") })).toBe(false)
  })
})
