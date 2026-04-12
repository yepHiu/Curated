<script setup lang="ts">
import type { Movie } from "@/domain/movie/types"
import MovieCard from "@/components/jav-library/MovieCard.vue"

defineProps<{
  title: string
  subtitle?: string
  movies: Movie[]
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
}>()
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-end justify-between gap-3">
      <div class="space-y-1">
        <h2 class="text-lg font-semibold tracking-tight text-foreground sm:text-xl">
          {{ title }}
        </h2>
        <p v-if="subtitle" class="text-sm text-muted-foreground">
          {{ subtitle }}
        </p>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
      <MovieCard
        v-for="movie in movies"
        :key="movie.id"
        :movie="movie"
        :show-favorite="false"
        poster-loading="lazy"
        poster-fetch-priority="low"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
      />
    </div>
  </section>
</template>
