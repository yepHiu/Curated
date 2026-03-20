<script setup lang="ts">
import { computed } from "vue"
import { DynamicScroller, DynamicScrollerItem } from "vue-virtual-scroller"
import type { Movie } from "@/domain/movie/types"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import MovieCard from "@/components/jav-library/MovieCard.vue"

interface MovieChunk {
  id: string
  items: Movie[]
  /** 稳定串，供虚拟列表测量缓存；避免每次渲染新建数组触发反复重算 */
  sizeKey: string
}

const props = defineProps<{
  movies: readonly Movie[]
  selectedMovieId?: string
}>()

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()

/** 单块卡片数过大时虚拟滚动几乎失效（一整块几十张 DOM）；12～16 较均衡 */
const CHUNK_SIZE = 12
/** 估算每行最少列数（用于虚拟列表高度粗算） */
const MIN_COLUMNS_ESTIMATE = 3
/** 卡片略放大后虚拟行高略增 */
const ESTIMATED_CARD_HEIGHT = 300
const ESTIMATED_GAP = 22

const estimatedChunkHeight =
  Math.ceil(CHUNK_SIZE / MIN_COLUMNS_ESTIMATE) * (ESTIMATED_CARD_HEIGHT + ESTIMATED_GAP)

/** 网格列最小宽度：窄屏不过小、随视口变宽、大屏有上限 */
const GRID_COL_MIN = "clamp(11.25rem, 9vw + 8.75rem, 15rem)"
/** 单卡最大宽度：避免超宽屏单卡拉得过大；与列 min 解耦 */
const CARD_MAX_WIDTH = "min(100%, clamp(12rem, 28vw, 19rem))"
/** 间距随视口在区间内变化 */
const GRID_GAP = "clamp(1rem, 2vw, 1.5rem)"

function buildChunkSizeKey(items: Movie[]): string {
  if (items.length === 0) {
    return ""
  }
  let key = ""
  for (let i = 0; i < items.length; i++) {
    const m = items[i]!
    key += `${m.id}:${m.isFavorite ? 1 : 0}:${m.tags.length}:`
  }
  return key
}

const movieChunks = computed<MovieChunk[]>(() => {
  const { movies } = props
  const total = movies.length
  if (total === 0) {
    return []
  }
  const chunks: MovieChunk[] = []
  for (let offset = 0; offset < total; offset += CHUNK_SIZE) {
    const items = movies.slice(offset, offset + CHUNK_SIZE)
    chunks.push({
      id: `chunk-${offset}`,
      items,
      sizeKey: buildChunkSizeKey(items),
    })
  }
  return chunks
})

const isMovieChunk = (value: unknown): value is MovieChunk =>
  typeof value === "object" &&
  value !== null &&
  "id" in value &&
  "items" in value &&
  "sizeKey" in value &&
  Array.isArray((value as MovieChunk).items)

const getChunk = (value: unknown): MovieChunk =>
  isMovieChunk(value)
    ? value
    : {
        id: "invalid-chunk",
        items: [],
        sizeKey: "",
      }
</script>

<template>
  <div v-if="props.movies.length" class="h-full min-h-0">
    <DynamicScroller
      :items="movieChunks"
      key-field="id"
      :min-item-size="estimatedChunkHeight"
      :buffer="220"
      class="h-full min-h-0 overflow-y-auto pr-2"
      list-class="flex flex-col gap-5"
      item-class="pb-5"
    >
      <template #default="{ item, index, active }">
        <DynamicScrollerItem
          :item="item"
          :active="active"
          :data-index="index"
          :size-dependencies="[getChunk(item).sizeKey]"
          :min-size="estimatedChunkHeight"
        >
          <!-- 列宽 min 用 clamp 响应式；1fr 吃满行；卡片区 max-width 限制单卡上限并居中 -->
          <div
            class="grid w-full"
            :style="{
              gridTemplateColumns: `repeat(auto-fill, minmax(min(100%, ${GRID_COL_MIN}), 1fr))`,
              columnGap: GRID_GAP,
              rowGap: GRID_GAP,
            }"
          >
            <div
              v-for="movie in getChunk(item).items"
              :key="movie.id"
              class="flex min-w-0 justify-center"
            >
              <div class="w-full" :style="{ maxWidth: CARD_MAX_WIDTH }">
              <MovieCard
                :movie="movie"
                :selected="movie.id === props.selectedMovieId"
                :show-favorite="false"
                @select="emit('select', $event)"
                @open-details="emit('openDetails', $event)"
                @open-player="emit('openPlayer', $event)"
                @toggle-favorite="emit('toggleFavorite', $event)"
              />
              </div>
            </div>
          </div>
        </DynamicScrollerItem>
      </template>
    </DynamicScroller>
  </div>

  <Card v-else class="rounded-3xl border-border/70 bg-card/80">
    <CardHeader>
      <CardTitle>No matches found</CardTitle>
      <CardDescription>
        Try another query or switch to a different library tab.
      </CardDescription>
    </CardHeader>
    <CardContent class="text-sm text-muted-foreground">
      The current filters do not return any movies in this route view.
    </CardContent>
  </Card>
</template>
