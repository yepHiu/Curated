<script setup lang="ts">
import { computed } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { useI18n } from "vue-i18n"
import { Pause, PictureInPicture2, Play, Square, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { usePlaybackHost } from "@/composables/use-playback-host"
import { useActivePlaybackSession } from "@/composables/use-active-playback-session"

const props = withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })
const { t } = useI18n()
const route = useRoute()
const { activePlaybackSession: resumeSession, playbackSessionSnapshot, dismissActivePlaybackSession } = useActivePlaybackSession()
const playbackHost = usePlaybackHost()
const livePlayback = computed(() => Boolean(playbackHost?.target.value && playbackHost.pipActive.value && !playbackHost.visible.value))
const playing = computed(() => playbackHost?.playing.value ?? false)
// 原有续播规则会隐藏起播、近结尾、结束或手动隐藏的快照，活动小窗仍须保留操作入口。
const activePlaybackSession = computed(() => {
  if (!livePlayback.value) return resumeSession.value
  const movie = playbackHost?.target.value?.movie
  if (!movie) return null
  const snapshot = playbackSessionSnapshot.value
  if (snapshot?.movieId === movie.id) return snapshot
  return { movieId: movie.id, title: movie.title || movie.code, positionSec: 0, progressPercent: 0, resumeRouteTarget: { name: "player", params: { id: movie.id } } }
})
const entryLabel = computed(() => livePlayback.value ? t("player.backgroundPlayback") : t("nav.continuePlayback"))

/** 控制只操作宿主活动实例，不改变当前浏览页面。 */
function togglePlayback() {
  void playbackHost?.togglePlayback()
}

/** 停止先保存并暂停，再释放原生小窗和会话。 */
function stopPlayback() {
  playbackHost?.stop()
}

function formatSidebarPlaybackClock(seconds: number): string {
  const total = Math.max(0, Math.floor(Number.isFinite(seconds) ? seconds : 0))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const secs = total % 60
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(secs).padStart(2, "0")}`
  }
  return `${minutes}:${String(secs).padStart(2, "0")}`
}

/** 活动实例返回时不携带快照 t，避免重新 seek 到侧栏上次发布的位置。 */
const activePlaybackResumeTarget = computed(() => {
  const active = activePlaybackSession.value
  if (!active || !playbackHost?.hasMovie(active.movieId)) return active?.resumeRouteTarget
  const query = { ...playbackHost.playerRoute.query }
  delete query.t
  delete query.autoplay
  return { name: "player", params: { id: active.movieId }, query, hash: playbackHost.playerRoute.hash }
})

const isSameActivePlayerRoute = computed(() => {
  const active = activePlaybackSession.value
  if (!active || route.name !== "player") return false
  return typeof route.params.id === "string" && route.params.id === active.movieId
})

const showActivePlaybackEntry = computed(() =>
  Boolean(activePlaybackSession.value && !isSameActivePlayerRoute.value),
)

const activePlaybackTimeLabel = computed(() => {
  const active = activePlaybackSession.value
  if (!active) return ""
  return formatSidebarPlaybackClock(active.positionSec)
})

const activePlaybackProgressValue = computed(() =>
  Math.max(0, Math.min(100, activePlaybackSession.value?.progressPercent ?? 0)),
)

const activePlaybackAriaLabel = computed(() => {
  const active = activePlaybackSession.value
  if (!active) return entryLabel.value
  return t(livePlayback.value ? "nav.currentPlaybackAria" : "nav.continuePlaybackAria", {
    title: active.title,
    time: activePlaybackTimeLabel.value,
  })
})

const activePlaybackCompactTitle = computed(() => {
  const active = activePlaybackSession.value
  if (!active) return entryLabel.value
  return `${entryLabel.value}: ${active.title} · ${activePlaybackTimeLabel.value}`
})
</script>

<template>
  <section
    v-if="showActivePlaybackEntry && activePlaybackSession"
    class="mb-2 min-w-0"
    :class="props.compact ? 'flex flex-col items-center gap-1' : livePlayback ? 'flex flex-col rounded-lg border border-border/60 bg-background/45' : 'flex flex-col'"
  >
    <div
      v-if="livePlayback"
      data-active-playback-controls
      class="flex gap-1"
      :class="props.compact ? 'flex-col items-center' : 'justify-end px-3 pt-2'"
    >
      <RouterLink
        data-active-playback-return
        :data-active-playback-compact="props.compact ? '' : undefined"
        :to="activePlaybackResumeTarget ?? activePlaybackSession.resumeRouteTarget"
        class="relative inline-flex shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
        :class="props.compact ? 'size-11' : 'size-11 lg:size-8'"
        :title="props.compact ? activePlaybackCompactTitle : activePlaybackAriaLabel"
        :aria-label="activePlaybackAriaLabel"
      >
        <PictureInPicture2 class="size-4" aria-hidden="true" />
        <span v-if="props.compact" class="absolute inset-x-1.5 bottom-1.5 h-0.5 overflow-hidden rounded-full bg-primary-foreground/20" aria-hidden="true">
          <span class="block h-full rounded-full bg-primary-foreground" :style="{ width: `${activePlaybackProgressValue}%` }" />
        </span>
      </RouterLink>
      <Button
        type="button"
        variant="ghost"
        :class="props.compact ? 'size-11' : 'size-11 lg:size-8'"
        :aria-label="playing ? t('player.ariaPause') : t('player.ariaPlay')"
        :title="playing ? t('player.ariaPause') : t('player.ariaPlay')"
        @click="togglePlayback"
      >
        <Pause v-if="playing" class="size-4" aria-hidden="true" />
        <Play v-else class="size-4" aria-hidden="true" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        :class="props.compact ? 'size-11' : 'size-11 lg:size-8'"
        :aria-label="t('player.stopBackgroundPlayback')"
        :title="t('player.stopBackgroundPlayback')"
        @click="stopPlayback"
      >
        <Square class="size-4" aria-hidden="true" />
      </Button>
    </div>
    <div
      v-if="!props.compact"
      class="relative min-w-0 rounded-lg"
      :class="livePlayback ? '' : 'border border-border/60 bg-background/45'"
    >
      <RouterLink
        data-active-playback-card
        :to="activePlaybackResumeTarget ?? activePlaybackSession.resumeRouteTarget"
        class="group flex min-w-0 flex-col gap-2 rounded-lg px-3 py-2.5 text-sidebar-foreground outline-none transition-colors hover:bg-sidebar-accent/60 focus-visible:ring-2 focus-visible:ring-ring/60"
        :aria-label="activePlaybackAriaLabel"
      >
        <span
          v-if="!livePlayback"
          class="flex min-w-0 items-center justify-between gap-2 pr-10"
        >
          <span class="inline-flex min-w-0 items-center gap-2 text-xs font-medium text-primary">
            <span class="size-2 shrink-0 rounded-full bg-primary shadow-[0_0_0_4px_hsl(var(--primary)/0.16)]" aria-hidden="true" />
            <span class="truncate">{{ entryLabel }}</span>
          </span>
          <span class="inline-flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Play class="size-4 fill-current" aria-hidden="true" />
          </span>
        </span>
        <span class="line-clamp-2 min-w-0 text-sm font-medium leading-snug">
          {{ activePlaybackSession.title }}
        </span>
        <span class="flex min-w-0 items-center justify-between gap-2 text-xs text-muted-foreground">
          <span class="truncate">
            {{ t(livePlayback ? "nav.currentPlaybackAt" : "nav.continuePlaybackAt", { time: activePlaybackTimeLabel }) }}
          </span>
          <span class="shrink-0 tabular-nums">{{ Math.round(activePlaybackProgressValue) }}%</span>
        </span>
        <Progress
          :model-value="activePlaybackProgressValue"
          class="h-1 bg-primary/15"
          aria-hidden="true"
        />
      </RouterLink>
      <Button
        v-if="!livePlayback"
        type="button"
        variant="ghost"
        size="icon"
        data-active-playback-dismiss
        class="absolute right-3 top-2.5 size-8 rounded-lg text-muted-foreground hover:text-foreground"
        :aria-label="t('nav.dismissContinuePlayback')"
        @click="dismissActivePlaybackSession(activePlaybackSession.movieId)"
      >
        <X class="size-4" aria-hidden="true" />
      </Button>
    </div>

    <RouterLink
      v-else-if="!livePlayback"
      data-active-playback-compact
      :to="activePlaybackResumeTarget ?? activePlaybackSession.resumeRouteTarget"
      class="relative mx-auto inline-flex size-11 shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/45 text-primary outline-none transition-colors hover:bg-sidebar-accent/60 focus-visible:ring-2 focus-visible:ring-ring/60"
      :title="activePlaybackCompactTitle"
      :aria-label="activePlaybackAriaLabel"
    >
      <Play class="size-4 fill-current" aria-hidden="true" />
      <span class="absolute inset-x-1.5 bottom-1.5 h-0.5 overflow-hidden rounded-full bg-primary/15" aria-hidden="true">
        <span
          class="block h-full rounded-full bg-primary"
          :style="{ width: `${activePlaybackProgressValue}%` }"
        />
      </span>
    </RouterLink>
  </section>
</template>
