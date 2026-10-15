import { writeFile } from "node:fs/promises"
import { describe, expect, it } from "vitest"
import { NativeMpvPlayer } from "./native-mpv-player"
import { defaultSubtitleDisplay } from "./subtitle-format"

/** 模拟 IPC 的迟到截图响应，验证编码期间的控制和文件完成边界。 */
function fixture(status: "playing" | "paused") {
  const player = new NativeMpvPlayer()
  player.state.status = status
  const transport = player as unknown as {
    loaded: boolean
    socket: { destroyed: boolean; write(line: string): void }
    receive(line: string): void
  }
  transport.loaded = true
  const commands: { command: unknown[]; request_id: number; async: boolean }[] = []
  let finish: ((error?: string) => void) | undefined
  let ready!: () => void
  const writing = new Promise<void>(resolve => { ready = resolve })
  transport.socket = {
    destroyed: false,
    write(line) {
      const message = JSON.parse(line) as typeof commands[number]
      commands.push(message)
      const reply = (data: unknown, error = "success") => transport.receive(JSON.stringify({ request_id: message.request_id, error, data }) + "\n")
      if (message.command[0] === "screenshot-to-file") {
        const png = Buffer.alloc(24)
        Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]).copy(png)
        png.writeUInt32BE(1920, 16)
        png.writeUInt32BE(1080, 20)
        void writeFile(String(message.command[1]), png).then(() => {
          finish = (error) => reply(null, error)
          ready()
        })
      } else {
        queueMicrotask(() => {
          if (message.command[0] === "set_property" && message.command[1] === "pause") {
            transport.receive(JSON.stringify({ event: "property-change", name: "pause", data: message.command[2] }) + "\n")
          }
          reply(message.command[1] === "time-pos" ? 12.5 : false)
        })
      }
    },
  }
  return { player, commands, writing, finish: (error?: string) => finish!(error) }
}

describe("native frame capture during playback", () => {
  it("loads a main-resolved subtitle once, applies positive delay and keeps captures video-only", async () => {
    const f = fixture("paused")
    await f.player.subtitles("/main-owned/chinese.srt", { ...defaultSubtitleDisplay, offset: 1.5 })
    await f.player.subtitles("/main-owned/chinese.srt", { ...defaultSubtitleDisplay, size: 32 })
    await f.player.subtitles(undefined, { ...defaultSubtitleDisplay, mode: "off" })
    expect(f.commands.filter(item => item.command[0] === "sub-add")).toHaveLength(1)
    expect(f.commands.some(item => JSON.stringify(item.command) === JSON.stringify(["set_property", "sub-delay", 1.5]))).toBe(true)
    expect(f.commands.at(-1)?.command).toEqual(["set_property", "sid", "no"])
    const frame = f.player.captureFrame(); await f.writing; f.finish(); await frame
    expect(f.commands.find(item => item.command[0] === "screenshot-to-file")?.command[2]).toBe("video")
  })
  it.each(["playing", "paused"] as const)("keeps %s state while waiting for PNG completion", async status => {
    const f = fixture(status)
    let completed = false
    const capture = f.player.captureFrame().then(frame => { completed = true; return frame })
    await f.writing
    expect(completed).toBe(false)
    expect(f.player.state.status).toBe(status)
    expect(f.commands.some(message => message.command[0] === "set_property")).toBe(false)
    expect(f.commands.find(message => message.command[0] === "screenshot-to-file")!.async).toBe(true)
    // 截图尚未完成时仍可查询媒体时钟。
    expect(await f.player.playbackPosition()).toBe(12.5)
    f.finish()
    expect(await capture).toMatchObject({ positionSec: 12.5, image: expect.any(Buffer) })
    expect(f.player.state.status).toBe(status)
  })

  it("preserves a user pause issued while encoding the screenshot", async () => {
    const f = fixture("playing")
    const capture = f.player.captureFrame()
    await f.writing
    await f.player.control({ action: "pause" })
    f.finish()
    await capture
    expect(f.player.state.status).toBe("paused")
    expect(f.commands.filter(message => message.command[0] === "set_property").map(message => message.command)).toEqual([["set_property", "pause", true]])
  })

  it("leaves playback running when the screenshot command fails", async () => {
    const f = fixture("playing")
    const result = expect(f.player.captureFrame()).rejects.toThrow("MPV_COMMAND_FAILED")
    await f.writing
    f.finish("error")
    await result
    expect(f.player.state.status).toBe("playing")
    expect(f.commands.some(message => message.command[0] === "set_property")).toBe(false)
  })
})
