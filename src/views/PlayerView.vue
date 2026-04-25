<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useRoute } from "vue-router"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"
import PlayerPage from "@/components/jav-library/PlayerPage.vue"
import { recordMoviePlayed } from "@/lib/played-movies-storage"
import { useLibraryService } from "@/services/library-service"

const USE_WEB_API = import.meta.env.VITE_USE_WEB_API === "true"

const route = useRoute()
const libraryService = useLibraryService()

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const hydrating = ref(false)

watch(
  movieId,
  async (id) => {
    if (!id) {
      hydrating.value = false
      return
    }
    if (libraryService.getMovieById(id)) {
      hydrating.value = false
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
      hydrating.value = false
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
      正在加载播放目标…
    </div>
    <PlayerPage
      v-else-if="selectedMovie"
      :movie="selectedMovie"
      :autoplay="route.query.autoplay === '1'"
    />
    <NotFoundState
      v-else
      title="Player target not found"
      description="This player route points to a movie id that is not available in the current library."
    />
  </div>
</template>
