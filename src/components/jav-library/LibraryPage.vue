<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { CheckSquare, ListChecks, X } from "lucide-vue-next"
import type { LibraryMode } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { Button } from "@/components/ui/button"
import ActorProfileCard from "@/components/jav-library/ActorProfileCard.vue"
import LibrarySavedViewsControls from "@/components/jav-library/LibrarySavedViewsControls.vue"
import VirtualMovieMasonry from "@/components/jav-library/VirtualMovieMasonry.vue"

const props = defineProps<{
  mode: LibraryMode
  visibleMovies: readonly Movie[]
  batchMode?: boolean
  /** 多选 id 列表（来自父级 Set 快照，用于卡片选中态） */
  batchSelectedIds?: readonly string[]
  /** 当前 URL 精确演员筛选（`actor=`） */
  activeActorFilter?: string
  /** 当前 URL 精确厂商筛选（`studio=`） */
  activeStudioFilter?: string
  /** 演员资料卡「用户标签」联想候选（影片 userTags 等，与演员库卡同源） */
  actorUserTagSuggestions?: readonly string[]
  scrollPreserveKey?: string
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId?: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
  contextMenu: [payload: { event: MouseEvent; movie: Movie }]
  clearExactActorFilter: []
  clearExactStudioFilter: []
  enterBatchMode: []
  exitBatchMode: []
  selectAllVisibleInBatch: []
  toggleBatchSelect: [payload: { movieId: string; shiftKey: boolean }]
}>()

const { t } = useI18n()
const activeActorTrimmed = computed(() => props.activeActorFilter?.trim() ?? "")
const activeStudioTrimmed = computed(() => props.activeStudioFilter?.trim() ?? "")

const batchModeOn = computed(() => props.batchMode === true)
const pageTitleKey = computed(() => {
  switch (props.mode) {
    case "favorites":
      return "nav.favorites"
    case "trash":
      return "nav.trash"
    default:
      return "nav.library"
  }
})
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 w-full flex-1 flex-col gap-3">
    <h1 class="sr-only">{{ t(pageTitleKey) }}</h1>
    <div
      v-if="activeStudioTrimmed"
      class="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border/70 bg-card/85 px-4 py-3 shadow-sm shadow-black/5"
    >
      <p class="min-w-0 text-sm text-muted-foreground">
        {{ t("library.filterActive") }}<span class="font-medium text-foreground">{{ activeStudioTrimmed }}</span>
      </p>
      <Button
        type="button"
        variant="outline"
        size="sm"
        class="min-h-11 shrink-0 rounded-xl sm:min-h-8"
        @click="emit('clearExactStudioFilter')"
      >
        {{ t("library.clearFilter") }}
      </Button>
    </div>
    <div
      v-if="props.mode === 'trash'"
      class="flex flex-wrap items-center justify-end gap-1.5"
    >
      <template v-if="!batchModeOn">
        <Button
          type="button"
          variant="outline"
          class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
          @click="emit('enterBatchMode')"
        >
          <ListChecks data-icon="inline-start" aria-hidden="true" />
          {{ t("library.batchManage") }}
        </Button>
      </template>
      <template v-else>
        <Button
          type="button"
          variant="outline"
          class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
          @click="emit('selectAllVisibleInBatch')"
        >
          <CheckSquare data-icon="inline-start" aria-hidden="true" />
          {{ t("library.batchSelectVisible") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
          @click="emit('exitBatchMode')"
        >
          <X data-icon="inline-start" aria-hidden="true" />
          {{ t("library.batchExitToolbar") }}
        </Button>
      </template>
    </div>

    <div
      v-else
      class="flex min-w-0 w-full items-center justify-end"
    >
      <template v-if="!batchModeOn">
        <LibrarySavedViewsControls>
          <Button
            type="button"
            variant="outline"
            data-library-batch-toggle
            class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
            @click="emit('enterBatchMode')"
          >
            <ListChecks data-icon="inline-start" aria-hidden="true" />
            {{ t("library.batchManage") }}
          </Button>
        </LibrarySavedViewsControls>
      </template>
      <template v-else>
        <div class="flex w-full min-w-0 flex-nowrap items-center justify-end gap-1.5">
          <Button
            type="button"
            variant="outline"
            class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
            @click="emit('selectAllVisibleInBatch')"
          >
            <CheckSquare data-icon="inline-start" aria-hidden="true" />
            {{ t("library.batchSelectVisible") }}
          </Button>
          <Button
            type="button"
            variant="ghost"
            class="min-h-11 shrink-0 rounded-full px-3 sm:min-h-8"
            @click="emit('exitBatchMode')"
          >
            <X data-icon="inline-start" aria-hidden="true" />
            {{ t("library.batchExitToolbar") }}
          </Button>
        </div>
      </template>
    </div>

    <div class="min-h-0 flex-1">
      <VirtualMovieMasonry
        :movies="props.visibleMovies"
        :batch-mode="batchModeOn"
        :batch-selected-ids="props.batchSelectedIds ?? []"
        :scroll-preserve-key="props.scrollPreserveKey"
        :empty-title="props.mode === 'trash' ? t('library.trashEmptyTitle') : undefined"
        :empty-description="props.mode === 'trash' ? t('library.trashEmptyDesc') : undefined"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
        @toggle-favorite="emit('toggleFavorite', $event)"
        @context-menu="emit('contextMenu', $event)"
        @toggle-batch-select="emit('toggleBatchSelect', $event)"
      >
        <template v-if="activeActorTrimmed" #header>
          <ActorProfileCard
            :actor-name="activeActorTrimmed"
            :user-tag-suggestions="props.actorUserTagSuggestions ?? []"
            @clear-filter="emit('clearExactActorFilter')"
          />
        </template>
      </VirtualMovieMasonry>
    </div>
  </div>
</template>
