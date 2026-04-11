<script setup lang="ts">
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute } from "vue-router"
import { CheckSquare, ChevronDown, ListChecks, X } from "lucide-vue-next"
import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import ActorProfileCard from "@/components/jav-library/ActorProfileCard.vue"
import VirtualMovieMasonry from "@/components/jav-library/VirtualMovieMasonry.vue"
import {
  aggregateMetadataTagCounts,
  aggregateUserTagCounts,
} from "@/lib/library-stats"
import { getLibraryTagExactQuery } from "@/lib/library-query"

const props = defineProps<{
  mode: LibraryMode
  allMovies: readonly Movie[]
  visibleMovies: readonly Movie[]
  selectedMovie?: Movie
  activeTab: LibraryTab
  batchMode?: boolean
  /** 多选 id 列表（来自父级 Set 快照，用于卡片勾选态） */
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
  "update:activeTab": [value: LibraryTab]
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId?: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
  contextMenu: [payload: { event: MouseEvent; movie: Movie }]
  /** 与详情页点标签一致：写入 `tag`、清除 `q`/`actor` */
  browseByExactTag: [tag: string]
  clearExactTagFilter: []
  clearExactActorFilter: []
  clearExactStudioFilter: []
  enterBatchMode: []
  exitBatchMode: []
  selectAllVisibleInBatch: []
  toggleBatchSelect: [movieId: string]
}>()

const { t, locale } = useI18n()
const route = useRoute()
/** 折叠时各区块默认展示的标签个数（其余用「展开」） */
const TAG_PREVIEW_COUNT = 14

const metadataTagRanked = computed(() =>
  aggregateMetadataTagCounts(props.allMovies, locale.value),
)

const userTagRanked = computed(() => aggregateUserTagCounts(props.allMovies, locale.value))

const metaTagsExpanded = ref(false)
const userTagsExpanded = ref(false)

const visibleMetaTags = computed(() => {
  const all = metadataTagRanked.value
  if (metaTagsExpanded.value || all.length <= TAG_PREVIEW_COUNT) return all
  return all.slice(0, TAG_PREVIEW_COUNT)
})

const visibleUserTags = computed(() => {
  const all = userTagRanked.value
  if (userTagsExpanded.value || all.length <= TAG_PREVIEW_COUNT) return all
  return all.slice(0, TAG_PREVIEW_COUNT)
})

const metaTagsHiddenCount = computed(() =>
  Math.max(0, metadataTagRanked.value.length - TAG_PREVIEW_COUNT),
)

const userTagsHiddenCount = computed(() =>
  Math.max(0, userTagRanked.value.length - TAG_PREVIEW_COUNT),
)

/** 与列表筛选同源：直接读 route.query，避免首击 replace 后父级 prop 与 URL 短暂不同步导致芯片不高亮 */
const activeTagTrimmed = computed(() => {
  if (props.mode === "trash") {
    return ""
  }
  return getLibraryTagExactQuery(route.query).trim()
})
const activeActorTrimmed = computed(() => props.activeActorFilter?.trim() ?? "")
const activeStudioTrimmed = computed(() => props.activeStudioFilter?.trim() ?? "")

const batchModeOn = computed(() => props.batchMode === true)

const handleTabChange = (value: string | number) => {
  emit("update:activeTab", String(value) as LibraryTab)
}

function onTagChipClick(tag: string) {
  const t = tag.trim()
  if (!t) return
  emit("browseByExactTag", t)
}

function isChipActive(tag: string): boolean {
  return activeTagTrimmed.value !== "" && activeTagTrimmed.value === tag
}
</script>

<template>
  <div class="flex h-full min-h-0 min-w-0 w-full flex-1 flex-col gap-5 lg:gap-6">
    <ActorProfileCard
      v-if="activeActorTrimmed"
      :actor-name="activeActorTrimmed"
      :user-tag-suggestions="props.actorUserTagSuggestions ?? []"
      @clear-filter="emit('clearExactActorFilter')"
    />
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
        class="shrink-0 rounded-xl"
        @click="emit('clearExactStudioFilter')"
      >
        {{ t("library.clearFilter") }}
      </Button>
    </div>
    <Card
      v-if="props.mode === 'tags'"
      class="gap-3 py-5 rounded-3xl border-border/70 bg-card/85 shadow-lg shadow-black/5"
    >
      <CardHeader class="gap-3">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 flex-1">
            <CardTitle>{{ t("library.tagBrowseTitle") }}</CardTitle>
          </div>
          <Button
            v-if="activeTagTrimmed"
            type="button"
            variant="outline"
            size="sm"
            class="shrink-0 rounded-xl"
            @click="emit('clearExactTagFilter')"
          >
            {{ t("library.clearFilter") }}
          </Button>
        </div>
        <p
          v-if="activeTagTrimmed"
          class="text-sm text-muted-foreground"
        >
          {{ t("library.filterActive") }}<span class="font-medium text-foreground">{{ activeTagTrimmed }}</span>
        </p>
      </CardHeader>
      <CardContent class="flex flex-col gap-6">
        <section class="flex flex-col gap-2">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">
              {{ t("library.metaTags") }}
            </p>
            <Button
              v-if="metaTagsHiddenCount > 0"
              type="button"
              variant="ghost"
              size="sm"
              class="h-8 shrink-0 gap-1 rounded-xl px-2 text-xs text-muted-foreground hover:text-foreground"
              :aria-expanded="metaTagsExpanded"
              @click="metaTagsExpanded = !metaTagsExpanded"
            >
              {{
                metaTagsExpanded
                  ? t("library.tagsShowLess")
                  : t("library.tagsShowMore", { count: metaTagsHiddenCount })
              }}
              <ChevronDown
                class="size-3.5 opacity-70 transition-transform duration-200"
                :class="metaTagsExpanded ? 'rotate-180' : ''"
                aria-hidden="true"
              />
            </Button>
          </div>
          <p
            v-if="metadataTagRanked.length === 0"
            class="text-sm text-muted-foreground"
          >
            {{ t("library.noMetaTags") }}
          </p>
          <div v-else class="flex flex-wrap gap-2">
            <Badge
              v-for="row in visibleMetaTags"
              :key="`meta-${row.tag}`"
              as-child
              :variant="isChipActive(row.tag) ? 'default' : 'secondary'"
              :class="[
                'rounded-full border px-3 py-1 text-sm font-normal transition-colors',
                isChipActive(row.tag)
                  ? 'border-primary/40'
                  : 'cursor-pointer border-border/60 bg-secondary/70 hover:bg-secondary hover:text-secondary-foreground',
              ]"
            >
              <button
                type="button"
                class="inline-flex max-w-full min-w-0 items-center gap-1.5"
                :aria-pressed="isChipActive(row.tag)"
                :aria-label="t('library.ariaFilterTag', { tag: row.tag, count: row.count })"
                @click="onTagChipClick(row.tag)"
              >
                <span class="truncate">{{ row.tag }}</span>
                <span
                  class="tabular-nums text-xs opacity-80"
                  :class="isChipActive(row.tag) ? 'text-primary-foreground/90' : 'text-muted-foreground'"
                >
                  · {{ row.count }}
                </span>
              </button>
            </Badge>
          </div>
        </section>

        <section class="flex flex-col gap-2">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">
              {{ t("library.userTags") }}
            </p>
            <Button
              v-if="userTagsHiddenCount > 0"
              type="button"
              variant="ghost"
              size="sm"
              class="h-8 shrink-0 gap-1 rounded-xl px-2 text-xs text-muted-foreground hover:text-foreground"
              :aria-expanded="userTagsExpanded"
              @click="userTagsExpanded = !userTagsExpanded"
            >
              {{
                userTagsExpanded
                  ? t("library.tagsShowLess")
                  : t("library.tagsShowMore", { count: userTagsHiddenCount })
              }}
              <ChevronDown
                class="size-3.5 opacity-70 transition-transform duration-200"
                :class="userTagsExpanded ? 'rotate-180' : ''"
                aria-hidden="true"
              />
            </Button>
          </div>
          <p
            v-if="userTagRanked.length === 0"
            class="text-sm text-muted-foreground"
          >
            {{ t("library.noUserTags") }}
          </p>
          <div v-else class="flex flex-wrap gap-2">
            <Badge
              v-for="row in visibleUserTags"
              :key="`user-${row.tag}`"
              as-child
              :variant="isChipActive(row.tag) ? 'default' : 'secondary'"
              :class="[
                'rounded-full border px-3 py-1 text-sm font-normal transition-colors',
                isChipActive(row.tag)
                  ? 'border-primary/40'
                  : 'cursor-pointer border-primary/25 bg-primary/10 hover:bg-primary/20',
              ]"
            >
              <button
                type="button"
                class="inline-flex max-w-full min-w-0 items-center gap-1.5"
                :aria-pressed="isChipActive(row.tag)"
                :aria-label="t('library.ariaFilterUserTag', { tag: row.tag, count: row.count })"
                @click="onTagChipClick(row.tag)"
              >
                <span class="truncate">{{ row.tag }}</span>
                <span
                  class="tabular-nums text-xs opacity-80"
                  :class="isChipActive(row.tag) ? 'text-primary-foreground/90' : 'text-muted-foreground'"
                >
                  · {{ row.count }}
                </span>
              </button>
            </Badge>
          </div>
        </section>
      </CardContent>
    </Card>

    <div
      v-if="props.mode === 'trash'"
      class="flex flex-wrap items-center justify-end gap-2"
    >
      <template v-if="!batchModeOn">
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="shrink-0 gap-1.5 rounded-xl"
          @click="emit('enterBatchMode')"
        >
          <ListChecks class="size-4 opacity-80" aria-hidden="true" />
          {{ t("library.batchManage") }}
        </Button>
      </template>
      <template v-else>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="shrink-0 gap-1.5 rounded-xl"
          @click="emit('selectAllVisibleInBatch')"
        >
          <CheckSquare class="size-4 opacity-80" aria-hidden="true" />
          {{ t("library.batchSelectVisible") }}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="shrink-0 gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground"
          @click="emit('exitBatchMode')"
        >
          <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
          {{ t("library.batchExitToolbar") }}
        </Button>
      </template>
    </div>

    <div
      v-else
      class="flex flex-wrap items-center justify-between gap-3 pb-1"
    >
      <Tabs
        :model-value="props.activeTab"
        class="min-w-0 flex-1 gap-4"
        @update:model-value="handleTabChange"
      >
        <TabsList class="h-auto w-fit max-w-full flex-wrap rounded-2xl bg-muted/60 p-1">
          <TabsTrigger value="all" class="rounded-xl px-4 py-2">
            {{ t("library.tabAll") }}
          </TabsTrigger>
          <TabsTrigger value="new" class="rounded-xl px-4 py-2">
            {{ t("library.tabNew") }}
          </TabsTrigger>
          <TabsTrigger value="top-rated" class="rounded-xl px-4 py-2">
            {{ t("library.tabTop") }}
          </TabsTrigger>
        </TabsList>
      </Tabs>
      <div class="flex shrink-0 flex-wrap items-center gap-2">
        <template v-if="!batchModeOn">
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl"
            @click="emit('enterBatchMode')"
          >
            <ListChecks class="size-4 opacity-80" aria-hidden="true" />
            {{ t("library.batchManage") }}
          </Button>
        </template>
        <template v-else>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="gap-1.5 rounded-xl"
            @click="emit('selectAllVisibleInBatch')"
          >
            <CheckSquare class="size-4 opacity-80" aria-hidden="true" />
            {{ t("library.batchSelectVisible") }}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="gap-1.5 rounded-xl text-muted-foreground hover:bg-muted/80 hover:text-foreground"
            @click="emit('exitBatchMode')"
          >
            <X class="size-4 shrink-0 opacity-80" aria-hidden="true" />
            {{ t("library.batchExitToolbar") }}
          </Button>
        </template>
      </div>
    </div>

    <div class="min-h-0 flex-1">
      <VirtualMovieMasonry
        :movies="props.visibleMovies"
        :selected-movie-id="props.selectedMovie?.id"
        :batch-mode="batchModeOn"
        :batch-selected-ids="props.batchSelectedIds ?? []"
        :scroll-preserve-key="props.scrollPreserveKey"
        :empty-title="props.mode === 'trash' ? t('library.trashEmptyTitle') : undefined"
        :empty-description="props.mode === 'trash' ? t('library.trashEmptyDesc') : undefined"
        @select="emit('select', $event)"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
        @toggle-favorite="emit('toggleFavorite', $event)"
        @context-menu="emit('contextMenu', $event)"
        @toggle-batch-select="emit('toggleBatchSelect', $event)"
      />
    </div>
  </div>
</template>
