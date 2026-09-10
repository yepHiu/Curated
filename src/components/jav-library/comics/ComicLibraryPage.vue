<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { CheckSquare, ListChecks, X } from "lucide-vue-next"
import type { ComicBook } from "@/domain/comic/types"
import type { ComicLibrarySortValue } from "@/lib/comic-sort"
import BookLibraryToolbar from "@/components/jav-library/books/BookLibraryToolbar.vue"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from "@/components/ui/empty"
import { Button } from "@/components/ui/button"
import VirtualComicGrid from "@/components/jav-library/comics/VirtualComicGrid.vue"

const props = withDefaults(
  defineProps<{
    comics: readonly ComicBook[]
    activeSort: ComicLibrarySortValue
    searchQuery?: string
    loading?: boolean
    loadError?: string
    batchMode?: boolean
    batchSelectedIds?: readonly string[]
  }>(),
  {
    searchQuery: "",
    loadError: "",
    loading: false,
    batchMode: false,
    batchSelectedIds: () => [],
  },
)

const emit = defineEmits<{
  retry: []
  updateSearch: [value: string]
  "update:sort": [value: ComicLibrarySortValue]
  openDetails: [comicId: string]
  openReader: [comicId: string, pageIndex: number]
  toggleFavorite: [payload: { comicId: string; nextValue: boolean }]
  enterBatchMode: []
  exitBatchMode: []
  selectAllVisibleInBatch: []
  toggleBatchSelect: [comicId: string]
}>()

const { t } = useI18n()
const batchModeOn = computed(() => props.batchMode === true)

</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 w-full flex-1 flex-col gap-3">
    <BookLibraryToolbar data-comic-library-toolbar kind="comics" :count="props.comics.length" :sort="activeSort" :search-query="searchQuery" @sort="emit('update:sort', $event)" @clear-search="emit('updateSearch', '')">
      <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
        <template v-if="!batchModeOn">
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="min-h-11 shrink-0 gap-1.5 rounded-full lg:min-h-8"
            data-comic-enter-batch
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
            size="sm"
            class="min-h-11 shrink-0 gap-1.5 rounded-full lg:min-h-8"
            data-comic-select-visible
            :disabled="props.comics.length === 0"
            @click="emit('selectAllVisibleInBatch')"
          >
            <CheckSquare data-icon="inline-start" aria-hidden="true" />
            {{ t("comics.batchSelectVisible") }}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="min-h-11 shrink-0 gap-1.5 rounded-full lg:min-h-8 text-muted-foreground hover:bg-muted/80 hover:text-foreground"
            data-comic-exit-batch
            @click="emit('exitBatchMode')"
          >
            <X data-icon="inline-start" aria-hidden="true" />
            {{ t("comics.batchExitToolbar") }}
          </Button>
        </template>
      </div>
    </BookLibraryToolbar>

    <Alert v-if="props.loadError" data-comic-load-error variant="destructive"><AlertDescription>{{ props.loadError }}<Button variant="outline" class="mt-3 min-h-11 w-fit rounded-full" @click="emit('retry')">{{ t('common.retry') }}</Button></AlertDescription></Alert>
    <div v-if="props.loading && !props.loadError && !props.comics.length" data-book-library-loading class="grid grid-cols-2 gap-4 sm:grid-cols-4" role="status" :aria-label="t('comics.detailLoading')"><Skeleton v-for="index in 8" :key="index" class="aspect-[2/3] rounded-2xl" /></div>

    <div
      v-else-if="props.comics.length"
      data-comic-grid-scroll
      class="min-h-0 flex-1 overflow-y-auto pr-2"
    >
      <VirtualComicGrid
        :comics="props.comics"
        :batch-mode="batchModeOn"
        :batch-selected-ids="props.batchSelectedIds"
        @open-details="emit('openDetails', $event)"
        @open-reader="(comicId, pageIndex) => emit('openReader', comicId, pageIndex)"
        @toggle-favorite="emit('toggleFavorite', $event)"
        @toggle-batch-select="emit('toggleBatchSelect', $event)"
      />
    </div>

    <Empty v-else class="min-h-72 rounded-3xl border border-dashed border-border/70 bg-muted/20">
      <EmptyHeader><EmptyTitle>{{ t(searchQuery ? 'bookBrowser.noResults' : 'comics.emptyTitle') }}</EmptyTitle><EmptyDescription>{{ t(searchQuery ? 'bookBrowser.noResultsHint' : 'comics.emptyDesc') }}</EmptyDescription></EmptyHeader>
      <Button v-if="searchQuery" variant="outline" class="min-h-11 rounded-full" @click="emit('updateSearch', '')">{{ t('comics.clearSearch') }}</Button>
    </Empty>
  </div>
</template>
