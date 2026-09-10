<script setup lang="ts">
import { computed } from "vue"
import { useMediaQuery } from "@vueuse/core"
import type { ComicBook } from "@/domain/comic/types"
import ComicCard from "@/components/jav-library/comics/ComicCard.vue"
import {
  RETINA_DESKTOP_DENSITY_QUERY,
  resolveMovieGridDensity,
} from "@/lib/display-density"
import { buildMovieGridChunkStyle } from "@/lib/movie-grid-template"

const props = withDefaults(
  defineProps<{
    comics: readonly ComicBook[]
    selectedComicId?: string
    batchMode?: boolean
    batchSelectedIds?: readonly string[]
  }>(),
  {
    batchMode: false,
    batchSelectedIds: () => [],
  },
)

const emit = defineEmits<{
  openDetails: [comicId: string]
  openReader: [comicId: string, pageIndex: number]
  toggleFavorite: [payload: { comicId: string; nextValue: boolean }]
  toggleBatchSelect: [comicId: string]
}>()

const batchSelectedSet = computed(() => new Set(props.batchSelectedIds ?? []))
const retinaDesktopCompact = useMediaQuery(RETINA_DESKTOP_DENSITY_QUERY)
const comicGridDensity = computed(() => resolveMovieGridDensity(retinaDesktopCompact.value))

const comicGridStyle = computed(() =>
  buildMovieGridChunkStyle({
    minTrackWidth: comicGridDensity.value.minTrackWidth,
    gap: comicGridDensity.value.gap,
  }),
)
const comicCardFrameStyle = computed(() => ({
  maxWidth: comicGridDensity.value.cardMaxWidth,
}))
</script>

<template>
  <div
    data-virtual-comic-grid
    class="grid w-full overflow-x-hidden"
    :style="comicGridStyle"
  >
    <div
      v-for="comic in props.comics"
      :key="comic.id"
      data-comic-card-shell
      class="flex min-w-0 justify-center"
    >
      <div
        data-comic-card-frame
        class="w-full min-w-0"
        :style="comicCardFrameStyle"
      >
        <ComicCard
          :comic="comic"
          :selected="comic.id === props.selectedComicId"
          :batch-mode="props.batchMode"
          :batch-checked="batchSelectedSet.has(comic.id)"
          @open-details="emit('openDetails', $event)"
          @open-reader="(comicId, pageIndex) => emit('openReader', comicId, pageIndex)"
          @toggle-favorite="emit('toggleFavorite', $event)"
          @toggle-batch-select="emit('toggleBatchSelect', $event)"
        />
      </div>
    </div>
  </div>
</template>
