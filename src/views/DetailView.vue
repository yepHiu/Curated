<script setup lang="ts">
import { computed } from "vue"
import { useRoute, useRouter } from "vue-router"
import DetailPage from "@/components/jav-library/DetailPage.vue"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"
import { buildMovieRouteQuery, getBrowseSourceMode } from "@/lib/library-query"
import { useLibraryService } from "@/services/library-service"

const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const selectedMovie = computed(() =>
  movieId.value ? libraryService.getMovieById(movieId.value) : undefined,
)
const relatedMovies = computed(() =>
  selectedMovie.value ? libraryService.getRelatedMovies(selectedMovie.value.id) : [],
)

const selectMovie = async (nextMovieId: string) => {
  await router.replace({
    name: "detail",
    params: { id: nextMovieId },
  })
}

const openDetails = async (nextMovieId: string) => {
  await router.push({
    name: "detail",
    params: { id: nextMovieId },
    query: buildMovieRouteQuery(route.query, getBrowseSourceMode(route.query), nextMovieId),
  })
}

const openPlayer = async (nextMovieId: string) => {
  await router.push({
    name: "player",
    params: { id: nextMovieId },
    query: buildMovieRouteQuery(route.query, getBrowseSourceMode(route.query), nextMovieId),
  })
}

const toggleFavorite = (payload: { movieId: string; nextValue: boolean }) => {
  libraryService.toggleFavorite(payload.movieId, payload.nextValue)
}
</script>

<template>
  <div class="h-full overflow-y-auto pr-2">
    <DetailPage
      v-if="selectedMovie"
      :movie="selectedMovie"
      :related-movies="relatedMovies"
      @select="selectMovie"
      @open-details="openDetails"
      @open-player="openPlayer"
      @toggle-favorite="toggleFavorite"
    />
    <NotFoundState
      v-else
      title="Movie not found"
      description="The requested detail page does not match any movie in the current mock library."
    />
  </div>
</template>
