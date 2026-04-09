<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import {
  ExternalLink,
  Info,
  Loader2,
  Maximize2,
  Pause,
  PictureInPicture2,
  Play,
  SkipBack,
  SkipForward,
  Volume2,
  VolumeX,
} from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { HttpClientError } from "@/api/http-client"
import { api } from "@/api/endpoints"
import { moviePlaybackAbsoluteUrl } from "@/api/playback-url"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { pushAppToast } from "@/composables/use-app-toast"
import {
  canPlayHlsNatively,
  loadHlsLibrary,
  prewarmHlsResources,
  preloadHlsLibrary,
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
import {
  descriptorMatchesRequestedPlaybackTarget,
  resolveHlsLocalSeekTargetSec,
  resolvePreferredPlaybackTargetSec,
} from "@/lib/playback-targets"
import type { PlaybackDescriptorDTO } from "@/api/types"
import { cn } from "@/lib/utils"
import {
  getPlayerAudioPrefs,
  savePlayerAudioPrefs,
} from "@/lib/player-volume-storage"
import {
  buildNativePlayerLaunchUrl,
  looksLikeBrowserProtocolLaunchTarget,
  normalizeNativePlayerPresetForBrowserLaunch,
  resolveNativePlayerBrowserTemplate,
} from "@/lib/native-player-launch"
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
  currentBitrateKbps: number | null
  bandwidthEstimateKbps: number | null
  width: number | null
  height: number | null
  fps: number | null
}

const playerContextMenu = ref<PlayerContextMenuState | null>(null)
const detailedStatsVisible = ref(false)
const playbackStats = ref<PlaybackStatsState>({
  audioBitrateKbps: null,
  videoBitrateKbps: null,
  currentBitrateKbps: null,
  bandwidthEstimateKbps: null,
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
let playbackLoadSeq = 0
let hlsDirectFallbackInFlight = false
let playbackFallbackNoticeKey = ""
let playbackSessionCleanupId: string | null = null
let resumePlaybackWhenReady = false
let hlsPrewarmSeq = 0

const playbackSrc = ref<string | null>(null)
const playbackDescriptor = ref<PlaybackDescriptorDTO | null>(null)
const playbackError = ref("")
const isResolvingPlayback = ref(false)
const isSwitchingPlaybackSession = ref(false)
const isPlaybackWaiting = ref(false)
const isPrewarmingHls = ref(false)
const hlsPrewarmProgress = ref(0)
const isPlaying = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const bufferedUntilSec = ref(0)
const progressSliderValue = ref([0])
const isScrubbingProgress = ref(false)
const scrubPreviewTimeSec = ref<number | null>(null)
const optimisticSeekTargetSec = ref<number | null>(null)
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
  if (mode === "direct" && playbackDescriptor.value?.canDirectPlay === false) {
    playbackError.value = t("player.decodeError")
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
        await fallbackHlsToDirect("hls.js unsupported in this browser")
        return
      }
      const player = new Hls({
        startFragPrefetch: true,
        enableWorker: true,
        lowLatencyMode: false,
        maxBufferLength: 30,
        maxMaxBufferLength: 60,
        backBufferLength: 90,
      })
      bindHlsStats(player, Hls.Events)
      player.loadSource(src)
      player.attachMedia(v)
      hlsInstance = player
      return
    } catch {
      await fallbackHlsToDirect("failed to initialize hls.js")
      return
    }
  }
  v.src = src
  refreshPlaybackStatsFromVideo()
}

async function prewarmHlsDescriptor(descriptor: PlaybackDescriptorDTO | null) {
  const seq = ++hlsPrewarmSeq
  const shouldPrewarm = descriptor?.mode === "hls" && Boolean(descriptor.url?.trim())
  if (!shouldPrewarm) {
    isPrewarmingHls.value = false
    hlsPrewarmProgress.value = 0
    return
  }
  isPrewarmingHls.value = true
  hlsPrewarmProgress.value = 0.06
  preloadHlsLibrary()
  try {
    await prewarmHlsResources(descriptor.url.trim(), {
      resourceCount: 2,
      timeoutMs: 3500,
      onProgress: (progress) => {
        if (seq !== hlsPrewarmSeq) return
        hlsPrewarmProgress.value = progress
      },
    })
  } catch {
    // Best-effort only. Playback startup still proceeds normally.
  } finally {
    if (seq === hlsPrewarmSeq) {
      isPrewarmingHls.value = false
      hlsPrewarmProgress.value = 0
    }
  }
}

function canBrowserDirectPlayFromFileName(fileName?: string | null): boolean {
  const normalized = (fileName ?? "").trim().toLowerCase()
  return [".mp4", ".m4v", ".webm", ".ogv", ".m3u8"].some((ext) => normalized.endsWith(ext))
}

async function tryStartPlaybackIfRequested(): Promise<boolean> {
  const v = videoRef.value
  if (!v || !playbackSrc.value) return false

  const shouldHandleRouteAutoplay =
    props.autoplay && autoplayConsumedForMovieId.value !== props.movie.id
  const shouldResumePlayback = resumePlaybackWhenReady
  if (!shouldHandleRouteAutoplay && !shouldResumePlayback) {
    return false
  }

  if (v.readyState < HTMLMediaElement.HAVE_CURRENT_DATA) {
    isPlaybackWaiting.value = true
    return false
  }

  if (shouldHandleRouteAutoplay) {
    autoplayConsumedForMovieId.value = props.movie.id
  }
  resumePlaybackWhenReady = false
  playbackError.value = ""

  try {
    await v.play()
    if (shouldHandleRouteAutoplay) {
      stripAutoplayFromRoute()
    }
    return true
  } catch {
    if (shouldResumePlayback) {
      resumePlaybackWhenReady = true
    }
    if (shouldHandleRouteAutoplay) {
      autoplayConsumedForMovieId.value = null
    }
    if (v.readyState < HTMLMediaElement.HAVE_FUTURE_DATA) {
      isPlaybackWaiting.value = true
      return false
    }
    playbackError.value = shouldHandleRouteAutoplay
      ? t("player.autoplayBlocked")
      : t("player.playStartError")
    return false
  }
}

function syncSrc() {
  flushScheduledPlaybackSessionCleanup()
  void releasePlaybackSession(playbackDescriptor.value?.sessionId)
  playbackError.value = ""
  isResolvingPlayback.value = true
  isPlaybackWaiting.value = false
  isPrewarmingHls.value = false
  hlsPrewarmProgress.value = 0
  optimisticSeekTargetSec.value = null
  currentTime.value = 0
  duration.value = 0
  bufferedUntilSec.value = 0
  progressSliderValue.value = [0]
  isScrubbingProgress.value = false
  scrubPreviewTimeSec.value = null
  playbackDescriptor.value = null
  playbackSrc.value = null
  void loadPlayback()
}

async function loadPlayback() {
  const seq = ++playbackLoadSeq
  const movieId = props.movie.id.trim()
  if (!movieId) {
    if (seq === playbackLoadSeq) {
      isResolvingPlayback.value = false
    }
    playbackDescriptor.value = null
    playbackSrc.value = null
    return
  }
  try {
    let descriptor = await libraryService.getMoviePlayback(movieId)
    const requestedStartSec = parseResumeSecondsFromQuery(route.query.t)
    if (
      descriptor?.mode === "hls" &&
      requestedStartSec !== undefined &&
      !descriptorMatchesRequestedPlaybackTarget(requestedStartSec, descriptor)
    ) {
      const reseekedDescriptor = await libraryService.createPlaybackSession(
        movieId,
        "hls",
        Math.max(0, requestedStartSec),
      )
      if (descriptor.sessionId) {
        await releasePlaybackSession(descriptor.sessionId)
      }
      descriptor = reseekedDescriptor
    }
    if (movieId !== props.movie.id.trim() || seq !== playbackLoadSeq) {
      if (descriptor?.sessionId) {
        await releasePlaybackSession(descriptor.sessionId)
      }
      return
    }
    playbackDescriptor.value = descriptor
    playbackSrc.value = descriptor?.url?.trim() || null
    void prewarmHlsDescriptor(descriptor)
    duration.value = resolveTotalDurationSec(descriptor, 0)
    currentTime.value = playbackTimelineOffsetSec(descriptor)
    const fallbackReason = descriptor?.reason?.trim() || ""
    const fallbackKey = `${movieId}:${descriptor?.mode ?? ""}:${fallbackReason}`
    if (
      fallbackReason &&
      descriptor?.mode === "direct" &&
      fallbackReason.toLowerCase().includes("fell back to direct playback") &&
      playbackFallbackNoticeKey !== fallbackKey
    ) {
      playbackFallbackNoticeKey = fallbackKey
      pushAppToast(fallbackReason, { variant: "warning", durationMs: 5200 })
    }
  } catch {
    if (movieId !== props.movie.id.trim() || seq !== playbackLoadSeq) {
      return
    }
    playbackDescriptor.value = null
    playbackSrc.value = null
    playbackError.value = t("player.errGeneric")
  } finally {
    if (seq === playbackLoadSeq) {
      isResolvingPlayback.value = false
    }
  }
}

async function fallbackHlsToDirect(reason?: string) {
  if (hlsDirectFallbackInFlight) return
  const current = playbackDescriptor.value
  const movieId = props.movie.id.trim()
  if (!current || current.mode !== "hls" || !movieId) return

  hlsDirectFallbackInFlight = true
  try {
    ++hlsPrewarmSeq
    isPrewarmingHls.value = false
    hlsPrewarmProgress.value = 0
    bufferedUntilSec.value = 0
    markPlaybackReady()
    const absolutePositionSec = getAbsolutePlaybackTime()
    const shouldResumePlayback = isPlaying.value && !videoRef.value?.paused
    const fallbackUrl = moviePlaybackAbsoluteUrl(movieId)
    await destroyHlsInstance()
    schedulePlaybackSessionCleanup(current.sessionId)
    resumeAppliedForMovieId.value = null
    if (shouldResumePlayback) {
      resumePlaybackWhenReady = true
    }
    playbackDescriptor.value = {
      ...current,
      mode: "direct",
      sessionId: undefined,
      url: fallbackUrl,
      mimeType: "video/mp4",
      startPositionSec: undefined,
      resumePositionSec: absolutePositionSec,
      canDirectPlay: canBrowserDirectPlayFromFileName(current.fileName),
      reason: reason?.trim() || "hls fallback to direct playback",
    }
    playbackSrc.value = fallbackUrl
    currentTime.value = absolutePositionSec
    playbackError.value = ""
    pushAppToast(reason?.trim() || t("player.hlsFallbackToDirect"), { variant: "warning", durationMs: 5200 })
  } finally {
    hlsDirectFallbackInFlight = false
  }
}

function flushPlaybackProgress() {
  const v = videoRef.value
  if (!v || !playbackSrc.value) return
  const durRaw = resolveTotalDurationSec(playbackDescriptor.value, Number.isFinite(v.duration) ? v.duration : 0)
  const dur = Number.isFinite(durRaw) && durRaw > 0 ? durRaw : 0
  const pos = getAbsolutePlaybackTime()
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
  refreshPipSupport()
  const probe = document.createElement("video")
  if (!canPlayHlsNatively(probe)) {
    preloadHlsLibrary()
  }
  document.addEventListener("pictureinpicturechange", onDocumentPictureInPictureChange)
  window.addEventListener("keydown", onPlaybackKeydown)
  document.addEventListener("visibilitychange", onVisibilityChange)
  window.addEventListener("beforeunload", onWindowBeforeUnload)
})

onUnmounted(() => {
  flushPlaybackProgress()
  flushScheduledPlaybackSessionCleanup()
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

function getDescriptorDurationSec(descriptor: PlaybackDescriptorDTO | null = playbackDescriptor.value): number {
  const raw = descriptor?.durationSec
  return raw != null && Number.isFinite(raw) && raw > 0 ? raw : 0
}

function resolveTotalDurationSec(
  descriptor: PlaybackDescriptorDTO | null = playbackDescriptor.value,
  mediaDurationSec: number = duration.value,
): number {
  const finiteMediaDuration =
    Number.isFinite(mediaDurationSec) && mediaDurationSec > 0 ? mediaDurationSec : 0
  return Math.max(finiteMediaDuration, getDescriptorDurationSec(descriptor))
}

const totalDurationSec = computed(() => resolveTotalDurationSec())

const displayedCurrentTimeSec = computed(() =>
  isScrubbingProgress.value && scrubPreviewTimeSec.value != null
    ? scrubPreviewTimeSec.value
    : currentTime.value,
)

const progressPercent = computed(() => {
  const total = totalDurationSec.value
  if (!total) return 0
  return Math.max(0, Math.min(100, (currentTime.value / total) * 100))
})

const playedTrackPercent = computed(() => {
  const total = totalDurationSec.value
  if (!total) return 0
  const playedSec = normalizeProgressTargetSec(progressSliderValue.value[0] ?? currentTime.value)
  return Math.max(0, Math.min(100, (playedSec / total) * 100))
})

const bufferedTrackPercent = computed(() => {
  const total = totalDurationSec.value
  if (!total) return 0
  return Math.max(playedTrackPercent.value, Math.min(100, (bufferedUntilSec.value / total) * 100))
})

const bufferedAheadStyle = computed(() => {
  const left = playedTrackPercent.value
  const width = Math.max(0, bufferedTrackPercent.value - left)
  return {
    left: `${left}%`,
    width: `${width}%`,
  }
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
  currentTime.value = getAbsolutePlaybackTime(v.currentTime)
  syncBufferedRangeFromVideo()
  clearOptimisticSeekIfSettled(currentTime.value)
  if (v.readyState >= HTMLMediaElement.HAVE_FUTURE_DATA) {
    isPlaybackWaiting.value = false
  }
  const now = Date.now()
  if (now - lastProgressSaveAt < PROGRESS_SAVE_INTERVAL_MS) return
  lastProgressSaveAt = now
  flushPlaybackProgress()
}

function onLoadedMetadata() {
  const v = videoRef.value
  if (!v) return
  markPlaybackReady()
  const videoDuration = Number.isFinite(v.duration) ? v.duration : 0
  duration.value = resolveTotalDurationSec(playbackDescriptor.value, videoDuration)
  refreshPlaybackStatsFromVideo()
  startFpsTracking()
  const pct = volume.value[0] ?? 100
  v.volume = pct / 100
  v.muted = playbackMuted.value
  syncBufferedRangeFromVideo()
  flushScheduledPlaybackSessionCleanup()

  const fromQuery = parseResumeSecondsFromQuery(route.query.t)
  const targetSec = resolvePreferredPlaybackTargetSec(
    fromQuery,
    playbackDescriptor.value,
    getProgress(props.movie.id)?.positionSec,
  )

  if (playbackDescriptor.value?.mode === "hls") {
    if (resumeAppliedForMovieId.value !== props.movie.id && targetSec !== undefined) {
      const localSeekTarget = resolveHlsLocalSeekTargetSec(
        targetSec,
        playbackDescriptor.value?.startPositionSec,
      )
      if (
        localSeekTarget !== undefined &&
        Math.abs((Number.isFinite(v.currentTime) ? v.currentTime : 0) - localSeekTarget) > 0.25
      ) {
        v.currentTime = localSeekTarget
      }
      currentTime.value = getAbsolutePlaybackTime(localSeekTarget ?? v.currentTime)
      resumeAppliedForMovieId.value = props.movie.id
    } else {
      currentTime.value = getAbsolutePlaybackTime(v.currentTime)
      resumeAppliedForMovieId.value = props.movie.id
    }
    if (fromQuery !== undefined) {
      stripTFromRoute()
    }
    void tryStartPlaybackIfRequested()
    return
  }

  const dur = totalDurationSec.value
  if (resumeAppliedForMovieId.value === props.movie.id) return
  if (dur <= 0) return

  if (targetSec === undefined) return

  const clamped = Math.min(Math.max(0, targetSec), Math.max(0, dur - 0.25))
  v.currentTime = clamped
  currentTime.value = clamped
  resumeAppliedForMovieId.value = props.movie.id
  stripTFromRoute()
  void tryStartPlaybackIfRequested()
}

function normalizeProgressTargetSec(rawValue: number): number {
  const normalized = Number.isFinite(rawValue) ? rawValue : 0
  const total = totalDurationSec.value
  if (total <= 0) {
    return Math.max(0, normalized)
  }
  return Math.min(Math.max(0, normalized), total)
}

function syncBufferedRangeFromVideo() {
  const v = videoRef.value
  const descriptor = playbackDescriptor.value
  const total = totalDurationSec.value
  if (!v || !descriptor || !playbackSrc.value || total <= 0) {
    bufferedUntilSec.value = 0
    return
  }

  const ranges = v.buffered
  if (!ranges || ranges.length <= 0) {
    bufferedUntilSec.value = 0
    return
  }

  const absoluteCurrent = getAbsolutePlaybackTime(v.currentTime, descriptor)
  let matchedEndSec = 0
  let nearestAheadStartSec = Number.POSITIVE_INFINITY
  let nearestAheadEndSec = 0

  for (let index = 0; index < ranges.length; index += 1) {
    const absoluteStart = getAbsolutePlaybackTime(ranges.start(index), descriptor)
    const absoluteEnd = getAbsolutePlaybackTime(ranges.end(index), descriptor)
    if (absoluteCurrent >= absoluteStart - 0.35 && absoluteCurrent <= absoluteEnd + 0.35) {
      matchedEndSec = Math.max(matchedEndSec, absoluteEnd)
      continue
    }
    if (absoluteStart > absoluteCurrent && absoluteStart < nearestAheadStartSec) {
      nearestAheadStartSec = absoluteStart
      nearestAheadEndSec = absoluteEnd
    }
  }

  const nextBufferedEnd = matchedEndSec || nearestAheadEndSec || 0
  bufferedUntilSec.value = Math.min(total, Math.max(0, nextBufferedEnd))
}

function syncProgressSliderFromPlayback() {
  if (isScrubbingProgress.value) return
  progressSliderValue.value = [normalizeProgressTargetSec(currentTime.value)]
}

function clearOptimisticSeekIfSettled(absoluteTimeSec: number = currentTime.value) {
  const target = optimisticSeekTargetSec.value
  if (target == null) return
  if (Math.abs(absoluteTimeSec - target) <= 1) {
    optimisticSeekTargetSec.value = null
  }
}

function markPlaybackReady() {
  isPlaybackWaiting.value = false
  clearOptimisticSeekIfSettled()
}

function startOptimisticSeek(targetSec: number) {
  const clamped = clampAbsolutePlaybackTarget(targetSec)
  optimisticSeekTargetSec.value = clamped
  currentTime.value = clamped
  progressSliderValue.value = [clamped]
  isPlaybackWaiting.value = true
}

function onProgressSliderInput(values?: number[]) {
  onChromePointerActivity()
  const next = normalizeProgressTargetSec(values?.[0] ?? 0)
  isScrubbingProgress.value = true
  scrubPreviewTimeSec.value = next
  progressSliderValue.value = [next]
}

function onProgressSliderCommit(values?: number[]) {
  const next = normalizeProgressTargetSec(values?.[0] ?? 0)
  const previous = currentTime.value
  progressSliderValue.value = [next]
  isScrubbingProgress.value = false
  scrubPreviewTimeSec.value = null
  startOptimisticSeek(next)
  void seekToAbsolutePlaybackTime(next, { previousDisplayedTimeSec: previous })
}

watch([currentTime, totalDurationSec, isScrubbingProgress], () => {
  syncProgressSliderFromPlayback()
}, { immediate: true })

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
  markPlaybackReady()
  syncBufferedRangeFromVideo()
  void tryStartPlaybackIfRequested()
}

function onPlay() {
  isPlaying.value = true
  markPlaybackReady()
  startFpsTracking()
}

function onPause() {
  isPlaying.value = false
  flushPlaybackProgress()
}

async function terminateActiveHlsPlaybackSession(reason?: string) {
  const descriptor = playbackDescriptor.value
  const sessionId = descriptor?.sessionId?.trim()
  if (!descriptor || descriptor.mode !== "hls" || !sessionId) return

  await destroyHlsInstance()
  markPlaybackReady()
  if (playbackDescriptor.value?.sessionId === sessionId) {
    playbackDescriptor.value = {
      ...playbackDescriptor.value,
      sessionId: undefined,
      reason: reason?.trim() || playbackDescriptor.value.reason,
    }
  }
  await releasePlaybackSession(sessionId)
}

function onVideoEnded() {
  flushPlaybackProgress()
  isPlaying.value = false
  markPlaybackReady()
  void terminateActiveHlsPlaybackSession("HLS session closed after playback ended")
}

function onVideoError() {
  stopFpsTracking()
  optimisticSeekTargetSec.value = null
  isPlaybackWaiting.value = false
  const v = videoRef.value
  if (playbackDescriptor.value?.mode === "hls") {
    void fallbackHlsToDirect("video element failed while playing HLS")
    return
  }
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
  const descriptor = playbackDescriptor.value
  if (!v || !playbackSrc.value || !descriptor) return
  try {
    if (v.paused) {
      if (descriptor.mode === "direct" && descriptor.canDirectPlay === false) {
        playbackError.value = t("player.decodeError")
        return
      }
      if (
        descriptor.mode === "hls" &&
        (!descriptor.sessionId || currentTime.value >= Math.max(0, totalDurationSec.value - 0.15))
      ) {
        const restartAtSec =
          currentTime.value >= Math.max(0, totalDurationSec.value - 0.15) ? 0 : currentTime.value
        void seekToAbsolutePlaybackTime(restartAtSec, {
          forceSessionSwap: true,
          resumeAfterSwap: true,
        })
        return
      }
      autoplayConsumedForMovieId.value = props.movie.id
      resumePlaybackWhenReady = true
      const started = await tryStartPlaybackIfRequested()
      if (!started && resumePlaybackWhenReady) {
        isPlaybackWaiting.value = true
      }
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
  const previous = currentTime.value
  startOptimisticSeek(previous + deltaSec)
  void seekToAbsolutePlaybackTime(previous + deltaSec, { previousDisplayedTimeSec: previous })
}

function onVideoWaiting() {
  if (!playbackSrc.value) return
  isPlaybackWaiting.value = true
}

function onVideoSeeking() {
  if (!playbackSrc.value) return
  isPlaybackWaiting.value = true
}

function onVideoSeeked() {
  currentTime.value = getAbsolutePlaybackTime()
  syncBufferedRangeFromVideo()
  markPlaybackReady()
}

function onVideoLoadedData() {
  syncBufferedRangeFromVideo()
  markPlaybackReady()
  void tryStartPlaybackIfRequested()
}

function onVideoProgress() {
  syncBufferedRangeFromVideo()
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

  const result = await saveCuratedCaptureFromVideo(v, props.movie, {
    positionSecOverride: getAbsolutePlaybackTime(v.currentTime),
  })
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
  if (isResolvingPlayback.value) return t("common.loading")
  if (playbackSrc.value) return ""
  return import.meta.env.VITE_USE_WEB_API === "true" ? t("player.errNoSrc") : t("player.mockNoPlay")
})

const nativePlayerPreset = computed(() =>
  normalizeNativePlayerPresetForBrowserLaunch(
    libraryService.playerSettings.value.nativePlayerPreset,
    libraryService.playerSettings.value.nativePlayerCommand,
  ),
)

const nativePlayerLabel = computed(() => {
  switch (nativePlayerPreset.value) {
    case "potplayer":
      return "PotPlayer"
    case "mpv":
      return "MPV"
    case "custom":
    default:
      return t("player.externalPlayer")
  }
})

const playbackBusyLabel = computed(() => {
  if (!playbackSrc.value && !isResolvingPlayback.value) return ""
  if (isSwitchingPlaybackSession.value) return t("player.preparingSeek")
  if (isResolvingPlayback.value) return t("player.preparingPlayback")
  if (isPlaybackWaiting.value && optimisticSeekTargetSec.value != null) return t("player.bufferingSeek")
  if (isPlaybackWaiting.value) return t("player.buffering")
  if (isPrewarmingHls.value && playbackDescriptor.value?.mode === "hls" && !isPlaying.value) {
    return t("player.prewarmingStream")
  }
  return ""
})

const showPlaybackBusyState = computed(() => Boolean(playbackBusyLabel.value))
const showCenteredBusyOverlay = computed(() => showPlaybackBusyState.value)
const prewarmProgressPercent = computed(() =>
  Math.max(0, Math.min(100, Math.round(hlsPrewarmProgress.value * 100))),
)

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
  if (libraryService.playerSettings.value.nativePlayerEnabled === false) {
    pushAppToast(t("player.nativeLaunchDisabled", { player: nativePlayerLabel.value }), {
      variant: "warning",
      durationMs: 4200,
    })
    return
  }
  const template = resolveNativePlayerBrowserTemplate(nativePlayerPreset.value)
  if (!template.trim()) {
    pushAppToast(t("player.nativeLaunchTemplateMissing", { player: nativePlayerLabel.value }), {
      variant: "warning",
      durationMs: 5200,
    })
    return
  }

  const launchTarget = buildNativePlayerLaunchUrl(template, {
    url: moviePlaybackAbsoluteUrl(props.movie.id),
    path: props.movie.location,
    movieId: props.movie.id,
    code: props.movie.code,
    startSec: currentTime.value,
    startMs: Math.round(currentTime.value * 1000),
  })

  if (!looksLikeBrowserProtocolLaunchTarget(launchTarget)) {
    pushAppToast(t("player.nativeLaunchTargetInvalid", { player: nativePlayerLabel.value }), {
      variant: "destructive",
      durationMs: 5200,
    })
    return
  }

  let pageHidden = false
  const onVisibilityChange = () => {
    if (document.visibilityState === "hidden") {
      pageHidden = true
    }
  }

  try {
    document.addEventListener("visibilitychange", onVisibilityChange)
    const anchor = document.createElement("a")
    anchor.href = launchTarget
    anchor.rel = "noopener"
    anchor.style.display = "none"
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    void terminateActiveHlsPlaybackSession("HLS session closed after browser native-player handoff")
    pushAppToast(t("player.nativeLaunchRequested", { player: nativePlayerLabel.value }), {
      variant: "success",
      durationMs: 3200,
    })
    window.setTimeout(() => {
      if (!pageHidden && document.visibilityState === "visible") {
        pushAppToast(t("player.nativeLaunchStillHere", { player: nativePlayerLabel.value }), {
          variant: "warning",
          durationMs: 5600,
        })
      }
    }, 1400)
  } catch (err) {
    pushAppToast(
      formatClientError(err, t("player.nativeLaunchFailed", { player: nativePlayerLabel.value })),
      {
        variant: "destructive",
      },
    )
  } finally {
    window.setTimeout(() => {
      document.removeEventListener("visibilitychange", onVisibilityChange)
    }, 2200)
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
    currentBitrateKbps: null,
    bandwidthEstimateKbps: null,
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

function updatePlaybackStatsFromHlsBandwidthEstimate(value: unknown) {
  const bandwidthEstimate = toFiniteNumber(value)
  if (!bandwidthEstimate || bandwidthEstimate <= 0) {
    return
  }

  playbackStats.value = {
    ...playbackStats.value,
    bandwidthEstimateKbps: Math.round(bandwidthEstimate / 1000),
  }
}

function updatePlaybackStatsFromHlsFragment(data?: unknown) {
  if (typeof data !== "object" || data === null) return

  const stats =
    "stats" in data && typeof (data as { stats?: unknown }).stats === "object"
      ? ((data as { stats?: Record<string, unknown> }).stats ?? null)
      : null
  const frag =
    "frag" in data && typeof (data as { frag?: unknown }).frag === "object"
      ? ((data as { frag?: Record<string, unknown> }).frag ?? null)
      : null

  const loadedBytes =
    toFiniteNumber(stats?.loaded) ??
    toFiniteNumber(stats?.total) ??
    toFiniteNumber(frag?.loaded)
  const durationSec =
    toFiniteNumber(frag?.duration) ??
    toFiniteNumber(frag?.maxStartPts) ??
    null
  const bandwidthEstimate =
    toFiniteNumber(stats?.bwEstimate) ??
    toFiniteNumber(stats?.bandwidthEstimate) ??
    null

  let currentBitrateKbps = playbackStats.value.currentBitrateKbps
  if (loadedBytes && loadedBytes > 0 && durationSec && durationSec > 0) {
    const measuredBitrateKbps = Math.round((loadedBytes * 8) / durationSec / 1000)
    if (Number.isFinite(measuredBitrateKbps) && measuredBitrateKbps > 0) {
      currentBitrateKbps =
        currentBitrateKbps && currentBitrateKbps > 0
          ? Math.round(currentBitrateKbps * 0.65 + measuredBitrateKbps * 0.35)
          : measuredBitrateKbps
    }
  }

  playbackStats.value = {
    ...playbackStats.value,
    currentBitrateKbps,
    bandwidthEstimateKbps:
      bandwidthEstimate && bandwidthEstimate > 0
        ? Math.round(bandwidthEstimate / 1000)
        : playbackStats.value.bandwidthEstimateKbps,
  }
}

function bindHlsStats(player: HlsInstance, events?: Record<string, string>) {
  detachHlsStatsListeners?.()
  detachHlsStatsListeners = null
  const on = player.on
  if (!on || !events) return

  const extractLevelIndex = (data?: unknown): number | null => {
    if (typeof data !== "object" || data === null) {
      return null
    }
    const explicitLevel =
      "level" in data ? toFiniteNumber((data as { level?: unknown }).level) : null
    if (explicitLevel !== null && explicitLevel >= 0) {
      return explicitLevel
    }
    if ("frag" in data && typeof (data as { frag?: unknown }).frag === "object") {
      const fragLevel = toFiniteNumber(
        ((data as { frag?: Record<string, unknown> }).frag ?? null)?.level,
      )
      if (fragLevel !== null && fragLevel >= 0) {
        return fragLevel
      }
    }
    return null
  }

  const updateCurrentLevelStats = (data?: unknown) => {
    const levels = player.levels
    if (!Array.isArray(levels) || levels.length === 0) return

    const levelCandidates = [
      extractLevelIndex(data),
      toFiniteNumber(player.currentLevel),
      toFiniteNumber(player.loadLevel),
      toFiniteNumber(player.nextLoadLevel),
      toFiniteNumber(player.nextLevel),
    ]
    const resolvedLevelIndex = levelCandidates.find(
      (levelIndex) => levelIndex !== null && levelIndex >= 0 && levelIndex < levels.length,
    )

    if (resolvedLevelIndex != null) {
      updatePlaybackStatsFromHlsLevel(levels[resolvedLevelIndex] ?? null)
    }
    updatePlaybackStatsFromHlsBandwidthEstimate(player.bandwidthEstimate)
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
    updateCurrentLevelStats(data)
  })
  register(events.FRAG_CHANGED, (_event, data) => {
    updateCurrentLevelStats(data)
    refreshPlaybackStatsFromVideo()
  })
  register(events.FRAG_LOADED, (_event, data) => {
    updatePlaybackStatsFromHlsFragment(data)
    updateCurrentLevelStats(data)
  })
  register(events.FRAG_BUFFERED, (_event, data) => {
    updatePlaybackStatsFromHlsFragment(data)
    updateCurrentLevelStats(data)
  })
  register(events.ERROR, (_event, data) => {
    const fatal =
      typeof data === "object" &&
      data !== null &&
      "fatal" in data &&
      (data as { fatal?: unknown }).fatal === true
    if (!fatal) return
    void fallbackHlsToDirect("fatal hls playback error")
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
  if (!kbps || kbps <= 0) return "N/A"
  if (kbps >= 1000) return `${(kbps / 1000).toFixed(2)} Mbps`
  return `${Math.round(kbps)} kbps`
}

function playbackTimelineOffsetSec(descriptor: PlaybackDescriptorDTO | null = playbackDescriptor.value): number {
  if (!descriptor || descriptor.mode !== "hls") {
    return 0
  }
  const offset = Number(descriptor.startPositionSec ?? 0)
  return Number.isFinite(offset) && offset > 0 ? offset : 0
}

function getAbsolutePlaybackTime(
  localTimeSec: number = Number(videoRef.value?.currentTime ?? 0),
  descriptor: PlaybackDescriptorDTO | null = playbackDescriptor.value,
): number {
  const local = Number.isFinite(localTimeSec) && localTimeSec > 0 ? localTimeSec : 0
  return local + playbackTimelineOffsetSec(descriptor)
}

function schedulePlaybackSessionCleanup(sessionId?: string) {
  const id = sessionId?.trim()
  if (!id) return
  playbackSessionCleanupId = id
}

function flushScheduledPlaybackSessionCleanup() {
  const id = playbackSessionCleanupId
  if (!id) return
  playbackSessionCleanupId = null
  void releasePlaybackSession(id)
}

function clampAbsolutePlaybackTarget(targetSec: number): number {
  const total = totalDurationSec.value
  const normalized = Number.isFinite(targetSec) ? targetSec : 0
  if (total <= 0) {
    return Math.max(0, normalized)
  }
  return Math.min(Math.max(0, normalized), Math.max(0, total - 0.25))
}

async function seekToAbsolutePlaybackTime(
  targetSec: number,
  options: {
    forceSessionSwap?: boolean
    resumeAfterSwap?: boolean
    previousDisplayedTimeSec?: number
  } = {},
) {
  const v = videoRef.value
  const descriptor = playbackDescriptor.value
  if (!v || !descriptor || !playbackSrc.value) return

  const clampedTarget = clampAbsolutePlaybackTarget(targetSec)
  isPlaybackWaiting.value = true
  if (descriptor.mode !== "hls") {
    v.currentTime = clampedTarget
    currentTime.value = clampedTarget
    return
  }

  const localTarget = clampedTarget - playbackTimelineOffsetSec(descriptor)
  const localDuration = Number.isFinite(v.duration) && v.duration > 0 ? v.duration : 0
  if (
    !options.forceSessionSwap &&
    localTarget >= 0 &&
    (localDuration <= 0 || localTarget <= localDuration + 0.25)
  ) {
    v.currentTime = localTarget
    currentTime.value = clampedTarget
    return
  }

  const movieId = props.movie.id.trim()
  if (!movieId) return
  const previousSessionId = descriptor.sessionId
  const shouldResumePlayback = options.resumeAfterSwap || (isPlaying.value && !videoRef.value?.paused)
  const seq = ++playbackLoadSeq
  isResolvingPlayback.value = true
  isSwitchingPlaybackSession.value = true
  playbackError.value = ""

  try {
    const nextDescriptor = await libraryService.createPlaybackSession(
      movieId,
      "hls",
      clampedTarget,
    )
    if (!nextDescriptor) return
    if (movieId !== props.movie.id.trim() || seq !== playbackLoadSeq) {
      if (nextDescriptor.sessionId) {
        await releasePlaybackSession(nextDescriptor.sessionId)
      }
      return
    }

    if (shouldResumePlayback) {
      resumePlaybackWhenReady = true
    }
    resumeAppliedForMovieId.value = null
    schedulePlaybackSessionCleanup(previousSessionId)
    playbackDescriptor.value = nextDescriptor
    playbackSrc.value = nextDescriptor.url?.trim() || null
    void prewarmHlsDescriptor(nextDescriptor)
    currentTime.value = clampedTarget
    progressSliderValue.value = [clampedTarget]
  } catch (err) {
    if (seq === playbackLoadSeq) {
      optimisticSeekTargetSec.value = null
      isPlaybackWaiting.value = false
      if (options.previousDisplayedTimeSec != null && Number.isFinite(options.previousDisplayedTimeSec)) {
        currentTime.value = clampAbsolutePlaybackTarget(options.previousDisplayedTimeSec)
        progressSliderValue.value = [currentTime.value]
      }
      pushAppToast(formatClientError(err, t("player.errGeneric")), {
        variant: "destructive",
      })
    }
  } finally {
    if (seq === playbackLoadSeq) {
      isResolvingPlayback.value = false
      isSwitchingPlaybackSession.value = false
    }
  }
}

function formatResolutionLabel(width: number | null, height: number | null): string {
  if (!width || !height) return "N/A"
  return `${width} × ${height}`
}

function formatFpsLabel(fps: number | null): string {
  if (!fps || fps <= 0) return "N/A"
  return `${fps.toFixed(fps >= 100 ? 0 : 2)} fps`
}

function formatTimecodeLabel(seconds: number | null | undefined): string {
  if (seconds == null || !Number.isFinite(seconds) || seconds < 0) return "N/A"
  return formatClock(seconds)
}

function formatPercentLabel(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return "N/A"
  return `${Math.max(0, Math.min(100, value)).toFixed(1)}%`
}

function formatCountLabel(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return "N/A"
  return String(Math.max(0, Math.trunc(value)))
}

function formatReasonLabel(reason: string | null | undefined): string {
  const text = reason?.trim() || ""
  if (!text) return "N/A"
  return text
}

function formatSessionKindLabel(sessionKind: string | null | undefined): string {
  switch ((sessionKind ?? "").trim().toLowerCase()) {
    case "direct-file":
      return "Direct File"
    case "remux-hls":
      return "Remux HLS"
    case "transcode-hls":
      return "Transcode HLS"
    default:
      return "N/A"
  }
}

function formatSourceFormatLabel(descriptor: PlaybackDescriptorDTO | null | undefined): string {
  const container = descriptor?.sourceContainer?.trim()
  const videoCodec = descriptor?.sourceVideoCodec?.trim()
  const audioCodec = descriptor?.sourceAudioCodec?.trim()
  const parts = [container, videoCodec, audioCodec].filter((value) => Boolean(value))
  if (parts.length === 0) return "N/A"
  return parts.join(" / ")
}

function formatTranscodeProfileLabel(profile: string | null | undefined): string {
  switch ((profile ?? "").trim().toLowerCase()) {
    case "remux_copy":
      return "FFmpeg Stream Copy"
    case "h264_amf":
      return "AMD AMF"
    case "h264_qsv":
      return "Intel QSV"
    case "h264_nvenc":
      return "NVIDIA NVENC"
    case "h264_videotoolbox":
      return "VideoToolbox"
    case "libx264":
      return "libx264"
    default:
      return "N/A"
  }
}

function isPlaybackStatUnavailable(value: string): boolean {
  return value === "N/A"
}

const playbackStateLabel = computed(() => {
  if (playbackError.value) return "Error"
  if (!playbackSrc.value) return "Idle"
  if (totalDurationSec.value > 0 && currentTime.value >= Math.max(0, totalDurationSec.value - 0.15)) return "Ended"
  return isPlaying.value ? "Playing" : "Paused"
})

const pipStatusLabel = computed(() => {
  if (!pipSupported.value) return "Unsupported"
  return isPipActive.value ? "On" : "Off"
})

const volumeStatusLabel = computed(() => {
  if (playbackMuted.value) return "Muted"
  return `${Math.round(volumePercent.value)}%`
})

const currentPlaybackBitrateLabel = computed(() =>
  formatBitrateLabel(
    playbackDescriptor.value?.mode === "hls"
      ? playbackStats.value.currentBitrateKbps ??
          playbackStats.value.bandwidthEstimateKbps ??
          playbackStats.value.videoBitrateKbps
      : playbackStats.value.videoBitrateKbps,
  ),
)

const bandwidthEstimateLabel = computed(() =>
  formatBitrateLabel(playbackStats.value.bandwidthEstimateKbps),
)

const currentTimeRangeLabel = computed(() => {
  return `${formatTimecodeLabel(displayedCurrentTimeSec.value)} / ${formatTimecodeLabel(totalDurationSec.value)}`
})

const resumePositionLabel = computed(() =>
  formatTimecodeLabel(playbackDescriptor.value?.resumePositionSec),
)

const playbackStatsRows = computed(() => {
  const descriptor = playbackDescriptor.value
  const rows = [
    {
      key: "state",
      label: "State",
      value: playbackStateLabel.value,
    },
    {
      key: "mode",
      label: "Mode",
      value: detailedStatsModeLabel.value,
    },
    {
      key: "session-kind",
      label: "Session",
      value: formatSessionKindLabel(descriptor?.sessionKind),
    },
    {
      key: "file",
      label: "File",
      value: fileBasename.value || "N/A",
    },
    {
      key: "mime",
      label: "MIME Type",
      value: descriptor?.mimeType?.trim() || "N/A",
    },
    {
      key: "transcode-profile",
      label: "Transcoder",
      value: formatTranscodeProfileLabel(descriptor?.transcodeProfile),
    },
    {
      key: "reason-code",
      label: "Reason Code",
      value: formatReasonLabel(descriptor?.reasonCode),
    },
    {
      key: "reason",
      label: "Reason",
      value: formatReasonLabel(descriptor?.reasonMessage ?? descriptor?.reason),
    },
    {
      key: "source-format",
      label: "Source Format",
      value: formatSourceFormatLabel(descriptor),
    },
    {
      key: "current",
      label: "Current / Total",
      value: currentTimeRangeLabel.value,
    },
    {
      key: "progress",
      label: "Progress",
      value: formatPercentLabel(progressPercent.value),
    },
    {
      key: "resume",
      label: "Resume At",
      value: resumePositionLabel.value,
    },
    {
      key: "volume",
      label: "Volume",
      value: volumeStatusLabel.value,
    },
    {
      key: "pip",
      label: "Picture-in-Picture",
      value: pipStatusLabel.value,
    },
    {
      key: "direct-play",
      label: "Direct Play",
      value: descriptor?.canDirectPlay ? "Yes" : "No",
    },
    {
      key: "audio-tracks",
      label: "Audio Tracks",
      value: formatCountLabel(descriptor?.audioTracks?.length),
    },
    {
      key: "subtitle-tracks",
      label: "Subtitle Tracks",
      value: formatCountLabel(descriptor?.subtitleTracks?.length),
    },
    {
      key: "audio",
      label: "Audio Bitrate",
      value: formatBitrateLabel(playbackStats.value.audioBitrateKbps),
    },
    {
      key: "video-current",
      label: descriptor?.mode === "hls" ? "Current Bitrate" : "Video Bitrate",
      value: currentPlaybackBitrateLabel.value,
    },
    {
      key: "resolution",
      label: "Resolution",
      value: formatResolutionLabel(playbackStats.value.width, playbackStats.value.height),
    },
    {
      key: "fps",
      label: "Frame Rate",
      value: formatFpsLabel(playbackStats.value.fps),
    },
  ]

  if (descriptor?.mode === "hls") {
    rows.splice(
      rows.length - 2,
      0,
      {
        key: "video-variant",
        label: "Variant Target",
        value: formatBitrateLabel(playbackStats.value.videoBitrateKbps),
      },
      {
        key: "bandwidth",
        label: "Bandwidth Estimate",
        value: bandwidthEstimateLabel.value,
      },
    )
  }

  return rows
})

const playbackStatsColumns = computed(() => {
  const rows = playbackStatsRows.value
  const midpoint = Math.ceil(rows.length / 2)
  return [rows.slice(0, midpoint), rows.slice(midpoint)]
})

const detailedStatsModeLabel = computed(() => {
  const descriptor = playbackDescriptor.value
  const mode = descriptor?.mode
  if (mode === "hls") {
    if (descriptor?.sessionKind === "remux-hls") return "HLS (Remux)"
    if (descriptor?.sessionKind === "transcode-hls") return "HLS (Transcode)"
    return "HLS"
  }
  if (mode === "native") return "Native"
  if (mode === "direct") return "Direct"
  return "N/A"
})

const videoPreloadMode = computed(() =>
  playbackDescriptor.value?.mode === "hls" || playbackDescriptor.value?.mode === "direct"
    ? "auto"
    : "metadata",
)
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
        class="player-stats-panel absolute left-2 top-16 z-[18] w-[min(44rem,calc(100%-1rem))] overflow-hidden rounded-[2px] border border-white/10 bg-[#1a1a1a]/96 text-white shadow-[0_18px_36px_rgba(0,0,0,0.42)] sm:left-4 sm:top-20"
        @click.stop
      >
        <div class="px-4 py-3 sm:px-5 sm:py-3.5">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <p class="text-[11px] font-semibold leading-none tracking-[0.08em] text-white/92 sm:text-[12px]">
                Video Stats
              </p>
              <p class="mt-2 truncate font-mono text-[11px] tracking-[0.05em] text-white/78 sm:text-[12px]">
                {{ movie.code }} / {{ detailedStatsModeLabel }} / {{ fileBasename || movie.location }}
              </p>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="stats-close-button h-6 w-auto shrink-0 rounded-none px-1.5 text-[12px] font-semibold leading-none text-white/68 hover:bg-transparent hover:text-white"
              aria-label="关闭详细统计信息"
              @click="closeDetailedStats"
            >
              [X]
            </Button>
          </div>
        </div>
        <div class="grid gap-x-5 px-3 pb-3 md:grid-cols-2 sm:px-4 sm:pb-4">
          <dl
            v-for="(column, columnIndex) in playbackStatsColumns"
            :key="columnIndex"
            class="min-w-0"
          >
            <div
              v-for="row in column"
              :key="row.key"
              class="grid grid-cols-[8.5rem_minmax(0,1fr)] items-baseline gap-x-2 py-[1px] sm:grid-cols-[9.5rem_minmax(0,1fr)]"
            >
              <dt class="truncate text-right text-[10px] font-semibold leading-5 text-white/70 sm:text-[11px]">
                {{ row.label }}
              </dt>
              <dd
                :class="
                  cn(
                    'truncate pl-2 text-left font-mono text-[10px] font-semibold leading-5 tracking-[0.01em] tabular-nums sm:text-[11px]',
                    isPlaybackStatUnavailable(row.value) ? 'text-white/45' : 'text-white',
                  )
                "
              >
                {{ row.value }}
              </dd>
            </div>
          </dl>
        </div>
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
          :preload="videoPreloadMode"
          @click.stop="onVideoSurfaceClick"
          @timeupdate="onTimeUpdate"
          @progress="onVideoProgress"
          @loadedmetadata="onLoadedMetadata"
          @loadeddata="onVideoLoadedData"
          @canplay="onCanPlayForAutoplay"
          @play="onPlay"
          @pause="onPause"
          @waiting="onVideoWaiting"
          @seeking="onVideoSeeking"
          @seeked="onVideoSeeked"
          @ended="onVideoEnded"
          @error="onVideoError"
          @enterpictureinpicture="onVideoEnterPictureInPicture"
          @leavepictureinpicture="onVideoLeavePictureInPicture"
        />

        <div
          v-else
          class="pointer-events-none flex max-w-lg flex-col items-center gap-3 px-4 text-center text-white/80"
        >
          <p class="text-lg font-semibold text-white">
            {{ isResolvingPlayback ? t("common.loading") : t("player.noOnlineSrc") }}
          </p>
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

        <Transition
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="opacity-0 scale-95"
          enter-to-class="opacity-100 scale-100"
          leave-active-class="transition duration-200 ease-in"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-95"
        >
          <div
            v-if="showCenteredBusyOverlay"
            class="pointer-events-none absolute inset-0 z-20 flex items-center justify-center px-6"
          >
            <div class="flex min-w-[12rem] max-w-sm flex-col items-center gap-3 rounded-[1.5rem] border border-white/12 bg-black/48 px-6 py-5 text-center text-white shadow-[0_18px_50px_rgba(0,0,0,0.38)] backdrop-blur-md">
              <Loader2 class="size-8 animate-spin text-white/85" aria-hidden="true" />
              <span class="text-sm font-medium tracking-[0.01em] text-white/88">{{ playbackBusyLabel }}</span>
              <span
                v-if="isPrewarmingHls && playbackDescriptor?.mode === 'hls' && !isPlaying"
                class="text-xs font-medium tabular-nums text-white/55"
              >
                {{ prewarmProgressPercent }}%
              </span>
            </div>
          </div>
        </Transition>
      </div>

      <div
        class="absolute inset-x-0 bottom-0 z-10 bg-gradient-to-t from-black/90 via-black/65 to-transparent p-4 sm:p-5"
        :class="[CHROME_LAYER_TRANSITION, chromeLayerVisibleClass]"
      >
        <div class="flex w-full flex-col gap-4">
          <div class="flex items-center justify-between gap-3 text-sm text-white/70">
            <span>{{ formatClock(displayedCurrentTimeSec) }}</span>
            <span>{{ totalDurationSec > 0 ? formatClock(totalDurationSec) : "\u2014" }}</span>
          </div>

          <div class="relative">
            <Slider
              :model-value="progressSliderValue"
              :max="Math.max(totalDurationSec, 0.25)"
              :step="0.1"
              :disabled="!playbackSrc || totalDurationSec <= 0"
              :aria-label="t('player.progressAria')"
              class="relative z-10 w-full"
              @update:model-value="onProgressSliderInput"
              @value-commit="onProgressSliderCommit"
            />

            <div
              v-if="bufferedTrackPercent > playedTrackPercent"
              class="pointer-events-none absolute inset-x-0 top-1/2 z-[11] h-1.5 -translate-y-1/2 overflow-hidden rounded-full"
              aria-hidden="true"
            >
              <div
                class="absolute inset-y-0 rounded-full bg-white/26"
                :style="bufferedAheadStyle"
              />
            </div>
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
                {{ nativePlayerLabel }}
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
.player-stats-panel::before {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.03);
}

.stats-close-button {
  box-shadow: none;
}

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
