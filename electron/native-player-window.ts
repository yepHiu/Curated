import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process"
import { EventEmitter } from "node:events"

export interface NativeWindowState {
  handle: string
  x: number
  y: number
  width: number
  height: number
  dpi: number
  visible: boolean
  fullscreen: boolean
}
type HostAction = "attach" | "fullscreen" | "minimize" | "restore" | "resize" | "quit"
interface Pending { resolve(): void; reject(error: Error): void; timer: ReturnType<typeof setTimeout> }

/** 自有 Win32 宿主的内部协议；HWND 与测试用几何命令不进入 renderer。 */
export class NativePlayerWindow extends EventEmitter {
  private child?: ChildProcessWithoutNullStreams
  private text = ""
  private sequence = 0
  private stopping = false
  private pending = new Map<number, Pending>()
  state?: NativeWindowState

  /** 宿主只接收父 PID，等待 ready 后才允许启动播放器。 */
  async start(executable: string): Promise<NativeWindowState> {
    if (this.child || this.stopping) throw new Error("NATIVE_HOST_ALREADY_STARTED")
    const child = spawn(executable, [String(process.pid)], { windowsHide: true, stdio: "pipe", shell: false })
    this.child = child
    child.stdout.on("data", (chunk: Buffer) => { /* NDJSON 有界读取；非协议日志视为故障。 */ this.receive(chunk.toString("utf8")) })
    child.stdin.on("error", () => { /* 断管立即拒绝待响应命令。 */ this.fail() })
    child.on("error", () => { /* 无敏感启动参数进入错误消息。 */ this.fail() })
    child.on("exit", () => {
      // 父进程可据此有序停止 mpv，正常 dispose 不误报故障。
      this.child = undefined
      this.rejectPending()
      if (!this.stopping) this.emit("fault")
      this.emit("exit")
    })
    try {
      return await new Promise<NativeWindowState>((resolve, reject) => {
        // 就绪、故障和期限竞争后清除所有临时监听。
        const cleanup = () => { clearTimeout(timer); this.off("ready", ready); this.off("fault", failed) }
        const ready = (state: NativeWindowState) => { /* 首次 ready 返回宿主身份及客户端区域。 */ cleanup(); resolve(state) }
        const failed = () => { /* helper 启动失败不会退回无宿主的嵌入模式。 */ cleanup(); reject(new Error("NATIVE_HOST_START_FAILED")) }
        const timer = setTimeout(failed, 5000)
        this.once("ready", ready)
        this.once("fault", failed)
      })
    } catch (error) { await this.dispose(); throw error }
  }

  /** 只向自己创建的 helper 发固定协议消息，每个请求独立计时。 */
  command(action: HostAction, fields: { handle?: string; width?: number; height?: number } = {}): Promise<void> {
    if (!this.child || this.child.stdin.destroyed) return Promise.reject(new Error("NATIVE_HOST_CLOSED"))
    const id = ++this.sequence
    return new Promise<void>((resolve, reject) => {
      // native 命令超时不留下悬挂 Promise。
      const timer = setTimeout(() => { this.pending.delete(id); reject(new Error("NATIVE_HOST_TIMEOUT")) }, 2000)
      this.pending.set(id, { resolve, reject, timer })
      this.child!.stdin.write(`${JSON.stringify({ id, action, ...fields })}\n`)
    })
  }

  /** 有界解析宿主状态，拒绝超大行和无效几何值。 */
  private receive(chunk: string): void {
    this.text += chunk
    if (this.text.length > 65536) { this.fail(); return }
    let end: number
    while ((end = this.text.indexOf("\n")) >= 0) {
      const line = this.text.slice(0, end)
      this.text = this.text.slice(end + 1)
      let message: Record<string, unknown>
      try { message = JSON.parse(line) } catch { this.fail(); return }
      if (message.event === "ack") {
        const pending = this.pending.get(Number(message.id))
        if (pending) {
          clearTimeout(pending.timer); this.pending.delete(Number(message.id))
          if (message.ok === true) pending.resolve()
          else pending.reject(new Error("NATIVE_HOST_COMMAND_REJECTED"))
        }
      } else if (message.event === "ready" || message.event === "bounds") {
        if (typeof message.handle !== "string" || !/^[1-9]\d{0,19}$/.test(message.handle)
          || ![message.x, message.y, message.width, message.height, message.dpi].every((value) => {
            /* 屏幕坐标允许负数，尺寸在最小化时可为零。 */ return typeof value === "number" && Number.isFinite(value)
          }) || Number(message.width) < 0 || Number(message.height) < 0
          || Number(message.dpi) < 48 || typeof message.visible !== "boolean" || typeof message.fullscreen !== "boolean") {
          this.fail(); return
        }
        this.state = { handle: message.handle, x: Number(message.x), y: Number(message.y), width: Number(message.width),
          height: Number(message.height), dpi: Number(message.dpi), visible: message.visible, fullscreen: message.fullscreen }
        this.emit(message.event, this.state)
      } else if (message.event === "close") this.emit("close-request")
    }
  }

  /** 故障仅报告稳定诊断事件，由调用方回收播放会话。 */
  private fail(): void { this.rejectPending(); if (!this.stopping) this.emit("fault") }
  /** 断管与退出都释放本实例命令槽位。 */
  private rejectPending(): void {
    for (const pending of this.pending.values()) { clearTimeout(pending.timer); pending.reject(new Error("NATIVE_HOST_CLOSED")) }
    this.pending.clear()
  }

  /** 调用方先停止 mpv；quit/EOF/最后 kill 都只针对本实例进程。 */
  async dispose(): Promise<void> {
    if (this.stopping) return
    this.stopping = true
    const child = this.child
    if (!child) return
    const exited = new Promise<void>((resolve) => {
      // 超时仅回收当前 helper；不枚举或杀死其它播放器。
      const timer = setTimeout(() => { child.kill(); resolve() }, 2500)
      child.once("exit", () => { clearTimeout(timer); resolve() })
    })
    await this.command("quit").catch(() => { /* 子进程退出可能先于 ack，EOF 是第二退出通道。 */ })
    child.stdin.end()
    await exited
  }
}
