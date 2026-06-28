<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { ComicBook, ComicReadStatus } from "@/domain/comic/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import VirtualComicGrid from "@/components/jav-library/comics/VirtualComicGrid.vue"

export type ComicLibraryFilterValue = "all" | "favorite" | ComicReadStatus

const props = withDefaults(
  defineProps<{
    comics: readonly ComicBook[]
    activeFilter: ComicLibraryFilterValue
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
  updateActiveFilter: [value: ComicLibraryFilterValue]
  openDetails: [comicId: string]
  openReader: [comicId: string, pageIndex: number]
  toggleFavorite: [payload: { comicId: string; nextValue: boolean }]
}>()

const { t } = useI18n()

function onFilterChange(value: string | number) {
  emit("updateActiveFilter", String(value) as ComicLibraryFilterValue)
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 flex-col gap-5">
    <header class="flex flex-col gap-4">
      <div class="flex flex-wrap items-end justify-end gap-3">
        <Input
          class="h-10 w-full max-w-sm rounded-2xl"
          :model-value="props.searchQuery"
          :placeholder="t('comics.searchPlaceholder')"
          @update:model-value="emit('updateSearch', String($event))"
        />
      </div>

      <Tabs
        :model-value="props.activeFilter"
        class="min-w-0"
        @update:model-value="onFilterChange"
      >
        <TabsList class="h-auto w-fit max-w-full flex-wrap rounded-2xl bg-muted/60 p-1">
          <TabsTrigger value="all" class="rounded-xl px-4 py-2">
            {{ t("comics.filterAll") }}
          </TabsTrigger>
          <TabsTrigger value="favorite" class="rounded-xl px-4 py-2">
            {{ t("comics.filterFavorite") }}
          </TabsTrigger>
          <TabsTrigger value="unread" class="rounded-xl px-4 py-2">
            {{ t("comics.filterUnread") }}
          </TabsTrigger>
          <TabsTrigger value="reading" class="rounded-xl px-4 py-2">
            {{ t("comics.filterReading") }}
          </TabsTrigger>
          <TabsTrigger value="read" class="rounded-xl px-4 py-2">
            {{ t("comics.filterRead") }}
          </TabsTrigger>
        </TabsList>
      </Tabs>
    </header>

    <p
      v-if="props.loadError"
      data-comic-load-error
      role="alert"
      class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {{ props.loadError }}
    </p>

    <div v-if="props.comics.length" class="min-h-0 flex-1 overflow-y-auto pr-1">
      <VirtualComicGrid
        :comics="props.comics"
        @open-details="emit('openDetails', $event)"
        @open-reader="(comicId, pageIndex) => emit('openReader', comicId, pageIndex)"
        @toggle-favorite="emit('toggleFavorite', $event)"
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
