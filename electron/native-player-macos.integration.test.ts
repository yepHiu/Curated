import { execFile, spawn } from "node:child_process"
import { createRequire } from "node:module"
import { existsSync, mkdtempSync, readFileSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import path from "node:path"
import { promisify } from "node:util"
import { describe, expect, it } from "vitest"

const enabled = process.platform === "darwin" && process.env.CURATED_NATIVE_MAC_TEST === "1"
const runFile = promisify(execFile)
describe.skipIf(!enabled)("actual macOS Electron/libmpv playback", () => {
  it("shares authenticated playback, controls, frame capture, fullscreen, part switching and cleanup", async () => {
    expect(existsSync("electron-dist/native-player-macos.node"), "Build pnpm build:electron:main first").toBe(true)
    const directory = mkdtempSync(path.join(tmpdir(), "curated-mac-native-test-"))
    try {
      await runFile(process.env.CURATED_NATIVE_FFMPEG ?? "ffmpeg", ["-hide_banner", "-loglevel", "error", "-f", "lavfi",
        "-i", "testsrc2=size=960x540:rate=30", "-t", "30", "-c:v", "libx264", "-preset", "ultrafast", "-g", "30",
        "-pix_fmt", "yuv420p", "-movflags", "+faststart", path.join(directory, "fixture.mp4")])
      const executable = createRequire(import.meta.url)("electron") as string
      const env = { ...process.env, CURATED_MAC_QA_DIR: directory }
      delete env.ELECTRON_RUN_AS_NODE
      const child = spawn(executable, ["electron/fixtures/macos-native-playback.cjs"], { env, stdio: "pipe" })
      let output = ""
      child.stdout.on("data", chunk => { output += chunk.toString() })
      child.stderr.on("data", chunk => { output += chunk.toString() })
      const code = await new Promise<number | null>((resolve, reject) => {
        const timer = setTimeout(() => { child.kill(); reject(new Error(`MAC_QA_TIMEOUT\n${output}`)) }, 50000)
        child.once("error", error => { clearTimeout(timer); reject(error) })
        child.once("exit", value => { clearTimeout(timer); resolve(value) })
      })
      expect(code, output).toBe(0)
      const result = JSON.parse(readFileSync(path.join(directory, "result.json"), "utf8"))
      expect(result).toMatchObject({ progressSaved: true, socketsRemoved: true, closed: true,
        state: { fileId: "p2", diagnostics: { videoOutput: "libmpv", width: 960, height: 540 } } })
      expect(result.rangeRequests).toBeGreaterThanOrEqual(2)
    } finally { rmSync(directory, { recursive: true, force: true }) }
  }, 60000)
})
