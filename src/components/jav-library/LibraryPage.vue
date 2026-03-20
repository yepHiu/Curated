<script setup lang="ts">
import { computed } from "vue"
import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import VirtualMovieMasonry from "@/components/jav-library/VirtualMovieMasonry.vue"

const props = defineProps<{
  mode: LibraryMode
  allMovies: Movie[]
  visibleMovies: Movie[]
  selectedMovie?: Movie
  activeTab: LibraryTab
}>()

const emit = defineEmits<{
  "update:activeTab": [value: LibraryTab]
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId?: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()

const popularTags = computed(() => {
  const collectedTags: string[] = []

  props.allMovies.forEach((movie) => {
    movie.tags.forEach((tag) => {
      if (collectedTags.indexOf(tag) === -1) {
        collectedTags.push(tag)
      }
    })
  })

  return collectedTags.slice(0, 8)
})

const handleTabChange = (value: string | number) => {
  emit("update:activeTab", String(value) as LibraryTab)
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-7">
    <Card
      v-if="props.mode === 'tags'"
      class="rounded-3xl border-border/70 bg-card/85 shadow-lg shadow-black/5"
    >
      <CardHeader>
        <CardTitle>Popular tag clusters</CardTitle>
        <CardDescription>
          Tags are shown as ready-made groups for the future filter and sorting service layer.
        </CardDescription>
      </CardHeader>
      <CardContent class="flex flex-wrap gap-2">
        <Badge
          v-for="tag in popularTags"
          :key="tag"
          variant="secondary"
          class="rounded-full border border-border/60 bg-secondary/70 px-3 py-1"
        >
          {{ tag }}
        </Badge>
      </CardContent>
    </Card>

    <Tabs
      :model-value="props.activeTab"
      class="gap-5"
      @update:model-value="handleTabChange"
    >
      <TabsList class="h-auto w-fit flex-wrap rounded-2xl bg-muted/60 p-1">
        <TabsTrigger value="all" class="rounded-xl px-4 py-2">
          All titles
        </TabsTrigger>
        <TabsTrigger value="new" class="rounded-xl px-4 py-2">
          New
        </TabsTrigger>
        <TabsTrigger value="favorites" class="rounded-xl px-4 py-2">
          Favorites
        </TabsTrigger>
        <TabsTrigger value="top-rated" class="rounded-xl px-4 py-2">
          Top rated
        </TabsTrigger>
      </TabsList>
    </Tabs>

    <div class="min-h-0 flex-1">
      <VirtualMovieMasonry
        :movies="props.visibleMovies"
        :selected-movie-id="props.selectedMovie?.id"
        @select="emit('select', $event)"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
        @toggle-favorite="emit('toggleFavorite', $event)"
      />
    </div>
  </div>
</template>
