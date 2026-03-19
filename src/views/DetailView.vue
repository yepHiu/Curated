<script setup lang="ts">
import { computed } from "vue"
import { useRoute, useRouter } from "vue-router"
import DetailPage from "@/components/jav-library/DetailPage.vue"
import { getMovieById, getRelatedMovies } from "@/lib/jav-library"

const route = useRoute()
const router = useRouter()

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const selectedMovie = computed(() => getMovieById(movieId.value))
const relatedMovies = computed(() => getRelatedMovies(selectedMovie.value.id))

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
  })
}

const openPlayer = async (nextMovieId: string) => {
  await router.push({
    name: "player",
    params: { id: nextMovieId },
  })
}
</script>

<template>
  <div class="h-full overflow-y-auto pr-2">
    <DetailPage
      :movie="selectedMovie"
      :related-movies="relatedMovies"
      @select="selectMovie"
      @open-details="openDetails"
      @open-player="openPlayer"
    />
  </div>
</template>
