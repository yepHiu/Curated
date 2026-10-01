<script setup lang="ts">
import type { Movie } from "@/domain/movie/types"
import MovieCard from "@/components/jav-library/MovieCard.vue"

defineProps<{
  title: string
  movies: Movie[]
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
}>()
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex min-h-11 flex-wrap items-center justify-between gap-x-3 gap-y-2 sm:min-h-8">
      <div class="min-w-0">
        <h2 class="text-lg font-semibold tracking-tight text-foreground sm:text-xl">
          {{ title }}
        </h2>
      </div>

      <div v-if="$slots.action" class="flex flex-wrap items-center gap-1">
        <slot name="action" />
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
      <template v-for="movie in movies" :key="movie.id">
        <slot name="item" :movie="movie">
          <MovieCard
            :movie="movie"
            :show-favorite="false"
            poster-loading="lazy"
            poster-fetch-priority="low"
            @open-details="emit('openDetails', $event)"
            @open-player="emit('openPlayer', $event)"
          />
        </slot>
      </template>
    </div>
  </section>
</template>
