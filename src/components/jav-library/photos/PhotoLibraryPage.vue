<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { PhotoBook } from "@/domain/photo/types"
import type { PhotoLibrarySortValue } from "@/lib/photo-sort"
import BookLibraryToolbar from "@/components/jav-library/books/BookLibraryToolbar.vue"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from "@/components/ui/empty"
import { Button } from "@/components/ui/button"
import VirtualPhotoGrid from "@/components/jav-library/photos/VirtualPhotoGrid.vue"

const props = withDefaults(
  defineProps<{
    photos: readonly PhotoBook[]
    activeSort: PhotoLibrarySortValue
    searchQuery?: string
    loading?: boolean
    loadError?: string
  }>(),
  {
    searchQuery: "",
    loadError: "",
    loading: false,
  },
)

const emit = defineEmits<{
  retry: []
  updateSearch: [value: string]
  "update:sort": [value: PhotoLibrarySortValue]
  openDetails: [photoId: string]
  openViewer: [photoId: string, pageIndex: number]
}>()

const { t } = useI18n()

</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 w-full flex-1 flex-col gap-3">
    <BookLibraryToolbar data-photo-library-toolbar kind="photos" :count="props.photos.length" :sort="activeSort" :search-query="searchQuery" @sort="emit('update:sort', $event)" @clear-search="emit('updateSearch', '')">
    </BookLibraryToolbar>

    <Alert v-if="props.loadError" data-photo-load-error variant="destructive"><AlertDescription>{{ props.loadError }}<Button variant="outline" class="mt-3 min-h-11 w-fit rounded-full" @click="emit('retry')">{{ t('common.retry') }}</Button></AlertDescription></Alert>
    <div v-if="props.loading && !props.loadError && !props.photos.length" data-book-library-loading class="grid grid-cols-2 gap-4 sm:grid-cols-4" role="status" :aria-label="t('photos.detailLoading')"><Skeleton v-for="index in 8" :key="index" class="aspect-[2/3] rounded-2xl" /></div>

    <div
      v-else-if="props.photos.length"
      data-photo-grid-scroll
      class="min-h-0 flex-1 overflow-y-auto pr-2"
    >
      <VirtualPhotoGrid
        :photos="props.photos"
        @open-details="emit('openDetails', $event)"
        @open-viewer="(photoId, pageIndex) => emit('openViewer', photoId, pageIndex)"
      />
    </div>

    <Empty v-else class="min-h-72 rounded-3xl border border-dashed border-border/70 bg-muted/20">
      <EmptyHeader><EmptyTitle>{{ t(searchQuery ? 'bookBrowser.noResults' : 'photos.emptyTitle') }}</EmptyTitle><EmptyDescription>{{ t(searchQuery ? 'bookBrowser.noResultsHint' : 'photos.emptyDesc') }}</EmptyDescription></EmptyHeader>
      <Button v-if="searchQuery" variant="outline" class="min-h-11 rounded-full" @click="emit('updateSearch', '')">{{ t('photos.clearSearch') }}</Button>
    </Empty>
  </div>
</template>
