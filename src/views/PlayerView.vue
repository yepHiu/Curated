<script setup lang="ts">
import { computed } from "vue"
import { useRoute } from "vue-router"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"
import PlayerPage from "@/components/jav-library/PlayerPage.vue"
import { useLibraryService } from "@/services/library-service"

const route = useRoute()
const libraryService = useLibraryService()

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const selectedMovie = computed(() =>
  movieId.value ? libraryService.getMovieById(movieId.value) : undefined,
)
</script>

<template>
  <div class="h-full overflow-y-auto pr-2">
    <PlayerPage v-if="selectedMovie" :movie="selectedMovie" />
    <NotFoundState
      v-else
      title="Player target not found"
      description="This player route points to a movie id that is not available in the current library."
    />
  </div>
</template>
