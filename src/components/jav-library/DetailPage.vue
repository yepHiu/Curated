<script setup lang="ts">
import { computed, ref } from "vue"
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

const previewImages = computed(() => props.movie.previewImages?.slice(0, 18) ?? [])
const hasPreviews = computed(() => previewImages.value.length > 0)

const previewViewerOpen = ref(false)
const previewViewerStartIndex = ref(0)

function openPreviewViewer(index: number) {
  previewViewerStartIndex.value = index
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
  updateMetadataTags: [payload: { movieId: string; tags: string[] }]
  deleteMovie: [movieId: string]
  refreshMetadata: [movieId: string]
}>()
</script>

<template>
  <div class="flex min-w-0 w-full flex-col gap-6">
    <DetailPanel
      :movie="movie"
      :user-tag-suggestions="props.userTagSuggestions"
      :metadata-refresh-busy="props.metadataRefreshBusy"
      @open-player="emit('openPlayer', $event)"
      @update-user-rating="emit('updateUserRating', $event)"
      @update-user-tags="emit('updateUserTags', $event)"
      @browse-by-tag="emit('browseByTag', $event)"
      @browse-by-actor="emit('browseByActor', $event)"
      @update-metadata-tags="emit('updateMetadataTags', $event)"
      @delete-movie="emit('deleteMovie', $event)"
      @refresh-metadata="emit('refreshMetadata', $event)"
    />

    <Card class="rounded-3xl border-border/70 bg-card/85">
      <CardHeader>
        <CardTitle>Preview gallery</CardTitle>
        <CardDescription>
          来自元数据刮削的样本图；点击缩略图打开查看器浏览大图，支持左右键切换。若源站限制外链，图片可能无法显示（可后续接后端代理）。
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
            :alt="`${movie.code} 样本图 ${index + 1}`"
            class="absolute inset-0 z-0"
          />
          <span class="sr-only">在查看器中打开第 {{ index + 1 }} 张预览图</span>
        </button>
      </CardContent>
      <CardContent v-else class="grid w-full gap-4 sm:grid-cols-3">
        <div
          v-for="index in 3"
          :key="index"
          class="aspect-[16/9] rounded-[1.25rem] border border-dashed border-border/70 bg-muted/20"
        />
        <p class="col-span-full text-sm text-muted-foreground">
          当前条目没有样本图 URL（例如仅刮削到文本、或提供商未返回预览图）。
        </p>
      </CardContent>
    </Card>

    <PreviewImageViewer
      v-if="hasPreviews"
      v-model:open="previewViewerOpen"
      :images="previewImages"
      :initial-index="previewViewerStartIndex"
      :movie-code="movie.code"
    />

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
