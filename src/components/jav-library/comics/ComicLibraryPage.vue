<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { CheckSquare, ListChecks, X } from "lucide-vue-next"
import type { ComicBook, ComicReadStatus } from "@/domain/comic/types"
import type { BookLibrarySortValue } from "@/lib/book-library-query"
import BookLibraryToolbar from "@/components/jav-library/books/BookLibraryToolbar.vue"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import MediaEmptyState from "@/components/jav-library/MediaEmptyState.vue"
import VirtualComicGrid from "@/components/jav-library/comics/VirtualComicGrid.vue"

const props = withDefaults(
  defineProps<{
    comics: readonly ComicBook[]
    activeSort: BookLibrarySortValue
    searchQuery?: string
    tag?: string
    favorite?: boolean
    readStatus?: ComicReadStatus | "all"
    hasConstraints?: boolean
    loading?: boolean
    loadError?: string
    batchMode?: boolean
    batchSelectedIds?: readonly string[]
  }>(),
  {
    searchQuery: "",
    tag: "",
    favorite: false,
    readStatus: "all",
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
  updateFavorite: [value: boolean]
  updateReadStatus: [value: ComicReadStatus | "all"]
  clearFilters: []
  "update:sort": [value: BookLibrarySortValue]
  openDetails: [comicId: string]
  openReader: [comicId: string, pageIndex: number]
  toggleFavorite: [payload: { comicId: string; nextValue: boolean }]
  enterBatchMode: []
  exitBatchMode: []
  selectAllVisibleInBatch: []
  toggleBatchSelect: [comicId: string]
}>()

const { t } = useI18n()
/** 批量模式由父级持有；页面只负责把开关投影到工具栏与网格。 */
const batchModeOn = computed(() => props.batchMode === true)

/** 把网格的阅读入口转发给资料库页。 */
function openReader(comicId: string, pageIndex: number) {
  emit("openReader", comicId, pageIndex)
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 w-full flex-1 flex-col gap-3">
    <BookLibraryToolbar
      data-comic-library-toolbar
      kind="comics"
      :count="props.comics.length"
      :sort="activeSort"
      :search-query="searchQuery"
      :tag="tag"
      :favorite="favorite"
      :read-status="readStatus"
      :batch-mode="batchModeOn"
      :batch-selected-count="props.batchSelectedIds.length"
      @sort="emit('update:sort', $event)"
      @clear-search="emit('updateSearch', '')"
      @clear-tag="emit('clearTag')"
      @update-favorite="emit('updateFavorite', $event)"
      @update-read-status="emit('updateReadStatus', $event)"
    >
      <template v-if="!batchModeOn">
        <Button
          type="button"
          variant="outline"
          data-comic-enter-batch
          class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
          @click="emit('enterBatchMode')"
        >
          <ListChecks data-icon="inline-start" aria-hidden="true" />
          {{ t("comics.batchManage") }}
        </Button>
      </template>
      <template v-else>
        <Button
          type="button"
          variant="outline"
          data-comic-select-visible
          class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
          :disabled="props.comics.length === 0"
          @click="emit('selectAllVisibleInBatch')"
        >
          <CheckSquare data-icon="inline-start" aria-hidden="true" />
          {{ t("comics.batchSelectVisible") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          data-comic-exit-batch
          class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
          @click="emit('exitBatchMode')"
        >
          <X data-icon="inline-start" aria-hidden="true" />
          {{ t("comics.batchExitToolbar") }}
        </Button>
      </template>
    </BookLibraryToolbar>

    <Alert v-if="props.loadError" data-comic-load-error variant="destructive"><AlertDescription>{{ props.loadError }}<Button variant="outline" class="mt-3 min-h-11 w-fit rounded-full sm:min-h-8" @click="emit('retry')">{{ t('common.retry') }}</Button></AlertDescription></Alert>
    <div v-if="props.loading && !props.loadError && !props.comics.length" data-book-library-loading class="grid w-full overflow-x-hidden" :style="{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(100%, var(--movie-grid-min-track)), 1fr))', columnGap: 'var(--movie-grid-gap)', rowGap: 'var(--movie-grid-gap)' }" role="status" :aria-label="t('comics.detailLoading')"><Skeleton v-for="index in 8" :key="index" class="aspect-[358/537] rounded-[1.2rem]" /></div>

    <div
      v-else-if="props.comics.length"
      data-comic-grid-scroll
      class="min-h-0 flex-1"
    >
      <VirtualComicGrid
        :comics="props.comics"
        :batch-mode="batchModeOn"
        :batch-selected-ids="props.batchSelectedIds"
        @open-details="emit('openDetails', $event)"
        @open-reader="openReader"
        @toggle-favorite="emit('toggleFavorite', $event)"
        @toggle-batch-select="emit('toggleBatchSelect', $event)"
      />
    </div>

    <MediaEmptyState v-else-if="!props.loadError" :filtered="hasConstraints" :description="t('comics.emptyDesc')">
      <template v-if="hasConstraints" #default>
        <Button variant="outline" class="min-h-11 rounded-full sm:min-h-8" @click="emit('clearFilters')">{{ t('bookBrowser.clearFilters') }}</Button>
      </template>
    </MediaEmptyState>
  </div>
</template>
