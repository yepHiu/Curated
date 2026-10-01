<script setup lang="ts">
import { onBeforeUnmount, provide, ref, watch } from "vue"
import { routeLocationKey, useRoute } from "vue-router"
import PlayerPage from "@/components/jav-library/PlayerPage.vue"
import { parseResumeSecondsFromQuery } from "@/lib/playback-progress-storage"
import type { PlaybackHost } from "@/composables/use-playback-host"

const props = defineProps<{ host: PlaybackHost }>()
const currentRoute = useRoute()
const player = ref<InstanceType<typeof PlayerPage> | null>(null)
const target = props.host.target
const visible = props.host.visible
const unregisterControls = props.host.registerMediaControls({
  // 所有停止路径（包括认证失效）同步保存、暂停，再移除组件。
  stop: () => { player.value?.stopHostedPlayback() },
  toggle: async () => { await player.value?.togglePlayPause() },
  exitPip: async () => { await player.value?.exitHostedPictureInPicture() },
})
onBeforeUnmount(unregisterControls)
// 播放器及播放列表读取自己的来源快照，后台页面不会改变其导航上下文。
provide(routeLocationKey, props.host.playerRoute)
watch(() => [currentRoute.name, currentRoute.params.id, currentRoute.query.t], () => {
  // 外部“从此帧播放”可定位同一实例；普通返回入口不携带旧 t。
  const seconds = parseResumeSecondsFromQuery(currentRoute.query.t)
  if (!visible.value || seconds === undefined || !player.value) return
  Object.assign(props.host.playerRoute, { query: { ...currentRoute.query }, hash: currentRoute.hash })
  void player.value.seekHostedPlayback(seconds, currentRoute.query.autoplay === "1")
}, { flush: "post" })

</script>

<template>
  <div
    v-show="visible"
    data-active-player-host
    class="absolute inset-0 z-[1] min-h-0 min-w-0"
    :aria-hidden="!visible"
    :inert="!visible || undefined"
  >
    <PlayerPage
      v-if="target"
      :key="target.movie.id"
      ref="player"
      :movie="target.movie"
      :autoplay="target.autoplay"
      :foreground="visible"
      @pip-change="host.setPipActive"
      @playing-change="host.setPlaying"
    />
  </div>
</template>
