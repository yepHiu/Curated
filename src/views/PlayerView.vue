<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute } from "vue-router"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"
import PlayerPage from "@/components/jav-library/PlayerPage.vue"
import { recordMoviePlayed } from "@/lib/played-movies-storage"
import { parseResumeSecondsFromQuery } from "@/lib/playback-progress-storage"
import { useLibraryService } from "@/services/library-service"

const USE_WEB_API = import.meta.env.VITE_USE_WEB_API === "true"

const route = useRoute()
const libraryService = useLibraryService()
const { t } = useI18n()

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const hydrating = ref(false)

watch(
  movieId,
  async (id, _old, onCleanup) => {
    let cancelled = false
    onCleanup(() => { cancelled = true })
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
    const cancelPrefetch = start === undefined
      ? libraryService.prefetchMoviePlayback(id)
      : libraryService.prefetchMoviePlayback(id, start)
    if (cancelPrefetch) onCleanup(cancelPrefetch)
    if (libraryService.getMovieById(id)) {
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
  [selectedMovie, hydrating],
  ([movie, busy]) => {
    if (busy || !movie) {
      return
    }
    recordMoviePlayed(movie.id)
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
      v-else-if="selectedMovie"
      :movie="selectedMovie"
      :autoplay="route.query.autoplay === '1'"
    />
    <NotFoundState
      v-else
      :title="t('player.notFoundTitle')"
      :description="t('player.notFoundDesc')"
    />
  </div>
</template>
