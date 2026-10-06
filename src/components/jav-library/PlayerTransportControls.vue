<script setup lang="ts">
import { FastForward, Pause, Play, Rewind, Volume2, VolumeX } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"

defineProps<{
  disabled?: boolean
  playing: boolean
  volumeValues: number[]
  volumePercent: number
  muted: boolean
  labels: { play: string; pause: string; seekBack: string; seekForward: string; volume: string; mute: string; unmute: string }
}>()
const emit = defineEmits<{
  toggle: []
  seekBack: []
  seekForward: []
  mute: []
  volume: [values: number[] | undefined]
}>()
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
      <div class="flex h-9 min-w-[min(100%,14rem)] max-w-full flex-1 items-center gap-2 rounded-full bg-white/8 px-3 text-white/80 backdrop-blur sm:min-w-[14rem] sm:flex-initial sm:gap-3 sm:px-4" role="group" :aria-label="labels.volume">
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
