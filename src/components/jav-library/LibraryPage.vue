<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import VirtualMovieMasonry from "@/components/jav-library/VirtualMovieMasonry.vue"
import {
  aggregateMetadataTagCounts,
  aggregateUserTagCounts,
} from "@/lib/library-stats"

const props = defineProps<{
  mode: LibraryMode
  allMovies: readonly Movie[]
  visibleMovies: readonly Movie[]
  selectedMovie?: Movie
  activeTab: LibraryTab
  /** 当前 URL 精确标签筛选（用于高亮） */
  activeTagFilter?: string
}>()

const emit = defineEmits<{
  "update:activeTab": [value: LibraryTab]
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId?: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
  /** 与详情页点标签一致：写入 `tag`、清除 `q`/`actor` */
  browseByExactTag: [tag: string]
  clearExactTagFilter: []
}>()

const { t, locale } = useI18n()
const TAG_SECTION_LIMIT = 36

const metadataTagRanked = computed(() =>
  aggregateMetadataTagCounts(props.allMovies, locale.value).slice(0, TAG_SECTION_LIMIT),
)

const userTagRanked = computed(() =>
  aggregateUserTagCounts(props.allMovies, locale.value).slice(0, TAG_SECTION_LIMIT),
)

const activeTagTrimmed = computed(() => props.activeTagFilter?.trim() ?? "")

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
  <div class="flex h-full min-h-0 flex-col gap-7">
    <Card
      v-if="props.mode === 'tags'"
      class="rounded-3xl border-border/70 bg-card/85 shadow-lg shadow-black/5"
    >
      <CardHeader class="gap-3">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 flex-1 space-y-1">
            <CardTitle>{{ t("library.tagBrowseTitle") }}</CardTitle>
            <CardDescription class="text-pretty">
              {{ t("library.tagBrowseDesc") }}
            </CardDescription>
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
          <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            {{ t("library.metaTags") }}
          </p>
          <p
            v-if="metadataTagRanked.length === 0"
            class="text-sm text-muted-foreground"
          >
            {{ t("library.noMetaTags") }}
          </p>
          <div v-else class="flex flex-wrap gap-2">
            <Badge
              v-for="row in metadataTagRanked"
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
          <p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            {{ t("library.userTags") }}
          </p>
          <p
            v-if="userTagRanked.length === 0"
            class="text-sm text-muted-foreground"
          >
            {{ t("library.noUserTags") }}
          </p>
          <div v-else class="flex flex-wrap gap-2">
            <Badge
              v-for="row in userTagRanked"
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

    <Tabs
      :model-value="props.activeTab"
      class="gap-5"
      @update:model-value="handleTabChange"
    >
      <TabsList class="h-auto w-fit flex-wrap rounded-2xl bg-muted/60 p-1">
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

    <div class="min-h-0 flex-1">
      <VirtualMovieMasonry
        :movies="props.visibleMovies"
        :selected-movie-id="props.selectedMovie?.id"
        @select="emit('select', $event)"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
        @toggle-favorite="emit('toggleFavorite', $event)"
      />
    </div>
  </div>
</template>
