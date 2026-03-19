<script setup lang="ts">
import { computed } from "vue"
import { DynamicScroller, DynamicScrollerItem } from "vue-virtual-scroller"
import type { Movie } from "@/lib/jav-library"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import MovieCard from "@/components/jav-library/MovieCard.vue"

interface MovieChunk {
  id: string
  items: Movie[]
}

const props = defineProps<{
  movies: Movie[]
  selectedMovieId: string
}>()

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
}>()

const CHUNK_SIZE = 64

const masonryColumnWidth = "10.5rem"
const masonryGap = "1.25rem"
const masonryMaxWidth = "calc(8 * 11.25rem + 7 * 1.25rem)"

const movieChunks = computed<MovieChunk[]>(() =>
  Array.from({ length: Math.ceil(props.movies.length / CHUNK_SIZE) }, (_, index) => ({
    id: `chunk-${index}`,
    items: props.movies.slice(index * CHUNK_SIZE, (index + 1) * CHUNK_SIZE),
  })),
)

const isMovieChunk = (value: unknown): value is MovieChunk =>
  typeof value === "object" &&
  value !== null &&
  "id" in value &&
  "items" in value &&
  Array.isArray(value.items)

const getChunk = (value: unknown): MovieChunk =>
  isMovieChunk(value)
    ? value
    : {
        id: "invalid-chunk",
        items: [],
      }

const getChunkDependencies = (chunk: MovieChunk) =>
  chunk.items.map((movie) =>
    [
      movie.id,
      movie.summary.length,
      movie.tags.length,
      movie.isFavorite ? "fav" : "std",
    ].join(":"),
  )
</script>

<template>
  <div v-if="props.movies.length" class="h-full min-h-0">
    <DynamicScroller
      :items="movieChunks"
      key-field="id"
      :min-item-size="2200"
      :buffer="600"
      class="h-full min-h-0 overflow-y-auto pr-2"
      list-class="flex flex-col gap-5"
      item-class="pb-5"
    >
      <template #default="{ item, index, active }">
        <DynamicScrollerItem
          :item="item"
          :active="active"
          :data-index="index"
          :size-dependencies="getChunkDependencies(getChunk(item))"
          :min-size="2200"
        >
          <div
            class="mx-auto w-full"
            :style="{
              columnWidth: masonryColumnWidth,
              columnGap: masonryGap,
              maxWidth: masonryMaxWidth,
            }"
          >
            <div
              v-for="movie in getChunk(item).items"
              :key="movie.id"
              class="mb-5 inline-block w-full break-inside-avoid align-top"
            >
              <MovieCard
                :movie="movie"
                :selected="movie.id === props.selectedMovieId"
                :show-favorite="false"
                @select="emit('select', $event)"
                @open-details="emit('openDetails', $event)"
                @open-player="emit('openPlayer', $event)"
              />
            </div>
          </div>
        </DynamicScrollerItem>
      </template>
    </DynamicScroller>
  </div>

  <Card v-else class="rounded-3xl border-border/70 bg-card/80">
    <CardHeader>
      <CardTitle>No matches found</CardTitle>
      <CardDescription>
        Try another query or switch to a different library tab.
      </CardDescription>
    </CardHeader>
    <CardContent class="text-sm text-muted-foreground">
      The current filters do not return any movies in this route view.
    </CardContent>
  </Card>
</template>
