<script setup lang="ts">
import { computed, onBeforeUnmount, provide, ref, watch } from "vue"
import { routeLocationKey, useRoute, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { Pause, Play, Square, PictureInPicture2 } from "lucide-vue-next"
import PlayerPage from "@/components/jav-library/PlayerPage.vue"
import { Button } from "@/components/ui/button"
import { parseResumeSecondsFromQuery } from "@/lib/playback-progress-storage"
import type { PlaybackHost } from "@/composables/use-playback-host"

const props = defineProps<{ host: PlaybackHost }>()
const { t } = useI18n()
const router = useRouter()
const currentRoute = useRoute()
const player = ref<InstanceType<typeof PlayerPage> | null>(null)
const target = props.host.target
const visible = props.host.visible
const playing = props.host.playing
const isDev = import.meta.env.DEV
const unregisterStop = props.host.registerMediaStop(() => {
  // 所有停止路径（包括认证失效）同步保存、暂停，再移除组件。
  player.value?.stopHostedPlayback()
})
onBeforeUnmount(unregisterStop)
const movieId = computed(() => target.value?.movie.id ?? "")
// 播放器及播放列表读取自己的来源快照，后台页面不会改变其导航上下文。
provide(routeLocationKey, props.host.playerRoute)
watch(() => [currentRoute.name, currentRoute.params.id, currentRoute.query.t], () => {
  // 外部“从此帧播放”可定位同一实例；普通返回入口不携带旧 t。
  const seconds = parseResumeSecondsFromQuery(currentRoute.query.t)
  if (!visible.value || seconds === undefined || !player.value) return
  Object.assign(props.host.playerRoute, { query: { ...currentRoute.query }, hash: currentRoute.hash })
  void player.value.seekHostedPlayback(seconds, currentRoute.query.autoplay === "1")
}, { flush: "post" })

/** 返回同一播放器，去掉过时的启动指令，不寻址或重新创建播放会话。 */
function returnToPlayer() {
  const query = { ...props.host.playerRoute.query }
  delete query.t
  delete query.autoplay
  void router.push({ name: "player", params: { id: movieId.value }, query, hash: props.host.playerRoute.hash })
}

/** 后台播放的暂停操作只作用于活动实例，不劫持当前页面的键盘操作。 */
function togglePlayback() {
  void player.value?.togglePlayPause()
}

/** 在移除实例前立即暂停并保存进度，再由组件卸载释放媒体和 HLS 会话。 */
function stopPlayback() {
  props.host.stop()
}
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
  <div
    v-if="target && !visible"
    data-background-playback
    class="absolute inset-x-3 z-30 flex min-w-0 items-center gap-1 rounded-xl border border-border bg-card p-2 text-card-foreground shadow-lg sm:left-auto sm:right-4 sm:max-w-sm"
    :class="isDev ? 'bottom-14' : 'bottom-3'"
  >
    <Button type="button" variant="ghost" class="h-auto min-h-11 min-w-0 flex-1 justify-start gap-2 px-2" @click="returnToPlayer">
      <PictureInPicture2 class="size-4 shrink-0 text-primary" aria-hidden="true" />
      <span class="min-w-0 text-left">
        <span class="block text-xs text-muted-foreground">{{ t('player.backgroundPlayback') }}</span>
        <span class="block truncate text-sm">{{ target.movie.title || target.movie.code }}</span>
      </span>
    </Button>
    <Button type="button" variant="ghost" size="icon" class="size-11 shrink-0" :aria-label="playing ? t('player.ariaPause') : t('player.ariaPlay')" @click="togglePlayback">
      <Pause v-if="playing" class="size-4" aria-hidden="true" />
      <Play v-else class="size-4" aria-hidden="true" />
    </Button>
    <Button type="button" variant="ghost" size="icon" class="size-11 shrink-0" :aria-label="t('player.stopBackgroundPlayback')" @click="stopPlayback">
      <Square class="size-4" aria-hidden="true" />
    </Button>
  </div>
</template>
