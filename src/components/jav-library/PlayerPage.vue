<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import {
  Maximize2,
  Pause,
  Play,
  SkipBack,
  SkipForward,
  Volume2,
  VolumeX,
} from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { recordMoviePlayed } from "@/lib/played-movies-storage"
import { saveCuratedCaptureFromVideo } from "@/lib/curated-frames/save-capture"
import {
  getProgress,
  parseResumeSecondsFromQuery,
  saveProgress,
} from "@/lib/playback-progress-storage"
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

const videoRef = ref<HTMLVideoElement | null>(null)
const surfaceRef = ref<HTMLElement | null>(null)

/** 每条片源只尝试一次入口自动播放，避免 canplay 重复触发 */
const autoplayConsumedForMovieId = ref<string | null>(null)
/** 每条片源只应用一次 URL / 本地续播 seek，避免重复跳转 */
const resumeAppliedForMovieId = ref<string | null>(null)

const PROGRESS_SAVE_INTERVAL_MS = 4000
let lastProgressSaveAt = 0

const playbackSrc = ref<string | null>(null)
const playbackError = ref("")
const isPlaying = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const initialAudio = getPlayerAudioPrefs()
const volume = ref([initialAudio.volumePercent])
/** 与 video.muted 同步，用于 UI 与持久化（音量滑块为 0 时不使用静音标志） */
const playbackMuted = ref(initialAudio.muted)

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

watch(playbackSrc, (src) => {
  if (!src) {
    clearIdleHideTimer()
    chromeVisible.value = true
  }
})

function syncSrc() {
  playbackError.value = ""
  playbackSrc.value = libraryService.getMoviePlaybackUrl(props.movie.id)
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
  window.addEventListener("keydown", onPlaybackKeydown)
  document.addEventListener("visibilitychange", onVisibilityChange)
  window.addEventListener("beforeunload", onWindowBeforeUnload)
})

onUnmounted(() => {
  flushPlaybackProgress()
  window.removeEventListener("keydown", onPlaybackKeydown)
  document.removeEventListener("visibilitychange", onVisibilityChange)
  window.removeEventListener("beforeunload", onWindowBeforeUnload)
  clearIdleHideTimer()
  if (curatedPlusOneTimer) clearTimeout(curatedPlusOneTimer)
  if (curatedShutterTimer) clearTimeout(curatedShutterTimer)
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
  const pct = volume.value[0] ?? 100
  v.volume = pct / 100
  v.muted = pct > 0 && playbackMuted.value

  const dur = duration.value
  if (resumeAppliedForMovieId.value === props.movie.id) return
  if (dur <= 0) return

  const fromQuery = parseResumeSecondsFromQuery(route.query.t)
  const stored = getProgress(props.movie.id)?.positionSec
  const targetSec = fromQuery !== undefined ? fromQuery : stored
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
  if (!playbackSrc.value) return
  void togglePlayPause()
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
      seekDelta(-10)
      break
    case "ArrowRight":
    case "KeyL":
      e.preventDefault()
      seekDelta(10)
      break
    case "KeyF":
      e.preventDefault()
      void toggleFullscreen()
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
</script>

<template>
  <div class="flex h-full min-h-0 flex-col p-1 sm:p-2">
    <div
      ref="surfaceRef"
      class="relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.75rem] border border-border/50 bg-gradient-to-br from-black via-zinc-950 to-card"
      :class="surfaceCursorClass"
      @mousedown="onChromePointerActivity"
      @mousemove="onChromePointerActivity"
      @mouseenter="onChromePointerActivity"
      @mouseleave="onChromePointerLeave"
    >
      <div
        class="absolute inset-x-0 top-0 z-10 flex items-start justify-between gap-3 bg-gradient-to-b from-black/85 via-black/40 to-transparent p-4 sm:p-5"
        :class="[CHROME_LAYER_TRANSITION, chromeLayerVisibleClass]"
      >
        <div class="flex min-w-0 flex-col items-start gap-2 text-left">
          <Badge variant="secondary" class="rounded-full border border-border/60 bg-background/30">
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
          :src="playbackSrc"
          @click.stop="onVideoSurfaceClick"
          @timeupdate="onTimeUpdate"
          @loadedmetadata="onLoadedMetadata"
          @canplay="onCanPlayForAutoplay"
          @play="onPlay"
          @pause="onPause"
          @ended="onVideoEnded"
          @error="onVideoError"
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
                @click="seekDelta(-10)"
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
                @click="seekDelta(10)"
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
            {{ t("player.hintBar") }}
          </p>
        </div>
      </div>
    </div>
  </div>
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
