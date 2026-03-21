<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue"
import {
  Maximize2,
  Pause,
  Play,
  SkipBack,
  SkipForward,
  Volume2,
} from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{
  movie: Movie
}>()

const libraryService = useLibraryService()

const videoRef = ref<HTMLVideoElement | null>(null)
const surfaceRef = ref<HTMLElement | null>(null)

const playbackSrc = ref<string | null>(null)
const playbackError = ref("")
const isPlaying = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const volume = ref([100])

function syncSrc() {
  playbackError.value = ""
  playbackSrc.value = libraryService.getMoviePlaybackUrl(props.movie.id)
}

watch(
  () => props.movie.id,
  async () => {
    syncSrc()
    await nextTick()
    videoRef.value?.load()
  },
  { immediate: true },
)

onMounted(() => {
  syncSrc()
  window.addEventListener("keydown", onPlaybackKeydown)
})

onUnmounted(() => {
  window.removeEventListener("keydown", onPlaybackKeydown)
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

function onTimeUpdate() {
  const v = videoRef.value
  if (!v) return
  currentTime.value = v.currentTime
}

function onLoadedMetadata() {
  const v = videoRef.value
  if (!v) return
  duration.value = Number.isFinite(v.duration) ? v.duration : 0
  v.volume = (volume.value[0] ?? 100) / 100
}

function onPlay() {
  isPlaying.value = true
}

function onPause() {
  isPlaying.value = false
}

function onVideoError() {
  const v = videoRef.value
  const code = v?.error?.code
  if (code === MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED) {
    playbackError.value =
      "当前浏览器无法解码该视频格式，可尝试转为 MP4（H.264 + AAC）或使用后续桌面版播放器。"
  } else if (code != null) {
    playbackError.value = "无法播放该视频，请检查文件是否存在或已被移动。"
  } else {
    playbackError.value = "播放出错，请稍后重试。"
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
    playbackError.value = "无法开始播放（浏览器可能阻止自动播放，请再试一次）。"
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

function onVolumeSlider(vols: number[]) {
  volume.value = vols
  const v = videoRef.value
  if (v) {
    v.volume = (vols[0] ?? 100) / 100
    if (v.volume > 0) v.muted = false
  }
}

function toggleMute() {
  const v = videoRef.value
  if (!v) return
  v.muted = !v.muted
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
    default:
      break
  }
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
  if (playbackSrc.value) return ""
  return import.meta.env.VITE_USE_WEB_API === "true"
    ? "无法解析播放地址，请确认已登录同一后端且该片在库中。"
    : "本地演示模式无法播放主视频，请在 .env 中启用 VITE_USE_WEB_API 并连接后端。"
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-col p-1 sm:p-2">
    <div
      ref="surfaceRef"
      class="relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.75rem] border border-border/50 bg-gradient-to-br from-black via-zinc-950 to-card"
    >
      <div class="absolute inset-x-0 top-0 z-10 flex items-start justify-between gap-3 bg-gradient-to-b from-black/85 via-black/40 to-transparent p-4 sm:p-5">
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
        class="relative flex min-h-0 flex-1 cursor-pointer items-center justify-center p-4 sm:p-6 lg:p-8"
        @click="onVideoSurfaceClick"
      >
        <video
          v-if="playbackSrc"
          ref="videoRef"
          class="h-full max-h-full w-full max-w-full object-contain"
          playsinline
          preload="metadata"
          :src="playbackSrc"
          @click.stop="onVideoSurfaceClick"
          @timeupdate="onTimeUpdate"
          @loadedmetadata="onLoadedMetadata"
          @play="onPlay"
          @pause="onPause"
          @error="onVideoError"
        />

        <div
          v-else
          class="pointer-events-none flex max-w-lg flex-col items-center gap-3 px-4 text-center text-white/80"
        >
          <p class="text-lg font-semibold text-white">暂无在线片源</p>
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

      <div class="absolute inset-x-0 bottom-0 z-10 bg-gradient-to-t from-black/90 via-black/65 to-transparent p-4 sm:p-5">
        <div class="flex w-full flex-col gap-4">
          <div class="flex items-center justify-between gap-3 text-sm text-white/70">
            <span>{{ formatClock(currentTime) }}</span>
            <span>{{ duration > 0 ? formatClock(duration) : "—" }}</span>
          </div>

          <div
            class="relative h-2.5 w-full cursor-pointer rounded-full bg-white/10"
            role="button"
            tabindex="0"
            aria-label="播放进度"
            @click="onProgressBarClick"
          >
            <div
              class="bg-primary pointer-events-none absolute inset-y-0 left-0 rounded-full transition-[width] duration-150"
              :style="{ width: `${progressPercent}%` }"
            />
          </div>

          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex items-center gap-2">
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

            <div class="flex flex-wrap items-center gap-3">
              <div class="flex min-w-[14rem] items-center gap-3 rounded-full bg-white/8 px-4 py-2 text-white/80 backdrop-blur">
                <Volume2 />
                <Slider
                  :model-value="volume"
                  :max="100"
                  :step="1"
                  class="flex-1"
                  :disabled="!playbackSrc"
                  @update:model-value="onVolumeSlider"
                />
                <span class="w-10 text-right text-sm">{{ volume[0] }}%</span>
              </div>

              <Button
                variant="secondary"
                class="rounded-2xl bg-white/10 text-white hover:bg-white/20"
                :disabled="!playbackSrc"
                @click="toggleFullscreen"
              >
                <Maximize2 data-icon="inline-start" />
                全屏
              </Button>
            </div>
          </div>

          <p
            v-if="playbackSrc"
            class="text-center text-[10px] leading-relaxed text-white/40 sm:text-xs"
          >
            单击画面 播放/暂停 · 快捷键：空格 / K · ← / J 后退 10 秒 · → / L 前进 10 秒 · F 全屏 · Esc 退出全屏 · M 静音 · ↑ / ↓ 音量
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
