import { execFileSync, spawn } from "node:child_process"
import { existsSync, mkdtempSync, readdirSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import path from "node:path"
import { pathToFileURL } from "node:url"
import { describe, expect, it } from "vitest"
import { NativePlayerWindow } from "./native-player-window"
import { NativeMpvPlayer } from "./native-mpv-player"

const hostExe = path.resolve(".workspace/native-player-dist/native-player-host.exe")
/** 状态驱动等待，不依赖桌面机器渲染时序。 */
async function until(predicate: () => boolean): Promise<void> {
  const deadline = Date.now() + 5000
  while (!predicate()) {
    if (Date.now() > deadline) throw new Error("host condition timeout")
    await new Promise<void>((resolve) => { /* 让 UI 消息循环运行。 */ setTimeout(resolve, 30) })
  }
}

describe.skipIf(process.platform !== "win32" || !existsSync(hostExe))("real Windows native host", () => {
  it("reports client bounds and fullscreen, rejects foreign HWND, minimizes and restores", async () => {
    const host = new NativePlayerWindow()
    const foreign = new NativePlayerWindow()
    try {
      const ready = await host.start(hostExe)
      expect(ready.visible).toBe(true)
      expect(ready.width).toBeGreaterThan(600)
      await foreign.start(hostExe)
      await expect(host.command("attach", { handle: foreign.state!.handle })).rejects.toThrow("NATIVE_HOST_COMMAND_REJECTED")
      await expect(host.command("resize", { width: 1, height: 1 })).rejects.toThrow("NATIVE_HOST_COMMAND_REJECTED")
      await host.command("resize", { width: 720, height: 560 })
      await until(() => { /* 边框使客户端尺寸略小于外框。 */ return host.state!.width < ready.width })
      const normal = { ...host.state! }
      await host.command("fullscreen")
      expect(host.state!.fullscreen).toBe(true)
      expect(host.state!.width).toBeGreaterThan(normal.width)
      await host.command("fullscreen")
      expect(host.state!.fullscreen).toBe(false)
      expect(host.state!.width).toBe(normal.width)
      await host.command("minimize")
      await until(() => { /* 最小化不能把透明控件留在原位置。 */ return !host.state!.visible })
      await host.command("restore")
      await until(() => { /* 恢复后客户端区域仍可用。 */ return host.state!.visible })
    } finally { await host.dispose(); await foreign.dispose() }
    await expect(host.command("restore")).rejects.toThrow("NATIVE_HOST_CLOSED")
  }, 20000)

  it("keeps system-chrome maximize, focus, minimize and fullscreen on the native host", async () => {
    const host = new NativePlayerWindow()
    try {
      const normal = await host.start(hostExe)
      expect(normal.maximized).toBe(false)
      await host.command("maximize")
      expect(host.state!.maximized).toBe(true)
      expect(host.state!.width).toBeGreaterThan(normal.width)
      await host.command("focus")
      expect(host.state!.maximized).toBe(true)
      await host.command("minimize")
      await until(() => !host.state!.visible)
      await host.command("focus")
      expect(host.state!).toMatchObject({ visible: true, maximized: true })
      await host.command("fullscreen")
      expect(host.state!).toMatchObject({ fullscreen: true, maximized: false })
      await host.command("maximize")
      expect(host.state!).toMatchObject({ fullscreen: false, maximized: true })
      await host.command("maximize")
      expect(host.state!.maximized).toBe(false)
      expect(host.state!.width).toBe(normal.width)
    } finally { await host.dispose() }
  }, 20000)

  it("exits on control-pipe EOF without orphaning a native window", async () => {
    const child = spawn(hostExe, [String(process.pid)], { windowsHide: true, stdio: "pipe" })
    const ended = new Promise<number | null>((resolve) => { /* 只等待当前 helper。 */ child.once("exit", resolve) })
    try {
      await new Promise<void>((resolve, reject) => {
        // 就绪后关闭 stdin，模拟父进程丢失控制通道。
        let text = ""
        child.on("error", reject)
        child.stdout.on("data", (chunk) => { text += chunk.toString(); if (text.includes('"event":"ready"')) resolve() })
      })
      child.stdin.end()
      expect(await ended).toBe(0)
    } finally { child.kill() }
  }, 10000)

  it("exits when its parent dies even while an inherited writer keeps stdin open", async () => {
    // 模拟指定的 Electron PID 退出，同时 Vitest 保持 helper 的 stdin 打开。
    const parent = spawn(process.execPath, ["-e", "setInterval(()=>{},1000)"], { stdio: "pipe", windowsHide: true })
    const child = spawn(hostExe, [String(parent.pid)], { stdio: "pipe", windowsHide: true })
    let text = ""
    child.stdout.on("data", (chunk) => { /* helper 的状态管道不依赖模拟 parent 的存活。 */ text += chunk.toString() })
    const ended = new Promise<number | null>((resolve) => { /* 确认进程退出，不能只依赖 closed 文本。 */ child.once("exit", resolve) })
    try {
      await until(() => { return text.includes('"event":"ready"') })
      parent.kill()
      await until(() => { return text.includes('"event":"closed"') })
      expect(await ended).toBe(0)
    } finally { child.stdin.end(); child.kill(); parent.kill() }
  }, 10000)

  it.skipIf(!process.env.CURATED_NATIVE_MPV)("plays inside the native HWND and survives host fullscreen and resize", async () => {
    const temp = mkdtempSync(path.join(tmpdir(), "curated-native-host-"))
    const video = path.join(temp, "fixture.mp4")
    const host = new NativePlayerWindow()
    let player: NativeMpvPlayer | undefined
    const captureDirectories = new Set(readdirSync(tmpdir()).filter(name => name.startsWith("curated-native-frame-")))
    try {
      execFileSync(process.env.CURATED_NATIVE_FFMPEG ?? "ffmpeg", ["-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i",
        "testsrc2=size=320x180:rate=30", "-t", "15", "-c:v", "libx264", "-preset", "ultrafast", video])
      const state = await host.start(hostExe)
      player = new NativeMpvPlayer(state.handle)
      await player.start(process.env.CURATED_NATIVE_MPV!, pathToFileURL(video).href, 2)
      await until(() => { return player!.state.positionSec > 2.2 })
      await host.command("fullscreen")
      await player.control({ action: "pause" })
      await player.control({ action: "seek", value: 7 })
      await until(() => { return Math.abs(player!.state.positionSec - 7) < 0.2 })
      // 视频 PNG 保持源尺寸；宿主/标题栏/HUD 的尺寸不进入萃取帧。
      let pausedFrame: Awaited<ReturnType<NativeMpvPlayer["captureFrame"]>> | undefined
      // seek 完成的属性事件与截图就绪可能错开，允许有界的就绪重试。
      for (let attempt = 0; attempt < 20 && !pausedFrame; attempt++) {
        try { pausedFrame = await player.captureFrame() }
        catch (error) { if (!(error instanceof Error) || error.message !== "CAPTURE_NOT_READY") throw error }
        if (!pausedFrame) await new Promise<void>(resolve => setTimeout(resolve, 30))
      }
      expect(pausedFrame).toBeDefined()
      expect(pausedFrame!.image.subarray(0, 8)).toEqual(Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]))
      expect([pausedFrame!.image.readUInt32BE(16), pausedFrame!.image.readUInt32BE(20)]).toEqual([320, 180])
      expect(pausedFrame!.positionSec).toBeCloseTo(7, 1)
      expect(await player.playbackPosition()).toBeCloseTo(pausedFrame!.positionSec, 2)
      expect(player.state.status).toBe("paused")
      await host.command("fullscreen")
      await host.command("resize", { width: 700, height: 540 })
      await player.control({ action: "resume" })
      await until(() => { return player!.state.positionSec > 7.3 })
      const playingFrame = await player.captureFrame()
      expect(playingFrame.positionSec).toBeGreaterThan(7.3)
      await until(() => player!.state.status === "playing" && player!.state.positionSec > playingFrame.positionSec + 0.1)
      expect(await player.playbackPosition()).toBeGreaterThan(playingFrame.positionSec)
      expect(readdirSync(tmpdir()).filter(name => name.startsWith("curated-native-frame-") && !captureDirectories.has(name))).toEqual([])
      expect(player.state.codec).toMatch(/h\.?264/i)
    } finally {
      await player?.stop(); await host.dispose()
      // mkdtemp 的明确测试目录，删除范围不包含用户文件。
      rmSync(temp, { recursive: true, force: true })
    }
  }, 20000)
})
