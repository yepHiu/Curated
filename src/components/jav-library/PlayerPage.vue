<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import {
  ExternalLink,
  Info,
  Maximize2,
  Pause,
  PictureInPicture2,
  Play,
  SkipBack,
  SkipForward,
  Volume2,
  VolumeX,
  X,
} from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { HttpClientError } from "@/api/http-client"
import { api } from "@/api/endpoints"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { pushAppToast } from "@/composables/use-app-toast"
import {
  canPlayHlsNatively,
  loadHlsLibrary,
  type HlsInstance,
  type HlsLevel,
} from "@/lib/hls-player"
import { recordMoviePlayed } from "@/lib/played-movies-storage"
import { saveCuratedCaptureFromVideo } from "@/lib/curated-frames/save-capture"
import {
  getProgress,
  parseResumeSecondsFromQuery,
  saveProgress,
} from "@/lib/playback-progress-storage"
import type { PlaybackDescriptorDTO } from "@/api/types"
import {
  getPlayerAudioPrefs,
  savePlayerAudioPrefs,
} from "@/lib/player-volume-storage"
import { useLibraryService } from "@/services/library-service"

const props = withDefaults(
  defineProps<{
    movie: Movie
    /** 为 true 时在首帧可播后尝试自动播放（通常由路由 `?autoplay=1` 驱动） */
    autoplay?: boolean
  }>(),
  { autoplay: false },
)

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()
const playbackSeekBackwardStep = computed(() =>
  Math.max(1, Number(libraryService.playerSettings.value.seekBackwardStepSec ?? 10)),
)
const playbackSeekForwardStep = computed(() =>
  Math.max(1, Number(libraryService.playerSettings.value.seekForwardStepSec ?? 10)),
)

const videoRef = ref<HTMLVideoElement | null>(null)
const surfaceRef = ref<HTMLElement | null>(null)
let hlsInstance: HlsInstance | null = null
let detachHlsStatsListeners: (() => void) | null = null

type PlayerContextMenuState = {
  x: number
  y: number
}

type PlaybackStatsState = {
  audioBitrateKbps: number | null
  videoBitrateKbps: number | null
  width: number | null
  height: number | null
  fps: number | null
}

const playerContextMenu = ref<PlayerContextMenuState | null>(null)
const detailedStatsVisible = ref(false)
const playbackStats = ref<PlaybackStatsState>({
  audioBitrateKbps: null,
  videoBitrateKbps: null,
  width: null,
  height: null,
  fps: null,
})
let frameCallbackId: number | null = null
let fallbackFpsTimer: ReturnType<typeof setInterval> | null = null
let fpsSampleWindowStartAt = 0
let fpsSampleFrameCount = 0
let fallbackLastDecodedFrames = 0
let fallbackLastDecodedAt = 0

/** 每条片源只尝试一次入口自动播放，避免 canplay 重复触发 */
const autoplayConsumedForMovieId = ref<string | null>(null)
/** 每条片源只应用一次 URL / 本地续播 seek，避免重复跳转 */
const resumeAppliedForMovieId = ref<string | null>(null)

const PROGRESS_SAVE_INTERVAL_MS = 4000
let lastProgressSaveAt = 0

const playbackSrc = ref<string | null>(null)
const playbackDescriptor = ref<PlaybackDescriptorDTO | null>(null)
const playbackError = ref("")
const isPlaying = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const initialAudio = getPlayerAudioPrefs()
const volume = ref([initialAudio.volumePercent])
/** 与 video.muted 同步，用于 UI 与持久化（音量滑块为 0 时不使用静音标志） */
const playbackMuted = ref(initialAudio.muted)

/** 浏览器原生画中画（Document Picture-in-Picture 除外） */
const pipSupported = ref(false)
const isPipActive = ref(false)

function refreshPipSupport() {
  try {
    pipSupported.value =
      typeof document !== "undefined" &&
      document.pictureInPictureEnabled === true &&
      typeof HTMLVideoElement !== "undefined" &&
      typeof HTMLVideoElement.prototype.requestPictureInPicture === "function"
  } catch {
    pipSupported.value = false
  }
}

function syncPipActiveFromDocument() {
  const v = videoRef.value
  isPipActive.value = Boolean(v && document.pictureInPictureElement === v)
}

function onVideoEnterPictureInPicture() {
  isPipActive.value = true
}

function onVideoLeavePictureInPicture() {
  isPipActive.value = false
}

function onDocumentPictureInPictureChange() {
  syncPipActiveFromDocument()
}

const curatedShutterActive = ref(false)
const curatedPlusOne = ref(false)
const curatedCaptureError = ref("")
let curatedPlusOneTimer: ReturnType<typeof setTimeout> | null = null
let curatedShutterTimer: ReturnType<typeof setTimeout> | null = null

/** 播放中鼠标静止一段时间后隐藏控件与指针；移动鼠标恢复 */
const IDLE_HIDE_MS = 5000
const chromeVisible = ref(true)
let idleHideTimer: ReturnType<typeof setTimeout> | null = null

function clearIdleHideTimer() {
  if (idleHideTimer !== null) {
    clearTimeout(idleHideTimer)
    idleHideTimer = null
  }
}

function scheduleChromeIdleHide() {
  clearIdleHideTimer()
  if (!playbackSrc.value || !isPlaying.value) {
    chromeVisible.value = true
    return
  }
  idleHideTimer = window.setTimeout(() => {
    idleHideTimer = null
    chromeVisible.value = false
  }, IDLE_HIDE_MS)
}

function onChromePointerActivity() {
  chromeVisible.value = true
  scheduleChromeIdleHide()
}

function onChromePointerLeave() {
  clearIdleHideTimer()
  chromeVisible.value = true
}

const CHROME_LAYER_TRANSITION =
  "transition-opacity duration-300 ease-out motion-reduce:transition-none"

const chromeLayerVisibleClass = computed(() =>
  chromeVisible.value
    ? "pointer-events-auto opacity-100"
    : "pointer-events-none opacity-0",
)

const surfaceCursorClass = computed(() =>
  playbackSrc.value && isPlaying.value && !chromeVisible.value ? "cursor-none" : "",
)

/** 子层 cursor-pointer 会盖过父级 cursor-none，播放中隐藏 UI 时需一并关掉 */
const videoAreaCursorClass = computed(() => {
  if (!playbackSrc.value) return ""
  if (isPlaying.value && !chromeVisible.value) return "cursor-none"
  return "cursor-pointer"
})

watch(isPlaying, (playing) => {
  if (!playing) {
    clearIdleHideTimer()
    chromeVisible.value = true
  } else if (playbackSrc.value) {
    scheduleChromeIdleHide()
  }
})

watch(
  playbackSrc,
  async (src) => {
    closePlayerContextMenu()
    if (!src) {
      detailedStatsVisible.value = false
      resetPlaybackStats()
    }
    await syncVideoSource()
    if (!src) {
      clearIdleHideTimer()
      chromeVisible.value = true
      isPipActive.value = false
      return
    }
    await nextTick()
    refreshPipSupport()
    syncPipActiveFromDocument()
  },
  { immediate: true },
)

async function destroyHlsInstance() {
  detachHlsStatsListeners?.()
  detachHlsStatsListeners = null
  if (hlsInstance) {
    hlsInstance.destroy()
    hlsInstance = null
  }
}

async function syncVideoSource() {
  await nextTick()
  const v = videoRef.value
  const src = playbackSrc.value?.trim() ?? ""
  const mode = playbackDescriptor.value?.mode ?? "direct"
  await destroyHlsInstance()
  if (!v) return
  if (!src) {
    v.removeAttribute("src")
    v.load()
    return
  }
  if (mode === "hls") {
    if (canPlayHlsNatively(v)) {
      v.src = src
      refreshPlaybackStatsFromVideo()
      return
    }
    try {
      const Hls = await loadHlsLibrary()
      if (!playbackSrc.value || videoRef.value !== v || playbackDescriptor.value?.mode !== "hls") {
        return
      }
      if (!Hls.isSupported()) {
        playbackError.value = t("player.decodeError")
        return
      }
      const player = new Hls()
      bindHlsStats(player, Hls.Events)
      player.loadSource(src)
      player.attachMedia(v)
      hlsInstance = player
      return
    } catch {
      playbackError.value = t("player.errGeneric")
      return
    }
  }
  v.src = src
  refreshPlaybackStatsFromVideo()
}

function syncSrc() {
  void releasePlaybackSession(playbackDescriptor.value?.sessionId)
  playbackError.value = ""
  playbackDescriptor.value = null
  playbackSrc.value = null
  void loadPlayback()
}

async function loadPlayback() {
  const movieId = props.movie.id.trim()
  if (!movieId) {
    playbackDescriptor.value = null
    playbackSrc.value = null
    return
  }
  try {
    const descriptor = await libraryService.getMoviePlayback(movieId)
    if (movieId !== props.movie.id.trim()) {
      return
    }
    playbackDescriptor.value = descriptor
    playbackSrc.value = descriptor?.url?.trim() || null
  } catch {
    if (movieId !== props.movie.id.trim()) {
      return
    }
    playbackDescriptor.value = null
    playbackSrc.value = null
    playbackError.value = t("player.errGeneric")
  }
}

function flushPlaybackProgress() {
  const v = videoRef.value
  if (!v || !playbackSrc.value) return
  const durRaw = Number.isFinite(v.duration) && v.duration > 0 ? v.duration : duration.value
  const dur = Number.isFinite(durRaw) && durRaw > 0 ? durRaw : 0
  const pos = v.currentTime
  if (!Number.isFinite(pos) || pos < 0) return
  saveProgress(props.movie.id, pos, dur)
  recordMoviePlayed(props.movie.id)
}

function stripTFromRoute() {
  if (route.query.t === undefined || route.query.t === null || route.query.t === "") return
  const nextQuery = { ...route.query }
  delete nextQuery.t
  void router.replace({
    name: "player",
    params: { id: props.movie.id },
    query: nextQuery,
    hash: route.hash,
  })
}

watch(
  () => props.movie.id,
  async () => {
    autoplayConsumedForMovieId.value = null
    resumeAppliedForMovieId.value = null
    lastProgressSaveAt = 0
    syncSrc()
    await nextTick()
    await syncVideoSource()
    videoRef.value?.load()
  },
  { immediate: true },
)

function onVisibilityChange() {
  if (document.visibilityState === "hidden") {
    flushPlaybackProgress()
  }
}

function onWindowBeforeUnload() {
  flushPlaybackProgress()
}

onMounted(() => {
  syncSrc()
  refreshPipSupport()
  document.addEventListener("pictureinpicturechange", onDocumentPictureInPictureChange)
  window.addEventListener("keydown", onPlaybackKeydown)
  document.addEventListener("visibilitychange", onVisibilityChange)
  window.addEventListener("beforeunload", onWindowBeforeUnload)
})

onUnmounted(() => {
  flushPlaybackProgress()
  void releasePlaybackSession(playbackDescriptor.value?.sessionId)
  void destroyHlsInstance()
  document.removeEventListener("pictureinpicturechange", onDocumentPictureInPictureChange)
  window.removeEventListener("keydown", onPlaybackKeydown)
  document.removeEventListener("visibilitychange", onVisibilityChange)
  window.removeEventListener("beforeunload", onWindowBeforeUnload)
  clearIdleHideTimer()
  if (document.pictureInPictureElement) {
    void document.exitPictureInPicture().catch(() => {
      // ignore
    })
  }
  if (curatedPlusOneTimer) clearTimeout(curatedPlusOneTimer)
  if (curatedShutterTimer) clearTimeout(curatedShutterTimer)
  stopFpsTracking()
  closePlayerContextMenu()
})

function formatClock(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "00:00"
  const s = Math.floor(seconds % 60)
  const m = Math.floor(seconds / 60) % 60
  const h = Math.floor(seconds / 3600)
  const pad = (n: number) => String(n).padStart(2, "0")
  if (h > 0) return `${pad(h)}:${pad(m)}:${pad(s)}`
  return `${pad(m)}:${pad(s)}`
}

const progressPercent = computed(() => {
  if (!duration.value) return 0
  return (currentTime.value / duration.value) * 100
})

/** 实际记忆的音量档位（静音时仍保留，用于取消静音后恢复） */
const volumePercent = computed(() => volume.value[0] ?? 0)
/** 静音时滑块与百分比显示为 0，避免与「有声但静音」状态不一致 */
const volumeSliderDisplay = computed(() =>
  playbackMuted.value ? [0] : volume.value,
)
const volumePercentLabel = computed(() =>
  playbackMuted.value ? 0 : volumePercent.value,
)
const volumeIconIsMuted = computed(
  () => volumePercent.value <= 0 || playbackMuted.value,
)

function onTimeUpdate() {
  const v = videoRef.value
  if (!v) return
  currentTime.value = v.currentTime
  const now = Date.now()
  if (now - lastProgressSaveAt < PROGRESS_SAVE_INTERVAL_MS) return
  lastProgressSaveAt = now
  flushPlaybackProgress()
}

function onLoadedMetadata() {
  const v = videoRef.value
  if (!v) return
  duration.value = Number.isFinite(v.duration) ? v.duration : 0
  refreshPlaybackStatsFromVideo()
  startFpsTracking()
  const pct = volume.value[0] ?? 100
  v.volume = pct / 100
  v.muted = playbackMuted.value

  const dur = duration.value
  if (resumeAppliedForMovieId.value === props.movie.id) return
  if (dur <= 0) return

  const fromQuery = parseResumeSecondsFromQuery(route.query.t)
  const descriptorResume = playbackDescriptor.value?.resumePositionSec
  const stored = getProgress(props.movie.id)?.positionSec
  const targetSec = fromQuery !== undefined ? fromQuery : descriptorResume ?? stored
  if (targetSec === undefined) return

  const clamped = Math.min(Math.max(0, targetSec), Math.max(0, dur - 0.25))
  v.currentTime = clamped
  currentTime.value = v.currentTime
  resumeAppliedForMovieId.value = props.movie.id
  stripTFromRoute()
}

function stripAutoplayFromRoute() {
  if (route.query.autoplay !== "1") return
  const nextQuery = { ...route.query }
  delete nextQuery.autoplay
  void router.replace({
    name: "player",
    params: { id: props.movie.id },
    query: nextQuery,
    hash: route.hash,
  })
}

/** 从详情/资料库点「播放」进入本页时，在可播后自动 play；成功后去掉 ?autoplay=1 */
async function onCanPlayForAutoplay() {
  if (!props.autoplay || !playbackSrc.value) return
  if (autoplayConsumedForMovieId.value === props.movie.id) return
  const v = videoRef.value
  if (!v) return

  autoplayConsumedForMovieId.value = props.movie.id
  try {
    await v.play()
    stripAutoplayFromRoute()
  } catch {
    autoplayConsumedForMovieId.value = null
    playbackError.value = t("player.autoplayBlocked")
  }
}

function onPlay() {
  isPlaying.value = true
  startFpsTracking()
}

function onPause() {
  isPlaying.value = false
  flushPlaybackProgress()
}

function onVideoEnded() {
  flushPlaybackProgress()
  isPlaying.value = false
}

function onVideoError() {
  stopFpsTracking()
  const v = videoRef.value
  const code = v?.error?.code
  if (code === MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED) {
    playbackError.value = t("player.decodeError")
  } else if (code != null) {
    playbackError.value = t("player.errFileMissing")
  } else {
    playbackError.value = t("player.errGeneric")
  }
}

async function togglePlayPause() {
  const v = videoRef.value
  if (!v || !playbackSrc.value) return
  try {
    if (v.paused) {
      await v.play()
    } else {
      v.pause()
    }
  } catch {
    playbackError.value = t("player.playStartError")
  }
}

/** 左键单击画面区域（视频或上下黑边）切换播放/暂停 */
function onVideoSurfaceClick() {
  closePlayerContextMenu()
  if (!playbackSrc.value) return
  void togglePlayPause()
}

function closePlayerContextMenu() {
  playerContextMenu.value = null
}

function onPlayerContextMenu(event: MouseEvent) {
  event.preventDefault()
  event.stopPropagation()
  onChromePointerActivity()
  playerContextMenu.value = {
    x: event.clientX,
    y: event.clientY,
  }
}

function toggleDetailedStats() {
  if (!playbackSrc.value) return
  detailedStatsVisible.value = !detailedStatsVisible.value
  closePlayerContextMenu()
}

function closeDetailedStats() {
  detailedStatsVisible.value = false
}

function seekDelta(deltaSec: number) {
  const v = videoRef.value
  if (!v) return
  const maxT = Number.isFinite(v.duration) ? v.duration : 0
  v.currentTime = Math.max(0, Math.min(maxT, v.currentTime + deltaSec))
}

function onProgressBarClick(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const v = videoRef.value
  if (!v || !duration.value) return
  const rect = el.getBoundingClientRect()
  const ratio = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
  v.currentTime = ratio * duration.value
}

function onVolumeSlider(vols?: number[]) {
  if (!vols?.length) return
  const pct = Math.max(0, Math.min(100, Math.round(vols[0] ?? 100)))
  // 静音时滑块展示为 0，忽略同步产生的 [0]，避免把底层音量清零
  if (playbackMuted.value && pct === 0) {
    return
  }
  volume.value = [pct]
  if (pct === 0) {
    playbackMuted.value = false
  }
  const v = videoRef.value
  if (v) {
    v.volume = pct / 100
    if (v.volume > 0) {
      v.muted = false
      playbackMuted.value = false
    } else {
      v.muted = false
    }
  }
  savePlayerAudioPrefs({
    volumePercent: pct,
    muted: playbackMuted.value,
  })
}

function toggleMute() {
  const v = videoRef.value
  if (!v) return
  v.muted = !v.muted
  playbackMuted.value = v.muted
  savePlayerAudioPrefs({
    volumePercent: volume.value[0] ?? 100,
    muted: playbackMuted.value,
  })
}

function adjustVolume(delta: number) {
  const v = videoRef.value
  if (!v) return
  const cur = volume.value[0] ?? 100
  const next = Math.max(0, Math.min(100, Math.round(cur + delta)))
  onVolumeSlider([next])
}

function isTypingTarget(el: EventTarget | null): boolean {
  if (!(el instanceof HTMLElement)) return false
  const tag = el.tagName
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return true
  return el.isContentEditable
}

function onPlaybackKeydown(e: KeyboardEvent) {
  if (playerContextMenu.value && e.key === "Escape") {
    e.preventDefault()
    closePlayerContextMenu()
    return
  }
  if (!playbackSrc.value) return
  if (e.ctrlKey || e.metaKey || e.altKey) return
  if (isTypingTarget(e.target)) return

  /** 快捷键也视为活动，避免仅键盘操作时界面被永久隐藏 */
  onChromePointerActivity()

  switch (e.code) {
    case "Space":
    case "KeyK":
      e.preventDefault()
      void togglePlayPause()
      break
    case "ArrowLeft":
    case "KeyJ":
      e.preventDefault()
      seekDelta(-playbackSeekBackwardStep.value)
      break
    case "ArrowRight":
    case "KeyL":
      e.preventDefault()
      seekDelta(playbackSeekForwardStep.value)
      break
    case "KeyF":
      e.preventDefault()
      void toggleFullscreen()
      break
    case "KeyP":
      if (pipSupported.value) {
        e.preventDefault()
        void togglePictureInPicture()
      }
      break
    case "KeyM":
      e.preventDefault()
      toggleMute()
      break
    case "ArrowUp":
      e.preventDefault()
      adjustVolume(5)
      break
    case "ArrowDown":
      e.preventDefault()
      adjustVolume(-5)
      break
    case "Escape":
      if (document.fullscreenElement && surfaceRef.value && document.fullscreenElement === surfaceRef.value) {
        e.preventDefault()
        void document.exitFullscreen()
      }
      break
    case "KeyC":
      e.preventDefault()
      void runCuratedCapture()
      break
    default:
      break
  }
}

async function runCuratedCapture() {
  curatedCaptureError.value = ""
  const v = videoRef.value
  if (!v || !playbackSrc.value) {
    curatedCaptureError.value = t("player.captureNoVideo")
    return
  }
  if (curatedShutterTimer) clearTimeout(curatedShutterTimer)
  if (curatedPlusOneTimer) clearTimeout(curatedPlusOneTimer)

  curatedShutterActive.value = true
  curatedShutterTimer = window.setTimeout(() => {
    curatedShutterActive.value = false
    curatedShutterTimer = null
  }, 600)

  const result = await saveCuratedCaptureFromVideo(v, props.movie)
  if (!result.ok) {
    curatedCaptureError.value = result.reason
    curatedShutterActive.value = false
    return
  }

  curatedPlusOne.value = true
  curatedPlusOneTimer = window.setTimeout(() => {
    curatedPlusOne.value = false
    curatedPlusOneTimer = null
  }, 800)
}

async function toggleFullscreen() {
  const el = surfaceRef.value
  if (!el) return
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen()
    } else {
      await el.requestFullscreen()
    }
  } catch {
    // ignore
  }
}

async function togglePictureInPicture() {
  onChromePointerActivity()
  const v = videoRef.value
  if (!v || !playbackSrc.value || !pipSupported.value) return
  try {
    if (document.pictureInPictureElement === v) {
      await document.exitPictureInPicture()
    } else {
      await v.requestPictureInPicture()
    }
  } catch {
    // 需用户手势或编解码器不支持时可能失败，静默处理
  }
}

const fileBasename = computed(() => {
  const loc = props.movie.location?.trim() ?? ""
  if (!loc) return ""
  const parts = loc.split(/[/\\]/)
  return parts[parts.length - 1] || loc
})

const noStreamHint = computed(() => {
  void locale.value
  if (playbackSrc.value) return ""
  return import.meta.env.VITE_USE_WEB_API === "true" ? t("player.errNoSrc") : t("player.mockNoPlay")
})

function formatClientError(err: unknown, fallback: string): string {
  if (err instanceof HttpClientError) {
    return err.apiError?.message?.trim() || fallback
  }
  if (err instanceof Error) {
    return err.message.trim() || fallback
  }
  return fallback
}

async function openNativePlayer() {
  if (!playbackSrc.value) return
  try {
    const launched = await libraryService.launchNativePlayback(props.movie.id, currentTime.value)
    pushAppToast(launched?.message?.trim() || "Native player launched.", {
      variant: "success",
      durationMs: 3200,
    })
  } catch (err) {
    pushAppToast(formatClientError(err, "Could not launch the native player."), {
      variant: "destructive",
    })
  }
}

async function releasePlaybackSession(sessionId?: string) {
  const id = sessionId?.trim()
  if (!id) return
  try {
    await api.deletePlaybackSession(id)
  } catch {
    // ignore session cleanup failures
  }
}

function resetPlaybackStats() {
  playbackStats.value = {
    audioBitrateKbps: null,
    videoBitrateKbps: null,
    width: null,
    height: null,
    fps: null,
  }
  stopFpsTracking()
}

function toFiniteNumber(value: unknown): number | null {
  const num =
    typeof value === "number"
      ? value
      : typeof value === "string" && value.trim()
        ? Number(value)
        : NaN
  return Number.isFinite(num) ? num : null
}

function refreshPlaybackStatsFromVideo() {
  const v = videoRef.value
  if (!v) return
  const width = Number.isFinite(v.videoWidth) && v.videoWidth > 0 ? v.videoWidth : null
  const height = Number.isFinite(v.videoHeight) && v.videoHeight > 0 ? v.videoHeight : null
  playbackStats.value = {
    ...playbackStats.value,
    width,
    height,
  }
}

function updatePlaybackStatsFromHlsLevel(level?: HlsLevel | null) {
  if (!level) return
  const width = toFiniteNumber(level.width)
  const height = toFiniteNumber(level.height)
  const frameRate =
    toFiniteNumber(level.frameRate) ??
    toFiniteNumber(level.attrs?.["FRAME-RATE"]) ??
    toFiniteNumber(level.attrs?.FRAME_RATE)
  const videoBitrate = toFiniteNumber(level.bitrate)

  playbackStats.value = {
    ...playbackStats.value,
    width: width && width > 0 ? width : playbackStats.value.width,
    height: height && height > 0 ? height : playbackStats.value.height,
    fps: frameRate && frameRate > 0 ? frameRate : playbackStats.value.fps,
    videoBitrateKbps:
      videoBitrate && videoBitrate > 0
        ? Math.round(videoBitrate / 1000)
        : playbackStats.value.videoBitrateKbps,
  }
}

function bindHlsStats(player: HlsInstance, events?: Record<string, string>) {
  detachHlsStatsListeners?.()
  detachHlsStatsListeners = null
  const on = player.on
  if (!on || !events) return

  const updateCurrentLevelStats = () => {
    const levels = player.levels
    const currentLevel = player.currentLevel ?? -1
    if (!Array.isArray(levels) || currentLevel < 0 || currentLevel >= levels.length) return
    updatePlaybackStatsFromHlsLevel(levels[currentLevel] ?? null)
  }

  const listeners: Array<{ event: string; handler: (event: string, data?: unknown) => void }> = []
  const register = (eventName: string | undefined, handler: (event: string, data?: unknown) => void) => {
    if (!eventName) return
    on.call(player, eventName, handler)
    listeners.push({ event: eventName, handler })
  }

  register(events.MANIFEST_PARSED, () => {
    updateCurrentLevelStats()
  })
  register(events.LEVEL_SWITCHED, (_event, data) => {
    const levelIndex =
      typeof data === "object" && data !== null && "level" in data
        ? toFiniteNumber((data as { level?: unknown }).level)
        : null
    if (Array.isArray(player.levels) && levelIndex !== null && levelIndex >= 0) {
      updatePlaybackStatsFromHlsLevel(player.levels[levelIndex] ?? null)
      return
    }
    updateCurrentLevelStats()
  })
  register(events.FRAG_CHANGED, () => {
    updateCurrentLevelStats()
    refreshPlaybackStatsFromVideo()
  })

  detachHlsStatsListeners = () => {
    if (!player.off) return
    for (const listener of listeners) {
      player.off(listener.event, listener.handler)
    }
  }
}

function stopFpsTracking() {
  const v = videoRef.value
  if (frameCallbackId !== null && v && typeof v.cancelVideoFrameCallback === "function") {
    v.cancelVideoFrameCallback(frameCallbackId)
  }
  frameCallbackId = null
  if (fallbackFpsTimer !== null) {
    clearInterval(fallbackFpsTimer)
    fallbackFpsTimer = null
  }
  fpsSampleWindowStartAt = 0
  fpsSampleFrameCount = 0
  fallbackLastDecodedFrames = 0
  fallbackLastDecodedAt = 0
}

function startFpsTracking() {
  stopFpsTracking()
  const v = videoRef.value
  if (!v || !playbackSrc.value) return

  if (typeof v.requestVideoFrameCallback === "function") {
    const onFrame = (now: number) => {
      const currentVideo = videoRef.value
      if (!currentVideo || currentVideo !== v || !playbackSrc.value) return
      if (fpsSampleWindowStartAt === 0) {
        fpsSampleWindowStartAt = now
      }
      fpsSampleFrameCount += 1
      const elapsed = now - fpsSampleWindowStartAt
      if (elapsed >= 1000) {
        playbackStats.value = {
          ...playbackStats.value,
          fps: Number((fpsSampleFrameCount / (elapsed / 1000)).toFixed(2)),
        }
        fpsSampleWindowStartAt = now
        fpsSampleFrameCount = 0
      }
      frameCallbackId = currentVideo.requestVideoFrameCallback(onFrame)
    }
    frameCallbackId = v.requestVideoFrameCallback(onFrame)
    return
  }

  if (typeof v.getVideoPlaybackQuality !== "function") return

  fallbackFpsTimer = window.setInterval(() => {
    const currentVideo = videoRef.value
    if (!currentVideo || currentVideo !== v || !playbackSrc.value) return
    const quality = currentVideo.getVideoPlaybackQuality()
    const now = performance.now()
    const decodedFrames = quality.totalVideoFrames
    if (fallbackLastDecodedAt > 0) {
      const elapsed = now - fallbackLastDecodedAt
      const frameDelta = decodedFrames - fallbackLastDecodedFrames
      if (elapsed >= 500 && frameDelta >= 0) {
        playbackStats.value = {
          ...playbackStats.value,
          fps: Number((frameDelta / (elapsed / 1000)).toFixed(2)),
        }
      }
    }
    fallbackLastDecodedFrames = decodedFrames
    fallbackLastDecodedAt = now
  }, 1000)
}

function formatBitrateLabel(kbps: number | null): string {
  if (!kbps || kbps <= 0) return "暂不可用"
  if (kbps >= 1000) return `${(kbps / 1000).toFixed(2)} Mbps`
  return `${Math.round(kbps)} kbps`
}

function formatResolutionLabel(width: number | null, height: number | null): string {
  if (!width || !height) return "暂不可用"
  return `${width} × ${height}`
}

function formatFpsLabel(fps: number | null): string {
  if (!fps || fps <= 0) return "暂不可用"
  return `${fps.toFixed(fps >= 100 ? 0 : 2)} fps`
}

const playbackStatsRows = computed(() => [
  {
    key: "audio",
    label: "音频码率",
    value: formatBitrateLabel(playbackStats.value.audioBitrateKbps),
  },
  {
    key: "video",
    label: "视频码率",
    value: formatBitrateLabel(playbackStats.value.videoBitrateKbps),
  },
  {
    key: "resolution",
    label: "视频清晰度",
    value: formatResolutionLabel(playbackStats.value.width, playbackStats.value.height),
  },
  {
    key: "fps",
    label: "视频帧率",
    value: formatFpsLabel(playbackStats.value.fps),
  },
])

const detailedStatsModeLabel = computed(() => {
  const mode = playbackDescriptor.value?.mode
  if (mode === "hls") return "HLS"
  if (mode === "native") return "Native"
  if (mode === "direct") return "Direct"
  return "暂不可用"
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-col p-1 sm:p-2">
    <div
      ref="surfaceRef"
      class="relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.75rem] border border-border/50 bg-gradient-to-br from-black via-zinc-950 to-black"
      :class="surfaceCursorClass"
      @mousedown="onChromePointerActivity"
      @mousemove="onChromePointerActivity"
      @mouseenter="onChromePointerActivity"
      @mouseleave="onChromePointerLeave"
      @contextmenu="onPlayerContextMenu"
    >
      <div
        class="absolute inset-x-0 top-0 z-10 flex items-start justify-between gap-3 bg-gradient-to-b from-black/85 via-black/40 to-transparent p-4 sm:p-5"
        :class="[CHROME_LAYER_TRANSITION, chromeLayerVisibleClass]"
      >
        <div class="flex min-w-0 flex-col items-start gap-2 text-left">
          <Badge variant="secondary" class="rounded-full border border-white/20 bg-white/10 text-white">
            {{ movie.code }}
          </Badge>
          <div class="flex min-w-0 flex-col gap-1">
            <p class="text-lg font-semibold text-white sm:text-xl">{{ movie.title }}</p>
            <p v-if="fileBasename" class="truncate text-sm text-white/65" :title="movie.location">
              {{ fileBasename }}
            </p>
          </div>
        </div>
      </div>

      <div
        v-if="detailedStatsVisible"
        class="absolute left-4 top-20 z-[18] w-[min(22rem,calc(100%-2rem))] rounded-2xl border border-white/12 bg-black/48 p-3 text-white shadow-2xl backdrop-blur-md sm:left-5 sm:top-24 sm:p-4"
        @click.stop
      >
        <div class="mb-3 flex items-start justify-between gap-3">
          <div class="min-w-0">
            <p class="text-[11px] font-medium uppercase tracking-[0.22em] text-white/55">
              详细统计信息
            </p>
            <p class="mt-1 truncate text-sm font-semibold text-white/90">
              {{ movie.code }} · {{ detailedStatsModeLabel }}
            </p>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            class="size-8 shrink-0 rounded-full text-white/75 hover:bg-white/10 hover:text-white"
            aria-label="关闭详细统计信息"
            @click="closeDetailedStats"
          >
            <X class="size-4" aria-hidden="true" />
          </Button>
        </div>
        <dl class="grid gap-2.5">
          <div
            v-for="row in playbackStatsRows"
            :key="row.key"
            class="grid grid-cols-[7.25rem_minmax(0,1fr)] items-baseline gap-x-3 rounded-xl bg-white/[0.045] px-3 py-2"
          >
            <dt class="text-xs text-white/56">
              {{ row.label }}
            </dt>
            <dd class="truncate text-right text-sm font-medium text-white/92">
              {{ row.value }}
            </dd>
          </div>
        </dl>
      </div>

      <div
        class="relative flex min-h-0 flex-1 items-center justify-center p-4 sm:p-6 lg:p-8"
        :class="videoAreaCursorClass"
        @click="onVideoSurfaceClick"
      >
        <div
          class="pointer-events-none absolute inset-4 z-[5] rounded-2xl sm:inset-6 lg:inset-8"
          :class="curatedShutterActive ? 'curated-shutter-ring' : ''"
          aria-hidden="true"
        />
        <video
          v-if="playbackSrc"
          ref="videoRef"
          class="h-full max-h-full w-full max-w-full object-contain"
          :class="videoAreaCursorClass"
          playsinline
          preload="metadata"
          @click.stop="onVideoSurfaceClick"
          @timeupdate="onTimeUpdate"
          @loadedmetadata="onLoadedMetadata"
          @canplay="onCanPlayForAutoplay"
          @play="onPlay"
          @pause="onPause"
          @ended="onVideoEnded"
          @error="onVideoError"
          @enterpictureinpicture="onVideoEnterPictureInPicture"
          @leavepictureinpicture="onVideoLeavePictureInPicture"
        />

        <div
          v-else
          class="pointer-events-none flex max-w-lg flex-col items-center gap-3 px-4 text-center text-white/80"
        >
          <p class="text-lg font-semibold text-white">{{ t("player.noOnlineSrc") }}</p>
          <p class="text-sm text-white/65">
            {{ noStreamHint }}
          </p>
        </div>

        <div
          v-if="playbackError"
          class="pointer-events-auto absolute inset-x-6 bottom-24 z-20 rounded-2xl border border-destructive/40 bg-destructive/20 px-4 py-3 text-center text-sm text-destructive-foreground backdrop-blur-sm"
          @click.stop
        >
          {{ playbackError }}
        </div>
      </div>

      <div
        class="absolute inset-x-0 bottom-0 z-10 bg-gradient-to-t from-black/90 via-black/65 to-transparent p-4 sm:p-5"
        :class="[CHROME_LAYER_TRANSITION, chromeLayerVisibleClass]"
      >
        <div class="flex w-full flex-col gap-4">
          <div class="flex items-center justify-between gap-3 text-sm text-white/70">
            <span>{{ formatClock(currentTime) }}</span>
            <span>{{ duration > 0 ? formatClock(duration) : "\u2014" }}</span>
          </div>

          <div
            class="relative h-2.5 w-full cursor-pointer rounded-full bg-white/10"
            role="button"
            tabindex="0"
            :aria-label="t('player.progressAria')"
            @click="onProgressBarClick"
          >
            <div
              class="bg-primary pointer-events-none absolute inset-y-0 left-0 rounded-full transition-[width] duration-150"
              :style="{ width: `${progressPercent}%` }"
            />
          </div>

          <p
            v-if="curatedCaptureError"
            class="text-center text-xs text-amber-300/95"
          >
            {{ curatedCaptureError }}
          </p>

          <!-- 底栏：播放控制 | Curated（与音量同一行）| 音量 + 全屏 -->
          <div
            class="grid w-full items-center gap-x-3 gap-y-3 sm:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] sm:gap-x-4"
          >
            <div class="flex items-center justify-center gap-2 sm:justify-start">
              <Button
                variant="secondary"
                size="icon"
                class="rounded-full bg-white/10 text-white hover:bg-white/20"
                :disabled="!playbackSrc"
                @click="seekDelta(-playbackSeekBackwardStep)"
              >
                <SkipBack />
              </Button>
              <Button
                size="icon-lg"
                class="rounded-full"
                :disabled="!playbackSrc"
                @click="togglePlayPause"
              >
                <Pause v-if="isPlaying" />
                <Play v-else />
              </Button>
              <Button
                variant="secondary"
                size="icon"
                class="rounded-full bg-white/10 text-white hover:bg-white/20"
                :disabled="!playbackSrc"
                @click="seekDelta(playbackSeekForwardStep)"
              >
                <SkipForward />
              </Button>
            </div>

            <div class="relative flex justify-center justify-self-center">
              <div class="relative inline-flex items-center">
                <Transition
                  enter-active-class="transition duration-300 ease-out motion-reduce:transition-none"
                  enter-from-class="opacity-0 scale-95 motion-reduce:scale-100"
                  enter-to-class="opacity-100 scale-100"
                  leave-active-class="transition duration-400 ease-in motion-reduce:transition-none"
                  leave-from-class="opacity-100 scale-100"
                  leave-to-class="opacity-0 scale-95 motion-reduce:scale-100"
                >
                  <span
                    v-if="curatedPlusOne"
                    class="pointer-events-none absolute left-full ml-2 inline-block text-sm font-bold text-primary drop-shadow-md"
                  >
                    +1
                  </span>
                </Transition>
                <Button
                  type="button"
                  class="rounded-full border-0 bg-primary px-5 py-2 text-sm font-semibold tracking-wide text-primary-foreground hover:bg-primary/88 sm:px-6"
                  :disabled="!playbackSrc"
                  :aria-label="t('player.ariaCurated')"
                  @click="runCuratedCapture"
                >
                  {{ t("player.curatedLabel") }}
                </Button>
              </div>
            </div>

            <div
              class="flex flex-wrap items-center justify-center gap-3 sm:col-start-3 sm:justify-end"
            >
              <div
                class="flex h-9 min-w-[min(100%,14rem)] max-w-full flex-1 items-center gap-2 rounded-full bg-white/8 px-3 text-white/80 backdrop-blur sm:min-w-[14rem] sm:flex-initial sm:gap-3 sm:px-4"
                role="group"
                :aria-label="t('player.volumeAria')"
              >
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="size-9 shrink-0 rounded-full text-white hover:bg-white/15"
                  :disabled="!playbackSrc"
                  :aria-pressed="volumeIconIsMuted"
                  :aria-label="volumeIconIsMuted ? t('player.ariaUnmute') : t('player.ariaMute')"
                  @click="toggleMute"
                >
                  <VolumeX v-if="volumeIconIsMuted" class="size-5 shrink-0" aria-hidden="true" />
                  <Volume2 v-else class="size-5 shrink-0" aria-hidden="true" />
                </Button>
                <Slider
                  :model-value="volumeSliderDisplay"
                  :max="100"
                  :step="1"
                  class="flex-1"
                  :disabled="!playbackSrc"
                  @update:model-value="onVolumeSlider"
                />
                <span class="w-10 shrink-0 text-right text-sm leading-none tabular-nums">{{ volumePercentLabel }}%</span>
              </div>

              <Button
                variant="secondary"
                class="h-9 shrink-0 rounded-2xl bg-white/10 px-4 text-white hover:bg-white/20"
                :disabled="!playbackSrc"
                @click="openNativePlayer"
              >
                <ExternalLink class="size-4 shrink-0" data-icon="inline-start" aria-hidden="true" />
                Native
              </Button>

              <Button
                v-if="pipSupported"
                variant="secondary"
                class="h-9 shrink-0 rounded-2xl bg-white/10 px-4 text-white hover:bg-white/20"
                :disabled="!playbackSrc"
                :aria-pressed="isPipActive"
                :aria-label="isPipActive ? t('player.ariaPipExit') : t('player.ariaPipEnter')"
                @click="togglePictureInPicture"
              >
                <PictureInPicture2 class="size-4 shrink-0" data-icon="inline-start" aria-hidden="true" />
                {{ isPipActive ? t("player.pipExit") : t("player.pip") }}
              </Button>

              <Button
                variant="secondary"
                class="h-9 shrink-0 rounded-2xl bg-white/10 px-4 text-white hover:bg-white/20"
                :disabled="!playbackSrc"
                @click="toggleFullscreen"
              >
                <Maximize2 data-icon="inline-start" />
                {{ t("player.fullscreen") }}
              </Button>
            </div>
          </div>

          <p
            v-if="playbackSrc"
            class="text-center text-[10px] leading-relaxed text-white/40 sm:text-xs"
          >
            {{ t("player.hintBar", { backward: playbackSeekBackwardStep, forward: playbackSeekForwardStep }) }}
          </p>
        </div>
      </div>
    </div>
  </div>
  <Teleport to="body">
    <div
      v-if="playerContextMenu"
      class="fixed inset-0 z-[110]"
      @click="closePlayerContextMenu"
      @contextmenu.prevent
    >
      <div
        class="bg-popover text-popover-foreground fixed min-w-[12rem] rounded-md border p-1 shadow-md outline-none"
        :style="{ left: `${playerContextMenu.x}px`, top: `${playerContextMenu.y}px` }"
        @click.stop
      >
        <button
          type="button"
          role="menuitem"
          class="hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground relative flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm outline-hidden transition-colors disabled:pointer-events-none disabled:opacity-50"
          :disabled="!playbackSrc"
          @click="toggleDetailedStats"
        >
          <Info class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          {{ detailedStatsVisible ? "隐藏详细统计信息" : "详细统计信息" }}
        </button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.curated-shutter-ring {
  animation: curated-shutter-inset 0.55s ease-out forwards;
  box-shadow: inset 0 0 0 10px hsl(var(--primary) / 0.5);
}

@keyframes curated-shutter-inset {
  from {
    box-shadow: inset 0 0 0 14px hsl(var(--primary) / 0.55);
    opacity: 1;
  }
  to {
    box-shadow: inset 0 0 0 0 transparent;
    opacity: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .curated-shutter-ring {
    animation: curated-shutter-minimal 0.2s ease-out forwards;
  }
  @keyframes curated-shutter-minimal {
    from {
      opacity: 0.85;
    }
    to {
      opacity: 0;
    }
  }
}
</style>
