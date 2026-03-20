<script setup lang="ts">
import type { Movie } from "@/domain/movie/types"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import DetailPanel from "@/components/jav-library/DetailPanel.vue"
import MovieGrid from "@/components/jav-library/MovieGrid.vue"

defineProps<{
  movie: Movie
  relatedMovies: Movie[]
}>()

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()
</script>

<template>
  <div class="flex flex-col gap-6">
    <DetailPanel
      :movie="movie"
      @open-player="emit('openPlayer', $event)"
      @toggle-favorite="emit('toggleFavorite', $event)"
    />

    <Card class="rounded-3xl border-border/70 bg-card/85">
      <CardHeader>
        <CardTitle>Preview gallery</CardTitle>
        <CardDescription>
          Placeholder surfaces for poster, thumb strip, and future frame captures.
        </CardDescription>
      </CardHeader>
      <CardContent class="grid max-w-[52rem] gap-4 md:grid-cols-3">
        <div
          v-for="index in 3"
          :key="index"
          class="aspect-[16/9] rounded-[1.25rem] border border-border/70 bg-gradient-to-br from-primary/20 via-accent/40 to-card"
        />
      </CardContent>
    </Card>

    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-1">
        <h3 class="text-xl font-semibold">Related titles</h3>
        <p class="text-sm text-muted-foreground">
          More cards using the same movie grid system and selection contract.
        </p>
      </div>

      <MovieGrid
        :movies="relatedMovies"
        :selected-movie-id="movie.id"
        @select="emit('select', $event)"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
        @toggle-favorite="emit('toggleFavorite', $event)"
      />
    </div>
  </div>
</template>
