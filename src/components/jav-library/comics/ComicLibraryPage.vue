<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { CheckSquare, ListChecks, X } from "lucide-vue-next"
import type { ComicBook } from "@/domain/comic/types"
import type { ComicLibrarySortValue } from "@/lib/comic-sort"
import { Button } from "@/components/ui/button"
import VirtualComicGrid from "@/components/jav-library/comics/VirtualComicGrid.vue"

const props = withDefaults(
  defineProps<{
    comics: readonly ComicBook[]
    activeSort: ComicLibrarySortValue
    searchQuery?: string
    loadError?: string
    batchMode?: boolean
    batchSelectedIds?: readonly string[]
  }>(),
  {
    searchQuery: "",
    loadError: "",
    batchMode: false,
    batchSelectedIds: () => [],
  },
)

const emit = defineEmits<{
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

const sortOptions: { value: ComicLibrarySortValue; labelKey: string }[] = [
  { value: "addedAt", labelKey: "comics.sortByAdded" },
  { value: "fileName", labelKey: "comics.sortByFileName" },
  { value: "favorite", labelKey: "comics.sortByFavorite" },
]

function sortOptionState(value: ComicLibrarySortValue): "active" | "inactive" {
  return props.activeSort === value ? "active" : "inactive"
}

function sortOptionClasses(value: ComicLibrarySortValue): string {
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
      data-comic-library-toolbar
      class="flex flex-wrap items-center justify-between gap-3 pb-1"
    >
      <div
        data-comic-sort-tabs
        class="inline-flex h-auto w-fit max-w-full flex-wrap items-center justify-center rounded-2xl bg-muted/60 p-1"
        role="tablist"
        aria-label="Comic sort"
      >
        <button
          v-for="option in sortOptions"
          :key="option.value"
          type="button"
          role="tab"
          :aria-selected="props.activeSort === option.value"
          :data-state="sortOptionState(option.value)"
          :data-comic-sort-option="option.value"
          :class="sortOptionClasses(option.value)"
          @click="emit('update:sort', option.value)"
        >
          {{ t(option.labelKey) }}
        </button>
      </div>

      <div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
        <template v-if="!batchModeOn">
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="shrink-0 gap-1.5 rounded-xl"
            data-comic-enter-batch
            @click="emit('enterBatchMode')"
          >
            <ListChecks class="size-4 opacity-80" aria-hidden="true" />
            {{ t("comics.batchManage") }}
          </Button>
        </template>
        <template v-else>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="shrink-0 gap-1.5 rounded-xl"
            data-comic-select-visible
            :disabled="props.comics.length === 0"
            @click="emit('selectAllVisibleInBatch')"
          >
            <CheckSquare class="size-4 opacity-80" aria-hidden="true" />
            {{ t("comics.batchSelectVisible") }}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="shrink-0 gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground"
            data-comic-exit-batch
            @click="emit('exitBatchMode')"
          >
            <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
            {{ t("comics.batchExitToolbar") }}
          </Button>
        </template>
      </div>
    </header>

    <p
      v-if="props.loadError"
      data-comic-load-error
      role="alert"
      class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ props.loadError }}
    </p>

    <div
      v-if="props.comics.length"
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

    <div
      v-else
      class="flex min-h-72 flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/70 bg-muted/20 p-8 text-center"
    >
      <p class="text-base font-medium">{{ t("comics.emptyTitle") }}</p>
      <p class="max-w-md text-sm text-muted-foreground">{{ t("comics.emptyDesc") }}</p>
      <Button
        v-if="props.searchQuery"
        type="button"
        variant="outline"
        class="rounded-xl"
        @click="emit('updateSearch', '')"
      >
        {{ t("comics.clearSearch") }}
      </Button>
    </div>
  </div>
</template>
