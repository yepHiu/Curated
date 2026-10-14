<script setup lang="ts">
import { onScopeDispose, ref } from "vue"
import { FastForward, Pause, Play, Rewind, Volume2, VolumeX } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { createSliderPointerDrag, type SliderPointerScale } from "@/lib/slider-pointer-drag"

const props = defineProps<{
  disabled?: boolean
  playing: boolean
  volumeValues: number[]
  volumePercent: number
  muted: boolean
  /** 原生透明窗口里由页面跟踪音量条移动。组件库只在捕获目标上继续拖，捕获对不上时按下只会跳一次。 */
  followVolumePointer?: boolean
  labels: { play: string; pause: string; seekBack: string; seekForward: string; volume: string; mute: string; unmute: string }
}>()
const emit = defineEmits<{
  toggle: []
  seekBack: []
  seekForward: []
  mute: []
  volume: [values: number[] | undefined]
}>()
const volumeGroup = ref<HTMLElement>()
const volumeDrag = createSliderPointerDrag()

/** 找到这次按下命中的音量滑块；静音按钮和百分比文字不接管。 */
function volumeSliderFromEvent(event: PointerEvent) {
  if (event.button !== 0) return
  const target = event.target
  if (!(target instanceof Element)) return
  const slider = target.closest<HTMLElement>("[data-slot=slider]")
  if (!slider || !volumeGroup.value?.contains(slider) || slider.hasAttribute("data-disabled")) return
  return slider
}

/** 记录按下时的轨道、拇指宽度和抓取偏移。按在拇指上保留接触点，按在轨道上对齐拇指中心。 */
function readVolumeDragStart(slider: HTMLElement, event: PointerEvent) {
  const rect = slider.getBoundingClientRect()
  const thumb = slider.querySelector<HTMLElement>("[data-slot=slider-thumb]")
  const thumbRect = thumb?.getBoundingClientRect()
  const thumbWidth = thumbRect?.width ?? 0
  const onThumb = thumb != null && thumb.contains(event.target as Node)
  const grabOffset = onThumb && thumbRect ? event.clientX - thumbRect.left : thumbWidth / 2
  const scale: SliderPointerScale = { left: rect.left, width: rect.width, thumbWidth, min: 0, max: 100, step: 1 }
  return { grabOffset, scale, thumb }
}

/** 拖动中重新读取轨道左右边界。 */
function readVolumeTrack() {
  const slider = volumeGroup.value?.querySelector<HTMLElement>("[data-slot=slider]")
  if (!slider) return
  const rect = slider.getBoundingClientRect()
  return { left: rect.left, width: rect.width }
}

/** 按住期间监听整窗移动和松开，指针离开滑块后仍能继续改音量。 */
function listenVolumePointer() {
  window.addEventListener("pointermove", moveVolumePointer)
  window.addEventListener("pointerup", endVolumePointer)
  window.addEventListener("pointercancel", endVolumePointer)
  window.addEventListener("blur", blurVolumePointer)
}

/** 拖动结束或组件卸载时卸下窗口监听。 */
function silenceVolumePointer() {
  window.removeEventListener("pointermove", moveVolumePointer)
  window.removeEventListener("pointerup", endVolumePointer)
  window.removeEventListener("pointercancel", endVolumePointer)
  window.removeEventListener("blur", blurVolumePointer)
}

/** 按下音量条后改由页面写入数值，并拦住组件库那次只会跳点的按下处理。 */
function beginVolumePointerDrag(event: PointerEvent) {
  if (!props.followVolumePointer || volumeDrag.active()) return
  const slider = volumeSliderFromEvent(event)
  if (!slider) return
  event.preventDefault()
  event.stopPropagation()
  const start = readVolumeDragStart(slider, event)
  const value = volumeDrag.begin(event.pointerId, event.clientX, start.grabOffset, start.scale)
  emit("volume", [value])
  listenVolumePointer()
  start.thumb?.focus({ preventScroll: true })
  try {
    slider.setPointerCapture(event.pointerId)
  } catch {
    // 透明窗口拒绝捕获时，窗口级 pointermove 仍会更新音量。
  }
}

/** 按住时按当前轨道位置发出音量。 */
function moveVolumePointer(event: PointerEvent) {
  const track = readVolumeTrack()
  if (!track) {
    endVolumePointer(event)
    return
  }
  const value = volumeDrag.update(event.pointerId, event.clientX, track.left, track.width)
  if (value === undefined) return
  emit("volume", [value])
}

/** 松开或取消时提交最终位置并停止跟随。 */
function endVolumePointer(event: PointerEvent) {
  const id = volumeDrag.currentPointerId()
  if (id === undefined || event.pointerId !== id) return
  const track = readVolumeTrack()
  const value = volumeDrag.finish(id, track ? event.clientX : undefined, track?.left, track?.width)
  silenceVolumePointer()
  if (value !== undefined) emit("volume", [value])
}

/** 窗口失焦时取消跟踪，避免指针在窗外松开后音量条一直处于拖动。 */
function blurVolumePointer() {
  const id = volumeDrag.currentPointerId()
  if (id === undefined) return
  volumeDrag.finish(id)
  silenceVolumePointer()
}

/** 组件卸载时结束尚未松开的音量拖动。 */
function releaseVolumePointerDrag() {
  const id = volumeDrag.currentPointerId()
  if (id !== undefined) volumeDrag.finish(id)
  silenceVolumePointer()
}

onScopeDispose(releaseVolumePointerDrag)
</script>

<template>
  <!-- 保留既有播放页的形状和密度；不同引擎只绑定状态与业务动作。 -->
  <div class="grid w-full items-center gap-x-3 gap-y-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] sm:gap-x-4">
    <div class="flex items-center justify-center gap-2 sm:justify-start">
      <slot name="previous" />
      <Button variant="secondary" size="icon" class="rounded-full bg-white/10 text-white hover:bg-white/20" :disabled="disabled" :aria-label="labels.seekBack" @click="emit('seekBack')">
        <Rewind />
      </Button>
      <Button size="icon-lg" class="rounded-full" :disabled="disabled" :aria-label="playing ? labels.pause : labels.play" @click="emit('toggle')">
        <Pause v-if="playing" />
        <Play v-else />
      </Button>
      <Button variant="secondary" size="icon" class="rounded-full bg-white/10 text-white hover:bg-white/20" :disabled="disabled" :aria-label="labels.seekForward" @click="emit('seekForward')">
        <FastForward />
      </Button>
      <slot name="next" />
    </div>
    <div class="flex flex-wrap items-center justify-center gap-3 sm:col-start-2 sm:justify-end">
      <div ref="volumeGroup" class="flex h-9 min-w-[min(100%,14rem)] max-w-full flex-1 items-center gap-2 rounded-full bg-white/8 px-3 text-white/80 backdrop-blur sm:min-w-[14rem] sm:flex-initial sm:gap-3 sm:px-4" role="group" :aria-label="labels.volume" @pointerdown.capture="beginVolumePointerDrag">
        <Button type="button" variant="ghost" size="icon" class="size-9 shrink-0 rounded-full text-white hover:bg-white/15" :disabled="disabled" :aria-pressed="muted" :aria-label="muted ? labels.unmute : labels.mute" @click="emit('mute')">
          <VolumeX v-if="muted" aria-hidden="true" />
          <Volume2 v-else aria-hidden="true" />
        </Button>
        <Slider :model-value="volumeValues" :max="100" :step="1" class="flex-1" :disabled="disabled" :aria-label="labels.volume" @update:model-value="emit('volume', $event)" />
        <span class="w-10 shrink-0 text-right text-sm leading-none tabular-nums">{{ volumePercent }}%</span>
      </div>
      <slot />
    </div>
  </div>
</template>
