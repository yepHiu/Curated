import { spawn, type ChildProcess } from "node:child_process"
import { createConnection, type Socket } from "node:net"
import { randomUUID } from "node:crypto"
import { EventEmitter } from "node:events"
import { emptyNativePlayerState, type NativeLabControl, type NativePlayerState } from "./native-player-contract.js"

interface PendingCommand { resolve(value: unknown): void; reject(error: Error): void; timer: ReturnType<typeof setTimeout> }
const properties = ["time-pos", "duration", "pause", "speed", "volume", "frame-drop-count",
  "decoder-frame-drop-count", "video-codec", "hwdec-current"]

/** 一次会话对应一个 mpv 子进程；命令关联和状态事件均局限在该实例。 */
export class NativeMpvPlayer extends EventEmitter {
  private child?: ChildProcess
  private socket?: Socket
  private sequence = 0
  private text = ""
  private closed = false
  private loaded = false
  private paused = false
  private readonly pending = new Map<number, PendingCommand>()
  state: NativePlayerState = emptyNativePlayerState()

  /** 嵌入句柄只能由受信任的主进程宿主提供，不能由媒体请求设置。 */
  constructor(private readonly windowId?: string) {
    super()
    if (windowId !== undefined && !/^[1-9]\d{0,19}$/.test(windowId)) throw new Error("INVALID_NATIVE_WINDOW")
  }

  /** 通过观察状态即可验证播放效果；headless 仅供合成媒体集成测试。 */
  async start(executable: string, url: string, startSec: number, headless = false, initiallyPaused = false): Promise<void> {
    if (this.child || this.closed) throw new Error("PLAYER_ALREADY_STARTED")
    if (process.platform !== "win32") throw new Error("WINDOWS_REQUIRED")
    const pipe = `\\\\.\\pipe\\curated-native-${randomUUID()}`
    this.state.status = "starting"
    const args = ["--no-config", "--idle=yes", "--keep-open=no", "--terminal=no", "--ytdl=no",
      "--hwdec=auto-safe", `--input-ipc-server=${pipe}`, `--start=${startSec}`,
      ...(initiallyPaused ? ["--pause=yes"] : []),
      ...(this.windowId && !headless ? [`--wid=${this.windowId}`, "--osc=no", "--input-default-bindings=no", "--input-vo-keyboard=no", "--input-cursor=no"] : []),
      ...(headless ? ["--vo=null", "--ao=null"] : ["--force-window=yes", "--title=Curated · Windows 原生播放原型"])]
    const child = spawn(executable, args, { shell: false, windowsHide: true, stdio: "ignore" })
    this.child = child
    child.on("error", () => { // 启动错误不得包含可执行参数或媒体能力地址。
      this.fail("MPV_START_FAILED")
    })
    child.on("exit", (code) => { // 手动关闭视频窗口与异常退出都释放管道和待响应请求。
      this.child = undefined
      this.socket?.destroy()
      this.rejectPending("MPV_EXITED")
      if (!this.closed && this.state.status !== "error") {
        this.state.status = code === 0 ? "stopped" : "error"
        if (code !== 0) this.state.error = "MPV_EXITED"
        this.publish()
      }
      this.emit("exit")
    })
    try {
      await this.connect(pipe)
      this.socket!.on("data", (chunk: Buffer) => { // JSON IPC 以换行分隔，允许拆包及合包。
        this.receive(chunk.toString("utf8"))
      })
      this.socket!.on("error", () => { // 断管不继续等待已经无法到达的响应。
        if (!this.closed) this.fail("MPV_PIPE_FAILED")
      })
      this.socket!.on("close", () => { // 通知业务层连接故障，不尝试无限重启。
        this.rejectPending("MPV_PIPE_CLOSED")
        if (!this.closed && this.child && this.state.status !== "error") this.fail("MPV_PIPE_CLOSED")
      })
      for (const [index, property] of properties.entries()) await this.command(["observe_property", index + 1, property])
      // 先订阅后加载媒体，避免快速失败的片源在建立 IPC 前丢失 end-file。
      for (const property of properties) {
        const value = await this.command(["get_property", property]).catch(() => { /* 未加载时部分属性不可用。 */ return undefined })
        this.update(property, value)
      }
      await this.command(["loadfile", url, "replace"])
      await this.waitReady()
    } catch (failure) {
      const code = failure instanceof Error && /^[A-Z_]+$/.test(failure.message) ? failure.message : "MPV_START_FAILED"
      this.fail(this.state.error ?? code)
      await this.stop()
      throw new Error(this.state.error ?? "MPV_START_FAILED")
    }
  }

  /** file-loaded 才算启动成功，错误媒体和加载超时有明确终态。 */
  private async waitReady(): Promise<void> {
    if (this.state.status === "error") throw new Error(this.state.error)
    if (this.loaded) return
    await new Promise<void>((resolve, reject) => {
      // 监听当前实例，不接受其它会话的就绪事件。
      const changed = (state: NativePlayerState) => {
        if (!this.loaded && state.status !== "error") return
        clearTimeout(timer)
        this.removeListener("state", changed)
        if (state.status === "error") reject(new Error(state.error))
        else resolve()
      }
      const timer = setTimeout(() => {
        // 有界期限到达后卸下监听，由 start 的失败路径回收子进程。
        this.removeListener("state", changed)
        reject(new Error("MPV_MEDIA_LOAD_TIMEOUT"))
      }, 10000)
      this.on("state", changed)
    })
  }

  /** 仅在有界期限内等待自己创建的 Windows named pipe。 */
  private async connect(pipe: string): Promise<void> {
    const deadline = Date.now() + 8000
    while (Date.now() < deadline && this.child && this.state.status !== "error") {
      try {
        this.socket = await new Promise<Socket>((resolve, reject) => {
          // named pipe 不存在时等待下一轮，已连接即移交读取所有权。
          const socket = createConnection(pipe)
          socket.once("error", reject)
          socket.once("connect", () => {
            socket.removeListener("error", reject)
            resolve(socket)
          })
        })
        return
      } catch {
        await new Promise<void>((resolve) => { // 启动退避不阻塞主进程。
          setTimeout(resolve, 60)
        })
      }
    }
    throw new Error("MPV_START_TIMEOUT")
  }

  /** 命令以独立 request_id 匹配；错误和超时释放对应槽位。 */
  private command(command: unknown[]): Promise<unknown> {
    if (!this.socket || this.socket.destroyed) return Promise.reject(new Error("MPV_NOT_CONNECTED"))
    const id = ++this.sequence
    return new Promise((resolve, reject) => {
      // 定时器只拒绝本条命令，不保留已超时的 resolver。
      const timer = setTimeout(() => {
        this.pending.delete(id)
        reject(new Error("MPV_COMMAND_TIMEOUT"))
      }, 2000)
      this.pending.set(id, { resolve, reject, timer })
      this.socket!.write(JSON.stringify({ command, request_id: id }) + "\n")
    })
  }

  /** 处理 JSON 行及状态事件，限制意外超大消息占用内存。 */
  private receive(chunk: string): void {
    this.text += chunk
    if (this.text.length > 1024 * 1024) { this.fail("MPV_MESSAGE_TOO_LARGE"); return }
    let end: number
    while ((end = this.text.indexOf("\n")) >= 0) {
      const line = this.text.slice(0, end)
      this.text = this.text.slice(end + 1)
      let event: { request_id?: number; error?: string; data?: unknown; event?: string; name?: string; reason?: string }
      try { event = JSON.parse(line) } catch { this.fail("MPV_INVALID_MESSAGE"); return }
      if (event.request_id !== undefined) {
        const pending = this.pending.get(event.request_id)
        if (pending) {
          clearTimeout(pending.timer)
          this.pending.delete(event.request_id)
          if (event.error === "success") pending.resolve(event.data)
          else pending.reject(new Error("MPV_COMMAND_FAILED"))
        }
      }
      if (event.event === "property-change" && event.name) this.update(event.name, event.data)
      if (event.event === "file-loaded") {
        this.loaded = true
        this.state.status = this.paused ? "paused" : "playing"
        this.publish()
      }
      if (event.event === "end-file") {
        this.state.status = event.reason === "eof" ? "ended" : event.reason === "error" ? "error" : "stopped"
        if (event.reason === "error") this.state.error = "MEDIA_PLAYBACK_FAILED"
        this.publish()
      }
    }
  }

  /** 从引擎属性投影允许公开的状态，不泄露 mpv 的源地址。 */
  private update(name: string, value: unknown): void {
    if (typeof value === "number" && Number.isFinite(value)) {
      const fields: Record<string, "positionSec" | "durationSec" | "speed" | "volume" | "droppedFrames" | "decoderDroppedFrames"> = {
        "time-pos": "positionSec", duration: "durationSec", speed: "speed", volume: "volume",
        "frame-drop-count": "droppedFrames", "decoder-frame-drop-count": "decoderDroppedFrames",
      }
      if (fields[name]) this.state[fields[name]] = Math.max(0, value)
      if (name === "time-pos" && ["starting", "playing", "paused"].includes(this.state.status)) {
        // pause 属性独立更新，时间事件不覆盖明确暂停状态。
        if (this.state.status !== "paused") this.state.status = "playing"
      }
    }
    if (name === "pause" && typeof value === "boolean") {
      this.paused = value
      if (this.loaded && ["starting", "playing", "paused"].includes(this.state.status)) this.state.status = value ? "paused" : "playing"
    }
    if (name === "video-codec" && typeof value === "string") this.state.codec = value
    if (name === "hwdec-current" && typeof value === "string") this.state.hwdec = value
    this.publish()
  }

  /** 显式枚举控制映射，不把 renderer 内容作为任意 mpv 命令执行。 */
  async control(input: NativeLabControl): Promise<void> {
    if (input.action === "stop") { await this.stop(); return }
    if (input.action === "pause" || input.action === "resume") {
      await this.command(["set_property", "pause", input.action === "pause"])
      return
    }
    const value = input.value
    if (typeof value !== "number" || !Number.isFinite(value)) throw new Error("INVALID_CONTROL")
    if (input.action === "seek" && value >= 0 && (this.state.durationSec <= 0 || value <= this.state.durationSec)) {
      await this.command(["seek", value, "absolute+exact"])
    } else if (input.action === "speed" && value >= 0.25 && value <= 4) {
      await this.command(["set_property", "speed", value])
    } else if (input.action === "volume" && value >= 0 && value <= 100) {
      await this.command(["set_property", "volume", value])
    } else throw new Error("INVALID_CONTROL")
  }

  /** 通知状态订阅者时复制快照，避免外部修改引擎状态。 */
  private publish(): void { this.emit("state", { ...this.state }) }

  /** 失败使待响应请求全部结束，业务层负责关闭对应代理。 */
  private fail(code: string): void {
    this.state.status = "error"
    this.state.error = code
    this.rejectPending(code)
    this.publish()
  }

  /** 退出或断管时清理所有命令期限。 */
  private rejectPending(code: string): void {
    for (const pending of this.pending.values()) { clearTimeout(pending.timer); pending.reject(new Error(code)) }
    this.pending.clear()
  }

  /** 优先退出本次 mpv，超过期限只终止自己创建的子进程。 */
  async stop(): Promise<void> {
    if (this.closed) return
    this.closed = true
    const child = this.child
    if (child) {
      const exited = new Promise<void>((resolve) => { // 安装退出监听再发送 quit，避免竞争。
        child.once("exit", () => resolve())
      })
      await this.command(["quit"]).catch(() => { /* mpv 退出可能先于 quit 响应。 */ })
      let timer: ReturnType<typeof setTimeout> | undefined
      await Promise.race([exited, new Promise<void>((resolve) => {
        // 有界退出只作用于当前 child，保留其它系统播放器。
        timer = setTimeout(() => { child.kill(); resolve() }, 2000)
      })])
      clearTimeout(timer)
    }
    this.socket?.destroy()
    this.rejectPending("MPV_STOPPED")
    if (this.state.status !== "error" && this.state.status !== "ended") this.state.status = "stopped"
    this.publish()
  }
}
