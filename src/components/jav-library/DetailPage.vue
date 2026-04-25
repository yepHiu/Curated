<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { PatchMovieBody } from "@/api/types"
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
import MediaStill from "@/components/jav-library/MediaStill.vue"
import PreviewImageViewer from "@/components/jav-library/PreviewImageViewer.vue"
import MovieCommentSection from "@/components/jav-library/MovieCommentSection.vue"
import { useRelatedVisibleCount } from "@/composables/use-related-visible-count"

type PreviewImageLoadPayload = {
  naturalWidth: number
  naturalHeight: number
}

const PREVIEW_FALLBACK_ASPECT_RATIO = 1.7778
const PREVIEW_MIN_ASPECT_RATIO = 0.5
const PREVIEW_MAX_ASPECT_RATIO = 2.4

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    movie: Movie
    relatedMovies: Movie[]
    // Fuzzy suggestions for user tags, including library-wide user tags and movie metadata tags.
    userTagSuggestions?: readonly string[]
    metadataRefreshBusy?: boolean
  }>(),
  { metadataRefreshBusy: false, userTagSuggestions: () => [] },
)

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
  updateUserRating: [payload: { movieId: string; value: number | null }]
  updateUserTags: [payload: { movieId: string; tags: string[] }]
  browseByTag: [payload: { tag: string }]
  browseByActor: [payload: { actor: string }]
  browseByStudio: [payload: { studio: string }]
  updateMetadataTags: [payload: { movieId: string; tags: string[] }]
  deleteMovie: [movieId: string]
  restoreMovie: [movieId: string]
  deleteMoviePermanently: [movieId: string]
  refreshMetadata: [movieId: string]
  revealInFileManager: [movieId: string]
  patchMovieDisplay: [body: PatchMovieBody, done: (err?: unknown) => void]
}>()

const commentReadonly = computed(() => Boolean(props.movie.trashedAt?.trim()))

const previewImages = computed(() => props.movie.previewImages?.slice(0, 18) ?? [])
const hasPreviews = computed(() => previewImages.value.length > 0)
const previewAspectRatios = ref<Record<number, number>>({})

watch(
  previewImages,
  () => {
    previewAspectRatios.value = {}
  },
  { immediate: true },
)

function clampPreviewAspectRatio(raw: number) {
  if (!Number.isFinite(raw) || raw <= 0) return PREVIEW_FALLBACK_ASPECT_RATIO
  return Number(
    Math.min(PREVIEW_MAX_ASPECT_RATIO, Math.max(PREVIEW_MIN_ASPECT_RATIO, raw)).toFixed(4),
  )
}

function previewAspectRatioFor(index: number) {
  return previewAspectRatios.value[index] ?? PREVIEW_FALLBACK_ASPECT_RATIO
}

function previewCardStyle(index: number) {
  return {
    aspectRatio: String(previewAspectRatioFor(index)),
  }
}

function updatePreviewAspectRatio(index: number, payload: PreviewImageLoadPayload) {
  const { naturalWidth, naturalHeight } = payload
  if (naturalWidth <= 0 || naturalHeight <= 0) return
  previewAspectRatios.value = {
    ...previewAspectRatios.value,
    [index]: clampPreviewAspectRatio(naturalWidth / naturalHeight),
  }
}

// Viewer order is poster first, then deduped preview images.
const viewerImages = computed(() => {
  const poster = (props.movie.coverUrl || props.movie.thumbUrl || "").trim()
  const out: string[] = []
  if (poster) out.push(poster)
  for (const url of previewImages.value) {
    const value = (url || "").trim()
    if (!value || value === poster) continue
    out.push(value)
  }
  return out
})

const hasViewerImages = computed(() => viewerImages.value.length > 0)

const layoutRootRef = ref<HTMLElement | null>(null)
const { visibleCount } = useRelatedVisibleCount(layoutRootRef)

const relatedMoviesForGrid = computed(() =>
  props.relatedMovies.slice(0, visibleCount.value),
)

const previewViewerOpen = ref(false)
const previewViewerStartIndex = ref(0)

function openPreviewViewer(galleryIndex: number) {
  const poster = (props.movie.coverUrl || props.movie.thumbUrl || "").trim()
  const previews = previewImages.value
  const raw = previews[galleryIndex]?.trim()
  if (!raw) return

  const idxByUrl = viewerImages.value.indexOf(raw)
  if (idxByUrl >= 0) {
    previewViewerStartIndex.value = idxByUrl
    previewViewerOpen.value = true
    return
  }

  if (poster && raw === poster) {
    previewViewerStartIndex.value = 0
    previewViewerOpen.value = true
    return
  }

  let viewerIdx = poster ? 1 : 0
  for (let i = 0; i < previews.length; i++) {
    const value = previews[i]?.trim()
    if (!value || (poster && value === poster)) continue
    if (i === galleryIndex) {
      previewViewerStartIndex.value = viewerIdx
      previewViewerOpen.value = true
      return
    }
    viewerIdx++
  }

  previewViewerStartIndex.value = 0
  previewViewerOpen.value = true
}

function openPosterInViewer() {
  if (!hasViewerImages.value) return
  previewViewerStartIndex.value = 0
  previewViewerOpen.value = true
}
</script>

<template>
  <div ref="layoutRootRef" class="flex min-w-0 w-full flex-col gap-6">
    <DetailPanel
      :movie="movie"
      :user-tag-suggestions="props.userTagSuggestions"
      :metadata-refresh-busy="props.metadataRefreshBusy"
      @open-player="emit('openPlayer', $event)"
      @update-user-rating="emit('updateUserRating', $event)"
      @update-user-tags="emit('updateUserTags', $event)"
      @browse-by-tag="emit('browseByTag', $event)"
      @browse-by-actor="emit('browseByActor', $event)"
      @browse-by-studio="emit('browseByStudio', $event)"
      @update-metadata-tags="emit('updateMetadataTags', $event)"
      @delete-movie="emit('deleteMovie', $event)"
      @restore-movie="emit('restoreMovie', $event)"
      @delete-movie-permanently="emit('deleteMoviePermanently', $event)"
      @refresh-metadata="emit('refreshMetadata', $event)"
      @reveal-in-file-manager="emit('revealInFileManager', $event)"
      @patch-movie-display="(body, done) => emit('patchMovieDisplay', body, done)"
      @open-poster-viewer="openPosterInViewer"
    />

    <Card class="rounded-3xl border-border/70 bg-card/85">
      <CardHeader>
        <CardTitle>{{ t("detailPage.previewGalleryTitle") }}</CardTitle>
        <CardDescription>
          {{ t("detailPage.previewHelp") }}
        </CardDescription>
      </CardHeader>

      <CardContent
        v-if="hasPreviews"
        class="flex w-full flex-wrap items-start gap-3"
      >
        <button
          v-for="(url, index) in previewImages"
          :key="`${url}-${index}`"
          :data-preview-gallery-item="String(index)"
          :data-aspect-ratio="String(previewAspectRatioFor(index))"
          type="button"
          class="relative h-40 max-w-full min-w-[8rem] shrink-0 cursor-pointer overflow-hidden rounded-[1rem] border border-border/70 bg-muted/30 text-left outline-none ring-offset-background transition-[border-color,box-shadow] hover:border-primary/50 focus-visible:ring-2 focus-visible:ring-ring sm:h-44 xl:h-48"
          :style="previewCardStyle(index)"
          @click="openPreviewViewer(index)"
        >
          <MediaStill
            :src="url"
            :alt="t('detailPage.previewAlt', { code: movie.code, n: index + 1 })"
            class="absolute inset-0 z-0"
            fit="cover"
            loading="eager"
            :fetch-priority="index < 8 ? 'high' : undefined"
            @load="updatePreviewAspectRatio(index, $event)"
          />
          <span class="sr-only">{{ t("detailPage.previewSrOpen", { n: index + 1 }) }}</span>
        </button>
      </CardContent>

      <CardContent v-else class="grid w-full gap-4 sm:grid-cols-3">
        <div
          v-for="index in 3"
          :key="index"
          class="aspect-[16/9] rounded-[1.25rem] border border-dashed border-border/70 bg-muted/20"
        />
        <p class="col-span-full text-sm text-muted-foreground">
          {{ t("detailPage.previewEmpty") }}
        </p>
      </CardContent>
    </Card>

    <PreviewImageViewer
      v-if="hasViewerImages"
      v-model:open="previewViewerOpen"
      :images="viewerImages"
      :initial-index="previewViewerStartIndex"
      :movie-code="movie.code"
    />

    <MovieCommentSection :movie-id="movie.id" :readonly="commentReadonly" />

    <div class="flex flex-col gap-4">
      <h3 class="text-xl font-semibold">{{ t("detailPage.relatedTitle") }}</h3>

      <MovieGrid
        :movies="relatedMoviesForGrid"
        :selected-movie-id="movie.id"
        @select="emit('select', $event)"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
        @toggle-favorite="emit('toggleFavorite', $event)"
      />
    </div>
  </div>
</template>
