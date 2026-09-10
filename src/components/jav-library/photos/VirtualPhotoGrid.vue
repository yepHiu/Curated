<script setup lang="ts">
import { computed } from "vue"
import { useMediaQuery } from "@vueuse/core"
import type { PhotoBook } from "@/domain/photo/types"
import PhotoCard from "@/components/jav-library/photos/PhotoCard.vue"
import {
  RETINA_DESKTOP_DENSITY_QUERY,
  resolveMovieGridDensity,
} from "@/lib/display-density"
import { buildMovieGridChunkStyle } from "@/lib/movie-grid-template"

const props = defineProps<{
  photos: readonly PhotoBook[]
}>()

const emit = defineEmits<{
  openDetails: [photoId: string]
  openViewer: [photoId: string, pageIndex: number]
}>()

const retinaDesktopCompact = useMediaQuery(RETINA_DESKTOP_DENSITY_QUERY)
const photoGridDensity = computed(() => resolveMovieGridDensity(retinaDesktopCompact.value))

const photoGridStyle = computed(() =>
  buildMovieGridChunkStyle({
    minTrackWidth: photoGridDensity.value.minTrackWidth,
    gap: photoGridDensity.value.gap,
  }),
)
const photoCardFrameStyle = computed(() => ({
  maxWidth: photoGridDensity.value.cardMaxWidth,
}))
</script>

<template>
  <div
    data-virtual-photo-grid
    class="grid w-full overflow-x-hidden"
    :style="photoGridStyle"
  >
    <div
      v-for="photo in props.photos"
      :key="photo.id"
      data-photo-card-shell
      class="flex min-w-0 justify-center"
    >
      <div
        data-photo-card-frame
        class="w-full min-w-0"
        :style="photoCardFrameStyle"
      >
        <PhotoCard
          :photo="photo"
          @open-details="emit('openDetails', $event)"
          @open-viewer="(photoId, pageIndex) => emit('openViewer', photoId, pageIndex)"
        />
      </div>
    </div>
  </div>
</template>
