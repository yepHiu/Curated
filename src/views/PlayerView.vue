<script setup lang="ts">
import { computed } from "vue"
import { useRoute } from "vue-router"
import PlayerPage from "@/components/jav-library/PlayerPage.vue"
import { getMovieById } from "@/lib/jav-library"

const route = useRoute()

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const selectedMovie = computed(() => getMovieById(movieId.value))
</script>

<template>
  <div class="h-full overflow-y-auto pr-2">
    <PlayerPage :movie="selectedMovie" />
  </div>
</template>
