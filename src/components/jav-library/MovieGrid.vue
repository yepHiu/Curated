<script setup lang="ts">
import { computed } from "vue"
import type { Movie } from "@/domain/movie/types"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import MovieCard from "@/components/jav-library/MovieCard.vue"

const props = withDefaults(
  defineProps<{
    movies: Movie[]
    selectedMovieId: string
    /** 最多展示张数；详情推荐区由父组件切片时可仍用默认 */
    maxVisible?: number
  }>(),
  { maxVisible: 8 },
)

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()

const visibleMovies = computed(() => props.movies.slice(0, props.maxVisible))
</script>

<template>
  <div v-if="visibleMovies.length" class="overflow-x-auto pb-1">
    <div class="grid auto-cols-[11.25rem] grid-flow-col justify-start gap-5">
      <div v-for="movie in visibleMovies" :key="movie.id" class="w-[11.25rem]">
        <MovieCard
          :movie="movie"
          :selected="movie.id === props.selectedMovieId"
          @select="emit('select', $event)"
          @open-details="emit('openDetails', $event)"
          @open-player="emit('openPlayer', $event)"
          @toggle-favorite="emit('toggleFavorite', $event)"
        />
      </div>
    </div>
  </div>

  <Card v-else class="rounded-3xl border-border/70 bg-card/80">
    <CardHeader>
      <CardTitle>No matches found</CardTitle>
      <CardDescription>
        Try another query or switch to a different library tab.
      </CardDescription>
    </CardHeader>
    <CardContent class="text-sm text-muted-foreground">
      The shell is ready for future virtualized data, but your current filters do not return
      any movies.
    </CardContent>
  </Card>
</template>
