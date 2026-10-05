import { spawn, execFile } from "node:child_process"
import { createRequire } from "node:module"
import { existsSync, mkdtempSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import path from "node:path"
import { promisify } from "node:util"
import { describe, expect, it } from "vitest"

const runFile = promisify(execFile)
const root = process.cwd()
const entry = path.join(root, ".workspace/native-player-dist/native-player-prototype.js")
const hostExe = path.join(root, ".workspace/native-player-dist/native-player-host.exe")

describe.skipIf(process.platform !== "win32" || !existsSync(entry) || !existsSync(hostExe))("actual Electron native overlay", () => {
  it("leaves drag/resize to the native frame and follows its physical client bounds exactly", async () => {
    // 测试生产原型入口，避免复制窗口选项后产生假通过。
    const temp = mkdtempSync(path.join(tmpdir(), "curated-native-overlay-"))
    const electron = createRequire(import.meta.url)("electron") as string
    const env = { ...process.env, CURATED_NATIVE_PROFILE: path.join(temp, "profile") }
    delete env.ELECTRON_RUN_AS_NODE
    const child = spawn(electron, [entry], { cwd: root, windowsHide: true, stdio: "pipe",
      env })
    const ended = new Promise<void>((resolve) => { /* 只等待当前 Electron 实例。 */ child.once("exit", () => { resolve() }) })
    let output = ""
    child.stdout.on("data", (chunk) => { /* 失败诊断不含业务请求，测试未连接 Server。 */ output += chunk.toString() })
    child.stderr.on("data", (chunk) => { output += chunk.toString() })
    try {
      const probe = await runFile("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass", "-File",
        path.join(root, "electron/fixtures/native-player-window-probe.ps1"), "-ElectronProcessId", String(child.pid)],
      { windowsHide: true, timeout: 15000 })
      const result = JSON.parse(probe.stdout.trim()) as {
        overlayHits: number[]; finalOverlayHits: number[]; hostHits: number[]; overlayStyle: number
        samples: { label: string; hostClient: number[]; overlay: number[] }[]
        overlayHiddenWhenMinimized: boolean; overlayVisibleAfterRestore: boolean
      }
      // HTCLIENT=1：透明层八个边角不能成为自己的系统缩放/拖动区域。
      expect(result.overlayHits).toEqual(Array(8).fill(1))
      expect(result.finalOverlayHits).toEqual(Array(8).fill(1))
      expect(result.overlayStyle & 0x00040000).toBe(0) // WS_THICKFRAME
      expect(result.hostHits).toEqual([13, 17, 2]) // 外框可缩放，标题栏可拖动。
      expect(result.samples).toHaveLength(6)
      for (const sample of result.samples) expect(sample.overlay, sample.label).toEqual(sample.hostClient)
      expect(result.samples[4]!.hostClient).not.toEqual(result.samples[0]!.hostClient)
      expect(result.overlayHiddenWhenMinimized).toBe(true)
      expect(result.overlayVisibleAfterRestore).toBe(true)
      await Promise.race([ended, new Promise<never>((_resolve, reject) => {
        // 回收超时会失败，不能把残留进程作为成功结束。
        const timer = setTimeout(() => { reject(new Error(`NATIVE_OVERLAY_EXIT_TIMEOUT ${output}`)) }, 5000)
        ended.then(() => { clearTimeout(timer) })
      })])
    } catch (error) { throw new Error(`${error instanceof Error ? error.message : String(error)}\n${output}`) }
    finally {
      if (child.exitCode === null) { child.kill(); await ended }
      // mkdtemp 的明确测试 profile，不清除用户媒体/正式 profile。
      rmSync(temp, { recursive: true, force: true })
    }
  }, 25000)
})
