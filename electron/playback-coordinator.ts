import { randomUUID } from "node:crypto"
import { EventEmitter } from "node:events"
import { NativeMediaProxy, type MediaFetcher } from "./native-media-proxy.js"
import { NativeMpvPlayer } from "./native-mpv-player.js"
import { emptyNativePlayerState, type NativeLabMovie, type NativePlayerState } from "./native-player-contract.js"
import { projectPlaybackDiagnostics, playbackDiagnosticsReport } from "./native-playback-diagnostics.js"
import { createPlaybackWatchTimeTracker, type PlaybackWatchTimeTracker } from "./playback-watch-time-core.js"
import type { DesktopPlaybackCapture, DesktopPlaybackClip, DesktopPlaybackCommand, DesktopPlaybackOpen, DesktopPlaybackPreferences, DesktopPlaybackSnapshot, DesktopPlaybackSourceQuery } from "./playback-contract.js"

export interface PlaybackContext { origin: string; generation: string; fetch: MediaFetcher }
export interface PlaybackSurface {
  createPlayer(): NativeMpvPlayer
  setTitle(title: string): Promise<void>
  focus(): Promise<void>
  action(action: "fullscreen" | "minimize"): Promise<void>
  fullscreen(): boolean
  maximized?(): boolean
  dispose(): Promise<void>
}
interface Active {
  context: PlaybackContext
  player: NativeMpvPlayer
  proxy: NativeMediaProxy
  tracker: PlaybackWatchTimeTracker
  movie: NativeLabMovie
  fileId: string
  tracking: boolean
  locked: boolean
  saved?: string
}
function id(value: unknown): asserts value is string {
  if (typeof value !== "string" || !value.trim() || value.length > 200) throw new Error("INVALID_MEDIA_ID")
}
// 只保留现有 Curated 内部导航上下文；起点/引擎/文件由独立字段决定。
const sourceQueryKeys = new Set(["from", "browse", "back", "detailBack", "q", "tag", "actor", "studio", "tab", "playState",
  "userRating", "resolution", "addedWithinDays", "unrated", "year", "runtime", "catalog", "sort", "favorite", "selected", "cfq", "cft"])
function validateSourceQuery(value: unknown): DesktopPlaybackSourceQuery {
  if (value === undefined) return {}
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("INVALID_SOURCE_QUERY")
  const entries = Object.entries(value)
  if (entries.length > 32) throw new Error("INVALID_SOURCE_QUERY")
  const query: DesktopPlaybackSourceQuery = {}
  let size = 0
  for (const [key, raw] of entries) {
    if (!sourceQueryKeys.has(key)) continue
    const values = Array.isArray(raw) ? raw : [raw]
    if (values.length > 16 || values.some(item => item !== null && (typeof item !== "string" || item.length > 2048))) throw new Error("INVALID_SOURCE_QUERY")
    size += values.reduce((sum, item) => sum + (item?.length ?? 0), 0)
    if (size > 16384) throw new Error("INVALID_SOURCE_QUERY")
    query[key] = Array.isArray(raw) ? [...raw] : raw
  }
  return query
}
export function validatePlaybackOpen(value: unknown): DesktopPlaybackOpen {
  if (!value || typeof value !== "object") throw new Error("INVALID_PLAYBACK_REQUEST")
  const input = value as DesktopPlaybackOpen
  id(input.movieId)
  if (input.fileId !== undefined) id(input.fileId)
  if (input.startSec !== undefined && (!Number.isFinite(input.startSec) || input.startSec < 0 || input.startSec > 1e7)) throw new Error("INVALID_START")
  if (typeof input.autoplay !== "boolean" || input.autoAdvance !== undefined && typeof input.autoAdvance !== "boolean") throw new Error("INVALID_PLAYBACK_REQUEST")
  if (input.seekExisting !== undefined && typeof input.seekExisting !== "boolean") throw new Error("INVALID_PLAYBACK_REQUEST")
  if (input.locale !== undefined && !["zh-CN", "en-US", "ja-JP"].includes(input.locale)) throw new Error("INVALID_LOCALE")
  if (input.queue !== undefined) {
    if (!Array.isArray(input.queue) || input.queue.length > 5000) throw new Error("INVALID_QUEUE")
    input.queue.forEach(id)
  }
  return { movieId: input.movieId, fileId: input.fileId, startSec: input.startSec, autoplay: input.autoplay,
    seekExisting: input.seekExisting === true, autoAdvance: input.autoAdvance === true, queue: [...new Set(input.queue ?? [])], locale: input.locale ?? "zh-CN",
    sourceQuery: validateSourceQuery(input.sourceQuery) }
}

/** 正式本机播放的唯一所有者：身份、HTTP 写入和资源替换全部串行。 */
export class NativePlaybackCoordinator extends EventEmitter {
  private tail: Promise<unknown> = Promise.resolve()
  private active?: Active
  private lastContext?: PlaybackContext
  private surface?: PlaybackSurface
  private timer?: ReturnType<typeof setInterval>
  private periodicPending = false
  private capturing = false
  private captureCandidate?: { sessionId: string; image: Buffer; result: DesktopPlaybackCapture; actors: string[]; title: string; gifEndSec?: number }
  private captureExpiry?: ReturnType<typeof setTimeout>
  private clipJob?: { active: Active; sessionId: string; state: DesktopPlaybackClip; taskId?: string; started: number; failures: number; timer?: ReturnType<typeof setTimeout> }
  private snapshotState: DesktopPlaybackSnapshot = { sessionId: "", revision: 0, engine: "native", windowOpen: false,
    state: emptyNativePlayerState(), fullscreen: false, queue: [], autoAdvance: false, locale: "zh-CN" }

  constructor(private readonly executable: string,
    private readonly createSurface: () => Promise<PlaybackSurface>,
    private readonly preferences: () => DesktopPlaybackPreferences,
    private readonly savePreferences: (value: DesktopPlaybackPreferences) => void) { super() }

  run<T>(operation: () => Promise<T>): Promise<T> {
    const result = this.tail.then(operation)
    this.tail = result.catch(() => {})
    return result
  }
  snapshot(): DesktopPlaybackSnapshot {
    return structuredClone({ ...this.snapshotState, fullscreen: this.surface?.fullscreen() ?? false, maximized: this.surface?.maximized?.() ?? false })
  }
  /** 原生拖动/Snap 也可改变窗口状态，只在状态变化时刷新 UI。 */
  windowStateChanged(): void {
    const next = this.snapshot()
    if (this.snapshotState.fullscreen === next.fullscreen && this.snapshotState.maximized === next.maximized) return
    this.snapshotState.fullscreen = next.fullscreen
    this.snapshotState.maximized = next.maximized
    this.publish()
  }
  private publish(): void { this.snapshotState.revision++; this.emit("snapshot", this.snapshot()) }

  open(context: PlaybackContext, value: unknown): Promise<DesktopPlaybackSnapshot> {
    const input = validatePlaybackOpen(value)
    return this.run(async () => {
      const active = this.active
      if (active && active.context.generation === context.generation && active.movie.id === input.movieId
        && (!input.fileId || input.fileId === active.fileId)) {
        await this.ensureUnlocked(context)
        if (input.seekExisting && input.startSec !== undefined) {
          this.clearPreparedCapture()
          active.tracker.onSeeking(input.startSec)
          await active.player.control({ action: "seek", value: Math.min(input.startSec, active.player.state.durationSec || input.startSec) })
          if (input.autoplay) await active.player.control({ action: "resume" })
        }
        await this.surface?.focus()
        return this.snapshot()
      }
      await this.start(context, input)
      return this.snapshot()
    })
  }

  private async api<T>(context: PlaybackContext, route: string, init: RequestInit = {}): Promise<T> {
    const response = await context.fetch(`${context.origin}${route}`, { ...init, headers: init.body instanceof FormData ? undefined : { "Content-Type": "application/json" },
      redirect: "error", signal: AbortSignal.timeout(5000) })
    if (!response.ok) {
      await response.body?.cancel()
      throw new Error(response.status === 401 || response.status === 403 ? "SERVER_LOCKED" : `SERVER_HTTP_${response.status}`)
    }
    return response.status === 204 ? undefined as T : await response.json() as T
  }
  private async detail(context: PlaybackContext, movieId: string): Promise<NativeLabMovie> {
    const detail = await this.api<{ id: string; code: string; title: string; actors?: string[]; files: { id: string; fileName: string }[] }>(
      context, `/api/library/movies/${encodeURIComponent(movieId)}`)
    return { id: movieId, code: detail.code, title: detail.title, actors: (detail.actors ?? []).filter(actor => typeof actor === "string"), files: (detail.files ?? []).map(file => ({
      id: file.id, fileName: file.fileName, resumePositionSec: 0,
    })) }
  }
  private async ensureUnlocked(context: PlaybackContext): Promise<void> {
    try {
      const auth = await this.api<{ unlocked: boolean }>(context, "/api/auth/status")
      if (!auth.unlocked) throw new Error("SERVER_LOCKED")
    } catch (error) {
      if (error instanceof Error && error.message === "SERVER_LOCKED") await this.finish("SERVER_LOCKED")
      throw error
    }
  }
  private async start(context: PlaybackContext, input: DesktopPlaybackOpen): Promise<void> {
    await this.ensureUnlocked(context)
    const movie = await this.detail(context, input.movieId).catch(async error => {
      if (error instanceof Error && error.message === "SERVER_LOCKED") await this.finish("SERVER_LOCKED")
      throw error
    })
    if (input.fileId && !movie.files.some(file => file.id === input.fileId)) throw new Error("INVALID_MOVIE_FILE")
    await this.release()
    try {
    const descriptor = await this.api<{ mode: string; fileId?: string; resumePositionSec?: number; durationSec?: number }>(
      context, `/api/library/movies/${encodeURIComponent(input.movieId)}/playback-session${input.fileId ? `?fileId=${encodeURIComponent(input.fileId)}` : ""}`,
      { method: "POST", body: JSON.stringify({ mode: "direct" }) })
    const fileId = input.fileId ?? descriptor.fileId ?? movie.files[0]?.id
    if (descriptor.mode !== "direct" || !fileId || !movie.files.some(file => file.id === fileId)) throw new Error("INVALID_DIRECT_DESCRIPTOR")
    const resume = descriptor.resumePositionSec ?? 0
    const startSec = input.startSec ?? (resume > 0 && !(descriptor.durationSec && resume >= descriptor.durationSec * 0.95) ? resume : 0)
    this.snapshotState = { ...this.snapshotState, sessionId: randomUUID(), state: { ...emptyNativePlayerState(), status: "starting",
      movieId: movie.id, fileId }, movie, windowOpen: true, queue: input.queue ?? [], autoAdvance: input.autoAdvance === true,
      locale: input.locale ?? "zh-CN", sourceQuery: structuredClone(input.sourceQuery ?? {}) }
    this.lastContext = context
      this.surface ??= await this.createSurface()
      await this.surface.setTitle(movie.title?.trim() || movie.code || "Curated")
      const proxy = new NativeMediaProxy(`${context.origin}/api/library/movies/${encodeURIComponent(movie.id)}/stream?fileId=${encodeURIComponent(fileId)}`, async (url, init) => {
        const response = await context.fetch(url, init)
        if (response.status === 401 || response.status === 403) {
          active.locked = true
          void this.run(async () => { if (this.active === active) await this.finish("SERVER_LOCKED") })
        }
        return response
      })
      const player = this.surface.createPlayer()
      const tracker = createPlaybackWatchTimeTracker({ movieId: movie.id, addDelta: async (movieId, dayKey, watchedSec) => {
        await this.api(context, "/api/playback/watch-time/daily", { method: "POST", body: JSON.stringify({ movieId, dayKey, watchedSec }) })
      } })
      const active: Active = { context, player, proxy, tracker, movie, fileId, tracking: false, locked: false }
      this.active = active
      player.on("state", (state: NativePlayerState) => this.receive(active, state))
      proxy.on("diagnostics", () => {
        if (this.active !== active) return
        this.snapshotState.state.diagnostics = { ...(this.snapshotState.state.diagnostics ?? projectPlaybackDiagnostics({})), network: proxy.diagnostics() }
        this.publish()
      })
      this.publish()
      // 媒体先暂停加载，在音量/倍速偏好应用完成前不能短暂发声或推进。
      await player.start(this.executable, await proxy.start(), Math.max(0, startSec), false, true)
      const prefs = this.preferences()
      await player.control({ action: "volume", value: prefs.volume })
      await player.control({ action: "speed", value: prefs.speed })
      await player.control({ action: input.autoplay ? "resume" : "pause" })
      await this.api(context, `/api/library/played-movies/${encodeURIComponent(movie.id)}`, { method: "POST" })
      this.receive(active, player.state)
      await this.surface.focus()
      this.timer = setInterval(() => {
        if (this.periodicPending) return
        this.periodicPending = true
        void this.run(async () => {
          if (this.active !== active) return
          try { await this.ensureUnlocked(context) }
          catch { return } // 临时网络错误保留播放；401/403 已停止并发布锁定状态。
          await this.save(active)
        }).finally(() => { this.periodicPending = false })
      }, 5000)
    } catch (error) {
      const locked = this.active?.locked
      await this.release()
      await this.closeSurface()
      this.snapshotState.state.status = "error"
      this.snapshotState.state.error = locked ? "SERVER_LOCKED" : error instanceof Error ? error.message : "NATIVE_START_FAILED"
      this.publish()
      throw new Error(this.snapshotState.state.error)
    }
  }
  private receive(active: Active, state: NativePlayerState): void {
    if (active !== this.active) return
    if (state.status === "playing") {
      if (!active.tracking) active.tracker.onPlay(state.positionSec)
      else active.tracker.onTimeUpdate(state.positionSec)
      active.tracking = true
    } else if (active.tracking) { active.tracker.onPause(state.positionSec); active.tracking = false }
    this.snapshotState.state = { ...state, movieId: active.movie.id, fileId: active.fileId,
      diagnostics: { ...(state.diagnostics ?? projectPlaybackDiagnostics({})), network: active.proxy.diagnostics() },
      progressError: this.snapshotState.state.progressError }
    this.publish()
    if (["ended", "error", "stopped"].includes(state.status)) {
      void this.run(async () => {
        if (this.active !== active) return
        if (state.status === "ended" && this.snapshotState.autoAdvance) {
          const part = active.movie.files.findIndex(file => file.id === active.fileId) + 1
          const nextFile = active.movie.files[part]?.id
          const index = this.snapshotState.queue.indexOf(active.movie.id)
          const nextMovie = index < 0 ? undefined : this.snapshotState.queue[index + 1]
          if (nextFile || nextMovie) {
            try { await this.start(active.context, { movieId: nextFile ? active.movie.id : nextMovie!, fileId: nextFile,
              autoplay: true, queue: this.snapshotState.queue, autoAdvance: true, locale: this.snapshotState.locale, sourceQuery: this.snapshotState.sourceQuery }) }
            catch (error) {
              // 详情预检失败发生在 start 替换资源之前，EOF 仍须释放旧资源。
              if (this.active === active) await this.finish(error instanceof Error ? error.message : "NATIVE_START_FAILED")
            }
            return
          }
        }
        await this.release()
        this.publish()
      })
    }
  }
  /** 只读取当前播放信息；旧会话不能导出新影片的快照。 */
  diagnosticsReport(sessionId: string): string {
    if (!sessionId || sessionId !== this.snapshotState.sessionId) throw new Error("STALE_PLAYBACK_SESSION")
    if (!this.snapshotState.windowOpen) throw new Error("NO_ACTIVE_PLAYER")
    return playbackDiagnosticsReport(this.snapshotState.state)
  }
  command(sessionId: string, input: DesktopPlaybackCommand): Promise<void> {
    return this.run(async () => {
      if (!sessionId || sessionId !== this.snapshotState.sessionId) throw new Error("STALE_PLAYBACK_SESSION")
      if (!input || typeof input !== "object") throw new Error("INVALID_CONTROL")
      if (input.action === "stop" || input.action === "close") { await this.finish(); return }
      if (input.action === "focus") { await this.surface?.focus(); return }
      if (input.action === "fullscreen" || input.action === "minimize") {
        await this.surface?.action(input.action); this.publish(); return
      }
      if (input.action === "autoAdvance") {
        if (typeof input.enabled !== "boolean") throw new Error("INVALID_CONTROL")
        this.snapshotState.autoAdvance = input.enabled; this.publish(); return
      }
      const active = this.active
      const context = active?.context ?? this.lastContext
      if (input.action === "replay") {
        if (!context || !this.snapshotState.movie) throw new Error("NO_ACTIVE_PLAYER")
        await this.start(context, { movieId: this.snapshotState.movie.id, fileId: this.snapshotState.state.fileId,
          startSec: 0, autoplay: true, queue: this.snapshotState.queue, autoAdvance: this.snapshotState.autoAdvance, locale: this.snapshotState.locale, sourceQuery: this.snapshotState.sourceQuery })
        return
      }
      if (input.action === "web") {
        if (!this.snapshotState.movie) throw new Error("NO_ACTIVE_PLAYER")
        const fallback = { movieId: this.snapshotState.movie.id, fileId: this.snapshotState.state.fileId, startSec: this.snapshotState.state.positionSec }
        await this.finish()
        this.emit("web-fallback", fallback)
        return
      }
      if (input.action === "part" || input.action === "movie") {
        if (!context || !this.snapshotState.movie) throw new Error("NO_ACTIVE_PLAYER")
        const movieId = input.action === "part" ? this.snapshotState.movie.id : input.movieId
        id(movieId)
        if (input.action === "part") id(input.fileId)
        if (input.action === "movie" && !this.snapshotState.queue.includes(movieId)) throw new Error("INVALID_QUEUE_TARGET")
        await this.start(context, { movieId, fileId: input.action === "part" ? input.fileId : undefined, autoplay: true,
          queue: this.snapshotState.queue, autoAdvance: this.snapshotState.autoAdvance, locale: this.snapshotState.locale, sourceQuery: this.snapshotState.sourceQuery })
        return
      }
      if (!active) throw new Error("NO_ACTIVE_PLAYER")
      if (!["pause", "resume", "seek", "speed", "volume"].includes(input.action)) throw new Error("INVALID_CONTROL")
      if (input.action === "seek") { this.clearPreparedCapture(); active.tracker.onSeeking(input.value ?? active.player.state.positionSec) }
      await active.player.control(input)
      if (input.action === "speed" || input.action === "volume") {
        this.savePreferences({ ...this.preferences(), [input.action]: input.value })
      }
      this.receive(active, active.player.state)
      if (input.action === "pause") await this.save(active)
    })
  }
  /** 采集和重试均绑定原会话/分部；重试上传原图和同一 ID，避免重复入库。 */
  capture(sessionId: string, retryId?: string, mode: "save" | "prepare" | "gif" = "save"): Promise<DesktopPlaybackCapture> {
    if (this.capturing) return Promise.reject(new Error("CAPTURE_BUSY"))
    this.capturing = true
    return this.run(async () => {
      if (!sessionId || sessionId !== this.snapshotState.sessionId) throw new Error("STALE_PLAYBACK_SESSION")
      const active = this.active
      if (!active) throw new Error("NO_ACTIVE_PLAYER")
      await this.ensureUnlocked(active.context)
      let candidate = this.captureCandidate
      if (retryId !== undefined) {
        if (!candidate || candidate.sessionId !== sessionId || candidate.result.id !== retryId || !["prepared", "error"].includes(candidate.result.phase)) throw new Error("STALE_CAPTURE")
      } else {
        if (this.clipJob?.state.phase === "processing") throw new Error("CLIP_BUSY")
        this.snapshotState.clip = undefined
        const frame = await active.player.captureFrame()
        candidate = { sessionId, image: frame.image, actors: [...(active.movie.actors ?? [])], title: active.movie.title,
          result: { id: randomUUID(), movieId: active.movie.id, fileId: active.fileId, code: active.movie.code,
            positionSec: frame.positionSec, capturedAt: frame.capturedAt, preview: "", phase: mode === "prepare" ? "prepared" : "error" } }
        this.captureCandidate = candidate
      }
      const { image, result } = candidate!
      clearTimeout(this.captureExpiry)
      if (mode === "prepare") {
        this.captureExpiry = setTimeout(() => { if (this.captureCandidate === candidate) this.captureCandidate = undefined }, 60_000)
        return { ...result, preview: `data:image/png;base64,${image.toString("base64")}` }
      }
      if (mode === "gif" && candidate!.gifEndSec === undefined) {
        // 略低于上限，避免 JSON 浮点相减得到 6.00000000000001 被 Server 拒绝。
        const end = Math.min(await active.player.playbackPosition(), result.positionSec + 6 - 0.000001, active.player.state.durationSec || Infinity)
        // 暂停/缓冲期间未看到足够媒体内容时，只保存静态帧。
        if (end - result.positionSec >= 0.4) candidate!.gifEndSec = end
      }
      try {
        const form = new FormData()
        form.append("metadata", JSON.stringify({ id: result.id, movieId: result.movieId, fileId: result.fileId, code: result.code,
          title: candidate!.title, actors: candidate!.actors, positionSec: result.positionSec, capturedAt: result.capturedAt, tags: [] }))
        form.append("image", new Blob([new Uint8Array(image)], { type: "image/png" }), "frame.png")
        await this.api(active.context, "/api/curated-frames", { method: "POST", body: form })
        result.phase = "saved"; result.error = undefined
        this.snapshotState.captureRevision = (this.snapshotState.captureRevision ?? 0) + 1
        this.publish()
      } catch (error) {
        if (error instanceof Error && error.message === "SERVER_LOCKED") { await this.finish("SERVER_LOCKED"); throw error }
        result.phase = "error"; result.error = "CAPTURE_SAVE_FAILED"
      }
      if (result.phase === "saved") {
        this.captureCandidate = undefined
        if (candidate!.gifEndSec !== undefined) await this.startClip(active, result.id, result.positionSec, candidate!.gifEndSec)
      }
      return { ...result, preview: `data:image/png;base64,${image.toString("base64")}` }
    }).finally(() => { this.capturing = false })
  }
  private clearPreparedCapture(): void {
    if (this.captureCandidate?.result.phase === "prepared") { clearTimeout(this.captureExpiry); this.captureCandidate = undefined }
  }
  discardCapture(sessionId: string, captureId: string): Promise<void> {
    return this.run(async () => {
      if (this.captureCandidate?.sessionId === sessionId && this.captureCandidate.result.id === captureId) this.clearPreparedCapture()
    })
  }
  private async startClip(active: Active, frameId: string, startSec: number, endSec: number): Promise<void> {
    const job: NonNullable<typeof this.clipJob> = { active, sessionId: this.snapshotState.sessionId, started: performance.now(), failures: 0,
      state: { frameId, startSec, endSec, phase: "processing", progress: 0 } }
    this.clipJob = job
    this.snapshotState.clip = job.state
    this.publish()
    try {
      const task = await this.api<{ taskId: string }>(active.context, `/api/library/movies/${encodeURIComponent(active.movie.id)}/clips?fileId=${encodeURIComponent(active.fileId)}`,
        { method: "POST", body: JSON.stringify({ fileId: active.fileId, format: "gif", startSec, endSec, fps: 10, width: 640, curatedFrameId: frameId }) })
      id(task.taskId)
      job.taskId = task.taskId
      this.scheduleClip(job, 0)
    } catch (error) {
      if (error instanceof Error && error.message === "SERVER_LOCKED") { await this.finish("SERVER_LOCKED"); throw error }
      job.state.phase = "error"; job.state.error = "CLIP_FAILED"; this.publish()
    }
  }
  private scheduleClip(job: NonNullable<typeof this.clipJob>, delay: number): void {
    clearTimeout(job.timer)
    job.timer = setTimeout(() => { void this.run(() => this.pollClip(job)).catch(() => {}) }, Math.max(0, Math.min(delay, 150_000 - (performance.now() - job.started))))
  }
  private async pollClip(job: NonNullable<typeof this.clipJob>): Promise<void> {
    if (this.clipJob !== job || this.active !== job.active || job.state.phase !== "processing") return
    if (performance.now() - job.started >= 150_000) {
      job.state.phase = "error"; job.state.error = "CLIP_TIMED_OUT"
      await this.cancelClipTask(job); this.publish(); return
    }
    try {
      const task = await this.api<{ status: string; progress?: number; metadata?: { artifactUrl?: unknown } }>(job.active.context, `/api/tasks/${encodeURIComponent(job.taskId!)}`)
      job.failures = 0
      job.state.error = undefined
      job.state.progress = typeof task.progress === "number" && Number.isFinite(task.progress) ? Math.min(100, Math.max(0, task.progress)) : 0
      if (task.status === "completed") {
        const artifact = task.metadata?.artifactUrl
        const ready = typeof artifact === "string" && artifact === `/api/curated-frames/${encodeURIComponent(job.state.frameId)}/motion`
        job.state.phase = ready ? "saved" : "error"
        job.state.error = ready ? undefined : "CLIP_FAILED"
        if (ready) this.snapshotState.captureRevision = (this.snapshotState.captureRevision ?? 0) + 1
      } else if (["failed", "partial_failed", "cancelled"].includes(task.status)) {
        job.state.phase = task.status === "cancelled" ? "cancelled" : "error"; job.state.error = task.status === "cancelled" ? undefined : "CLIP_FAILED"
      } else if (["pending", "running"].includes(task.status)) this.scheduleClip(job, 1000)
      else { job.state.phase = "error"; job.state.error = "CLIP_FAILED" }
      this.publish()
    } catch (error) {
      if (error instanceof Error && error.message === "SERVER_LOCKED") { await this.finish("SERVER_LOCKED"); return }
      if (++job.failures >= 5) { job.state.phase = "error"; job.state.error = "CLIP_STATUS_UNAVAILABLE"; await this.cancelClipTask(job); this.publish() }
      else this.scheduleClip(job, Math.min(30_000, 1000 * 2 ** job.failures))
    }
  }
  private async cancelClipTask(job: NonNullable<typeof this.clipJob>): Promise<string | undefined> {
    clearTimeout(job.timer)
    if (!job.taskId) return
    try { await this.api(job.active.context, `/api/tasks/${encodeURIComponent(job.taskId)}/clip`, { method: "DELETE" }) }
    catch (error) { return error instanceof Error ? error.message : "CLIP_CANCEL_FAILED" }
  }
  cancelClip(sessionId: string): Promise<void> {
    return this.run(async () => {
      if (sessionId !== this.snapshotState.sessionId) throw new Error("STALE_PLAYBACK_SESSION")
      const job = this.clipJob
      if (!job || job.state.phase !== "processing") return
      const failure = await this.cancelClipTask(job)
      if (failure === "SERVER_LOCKED") { await this.finish("SERVER_LOCKED"); throw new Error(failure) }
      if (failure) {
        job.state.error = "CLIP_CANCEL_FAILED"; this.scheduleClip(job, 1000); this.publish()
        throw new Error("CLIP_CANCEL_FAILED")
      }
      job.state.phase = "cancelled"; job.state.error = undefined; this.publish()
    })
  }
  retryClip(sessionId: string): Promise<void> {
    return this.run(async () => {
      if (sessionId !== this.snapshotState.sessionId) throw new Error("STALE_PLAYBACK_SESSION")
      const job = this.clipJob
      if (!job || job.state.phase !== "error" || this.active !== job.active) throw new Error("STALE_CLIP")
      await this.ensureUnlocked(job.active.context)
      await this.startClip(job.active, job.state.frameId, job.state.startSec, job.state.endSec)
    })
  }
  private async save(active: Active): Promise<void> {
    const state = active.player.state
    try {
      if (state.durationSec > 0 && state.positionSec >= 0) {
        const body = JSON.stringify({ fileId: active.fileId, positionSec: Math.min(state.positionSec, state.durationSec), durationSec: state.durationSec })
        if (body !== active.saved) {
          await this.api(active.context, `/api/playback/progress/${encodeURIComponent(active.movie.id)}?fileId=${encodeURIComponent(active.fileId)}`, { method: "PUT", body })
          active.saved = body
        }
      }
      await active.tracker.flush()
      this.snapshotState.state.progressError = undefined
    } catch { this.snapshotState.state.progressError = "PROGRESS_SAVE_FAILED" }
    this.publish()
  }
  private async release(): Promise<void> {
    clearTimeout(this.captureExpiry)
    if (this.clipJob?.state.phase === "processing") await this.cancelClipTask(this.clipJob)
    this.clipJob = undefined; this.snapshotState.clip = undefined
    this.captureCandidate = undefined
    clearInterval(this.timer); this.timer = undefined
    const active = this.active
    this.active = undefined
    if (!active) return
    active.player.removeAllListeners("state")
    active.proxy.removeAllListeners("diagnostics")
    active.tracker.onPause(active.player.state.positionSec)
    await active.player.stop()
    await active.proxy.stop()
    this.snapshotState.state = { ...active.player.state, movieId: active.movie.id, fileId: active.fileId,
      diagnostics: { ...(active.player.state.diagnostics ?? projectPlaybackDiagnostics({})), network: active.proxy.diagnostics() } }
    await this.save(active)
  }
  private async closeSurface(): Promise<void> {
    const surface = this.surface; this.surface = undefined
    await surface?.dispose()
    this.snapshotState.windowOpen = false
    this.lastContext = undefined
  }
  private async finish(error?: string): Promise<void> {
    await this.release()
    await this.closeSurface()
    if (error) { this.snapshotState.state.status = "error"; this.snapshotState.state.error = error }
    this.publish()
  }
  stop(error?: string): Promise<void> { return this.run(() => this.finish(error)) }
}
