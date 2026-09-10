<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { PhotoBook } from "@/domain/photo/types"
import type { PhotoLibrarySortValue } from "@/lib/photo-sort"
import { Button } from "@/components/ui/button"
import VirtualPhotoGrid from "@/components/jav-library/photos/VirtualPhotoGrid.vue"

const props = withDefaults(
  defineProps<{
    photos: readonly PhotoBook[]
    activeSort: PhotoLibrarySortValue
    searchQuery?: string
    loadError?: string
  }>(),
  {
    searchQuery: "",
    loadError: "",
  },
)

const emit = defineEmits<{
  updateSearch: [value: string]
  "update:sort": [value: PhotoLibrarySortValue]
  openDetails: [photoId: string]
  openViewer: [photoId: string, pageIndex: number]
}>()

const { t } = useI18n()

const sortOptions: { value: PhotoLibrarySortValue; labelKey: string }[] = [
  { value: "addedAt", labelKey: "photos.sortByAdded" },
  { value: "fileName", labelKey: "photos.sortByFileName" },
  { value: "favorite", labelKey: "photos.sortByFavorite" },
]

function sortOptionState(value: PhotoLibrarySortValue): "active" | "inactive" {
  return props.activeSort === value ? "active" : "inactive"
}

function sortOptionClasses(value: PhotoLibrarySortValue): string {
  const base =
    "inline-flex h-9 items-center justify-center rounded-xl border border-transparent px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
  if (props.activeSort === value) {
    return `${base} bg-background text-foreground shadow-sm`
  }
  return `${base} text-muted-foreground hover:text-foreground`
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 w-full flex-1 flex-col gap-5 lg:gap-6">
    <header
      data-photo-library-toolbar
      class="flex flex-wrap items-center justify-between gap-3 pb-1"
    >
      <div
        data-photo-sort-tabs
        class="inline-flex h-auto w-fit max-w-full flex-wrap items-center justify-center rounded-2xl bg-muted/60 p-1"
        role="tablist"
        aria-label="Photo sort"
      >
        <button
          v-for="option in sortOptions"
          :key="option.value"
          type="button"
          role="tab"
          :aria-selected="props.activeSort === option.value"
          :data-state="sortOptionState(option.value)"
          :data-photo-sort-option="option.value"
          :class="sortOptionClasses(option.value)"
          @click="emit('update:sort', option.value)"
        >
          {{ t(option.labelKey) }}
        </button>
      </div>
    </header>

    <p
      v-if="props.loadError"
      data-photo-load-error
      role="alert"
      class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ props.loadError }}
    </p>

    <div
      v-if="props.photos.length"
      data-photo-grid-scroll
      class="min-h-0 flex-1 overflow-y-auto pr-2"
    >
      <VirtualPhotoGrid
        :photos="props.photos"
        @open-details="emit('openDetails', $event)"
        @open-viewer="(photoId, pageIndex) => emit('openViewer', photoId, pageIndex)"
      />
    </div>

    <div
      v-else
      class="flex min-h-72 flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/70 bg-muted/20 p-8 text-center"
    >
      <p class="text-base font-medium">{{ t("photos.emptyTitle") }}</p>
      <p class="max-w-md text-sm text-muted-foreground">{{ t("photos.emptyDesc") }}</p>
      <Button
        v-if="props.searchQuery"
        type="button"
        variant="outline"
        class="rounded-xl"
        @click="emit('updateSearch', '')"
      >
        {{ t("photos.clearSearch") }}
      </Button>
    </div>
  </div>
</template>
