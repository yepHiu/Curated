<script setup lang="ts">
import type { ComicBook } from "@/domain/comic/types"
import ComicCard from "@/components/jav-library/comics/ComicCard.vue"

const props = defineProps<{
  comics: readonly ComicBook[]
  selectedComicId?: string
}>()

const emit = defineEmits<{
  openDetails: [comicId: string]
  openReader: [comicId: string, pageIndex: number]
  toggleFavorite: [payload: { comicId: string; nextValue: boolean }]
}>()
</script>

<template>
  <div
    data-virtual-comic-grid
    class="grid grid-cols-[repeat(auto-fill,minmax(10.5rem,1fr))] gap-4 sm:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]"
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
