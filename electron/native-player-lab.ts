import { NativeMediaProxy, type MediaFetcher } from "./native-media-proxy.js"
import { NativeMpvPlayer } from "./native-mpv-player.js"
import { normalizeServerUrl } from "./server-connections.js"
import { emptyNativePlayerState, type NativeLabConnection, type NativeLabControl, type NativeLabMovie,
  type NativeLabStart, type NativeLabStatus, type NativePlayerState } from "./native-player-contract.js"

interface ActivePlayback {
  player: NativeMpvPlayer; proxy: NativeMediaProxy; origin: string; movieId: string; fileId?: string
  lastSaved?: { positionSec: number; durationSec: number }
}

/** 原型业务服务；HTTP 与进程均由主进程持有，不暴露通用网络/文件能力。 */
export class NativePlayerLab {
  private connection?: NativeLabConnection
  private active?: ActivePlayback
  private lastState = emptyNativePlayerState()
  private executable?: string
  private tail: Promise<unknown> = Promise.resolve()
  private timer?: ReturnType<typeof setInterval>
  private periodicPending = false

  /** 每个来源由调用方提供对应 Electron session，测试可以注入 fixture fetcher。 */
  constructor(private readonly fetchForOrigin: (origin: string) => MediaFetcher,
    private readonly createPlayer: () => NativeMpvPlayer = () => { /* 默认每次构造独立原生会话。 */ return new NativeMpvPlayer() }) {}

  /** 串行化源切换和控制，失败不阻塞后续操作。 */
  run<T>(operation: () => Promise<T>): Promise<T> {
    const result = this.tail.then(operation)
    this.tail = result.catch(() => { /* 将失败返回给调用者，队列继续服务。 */ })
    return result
  }

  /** 可执行路径只允许原型主进程从原生文件选择器设置。 */
  setExecutable(executable: string): void { this.executable = executable }

  /** 获取不含地址能力及认证凭据的状态快照。 */
  status(): NativeLabStatus {
    const state = this.active ? { ...this.active.player.state, movieId: this.active.movieId, fileId: this.active.fileId,
      progressError: this.lastState.progressError } : { ...this.lastState }
    return { engineReady: Boolean(this.executable), connection: this.connection ? { ...this.connection } : undefined, state }
  }

  /** 先验证候选 Server，成功后才回收旧会话并切换来源。 */
  async connect(input: string): Promise<NativeLabConnection> {
    const origin = normalizeServerUrl(input)
    const health = await this.api<{ name: string }>(origin, "/api/health")
    if (!health.name?.startsWith("curated")) throw new Error("INVALID_CURATED_SERVER")
    const auth = await this.api<{ unlocked: boolean; pinEnabled: boolean }>(origin, "/api/auth/status")
    await this.stop()
    this.connection = { origin, unlocked: auth.unlocked === true, requiresPin: auth.pinEnabled === true }
    return { ...this.connection }
  }

  /** PIN 只发到当前 session，renderer 之外不持久化或记录凭据。 */
  async unlock(pin: string): Promise<NativeLabConnection> {
    if (!this.connection || typeof pin !== "string" || !/^\d{4,12}$/.test(pin)) throw new Error("INVALID_PIN")
    const result = await this.api<{ unlocked: boolean }>(this.connection.origin, "/api/auth/unlock", { method: "POST",
      body: JSON.stringify({ pin, trustedForever: false }) })
    this.connection.unlocked = result.unlocked === true
    return { ...this.connection }
  }

  /** 查询有限的候选影片，不下载封面，也不探测/创建播放流。 */
  async search(query: string): Promise<NativeLabMovie[]> {
    const origin = this.requireConnection()
    if (typeof query !== "string" || query.length > 200) throw new Error("INVALID_SEARCH")
    const result = await this.api<{ items: { id: string; code: string; title: string }[] }>(origin,
      `/api/library/movies?q=${encodeURIComponent(query)}&limit=30&offset=0`)
    return result.items.slice(0, 30).map((item) => { // 列表只投影允许进入原型的影片信息。
      return { id: item.id, code: item.code, title: item.title, files: [] }
    })
  }

  /** 详情提供稳定 fileId 和文件名，Server 磁盘路径不会返回给原型页面。 */
  async detail(movieId: string): Promise<NativeLabMovie> {
    this.validateId(movieId)
    const value = await this.api<{ id: string; code: string; title: string; files?: { id: string; fileName: string }[] }>(
      this.requireConnection(), `/api/library/movies/${encodeURIComponent(movieId)}`)
    return { id: value.id, code: value.code, title: value.title, files: (value.files ?? []).map((file) => {
      // 没有查逐文件进度前不猜测续播位置。
      return { id: file.id, fileName: file.fileName, resumePositionSec: 0 }
    }) }
  }

  /** 显式 direct descriptor 只读取选定文件的进度，不触发自动 HLS 决策。 */
  async resume(input: { movieId: string; fileId?: string }): Promise<number> {
    this.validateId(input.movieId)
    if (input.fileId) this.validateId(input.fileId)
    const descriptor = await this.api<{ mode: string; resumePositionSec?: number; durationSec?: number }>(this.requireConnection(),
      `/api/library/movies/${encodeURIComponent(input.movieId)}/playback-session${this.fileQuery(input.fileId)}`,
      { method: "POST", body: JSON.stringify({ mode: "direct" }) })
    if (descriptor.mode !== "direct") throw new Error("DIRECT_DESCRIPTOR_REQUIRED")
    const position = descriptor.resumePositionSec ?? 0
    return Number.isFinite(position) && position > 0 && !(descriptor.durationSec && position >= descriptor.durationSec * 0.95) ? position : 0
  }

  /** 新会话前校验文件归属，再回收旧进程；只启动原始媒体路径。 */
  async start(input: NativeLabStart): Promise<void> {
    if (!this.executable) throw new Error("MPV_REQUIRED")
    const origin = this.requireConnection()
    this.validateId(input.movieId)
    if (!Number.isFinite(input.startSec) || input.startSec < 0) throw new Error("INVALID_START")
    const detail = await this.detail(input.movieId)
    const fileId = input.fileId || detail.files[0]?.id
    if (fileId && !detail.files.some((file) => { // 只能播放该作品的真实文件。
      return file.id === fileId
    })) throw new Error("INVALID_MOVIE_FILE")
    await this.stop()
    this.lastState = emptyNativePlayerState()
    const proxy = new NativeMediaProxy(`${origin}/api/library/movies/${encodeURIComponent(input.movieId)}/stream${this.fileQuery(fileId)}`,
      this.fetchForOrigin(origin))
    const player = this.createPlayer()
    const active: ActivePlayback = { player, proxy, origin, movieId: input.movieId, fileId }
    this.active = active
    player.on("state", (state: NativePlayerState) => { // 终态回收只作用于仍归当前实例所有的会话。
      if (this.active === active && ["error", "ended", "stopped"].includes(state.status)) {
        void this.run(async () => { if (this.active === active) await this.stop() })
      }
    })
    try {
      const source = await proxy.start()
      await player.start(this.executable, source, input.startSec)
      this.timer = setInterval(() => { // 周期保存走同一队列，防止旧文件进度串到新片。
        if (this.periodicPending) return
        this.periodicPending = true
        void this.run(async () => {
          // 慢 Server 最多保留一条周期任务，不能每五秒无限追加队列。
          if (this.active !== active) return
          const auth = await this.api<{ unlocked: boolean }>(active.origin, "/api/auth/status").catch(() => {
            // 临时网络失败可继续消费已缓冲内容，下次检查重试。
            return undefined
          })
          if (auth && !auth.unlocked) {
            if (this.connection?.origin === active.origin) this.connection.unlocked = false
            await this.stop()
            this.lastState.status = "error"
            this.lastState.error = "SERVER_LOCKED"
            return
          }
          await this.save(active, { ...player.state })
        }).finally(() => { /* 完成后才允许下一条周期任务。 */ this.periodicPending = false })
      }, 5000)
    } catch {
      await this.stop()
      throw new Error(this.lastState.error ?? "NATIVE_START_FAILED")
    }
  }

  /** 控制经过显式枚举和范围校验；暂停及时保存当前文件进度。 */
  async control(input: NativeLabControl): Promise<void> {
    if (input.action === "stop") { await this.stop(); return }
    const active = this.active
    if (!active) throw new Error("NO_ACTIVE_PLAYER")
    await active.player.control(input)
    if (input.action === "pause") await this.save(active, { ...active.player.state })
  }

  /** 保存并释放本次播放；失败仍回收媒体资源。 */
  async stop(): Promise<void> {
    clearInterval(this.timer)
    this.timer = undefined
    const active = this.active
    this.active = undefined
    if (!active) return
    active.player.removeAllListeners("state")
    await active.player.stop()
    this.lastState = { ...active.player.state, movieId: active.movieId, fileId: active.fileId }
    await active.proxy.stop()
    await this.save(active, this.lastState)
  }

  /** 保持逐文件 query 与正文身份一致；不把未加载的零时长写进资料库。 */
  private async save(active: ActivePlayback, state: NativePlayerState): Promise<void> {
    if (state.durationSec <= 0 || state.positionSec < 0) return
    if (active.lastSaved?.positionSec === state.positionSec && active.lastSaved.durationSec === state.durationSec) return
    try {
      await this.api(active.origin, `/api/playback/progress/${encodeURIComponent(active.movieId)}${this.fileQuery(active.fileId)}`,
        { method: "PUT", body: JSON.stringify({ fileId: active.fileId,
          positionSec: Math.min(state.positionSec, state.durationSec), durationSec: state.durationSec }) })
      this.lastState.progressError = undefined
      active.lastSaved = { positionSec: state.positionSec, durationSec: state.durationSec }
    } catch { this.lastState.progressError = "PROGRESS_SAVE_FAILED" }
  }

  /** 认证/API 请求限制来源、重定向及期限，错误正文不向 renderer 原样透传。 */
  private async api<T>(origin: string, route: string, init: RequestInit = {}): Promise<T> {
    const response = await this.fetchForOrigin(origin)(`${origin}${route}`, { ...init,
      headers: { "Content-Type": "application/json" }, redirect: "error", signal: AbortSignal.timeout(12000) })
    if (!response.ok) {
      await response.body?.cancel()
      throw new Error(response.status === 401 || response.status === 403 ? "SERVER_LOCKED" : `SERVER_HTTP_${response.status}`)
    }
    if (response.status === 204) return undefined as T
    return await response.json() as T
  }

  /** 没有解锁的来源不能读取业务 API。 */
  private requireConnection(): string {
    if (!this.connection?.unlocked) throw new Error("SERVER_LOCKED")
    return this.connection.origin
  }

  /** 限制标识长度，URL 段仍独立编码以免形成额外路径或查询。 */
  private validateId(id: string): void {
    if (typeof id !== "string" || !id.trim() || id.length > 200) throw new Error("INVALID_MEDIA_ID")
  }

  /** 所有逐文件请求使用相同查询字段。 */
  private fileQuery(fileId?: string): string { return fileId ? `?fileId=${encodeURIComponent(fileId)}` : "" }
}
