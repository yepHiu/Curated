<script setup lang="ts">
import type { ComicBook } from "@/domain/comic/types"
import ComicCard from "@/components/jav-library/comics/ComicCard.vue"
import { buildAutoFillMovieGridTemplate } from "@/lib/movie-grid-template"

const props = defineProps<{
  comics: readonly ComicBook[]
  selectedComicId?: string
}>()

const emit = defineEmits<{
  openDetails: [comicId: string]
  openReader: [comicId: string, pageIndex: number]
  toggleFavorite: [payload: { comicId: string; nextValue: boolean }]
}>()

const comicGridStyle = {
  gridTemplateColumns: buildAutoFillMovieGridTemplate("var(--movie-grid-min-track)"),
  columnGap: "var(--movie-grid-gap)",
  rowGap: "var(--movie-grid-gap)",
}
</script>

<template>
  <div
    data-virtual-comic-grid
    class="grid w-full overflow-x-hidden"
    :style="comicGridStyle"
  >
    <ComicCard
      v-for="comic in props.comics"
      :key="comic.id"
      :comic="comic"
      :selected="comic.id === props.selectedComicId"
      @open-details="emit('openDetails', $event)"
      @open-reader="(comicId, pageIndex) => emit('openReader', comicId, pageIndex)"
      @toggle-favorite="emit('toggleFavorite', $event)"
    />
  </div>
</template>
