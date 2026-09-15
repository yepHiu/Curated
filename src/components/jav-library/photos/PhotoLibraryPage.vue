<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { PhotoBook } from "@/domain/photo/types"
import type { BookLibrarySortValue } from "@/lib/book-library-query"
import BookLibraryToolbar from "@/components/jav-library/books/BookLibraryToolbar.vue"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import VirtualPhotoGrid from "@/components/jav-library/photos/VirtualPhotoGrid.vue"

const props = withDefaults(
  defineProps<{
    photos: readonly PhotoBook[]
    activeSort: BookLibrarySortValue
    searchQuery?: string
    tag?: string
    hasConstraints?: boolean
    loading?: boolean
    loadError?: string
  }>(),
  {
    searchQuery: "",
    tag: "",
    hasConstraints: false,
    loadError: "",
    loading: false,
  },
)

const emit = defineEmits<{
  retry: []
  updateSearch: [value: string]
  clearTag: []
  clearFilters: []
  "update:sort": [value: BookLibrarySortValue]
  openDetails: [photoId: string]
  openViewer: [photoId: string, pageIndex: number]
}>()

const { t } = useI18n()
const emptyTitle = computed(() =>
  props.hasConstraints ? "bookBrowser.noResults" : "photos.emptyTitle",
)
const emptyHint = computed(() =>
  props.hasConstraints ? "bookBrowser.noResultsHint" : "photos.emptyDesc",
)

/** 把网格的查看入口转发给写真库页。 */
function openViewer(photoId: string, pageIndex: number) {
  emit("openViewer", photoId, pageIndex)
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 w-full flex-1 flex-col gap-3">
    <BookLibraryToolbar
      data-photo-library-toolbar
      kind="photos"
      :count="props.photos.length"
      :sort="activeSort"
      :search-query="searchQuery"
      :tag="tag"
      @sort="emit('update:sort', $event)"
      @clear-search="emit('updateSearch', '')"
      @clear-tag="emit('clearTag')"
    />

    <Alert v-if="props.loadError" data-photo-load-error variant="destructive"><AlertDescription>{{ props.loadError }}<Button variant="outline" class="mt-3 min-h-11 w-fit rounded-full sm:min-h-8" @click="emit('retry')">{{ t('common.retry') }}</Button></AlertDescription></Alert>
    <div v-if="props.loading && !props.loadError && !props.photos.length" data-book-library-loading class="grid w-full overflow-x-hidden" :style="{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(100%, var(--movie-grid-min-track)), 1fr))', columnGap: 'var(--movie-grid-gap)', rowGap: 'var(--movie-grid-gap)' }" role="status" :aria-label="t('photos.detailLoading')"><Skeleton v-for="index in 8" :key="index" class="aspect-[358/537] rounded-[1.2rem]" /></div>

    <div
      v-else-if="props.photos.length"
      data-photo-grid-scroll
      class="min-h-0 flex-1"
    >
      <VirtualPhotoGrid
        :photos="props.photos"
        @open-details="emit('openDetails', $event)"
        @open-viewer="openViewer"
      />
    </div>

    <Card v-else class="rounded-3xl border-border/70 bg-card/80">
      <CardHeader>
        <CardTitle>{{ t(emptyTitle) }}</CardTitle>
        <CardDescription>{{ t(emptyHint) }}</CardDescription>
      </CardHeader>
      <CardContent v-if="hasConstraints">
        <Button variant="outline" class="min-h-11 rounded-full sm:min-h-8" @click="emit('clearFilters')">{{ t('bookBrowser.clearFilters') }}</Button>
      </CardContent>
    </Card>
  </div>
</template>
