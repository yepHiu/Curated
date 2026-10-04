<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute } from "vue-router"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"
import PlayerPage from "@/components/jav-library/PlayerPage.vue"
import { recordMoviePlayed } from "@/lib/played-movies-storage"
import { parseResumeSecondsFromQuery } from "@/lib/playback-progress-storage"
import { usePlaybackHost } from "@/composables/use-playback-host"
import { useLibraryService } from "@/services/library-service"

const USE_WEB_API = import.meta.env.VITE_USE_WEB_API === "true"

const route = useRoute()
const libraryService = useLibraryService()
const playbackHost = usePlaybackHost()
const { t } = useI18n()

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const fileId = computed(/* 请求文件变化时加载新播放目标，并取消旧目标的状态更新。 */ () => typeof route.query.fileId === "string" ? route.query.fileId : "")
const hydrating = ref(false)

watch(
  [movieId, fileId],
  /* 请求文件变化时加载新播放目标，并取消旧目标的状态更新。 */ async ([id, selectedFileId], _old, onCleanup) => {
    let cancelled = false
    onCleanup(() => { cancelled = true })
    // 返回当前宿主影片直接复用，不能预热出第二个 HLS 会话。
    if (id && playbackHost?.hasTarget(id, selectedFileId)) {
      hydrating.value = false
      return
    }
    if (!id) {
      hydrating.value = false
      return
    }
    // Warm the playback descriptor while this view hydrates the movie, so the
    // descriptor GET (which also boots the server-side HLS session) overlaps
    // movie hydration and the player page mount instead of serializing after
    // them. PlayerView only loads after the auth guard passes, so locked
    // startup never touches protected playback endpoints.
    const start = parseResumeSecondsFromQuery(route.query.t)
    const cancelPrefetch = selectedFileId ? undefined : start === undefined
      ? libraryService.prefetchMoviePlayback(id)
      : libraryService.prefetchMoviePlayback(id, start)
    if (cancelPrefetch) onCleanup(cancelPrefetch)
    const cached = libraryService.getMovieById(id)
    if (cached && ((cached.fileCount ?? 1) <= 1 || cached.files !== undefined)) {
      if (!cancelled) hydrating.value = false
      return
    }
    if (!USE_WEB_API) {
      hydrating.value = false
      return
    }
    hydrating.value = true
    try {
      await libraryService.ensureMovieCached(id)
    } finally {
      if (!cancelled) hydrating.value = false
    }
  },
  { immediate: true },
)

const selectedMovie = computed(() =>
  movieId.value ? libraryService.getMovieById(movieId.value) : undefined,
)

watch(
  [selectedMovie, hydrating, fileId],
  ([movie, busy]) => {
    if (busy || !movie) {
      return
    }
    recordMoviePlayed(movie.id)
    playbackHost?.start(movie, route.query.autoplay === "1", route)
  },
  { immediate: true },
)
</script>

<template>
  <div class="h-full overflow-hidden">
    <div
      v-if="hydrating"
      class="rounded-3xl border border-border/70 bg-card/80 p-8 text-sm text-muted-foreground"
    >
      {{ t("player.loadingTarget") }}
    </div>
    <PlayerPage
      v-else-if="selectedMovie && !playbackHost"
      :key="`${selectedMovie.id}:${fileId}`"
      :movie="selectedMovie"
      :autoplay="route.query.autoplay === '1'"
    />
    <NotFoundState
      v-else-if="!selectedMovie"
      :title="t('player.notFoundTitle')"
      :description="t('player.notFoundDesc')"
    />
  </div>
</template>
