import { execFileSync } from "node:child_process"
import { mkdtempSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import path from "node:path"
import { pathToFileURL } from "node:url"
import { describe, expect, it } from "vitest"
import { NativeMpvPlayer } from "./native-mpv-player"
import { NativeMediaProxy } from "./native-media-proxy"
import { createServer } from "node:http"
import { readFileSync } from "node:fs"

const executable = process.env.CURATED_NATIVE_MPV
/** 等到真实引擎满足状态条件，有界期限内失败而不是固定长等待。 */
async function until(predicate: () => boolean, timeout = 6000): Promise<void> {
  const deadline = Date.now() + timeout
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error("condition timeout")
    await new Promise<void>((resolve) => { setTimeout(resolve, 40) })
  }
}

describe.skipIf(process.platform !== "win32" || !executable)("real Windows mpv", () => {
  // 使用合成视频与受保护 Range fixture，验证真实原生链路，不声称问题影片观感已验收。
  it("plays authenticated raw MP4 through proxy and handles pause seek speed volume and cleanup", async () => {
    const temp = mkdtempSync(path.join(tmpdir(), "curated-native-mpv-"))
    const video = path.join(temp, "fixture.mp4")
    execFileSync(process.env.CURATED_NATIVE_FFMPEG ?? "ffmpeg", ["-hide_banner", "-loglevel", "error", "-f", "lavfi",
      "-i", "testsrc2=size=320x180:rate=30", "-t", "15", "-c:v", "libx264", "-preset", "ultrafast", "-g", "30", "-pix_fmt", "yuv420p", "-movflags", "+faststart", video])
    const bytes = readFileSync(video)
    let rangeRequests = 0
    const server = createServer((request, response) => {
      // 认证和第二片身份由 fixture 显式要求。
      if (request.url !== "/stream?fileId=part-2" || request.headers.cookie !== "fixture=secret") { response.writeHead(403).end(); return }
      const range = request.headers.range?.match(/^bytes=(\d+)-(\d*)$/)
      const start = range ? Number(range[1]) : 0
      const end = range?.[2] ? Math.min(Number(range[2]), bytes.length - 1) : bytes.length - 1
      if (range) rangeRequests++
      response.writeHead(range ? 206 : 200, { "Content-Type": "video/mp4", "Accept-Ranges": "bytes", "Content-Length": String(end - start + 1),
        ...(range ? { "Content-Range": `bytes ${start}-${end}/${bytes.length}` } : {}) })
      response.end(bytes.subarray(start, end + 1))
    })
    await new Promise<void>((resolve) => { server.listen(0, "127.0.0.1", resolve) })
    const address = server.address()
    if (!address || typeof address === "string") throw new Error("fixture bind failed")
    const proxy = new NativeMediaProxy(`http://127.0.0.1:${address.port}/stream?fileId=part-2`, async (url, init) => {
      // 只有代理接触认证，mpv 接收 loopback 能力 URL。
      return await fetch(url, { ...init, headers: { ...init?.headers, Cookie: "fixture=secret" } })
    })
    const player = new NativeMpvPlayer()
    try {
      await player.start(executable!, await proxy.start(), 1, true, true)
      await until(() => player.state.status === "paused" && player.state.positionSec >= 1 && player.state.durationSec > 14)
      const loadedPosition = player.state.positionSec
      await new Promise<void>(resolve => { setTimeout(resolve, 250) })
      expect(player.state.positionSec).toBeCloseTo(loadedPosition, 1)
      await player.control({ action: "resume" })
      await until(() => { return player.state.positionSec > 1.2 && player.state.durationSec > 14 })
      await player.control({ action: "pause" })
      await until(() => { return player.state.status === "paused" })
      const paused = player.state.positionSec
      await new Promise<void>((resolve) => { setTimeout(resolve, 250) })
      expect(player.state.positionSec).toBeCloseTo(paused, 1)
      await player.control({ action: "seek", value: 7 })
      await until(() => { return Math.abs(player.state.positionSec - 7) < 0.15 })
      await player.control({ action: "speed", value: 1.5 })
      await player.control({ action: "volume", value: 35 })
      await until(() => { return player.state.speed === 1.5 && player.state.volume === 35 })
      await player.control({ action: "resume" })
      await until(() => { return player.state.positionSec > 7.3 })
      await expect(player.control({ action: "speed", value: 100 })).rejects.toThrow("INVALID_CONTROL")
      expect(rangeRequests).toBeGreaterThan(0)
      expect(player.state.codec).toMatch(/h\.?264/i)
      await until(() => player.state.diagnostics?.width === 320)
      expect(player.state.diagnostics).toMatchObject({ width: 320, height: 180, sourceFps: 30, container: "mov,mp4,m4a,3gp,3g2,mj2",
        videoOutput: "null", audioCodec: null, audioSampleRate: null })
      // 硬解可以将片源 yuv420p 转成 nv12；此字段是解码后的像素格式。
      expect(player.state.diagnostics!.pixelFormat).toMatch(/^(nv12|yuv420p)$/)
      expect(player.state.diagnostics!.engineVersion).toMatch(/mpv/i)
      await until(() => (player.state.diagnostics?.videoBitrate ?? 0) > 0)
      expect(player.state.diagnostics!.videoBitrate).toBeGreaterThan(0)
      expect(proxy.diagnostics()).toMatchObject({ mime: "video/mp4" })
      expect(proxy.diagnostics().receivedBytes).toBeGreaterThan(0)
      await player.stop()
      expect(player.state.status).toBe("stopped")
    } finally {
      await player.stop(); await proxy.stop()
      await new Promise<void>((resolve) => { server.close(() => resolve()); server.closeAllConnections() })
      // temp 是 mkdtemp 返回的已知系统临时子目录，不包含用户媒体。
      rmSync(temp, { recursive: true, force: true })
    }
  }, 20000)

  // 错误媒体不能伪造持续播放状态。
  it("reports missing media without leaving a running process", async () => {
    const player = new NativeMpvPlayer()
    try {
      await player.start(executable!, pathToFileURL(path.join(tmpdir(), "nonexistent-curated-native-fixture.mp4")).href, 0, true).catch(() => undefined)
      await until(() => { return player.state.status === "error" })
      expect(player.state.error).toBeTruthy()
    } finally { await player.stop() }
  }, 15000)
})
