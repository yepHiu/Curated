<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { CheckSquare, ListChecks, X } from "lucide-vue-next"
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
    batchMode?: boolean
    batchSelectedIds?: readonly string[]
  }>(),
  {
    searchQuery: "",
    tag: "",
    hasConstraints: false,
    loadError: "",
    loading: false,
    batchMode: false,
    batchSelectedIds: () => [],
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
  enterBatchMode: []
  exitBatchMode: []
  selectAllVisibleInBatch: []
  toggleBatchSelect: [photoId: string]
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
      :batch-mode="props.batchMode"
      :batch-selected-count="props.batchSelectedIds.length"
      @sort="emit('update:sort', $event)"
      @clear-search="emit('updateSearch', '')"
      @clear-tag="emit('clearTag')"
    >
      <Button v-if="!props.batchMode" type="button" variant="outline" data-photo-enter-batch class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8" @click="emit('enterBatchMode')">
        <ListChecks data-icon="inline-start" aria-hidden="true" />{{ t('photos.batchManage') }}
      </Button>
      <template v-else>
        <Button type="button" variant="outline" data-photo-select-visible class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8" :disabled="props.photos.length === 0" @click="emit('selectAllVisibleInBatch')">
          <CheckSquare data-icon="inline-start" aria-hidden="true" />{{ t('photos.batchSelectVisible') }}
        </Button>
        <Button type="button" variant="ghost" data-photo-exit-batch class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8" @click="emit('exitBatchMode')">
          <X data-icon="inline-start" aria-hidden="true" />{{ t('photos.batchExitToolbar') }}
        </Button>
      </template>
    </BookLibraryToolbar>

    <Alert v-if="props.loadError" data-photo-load-error variant="destructive"><AlertDescription>{{ props.loadError }}<Button variant="outline" class="mt-3 min-h-11 w-fit rounded-full sm:min-h-8" @click="emit('retry')">{{ t('common.retry') }}</Button></AlertDescription></Alert>
    <div v-if="props.loading && !props.loadError && !props.photos.length" data-book-library-loading class="grid w-full overflow-x-hidden" :style="{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(100%, var(--movie-grid-min-track)), 1fr))', columnGap: 'var(--movie-grid-gap)', rowGap: 'var(--movie-grid-gap)' }" role="status" :aria-label="t('photos.detailLoading')"><Skeleton v-for="index in 8" :key="index" class="aspect-[358/537] rounded-[1.2rem]" /></div>

    <div
      v-else-if="props.photos.length"
      data-photo-grid-scroll
      class="min-h-0 flex-1"
    >
      <VirtualPhotoGrid
        :photos="props.photos"
        :batch-mode="props.batchMode"
        :batch-selected-ids="props.batchSelectedIds"
        @open-details="emit('openDetails', $event)"
        @open-viewer="openViewer"
        @toggle-batch-select="emit('toggleBatchSelect', $event)"
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
