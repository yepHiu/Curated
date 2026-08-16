<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import {
  clusterFrameMarkers,
  type FrameMarkerCluster,
  type FrameMarkerInput,
} from "@/lib/player-frame-markers"
import { formatPlaybackClock } from "@/lib/player-playback-timeline"

/**
 * 进度条萃取帧标记层：叠在播放进度 Slider 之上。
 * 层本身不拦截事件（pointer-events-none），仅刻度元素可点；
 * 自测容器宽度做密集合并，点击刻度跳转（合并簇跳簇内最早帧）。
 */
const props = defineProps<{
  markers: readonly FrameMarkerInput[]
  durationSec: number
}>()

const emit = defineEmits<{ seek: [sec: number] }>()

const { t } = useI18n()

const rootRef = ref<HTMLElement | null>(null)
const trackWidthPx = ref(0)
const hoverCluster = ref<FrameMarkerCluster | null>(null)

let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (typeof ResizeObserver === "undefined" || !rootRef.value) return
  resizeObserver = new ResizeObserver((entries) => {
    trackWidthPx.value = entries[0]?.contentRect.width ?? 0
  })
  resizeObserver.observe(rootRef.value)
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
})

const clusters = computed(() =>
  clusterFrameMarkers(props.markers, props.durationSec, trackWidthPx.value),
)

function markerKey(cluster: FrameMarkerCluster): string {
  return cluster.items.map((item) => item.id).join("|")
}

function markerAriaLabel(cluster: FrameMarkerCluster): string {
  const time = formatPlaybackClock(cluster.seekToSec)
  return cluster.items.length > 1
    ? t("player.frameMarkerClusterAria", { count: cluster.items.length, time })
    : t("player.frameMarkerAria", { time })
}

function clusterTip(cluster: FrameMarkerCluster): string {
  return t("player.frameMarkerClusterTip", {
    count: cluster.items.length,
    time: formatPlaybackClock(cluster.seekToSec),
  })
}

function onSeek(cluster: FrameMarkerCluster) {
  hoverCluster.value = null
  emit("seek", cluster.seekToSec)
}
</script>

<template>
  <div
    ref="rootRef"
    data-slot="player-progress-frame-markers"
    class="pointer-events-none absolute inset-x-0 top-1/2 z-[12] h-0"
  >
    <button
      v-for="cluster in clusters"
      :key="markerKey(cluster)"
      type="button"
      :data-frame-marker="cluster.items.length"
      :data-frame-marker-sec="cluster.seekToSec"
      class="group pointer-events-auto absolute top-1/2 flex h-11 w-3 -translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center border-0 bg-transparent p-0 focus-visible:outline-none"
      :style="{ left: `${cluster.ratio * 100}%` }"
      :aria-label="markerAriaLabel(cluster)"
      @click="onSeek(cluster)"
      @mouseenter="hoverCluster = cluster"
      @mouseleave="hoverCluster = null"
      @focus="hoverCluster = cluster"
      @blur="hoverCluster = null"
    >
      <span
        class="block rounded-full bg-primary shadow-[0_0_0_1px_rgba(0,0,0,0.55)] group-hover:bg-primary/85"
        :class="cluster.items.length > 1 ? 'h-3 w-[5px]' : 'h-[11px] w-[3px]'"
      />
      <span
        v-if="cluster.items.length > 1"
        class="absolute bottom-[9px] left-1/2 -translate-x-1/2 whitespace-nowrap rounded-full bg-primary px-[5px] py-[3px] text-[10px] font-bold leading-none text-primary-foreground"
      >
        ×{{ cluster.items.length }}
      </span>
    </button>

    <div
      v-if="hoverCluster"
      class="pointer-events-none absolute bottom-[30px] z-20 -translate-x-1/2 rounded-lg border border-white/15 bg-neutral-950/95 px-2 py-1 text-xs whitespace-nowrap text-white tabular-nums shadow-lg"
      :style="{ left: `${hoverCluster.ratio * 100}%` }"
      role="status"
    >
      {{ hoverCluster.items.length > 1 ? clusterTip(hoverCluster) : formatPlaybackClock(hoverCluster.seekToSec) }}
    </div>
  </div>
</template>
