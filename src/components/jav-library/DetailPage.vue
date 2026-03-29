<script setup lang="ts">
import { computed, ref } from "vue"
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

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    movie: Movie
    relatedMovies: Movie[]
    /** 添加用户标签时的模糊联想候选（全库 userTags + 当前片元数据标签等） */
    userTagSuggestions?: readonly string[]
    metadataRefreshBusy?: boolean
  }>(),
  { metadataRefreshBusy: false, userTagSuggestions: () => [] },
)

const commentReadonly = computed(() => Boolean(props.movie.trashedAt?.trim()))

const previewImages = computed(() => props.movie.previewImages?.slice(0, 18) ?? [])
const hasPreviews = computed(() => previewImages.value.length > 0)

/** 查看器内顺序：海报（封面优先）→ 样本图，去重 */
const viewerImages = computed(() => {
  const poster = (props.movie.coverUrl || props.movie.thumbUrl || "").trim()
  const out: string[] = []
  if (poster) out.push(poster)
  for (const u of previewImages.value) {
    const s = (u || "").trim()
    if (!s || s === poster) continue
    out.push(s)
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
    const s = previews[i]?.trim()
    if (!s || (poster && s === poster)) continue
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
        class="grid w-full gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      >
        <button
          v-for="(url, index) in previewImages"
          :key="`${url}-${index}`"
          type="button"
          class="relative aspect-[16/9] cursor-pointer overflow-hidden rounded-[1.25rem] border border-border/70 bg-muted/30 text-left outline-none ring-offset-background hover:border-primary/50 focus-visible:ring-2 focus-visible:ring-ring"
          @click="openPreviewViewer(index)"
        >
          <MediaStill
            :src="url"
            :alt="t('detailPage.previewAlt', { code: movie.code, n: index + 1 })"
            class="absolute inset-0 z-0"
            loading="eager"
            :fetch-priority="index < 8 ? 'high' : undefined"
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
